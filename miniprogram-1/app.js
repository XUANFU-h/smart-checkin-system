// app.js
const config = require('./config.js');
const request = require('./utils/request.js');

App({
  globalData: {
    userInfo: null,
    socketTask: null,
    isSocketConnected: false,
    reconnectTimer: null
  },

  onLaunch() {
    this.checkGlobalStatus();
  },

  onShow() {
    this.checkGlobalStatus();

    if (!this.globalData.isSocketConnected) {
      this.connectWebSocket();
    }
  },

  onHide() {
    if (this.globalData.isSocketConnected && this.globalData.socketTask) {
      this.globalData.socketTask.close();
    }
  },

  checkGlobalStatus() {
    const pages = getCurrentPages();
    const currentRoute = pages.length > 0 ? pages[pages.length - 1].route : '';

    if (wx.getStorageSync('is_banned') === 1) {
      if (currentRoute !== 'pages/banned/banned') {
        wx.reLaunch({ url: '/pages/banned/banned' });
      }
      return true;
    }

    const isFinished = wx.getStorageSync('is_finished');
    if (isFinished === 1) {
      if (currentRoute !== 'pages/summary/summary') {
        wx.reLaunch({ url: '/pages/summary/summary' });
      }
      return true;
    }
    return false;
  },

  connectWebSocket() {
    const userId = wx.getStorageSync('userId');
    const token = wx.getStorageSync('userToken'); 
    if (!userId || !token) return;

    this.globalData.socketTask = wx.connectSocket({
      url: `${config.wsUrl}?userId=${userId}&token=${token}`,
      success: () => { console.log("WebSocket 握手请求已发送"); }
    });

    this.globalData.socketTask.onOpen(() => {
      console.log("🚀 WebSocket 已连接，查岗雷达进入长连接模式");
      this.globalData.isSocketConnected = true;
      if (this.globalData.reconnectTimer) {
        clearTimeout(this.globalData.reconnectTimer);
        this.globalData.reconnectTimer = null;
      }
    });

    this.globalData.socketTask.onClose(() => {
      console.log("⚠️ WebSocket 已断开");
      this.globalData.isSocketConnected = false;
      
      if (this.globalData.reconnectTimer) {
        clearTimeout(this.globalData.reconnectTimer);
      }
      
      this.globalData.reconnectTimer = setTimeout(() => {
        this.connectWebSocket();
      }, 5000); 
    });

    this.globalData.socketTask.onError((err) => {
      console.error("WebSocket 错误", err);
    });

    this.globalData.socketTask.onMessage((res) => {
      try {
        const msg = JSON.parse(res.data);
        console.log("收到服务器实时指令:", msg);

        if (msg.type === 'KICK_OUT') {
          const payload = msg.data || {};
          
          // 1. 核心绝杀：管理员重置密码强驱，清除会话并强制跳转到重设密码页
          if (payload.kickType === 'RESET_PASSWORD' || msg.kickType === 'RESET_PASSWORD') {
            const rToken = payload.resetToken || msg.resetToken || '';
            wx.clearStorageSync();
            wx.showModal({
              title: '密码重置通知',
              content: payload.reason || msg.reason || '您的登录密码已被管理员重置，请立即重设密码！',
              showCancel: false,
              confirmText: '去重设',
              success: () => {
                wx.reLaunch({ url: `/pages/resetPassword/resetPassword?token=${rToken}` });
              }
            });
            return;
          }

          // 2. 强制下线强驱，清退 Token 并返回登录页
          if (payload.kickType === 'FORCE_LOGOUT' || msg.kickType === 'FORCE_LOGOUT') {
            wx.clearStorageSync();
            wx.showModal({
              title: '强制下线通知',
              content: payload.reason || msg.reason || '您的账号已被管理员强制踢下线，请重新登录！',
              showCancel: false,
              success: () => {
                wx.reLaunch({ url: '/pages/login/login' });
              }
            });
            return;
          }

          // 3. 封禁逻辑 (高兼容性提取，支持 payload/msg 双模)
          const bType = payload.banType !== undefined ? payload.banType : (msg.banType !== undefined ? msg.banType : 3);
          const bEndTime = payload.banEndTime || msg.banEndTime || 0;
          const bReason = payload.reason || msg.reason || '';

          wx.setStorageSync('is_banned', 1);
          wx.setStorageSync('banType', bType);

          if (bType === 1 && !wx.getStorageSync('banEndTime')) {
            wx.setStorageSync('banEndTime', new Date().getTime() + 24 * 3600 * 1000);
          } else if (bEndTime) {
            wx.setStorageSync('banEndTime', bEndTime);
          }

          if (bReason) wx.setStorageSync('banReason', bReason);

          const pages = getCurrentPages();
          if (pages.length > 0 && pages[pages.length - 1].route !== 'pages/banned/banned') {
            wx.reLaunch({ url: '/pages/banned/banned' });
          }
        }
        else if (msg.type === 'RESTORE') {
          if (wx.getStorageSync('is_banned') === 1) {
            wx.removeStorageSync('is_banned');
            wx.removeStorageSync('banType');
            wx.removeStorageSync('banEndTime');
            wx.removeStorageSync('banReason');
            wx.reLaunch({ url: '/pages/home/home' });
          }
        }
        // ✅ 核心新增：监听管理员审批通过指令
        else if (msg.type === 'AUTH_APPROVED') {
          wx.showModal({
            title: '🎉 审批已通过',
            content: '管理员已放行您的请求，请重新进入系统',
            showCancel: false,
            confirmText: '立即进入',
            success: () => {
              // 审批通过后，强制退回登录页，让用户走一遍标准登录流程获取最新的权限 Token
              wx.reLaunch({ url: '/pages/login/login' });
            }
          });
        }
        else if (msg.type === 'NOTIFY') {
          wx.showModal({
            title: msg.title || '🎁 系统奖励到账通知',
            content: msg.content,
            confirmText: '开心收下',
            showCancel: false,
            success: () => {
              request({
                url: `${config.baseUrl}/user/notifications/read`,
                method: 'POST',
                data: { notifyId: msg.notifyId }
              }).catch(err => {
                console.error("上报通知已读失败:", err);
              });
            }
          });
        }
        else if (msg.type === 'RECORD_DELETED') {
          if (msg.trackId) {
            wx.removeStorageSync(`lastSubmit_${msg.trackId}`);
          }
          wx.showModal({
            title: msg.title,
            content: msg.content,
            showCancel: false,
            confirmText: '我已知晓',
            confirmColor: '#F56C6C',
            success: () => {
              const pages = getCurrentPages();
              for (let i = 0; i < pages.length; i++) {
                if (pages[i].route === 'pages/checkin/checkin' && typeof pages[i].fetchRecordsForCurrentTrack === 'function') {
                  pages[i].fetchRecordsForCurrentTrack();
                  break;
                }
              }
            }
          });
        }
      } catch (e) {
        console.error("解析 WebSocket 消息失败", e);
      }
    });
  }
});