<template>
  <div class="h-screen flex flex-col overflow-hidden bg-slate-50 p-4 font-sans select-none">
    <div class="flex items-center justify-between pb-3 border-b border-slate-200 mb-3 shrink-0">
      <div class="flex items-center gap-2">
        <span class="text-xl font-black text-slate-800 tracking-tight">👑 体验资质注册专属特权审批中心</span>
        <el-tag type="warning" size="small" effect="dark" class="font-bold">待审豁免池</el-tag>
      </div>
    </div>

    <div class="flex-1 bg-white rounded-xl border border-slate-200 p-3 shadow-sm overflow-hidden flex flex-col min-h-0">
      <el-table :data="pendingUsers" height="100%" class="w-full text-xs" stripe size="default">
        <el-table-column type="index" label="序列" width="60" align="center" />

        <el-table-column label="体验用户资料" min-width="180">
          <template #default="{row}">
            <div class="flex items-center gap-2">
              <el-avatar :size="28" :src="row.avatar || '/logo.png'" />
              <div class="flex flex-col">
                <span class="font-bold text-slate-800">{{ row.real_name || '未命名体验者' }}</span>
                <span class="text-[10px] text-slate-400 font-mono">物理指纹: {{ row.current_device_id }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="class_name" label="申请划分群组" width="150" />
        <el-table-column prop="create_time" label="发起申请时间" width="180" />

        <el-table-column label="综合审查状态" width="120" align="center">
          <template #default>
            <el-tag type="info" size="small" class="font-bold">🔒 扣留在沙盒待批</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="特权审批裁决决断权" width="200" align="center" fixed="right">
          <template #default="{row}">
            <div class="flex gap-2 justify-center">
              <el-button type="success" size="small" class="font-bold" @click="handlePass(row)">一键核准放行</el-button>
              <el-button type="danger" size="small" plain @click="handleReject(row)">直接拒绝抹除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'

const pendingUsers = ref<any[]>([])

const loadPendingUsers = async () => {
  try {
    // 调用后端获取待审体验用户列表（后端过滤出未审批、status不为放行状态的记录）
    const res: any = await request.get('/admin/users/pending-experience')
    pendingUsers.value = res.data || []
  } catch {
    ElMessage.error('无法同步放行待审池')
  }
}

const handlePass = (row: any) => {
  ElMessageBox.confirm(`确定准予【${row.real_name}】获得临时体验特权，并放行至人员综合库吗？`, '特权放行核准', {
    type: 'success'
  }).then(async () => {
    // 改变状态落库：修改 account_status = 1，使用户正式进入主档案库的“体验用户”专栏
    await request.post('/admin/user/approve-experience', { userId: row.id, action: 'pass' })
    ElMessage.success('特权放行成功，已同步至全院大库中！')
    loadPendingUsers()
  })
}

const handleReject = (row: any) => {
  ElMessageBox.confirm(`确定要轰杀该条危险的非法体验注册申请吗？`, '拒绝隔离确认', {
    type: 'error'
  }).then(async () => {
    await request.post('/admin/user/approve-experience', { userId: row.id, action: 'reject' })
    ElMessage.warning('已成功拒绝并抹除沙盒垃圾申请记录')
    loadPendingUsers()
  })
}

onMounted(() => {
  loadPendingUsers()
})
</script>