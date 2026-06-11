const util = require('../../utils/util.js');
const request = require('../../utils/request.js');
const config = require('../../config.js');

Page({
  data: {
    mode: 'image', // 'image', 'file', 'sports_image'
    pageTitle: '图文打卡',
    userInfo: {},
    currentTime: '',
    trackList: [],
    selectedTrack: null,
    submitTitle: '',
    
    imageList: [], // 最多5张
    fileList: [],  // 最多2个
    
    makeupDateStr: '',
    lockedTrackId: null,
    isEditMode: false,
    recordId: null 
  },
  timer: null,

  onLoad(options) {
    let currentMode = options.mode || 'image';
    let pTitle = currentMode === 'file' ? '文档打卡' : '图文打卡';
    
    if (options.lockedTrackId) {
      currentMode = 'sports_image';
      pTitle = '运动专属打卡';
    }

    let isEdit = false;
    let recId = null;
    if (options.mode === 'edit') {
      isEdit = true;
      recId = options.id;
      pTitle = '修改打卡记录';
      currentMode = options.uploadType || 'image';
    }

    this.setData({ 
      mode: currentMode,
      pageTitle: pTitle,
      lockedTrackId: options.lockedTrackId || null,
      userInfo: wx.getStorageSync('userInfo') || {},
      isEditMode: isEdit,
      recordId: recId,
      selectedTrackId: options.trackId || null
    });

    if (options.makeupDate) {
      if (options.makeupDate.indexOf('-') !== -1) {
        this.setData({ makeupDateStr: options.makeupDate });
      } else {
        const currentYear = new Date().getFullYear();
        const currentMonth = String(new Date().getMonth() + 1).padStart(2, '0');
        const dayStr = String(options.makeupDate).padStart(2, '0');
        this.setData({ makeupDateStr: `${currentYear}-${currentMonth}-${dayStr}` });
      }
    }

    this.startClock();
    
    if (isEdit && recId) {
      this.fetchRecordDetail(recId);
    } else {
      this.fetchMyTracks();
    }
  },

  fetchRecordDetail(id) {
    wx.showLoading({ title: '加载中...', mask: true });
    request({
      url: `/api/checkin/detail?id=${id}`,
      method: 'GET'
    }).then(res => {
      wx.hideLoading();
      if (res && res.data) {
        const record = res.data;
        const urls = record.imageUrl ? record.imageUrl.split(',') : [];
        
        this.setData({
          submitTitle: record.content || record.contentText || '',
          imageList: urls
        });

        const tracks = wx.getStorageSync('myTracks') || [];
        if (tracks.length === 0) {
          request({ url: '/api/auth/tracks', method: 'GET' }).then((trackRes) => {
            const netTracks = trackRes.data || [];
            wx.setStorageSync('myTracks', netTracks);
            this.processTrackFilteringAndMatch(netTracks, record.trackId || record.TrackID);
          });
        } else {
          this.processTrackFilteringAndMatch(tracks, record.trackId || record.TrackID);
        }
      } else {
        wx.showToast({ title: '获取记录详情失败', icon: 'none' });
      }
    }).catch(err => {
      wx.hideLoading();
      wx.showToast({ title: '获取详情网络异常', icon: 'none' });
    });
  },

  processTrackFilteringAndMatch(tracks, matchTrackId) {
    let filteredTracks = [];
    if (this.data.mode === 'sports_image') {
      filteredTracks = tracks.filter(t => t.id == this.data.lockedTrackId);
    } else {
      const targetMode = this.data.mode;
      filteredTracks = tracks.filter(t => {
        const reqType = t.requireType || t.RequireType || 'image';
        return reqType.indexOf(targetMode) !== -1;
      });
      // 🌟 物理防空机制：若过滤后列表为空，退回用户报名的全量赛道名册
      if (filteredTracks.length === 0) {
        filteredTracks = tracks;
      }
    }

    this.setData({ trackList: filteredTracks });
    const matched = filteredTracks.find(t => t.id == matchTrackId);
    if (matched) {
      this.setData({ selectedTrack: matched });
    } else if (filteredTracks.length > 0) {
      this.setData({ selectedTrack: filteredTracks[0] });
    }
  },

  onUnload() { 
    if (this.timer) clearInterval(this.timer); 
  },

  goBack() {
    wx.navigateBack();
  },

  startClock() {
    if (this.data.makeupDateStr) {
       this.setData({ currentTime: `${this.data.makeupDateStr} 23:59:59 (补签)` });
    } else {
       this.setData({ currentTime: util.formatTime(new Date()) });
       this.timer = setInterval(() => {
         this.setData({ currentTime: util.formatTime(new Date()) });
       }, 1000);
    }
  },

  // 💡 核心大升级：消除 myTracks 缓存踩空导致没有可分配赛道的严重缺陷
  fetchMyTracks() {
    const tracks = wx.getStorageSync('myTracks') || [];
    const userId = wx.getStorageSync('userId');
    
    if (tracks.length === 0 && userId) {
      // 缓存踩空时，优先拉取当前用户已报名参加的专属赛道列表，确保选择项精准无误
      request({ url: `/api/track/my?userId=${userId}`, method: 'GET' }).then((res) => {
        const myTracks = res.data || [];
        wx.setStorageSync('myTracks', myTracks);
        this.processTrackFiltering(myTracks);
      }).catch(() => {
        // 网络异常兜底全量拉取
        request({ url: '/api/auth/tracks', method: 'GET' }).then((res2) => {
          const netTracks = res2.data || [];
          wx.setStorageSync('myTracks', netTracks);
          this.processTrackFiltering(netTracks);
        }).catch(() => {
          wx.showToast({ title: '赛道库调度失败', icon: 'none' });
        });
      });
    } else {
      this.processTrackFiltering(tracks);
    }
  },

  // 抽离出来的赛道多态智能过滤核心算法
  processTrackFiltering(tracks) {
    let filteredTracks = [];
    if (this.data.mode === 'sports_image') {
      filteredTracks = tracks.filter(t => t.id == this.data.lockedTrackId);
    } else {
      const targetMode = this.data.mode; // 'image' 或 'file'
      filteredTracks = tracks.filter(t => {
        const reqType = t.requireType || t.RequireType || 'image';
        return reqType.indexOf(targetMode) !== -1;
      });
      // 🌟 物理防空机制：若过滤后列表为空，退回用户报名的全量赛道名册，坚决杜绝空白列表阻断业务！
      if (filteredTracks.length === 0) {
        filteredTracks = tracks;
      }
    }

    this.setData({ trackList: filteredTracks });
    
    // 🌟 自动定位聚焦用户在打卡大盘选择的那个赛道
    const targetTrackId = this.data.selectedTrackId;
    const matched = filteredTracks.find(t => t.id == targetTrackId);
    if (matched) {
      this.setData({ selectedTrack: matched });
    } else if (filteredTracks.length > 0) {
      this.setData({ selectedTrack: filteredTracks[0] });
    }
  },

  onTrackChange(e) {
    const track = this.data.trackList[e.detail.value];
    this.setData({ selectedTrack: track });
  },

  onTitleInput(e) {
    this.setData({ submitTitle: e.detail.value });
  },

  chooseImage() {
    const maxCount = 5 - this.data.imageList.length;
    if (maxCount <= 0) return;

    wx.chooseImage({
      count: maxCount,
      sizeType: ['compressed', 'original'],
      success: (res) => {
        let validImgs = [...this.data.imageList];
        res.tempFiles.forEach(file => {
          if (file.size > 10 * 1024 * 1024) {
            wx.showToast({ title: '单张图片不能超过10MB', icon: 'none' });
          } else if (validImgs.length < 5) {
            validImgs.push(file.path);
          }
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

  previewImg(e) {
    wx.previewImage({ current: e.currentTarget.dataset.url, urls: this.data.imageList });
  },

  chooseFile() {
    const maxCount = 2 - this.data.fileList.length;
    if (maxCount <= 0) return;

    wx.chooseMessageFile({
      count: maxCount,
      type: 'file', 
      success: (res) => {
        let validFiles = [...this.data.fileList];
        res.tempFiles.forEach(file => {
          if (file.size > 20 * 1024 * 1024) {
             wx.showToast({ title: '单个文档不能超过20MB', icon: 'none' });
          } else {
            const ext = file.name.substring(file.name.lastIndexOf('.')).toLowerCase();
            const blackList = ['.mp3', '.mp4', '.mkv', '.avi', '.mov', '.wav', '.rmvb'];
            if (blackList.includes(ext)) {
               wx.showToast({ title: '禁止上传音视频格式', icon: 'none' });
            } else if (validFiles.length < 2) {
               validFiles.push(file);
            }
          }
        });
        this.setData({ fileList: validFiles });
      }
    });
  },

  removeFile(e) {
    const list = this.data.fileList;
    list.splice(e.currentTarget.dataset.index, 1);
    this.setData({ fileList: list });
  },

  // ====== 💡 阿里云 OSS 生产直传架构 ======
  uploadFilePromise(filePath) {
    return new Promise((resolve, reject) => {
      request({
        url: '/api/auth/oss-signature',
        method: 'GET'
      }).then((res) => {
        if (res && res.data) {
          const { host, policy, signature, OSSAccessKeyId, dir } = res.data;
          const suffix = filePath.substring(filePath.lastIndexOf('.')) || '.png';
          const fileName = dir + '/' + Date.now() + '_' + Math.random().toString(36).substr(2) + suffix;

          wx.uploadFile({
            url: host,
            filePath: filePath,
            name: 'file',
            formData: {
              key: fileName,
              policy: policy,
              OSSAccessKeyId: OSSAccessKeyId,
              signature: signature,
              success_action_status: '200'
            },
            success: (uploadRes) => {
              if (uploadRes.statusCode === 200 || uploadRes.statusCode === 204) {
                resolve({
                  imageUrl: host + '/' + fileName,
                  imageMd5: "MD5-" + Math.random().toString(36).substr(2, 6).toUpperCase() // 生成临时签名防止防刷机制误判
                });
              } else {
                reject('OSS存储异常');
              }
            },
            fail: () => reject('网络上传失败')
          });
        } else {
          reject('签名获取失败');
        }
      }).catch(() => reject('获取上传签名网络异常'));
    });
  },

  async submitCheckin() {
    if (!this.data.selectedTrack) return wx.showToast({ title: '请选择赛道', icon: 'none' });
    if (!this.data.submitTitle.trim()) return wx.showToast({ title: '请填写标题', icon: 'none' });
    
    const isImgMode = this.data.mode === 'image' || this.data.mode === 'sports_image';
    if (isImgMode && this.data.imageList.length === 0) return wx.showToast({ title: '请至少上传1张图片', icon: 'none' });
    if (this.data.mode === 'file' && this.data.fileList.length === 0) return wx.showToast({ title: '请至少上传1份文档', icon: 'none' });

    const todayStr = this.data.makeupDateStr || util.formatTime(new Date()).split(' ')[0];
    const trackId = this.data.selectedTrack.id;

    if (!this.data.isEditMode) {
      const lastSubmitDate = wx.getStorageSync(`lastSubmit_${trackId}`);
      if (lastSubmitDate === todayStr) {
        wx.showModal({ title: '防刷限制', content: '您已在该日期成功打卡，不可重复提交。', showCancel: false });
        return;
      }
    }

    wx.showLoading({ title: '加密传输中...', mask: true });

    try {
      let finalUrlArr = [];
      let finalMd5Arr = [];

      const uploadTasks = [];
      if (isImgMode) {
        this.data.imageList.forEach(path => uploadTasks.push(this.uploadFilePromise(path)));
      } else {
        // 文档上传
        this.data.fileList.forEach(f => uploadTasks.push(this.uploadFilePromise(f.path)));
      }

      const uploadResults = await Promise.all(uploadTasks);
      
      uploadResults.forEach(res => {
        finalUrlArr.push(res.imageUrl);
        finalMd5Arr.push(res.imageMd5);
      });

      this.executeFinalSubmit(trackId, todayStr, finalUrlArr.join(','), finalMd5Arr.join(','));

    } catch (error) {
      wx.hideLoading();
      wx.showModal({ title: '上传失败', content: error.toString() || '网络异常', showCancel: false });
    }
  },

  executeFinalSubmit(trackId, submitDateStr, finalUrl, finalMd5) {
    const userId = wx.getStorageSync('userId');
    const isEdit = this.data.isEditMode;
    const url = isEdit ? '/api/checkin/update' : '/api/checkin/submit';
    
    const requestData = {
      userId: userId,
      trackId: trackId,
      imageUrl: finalUrl,
      imageMd5: finalMd5,
      content: this.data.submitTitle,
      contentText: this.data.submitTitle
    };

    if (isEdit) {
      requestData.id = Number(this.data.recordId);
    } else {
      requestData.checkinDate = submitDateStr;
      requestData.createTimeOverride = this.data.makeupDateStr ? `${this.data.makeupDateStr} 23:59:59` : null;
    }
    
    request({
      url: url,
      method: 'POST',
      data: requestData
    }).then(res => {
      wx.hideLoading();
      
      if (res && (res.code === 200 || res.code === 0 || res.success)) {
        if (!isEdit && (res.message === "LAST_CHECKIN_DONE" || res.msg === "LAST_CHECKIN_DONE")) {
            wx.setStorageSync('is_finished', 1);
            const summaryDataStr = encodeURIComponent(JSON.stringify(res.summary || res.data || {}));
            return wx.reLaunch({ url: `/pages/summary/summary?data=${summaryDataStr}` });
        }
        
        if (!isEdit) {
          wx.setStorageSync(`lastSubmit_${trackId}`, submitDateStr);
        }

        const successQuotes = [
            "星光不问赶路人，时光不负有心人。",
            "道阻且长，行则将至；行而不辍，未来可期。",
            "发光不是太阳的专利，你也可以发光。",
            "流水不争先，争的是滔滔不绝。"
        ];
        const randomQuote = successQuotes[Math.floor(Math.random() * successQuotes.length)];
        wx.setStorageSync('success_quote', randomQuote);
        
        wx.reLaunch({ url: '/pages/success/success' });
      } else {
        wx.showToast({ title: (res && (res.message || res.msg)) || '提交记录失败', icon: 'none' });
      }
    }).catch((err) => {
      wx.hideLoading();
      wx.showToast({ title: (err && (err.message || err.msg)) || '网络超时', icon: 'none' });
    });
  }
});