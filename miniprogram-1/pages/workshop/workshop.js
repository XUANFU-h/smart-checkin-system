const request = require('../../utils/request.js');

Page({
  data: {
    num1: '00', num2: '00', num3: '00',
    spinCount: 0,
    hasSpinned: false,
    isSpinning: false,
    isLocked: false,
    hasFiveDayStreak: false, // 兼容字段
    isFreeEligible: false,
    trackCount: 0,
    todayCheckedInCount: 0,
    
    // 开奖状态
    isDrawTime: false,
    drawTimeStr: '周六晚 22:00',
    drawRevealed: false,
    drawPrizeName: '',

    luckyRecords: [],
    historyRecords: [], // 往期荣誉中奖归档列表
    spinCost: 100,      // 付费参与所需赋能分消耗设定
    activeStatus: 'closed', // 活动状态
    workshopActiveStatus: 'open', // 创意工坊开启状态

    // 即时彩蛋
    showEasterEgg: false
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) this.getTabBar().setData({ selected: 1 });
    this.fetchLottoStatus();
    this.fetchLottoHistory();
  },

  // 拉取用户的当前状态
  fetchLottoStatus() {
    request({ url: '/game/slot/status', method: 'GET' }).then(res => {
      const d = res || {};
      const tCount = d.trackCount || 0;
      const cCount = d.todayCheckedInCount || 0;
      const freeEligible = tCount > 0 && cCount >= tCount;
      const spins = d.remainSpins !== undefined ? 10 - d.remainSpins : 0;

      this.setData({
        trackCount: tCount,
        todayCheckedInCount: cCount,
        isFreeEligible: freeEligible,
        hasFiveDayStreak: freeEligible, // 兼容层字段
        spinCount: spins,
        isLocked: d.isLocked || false,
        isDrawTime: d.isDrawTime || false,
        drawTimeStr: d.drawTimeStr || '周六晚 22:00',
        luckyRecords: d.luckyRecords || [],
        hasSpinned: spins > 0,
        spinCost: d.spinCost || 100, // 同步动态积分划扣值
        activeStatus: d.activeStatus || 'closed', // 💡 管理员手动大乐透开启状态
        workshopActiveStatus: d.workshopActiveStatus || 'open' // 💡 创意工坊开启状态
      });

      // 如果用户之前已经锁定了号码，恢复显示
      if (d.currentNumbers && d.currentNumbers.length === 3) {
        this.setData({
          num1: d.currentNumbers[0],
          num2: d.currentNumbers[1],
          num3: d.currentNumbers[2],
          hasSpinned: true
        });
      }
    }).catch(()=>{});
  },

  // 开始摇号 (前端只负责动画，数据找后端要)
  startSpin(e) {
    const payType = e.currentTarget.dataset.type; // "free" or "points"
    if (this.data.isSpinning || this.data.isLocked || this.data.spinCount >= 10) return;

    if (payType === 'free') {
      // 💡 只有在第一次入场刷新时，才限制今日打卡是否完成！后续刷新直接免费放行！
      if (this.data.spinCount === 0 && (this.data.todayCheckedInCount < this.data.trackCount || this.data.trackCount === 0)) {
        wx.showToast({ title: '今日打卡尚未全部完成！', icon: 'none' });
        return;
      }
      this.executeSpinRequest('free');
    } else {
      wx.showModal({
        title: '付费刷新确认',
        content: `本次摇号将扣除 ${this.data.spinCost} 赋能分，确定继续吗？`,
        success: (res) => {
          if (res.confirm) {
            this.executeSpinRequest('points');
          }
        }
      });
    }
  },

  executeSpinRequest(payType) {
    wx.showLoading({ title: '摇号准备中', mask: true });
    
    // 向后端发请求，扣费并索要随机结果
    request({ 
      url: '/game/slot/spin', 
      method: 'POST',
      data: { payType: payType }
    }).then(res => {
      wx.hideLoading();
      if (res && res.success && res.data) {
        this.playSpinAnimation(res.data.numbers, res.data.isEasterEgg);
        this.setData({ 
          spinCount: res.data.spinCount || (this.data.spinCount + 1),
          isLocked: res.data.isLocked || false,
          hasSpinned: true
        });
      } else {
        wx.showToast({ title: (res && res.message) || '余额不足或发生异常', icon: 'none' });
      }
    }).catch((err) => {
      wx.hideLoading();
      wx.showToast({ title: (err && (err.message || err.msg)) || '请求失败', icon: 'none' });
    });
  },

  // 超高速数字滚动动画引擎
  playSpinAnimation(targetNums, isEasterEgg) {
    this.setData({ isSpinning: true });
    
    let ticks = 0;
    const maxTicks = 20; // 动画时长 2秒 (20 * 100ms)
    
    this.spinTimer = setInterval(() => {
      ticks++;
      // 随机生成假数字做视觉残留
      this.setData({
        num1: Math.floor(Math.random() * 100).toString().padStart(2, '0'),
        num2: Math.floor(Math.random() * 100).toString().padStart(2, '0'),
        num3: Math.floor(Math.random() * 100).toString().padStart(2, '0')
      });

      // 动画结束，锁定到后端给的真实数据
      if (ticks >= maxTicks) {
        clearInterval(this.spinTimer);
        this.setData({
          num1: targetNums[0],
          num2: targetNums[1],
          num3: targetNums[2],
          isSpinning: false
        });

        // 如果触发了内幕隐藏奖励，弹彩蛋
        if (isEasterEgg) {
          setTimeout(() => { this.setData({ showEasterEgg: true }); }, 500);
        }

        // 刷新老虎机状态，防止延迟或状态错同步
        this.fetchLottoStatus();
      }
    }, 100);
  },

  // 锁定当前号码，存入数据库凭证库
  lockNumbers() {
    if (this.data.isSpinning) return;
    const currentNumStr = `${this.data.num1}-${this.data.num2}-${this.data.num3}`;
    
    wx.showModal({
      title: '神圣契约封存',
      content: `确定锁定号码【${currentNumStr}】参与本期大乐透开奖吗？锁定后无法修改。`,
      success: (res) => {
        if (res.confirm) {
          wx.showLoading({ title: '封存中' });
          request({ 
            url: '/game/slot/lock', 
            method: 'POST'
          }).then(res2 => {
            wx.hideLoading();
            if(res2.success) {
              wx.showToast({ title: '锁定成功！', icon: 'success' });
              this.setData({ isLocked: true });
              this.fetchLottoStatus(); // 锁定后刷新状态同步
            } else {
              wx.showToast({ title: res2.message || '锁定失败', icon: 'none' });
            }
          }).catch((err) => {
            wx.hideLoading();
            wx.showToast({ title: (err && (err.message || err.msg)) || '锁定失败', icon: 'none' });
          });
        }
      }
    });
  },

  // ================= 真正的大乐透开奖动画 =================
  playDrawAnimation() {
    wx.showLoading({ title: '获取开奖结果', mask: true });
    // 找后端要本期的中奖号码和自己的奖品
    request({ url: '/game/lottery/draw-result', method: 'GET' }).then(res => {
      wx.hideLoading();
      const d = res || {};
      if (d.winningNumbers && d.winningNumbers.length === 3) {
        this.setData({ drawPrizeName: d.prizeName || '' });
        
        // 播放开奖动画：慢慢翻出中奖数字，覆盖掉用户原来的数字
        this.animateDrawReveal(d.winningNumbers);
      } else {
        wx.showToast({ title: '本期开奖数据异常，请重试', icon: 'none' });
      }
    }).catch((err) => {
      wx.hideLoading();
      wx.showToast({ title: (err && (err.message || err.msg)) || '本期未到开奖时间，或您未参与！', icon: 'none' });
    });
  },

  animateDrawReveal(winNums) {
    this.setData({ isSpinning: true }); // 复用模糊发光效果
    
    // 依次揭晓三个中奖数字作为华丽前奏
    setTimeout(() => { this.setData({ num1: winNums[0] }); }, 800);
    setTimeout(() => { this.setData({ num2: winNums[1] }); }, 1600);
    setTimeout(() => { 
      // 💯 核心大改写：大乐透揭晓后，原本的三组数字转为显示获奖级别、积分数值和积分类型
      const prizeName = this.data.drawPrizeName || '';
      let level = '未中大奖';
      let points = '+0';
      let type = '谢谢参与';

      if (prizeName) {
        if (prizeName.includes('一等奖')) {
          level = '一等奖';
        } else if (prizeName.includes('二等奖')) {
          level = '二等奖';
        } else if (prizeName.includes('三等奖')) {
          level = '三等奖';
        } else if (prizeName.includes('参与奖') || prizeName.includes('参与勉励')) {
          level = '参与奖';
        } else {
          level = '恭喜中奖';
        }

        // 正则高保真提取积分数值 (如 "500 赋能分", "10 赋能分 + 5 德育分")
        const numMatches = prizeName.match(/\d+/g);
        if (numMatches) {
          if (numMatches.length === 2) {
            points = `+${numMatches[0]}/${numMatches[1]}`;
          } else if (numMatches.length === 1) {
            points = `+${numMatches[0]}`;
          }
        } else {
          points = '+0';
        }

        // 解析分值类型
        if (prizeName.includes('赋能分') && prizeName.includes('德育分')) {
          type = '赋能+德育';
        } else if (prizeName.includes('赋能分')) {
          type = '仅赋能分';
        } else if (prizeName.includes('德育分')) {
          type = '仅德育分';
        } else {
          type = '积分奖励';
        }
      } else {
        if (!this.data.isLocked && this.data.spinCount === 0) {
          level = '未参与';
          points = '+0';
          type = '本期未投注';
        } else {
          level = '未中大奖';
          points = '+0';
          type = '再接再厉';
        }
      }

      this.setData({
        num1: level,
        num2: points,
        num3: type,
        isSpinning: false,
        drawRevealed: true
      }); 
    }, 2400);
  },

  closeEasterEgg() { this.setData({ showEasterEgg: false }); },
  
  // 开奖后点击返回抽奖页面，瞬间自动切换并过渡到新一期
  resetToSlots() {
    this.setData({
      drawRevealed: false,
      num1: '00',
      num2: '00',
      num3: '00'
    });
    this.fetchLottoStatus();
    this.fetchLottoHistory();
  },

  // 获取历史大乐透已开奖红榜归档
  fetchLottoHistory() {
    request({ url: '/game/lotto/history-records', method: 'GET' }).then(res => {
      if (res && res.data) {
        this.setData({ historyRecords: res.data || [] });
      }
    }).catch(() => {});
  },

  showRules() {
    wx.showModal({
      title: '学风大乐透规则',
      content: `1. 完成今日报名的全部赛道打卡，可获得免费入场摇号资格。\n2. 若未打卡或打卡未完成，可消耗 ${this.data.spinCost} 赋能分入场摇号。\n3. 参与活动入场后，可获得 10 次免费刷新数字的机会。\n4. 刷新达到 10 次后自动锁死，或者可在 10 次内点击手动锁定锁定心仪号码。\n5. 锁定号码后不支持修改，耐心等待系统管理员开奖结算。\n6. 用户参与活动扣除的 ${this.data.spinCost} 赋能分将悉数累加注入后台活动总奖池中！`,
      showCancel: false
    });
  }
});