const request = require('../../utils/request.js');

const quotes = [
  { text: '不要只因一次失败，就放弃你原来决心想达到的目的。', author: '莎士比亚' },
  { text: '那些看似不起波澜的日复一日，会突然在某天让人看到坚持的意义。', author: '佚名' },
  { text: '乾坤未定，你我皆是黑马。', author: '佚名' },
  { text: '长风破浪会有时，直挂云帆济沧海。', author: '李白' },
  { text: '不是看到希望才去坚持，而是坚持了才能看到希望。', author: '佚名' },
  { text: '天才就是无止境刻苦勤奋的能力。', author: '卡莱尔' },
  { text: '不论成功或失败，皆存乎自己。', author: '朗费罗' },
  { text: '业精于勤，荒于嬉；行成于思，毁于随。', author: '韩愈' },
  { text: '千淘万漉虽辛苦，吹尽狂沙始到金。', author: '刘禹锡' },
  { text: '只要路是对的，就不怕路远。', author: '佚名' },
  { text: '星光不问赶路人，时光不负有心人。', author: '佚名' },
  { text: '锲而舍之，朽木不折；锲而不舍，金石可镂。', author: '荀子' },
  { text: '路漫漫其修远兮，吾将上下而求索。', author: '屈原' },
  { text: '九层之台，起于累土；千里之行，始于足下。', author: '老子' },
  { text: '博观而约取，厚积而薄发。', author: '苏轼' },
  { text: '知之者不如好之者，好之者不如乐之者。', author: '孔子' },
  { text: '天行健，君子以自强不息。', author: '《周易》' },
  { text: '宝剑锋从磨砺出，梅花香自苦寒来。', author: '佚名' },
  { text: '绳锯木断，水滴石穿。', author: '罗大经' },
  { text: '人生没有白走的路，每一步都算数。', author: '李宗盛' },
  { text: '天才是百分之一的灵感，加上百分之九十九的汗水。', author: '爱迪生' },
  { text: '不要努力成为一个成功者，要努力成为一个有价值的人。', author: '爱因斯坦' },
  { text: '世上无难事，只要肯登攀。', author: '毛泽东' },
  { text: '每一次努力都是幸运的伏笔，每天都是好日子。', author: '佚名' }
];

Page({
  data: {
    quote: {},
    // 💡 视觉占位：因为现在是后端算命，前端只要负责提供视觉悬念即可
    lotteryItems: [
      { id: 1, name: '20 赋能分' }, { id: 2, name: '1 赋能分' }, 
      { id: 3, name: '谢谢参与' }, { id: 4, name: '15 赋能分' },
      { id: 5, name: '5 赋能分' }, { id: 6, name: '10 赋能分' }
    ],
    isRolling: false,
    hasDrawn: false,
    prizeInfo: { name: '', isReward: false },
    showPrizeModal: false
  },

  onLoad() {
    this.setData({ quote: quotes[Math.floor(Math.random() * quotes.length)] });
  },

  startLottery() {
    if (this.data.hasDrawn) return;

    this.setData({ isRolling: true });

    // 💡 请求全新的打卡结算抽奖接口
    request({ url: '/game/lottery/draw', method: 'POST' }).then(res => {
      if (res && res.success) {
        const resultData = res.data;
        
        // 刻意让滚动效果持续 1.5 秒
        setTimeout(() => {
          this.setData({ 
            isRolling: false, 
            hasDrawn: true,
            prizeInfo: {
              name: resultData.prizeName,
              isReward: resultData.isReward
            }
          });

          if (resultData.isReward) {
            setTimeout(() => { this.setData({ showPrizeModal: true }); }, 400);
          }
        }, 1500);

      } else {
        this.setData({ isRolling: false });
        wx.showToast({ title: res.message || '今日次数已用尽', icon: 'none' });
      }
    }).catch(() => {
      this.setData({ isRolling: false });
      wx.showToast({ title: '网络波动，请稍后再试', icon: 'none' });
    });
  },

  closePrizeModal() { this.setData({ showPrizeModal: false }); },
  preventClose() {},
  goBack() { wx.switchTab({ url: '/pages/checkin/checkin' }); }
});