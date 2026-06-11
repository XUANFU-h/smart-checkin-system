Page({
  data: {},
  onLoad(options) {
    if (options.type === 'reset') {
      wx.setNavigationBarTitle({ title: '异地重置审核' });
    }
  },
  goBack() {
    wx.reLaunch({ url: '/pages/login/login' });
  }
});