const request = require('../../utils/request.js');

Page({
  data: {
    banType: 'permanent', 
    banReason: '',
    titleStr: '账号异常拦截',
    countdownStr: ''
  },
  timer: null,

  onLoad() {
    if (wx.canIUse('hideHomeButton')) {
      try { wx.hideHomeButton(); } catch (e) {}
    }

    let rawBanType = wx.getStorageSync('banType') || 3;
    let banReason = wx.getStorageSync('banReason') || '';
    let endTime = wx.getStorageSync('banEndTime') || 0; 
    
    let parsedType = 'permanent';
    let titleStr = '账号异常拦截';

    if (rawBanType === 1 || rawBanType === 2) {
      parsedType = 'temporary';
    } else if (rawBanType === 3) {
      parsedType = 'permanent';
    } else if (rawBanType === 4) {
      parsedType = 'blacklisted';
      titleStr = '系统拒绝访问';
    } else if (rawBanType === 'device_locked') {
      parsedType = 'device_locked';
      titleStr = '连接安全阻断';
    } else if (rawBanType === 'cancelling') {
      parsedType = 'cancelling';
      titleStr = '账号休眠中';
    }

    if (!banReason && parsedType !== 'cancelling') {
       banReason = '系统风控大脑检测到您的账号存在违规操作。';
    }

    this.setData({ banType: parsedType, banReason, titleStr });

    if (parsedType === 'temporary' && endTime > 0) {
      this.startCountdown(endTime);
    }
  },

  onShow() {
    const userId = wx.getStorageSync('userId');
    if (userId && this.data.banType === 'cancelling') {
      request({
        url: `/user/info`, // 💡 修正：一般查信息的接口是 /user/info 而不是 /user/:id
        method: 'GET'
      }).then(res => {
        if (res && res.success) {
          this.clearBanCacheAndGoHome();
        }
      }).catch(() => {});
    }
  },

  startCountdown(targetTime) {
    if (this.timer) clearInterval(this.timer);
    this.timer = setInterval(() => {
      const now = new Date().getTime();
      let diff = Math.floor((targetTime - now) / 1000);

      if (diff <= 0) {
        clearInterval(this.timer);
        this.setData({ countdownStr: '即将解封...' });
        this.clearBanCacheAndGoHome();
        return;
      }

      const d = Math.floor(diff / (24 * 3600));
      const h = String(Math.floor((diff % (24 * 3600)) / 3600)).padStart(2, '0');
      const m = String(Math.floor((diff % 3600) / 60)).padStart(2, '0');
      const s = String(diff % 60).padStart(2, '0');

      this.setData({ countdownStr: d > 0 ? `${d}天 ${h}:${m}:${s}` : `${h}:${m}:${s}` });
    }, 1000);
  },

  onHide() {
    // 💡 优化：页面隐藏时暂停定时器，防止后台耗电
    if (this.timer) clearInterval(this.timer);
  },

  onUnload() {
    if (this.timer) clearInterval(this.timer);
  },

  clearBanCacheAndGoHome() {
     wx.removeStorageSync('is_banned');
     wx.removeStorageSync('banType');
     wx.removeStorageSync('banReason');
     wx.removeStorageSync('banEndTime');
     wx.reLaunch({ url: '/pages/home/home' });
  },

  handleLogout() {
    wx.showModal({
      title: '确认退出',
      content: '退出后将返回登录页面，是否继续？',
      success: (res) => {
        if (res.confirm) {
          // 💡 优化：一键彻底清空缓存最安全
          wx.clearStorageSync();
          wx.reLaunch({ url: '/pages/login/login' });
        }
      }
    });
  },

  handleWithdrawCancel() {
    wx.showModal({
      title: '撤销注销',
      content: '撤销后账号将恢复正常使用，是否确认？',
      success: (modRes) => {
        if(modRes.confirm) {
          wx.showLoading({ title: '系统恢复中...' });
          request({
             url: '/user/request-cancel', // 💡 请确保后端有处理撤销注销的通道 (或者用 /user/reject-cancel 变种)
             method: 'POST',
             data: { action: 'revoke' } 
          }).then(res => {
             wx.hideLoading();
             if (res && res.success) {
               wx.showToast({ title: '已成功撤销', icon: 'success' });
               setTimeout(() => { this.clearBanCacheAndGoHome(); }, 1500);
             } else {
               wx.showToast({ title: res.message || '撤销失败', icon: 'none' });
             }
          }).catch(() => { wx.hideLoading(); });
        }
      }
    });
  }
})