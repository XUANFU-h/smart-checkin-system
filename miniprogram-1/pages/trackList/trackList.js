const request = require('../../utils/request.js');

Page({
  data: {
    tracks: [] // 存放从后端拉取到的赛道数据
  },

  onLoad() {
    this.fetchTracks();
  },

  fetchTracks() {
    wx.showLoading({ title: '加载中...' });
    
    // 💡 接入统一安全拦截器，自动携带 JWT
    request({
      url: '/track/list',
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      if (res && res.success) {
        this.setData({
          tracks: res.data || []
        });
      } else {
        wx.showToast({ title: '获取赛道失败', icon: 'error' });
      }
    }).catch(() => {
      wx.hideLoading();
    });
  },

  // 点击某个赛道卡片时触发
  goToCheckin(e) {
    // 获取用户点击的是哪个赛道（从 wxml 的 data-id 和 data-name 里拿）
    const trackId = e.currentTarget.dataset.id;
    const trackName = e.currentTarget.dataset.name;
    
    console.log("准备进入打卡页，赛道ID：", trackId, "名称：", trackName);
    
    // 跳转到打卡页，并把赛道 ID 和名称通过 URL 传过去
    wx.navigateTo({
      url: `/pages/checkin/checkin?id=${trackId}&name=${trackName}`
    });
  }
})