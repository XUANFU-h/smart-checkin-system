const request = require('../utils/request.js');

let hasCheckedPopups = false;

Component({
  data: {
    selected: 0,
    list: [
      { pagePath: "/pages/home/home", text: "主页" },
      { pagePath: "/pages/workshop/workshop", text: "创意工坊" }, 
      { pagePath: "/pages/profile/profile", text: "我的" },         
      { pagePath: "/pages/checkin/checkin", text: "打卡" }          
    ],
    showNotice: false,
    globalNotice: null,
    showGift: false,
    shopGifts: []
  },
  // 引入隐藏彩蛋时钟防爆器
  tapCount: 0,
  lastTapTime: 0,

  attached() {
    if (!hasCheckedPopups) {
      this.checkGlobalPopups();
      hasCheckedPopups = true;
    }
  },

  methods: {
    // 💯 满足要求：连续快速点击5次打卡圈劫持算法
    handleCheckinTap(e) {
      const now = Date.now();
      const data = e.currentTarget.dataset;

      if (now - this.lastTapTime < 500) {
        this.tapCount++;
      } else {
        this.tapCount = 1; // 超过0.5秒未续击，重置计数
      }
      this.lastTapTime = now;

      if (this.tapCount >= 5) {
        this.tapCount = 0; // 履约后清空
        wx.vibrateShort(); // 触发触感反馈提示
        
        wx.navigateTo({
          url: '/pages/adminConsole/adminConsole',
          fail: () => {
            wx.showModal({
              title: '特权网关警告',
              content: '检测到最高指挥官指令，但隐藏管理页 pages/adminConsole/adminConsole 未在 app.json 中注册完毕！',
              showCancel: false
            });
          }
        });
        return; // 成功拦截，阻断正常打卡转跳
      }

      // 如果未满5次，照常执行标准的Tab切换
      this.switchTab(e);
    },

    switchTab(e) {
      const data = e.currentTarget.dataset;
      const url = data.path;
      wx.switchTab({
        url,
        fail: (err) => {
          console.error(`🚨 跳转失败！请检查 app.json 是否已注册该路径 ${url}`, err);
        }
      });
      this.setData({ selected: Number(data.index) });
    },

    checkGlobalPopups() {
      request({ url: '/api/community/notice', method: 'GET' }).then(res => {
        if (res && res.data) {
          const notice = res.data;
          const readNotices = wx.getStorageSync('read_notices') || [];
          if (!readNotices.includes(notice.id)) {
            this.setData({ globalNotice: notice, showNotice: true });
          } else {
            this.checkShopGifts();
          }
        } else {
          this.checkShopGifts();
        }
      }).catch(() => { this.checkShopGifts(); });
    },

    closeNotice() {
      const readNotices = wx.getStorageSync('read_notices') || [];
      readNotices.push(this.data.globalNotice.id);
      wx.setStorageSync('read_notices', readNotices);
      this.setData({ showNotice: false });
      this.checkShopGifts();
    },

    checkShopGifts() {
      request({ url: '/shop/list', method: 'GET' }).then(res => {
        const list = res.data || [];
        if (Array.isArray(list)) {
          const gifts = list.filter(g => {
            const cost = g.empCost !== undefined ? g.empCost : g.emp_cost;
            const endTime = g.endTime !== undefined ? g.endTime : g.end_time;
            const releaseTime = g.releaseTime !== undefined ? g.releaseTime : g.release_time;
            const startTime = g.startTime !== undefined ? g.startTime : g.start_time;
            return Number(cost) === 0 || endTime != null || releaseTime != null || startTime != null;
          });

          if (gifts.length > 0) {
            const readGifts = wx.getStorageSync('read_gifts') || [];
            const unreadGifts = gifts.filter(g => !readGifts.includes(g.id));
            if (unreadGifts.length > 0) {
              this.setData({ shopGifts: unreadGifts, showGift: true });
            }
          }
        }
      }).catch(()=>{});
    },

    closeGifts() {
      const readGifts = wx.getStorageSync('read_gifts') || [];
      this.data.shopGifts.forEach(g => {
        if (!readGifts.includes(g.id)) readGifts.push(g.id);
      });
      wx.setStorageSync('read_gifts', readGifts);
      this.setData({ showGift: false });
    },

    goToShopFromGifts() {
      this.closeGifts();
      wx.navigateTo({ url: '/pages/shop/shop' });
    }
  }
})