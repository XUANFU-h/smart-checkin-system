<template>
  <div class="min-h-screen bg-slate-50 flex overflow-hidden relative">
    <div class="absolute top-0 left-0 w-96 h-96 bg-blue-100/40 rounded-full blur-3xl pointer-events-none"></div>
    <div class="absolute bottom-0 right-0 w-96 h-96 bg-indigo-100/40 rounded-full blur-3xl pointer-events-none"></div>

    <!-- 💡 极致紧凑固定式侧边导航栏 (w-48, 192px) -->
    <aside
        class="h-[calc(100vh-2rem)] my-4 ml-4 w-48 transition-all duration-300 ease-in-out flex flex-col relative z-20 flex-shrink-0"
        style="
        background: rgba(255, 255, 255, 0.7);
        backdrop-filter: blur(16px);
        -webkit-backdrop-filter: blur(16px);
        border: 1px solid rgba(226, 232, 240, 0.8);
        border-radius: 24px;
        box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.05), 0 8px 10px -6px rgba(0, 0, 0, 0.05);
      "
    >
      <!-- 标志区域 -->
      <div class="p-4 flex items-center justify-start border-b border-slate-200/50 space-x-2">
        <div class="w-6 h-6 bg-blue-600 rounded-lg flex items-center justify-center text-white font-black text-xs shadow-md shadow-blue-200">
          📊
        </div>
        <span class="font-black text-slate-800 tracking-tight text-xs truncate">打卡后台·大中枢</span>
      </div>

      <!-- 导航链接 -->
      <nav class="flex-1 overflow-y-auto p-3 space-y-1 custom-scrollbar">
        <router-link
            v-for="item in visibleMenuItems"
            :key="item.path"
            :to="item.path"
            class="flex items-center space-x-2.5 px-3 py-2.5 rounded-xl transition-all group"
            :class="[
            $route.path === item.path
              ? 'bg-blue-600 text-white shadow-md shadow-blue-100 font-bold'
              : 'text-slate-600 hover:bg-slate-200/40 hover:text-slate-900'
          ]"
        >
          <span class="text-base flex-shrink-0 group-hover:scale-110 transition-transform">{{ item.icon }}</span>
          <span class="text-[11px] font-bold tracking-wide whitespace-nowrap">{{ item.name }}</span>
        </router-link>
      </nav>

      <!-- 底部区域 (管理员名称与状态展示 + 注销) -->
      <div class="p-3 border-t border-slate-200/50 flex flex-col gap-1.5">
        <div class="px-3 py-1 flex items-center space-x-2">
          <div class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></div>
          <span class="text-[11px] font-black text-slate-700 truncate">{{ adminName }}</span>
        </div>
        <button
            @click="handleLogout"
            class="w-full flex items-center space-x-3 px-3 py-2 text-rose-600 hover:bg-rose-50 rounded-xl transition-colors"
        >
          <span class="text-base">🚪</span>
          <span class="text-xs font-bold whitespace-nowrap">安全注销</span>
        </button>
      </div>
    </aside>

    <!-- 主视口区域 (全量删除 header，腾出 64px 垂直高度) -->
    <main class="flex-1 h-screen overflow-y-auto relative z-10 flex flex-col">
      <div class="flex-1 p-6 overflow-x-hidden">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { usePermStore } from '@/store/permStore'

const router = useRouter()
const permStore = usePermStore()
const adminName = ref('管理员')
let ws: WebSocket | null = null

const menuItems = [
  { name: '数据面板', path: '/admin/dashboard', icon: '📈', key: 'dashboard' },
  { name: '学生档案', path: '/admin/students', icon: '👥', key: 'students' },
  { name: '赛道管理', path: '/admin/audit', icon: '🤖', key: 'audit' },
  { name: '申诉大厅', path: '/admin/appeal', icon: '⚖️', key: 'appeal' },
  { name: '社区管理', path: '/admin/community', icon: '🌐', key: 'community' },
  { name: '创意玩法', path: '/admin/lotto', icon: '🎰', key: 'lotto' },
  { name: '学风驿站', path: '/admin/shop', icon: '🏪', key: 'shop' },
  { name: '系统设置', path: '/admin/settings', icon: '⚙️', key: 'settings' }
]

// 💡 根据权限过滤可见的菜单项
const visibleMenuItems = computed(() => {
  return menuItems.filter(item => permStore.hasMenu(item.key))
})

// 💡 提升为全局接管，实现长连接高可用
const initGlobalWebSocket = () => {
  if (ws) {
    ws.onclose = null // 🌟 核心防线：绝不允许老连接触发重试定时器！
    ws.close()
  }

  const token = localStorage.getItem('admin_token') || localStorage.getItem('userToken') || ''
  const deviceId = localStorage.getItem('admin_device_id') || 'unknown_panel'

  // 防呆设计：未登录不发起连接
  if (!token) return

  // 自适应配置正确的后端 WebSocket 独立路由
  let wsUrl = ''
  const hostname = window.location.hostname
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'

  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    wsUrl = `ws://127.0.0.1:8081/api/ws?token=${token}&deviceId=${deviceId}`
  } else if (window.location.port === '8080') {
    wsUrl = `${protocol}//${hostname}:8080/api/ws?token=${token}&deviceId=${deviceId}`
  } else {
    // 🌟 生产网关自适应：直接连通独立的 Go 后端 WebSocket 网关服务
    let wsHost = window.location.host
    if (hostname.startsWith('admin.')) {
      wsHost = hostname.replace(/^admin\./, 'api.')
    }
    wsUrl = `${protocol}//${wsHost}/api/ws?token=${token}&deviceId=${deviceId}`
  }

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    console.log('🚀 [全局通信中枢] WebSocket 开启成功！')
  }

  ws.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data)
      // 向浏览器大中枢全局分发自定义无感刷新通知事件
      window.dispatchEvent(new CustomEvent('global-ws-sync', { detail: payload }))
    } catch (e) {
      console.error('[全局通信中枢] WebSocket 消息解析失败:', e)
    }
  }

  ws.onclose = () => {
    console.log('📡 [全局通信中枢] WebSocket 链接断开，5秒后自动重连...')
    setTimeout(initGlobalWebSocket, 5000)
  }
}

const handleLogout = () => {
  ElMessageBox.confirm('确定要安全注销并交还当前大中枢管辖权吗？', '注销确认', {
    type: 'warning',
    confirmButtonText: '确定登出',
    cancelButtonText: '取消'
  }).then(() => {
    if (ws) {
      ws.onclose = null // 🌟 登出时彻底关闭重试
      ws.close()
      ws = null
    }
    permStore.reset()
    localStorage.clear()
    router.push('/login')
  })
}

onMounted(async () => {
  adminName.value = localStorage.getItem('admin_name') || '超级管理员'
  initGlobalWebSocket()
  await permStore.loadPermissions()
})

onUnmounted(() => {
  if (ws) {
    ws.onclose = null // 🌟 卸载时彻底关闭重试
    ws.close()
    ws = null
  }
})
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.3);
  border-radius: 4px;
}
</style>