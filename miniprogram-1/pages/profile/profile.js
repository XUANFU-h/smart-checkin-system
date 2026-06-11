// 💡 采用最新封装的全局拦截请求模块
const request = require('../../utils/request.js');

Page({
  data: {
    userInfo: {},
    myTracks: [],
    empowermentScore: 0,
    moralScore: 0,
    empLogs: [],
    moralLogs: [],
    showLogModal: false,
    currentLogType: 'emp', 
    
    userTypeLabel: '正式用户',
    userTypeClass: 'tag-official',
    currentLogTitle: '赋能分',
    currentTotalScore: 0,
    currentLogClass: 'emp-color',
    currentLogs: []
  },

  onShow() {
    this.init();
  },

  async init() {
    const userInfo = wx.getStorageSync('userInfo') || {};
    const tracks = wx.getStorageSync('myTracks') || [];
    
    // 内部实现赛道图标匹配，脱离外部依赖
    const myTracks = tracks.map(t => {
      let icon = '🎯';
      if (t.name.includes('早起')) icon = '🌅';
      if (t.name.includes('运动') || t.name.includes('跑步')) icon = '🏃';
      if (t.name.includes('单词') || t.name.includes('阅读')) icon = '📖';
      return { ...t, icon };
    });

    this.setData({
      userInfo,
      myTracks: myTracks,
      userTypeLabel: userInfo.userType === 2 ? '学生干部' : (userInfo.userType === 1 ? 'Beta内测' : '正式用户'),
      userTypeClass: userInfo.userType === 2 ? 'tag-cadre' : (userInfo.userType === 1 ? 'tag-beta' : 'tag-official')
    });
    
    this.fetchUserScores();
  },

  // 💡 使用封装好的 request，自动带 Token 鉴权并拦截 401 封号状态
  fetchUserScores() {
    request({
      url: '/user/score-info',
      method: 'GET'
    }).then(res => {
      if (res && res.success && res.data) {
        const data = res.data;

        // 💡 修复：极度防御性的时间格式化引擎
        const formatTimeExact = (timeStr) => {
          if (!timeStr) return '未知时间';
          try {
            const str = String(timeStr);
            const validTimeStr = str.replace(/-/g, '/').replace('T', ' ').split('.')[0];
            const d = new Date(validTimeStr);
            if (isNaN(d.getTime())) return str;
            
            return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日${d.getHours()}点${d.getMinutes()}分${d.getSeconds()}秒`;
          } catch (e) {
            return String(timeStr);
          }
        };

        const processLog = (l) => {
          if (!l) return {};
          const actionStr = l.action ? String(l.action).toLowerCase() : '';
          const score = Number(l.score) || 0;
          const isSub = actionStr === 'sub' || score < 0;
          
          let rawReason = l.reason || l.description || '系统变动';
          let rawOperator = l.operator || '';

          // 核心拦截：判断是否为自行兑换（如商城兑换德育分）
          let isSelfExchange = rawReason.includes('兑换') || rawReason.includes('商城') || l.type === 'EXCHANGE';
          
          let finalReason = rawReason;
          let finalOperator = rawOperator;

          if (isSelfExchange) {
            finalReason = '用户自行兑换';
            const uInfo = wx.getStorageSync('userInfo') || {};
            finalOperator = uInfo.realName || uInfo.nickname || '用户本人';
          } else {
            // 拦截：系统或人工处理
            if (!rawOperator || rawOperator === 'AI质检中心' || rawOperator === '系统自动') {
              finalOperator = '系统自动处理';
            } else {
              // 如果是真实人员/管理员，强行打上【一级管理员】后缀
              let cleanName = rawOperator.replace(/【.*?】/g, '').trim();
              if (cleanName.toUpperCase() === 'ADMIN') cleanName = '系统管理员';
              finalOperator = `${cleanName}【一级管理员】`;
            }
          }

          return {
            ...l,
            isSub,
            absScore: Math.abs(score),
            realScore: isSub ? -Math.abs(score) : Math.abs(score),
            formattedTime: formatTimeExact(l.createTime),
            displayOperator: finalOperator,
            reason: finalReason
          };
        };

        const processedEmpLogs = (data.empLogs || []).map(processLog);
        const processedMoralLogs = (data.moralLogs || []).map(processLog);

        const calculatedEmp = processedEmpLogs.reduce((acc, cur) => acc + (cur.realScore || 0), 0);
        const calculatedMoral = processedMoralLogs.reduce((acc, cur) => acc + (cur.realScore || 0), 0);

        // 💡 修复：确保传入 setData 的全都是安全的 Number，绝不是 undefined 或 null
        this.setData({
          empLogs: processedEmpLogs,
          moralLogs: processedMoralLogs,
          empowermentScore: data.empowermentScore != null ? Number(data.empowermentScore) : calculatedEmp,
          moralScore: data.moralScore != null ? Number(data.moralScore) : calculatedMoral
        });

        // 静默同步本地基础信息
        const updated = { ...this.data.userInfo, userType: data.userType || 0, studentId: data.studentId || '' };
        wx.setStorageSync('userInfo', updated);
        if (data.trackModifyCount != null) {
          wx.setStorageSync('trackModifyCount', data.trackModifyCount);
        }
      }
    }).catch((err) => {
      console.error("Score Info Fetch Error:", err);
    });
  },

  openEmpowermentModal() {
    this.setData({
      currentLogType: 'emp',
      currentLogTitle: '赋能分',
      currentTotalScore: this.data.empowermentScore,
      currentLogClass: 'emp-color',
      currentLogs: this.data.empLogs,
      showLogModal: true
    });
  },

  openMoralModal() {
    this.setData({
      currentLogType: 'moral',
      currentLogTitle: '德育分',
      currentTotalScore: this.data.moralScore,
      currentLogClass: 'moral-color',
      currentLogs: this.data.moralLogs,
      showLogModal: true
    });
  },

  closeModals() {
    this.setData({ showLogModal: false });
  },

  goToEditProfile() {
    wx.navigateTo({ url: '/pages/editProfile/editProfile' });
  },

  goToPage(e) {
    const url = e.currentTarget.dataset.url;
    if (url.includes('home')) {
      wx.switchTab({ url });
    } else {
      wx.navigateTo({ url });
    }
  },

  // 💡 已移除前端入口，函数保留为后端预留
  /*
  goToModifyTrack() {
    const DEADLINE = new Date('2026-05-18T23:59:59').getTime();
    if (Date.now() > DEADLINE) {
      return wx.showModal({ title: '修改通道已关闭', content: '活动已开始超过7天，为维护公平，赛道已锁定。', showCancel: false });
    }
    const modifyCount = wx.getStorageSync('trackModifyCount') || 0;
    if (modifyCount <= 0) {
      return wx.showModal({ title: '机会已用尽', content: '每位同学仅有一次修改赛道的机会。', showCancel: false });
    }
    wx.showModal({ 
      title: '确认修改', 
      content: '确定要消耗唯一的修改机会重新选择赛道吗？',
      success: (res) => {
        if (res.confirm) {
          wx.navigateTo({ url: '/pages/trackSelect/trackSelect?mode=edit' });
        }
      }
    });
  },
  */

  preventClose() {} 
});