const request = require('../../utils/request.js');

Page({
  data: {
    realName: '',
    className: '',
    studentId: '',
    captchaId: '',
    captchaAnswer: '',
    captchaImage: '',
    classList: [],
    showClassPicker: false,
    isSubmitting: false
  },

  onLoad() {
    this.fetchClassList();
    this.refreshCaptcha();
  },

  handleInput(e) {
    this.setData({ [e.currentTarget.dataset.field]: e.detail.value.trim() });
  },

  fetchClassList() {
    request({ url: '/api/auth/classes', method: 'GET' }).then((res) => {
      this.setData({ classList: res || [] });
    }).catch(() => {
      this.setData({ classList: ['软件G241', '软件G242', '大数据G241', '网络G241'] });
    });
  },

  openClassPicker() { this.setData({ showClassPicker: true }); },
  closeClassPicker() { this.setData({ showClassPicker: false }); },
  preventClose() {},
  selectClass(e) {
    this.setData({ className: e.currentTarget.dataset.class, showClassPicker: false });
  },

  // 💯 终极闭环绝杀：剥除多余的 .data 节点，一锤定音渲染验证码
  refreshCaptcha() {
    request({ url: '/api/auth/captcha/generate', method: 'GET' }).then((res) => {
      const data = res || {};
      this.setData({ 
        captchaImage: data.captchaImage, 
        captchaId: data.captchaId, 
        captchaAnswer: '' 
      });
    }).catch(() => { wx.showToast({ title: '验证码加载失败', icon: 'none' }); });
  },

  submitApply() {
    if (this.data.isSubmitting) return;

    const { realName, className, studentId, captchaId, captchaAnswer } = this.data;
    if(!realName || !className || !studentId || !captchaAnswer) {
      return wx.showToast({ title: '请完善身份验证信息', icon: 'none' });
    }

    const deviceId = wx.getStorageSync('deviceId');
    this.setData({ isSubmitting: true });
    wx.showLoading({ title: '安全核验中...' });

    request({
      url: '/api/auth/forgot-password/apply',
      method: 'POST',
      // 💯 核心对齐：将原始的 stuId 彻底改写为后端所需的统一标准字段下划线 student_id，消灭 400 参数拦截！
      data: { 
        name: realName, 
        class: className, 
        student_id: studentId, 
        device_id: deviceId, 
        captchaId: captchaId, 
        captchaAnswer: captchaAnswer 
      }
    }).then((res) => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();

      const data = res || {};
      if (data.action === 'ALLOW_RESET') {
        wx.showToast({ title: '设备验证通过', icon: 'success' });
        setTimeout(() => {
          wx.redirectTo({ url: `/pages/resetPassword/resetPassword?token=${data.resetToken}` });
        }, 1000);
      } else if (data.action === 'PENDING_APPROVAL') {
        wx.redirectTo({ url: '/pages/pendingApproval/pendingApproval?type=reset' });
      }
    }).catch((err) => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();
      wx.showModal({ title: '验证失败', content: err.message || err.msg || '信息不匹配或网络异常', showCancel: false });
      this.refreshCaptcha();
    });
  }
});