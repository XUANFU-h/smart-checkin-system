<template>
  <div class="min-h-screen bg-slate-900 flex items-center justify-center p-6 relative overflow-hidden font-sans">
    <!-- Gradient background blobs -->
    <div class="absolute top-1/4 left-1/4 w-96 h-96 bg-blue-600/20 rounded-full blur-3xl animate-pulse"></div>
    <div class="absolute bottom-1/4 right-1/4 w-96 h-96 bg-emerald-600/20 rounded-full blur-3xl animate-pulse" style="animation-delay: 2s;"></div>

    <div class="max-w-md w-full bg-slate-800/40 backdrop-blur-xl border border-slate-700/50 rounded-3xl p-8 text-center shadow-2xl relative z-10 hover:border-slate-600/50 transition-colors">
      <div class="text-9xl font-black bg-gradient-to-r from-blue-400 via-indigo-400 to-emerald-400 bg-clip-text text-transparent select-none animate-bounce">
        404
      </div>
      
      <h1 class="text-2xl font-bold text-white mt-6 mb-2">
        宇宙的尽头，什么都没有
      </h1>
      
      <p class="text-slate-400 text-sm mb-8 leading-relaxed">
        您访问的页面好像搭乘着火箭飞往了外太空，或者已被管理员物理抹除。请检查您的网址是否输入正确。
      </p>

      <div class="flex flex-col gap-3">
        <el-button 
          type="primary" 
          size="large"
          class="w-full !rounded-xl !h-12 font-bold shadow-lg shadow-blue-500/20 !border-0 bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-600 hover:to-indigo-750 transition-all transform hover:scale-[1.02]"
          @click="goHome"
        >
          🛸 返回管理系统主页
        </el-button>
        
        <el-button 
          size="large"
          plain
          class="w-full !rounded-xl !h-12 font-bold !bg-transparent !text-slate-350 !border-slate-700 hover:!border-slate-500 hover:!text-white transition-all transform hover:scale-[1.02]"
          @click="goBack"
        >
          ↩️ 返回上一页
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goHome = () => {
  router.push('/admin/dashboard')
}

const goBack = () => {
  router.go(-1)
}
</script>

<style scoped>
/* Optional custom CSS animations */
@keyframes pulse {
  0%, 100% { opacity: 0.6; transform: scale(1); }
  50% { opacity: 0.8; transform: scale(1.1); }
}
</style>
