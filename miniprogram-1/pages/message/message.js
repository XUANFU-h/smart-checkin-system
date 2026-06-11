const request = require('../../utils/request.js');

Page({
  data: {
    messages: []
  },

  onShow() {
    this.fetchMessages();
  },

  onPullDownRefresh() {
    this.fetchMessages();
    wx.stopPullDownRefresh();
  },

  fetchMessages() {
    wx.showLoading({ title: '加载中...' });
    request({
      url: '/message/list',
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      if (res && res.success) {
        this.setData({ messages: res.data || [] });
      }
    }).catch(() => { 
      wx.hideLoading(); 
    });
  },

  readAll() {
    const unreadExists = this.data.messages.some(m => m.isRead === 0);
    if (!unreadExists || this.data.messages.length === 0) return;

    // 💡 优化：乐观 UI 更新 (Optimistic UI)
    // 瞬间抹除红点，不等待网络响应，提供极其丝滑的体验
    const optimizedMsgs = this.data.messages.map(m => ({ ...m, isRead: 1 }));
    this.setData({ messages: optimizedMsgs });

    // 后台静默发送已读请求，不需要再 showLoading 打断用户操作
    request({
      url: '/message/read-all',
      method: 'POST'
    }).then(res => {
      if (!res || !res.success) {
        // 如果后端真出错了，再把数据重新拉回来恢复红点
        this.fetchMessages();
        wx.showToast({ title: '同步状态失败', icon: 'none' });
      }
    }).catch(() => {
      this.fetchMessages();
    });
  }
})