const request = require('../../utils/request.js');

Page({
  data: {
    resetToken: '',
    password: '',
    confirmPassword: '',
    isSubmitting: false
  },

  onLoad(options) {
    if (!options.token) {
      wx.showToast({ title: '非法访问', icon: 'error' });
      setTimeout(() => { wx.reLaunch({ url: '/pages/login/login' }); }, 1500);
      return;
    }
    this.setData({ resetToken: options.token });
  },

  handleInput(e) {
    this.setData({ [e.currentTarget.dataset.field]: e.detail.value.trim() });
  },

  submitReset() {
    if (this.data.isSubmitting) return;

    const { password, confirmPassword, resetToken } = this.data;
    if (!password || !confirmPassword) return wx.showToast({ title: '请填写完整', icon: 'none' });
    if (password !== confirmPassword) return wx.showModal({ title: '校验失败', content: '两次密码不一致', showCancel: false });

    this.setData({ isSubmitting: true });
    wx.showLoading({ title: '密码加密保存中...' });

    request({
      url: '/api/auth/forgot-password/reset',
      method: 'POST',
      data: { resetToken: resetToken, password: password }
    }).then(() => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();
      wx.showToast({ title: '密码重置成功', icon: 'success' });
      
      // 重置成功，直接遣返登录页重新登录
      setTimeout(() => {
        wx.reLaunch({ url: '/pages/login/login' });
      }, 1500);
    }).catch((err) => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();
      wx.showModal({ title: '重置失败', content: err.message || err.msg || '票据过期，请重新申请', showCancel: false });
    });
  }
});