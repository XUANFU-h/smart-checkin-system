// 💡 引入统一的请求拦截器
const request = require('../../utils/request.js'); 

// 全局变量：用于存放从服务器拉取的真实时间，防止本地篡改
let currentSystemDate = new Date(); 

// 💡 修复：高阶高质国密级时间优雅汉化转换工具函数
function formatTimeChinese(timeStr) {
  if (!timeStr) return '';
  let date;
  if (timeStr.indexOf('T') !== -1) {
    // 兼容 ISO 格式: 2026-05-17T17:48:52+08:00
    date = new Date(timeStr);
  } else {
    date = new Date(timeStr.replace(/-/g, '/'));
  }
  if (isNaN(date.getTime())) return timeStr;
  
  const y = date.getFullYear();
  const m = date.getMonth() + 1;
  const d = date.getDate();
  const h = String(date.getHours()).padStart(2, '0');
  const min = String(date.getMinutes()).padStart(2, '0');
  const s = String(date.getSeconds()).padStart(2, '0');
  
  return `${y}年${m}月${d}日 ${h}时${min}分${s}秒`;
}

Page({
  data: {
    emptyDays: [], 
    calendarDays: [], 
    todayStr: '', 
    selectedDate: '', 
    selectedStatus: 0, 
    makeupCards: 0, 
    deadlineTime: '22:00', // 默认截止时间，将被后端覆盖
    
    myTracks: [],
    currentTrackId: null,
    allRecords: [],      
    displayRecords: [],  
    
    showEvalModal: false,
    currentEval: {},
    
    showSportsPopup: false 
  },

  onShow() {
    if (typeof this.getTabBar === 'function' && this.getTabBar()) this.getTabBar().setData({ selected: 3 });
    
    if (wx.getStorageSync('is_finished') === 1) {
        wx.reLaunch({ url: '/pages/summary/summary' });
        return; 
    }

    this.fetchSystemSettings();
  },

  fetchSystemSettings() {
    request({
      url: '/api/admin/settings/get',
      method: 'GET'
    }).then(res => {
      if (res && res.data) {
        const remoteDeadline = res.data.deadline || res.data.checkin_deadline;
        if (remoteDeadline) this.setData({ deadlineTime: remoteDeadline });

        if (res.data.current_time) {
           currentSystemDate = new Date(res.data.current_time.replace(/-/g, '/'));
        }
      }
      this.generateRealCalendar(); 
      this.syncMyTracks(); 
    }).catch(() => {
      this.generateRealCalendar(); 
      this.syncMyTracks(); 
    });
  },

  generateRealCalendar() {
    const yyyy = currentSystemDate.getFullYear();
    const mm = String(currentSystemDate.getMonth() + 1).padStart(2, '0');
    const dd = String(currentSystemDate.getDate()).padStart(2, '0');
    const todayStr = `${yyyy}-${mm}-${dd}`;

    const days = [];
    const emptyDays = [0];

    for (let i = 11; i <= 31; i++) {
      let isBeta = (i >= 11 && i <= 17);
      days.push({ month: 5, date: i, fullDate: `2026-05-${String(i).padStart(2,'0')}`, status: 0, isBeta: isBeta });
    }
    for (let i = 1; i <= 8; i++) {
      days.push({ month: 6, date: i, fullDate: `2026-06-${String(i).padStart(2,'0')}`, status: 0, isBeta: false });
    }
    
    const validDates = days.map(d => d.fullDate);
    let initialSelected = validDates.includes(todayStr) ? todayStr : '2026-05-11';

    this.setData({ 
      calendarDays: days, 
      emptyDays: emptyDays,
      todayStr: todayStr,
      selectedDate: initialSelected
    });
  },

  syncMyTracks() {
    const userId = wx.getStorageSync('userId');
    if (!userId) return;

    request({
      url: `/api/track/my?userId=${userId}`, 
      method: 'GET'
    }).then(res => {
      if (res && res.data) {
        const freshTracks = res.data || [];
        this.setData({ myTracks: freshTracks });
        
        if (freshTracks.length > 0) {
          const isCurrentValid = freshTracks.find(t => t.id === this.data.currentTrackId);
          if (!this.data.currentTrackId || !isCurrentValid) {
            this.setData({ currentTrackId: freshTracks[0].id });
          }
        } else {
           this.setData({ currentTrackId: null });
        }
        
        this.fetchInitData(userId); 
      }
    }).catch(()=>{});
  },

  fetchInitData(userId) {
    if (userId) {
      request({
        url: `/api/user/info`, 
        method: 'GET'
      }).then(res => {
        if (res && res.data) {
           this.setData({ makeupCards: res.data.makeupCards || res.data.makeup_cards || 0 });
           
           // 💡 云端自动同步：换手机、清理缓存后，无缝从云端同步所有已解锁日期，体验自愈性极致丝滑
           if (res.data.unlockedMakeupDays) {
              wx.setStorageSync('unlocked_makeup_days', res.data.unlockedMakeupDays);
              this.renderCalendarStatus();
           }
        }
      }).catch(()=>{});

      if (this.data.myTracks.length > 0) {
         this.fetchAllRecords(userId);
      }
    }
  },

  fetchAllRecords(userId) {
    const tracks = this.data.myTracks;
    wx.showLoading({ title: '同步数据中...' });
    
    const promises = tracks.map(t => new Promise(resolve => {
      request({
        url: `/api/checkin/my?userId=${userId}&trackId=${t.id}`,
        method: 'GET'
      }).then(res => {
        resolve({ trackId: t.id, records: res && res.data ? res.data : [] });
      }).catch(() => {
        resolve({ trackId: t.id, records: [] });
      });
    }));

    Promise.all(promises).then(results => {
      wx.hideLoading();
      let allRecs = [];
      results.forEach(res => {
        res.records.forEach(r => {
          r.trackId = res.trackId; 
          // 🛡️ 在汉化格式时间前，优先对齐备份日期判定键，保障状态点渲染无损
          if (!r.checkinDate && !r.checkin_date && r.createTime) {
            r.checkinDate = r.createTime.substring(0, 10);
          }
          // 💡 修复：将死板的原始时间规范汉化为可读大白话
          if (r.createTime) {
            r.createTime = formatTimeChinese(r.createTime);
          }
          allRecs.push(r);
        });
      });
      
      this.setData({ allRecords: allRecs }, () => {
        this.renderCalendarStatus();
        this.filterDisplayRecords();
      });
    });
  },

  onDateClick(e) {
    const item = e.currentTarget.dataset.item;
    this.setData({ selectedDate: item.fullDate }, () => {
      this.renderCalendarStatus(); 
      this.filterDisplayRecords();
    });
  },

  switchTrack(e) {
    this.setData({ currentTrackId: e.currentTarget.dataset.id }, () => {
      this.filterDisplayRecords();
    });
  },

  filterDisplayRecords() {
    if (!this.data.allRecords || this.data.allRecords.length === 0) {
      return this.setData({ displayRecords: [] });
    }
    
    const display = this.data.allRecords.filter(r => {
      const rDate = r.checkinDate || r.checkin_date;
      return r.trackId === this.data.currentTrackId && rDate === this.data.selectedDate;
    });
    this.setData({ displayRecords: display });
  },

  renderCalendarStatus() {
    if (this.data.calendarDays.length === 0) return;
    
    const days = [...this.data.calendarDays];
    const allRecords = this.data.allRecords || [];
    const requiredCount = this.data.myTracks.length; 
    const today = this.data.todayStr;
    const unlockedDays = wx.getStorageSync('unlocked_makeup_days') || []; 

    const now = currentSystemDate;
    const timeParts = this.data.deadlineTime.split(':');
    const deadline = new Date(now.getTime());
    deadline.setHours(parseInt(timeParts[0] || 22), parseInt(timeParts[1] || 0), 0, 0);
    const isPastDeadlineToday = (now > deadline);

    days.forEach(d => {
      const dateStr = d.fullDate;
      const recordsForDay = allRecords.filter(r => (r.checkinDate || r.checkin_date) === dateStr);
      const uniqueTracksCompleted = new Set(recordsForDay.map(r => r.trackId)).size;

      if (dateStr > today) {
        d.status = 0;
      } else if (dateStr === today) {
        if (uniqueTracksCompleted >= requiredCount && requiredCount > 0) {
          d.status = 1; 
        } else if (uniqueTracksCompleted > 0) {
          d.status = 2; 
        } else {
          d.status = isPastDeadlineToday ? 3 : 0; 
        }
      } else {
        if (uniqueTracksCompleted >= requiredCount && requiredCount > 0) d.status = 1; 
        else if (uniqueTracksCompleted > 0) d.status = 2; 
        else d.status = 3; 
      }
      
      if (unlockedDays.includes(dateStr) && d.status === 3) {
        d.status = 4; 
      }
    });

    const selectedDayItem = days.find(d => d.fullDate === this.data.selectedDate);
    this.setData({ 
      calendarDays: days,
      selectedStatus: selectedDayItem ? selectedDayItem.status : 0
    });
  },

  showSportsMenu() {
    const isRunningSelected = this.data.myTracks.some(t => t.name.includes('跑') || t.name.includes('运动'));
    if (!isRunningSelected) return wx.showModal({ title: '未解锁赛道', content: '您未选择相关赛道，无法使用。', showCancel: false });
    this.setData({ showSportsPopup: true });
  },
  closeSportsPopup() { this.setData({ showSportsPopup: false }); },
  goToSportsRecord() { this.closeSportsPopup(); wx.navigateTo({ url: '/pages/stepTracker/stepTracker' }); },
  goToSportsImage() {
    this.closeSportsPopup();
    const sportsTrack = this.data.myTracks.find(t => t.name.includes('跑') || t.name.includes('运动'));
    if (sportsTrack) {
      let targetDate = (this.data.selectedDate !== this.data.todayStr) ? this.data.selectedDate : '';
      wx.navigateTo({ url: `/pages/checkinForm/checkinForm?mode=image&makeupDate=${targetDate}&lockedTrackId=${sportsTrack.id}` });
    }
  },

  checkIfActionAllowed() {
    const now = currentSystemDate;
    const timeParts = this.data.deadlineTime.split(':');
    const deadline = new Date(now.getTime());
    deadline.setHours(parseInt(timeParts[0] || 22), parseInt(timeParts[1] || 0), 0, 0);

    if (this.data.selectedDate === this.data.todayStr && now > deadline) {
      wx.showToast({ title: `已过今日截止时间(${this.data.deadlineTime})，禁止操作`, icon: 'none' });
      return false;
    }
    return true;
  },

  goToCheckin(e) {
    const mode = e.currentTarget.dataset.mode;

    if (this.data.selectedStatus === 1) return wx.showToast({ title: '该日期您已完成全部打卡！', icon: 'none' });

    const unlockedDays = wx.getStorageSync('unlocked_makeup_days') || [];
    const isUnlocked = unlockedDays.includes(this.data.selectedDate);

    if (this.data.selectedDate !== this.data.todayStr && !isUnlocked && this.data.selectedStatus !== 4) {
      if (this.data.selectedDate > this.data.todayStr) {
        return wx.showToast({ title: '尚未到达该日期，无法提前打卡', icon: 'none' });
      } else {
        return wx.showToast({ title: '该日已缺卡，请先点击右上角【补签】解锁', icon: 'none' });
      }
    }

    if (!this.checkIfActionAllowed()) return;

    // 💡 修复：解绑下方赛道和打卡按钮。移除 displayRecords.length > 0 强拦截硬锁，允许点击覆盖提交！
    let targetDate = (this.data.selectedDate !== this.data.todayStr) ? this.data.selectedDate : '';
    wx.navigateTo({ url: `/pages/checkinForm/checkinForm?mode=${mode}&makeupDate=${targetDate}&trackId=${this.data.currentTrackId}` });
  },

  useMakeupCard() {
    if (this.data.selectedDate >= this.data.todayStr) {
      return wx.showToast({ title: '当日及未来日期请正常打卡，无需补签', icon: 'none' });
    }
    if (this.data.selectedStatus === 1) {
      return wx.showToast({ title: '该日已完成全部打卡，无需补签', icon: 'none' });
    }

    const unlockedDays = wx.getStorageSync('unlocked_makeup_days') || [];
    const isUnlocked = unlockedDays.includes(this.data.selectedDate);
    if (this.data.selectedStatus === 4 || isUnlocked) {
      return wx.showToast({ title: '补签通道已解锁，请直接下方打卡', icon: 'none' });
    }

    if (this.data.makeupCards <= 0) {
      return wx.showModal({ title: '补签卡不足', content: '是否前往积分商城兑换？', success: (res) => { if (res.confirm) wx.navigateTo({ url: '/pages/shop/shop' }); } });
    }

    wx.showModal({
      title: '开启补签通道',
      content: `将消耗 1 张补签卡解锁 ${this.data.selectedDate} 的权限，是否继续？`,
      success: (res) => {
        if (res.confirm) {
           wx.showLoading({ title: '通道解锁中...' });
           request({
             url: '/api/user/use-makeup-card',
             method: 'POST',
             data: { date: this.data.selectedDate }
           }).then(apiRes => {
             wx.hideLoading();
             if (apiRes && (apiRes.code === 200 || apiRes.code === 0)) {
               wx.showToast({ title: '补签已开启，请打卡', icon: 'success' });
               this.setData({ makeupCards: this.data.makeupCards - 1 });
               
               let unlocked = wx.getStorageSync('unlocked_makeup_days') || [];
               if (!unlocked.includes(this.data.selectedDate)) {
                  unlocked.push(this.data.selectedDate);
                  wx.setStorageSync('unlocked_makeup_days', unlocked);
               }
               this.renderCalendarStatus();
             } else {
               wx.showToast({ title: (apiRes && apiRes.message) || '核销失败或接口异常', icon: 'none' });
             }
           }).catch(() => { wx.hideLoading(); });
        }
      }
    });
  },

  handleDelete(e) {
    const item = e.currentTarget.dataset.item;
    if (!this.checkIfActionAllowed()) return;
    
    wx.showModal({
      title: '删除警告', content: '确认要删除该记录吗？删除后需重新打卡',
      success: (res) => {
        if (res.confirm) {
          wx.showLoading({ title: '删除中...' });
          const userId = wx.getStorageSync('userId');
          
          request({
            url: `/api/checkin/delete?id=${item.id}`,
            method: 'DELETE'
          }).then(delRes => {
            wx.hideLoading();
            if(delRes && (delRes.code === 200 || delRes.code === 0 || delRes.success)) {
                wx.removeStorageSync(`lastSubmit_${item.trackId || item.TrackID || this.data.currentTrackId}`);
                wx.showToast({ title: '删除成功', icon: 'success' });
                this.fetchAllRecords(userId);
            } else {
                wx.showToast({ title: (delRes && delRes.message) || '删除失败', icon: 'none' });
            }
          }).catch(() => { wx.hideLoading(); });
        }
      }
    });
  },

  handleEdit(e) {
    const item = e.currentTarget.dataset.item;
    if (!this.checkIfActionAllowed()) return;
    
    // 💡 修复：自动判定当前打卡对象的类型（带ImageUrl则为图文打卡，否则为文档提交）。
    // 将 uploadType 动态合并至路由参数，防止 checkinForm 因缺少类型信息而将多媒体上传区完全锁定遮蔽。
    const uploadType = item.imageUrl ? 'image' : 'file';
    wx.navigateTo({ 
      url: `/pages/checkinForm/checkinForm?mode=edit&id=${item.id}&trackId=${item.trackId || this.data.currentTrackId}&uploadType=${uploadType}` 
    });
  },

  handleEvaluate(e) {
    const item = e.currentTarget.dataset.item;
    this.setData({ currentEval: item, showEvalModal: true });
  },

  closeEval() { this.setData({ showEvalModal: false }); },
  preventClose() {},

  previewImage(e) {
    const url = e.currentTarget.dataset.url;
    if (url) wx.previewImage({ urls: [url] });
  }
});