// utils/request.js
const config = require('../config.js'); // 💡 引入全局配置文件

const request = (options) => {
  return new Promise((resolve, reject) => {
      const token = wx.getStorageSync('userToken') || '';
      const deviceId = wx.getStorageSync('deviceId') || '';

      // 核心请求头：只保留最纯粹的标准 JSON 声明、JWT 身份令牌和物理设备标识
      const header = {
          'Content-Type': 'application/json',
          ...options.header
      };

      if (token) header['Authorization'] = `Bearer ${token}`;
      if (deviceId) header['X-Device-Id'] = deviceId;

      // 💯 智能防呆平铺：清洗 baseUrl 末尾的斜杠与 /api
      let cleanBase = config.baseUrl.trim().replace(/\/+$/, '');
      if (!cleanBase.endsWith('/api')) {
          cleanBase += '/api';
      }

      // 💯 智能洗涤：无论页面传入的是 /api/auth 还是 /auth，或者是带不带斜杠，统一纠偏
      let subPath = options.url.trim();
      if (subPath.startsWith('http')) {
          subPath = '';
      } else {
          // 强行清洗掉可能重复出现的 /api 前缀，交由中枢统一调度
          subPath = subPath.replace(/^\/?api/, '');
          subPath = subPath.startsWith('/') ? subPath : '/' + subPath;
      }

      // 最终拼装出 100% 绝对合规无污染的请求靶向
      const requestUrl = options.url.startsWith('http') ? options.url : `${cleanBase}${subPath}`;

      wx.request({
          url: requestUrl,
          method: options.method || 'GET',
          data: options.data || {},
          header: header,
          success: (res) => {
              const data = res.data;

              // 🌟 1. 401 拦截处理：从未授权或登录态过期，清空缓存并强制退回登录页
              if (res.statusCode === 401 || (data && data.code === 401)) {
                  wx.showToast({ title: data.message || '登录失效，请重新核验', icon: 'none' });
                  wx.removeStorageSync('userToken');
                  wx.removeStorageSync('hasSelectedTracks');
                  setTimeout(() => { wx.reLaunch({ url: '/pages/login/login' }); }, 1500);
                  return reject(data);
              }

              // 🌟 2. 403 拦截处理：完美配合后端的风控、设备拉黑和超管封禁提示
              if (res.statusCode === 403 || (data && data.code === 403)) {
                  wx.showToast({ 
                      title: data.msg || data.error || '您的访问已被安全审计拦截', 
                      icon: 'none',
                      duration: 3000
                  });
                  return reject(data);
              }

              // 🌟 3. 基础成功判定：完美兼容你的 Go 后端标准规范
              const isSuccess = (res.statusCode === 200) && 
                                (!data.code || data.code === 200 || data.code === 0 || data.success === true);

              if (isSuccess) {
                  // 💡 极其强悍的智能多模 Proxy 网关：
                  // 1. 如果返回的 payload (data.data) 是一个数组或对象，我们直接代理它，使其表现为最纯粹的数据主体 (Array.isArray(res) 完美为 true，wx:for 完美迭代)
                  // 2. 当老页面试图访问 res.data 时，Proxy 智能重定向返回数据主体自身，实现完美向下兼容。
                  // 3. 当老页面检测 res.success 时，智能返回 true 绕过包封校验。
                  const payload = data.data;
                  if (payload !== undefined && payload !== null && typeof payload === 'object') {
                      resolve(new Proxy(payload, {
                          get(target, prop, receiver) {
                              if (prop === 'data') {
                                  return target; // res.data 返回 payload 自身
                              }
                              if (prop === 'success') {
                                  return true; // 兼容 res.success 的真值检测
                              }
                              if (prop === 'code') {
                                  return 200; // 兼容 res.code === 200
                              }
                              return Reflect.get(target, prop, receiver);
                          }
                      }));
                      return;
                  }
                  
                  // 如果没有内层载荷，或者载荷是基本数据类型，直接代理外层包封对象
                  resolve(new Proxy(data, {
                      get(target, prop, receiver) {
                          if (prop === 'data') {
                              return target.data !== undefined ? target.data : target;
                          }
                          if (prop === 'success') {
                              return target.success !== undefined ? target.success : true;
                          }
                          if (prop in target) {
                              return Reflect.get(target, prop, receiver);
                          }
                          if (target.data && typeof target.data === 'object' && prop in target.data) {
                              return Reflect.get(target.data, prop, receiver);
                          }
                          return undefined;
                      }
                  }));
              } else {
                  reject(data);
              }
          },
          fail: (err) => {
              console.error("微信网络层链路 fail 异常:", err);
              wx.showToast({ title: '网络链路异常或被安全策略拦截', icon: 'none' });
              reject(err);
          }
      });
  });
};

module.exports = request;