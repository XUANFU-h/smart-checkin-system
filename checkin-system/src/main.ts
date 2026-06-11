import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'

// 引入全局样式和 Tailwind CSS
import './style.css'
// 💡 注意：虽然我们配置了按需自动导入组件，但 Element Plus 的全局反馈样式（如 ElMessage 弹窗）仍需要在此处手动引入
import 'element-plus/dist/index.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')