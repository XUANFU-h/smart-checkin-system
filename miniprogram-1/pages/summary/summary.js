const innerAudioContext = wx.createInnerAudioContext();

Page({
  data: {
    isPlaying: false,
    currentIndex: 0,
    summaryData: null, // 从打卡页面传过来的总体评价数据
    memories: [],      // 后端拉取的照片回忆记�?
    empowermentScore: 0,
    moralScore: 0,
    goodsList: []      // 后端拉取的商�?
  },

  onLoad(options) {
    // 1. 隐藏左上角返回主页按�?
    wx.hideHomeButton();

    // 2. 接收打卡页面传过来的总结评价数据
    if (options.data) {
      try {
        const decodedData = JSON.parse(decodeURIComponent(options.data));
        this.setData({ summaryData: decodedData });
      } catch (e) {
        console.error("解析总结数据失败", e);
      }
    }

    // 3. 加载积分、相册和商品数据
    this.fetchUserInfo(); // 仅拉取积分，因为照片需要后端写专门的聚合接口，如果没有可以先置�?
    this.fetchGoodsList();
    
    // 4. 配置并播�?BGM
    innerAudioContext.src = 'http://127.0.0.1:8080/bgm.mp3'; // 请确保后�?static 目录下有该文�?
    innerAudioContext.loop = true;
    
    // 微信机制要求音频必须在用户交互或特定生命周期触发，加个延时保�?
    setTimeout(() => {
      innerAudioContext.play();
    }, 500);

    innerAudioContext.onPlay(() => { this.setData({ isPlaying: true }); });
    innerAudioContext.onError((err) => { console.error('BGM播放失败', err); });
  },

  onUnload() {
    innerAudioContext.stop(); 
  },

  toggleMusic() {
    if (this.data.isPlaying) {
      innerAudioContext.pause();
      this.setData({ isPlaying: false });
    } else {
      innerAudioContext.play();
      this.setData({ isPlaying: true });
    }
  },

  // 💡 仅拉取最新的赋能分和德育�?
  fetchUserInfo() {
    const userId = wx.getStorageSync('userId');
    wx.request({
      url: `http://127.0.0.1:8080/api/user/info?userId=${userId}`, // 假设你有这个获取用户信息的接�?
      method: 'GET',
      success: (res) => {
        if (res.data.success) {
          const user = res.data.data;
          this.setData({
            empowermentScore: user.empowermentScore || 0,
            moralScore: user.moralScore || 0
          });
        }
      }
    });
  },

  fetchGoodsList() {
    wx.request({
      url: 'http://127.0.0.1:8080/api/shop/list',
      success: (res) => { 
        if (res.data.success) {
          this.setData({ goodsList: res.data.data }); 
        }
      }
    });
  },

  // 商品兑换逻辑
  doExchange(e) {
    const item = e.currentTarget.dataset.item;
    const userId = wx.getStorageSync('userId');
    
    if (this.data.empowermentScore < item.empCost) {
      return wx.showToast({ title: '赋能分不�?, icon: 'error' });
    }

    wx.showModal({
      title: '终极清算确认',
      content: `确认消�?${item.empCost} 赋能分兑换�?{item.name}】？兑换后请截图凭证！`,
      success: (res) => {
        if(res.confirm) {
          wx.showLoading({ title: '正在兑换...' });
          wx.request({
            url: 'http://127.0.0.1:8080/api/shop/exchange',
            method: 'POST',
            data: { userId: userId, itemId: item.id },
            success: (exchangeRes) => {
              wx.hideLoading();
              if (exchangeRes.data.success) {
                wx.showToast({ title: '兑换成功', icon: 'success' });
                this.fetchUserInfo(); // 刷新剩余积分
              } else {
                wx.showToast({ title: exchangeRes.data.message || '兑换失败', icon: 'none' });
              }
            },
            fail: () => {
              wx.hideLoading();
              wx.showToast({ title: '网络异常', icon: 'none' });
            }
          });
        }
      }
    });
  }
})