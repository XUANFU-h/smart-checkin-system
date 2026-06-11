const request = require('../../utils/request.js');

Page({
  data: {
    step: 0,
    isReturningUser: false,
    avatarUrl: '', 
    
    // 表单状�?
    realName: '',
    className: '',
    password: '',
    captchaId: '',
    captchaAnswer: '',
    captchaImage: '',
    classList: [],
    
    // 悬浮动画的焦点捕获状�?
    nameFocus: false,
    classFocus: false,
    pwdFocus: false,
    captchaFocus: false,

    isSubmitting: false,
    showClassPicker: false
  },

  animTimer: null,

  onShow() {
    if (!wx.getStorageSync('isFirstTimeEntered')) {
      wx.setStorageSync('isFirstTimeEntered', true);
      wx.navigateTo({ url: '/pages/register/register' });
      return;
    }

    let deviceId = wx.getStorageSync('deviceId');
    if (!deviceId) {
      deviceId = 'dev_' + Math.random().toString(36).substr(2) + Date.now().toString(36);
      wx.setStorageSync('deviceId', deviceId);
    }

    if (wx.getStorageSync('forceLogout')) {
      wx.removeStorageSync('forceLogout');
      wx.showModal({ title: '系统警告', content: '您的设备或账号已被安全系统拦截并强制下线�?, showCancel: false });
    }

    const token = wx.getStorageSync('userToken');
    if (token) {
      const hasSelected = wx.getStorageSync('hasSelectedTracks');
      if (hasSelected) return wx.switchTab({ url: '/pages/home/home' });
      else return wx.reLaunch({ url: '/pages/trackSelect/trackSelect?mode=new' });
    }

    this.loadSavedUser();
    this.startAnimation();
    this.fetchClassList(); 

    if (!this.data.isReturningUser) {
      this.refreshCaptcha();
    }
  },

  onHide() {
    if (this.animTimer) clearTimeout(this.animTimer);
  },

  startAnimation() {
    this.setData({ step: 0 });
    if (this.animTimer) clearTimeout(this.animTimer);
    this.animTimer = setTimeout(() => {
      this.setData({ step: 1 });
    }, 3500);
  },

  fetchClassList() {
    request({
      url: '/api/auth/classes',
      method: 'GET'
    }).then((res) => {
      let list = res || [];
      if (!list.includes('体验账户')) {
        list.push('体验账户');
      }
      this.setData({ classList: list });
    }).catch(() => {
      this.setData({ classList: ['软件G241', '软件G242', '大数据G241', '网络G241', '体验账户'] });
    });
  },

  // 💯 终极修复验证码：因为中枢已经自动脱壳，这里的 res 就是高亮原生生成的真实算�?SVG
  refreshCaptcha() {
    request({
      url: '/api/auth/captcha/generate',
      method: 'GET'
    }).then((res) => {
      const data = res || {};
      this.setData({
        captchaImage: data.captchaImage, // 完美直连本地算术 Base64 矢量图形�?
        captchaId: data.captchaId,
        captchaAnswer: ''
      });
    }).catch(() => {
      wx.showToast({ title: '验证码加载失�?, icon: 'none' });
    });
  },

  loadSavedUser() {
    const saved = wx.getStorageSync('savedLoginInfo');
    if (saved && saved.name) { 
      this.setData({
        isReturningUser: true,
        avatarUrl: saved.avatar || 'http://127.0.0.1:8080/static/default-avatar.png',
        realName: saved.name || '',
        className: saved.clazz || ''
      });
    }
  },

  switchAccount() {
    this.setData({
      isReturningUser: false,
      avatarUrl: '',
      realName: '',
      className: '',
      password: '',
      captchaAnswer: ''
    });
    this.refreshCaptcha();
  },

  fillAndSwitch() {
    this.setData({
      isReturningUser: false,
      pwdFocus: true 
    });
    this.refreshCaptcha();
  },

  onFocusName() { this.setData({ nameFocus: true }); },
  onBlurName() { this.setData({ nameFocus: false }); },
  onFocusPwd() { this.setData({ pwdFocus: true }); },
  onBlurPwd() { this.setData({ pwdFocus: false }); },
  onFocusCaptcha() { this.setData({ captchaFocus: true }); },
  onBlurCaptcha() { this.setData({ captchaFocus: false }); },

  handleNameInput(e) { this.setData({ realName: e.detail.value.trim() }); },
  handlePwdInput(e) { this.setData({ password: e.detail.value.trim() }); },
  handleCaptchaInput(e) { this.setData({ captchaAnswer: e.detail.value.trim() }); },

  openClassPicker() { 
    this.setData({ showClassPicker: true, classFocus: true }); 
  },
  closeClassPicker() { 
    this.setData({ showClassPicker: false, classFocus: false }); 
  },
  preventClose() {},

  selectClass(e) {
    const selected = e.currentTarget.dataset.class;
    this.setData({ className: selected, showClassPicker: false, classFocus: false });
  },

  goToRegister() {
    wx.navigateTo({ url: '/pages/register/register' });
  },
  goToForgotPwd() {
    wx.navigateTo({ url: '/pages/forgotPassword/forgotPassword' });
  },

  submitLogin() {
    if (this.data.isSubmitting) return;

    const { realName, className, password, captchaId, captchaAnswer } = this.data;
    const deviceId = wx.getStorageSync('deviceId');

    if(!realName || !className || !password || !captchaAnswer) {
      return wx.showToast({ title: '请填写完整的登录信息', icon: 'none' });
    }

    const pwdReg = /^[a-zA-Z0-9_@]{1,20}$/;
    if (!pwdReg.test(password)) {
      return wx.showModal({
        title: '凭证特征不合�?,
        content: '密码长度必须�?0个字符以内，且只能包含英文字母、数字、下划线(_)或特殊符�?@)�?,
        showCancel: false
      });
    }

    this.setData({ isSubmitting: true });
    wx.showLoading({ title: '身份安全核验�?..' });

    request({
      url: '/api/auth/login', 
      method: 'POST',
      data: {
        name: realName,
        class: className,     // 🌟 强效对齐后端标准 class 字段
        clazz: className,     // 🌟 兼容传统 clazz 字段
        password: password,
        device_id: deviceId,  // 🌟 强效对齐后端标准 snake_case device_id 字段
        deviceId: deviceId,   // 🌟 兼容传统驼峰�?deviceId 字段
        captchaId: captchaId,
        captchaAnswer: captchaAnswer
      }
    }).then((backendRes) => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();

      const data = backendRes || {}; 

      /*
      if (data.account_status === 1) {
        return wx.reLaunch({ url: '/pages/pendingApproval/pendingApproval' });
      }
      */

      if (data.account_status === 2) {
        wx.setStorageSync('is_banned', 1);
        wx.setStorageSync('banType', 4);
        return wx.reLaunch({ url: '/pages/banned/banned' });
      }

      const finalAvatarUrl = data.avatar || 'http://127.0.0.1:8080/static/default-avatar.png';
      const finalRealName = data.real_name || realName;
      const finalClassName = data.class_name || className;
      const finalStudentId = data.student_id || '未知学号';

      wx.setStorageSync('userToken', data.token); 
      wx.setStorageSync('userId', data.userId);
      wx.setStorageSync('hasSelectedTracks', true); 
      
      // 💡 智能双向兼容存储：同时写入驼�?(camelCase) 与蛇�?(snake_case) 字段，彻底解决大小写及命名不一致的隐患
      wx.setStorageSync('userInfo', {
        // 驼峰命名 (供大部分核心页面展示读取�?profile.wxml)
        realName: finalRealName, 
        className: finalClassName, 
        studentId: finalStudentId, 
        avatar: finalAvatarUrl,
        userType: data.user_type || data.userType || 0,
        accountStatus: data.account_status || data.accountStatus || 0,

        // 蛇形下划线命�?(兼容老旧分支逻辑)
        real_name: finalRealName, 
        class_name: finalClassName, 
        student_id: finalStudentId, 
        user_type: data.user_type || data.userType || 0,
        account_status: data.account_status || data.accountStatus || 0
      });
      wx.setStorageSync('savedLoginInfo', {
        name: finalRealName, 
        clazz: finalClassName, 
        stuId: finalStudentId, 
        avatar: finalAvatarUrl
      });

      if (data.user_type === 2) {
        wx.setStorageSync('mfa_bound_secret', true);
      }

      wx.switchTab({ url: '/pages/home/home' });

    }).catch((err) => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();
      
      const errorMsg = (err && (err.message || err.msg)) || '';
      
      /*
      if (errorMsg.includes('沙盒') || errorMsg.includes('待审')) {
        return wx.reLaunch({ url: '/pages/pendingApproval/pendingApproval' });
      }
      */

      if (errorMsg) {
        wx.showModal({
          title: '核验不通过',
          content: errorMsg,
          showCancel: false
        });
        if (this.data.isReturningUser && (errorMsg.includes('不存�?) || errorMsg.includes('白名�?))) {
          wx.removeStorageSync('savedLoginInfo');
          this.setData({ isReturningUser: false });
        }
      } else {
        wx.showToast({ title: '网络通信超时或服务异�?, icon: 'none' });
      }
      this.refreshCaptcha();
    });
  }
});