const request = require('../../utils/request.js');
const config = require('../../config.js');

Page({
  data: {
    userInfo: {},
    currentTime: '',
    rawCategories: [],
    flatCategories: [],
    selectedTypeIndex: -1,
    reason: '',
    imageList: [],
    isSubmitting: false, 
    isEditMode: false,
    editAppealId: null,

    // 智能动态表单状态
    isCheckinIssue: false,
    trackList: [],
    trackNames: [],
    selectedTrackIndex: -1,
    abnormalDate: '',
    today: ''
  },

  goBack() {
    wx.navigateBack({
      fail: () => {
        wx.reLaunch({ url: '/pages/home/home' });
      }
    });
  },
  
  onLoad(options) {
    this.setData({ userInfo: wx.getStorageSync('userInfo') || {} });
    this.setCurrentTime(); 
    this.loadDynamicCategories();
    this.fetchTracks(); 

    // 💡 修复1：使用本地缓存接收巨型 JSON，杜绝 URL 超长崩溃
    if (options.isEdit === '1') {
      const data = wx.getStorageSync('editTicketData');
      if (data) {
        this.setData({ 
          isEditMode: true, 
          editAppealId: data.id,
          reason: data.reason.includes('】') ? data.reason.split('】 ')[1] : data.reason,
          imageList: data.imageUrl ? data.imageUrl.split(',') : []
        });
        wx.removeStorageSync('editTicketData'); // 用完即毁
      }
    }
  },

  setCurrentTime() {
    const now = new Date();
    const pad = (n) => String(n).padStart(2, '0');
    const todayStr = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
    this.setData({ 
        currentTime: `${todayStr} ${pad(now.getHours())}:${pad(now.getMinutes())}`,
        today: todayStr 
    });
  },

  loadDynamicCategories() {
    request({ url: '/ticket/categories', method: 'GET' }).then(res => {
      if (res && res.success && res.data) {
        const raw = res.data;
        let flatArr = [];
        raw.forEach(item => {
          if (!raw.find(child => child.parentId === item.id)) {
            let path = item.name;
            let current = item;
            while (current.parentId !== 0) {
              let parent = raw.find(p => p.id === current.parentId);
              if (parent) { path = parent.name + ' - ' + path; current = parent; } 
              else break;
            }
            flatArr.push({ id: item.id, label: path, requireImage: item.requireImage });
          }
        });
        this.setData({ rawCategories: flatArr, flatCategories: flatArr.map(c => c.label) });
      }
    }).catch(()=>{});
  },

  fetchTracks() {
    request({ url: '/track/my', method: 'GET' }).then(res => {
      if (res && res.success) {
        this.setData({
          trackList: res.data || [],
          trackNames: (res.data || []).map(t => t.name)
        });
      }
    }).catch(()=>{});
  },

  onTypeChange(e) { 
      const idx = e.detail.value;
      const label = this.data.flatCategories[idx];
      // 💡 智能判断：如果分类包含 打卡/成绩/赛道 等关键字，才要求提供赛道和日期
      const isCheckin = label.includes('打卡') || label.includes('赛道') || label.includes('成绩');
      this.setData({ 
          selectedTypeIndex: idx,
          isCheckinIssue: isCheckin
      }); 
  },
  
  onTrackChange(e) { this.setData({ selectedTrackIndex: e.detail.value }); },
  onDateChange(e) { this.setData({ abnormalDate: e.detail.value }); },
  handleReasonInput(e) { this.setData({ reason: e.detail.value.trim() }); },

  chooseImage() {
    wx.chooseImage({
      count: 3 - this.data.imageList.length,
      sizeType: ['compressed'],
      success: (res) => {
        let validImgs = [...this.data.imageList];
        res.tempFiles.forEach(file => {
          if (file.size > 2 * 1024 * 1024) wx.showToast({ title: '单张图片不能超2MB', icon: 'none' });
          else if (validImgs.length < 3) validImgs.push(file.path);
        });
        this.setData({ imageList: validImgs });
      }
    });
  },
  
  removeImg(e) {
    const list = this.data.imageList;
    list.splice(e.currentTarget.dataset.index, 1); 
    this.setData({ imageList: list });
  },
  
  previewImg(e) { wx.previewImage({ current: e.currentTarget.dataset.url, urls: this.data.imageList }); },

  uploadSingleImage(filePath) {
    return new Promise((resolve, reject) => {
      if(filePath.startsWith('http')) return resolve(filePath);
      wx.uploadFile({
        url: `${config.baseUrl}/checkin/upload`, 
        filePath: filePath, 
        name: 'file',
        header: { 'Authorization': 'Bearer ' + wx.getStorageSync('userToken') },
        success: (res) => {
          try {
            let data = JSON.parse(res.data);
            const domain = config.baseUrl.replace('/api', '');
            resolve(data.imageUrl.startsWith('/') ? domain + data.imageUrl : data.imageUrl);
          } catch(e) { reject(new Error("服务器返回异常")); }
        }, 
        fail: reject
      });
    });
  },

  async submitAppeal() {
    if (this.data.isSubmitting) return;

    if (this.data.selectedTypeIndex === -1 && !this.data.isEditMode) return wx.showToast({ title: '请选择分类', icon: 'none' });
    
    if (this.data.isCheckinIssue) {
        if (this.data.selectedTrackIndex === -1) return wx.showToast({ title: '请选择关联赛道', icon: 'none' });
        if (!this.data.abnormalDate) return wx.showToast({ title: '请选择异常日期', icon: 'none' });
    }

    if (!this.data.reason) return wx.showToast({ title: '请填写事由', icon: 'none' });

    const selectedCategory = this.data.rawCategories[this.data.selectedTypeIndex] || {};
    if (selectedCategory.requireImage === 1 && this.data.imageList.length === 0) {
       return wx.showModal({ title: '必须提供凭证', content: '开发者要求该类型问题必须上传截图！', showCancel: false });
    }

    // 这里可以放订阅消息的 TMplID
    this.processUploadAndSubmit(selectedCategory);
  },

  async processUploadAndSubmit(selectedCategory) {
    this.setData({ isSubmitting: true });
    wx.showLoading({ title: '云端同步中...', mask: true });

    try {
      let uploadedUrls = [];
      for (let img of this.data.imageList) uploadedUrls.push(await this.uploadSingleImage(img));
      
      let finalReason = this.data.reason;
      if (this.data.isCheckinIssue) {
          const trackName = this.data.trackNames[this.data.selectedTrackIndex];
          // ⚠️ 建议后端后续分离出独立字段，此处暂保兼容
          finalReason = `【关联：${trackName} | 日期：${this.data.abnormalDate}】 ` + finalReason;
      }

      request({
        url: '/ticket/appeal', // 后端接收所有的申诉和Bug反馈
        method: 'POST',
        data: {
          id: this.data.editAppealId,
          appealType: this.data.isEditMode ? null : selectedCategory.label,
          reason: finalReason,
          imageUrl: uploadedUrls.join(',')
        }
      }).then(res => {
        wx.hideLoading();
        if (res.success) {
          wx.redirectTo({ url: '/pages/appealSuccess/appealSuccess' });
        } else {
          wx.showToast({ title: res.message || '提交失败', icon: 'none' });
          this.setData({ isSubmitting: false }); 
        }
      }).catch(() => {
        wx.hideLoading();
        this.setData({ isSubmitting: false });
      });

    } catch(e) {
      wx.hideLoading();
      this.setData({ isSubmitting: false });
      wx.showToast({ title: '凭证上传失败', icon: 'none' });
    }
  }
})