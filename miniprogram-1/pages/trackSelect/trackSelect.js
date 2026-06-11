const request = require('../../utils/request.js');

Page({
  data: {
    mode: 'new',
    selectedCount: 0,
    tracks: []
  },

  onLoad(options) {
    const currentMode = options.mode || 'new';
    this.setData({ mode: currentMode });

    this.fetchDynamicTracks();

    if (currentMode === 'edit') {
      this.checkEditDeadline();
    }
  },

  checkEditDeadline() {
    wx.showLoading({ title: '核验权限中...' });
    request({
      url: '/api/admin/settings/get', 
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      const data = res.data || res || {};
      let deadlineStr = '2026-05-31 23:59:59';
      if (data.track_edit_deadline) {
        deadlineStr = data.track_edit_deadline;
      }

      const deadlineDate = new Date(deadlineStr.replace(/-/g, '/'));
      const now = new Date();

      if (now > deadlineDate) {
        wx.showModal({
          title: '修改通道已关闭',
          content: `赛道修改期限已过，当前禁止调整。`,
          showCancel: false,
          success: () => wx.navigateBack()
        });
      }
    }).catch(() => { wx.hideLoading(); });
  },

  fetchDynamicTracks() {
    wx.showLoading({ title: '正在同步赛道...' });
    
    request({
      url: '/api/auth/tracks', 
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      const list = res.data || res || [];
      if (Array.isArray(list) && list.length > 0) {
        const formattedTracks = list.map(track => {
          let icon = '🎯'; 
          const name = track.name || '';
          if (name.includes('运动') || name.includes('逐光') || name.includes('跑步')) icon = '🏃';
          else if (name.includes('语路') || name.includes('英语')) icon = '🗣️';
          else if (name.includes('阅读') || name.includes('墨香')) icon = '📚';
          else if (name.includes('代码') || name.includes('码上')) icon = '💻';
          
          return {
            id: track.id,
            name: track.name,
            desc: track.description || track.desc || '暂无详细要求说明',
            type: track.type,
            icon: icon,
            selected: false 
          };
        });
        this.setData({ tracks: formattedTracks });
      } else {
        wx.showToast({ title: '赛道列表为空或拉取失败', icon: 'none' });
      }
    }).catch(() => { wx.hideLoading(); });
  },

  toggleSelect(e) {
    const index = e.currentTarget.dataset.index;
    const tracks = this.data.tracks;
    tracks[index].selected = !tracks[index].selected;
    
    const count = tracks.filter(t => t.selected).length;
    this.setData({ 
      tracks: tracks,
      selectedCount: count 
    });
  },

  confirmSelection() {
    if (this.data.selectedCount < 2) {
      wx.showToast({ title: '为了全面发展，请至少点选2个及以上运行赛道！', icon: 'none' });
      return;
    }
    
    const selectedTracks = this.data.tracks.filter(t => t.selected);
    const trackIds = selectedTracks.map(t => t.id);
    const userId = wx.getStorageSync('userId');

    if (this.data.mode === 'edit') {
      wx.showModal({
        title: '⚠️ 危险操作警告',
        content: '修改赛道后，系统将彻底重置您之前的打卡记录以资重新审计，确定吗？',
        confirmText: '确认修改',
        confirmColor: '#FF4757',
        success: (res) => {
          if (res.confirm) this.submitData(userId, trackIds, selectedTracks);
        }
      });
    } else {
      this.submitData(userId, trackIds, selectedTracks);
    }
  },

  submitData(userId, trackIds, selectedTracks) {
    wx.showLoading({ title: '全原子档案落库中...' });
    
    if (this.data.mode === 'new') {
      // 💯 新流程二合一：提取第一阶段暂存的完整注册 Payload
      const tempPayload = wx.getStorageSync('temp_reg_payload');
      if (!tempPayload) {
        wx.hideLoading();
        return wx.showModal({
          title: '会话已失效',
          content: '未检测到合规的临时注册表单，请返回重新填写学籍资料。',
          showCancel: false,
          success: () => wx.reLaunch({ url: '/pages/register/register' })
        });
      }

      // 级联注入赛道切片 ID 数组，组合出最高合规数据大包
      tempPayload.track_ids = trackIds;

      request({
        url: '/api/auth/register', 
        method: 'POST',
        data: tempPayload
      }).then(res => {
        wx.hideLoading();
        console.log("👉 [注册落库全量回执核对]:", res);

        const finalToken = res.token || (res.data && res.data.token) || res.Token || '';
        if (!finalToken) {
          return wx.showModal({
            title: '令牌核发失败',
            content: '未能从小程序网络层解析出合规的 JWT Token，请检查解包层级。',
            showCancel: false
          });
        }

        const data = res.data || res || {};

        // Only after 100% confirmation of token, we proceed to clear the temporary registration payload cache!
        wx.removeStorageSync('temp_reg_payload');

        // Check account status for veteran students
        /*
        if (data.account_status === 1) {
          return wx.showModal({
            title: '资质扣留沙盒待审',
            content: '由于您的学号前缀触发了退伍老生/老兵/体验账户分流网关，您的档案已提交人工特权审批，请联系学生指挥官进行一键放行！',
            showCancel: false,
            success: () => { wx.reLaunch({ url: '/pages/login/login' }); }
          });
        }
        */

        // Mainstream students continue
        wx.setStorageSync('userToken', finalToken);
        wx.setStorageSync('userId', data.userId);
        wx.setStorageSync('hasSelectedTracks', true);
        wx.setStorageSync('myTracks', selectedTracks);
        
        // 💡 智能双向兼容存储：同时写入驼峰 (camelCase) 与蛇形 (snake_case) 字段，彻底解决前后端及页面间字段不一致的隐患
        wx.setStorageSync('userInfo', {
          // 驼峰命名 (供大部分核心页面展示读取)
          realName: tempPayload.name,
          className: tempPayload.class,
          studentId: tempPayload.student_id,
          avatar: tempPayload.avatar_url,
          accountStatus: 0,
          userType: data.user_type || data.userType || 0,
          currentDeviceId: tempPayload.device_id,

          // 蛇形下划线命名 (兼容老旧分支逻辑)
          real_name: tempPayload.name,
          class_name: tempPayload.class,
          student_id: tempPayload.student_id,
          account_status: 0,
          user_type: data.user_type || data.userType || 0,
          current_device_id: tempPayload.device_id
        });

        wx.setStorageSync('savedLoginInfo', {
          name: tempPayload.name,
          clazz: tempPayload.class,
          stuId: tempPayload.student_id,
          avatar: tempPayload.avatar_url
        });

        wx.showToast({ title: '全套档案建档成功！', icon: 'success', duration: 1200 });
        setTimeout(() => { wx.reLaunch({ url: '/pages/home/home' }); }, 1200);

      }).catch((err) => {
        wx.hideLoading();
        // Strict warning: DO NOT clear temporary cache on failures/errors to permit retries and prevent data loss.
        wx.showModal({ 
          title: '全网原子拦截阻断', 
          content: (err && (err.message || err.msg || err.error || err.errMsg)) || '由于网络超时或安全链路阻断，建档未通过，请检查网络或重试。', 
          showCancel: false 
        });
      });
    } 
    else {
      // 老用户修改赛道保持不变，无缝通车
      request({
        url: '/api/user/changeTracks',
        method: 'POST',
        data: {
          userId: userId,
          trackIds: trackIds
        }
      }).then(res => {
        wx.hideLoading();
        wx.setStorageSync('hasSelectedTracks', true);
        wx.setStorageSync('myTracks', selectedTracks); 
        wx.reLaunch({ url: '/pages/home/home' });
      }).catch((err) => {
        wx.hideLoading();
        wx.showToast({ title: (err && (err.message || err.msg)) || '调整失败', icon: 'none' });
      });
    }
  }
});