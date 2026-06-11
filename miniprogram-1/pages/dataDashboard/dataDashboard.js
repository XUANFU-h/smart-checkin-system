const request = require('../../utils/request.js');

Page({
  data: {
    userInfo: {},
    myTracks: [],
    currentTab: 'global', // 'global' 全系 | 'class' 班级 | '赛道ID'
    rankData: []
  },

  onLoad(options) {
    if (options.tab) {
      this.setData({ currentTab: options.tab === 'track' ? 'global' : options.tab });
    }
    // 拉取缓存在本地的用户参赛赛道
    const tracks = wx.getStorageSync('myTracks') || [];
    this.setData({ myTracks: tracks });
  },

  onShow() {
    this.fetchUserScore();
    this.fetchLeaderboard();
  },

  // 1. 获取个人最新的资产分值
  fetchUserScore() {
    request({
      url: '/user/score-info', // 复用我们给“我的”页面写的聚合接口
      method: 'GET'
    }).then(res => {
      if (res && res.success && res.data) {
        this.setData({ userInfo: res.data });
      }
    }).catch(()=>{});
  },

  // 2. 切换榜单
  switchTab(e) {
    const tab = e.currentTarget.dataset.tab;
    if (tab === this.data.currentTab) return;
    this.setData({ currentTab: tab, rankData: [] });
    this.fetchLeaderboard();
  },

  // 3. 拉取榜单数据
  fetchLeaderboard() {
    wx.showLoading({ title: '拉取榜单中...' });
    
    // 我们将统一请求后端的专门的大盘接口，传入当前的维度
    request({
      url: `/track/leaderboard?type=${this.data.currentTab}`,
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      if (res && res.success && res.data) {
        const rawList = res.data || [];
        
        // 核心脱敏函数：如果是班级榜则显示全名，否则匿名
        const processName = (name) => {
          if (this.data.currentTab === 'class') return name; // 班级内不脱敏
          if (!name) return '匿*';
          if (name.length <= 2) return name.substring(0, 1) + '*';
          return name.substring(0, 1) + '*' + name.substring(name.length - 1);
        };

        const processedList = rawList.map(item => ({
           ...item,
           display_name: processName(item.real_name || item.nickname)
        }));
        
        this.setData({ rankData: processedList });
      }
    }).catch(() => { wx.hideLoading(); });
  }
})