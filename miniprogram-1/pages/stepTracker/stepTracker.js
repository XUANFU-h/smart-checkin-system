const request = require('../../utils/request.js');

Page({
  data: {
    latitude: 39.9042, longitude: 116.4074,
    status: 'idle', 
    statusText: 'GO',
    progress: 0, 
    distance: "0.00", 
    duration: "00:00",
    sessionSteps: 0, 
    wechatSteps: "--", 
    polylines: [{ points: [], color: "#07c160", width: 6, arrowLine: true }]
  },
  
  timer: null, 
  seconds: 0,
  holdTimer: null, 
  lastLat: null,
  lastLng: null,
  lastAccelMagnitude: 0, 
  lastStepTime: 0, 
  
  // 💡 优化：将计算数据与渲染数据剥离，防止 JS 浮点数精度爆炸
  rawTotalDistance: 0,

  onLoad() {
    this.mapCtx = wx.createMapContext('sportMap');
    this.moveToLocation(); 
    this.fetchRealWechatSteps(); 
  },

  onUnload() {
    this.stopHardwareTracking();
  },

  fetchRealWechatSteps() {
    wx.login({
      success: (loginRes) => {
        if (loginRes.code) {
          wx.getWeRunData({
            success: (runRes) => {
              request({
                url: '/wechat/decrypt-steps',
                method: 'POST',
                data: { code: loginRes.code, encryptedData: runRes.encryptedData, iv: runRes.iv }
              }).then(decRes => {
                if(decRes && decRes.success && decRes.data) {
                  const stepList = decRes.data.stepInfoList;
                  if (stepList && stepList.length > 0) {
                    this.setData({ wechatSteps: stepList[stepList.length - 1].step }); 
                  }
                }
              }).catch(()=>{});
            },
            fail: () => {
              // 不强行 Toast 弹窗，防止干扰正常跑步体验，静默降级即可
              this.setData({ wechatSteps: "未授权" });
            }
          });
        }
      }
    });
  },

  moveToLocation() {
    wx.getLocation({
      type: 'gcj02',
      isHighAccuracy: true, 
      success: (res) => { 
        this.setData({ latitude: res.latitude, longitude: res.longitude }); 
        this.mapCtx.moveToLocation(); 
      }
    });
  },

  onBtnTap() {
    if (this.data.status === 'idle') {
      this.startTracking();
    } else if (this.data.status === 'error') {
      // 💡 优化：如果是上传失败状态，点击重试直接复用数据上传
      this.executeUpload();
    }
  },

  startTracking() {
    this.setData({ status: 'running', statusText: '运动中', progress: 0 });
    
    this.timer = setInterval(() => {
      this.seconds++;
      const m = String(Math.floor(this.seconds / 60)).padStart(2, '0');
      const s = String(this.seconds % 60).padStart(2, '0');
      this.setData({ duration: `${m}:${s}` });
    }, 1000);

    wx.startLocationUpdate({
      success: () => { wx.onLocationChange(this.handleLocationChange); }
    });

    wx.stopAccelerometer();
    wx.startAccelerometer({ interval: 'normal' });
    wx.onAccelerometerChange(this.handleAccelerometer);
  },

  handleLocationChange(res) {
    if (this.lastLat && this.lastLng) {
      const dist = this.getDistance(this.lastLat, this.lastLng, res.latitude, res.longitude);
      
      if (dist < 0.003) return;
      if (dist > 0.05) {
         this.lastLat = res.latitude; 
         this.lastLng = res.longitude;
         return;
      }

      // 💡 优化：使用内部变量累加防精度丢失
      this.rawTotalDistance += dist;
      const newPoints = [...this.data.polylines[0].points, {latitude: res.latitude, longitude: res.longitude}];
      this.setData({
        distance: this.rawTotalDistance.toFixed(2),
        'polylines[0].points': newPoints
      });
    }
    
    this.lastLat = res.latitude;
    this.lastLng = res.longitude;
  },

  getDistance(lat1, lng1, lat2, lng2) {
    const radLat1 = lat1 * Math.PI / 180.0;
    const radLat2 = lat2 * Math.PI / 180.0;
    const a = radLat1 - radLat2;
    const b = lng1 * Math.PI / 180.0 - lng2 * Math.PI / 180.0;
    const s = 2 * Math.asin(Math.sqrt(Math.pow(Math.sin(a / 2), 2) + Math.cos(radLat1) * Math.cos(radLat2) * Math.pow(Math.sin(b / 2), 2)));
    return s * 6378.137; 
  },

  handleAccelerometer(res) {
    const magnitude = Math.sqrt(res.x * res.x + res.y * res.y + res.z * res.z);
    const now = Date.now();
    
    if (magnitude > 1.35 && this.lastAccelMagnitude <= 1.35 && (now - this.lastStepTime > 300)) {
      this.setData({ sessionSteps: this.data.sessionSteps + 1 });
      this.lastStepTime = now;
    }
    this.lastAccelMagnitude = magnitude;
  },

  stopHardwareTracking() {
    clearInterval(this.timer);
    wx.stopLocationUpdate();
    wx.offLocationChange(this.handleLocationChange);
    wx.stopAccelerometer();
    wx.offAccelerometerChange(this.handleAccelerometer);
  },

  onBtnTouchStart() {
    if (this.data.status !== 'running') return;
    let currentProgress = 0;
    this.holdTimer = setInterval(() => {
      currentProgress += 2; 
      this.setData({ progress: currentProgress });
      if (currentProgress >= 100) this.stopTrackingAndSubmit();
    }, 20);
  },

  onBtnTouchEnd() {
    if (this.data.status !== 'running') return;
    clearInterval(this.holdTimer); 
    if (this.data.progress < 100) this.setData({ progress: 0 });
  },

  stopTrackingAndSubmit() {
    this.stopHardwareTracking();
    clearInterval(this.holdTimer);
    
    if (this.seconds < 60) {
      wx.showToast({ title: '时长不足1分钟，数据已废弃', icon: 'none', duration: 2000 });
      this.setData({ 
        status: 'idle', statusText: 'GO', progress: 0, 
        duration: "00:00", distance: "0.00", sessionSteps: 0, 'polylines[0].points': [] 
      });
      this.seconds = 0;
      this.rawTotalDistance = 0;
      this.lastLat = null;
      this.lastLng = null;
      return;
    }

    this.executeUpload();
  },

  // 💡 优化：将上传核心逻辑独立，方便网络异常时一键重试
  executeUpload() {
    this.setData({ status: 'finished', statusText: '上传中', progress: 100 });
    wx.showLoading({ title: '加密封装档案中...', mask: true });

    const myTracks = wx.getStorageSync('myTracks') || [];
    const sportTrack = myTracks.find(t => t.name.includes('跑步') || t.name.includes('运动'));
    
    if (!sportTrack) {
      wx.hideLoading();
      this.setData({ status: 'error', statusText: '重试' });
      return wx.showToast({ title: '未绑定运动赛道', icon: 'none' });
    }

    const finalContent = `【🏃 运动试炼打卡】\n⏱ 运动时长：${this.data.duration}\n🛣 真实里程：${this.data.distance} 公里\n👣 传感器实测步数：${this.data.sessionSteps} 步\n📱 今日微信总步数：${this.data.wechatSteps} 步\n(注：本数据由系统通过双保险 GPS 与加速度硬件防作弊引擎生成，严禁作假)`;

    request({
      url: '/checkin/submit',
      method: 'POST',
      data: { trackId: sportTrack.id, contentText: finalContent, imageUrl: '' }
    }).then(submitRes => {
      wx.hideLoading();
      if(submitRes && submitRes.success) {
         wx.redirectTo({ url: '/pages/success/success' });
      } else {
         wx.showToast({ title: submitRes.message || '服务器拒收数据', icon: 'none' });
         // 💡 兜底：解开状态锁，允许重传
         this.setData({ status: 'error', statusText: '重试' });
      }
    }).catch(() => {
      wx.hideLoading();
      this.setData({ status: 'error', statusText: '重试' });
    });
  }
})