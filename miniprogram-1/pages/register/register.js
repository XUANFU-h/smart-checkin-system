const request = require('../../utils/request.js');

Page({
  data: {
    avatarUrl: '',
    realName: '',
    className: '',
    studentId: '',
    password: '',
    confirmPassword: '',
    captchaId: '',
    captchaAnswer: '',
    captchaImage: '',
    classList: [],
    
    // 悬浮动画焦点状�?
    nameFocus: false,
    classFocus: false,
    stuIdFocus: false,
    pwdFocus: false,
    confirmPwdFocus: false,
    captchaFocus: false,

    isSubmitting: false,
    showClassPicker: false
  },

  onLoad() {
    this.fetchClassList();
    this.refreshCaptcha();
  },

  // === 焦点状态管�?===
  onFocus(e) { this.setData({ [e.currentTarget.dataset.focus]: true }); },
  onBlur(e) { this.setData({ [e.currentTarget.dataset.focus]: false }); },
  
  // === 数据双向绑定 ===
  handleInput(e) {
    this.setData({ [e.currentTarget.dataset.field]: e.detail.value.trim() });
  },

  goToLogin() {
    wx.navigateBack({ delta: 1 }); // 返回登录�?
  },

  // 获取标准班级字典列表
  fetchClassList() {
    request({ url: '/api/auth/classes', method: 'GET' }).then((res) => {
      let list = res || [];
      if (!list.includes('体验账户')) list.push('体验账户');
      this.setData({ classList: list });
    }).catch(() => {
      this.setData({ classList: ['软件G241', '软件G242', '大数据G241', '网络G241', '体验账户'] });
    });
  },

  openClassPicker() { this.setData({ showClassPicker: true, classFocus: true }); },
  closeClassPicker() { this.setData({ showClassPicker: false, classFocus: false }); },
  preventClose() {},
  selectClass(e) {
    this.setData({ className: e.currentTarget.dataset.class, showClassPicker: false, classFocus: false });
  },

  // 💯 终极闭环绝杀：直接读取解包后的核心数据，轰塌验证码空白现�?
  refreshCaptcha() {
    request({ url: '/api/auth/captcha/generate', method: 'GET' }).then((res) => {
      const data = res || {};
      this.setData({ 
        captchaImage: data.captchaImage, 
        captchaId: data.captchaId, 
        captchaAnswer: '' 
      });
    }).catch(() => { wx.showToast({ title: '验证码加载失�?, icon: 'none' }); });
  },

  // 头像直传阿里�?OSS
  onChooseAvatar(e) {
    const { avatarUrl } = e.detail;
    if (!avatarUrl) return;

    wx.showLoading({ title: '安全上传�?..' });

    request({ url: '/api/auth/oss-signature', method: 'GET' }).then((res) => {
      const data = res;
      let finalHost = data.host.trim().replace(/\s+/g, '').replace(/^https?:\/\//, ''); 
      finalHost = 'https://' + finalHost;
      if (finalHost === 'https://') finalHost = 'https://xuanfu-checkin.oss-cn-beijing.aliyuncs.com';

      let dir = data.dir ? data.dir.replace(/\s+/g, '') : 'checkin';
      if (!dir.endsWith('/')) dir += '/';

      const suffix = avatarUrl.substring(avatarUrl.lastIndexOf('.')) || '.png';
      const fileName = dir + Date.now() + '_' + Math.random().toString(36).substr(2) + suffix;

      wx.uploadFile({
        url: finalHost, 
        filePath: avatarUrl,
        name: 'file',
        formData: {
          key: fileName,
          policy: data.policy,
          OSSAccessKeyId: data.OSSAccessKeyId || data.accessid,
          signature: data.signature,
          success_action_status: '200'
        },
        success: (uploadRes) => {
          wx.hideLoading();
          if (uploadRes.statusCode === 200 || uploadRes.statusCode === 204) {
            this.setData({ avatarUrl: finalHost + '/' + fileName });
            wx.showToast({ title: '头像设置成功', icon: 'success' });
          } else {
            wx.showToast({ title: '上传失败', icon: 'none' });
          }
        },
        fail: () => { wx.hideLoading(); wx.showToast({ title: '网络拦截', icon: 'none' }); }
      });
    }).catch(() => { wx.hideLoading(); wx.showToast({ title: '签名获取失败', icon: 'none' }); });
  },

  // 核心前端准入预审查与挂载下一�?
  submitRegister() {
    if (this.data.isSubmitting) return;

    const { realName, className, studentId, password, confirmPassword, avatarUrl, captchaId, captchaAnswer } = this.data;
    
    // 1. 必填项校�?
    if(!realName || !className || !studentId || !password || !confirmPassword || !captchaId || !captchaAnswer) {
      return wx.showToast({ title: '请完善所有注册信�?, icon: 'none' });
    }

    // 2. 两次密码一致性校�?
    if(password !== confirmPassword) {
      return wx.showModal({ title: '密码校验失败', content: '两次输入的密码不一致，请重新输�?, showCancel: false });
    }

    const pwdReg = /^[a-zA-Z0-9_@]{1,20}$/;
    if (!pwdReg.test(password)) {
      return wx.showModal({
        title: '密码凭证不合�?,
        content: '密码长度必须�?0个字符以内，且只能包含英文字母、数字、下划线(_)或特殊符�?@)�?,
        showCancel: false
      });
    }

    // 3. 学号格式校验 
    if (className !== '体验账户') {
      const idReg = /^(19|20|21|22|23|24|25)\d{8}$/; // 只允�?19~25 开头的 10 位数
      if (!idReg.test(studentId)) {
        return wx.showModal({
          title: '学号格式异常',
          content: '学号必须强制�?0位数字，且开头必须是19-25区间的年份数字�?,
          showCancel: false
        });
      }
    }

    let finalAvatarUrl = avatarUrl || 'http://127.0.0.1:8080/static/default-avatar.png';
    let deviceId = wx.getStorageSync('deviceId');
    if (!deviceId) {
      deviceId = 'dev_fingerprint_' + Math.random().toString(36).substr(2, 9).toUpperCase() + Date.now();
      wx.setStorageSync('deviceId', deviceId);
    }

    // 💯 绝杀硬编码：全面去除任何写死的特权判断。是否是管理员权限全由后端去校验 student_roster 底册表落库判定�?
    wx.setStorageSync('temp_reg_payload', {
      name: realName,
      class: className,
      student_id: studentId,
      password: password,
      device_id: deviceId,
      captchaId: captchaId,
      captchaAnswer: captchaAnswer,
      is_admin_reg: false, // 默认不进行前端盲目勾�?
      backdoor: '',        // 彻底清除测试硬编码后门密�?
      avatar_url: finalAvatarUrl
    });

    wx.showToast({ title: '实名特征核验通过', icon: 'success', duration: 800 });
    
    // 转跳到赛道选择页，等待二合一最终一键收网落�?
    setTimeout(() => {
      wx.navigateTo({ url: '/pages/trackSelect/trackSelect?mode=new' });
    }, 800);
  }
});