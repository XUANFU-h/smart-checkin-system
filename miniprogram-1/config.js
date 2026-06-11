// config.js 最终完美版
const ENV = 'dev'; // 项目已转为本地开发，默认对接本机 Go 后端

const config = {
  dev: {
    baseUrl: 'http://127.0.0.1:8080/api', // 💡 使用 127.0.0.1 避免解析问题
    wsUrl: 'ws://127.0.0.1:8080/api/ws'
  },
  prod: {
    baseUrl: 'http://127.0.0.1:8080/api',
    // ✅ 这是唯一正确的 WebSocket 地址
    wsUrl: 'ws://127.0.0.1:8080/api/ws'
  }
};

module.exports = config[ENV];
