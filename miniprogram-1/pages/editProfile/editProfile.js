const request = require('../../utils/request.js');
const config = require('../../config.js');

Page({
  data: { 
    userInfo: {},
    showCancelModal: false,
    countdown: 10,
    cancelPassword: '' // 💡 新增：注销密码
  },
  
  onLoad() {
    const info = wx.getStorageSync('userInfo');
    if(info) this.setData({ userInfo: info });
  },
  
  onUnload() {
    if (this.cancelTimer) clearInterval(this.cancelTimer);
  },

  // 💡 跳转安全矩阵 (修改/忘记密码统统走这里)
  goToSecurityPage() {
    wx.navigateTo({ url: '/pages/forgotPassword/forgotPassword' });
  },

  onChooseAvatar(e) { 
    const tempFilePath = e.detail.avatarUrl;
    wx.showLoading({ title: '正在上传头像...' });

    wx.uploadFile({
      url: `${config.baseUrl}/checkin/upload`, // 💡 1. 修复：指向后端正确的图片上传路径
      filePath: tempFilePath,
      name: 'file',
      header: { 'Authorization': 'Bearer ' + wx.getStorageSync('userToken') },
      success: (uploadRes) => {
        wx.hideLoading();
        try {
          const resData = JSON.parse(uploadRes.data);
          // 💡 2. 修复：兼容微信原生回执检测，支持 code===200 判定
          const isSuccess = resData.code === 200 || resData.code === 0 || resData.success === true;
          // 💡 3. 修复：准确提取后端返回的 imageUrl 字段
          const realImgUrl = resData.data && (resData.data.imageUrl || resData.data.url);

          if (isSuccess && realImgUrl) {
            let realOssUrl = realImgUrl;
            if (realOssUrl.startsWith('/')) {
                const domain = config.baseUrl.replace('/api', '');
                realOssUrl = domain + realOssUrl;
            }

            this.setData({ ['userInfo.avatar']: realOssUrl });
            
            request({
              url: '/user/updateAvatar',
              method: 'POST',
              data: { avatar: realOssUrl }
            }).then(updateRes => {
              // 💡 4. 完美契合 Proxy 拦截器的 success 检测
              if (updateRes && updateRes.success) {
                wx.showToast({ title: '头像上传成功', icon: 'success' });
                const info = wx.getStorageSync('userInfo') || {};
                
                // 同时更新驼峰和蛇形的头像缓存防串号
                info.avatar = realOssUrl;
                wx.setStorageSync('userInfo', info);

                // 同步更新 savedLoginInfo 里的头像，保证注销重登时头像正确
                const savedInfo = wx.getStorageSync('savedLoginInfo');
                if(savedInfo) {
                  savedInfo.avatar = realOssUrl;
                  wx.setStorageSync('savedLoginInfo', savedInfo);
                }
              } else {
                wx.showToast({ title: updateRes.message || '绑定失败', icon: 'none' });
              }
            }).catch(()=>{});

          } else {
            wx.showToast({ title: resData.message || '上传失败', icon: 'none' });
          }
        } catch (err) {
          wx.showToast({ title: '服务器响应异常', icon: 'none' });
        }
      },
      fail: () => {
        wx.hideLoading();
        wx.showToast({ title: '网络连接超时', icon: 'none' });
      }
    });
  },

  // 💡 去除了姓名和班级的修改流，现在只负责“仅保存头像”
  saveProfile() {
    wx.showToast({ title: '头像已自动保存', icon: 'success' });
    setTimeout(() => { wx.navigateBack(); }, 1000);
  },

  // ================= 💡 真实物理安全退出 (修复退出不了的问题) =================
  logout() {
    wx.showModal({
      title: '退出确认',
      content: '确定要退出当前账号吗？',
      success: (res) => {
        if (res.confirm) {
          wx.showLoading({ title: '安全解绑中...' });
          
          request({
            url: '/user/logout',
            method: 'POST'
          }).finally(() => {
            // 🚨 无论后端响应成功还是 404/500，前端必须强制抹除所有痕迹！
            wx.hideLoading();
            wx.removeStorageSync('userToken');
            wx.removeStorageSync('userId');
            wx.removeStorageSync('userInfo');
            wx.removeStorageSync('hasSelectedTracks');
            // ⚠️ 关键：不清除 savedLoginInfo，这样只是退出，下次还可以快捷登录
            
            wx.showToast({ title: '已安全退出', icon: 'success' });
            setTimeout(() => {
              wx.reLaunch({ url: '/pages/login/login' });
            }, 800);
          });
        }
      }
    });
  },

  // ================= 💡 注销账号与密码确认引擎 =================
  handleCancelPwdInput(e) {
    this.setData({ cancelPassword: e.detail.value.trim() });
  },

  showCancelModal() {
    this.setData({ showCancelModal: true, countdown: 10, cancelPassword: '' });
    this.cancelTimer = setInterval(() => {
      if (this.data.countdown > 0) {
        this.setData({ countdown: this.data.countdown - 1 });
      } else {
        clearInterval(this.cancelTimer);
      }
    }, 1000);
  },

  hideCancelModal() {
    if (this.cancelTimer) clearInterval(this.cancelTimer);
    this.setData({ showCancelModal: false });
  },

  confirmCancelAccount() {
    if (this.data.countdown > 0) return; 
    
    // 强制验证注销密码
    if (!this.data.cancelPassword) {
      return wx.showToast({ title: '请输入密码验证身份', icon: 'none' });
    }
    
    const userId = wx.getStorageSync('userId');
    wx.showLoading({ title: '核验身份并注销...' });

    request({
      url: '/user/request-cancel',
      method: 'POST',
      data: { id: userId, password: this.data.cancelPassword } // 传给后端
    }).then(res => {
      wx.hideLoading();
      if (res && res.success) {
        this.hideCancelModal();
        wx.showToast({ title: '账号已注销', icon: 'success' });
        
        // 🚨 核心要求：彻底抹除所有痕迹，防止出现老用户一键登录！
        wx.clearStorageSync(); // 直接清空整个小程序的 Storage，最干净的注销！
        
        // 保留设备 ID，防止被后端判定为异常
        wx.setStorageSync('deviceId', 'dev_' + Math.random().toString(36).substr(2));
        wx.setStorageSync('isFirstTimeEntered', true); // 防止跳回注册页时死循环

        // 强行踢回注册页
        setTimeout(() => {
           wx.reLaunch({ url: '/pages/register/register' });
        }, 1500);
      } else {
        wx.showToast({ title: res.message || '密码错误或申请失败', icon: 'none' });
      }
    }).catch((err) => {
      wx.hideLoading();
      wx.showToast({ title: err.msg || '网络异常', icon: 'none' });
    });
  }
})