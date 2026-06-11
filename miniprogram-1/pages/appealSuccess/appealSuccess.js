Page({
  goBack() {
    // 💡 因为我们是用的 wx.redirectTo 跳转过来的，所以直接 navigateBack 退回即可精确回到“历史大厅”
    wx.navigateBack({
      delta: 1,
      fail: () => {
        // 兜底方案：如果页面栈乱了，强制重定向回申诉大厅
        wx.redirectTo({ url: '/pages/appealList/appealList' });
      }
    });
  }
})