const request = require('../../utils/request.js');

Page({
  data: {
    userInfo: {},
    unreadCount: 0,
    
    // 数据看板
    todayProgress: '',
    currentRank: 0,
    yesterdayEmpowerment: 0, 
    yesterdayMoral: 0,
    
    // 悬赏�?
    bounties: [],

    // 社区
    currentChannel: 'recommend',
    myTracks: [], 
    posts: [],
    
    showCommentModal: false,
    commentText: '',
    currentPostId: null,
    isSubmitting: false
  },

  onShow() {
    if (wx.getStorageSync('is_finished') === 1) return wx.reLaunch({ url: '/pages/summary/summary' });
    if (typeof this.getTabBar === 'function' && this.getTabBar()) this.getTabBar().setData({ selected: 0 });
    
    this.setData({ userInfo: wx.getStorageSync('userInfo') || {} });

    this.fetchUserDashboardStats(); 
    this.fetchUnreadCount();
    this.fetchBounties();
    this.syncMyTracks();
    this.fetchPosts(this.data.currentChannel);
    
    this.silentBindWechatOpenID();
  },

  silentBindWechatOpenID() {
    const isBound = wx.getStorageSync('wechat_openid_bound');
    if (isBound) return;

    wx.login({
      success: (loginRes) => {
        if (loginRes.code) {
          request({
            url: '/user/bind-wechat',
            method: 'POST',
            data: { code: loginRes.code }
          }).then(res => {
            if (res && (res.code === 200 || res.code === 0 || res.success)) {
              wx.setStorageSync('wechat_openid_bound', true);
              console.log("📡 [微信通知网关] 用户的真实微�?OpenID 已静默绑定完成！");
            }
          }).catch(()=>{});
        }
      }
    });
  },

  onPullDownRefresh() {
    this.onShow();
    wx.stopPullDownRefresh();
  },

  fetchUserDashboardStats() {
    request({ url: '/user/dashboard-stats', method: 'GET' }).then(res => {
      if (res && res.data) {
        this.setData({
          yesterdayEmpowerment: res.data.yesterdayEmpowerment || 0,
          yesterdayMoral: res.data.yesterdayMoral || 0,
          todayProgress: res.data.todayProgress || '未打�?, 
          currentRank: res.data.currentRank || 0
        });
      }
    }).catch(()=>{});
  },

  fetchUnreadCount() {
    request({ url: '/message/unread-count', method: 'GET' }).then(res => {
      if (res && res.data !== undefined) this.setData({ unreadCount: res.data });
    }).catch(()=>{});
  },

  fetchBounties() {
    request({ url: '/community/bounties', method: 'GET' }).then(res => {
      if (res && res.data) {
        const processed = (res.data || []).map(b => {
          let formattedEndTime = '永久有效';
          if (b.endTime) {
            const parts = b.endTime.split(/[- :]/); 
            if (parts.length >= 5) formattedEndTime = `${parseInt(parts[1], 10)}�?{parseInt(parts[2], 10)}�?{parts[3]}:${parts[4]}`;
          }
          return { ...b, formattedEndTime };
        });
        this.setData({ bounties: processed });
      }
    }).catch(()=>{});
  },

  // === 导航 ===
  goToMessage() { wx.navigateTo({ url: '/pages/message/message' }); },
  goToDashboard(e) { wx.navigateTo({ url: `/pages/dataDashboard/dataDashboard?tab=${e.currentTarget.dataset.tab}` }); },

  // === 社区及帖子逻辑 ===
  syncMyTracks() {
    request({ url: '/track/my', method: 'GET' }).then(res => {
      if (res && res.data) {
        wx.setStorageSync('myTracks', res.data || []); 
        this.initDynamicTabs(res.data || []); 
      }
    }).catch(()=>{});
  },

  initDynamicTabs(tracks) {
    const formattedTracks = tracks.map(t => {
      let icon = '🎯';
      if (t.name.includes('逐光运动') || t.name.includes('�?)) icon = '🏃';
      else if (t.name.includes('语路同行') || t.name.includes('英语')) icon = '🗣�?;
      else if (t.name.includes('墨香伴读') || t.name.includes('�?)) icon = '📚';
      else if (t.name.includes('码上精进') || t.name.includes('代码')) icon = '💻';
      return { ...t, icon: icon };
    });
    this.setData({ myTracks: formattedTracks });
  },

  switchTab(e) {
    const channel = e.currentTarget.dataset.id;
    if(channel === this.data.currentChannel) return;
    this.setData({ currentChannel: channel, posts: [] });
    this.fetchPosts(channel); 
  },

  fetchPosts(channelId) {
    wx.showLoading({ title: '加载�?..' });
    request({ url: `/community/posts?channel=${channelId}&t=${Date.now()}`, method: 'GET' }).then(res => {
      wx.hideLoading();
      if (!res || !res.data) return this.setData({ posts: [] });
      
      const processedPosts = (res.data || []).map(p => ({ 
        ...p, 
        isOfficial: p.type === 'official' || (p.adminComment && p.adminComment.startsWith('OFFICIAL_POST')),
        isAnonymous: p.isAnonymous === 1 || String(p.isAnonymous).toLowerCase() === 'true',
        userName: (p.isAnonymous === 1 || String(p.isAnonymous).toLowerCase() === 'true') ? '神秘同学' : (p.authorName || p.userName || '官方助手'),
        title: p.pushTitle || p.topic || p.title || '',
        content: p.pushContent || p.content || p.contentText || '',
        pushComment: p.pushComment || (!p.adminComment?.startsWith('OFFICIAL') ? p.adminComment : ''),
        isPinned: p.isPinned === 1 || String(p.isPinned).toLowerCase() === 'true',
        imageUrl: (p.imageUrl && !p.imageUrl.startsWith('http')) ? 'http://127.0.0.1:8080' + p.imageUrl : (p.imageUrl || ''), 
        dateStr: p.createTime || '',
        likeCount: p.likeCount || 0,
        commentCount: p.commentCount || 0,
        isLiked: p.isLiked || false
      }));
      this.setData({ posts: processedPosts });
    }).catch(() => { wx.hideLoading(); if(this) this.setData({ posts: [] }); });
  },

  handleLike(e) {
    const { id, index } = e.currentTarget.dataset;
    const post = this.data.posts[index];
    const keyLike = `posts[${index}].isLiked`;
    const keyCount = `posts[${index}].likeCount`;
    this.setData({ [keyLike]: !post.isLiked, [keyCount]: post.isLiked ? Math.max(0, post.likeCount - 1) : post.likeCount + 1 });

    request({ url: '/community/like', method: 'POST', data: { postId: id, action: !post.isLiked ? 'like' : 'cancel' } })
      .catch(() => { this.setData({ [keyLike]: post.isLiked, [keyCount]: post.likeCount }); });
  },

  openCommentModal(e) { this.setData({ showCommentModal: true, currentPostId: e.currentTarget.dataset.id, commentText: '' }); },
  closeCommentModal() { this.setData({ showCommentModal: false, currentPostId: null }); },

  submitComment() {
    if (!this.data.commentText.trim()) return wx.showToast({ title: '评论不能为空', icon: 'none' });
    this.setData({ isSubmitting: true });
    wx.showLoading({ title: '发送中...' });

    request({ url: '/community/comment', method: 'POST', data: { postId: this.data.currentPostId, content: this.data.commentText } }).then(res => {
      this.setData({ isSubmitting: false });
      wx.hideLoading();
      if (res && (res.code === 200 || res.code === 0)) {
        wx.showToast({ title: '评论成功' });
        this.closeCommentModal();
        this.fetchPosts(this.data.currentChannel);
      } else { wx.showToast({ title: res.message || '评论失败', icon: 'none' }); }
    }).catch(() => { this.setData({ isSubmitting: false }); wx.hideLoading(); });
  },
  
  previewImage(e) { wx.previewImage({ current: e.currentTarget.dataset.url, urls: [e.currentTarget.dataset.url] }); }
});