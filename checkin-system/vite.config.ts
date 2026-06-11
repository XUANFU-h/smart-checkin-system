import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path' // 用于处理路径别名
import AutoImport from 'unplugin-auto-import/vite' // 用于 Element Plus 自动导入
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    // 💡 自动导入 Element Plus 组件，无需在每个页面手动 import
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    // 💡 必须配置别名，与 tsconfig.app.json 中的 @/* 对应
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    open: true, // 自动打开浏览器
  }
})