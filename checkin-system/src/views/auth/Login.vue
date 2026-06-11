<template>
  <div class="min-h-screen flex items-center justify-center bg-slate-50 px-4 py-12 sm:px-6 lg:px-8 relative overflow-hidden font-sans select-none">
    <div class="absolute -top-40 -right-40 w-96 h-96 bg-blue-100 rounded-full blur-3xl opacity-60"></div>
    <div class="absolute -bottom-40 -left-40 w-96 h-96 bg-indigo-100 rounded-full blur-3xl opacity-60"></div>

    <div class="max-w-md w-full space-y-6 bg-white/80 backdrop-blur-md p-8 sm:p-10 rounded-3xl shadow-xl border border-slate-200/60 relative z-10">
      
      <div class="text-center">
        <div class="inline-flex items-center justify-center w-20 h-20 bg-white rounded-2xl shadow-md border border-slate-100 mb-4 overflow-hidden p-2 transition-transform hover:rotate-6">
          <img src="/logo.png" alt="System Logo" class="w-full h-full object-contain" @error="(e: any) => e.target.src='https://element-plus.org/images/element-plus-logo.svg'" />
        </div>
        <h2 class="text-2xl font-black text-slate-900 tracking-tight">智能打卡系统 · 管理平台</h2>
        <p class="mt-1.5 text-[10px] text-slate-400 font-bold uppercase tracking-widest">
          Information Engineering College · Learning Department
        </p>
      </div>

      <!-- 1. 账号密码登录 -->
      <el-form
          :model="adminForm"
          :rules="adminRules"
          ref="adminFormRef"
          label-position="top"
          class="space-y-4"
      >
        <el-form-item label="账户" prop="username">
          <el-input
              v-model="adminForm.username"
              placeholder="请输入管理员账户/学号"
              class="custom-input"
              clearable
          >
            <template #prefix>
              <span class="text-slate-400 font-bold">👤</span>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
              v-model="adminForm.password"
              type="password"
              placeholder="请输入您的密码"
              class="custom-input"
              show-password
              @keyup.enter="handleAdminLogin"
          >
            <template #prefix>
              <span class="text-slate-400 font-bold">🔐</span>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="安全核算验证码" prop="captchaAnswer">
          <div class="flex items-center gap-3 w-full">
            <el-input
                v-model="adminForm.captchaAnswer"
                placeholder="算术题答案"
                class="custom-input flex-1"
                clearable
                @keyup.enter="handleAdminLogin"
            >
              <template #prefix>
                <span class="text-slate-400 font-bold">🧮</span>
              </template>
            </el-input>

            <div
                class="w-32 h-11 bg-slate-100 rounded-xl overflow-hidden border border-slate-200/80 cursor-pointer flex items-center justify-center shrink-0 shadow-sm hover:border-blue-400 transition-colors"
                title="看不清？点击刷新验证码"
                @click="fetchCaptchaImage"
                v-loading="captchaLoading"
            >
              <img
                  v-if="captchaImage"
                  :src="captchaImage"
                  alt="Captcha"
                  class="w-full h-full object-cover"
              />
              <span v-else class="text-[11px] text-slate-400 font-medium">加载中...</span>
            </div>
          </div>
        </el-form-item>

        <el-form-item class="!mt-8">
          <el-button
              type="primary"
              class="w-full h-12 font-bold tracking-wide text-base !rounded-xl bg-blue-600 border-none shadow-lg shadow-blue-500/25 hover:scale-[1.01] hover:bg-blue-700 transition-all"
              :loading="loading"
              @click="handleAdminLogin"
          >
            🚀 进入管理平台
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 微信扫码登录功能已由根管理员统一精简化移除 -->

      <div class="text-center text-xs text-slate-400 pt-6 border-t border-slate-100 flex items-center justify-center gap-1.5 font-medium">
        <span>🛡️</span>
        <span>信息工程学院 · 学习部</span>
      </div>

    </div>

    <!-- 首次登录微信特征绑定已由根管理员精简化移除 -->
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import request from '@/api/request'

const router = useRouter()
const loading = ref<boolean>(false)
const captchaLoading = ref<boolean>(false)
const adminFormRef = ref()

const captchaImage = ref<string>('')
const captchaId = ref<string>('')

// 扫码登录状态已精简移除

// 首次登录绑定状态已精简移除

const adminForm = reactive({
  username: '',
  password: '',
  captchaAnswer: ''
})

const adminRules = {
  username: [
    { required: true, message: '系统账户名不能为空', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '安全密码不能为空', trigger: 'blur' },
    { min: 6, message: '安全密码盾防御规则不能少于 6 位', trigger: 'blur' }
  ],
  captchaAnswer: [
    { required: true, message: '必须输入算术验证码得数以通过防刷拦截', trigger: 'blur' }
  ]
}

// 双模切换已精简移除

// -------------------------------------------------------------------------
// 1. 验证码图形拉取
// -------------------------------------------------------------------------
const fetchCaptchaImage = async () => {
  captchaLoading.value = true
  try {
    const cleanData: any = await request.get('/auth/captcha/generate')
    if (cleanData) {
      captchaId.value = cleanData.captchaId || ''
      const imgStr = cleanData.captchaImage || ''
      if (imgStr) {
        captchaImage.value = imgStr.startsWith('data:image') ? imgStr : 'data:image/png;base64,' + imgStr
      } else {
        captchaImage.value = ''
      }
    }
  } catch (e) {
    console.error('拉取验证码异常:', e)
    ElMessage.error('图形安全盾引擎初始化失败')
  } finally {
    captchaLoading.value = false
  }
}

// -------------------------------------------------------------------------
// 2. 账号密码安全登录与首次强制绑定拦截
// -------------------------------------------------------------------------
const handleAdminLogin = () => {
  adminFormRef.value.validate((valid: boolean) => {
    if (!valid) return
    loading.value = true

    request.post('/auth/login', {
      name: adminForm.username,
      password: adminForm.password,
      captchaId: captchaId.value,
      captchaAnswer: adminForm.captchaAnswer,
      deviceId: localStorage.getItem('admin_device_id') || 'WEB-DASHBOARD-SESSION-' + Math.random().toString(36).substr(2, 9)
    }).then((res: any) => {
      let target = res
      if (res && res.data) {
        target = res.data
      }

      // 🔍 首次登录微信特征绑定物理旁路通过，无需任何绑定直接放行

      if (res && (res.success || res.code === 200 || res.code === 0 || target.success)) {
        ElMessage.success('🎉 权限凭据盾验证成功，系统管理员已被核发安全登入许可！')
        completeLogin(res)
      } else {
        ElMessage.error(res.message || target.message || '登录异常，业务层拦截')
        fetchCaptchaImage()
        loading.value = false
      }
    }).catch(() => {
      loading.value = false
      fetchCaptchaImage()
    })
  })
}

const completeLogin = (res: any) => {
  let target = res
  if (res && res.data) {
    target = res.data
  }

  const tokenVal = res.token || target?.token || ''
  localStorage.setItem('userToken', tokenVal)
  localStorage.setItem('admin_token', tokenVal)

  if (target) {
    if (target.adminId) localStorage.setItem('admin_id', String(target.adminId))
    if (target.nickname) localStorage.setItem('admin_name', target.nickname)
    if (target.role) localStorage.setItem('admin_role', target.role)
  }

  router.push('/admin/dashboard')
}

// 微信扫码登录逻辑已精简移除

// 首次登录物理双因子绑定接口轮询已精减移除

onMounted(() => {
  fetchCaptchaImage()
})

// 销毁监听器已精简移除
</script>

<style scoped>
:deep(.el-form-item__label) {
  font-weight: 700;
  color: #334155;
  font-size: 13px;
  padding-bottom: 6px !important;
}

.custom-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  padding: 8px 12px;
  background-color: #f8fafc;
  border: 1px solid #e2e8f0;
  box-shadow: none !important;
  transition: all 0.25s ease;
}

.custom-input :deep(.el-input__wrapper:hover),
.custom-input :deep(.el-input__wrapper.is-focus) {
  background-color: #ffffff;
  border-color: #3b82f6;
  box-shadow: 0 0 0 1px #3b82f6 !important;
}
</style>