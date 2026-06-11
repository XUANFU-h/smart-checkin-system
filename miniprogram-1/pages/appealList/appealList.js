const request = require('../../utils/request.js');

Page({
  data: { appeals: [] },
  goBack() {
    wx.navigateBack({
      fail: () => {
        wx.reLaunch({ url: '/pages/home/home' });
      }
    });
  },
  onShow() { this.fetchAppeals(); },

  fetchAppeals() {
    request({ url: '/ticket/list', method: 'GET' }).then(res => {
      if (res && res.success) {
        const list = (res.data || []).map(item => ({
          ...item, 
          imgArr: item.imageUrl ? item.imageUrl.split(',') : []
        }));
        this.setData({ appeals: list });
      }
    }).catch(()=>{});
  },

  previewImg(e) { wx.previewImage({ current: e.currentTarget.dataset.url, urls: e.currentTarget.dataset.list }); },

  editAppeal(e) {
    const item = e.currentTarget.dataset.item;
    // 💡 修复1：使用安全本地缓存代替高危的 URL 传参
    wx.setStorageSync('editTicketData', item);
    wx.navigateTo({ url: `/pages/appeal/appeal?isEdit=1` });
  },

  revokeAppeal(e) {
    wx.showModal({
      title: '撤销确认', content: '是否确认撤销这条工单？撤销后将被物理删除。',
      success: (res) => {
        if (res.confirm) {
          request({
            url: '/ticket/revoke',
            method: 'POST',
            data: { id: e.currentTarget.dataset.id }
          }).then(resp => {
            if (resp.success) {
              wx.showToast({ title: '已撤销', icon: 'success' });
              this.fetchAppeals();
            } else {
              wx.showToast({ title: resp.message || '撤销失败', icon: 'none' });
            }
          }).catch(()=>{});
        }
      }
    });
  },

  goNewAppeal() { wx.navigateTo({ url: '/pages/appeal/appeal' }); }
})