const request = require('../../utils/request.js');

Page({
  data: {
    trackId: '', 
    trackName: '',
    isExpired: false, 
    showComment: false, 
    currentRecord: {},
    
    // 榜单数据
    globalLeaderboard: [],
    classLeaderboard: [],
    myGlobalRank: '未上榜',
    myClassRank: '未上榜',
    myYesterdayScore: 0,

    myRecords: [] 
  },

  onLoad(options) {
    this.setData({ trackId: options.id, trackName: options.name || '赛道详情' });
  },

  onShow() {
    this.fetchSystemDeadlineAndCheck();
    this.fetchTrackData();
  },

  // 💡 1. 动态拉取后端截止时间，并与当前时间比对 (判断是否还能修改/删除)
  fetchSystemDeadlineAndCheck() {
    request({ url: '/admin/settings/get', method: 'GET' }).then(res => {
      let deadlineStr = '22:00'; // 默认值
      if (res && res.success && res.data) {
        deadlineStr = res.data.deadline || res.data.checkin_deadline || '22:00';
      }
      
      const now = new Date();
      const timeParts = deadlineStr.split(':');
      const target = new Date(now.getTime());
      target.setHours(parseInt(timeParts[0] || 22), parseInt(timeParts[1] || 0), 0, 0);
      
      this.setData({ isExpired: now > target });
    }).catch(() => {
      // 失败则使用默认 22:00
      const target = new Date();
      target.setHours(22, 0, 0, 0);
      this.setData({ isExpired: new Date() > target });
    });
  },

  // 💡 2. 真实对接后端的详情和排名接口
  fetchTrackData() {
    wx.showLoading({ title: '拉取数据中' });

    request({
      url: `/track/detail?trackId=${this.data.trackId}`,
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      if (res && res.success && res.data) {
        const data = res.data;
        
        // 核心脱敏处理 (如：李四 -> 李*)
        const maskName = (name) => {
          if (!name) return '匿*';
          if (name.length <= 2) return name.substring(0, 1) + '*';
          return name.substring(0, 1) + '*' + name.substring(name.length - 1);
        };
        
        const processList = (list) => {
          return (list || []).map(item => ({ ...item, maskedName: maskName(item.userName || item.realName) }));
        };

        // 判断足迹是否是今天的 (控制删除/修改权限)
        const todayStr = new Date().toISOString().split('T')[0];
        const processedRecords = (data.myRecords || []).map(record => {
           // 处理 Go 后端的 RFC3339 日期格式
           const rDate = record.createTime ? record.createTime.split('T')[0] : '';
           return {
             ...record,
             date: record.createTime ? record.createTime.replace('T', ' ').split('+')[0] : '', // 显示用
             isToday: rDate === todayStr,
             score: record.aiScore || record.score
           }
        });

        this.setData({
          globalLeaderboard: processList(data.globalList),
          classLeaderboard: processList(data.classList),
          myGlobalRank: data.myGlobalRank > 0 ? data.myGlobalRank : '未上榜',
          myClassRank: data.myClassRank > 0 ? data.myClassRank : '未上榜',
          myYesterdayScore: data.myYesterdayScore || 0,
          myRecords: processedRecords
        });
      }
    }).catch(() => {
      wx.hideLoading();
    });
  },

  // 💡 3. 真实删除当日打卡记录
  deleteRecord(e) {
    const recordId = e.currentTarget.dataset.id;
    wx.showModal({
      title: '删除警告',
      content: '删除后无法恢复，且需要在今日截止前重新提交，是否确认删除？',
      confirmColor: '#F56C6C',
      success: (res) => {
        if (res.confirm) {
          wx.showLoading({ title: '抹除档案中...' });
          
          request({
             url: `/checkin/delete?recordId=${recordId}`,
             method: 'DELETE'
          }).then(delRes => {
             wx.hideLoading();
             if (delRes && delRes.success) {
                wx.showToast({ title: '已删除', icon: 'success' });
                this.fetchTrackData(); // 重新拉取足迹刷新 UI
             } else {
                wx.showToast({ title: delRes.message || '删除失败', icon: 'none' });
             }
          }).catch(() => {
             wx.hideLoading();
          });
        }
      }
    });
  },

  // 修改记录
  editRecord(e) { 
    const recordId = e.currentTarget.dataset.id;
    // 携带记录 ID 跳转回打卡页，触发覆盖更新
    wx.navigateTo({ url: `/pages/checkinForm/checkinForm?mode=edit&id=${recordId}` });
  },
  
  // 申诉功能 (附带微信订阅消息防崩处理)
  goToAppeal() { 
    try {
      wx.requestSubscribeMessage({
        tmplIds: ['aYdt-BAZoGs2BpiL24iHMtkHgPKc9zSOBiMeggYf5h8'], // 你真实申请到的模板ID
        success: () => { wx.navigateTo({ url: '/pages/appeal/appeal' }); },
        fail: () => { wx.navigateTo({ url: '/pages/appeal/appeal' }); }
      });
    } catch (e) {
      // 拦截开发者工具可能引发的异常
      wx.navigateTo({ url: '/pages/appeal/appeal' });
    }
  },

  // 评价模态框与复盘
  openCommentModal(e) { this.setData({ currentRecord: e.currentTarget.dataset.item, showComment: true }); },
  closeCommentModal() { this.setData({ showComment: false }); },
  preventDummpy() {},
  
  onSelfCommentInput(e) { this.setData({ ['currentRecord.selfComment']: e.detail.value }); },
  
  saveSelfComment() {
    if (!this.data.currentRecord || !this.data.currentRecord.id) return;
    
    // 如果复盘内容没有修改，直接关闭
    if (!this.data.currentRecord.selfComment) return this.closeCommentModal();

    wx.showLoading({ title: '保存中...' });
    
    // 💡 新增：向后端提交个人复盘的接口请求
    request({
      url: '/checkin/update-self-comment',
      method: 'POST',
      data: {
        recordId: this.data.currentRecord.id,
        selfComment: this.data.currentRecord.selfComment
      }
    }).then(res => {
      wx.hideLoading();
      if (res && res.success) {
        wx.showToast({ title: '复盘已保存', icon: 'success' });
        this.closeCommentModal();
        this.fetchTrackData(); // 刷新列表以保存持久化状态
      } else {
        wx.showToast({ title: '保存失败', icon: 'none' });
      }
    }).catch(() => { wx.hideLoading(); });
  }
})