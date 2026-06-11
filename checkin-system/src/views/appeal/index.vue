<template>
  <div class="p-6 max-w-[1600px] mx-auto bg-slate-50 min-h-screen flex flex-col gap-6 font-sans">
    <!-- 头部横幅 -->
    <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between pb-6 border-b border-slate-200 gap-4">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 flex items-center gap-2">
          ⚖️ 申诉审核大厅
        </h1>
        <p class="text-sm text-slate-500 mt-1">审阅并批改全院学生的异常打卡申诉，支持极速一键处理及奖励性补偿学风积分发放。</p>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <!-- 日期过滤 -->
        <el-date-picker
          v-model="filterDate"
          type="date"
          placeholder="选择日期过滤"
          format="YYYY-MM-DD"
          value-format="YYYY-MM-DD"
          @change="fetchAppeals"
          class="!w-40 shadow-sm"
          clearable
        />
        <!-- 模糊搜索 -->
        <el-input
          v-model="searchQuery"
          placeholder="搜姓名/学号/班级..."
          class="!w-64 shadow-sm"
          clearable
          @input="fetchAppeals"
        >
          <template #prefix>
            <span class="text-slate-400">🔍</span>
          </template>
        </el-input>
        <!-- 刷新按钮 -->
        <el-button type="primary" plain class="font-bold shadow-sm" @click="fetchAppeals">
          🔄 刷新大盘
        </el-button>
      </div>
    </div>

    <!-- 状态切换卡 -->
    <div class="bg-white rounded-xl border border-slate-200 p-1 shadow-sm shrink-0">
      <el-tabs v-model="activeTab" @tab-change="fetchAppeals" class="appeal-tabs px-4">
        <el-tab-pane label="⏳ 待审核工单" name="0"></el-tab-pane>
        <el-tab-pane label="🟢 已通过工单" name="1"></el-tab-pane>
        <el-tab-pane label="🔴 已驳回工单" name="2"></el-tab-pane>
      </el-tabs>
    </div>

    <!-- 数据大盘表格 -->
    <div class="bg-white rounded-xl border border-slate-200 p-4 shadow-sm flex-1 overflow-hidden flex flex-col min-h-[500px]">
      <el-table
        v-loading="loading"
        :data="tableData"
        style="width: 100%; height: 100%;"
        stripe
        class="text-slate-700"
      >
        <!-- 序号列 -->
        <el-table-column type="index" label="工单" width="70" align="center">
          <template #default="scope">
            <span class="font-mono text-xs text-slate-400 font-bold">#{{ scope.$index + 1 }}</span>
          </template>
        </el-table-column>

        <!-- 姓名 -->
        <el-table-column prop="name" label="姓名" width="120">
          <template #default="{ row }">
            <span class="font-bold text-slate-800">{{ row.name }}</span>
          </template>
        </el-table-column>

        <!-- 班级 -->
        <el-table-column prop="className" label="班级" width="140" show-overflow-tooltip />

        <!-- 学号 -->
        <el-table-column prop="studentId" label="学号" width="140">
          <template #default="{ row }">
            <span class="font-mono text-xs text-slate-500">{{ row.studentId }}</span>
          </template>
        </el-table-column>

        <!-- 凭证 -->
        <el-table-column label="申诉凭证" width="110" align="center">
          <template #default="{ row }">
            <el-image
              v-if="row.imageUrl"
              :src="row.imageUrl.split(',')[0]"
              :preview-src-list="row.imageUrl.split(',')"
              fit="cover"
              class="w-10 h-10 rounded-lg shrink-0 border border-slate-100 shadow-sm cursor-pointer hover:scale-105 transition-transform"
              preview-teleported
            />
            <span v-else class="text-xs text-slate-400 italic">无凭证图</span>
          </template>
        </el-table-column>

        <!-- 事由与分类 -->
        <el-table-column prop="reason" label="分类与事由" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="flex flex-col gap-1 py-1">
              <div>
                <span class="inline-block bg-amber-50 text-amber-700 border border-amber-200/60 text-[11px] px-1.5 py-0.5 rounded font-bold">
                  {{ getReasonPart0(row.reason).replace('【', '').replace('】', '') }}
                </span>
              </div>
              <p class="text-xs text-slate-600 line-clamp-2 leading-relaxed mt-0.5">
                {{ getReasonPart1(row.reason) }}
              </p>
            </div>
          </template>
        </el-table-column>

        <!-- 提交时间 -->
        <el-table-column prop="appealTime" label="提交时间" width="160">
          <template #default="{ row }">
            <span class="text-xs text-slate-400 font-mono">{{ row.appealTime }}</span>
          </template>
        </el-table-column>

        <!-- 处理档案（非待审核显示） -->
        <el-table-column label="处理档案" min-width="260" v-if="activeTab !== '0'">
          <template #default="{ row }">
            <div class="text-xs bg-slate-50 border border-slate-150 rounded-lg p-2.5 space-y-1">
              <div class="flex justify-between items-center pb-1 border-b border-slate-200/50">
                <span class="text-slate-500 font-medium">处理人: <b class="text-blue-600 font-semibold">{{ row.handlerName || '系统' }}</b></span>
                <span class="text-[10px] text-slate-400 font-mono">{{ row.handleTime }}</span>
              </div>
              <div class="text-slate-700 pt-0.5">
                <span class="text-slate-400 font-medium">意见:</span> {{ row.reply }}
              </div>
              <div v-if="row.rewardText" class="text-emerald-600 font-bold flex items-center gap-1 mt-1 text-[11px]">
                <span>🎁 追授补偿:</span>
                <span>【{{ row.rewardText }}】</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <!-- 操作 -->
        <el-table-column label="工单操作" width="180" fixed="right" align="center">
          <template #default="{ row }">
            <div class="flex gap-2 justify-center" v-if="row.status === 0">
              <el-button size="small" type="success" plain class="font-bold" @click="preparePass(row)">
                通过
              </el-button>
              <el-button size="small" type="danger" plain class="font-bold" @click="handleReject(row)">
                驳回
              </el-button>
            </div>
            <div v-else class="flex justify-center">
              <el-button
                size="small"
                type="danger"
                link
                class="font-semibold text-rose-600 hover:text-rose-800 transition-colors"
                @click="deleteRecord(row.id)"
              >
                🗑️ 抹除记录
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 审批通过弹窗 -->
    <el-dialog
      v-model="dialogReward"
      title="🎯 工单审批通过"
      width="480px"
      class="rounded-xl overflow-hidden custom-dialog"
      align-center
    >
      <el-form label-position="top" class="space-y-4">
        <el-form-item label="官方回复意见 (必填)" required>
          <el-input
            v-model="rewardForm.reply"
            type="textarea"
            :rows="3"
            placeholder="例如：经核实，为您补发对应分值奖励/补偿。"
            class="rounded-lg"
          />
        </el-form-item>

        <el-form-item label="是否需要附带补偿/奖励？">
          <el-radio-group v-model="rewardForm.needReward" class="w-full flex gap-3">
            <el-radio :value="false" border class="flex-1 !mr-0 text-center font-bold">
              不需要奖励
            </el-radio>
            <el-radio :value="true" border class="flex-1 !mr-0 text-center font-bold">
              核发荣誉补偿
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 奖励配置子卡片 -->
        <transition name="el-zoom-in-top">
          <div
            v-if="rewardForm.needReward"
            class="bg-slate-50 border border-slate-200 rounded-xl p-4 space-y-3 shadow-inner"
          >
            <div class="text-[11px] font-bold text-slate-400 uppercase tracking-wider mb-1 flex items-center gap-1">
              <span>🎁 设定发赠资产补偿参数</span>
            </div>
            <div class="grid grid-cols-2 gap-4">
              <el-form-item label="奖励资产类型" class="!mb-0">
                <el-select v-model="rewardForm.type" class="w-full">
                  <el-option label="🌟 赋能成长分 (EMP)" value="emp" />
                  <el-option label="🏅 思想德育分 (MORAL)" value="moral" />
                  <el-option label="🎫 补签卡" value="card" />
                </el-select>
              </el-form-item>
              <el-form-item label="奖励分值/数量" class="!mb-0">
                <el-input-number
                  v-model="rewardForm.value"
                  :min="1"
                  :max="100"
                  class="!w-full"
                />
              </el-form-item>
            </div>
          </div>
        </transition>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-3 pt-2">
          <el-button @click="dialogReward = false">取消</el-button>
          <el-button
            type="success"
            :loading="submitLoading"
            class="font-bold shadow-md shadow-emerald-50"
            @click="submitPass"
          >
            确认通过并结单
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'

// 申诉工单强类型数据定义
interface AppealRecord {
  id: number | string
  userId: number | string
  name: string
  className: string
  studentId: string
  imageUrl?: string
  reason: string
  appealTime: string
  status: number
  handlerName?: string
  reply?: string
  rewardText?: string
  handleTime?: string
}

// 补偿奖励表单强类型定义
interface RewardForm {
  reply: string
  needReward: boolean
  type: 'emp' | 'moral' | 'card'
  value: number
}

// 原子化响应式变量
const activeTab = ref<string>('0')
const searchQuery = ref<string>('')
const filterDate = ref<string>('')
const tableData = ref<AppealRecord[]>([])
const loading = ref<boolean>(false)
const submitLoading = ref<boolean>(false)

// 弹窗与控制变量
const dialogReward = ref<boolean>(false)
const currentRecord = ref<AppealRecord | null>(null)
const rewardForm = ref<RewardForm>({
  reply: '申诉已通过，数据已核实',
  needReward: false,
  type: 'emp',
  value: 10
})

// 分类事由安全切分工具函数
const getReasonPart0 = (reason: string | undefined): string => {
  if (!reason) return '【申诉】'
  return reason.split('】 ')[0] + '】 '
}

const getReasonPart1 = (reason: string | undefined): string => {
  if (!reason) return ''
  return reason.split('】 ')[1] || ''
}

// 1. 获取申诉列表数据 (精准支持筛选、条件过滤)
const fetchAppeals = async (isSilent = false) => {
  if (!isSilent) loading.value = true
  try {
    const res: any = await request.get('/api/admin/appeal/list', {
      params: {
        status: activeTab.value,
        keyword: searchQuery.value,
        date: filterDate.value
      }
    })
    tableData.value = res.data?.data || []
  } catch (e) {
    console.error('获取申诉列表失败:', e)
  } finally {
    if (!isSilent) loading.value = false
  }
}

// 2. 准备审批通过开窗
const preparePass = (row: AppealRecord) => {
  currentRecord.value = row
  rewardForm.value = {
    reply: '核实无误，申诉通过',
    needReward: false,
    type: 'emp',
    value: 10
  }
  dialogReward.value = true
}

// 3. 提交通过审批并结案
const submitPass = async () => {
  if (!currentRecord.value) return
  if (!rewardForm.value.reply) {
    ElMessage.warning('回复不能为空')
    return
  }

  submitLoading.value = true
  try {
    await request.post('/api/admin/appeal/process', {
      id: currentRecord.value.id,
      userId: currentRecord.value.userId, // 用于后端发奖绑定
      status: 1,
      reply: rewardForm.value.reply,
      handlerName: localStorage.getItem('admin_name') || '管理员',
      needReward: rewardForm.value.needReward,
      rewardType: rewardForm.value.type,
      rewardValue: rewardForm.value.value
    })
    ElMessage.success('工单已结案！')
    dialogReward.value = false
    await fetchAppeals()
  } catch (e) {
    ElMessage.error('处理失败')
  } finally {
    submitLoading.value = false
  }
}

// 4. 定点驳回工单（要求管理员输入明确原委）
const handleReject = (row: AppealRecord) => {
  ElMessageBox.prompt('请输入驳回的明确理由', '驳回工单', {
    confirmButtonText: '确认驳回',
    cancelButtonText: '取消',
    inputPlaceholder: '例如：所提交的申诉凭证不完整，请重新上传',
    inputValidator: (value) => {
      if (!value || !value.trim()) {
        return '驳回理由不能为空'
      }
      return true
    }
  }).then(async ({ value }) => {
    try {
      await request.post('/api/admin/appeal/process', {
        id: row.id,
        status: 2,
        reply: value || '申诉条件不足，已驳回',
        handlerName: localStorage.getItem('admin_name') || '管理员'
      })
      ElMessage.success('工单已驳回')
      await fetchAppeals()
    } catch (e) {
      ElMessage.error('操作失败')
    }
  }).catch(() => {})
}

// 5. 抹除记录（高危物理清理）
const deleteRecord = (id: number | string) => {
  ElMessageBox.confirm('彻底物理删除这条申诉记录？该操作不可逆，请谨慎操作。', '警告', {
    type: 'error',
    confirmButtonText: '确定抹除',
    cancelButtonText: '保留',
    confirmButtonClass: 'el-button--danger'
  }).then(async () => {
    try {
      await request.delete(`/api/admin/appeal/delete?id=${id}`)
      ElMessage.success('记录已抹除')
      await fetchAppeals()
    } catch (e) {
      ElMessage.error('删除记录失败')
    }
  }).catch(() => {})
}

// 全局无感同步监听处理器
const handleGlobalWsSync = (event: any) => {
  const socketPayload = event.detail
  // 💡 无感刷新：当收到打卡结果推送或申诉变动时，进行静默刷新
  if (socketPayload && (socketPayload.type === 'CHECKIN_RESULT' || socketPayload.type === 'NOTIFY')) {
    fetchAppeals(true)
  }
}

// 初始化钩子
onMounted(() => {
  fetchAppeals()
  window.addEventListener('global-ws-sync', handleGlobalWsSync)
})

onUnmounted(() => {
  window.removeEventListener('global-ws-sync', handleGlobalWsSync)
})
</script>

<style scoped>
.appeal-tabs :deep(.el-tabs__item) {
  color: #64748b;
  font-size: 14px;
  font-weight: 600;
  transition: all 0.3s;
}
.appeal-tabs :deep(.el-tabs__item.is-active) {
  color: #059669; /* Emerald 600 color */
}
.appeal-tabs :deep(.el-tabs__active-bar) {
  background-color: #059669;
  height: 3px;
  border-radius: 9999px;
}
.appeal-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: #f1f5f9;
}

:deep(.el-table) {
  background-color: transparent !important;
}
:deep(.el-table tr), :deep(.el-table th.el-table__cell) {
  background-color: transparent !important;
  color: #334155;
  border-bottom: 1px solid #f1f5f9;
}
:deep(.el-table--striped .el-table__body tr.el-table__row--striped td.el-table__cell) {
  background: #f8fafc !important;
}
:deep(.el-table tbody tr:hover > td) {
  background-color: #f0fdf4 !important; /* Soft emerald background on hover */
}
:deep(.el-table::before) {
  display: none;
}

.custom-dialog :deep(.el-dialog__header) {
  padding-bottom: 15px;
  border-b: 1px solid #f1f5f9;
  margin-right: 0;
}
.custom-dialog :deep(.el-dialog__body) {
  padding-top: 20px;
  padding-bottom: 20px;
}
</style>
