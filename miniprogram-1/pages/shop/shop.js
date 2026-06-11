const request = require('../../utils/request.js');

Page({
  data: {
    empowermentScore: 0,
    rawGoodsList: [], 
    regularGoods: [], 
    timedGoods: []    
  },

  onShow() {
    this.fetchGoodsList();
    this.syncScore();

    if (this.shopTimer) clearInterval(this.shopTimer);
    this.shopTimer = setInterval(() => {
      this.updateGoodsStatus();
    }, 1000);
  },

  onHide() {
    if (this.shopTimer) {
      clearInterval(this.shopTimer);
      this.shopTimer = null;
    }
  },

  onUnload() {
    if (this.shopTimer) {
      clearInterval(this.shopTimer);
      this.shopTimer = null;
    }
  },

  fetchGoodsList() {
    request({
      url: '/shop/list',
      method: 'GET'
    }).then(res => {
      if (res && res.success && res.data) {
        this.setData({ rawGoodsList: res.data }, () => {
          this.updateGoodsStatus();
        });
      }
    }).catch(() => {});
  },

  syncScore() {
    request({
      url: '/user/score-info', 
      method: 'GET'
    }).then(res => {
      if (res && res.success && res.data) {
        const score = res.data.empowermentScore !== undefined ? res.data.empowermentScore : res.data.empowerment_score;
        this.setData({ empowermentScore: score || 0 }, () => {
          this.updateGoodsStatus();
        });
      }
    }).catch(() => {});
  },

  updateGoodsStatus() {
    const currentScore = Number(this.data.empowermentScore);
    const now = new Date().getTime();
    
    let regular = [];
    let timed = [];

    this.data.rawGoodsList.forEach(rawItem => {
      const empCost = rawItem.empCost !== undefined ? rawItem.empCost : rawItem.emp_cost;
      const rewardValue = rawItem.rewardValue !== undefined ? rawItem.rewardValue : rawItem.reward_value;
      let releaseTime = rawItem.releaseTime || rawItem.release_time || rawItem.start_time || rawItem.startTime;
      
      // 💡 读取后端传来的限购数据
      let userLimit = rawItem.userLimit !== undefined ? rawItem.userLimit : rawItem.user_limit;
      let userBought = rawItem.userBought !== undefined ? rawItem.userBought : rawItem.user_bought;

      const item = {
        ...rawItem,
        empCost: Number(empCost),
        rewardValue: rewardValue,
        releaseTime: releaseTime,
        userLimit: Number(userLimit || 0),
        userBought: Number(userBought || 0)
      };

      let isReleased = true;
      if (item.releaseTime) {
        let targetTime = new Date(item.releaseTime).getTime();
        if (isNaN(targetTime)) {
          let timeStr = String(item.releaseTime).replace('T', ' ').split('.')[0];
          targetTime = new Date(timeStr.replace(/-/g, '/')).getTime();
        }
        isReleased = now >= targetTime;
      }

      let disabled = false;
      let btnText = item.empCost === 0 ? '免费领取' : '兑换'; // 0积分商品改文案

      // 💡 风控核心逻辑顺序
      if (item.userLimit > 0 && item.userBought >= item.userLimit) {
        disabled = true;
        btnText = item.empCost === 0 ? '已领取' : '已达上限';
      } else if (item.stock === 0) {
        disabled = true;
        btnText = '已售罄';
      } else if (!isReleased) {
        disabled = true;
        btnText = '未开售';
      } else if (currentScore < item.empCost) {
        disabled = true;
        btnText = '分不足';
      }

      const processedItem = { ...item, disabled, btnText, isReleased };

      if (item.releaseTime) {
        timed.push(processedItem);
      } else {
        regular.push(processedItem);
      }
    });

    this.setData({ 
      regularGoods: regular,
      timedGoods: timed
    });
  },

  doExchange(e) {
    const item = e.currentTarget.dataset.item;
    if (item.disabled) return;
    if (this.isExchanging) return;

    const costScore = Number(item.empCost);
    
    // 💡 智能弹窗提醒
    let contentStr = costScore === 0 
      ? `确认免费领取【${item.name}】吗？` 
      : `将消耗 ${costScore} 赋能分兑换【${item.name}】，是否继续？`;

    wx.showModal({
      title: costScore === 0 ? '专属福利' : '确认兑换',
      content: contentStr,
      success: (res) => {
        if(res.confirm) {
          this.isExchanging = true; 
          wx.showLoading({ title: costScore === 0 ? '领取中...' : '拼命抢购中...', mask: true });
          
          request({
            url: '/shop/exchange',
            method: 'POST',
            data: { itemId: item.id } 
          }).then(exchangeRes => {
            wx.hideLoading();
            setTimeout(() => {
              if (exchangeRes && exchangeRes.success) {
                wx.showToast({ title: costScore === 0 ? '领取成功' : '兑换成功', icon: 'success' });
                this.syncScore();      
                this.fetchGoodsList(); // 会重新向后端要数据，此时后端计算的 userBought 就会+1
              } else {
                wx.showToast({ title: (exchangeRes && exchangeRes.message) || '手慢无，操作失败', icon: 'none' });
                this.fetchGoodsList(); 
              }
              this.isExchanging = false; 
            }, 300);
          }).catch(() => {
            wx.hideLoading();
            setTimeout(() => {
              wx.showToast({ title: '网络拥堵，请重试', icon: 'none' });
              this.isExchanging = false; 
            }, 300);
          });
        }
      }
    });
  }
})