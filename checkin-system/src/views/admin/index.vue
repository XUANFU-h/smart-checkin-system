<template>
  <div class="p-4 max-w-7xl mx-auto min-h-full select-none">
    <div class="flex items-center justify-between pb-6 border-b border-slate-200/80 mb-6">
      <div class="flex items-center gap-2">
        <span class="text-xl">🛡️</span>
        <span class="text-lg font-black text-slate-800 tracking-tight">综合管理员核心指挥控制台</span>
        <el-tag v-if="currentLoginUser === '胡景耀'" type="danger" size="small" class="font-bold animate-pulse">SUPER_OWNER 超管状态</el-tag>
      </div>
      <div>
        <el-button type="primary" size="default" style="border-radius: 10px;" @click="openCreateDialog">
          ➕ 签发全新管理凭证
        </el-button>
      </div>
    </div>

    <div class="bg-white rounded-2xl shadow-sm border border-slate-200/80 overflow-hidden">
      <el-table :data="adminList" v-loading="loading" style="width: 100%" class="text-slate-700">
        <el-table-column prop="id" label="UID" width="70" align="center" />
        <el-table-column prop="username" label="登录账户/真实姓名" min-width="140">
          <template #default="scope">
            <span class="font-bold text-slate-800">{{ scope.row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="studentId" label="绑定的唯一学号" width="130" align="center">
          <template #default="scope">
            <span class="font-mono text-slate-600 font-semibold">{{ scope.row.studentId || 'N/A' }}</span>
          </template>
        </el-table-column>
        <!-- 微信特征绑定与密钥列已简化移除 -->
        <el-table-column prop="role" label="中枢特权定级" width="160">
          <template #default="scope">
            <el-tag :type="scope.row.username === '胡景耀' ? 'danger' : 'warning'" effect="plain" class="font-bold" style="border-radius: 6px;">
              {{ scope.row.username === '胡景耀' ? 'OWNER_DEVELOPER' : 'COMPREHENSIVE_ADMIN' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="accountStatus" label="履职风控状态" width="130" align="center">
          <template #default="scope">
            <el-tag :type="scope.row.accountStatus === 0 ? 'success' : 'danger'" style="border-radius: 6px;">
              {{ scope.row.accountStatus === 0 ? '🟢 正常履职' : '🔴 风险隔离' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="安全风控最高统帅指令" min-width="360" fixed="right">
          <template #default="scope">
            <div class="flex flex-wrap gap-1.5">
              <el-button size="small" type="warning" plain style="border-radius: 6px;" @click="openResetPassword(scope.row)">重置密码</el-button>

              <template v-if="currentLoginUser === '胡景耀' && scope.row.username !== '胡景耀'">
                <el-button size="small" type="primary" plain style="border-radius: 6px;" @click="handleRenameAdmin(scope.row)">更名</el-button>
                <el-button
                    size="small"
                    :type="scope.row.accountStatus === 0 ? 'danger' : 'success'"
                    plain
                    style="border-radius: 6px;"
                    @click="toggleBanStatus(scope.row)"
                >
                  {{ scope.row.accountStatus === 0 ? '风险封禁' : '安全解除' }}
                </el-button>
                <el-button size="small" type="danger" plain style="border-radius: 6px;" @click="handleForceKick(scope.row)">强踢下线</el-button>
                <el-button size="small" type="danger" style="border-radius: 6px;" @click="handleDeleteAdmin(scope.row)">物理驱逐</el-button>
              </template>
              <span v-else-if="scope.row.username === '胡景耀'" class="text-[11px] text-slate-400 font-bold px-2 py-1">最高核心受安全屏蔽保护</span>
              <span v-else class="text-[11px] text-slate-400 font-medium px-2 py-1">暂无处置越权</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 核发全新管理员凭证 Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '魔改管理员凭证' : '核发全新综合管理员凭证'" width="420px" style="border-radius: 20px;">
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="👤 核心登录账号/真实中文姓名" prop="username">
          <el-input v-model="form.username" placeholder="请输入真实姓名（登录、签字统一对齐）" clearable />
        </el-form-item>
        <el-form-item label="🆔 绑定学号（强制 10 位数）" prop="student_id">
          <el-input v-model="form.student_id" placeholder="请输入该人员的 10 位数学生学号" maxlength="10" show-word-limit clearable />
        </el-form-item>
        <el-form-item label="🔑 初始安全进入密码（不填则默认 123456）" prop="password">
          <el-input v-model="form.password" type="password" placeholder="默认密码为 123456" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button style="border-radius: 8px;" @click="dialogVisible = false">放弃</el-button>
          <el-button type="primary" style="border-radius: 8px;" :loading="submitLoading" @click="submitForm">确认签发入库</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 综合管理员凭证已签发提示框已精简 -->
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const adminList = ref<any[]>([])
const formRef = ref()
const currentLoginUser = ref<string>('')

const form = reactive({
  id: 0,
  username: '',
  student_id: '',
  password: ''
})

const rules = {
  username: [{ required: true, message: '系统账户名/姓名不能为空', trigger: 'blur' }],
  student_id: [
    { required: true, message: '绑定学号不能为空', trigger: 'blur' },
    { len: 10, message: '学号必须强制为 10 位数！', trigger: 'blur' }
  ]
}

const fetchAdminUsers = () => {
  loading.value = true
  request({ url: '/api/superadmin/sub-admins', method: 'GET' }).then((res: any) => {
    loading.value = false
    const targetData = res.data ? res.data : res
    if (Array.isArray(targetData)) {
      adminList.value = targetData
    }
  }).catch(() => { loading.value = false })
}

const openCreateDialog = () => {
  isEdit.value = false
  form.id = 0
  form.username = ''
  form.student_id = ''
  form.password = ''
  dialogVisible.value = true
}

const submitForm = () => {
  formRef.value.validate((valid: boolean) => {
    if (!valid) return
    submitLoading.value = true

    request({
      url: '/api/superadmin/menu-perms',
      method: 'POST',
      data: {
        action: 'save',
        username: form.username,
        student_id: form.student_id,
        password: form.password
      }
    }).then(() => {
      submitLoading.value = false
      dialogVisible.value = false
      ElMessage.success('🎉 综合管理员账号已成功签发！')
      
      // 微信绑定辅助指引已精简化移除
      fetchAdminUsers()
    }).catch(() => { submitLoading.value = false })
  })
}

// 1. 中枢密码重置：弹窗二次确认后直接重置为 123456
const openResetPassword = (row: any) => {
  ElMessageBox.confirm(`确定要执行最高风控命令，将管理员【${row.username}】的密码重置为默认密码【123456】吗？`, '中枢密码重置确认', {
    type: 'warning',
    confirmButtonText: '确定重置',
    cancelButtonText: '取消'
  }).then(() => {
    request({
      url: '/api/admin/auth/reset-self',
      method: 'POST',
      data: { adminId: row.id, newPassword: '123456' }
    }).then(() => {
      ElMessage.success(`管理员【${row.username}】密码已成功重置为 123456，且该账号被强制物理踢下机！`)
      fetchAdminUsers()
    })
  })
}

// 2. 更名
const handleRenameAdmin = (row: any) => {
  ElMessageBox.prompt(
      `请输入为该管理员账户【${row.username}】设定的全新登录账户名`,
      '超管最高更名魔改',
      {
        confirmButtonText: '强制更名并驱逐重登',
        cancelButtonText: '取消',
        inputValue: row.username,
        inputValidator: (value: string) => {
          if (!value || value.trim() === '') return '更名内容不能为空'
          return true
        }
      }
  ).then(({ value }) => {
    request({
      url: '/api/admin/user/rename', // Wait, or custom route
      method: 'POST',
      data: { adminId: row.id, newUsername: value }
    }).then(() => {
      ElMessage.success(`更名成功：已将其变更为【${value}】，其原有的所有登录态已全部作废强制重登！`)
      fetchAdminUsers()
    })
  })
}

// 3. 封禁/解封
const toggleBanStatus = (row: any) => {
  const targetStatus = row.accountStatus === 0 ? 1 : 0
  const actionText = targetStatus === 1 ? '剥夺权限并风险隔离(封号)' : '安全解除恢复履职'

  ElMessageBox.confirm(`警告：确定要执行最高风控命令，对管理员【${row.username}】进行${actionText}操作吗？`, '最高防线处决', {
    type: 'warning'
  }).then(() => {
    request({
      url: '/api/superadmin/toggle-status',
      method: 'POST',
      data: { adminId: row.id, status: targetStatus }
    }).then(() => {
      ElMessage.success(`统帅防线执行完毕：该管理权限已${actionText}`)
      fetchAdminUsers()
    })
  })
}

// 4. 强踢下线
const handleForceKick = (row: any) => {
  ElMessageBox.confirm(`确定要启动安全锁，对管理员【${row.username}】所有的在线设备执行即时强踢下线吗？`, '强制阻断强踢', {
    type: 'error'
  }).then(() => {
    request({
      url: '/api/superadmin/kick',
      method: 'POST',
      data: { adminId: row.id }
    }).then(() => {
      ElMessage.success(`已发射互踢指令，管理员【${row.username}】的所有在线网络通道已被强行切断！`)
    })
  })
}

// 5. 物理注销驱逐
const handleDeleteAdmin = (row: any) => {
  ElMessageBox.confirm(`高危：该操作将永久从数据库 admin_user 中注销擦除【${row.username}】的全部痕迹，确定驱逐吗？`, '高危物理抹除', {
    type: 'error',
    confirmButtonClass: 'el-button--danger'
  }).then(() => {
    request({
      url: '/api/admin/user/delete', // Or custom route
      method: 'DELETE',
      data: { userId: row.id }
    }).then(() => {
      ElMessage.success('该账户已从核心成员库中彻底物理蒸发名除。')
      fetchAdminUsers()
    })
  })
}

onMounted(() => {
  currentLoginUser.value = localStorage.getItem('admin_name') || ''
  fetchAdminUsers()
})
</script>