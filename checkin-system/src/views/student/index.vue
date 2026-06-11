<template>
  <div class="h-screen flex flex-col overflow-hidden bg-slate-50 p-4 font-sans select-none">
    
    <!-- 顶部控制条 (无标题，视觉极致减法) -->
    <div class="flex items-center justify-between pb-3 border-b border-slate-200 mb-3 shrink-0">
      <div class="flex items-center gap-3">
        <el-button-group>
          <el-button v-if="hasAction('user_kick')" type="warning" size="default" plain @click="handleBatchKick">
            {{ selectedUserIds.length > 0 ? '⚠️ 选中账户强制下线' : '⚠️ 全员应急强制作退' }}
          </el-button>
          <el-button v-if="hasAction('user_ban')" type="danger" size="default" plain @click="handleBatchBan">
            {{ selectedUserIds.length > 0 ? '🔒 选中账户临时封禁 (5分钟)' : '🔒 全系一键应急熔断封禁 (5分钟)' }}
          </el-button>
        </el-button-group>
      </div>
      
      <div class="flex items-center gap-3">
        <el-button type="success" size="default" plain :loading="exportLoading" @click="handleExportExcel">
          一键导出Excel
        </el-button>
        <el-input v-model="searchQuery" placeholder="智能检索姓名/学号/班级..." class="w-64" size="default" clearable>
          <template #prefix>🔍</template>
        </el-input>
      </div>
    </div>

    <!-- 顶置扁平化班级过滤栏 -->
    <div class="bg-white border border-slate-200 rounded-xl p-2 mb-3 shadow-sm flex flex-wrap gap-2 items-center shrink-0">
      <div
          class="px-4 py-1.5 rounded-lg cursor-pointer text-xs font-bold transition-all"
          :class="currentClass === 'all' ? 'bg-emerald-600 text-white shadow-sm' : 'text-slate-600 hover:bg-slate-100'"
          @click="handleClassTabClick('all')"
      >
        🗃️ 全部班级
      </div>
      <div
          class="px-4 py-1.5 rounded-lg cursor-pointer text-xs font-bold transition-all"
          :class="currentClass === 'experience' ? 'bg-amber-500 text-white shadow-sm' : 'text-amber-600 hover:bg-amber-50'"
          @click="handleClassTabClick('experience')"
      >
        🌟 体验用户
      </div>
      <div
          v-for="c in dynamicClassTabs"
          :key="c"
          class="px-4 py-1.5 rounded-lg cursor-pointer text-xs font-medium transition-all"
          :class="currentClass === c ? 'bg-emerald-600 text-white shadow-sm font-bold' : 'text-slate-600 hover:bg-slate-100'"
          @click="handleClassTabClick(c)"
      >
        👥 {{ c }}
      </div>
      <div class="ml-auto border-l border-slate-200 pl-2">
        <div
            class="px-4 py-1.5 rounded-lg cursor-pointer text-xs font-black transition-all"
            :class="currentClass === 'blacklist' ? 'bg-red-600 text-white shadow-sm' : 'text-red-500 hover:bg-red-50'"
            @click="handleClassTabClick('blacklist')"
        >
          🌑 设备黑名单库
        </div>
      </div>
    </div>

    <!-- 主体布局 (左侧花名册，右侧 2X3 矩阵特权面板) -->
    <div class="flex-1 flex gap-4 min-h-0">
      
      <!-- 左侧：学生花名册 -->
      <div class="flex-1 bg-white rounded-xl border border-slate-200 p-3 shadow-sm overflow-hidden flex flex-col">
        <el-table
            :data="pagedUsers"
            height="100%"
            @sort-change="handleSortChange"
            @selection-change="handleSelectionChange"
            @row-click="handleRowClick"
            :row-class-name="getRowClassName"
            class="w-full text-xs cursor-pointer"
            stripe
            size="default"
        >
          <!-- 序号多选列 -->
          <el-table-column type="selection" width="50" align="center" />
          <el-table-column type="index" label="序号" width="55" align="center" />

          <!-- 头像本地文件直传与姓名 -->
          <el-table-column label="学生信息" min-width="160">
            <template #default="{row}">
              <div class="flex items-center gap-2" @click.stop>
                <el-upload
                    action="/api/checkin/upload"
                    :headers="{ 'Authorization': 'Bearer ' + token }"
                    name="file"
                    :show-file-list="false"
                    :on-success="handleAvatarSuccessWrapper(row)"
                    :before-upload="beforeAvatarUpload"
                    class="flex shrink-0"
                >
                  <el-tooltip content="点击上传更换本地头像" placement="top">
                    <el-avatar
                        :size="30"
                        :src="row.avatar || '/logo.png'"
                        class="border border-slate-100 hover:scale-110 transition-transform cursor-pointer"
                    />
                  </el-tooltip>
                </el-upload>
                <div class="flex flex-col min-w-0">
                  <span class="font-bold text-slate-800 truncate">{{ row.real_name || row.nickname || '未实名' }}</span>
                  <span class="text-[10px] text-slate-400 font-mono truncate">{{ row.student_id || '无学号' }}</span>
                </div>
              </div>
            </template>
          </el-table-column>

          <!-- 身份分类 (精炼为三种) -->
          <el-table-column label="身份" width="95" align="center">
            <template #default="{row}">
              <el-tag
                  :type="row.user_type === 2 ? 'danger' : (row.user_type === 1 ? 'warning' : 'success')"
                  effect="dark"
                  size="small"
                  class="font-bold scale-90"
              >
                {{ row.user_type === 2 ? '管理员' : (row.user_type === 1 ? 'Beta内测' : '正式用户') }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column prop="class_name" label="所属班级" width="130" show-overflow-tooltip />

          <el-table-column prop="empowerment_score" label="赋能分" width="105" align="center" sortable="custom">
            <template #default="{row}">
              <div class="font-black px-2 py-0.5 rounded bg-amber-50 text-amber-600 font-mono text-center">
                {{ row.empowerment_score ?? 0 }}
              </div>
            </template>
          </el-table-column>

          <el-table-column prop="moral_score" label="德育分" width="105" align="center" sortable="custom">
            <template #default="{row}">
              <div class="font-black px-2 py-0.5 rounded bg-emerald-50 text-emerald-600 font-mono text-center">
                {{ row.moral_score ?? 0 }}
              </div>
            </template>
          </el-table-column>

          <el-table-column prop="makeup_cards" label="补签卡" width="95" align="center">
            <template #default="{row}">
              <div class="font-bold px-2 py-0.5 rounded bg-indigo-50 text-indigo-600 font-mono text-center">
                {{ row.makeup_cards ?? 0 }} 张
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 右侧：2X3 大管理员最高特权控制矩阵面板 -->
      <div class="w-80 shrink-0 bg-white rounded-xl border border-slate-200 p-4 shadow-sm flex flex-col gap-4">
        <div class="border-b border-slate-100 pb-2.5">
          <h3 class="font-bold text-slate-800 text-sm flex items-center gap-1.5">🕹️ 大管理员最高特权控制矩阵</h3>
          <p class="text-[11px] text-slate-400 mt-1">选定单个用户或使用表格左侧复选框进行批量干预调度。</p>
        </div>

        <!-- 当前操作对象指示器 -->
        <div class="bg-slate-50 border border-slate-100 p-3 rounded-lg text-xs flex flex-col gap-2">
          <div class="text-slate-500 font-bold flex justify-between items-center">
            <span>🎯 选中状态:</span>
            <el-button v-if="currUser || selectedUserIds.length > 0" type="primary" link size="small" @click="clearActiveSelection">清除选择</el-button>
          </div>
          <div v-if="selectedUserIds.length > 0" class="text-red-600 font-black flex items-center gap-1">
            <span>📦</span> 批量处理中：已勾选 {{ selectedUserIds.length }} 名学生
          </div>
          <div v-else-if="currUser" class="text-indigo-600 font-black flex items-center gap-2">
            <el-avatar :size="20" :src="currUser.avatar || '/logo.png'" />
            <span class="truncate">{{ currUser.real_name || currUser.nickname }} (单人精准干预)</span>
          </div>
          <div v-else class="text-slate-400 italic text-center py-1">
            暂无选定目标 (请点击左表行或勾选序号)
          </div>
        </div>

        <!-- 2X3 核心大管理员干预控制矩阵 -->
        <div class="grid grid-cols-3 gap-2.5 my-2">
          <!-- 修改 -->
          <el-button
              v-if="hasAction('user_rename')"
              type="primary"
              class="font-bold !m-0 !h-16 flex flex-col gap-1 text-xs shadow-sm shadow-blue-50"
              :disabled="!currUser && selectedUserIds.length === 0"
              @click="handleMatrixModify"
          >
            <span class="text-base">🔧</span><span>修改</span>
          </el-button>
          <!-- 奖励 -->
          <el-button
              v-if="hasAction('score_modify')"
              type="warning"
              class="font-bold !m-0 !h-16 flex flex-col gap-1 text-xs shadow-sm shadow-orange-50"
              :disabled="!currUser && selectedUserIds.length === 0"
              @click="handleMatrixReward"
          >
            <span class="text-base">🎁</span><span>奖励</span>
          </el-button>
          <!-- 处罚 -->
          <el-button
              v-if="hasAction('score_modify')"
              type="danger"
              class="font-bold !m-0 !h-16 flex flex-col gap-1 text-xs shadow-sm shadow-red-50"
              :disabled="!currUser && selectedUserIds.length === 0"
              @click="handleMatrixPunish"
          >
            <span class="text-base">📉</span><span>处罚</span>
          </el-button>

          <!-- 下线 -->
          <el-button
              v-if="hasAction('user_kick')"
              type="info"
              class="font-bold !m-0 !h-16 !bg-slate-700 !text-white flex flex-col gap-1 text-xs"
              :disabled="!currUser && selectedUserIds.length === 0"
              @click="handleMatrixKick"
          >
            <span class="text-base">⚡</span><span>下线</span>
          </el-button>
          <!-- 拉黑 -->
          <el-button
              v-if="hasAction('user_ban')"
              type="danger"
              class="font-bold !m-0 !h-16 flex flex-col gap-1 text-xs"
              :disabled="selectedUserIds.length > 0 || !currUser"
              @click="handleMatrixBlacklist"
          >
            <span class="text-base">🌑</span><span>拉黑</span>
          </el-button>
          <!-- 删除 -->
          <el-button
              v-if="hasAction('user_delete')"
              type="danger"
              plain
              class="font-bold !m-0 !h-16 flex flex-col gap-1 text-xs"
              :disabled="!currUser && selectedUserIds.length === 0"
              @click="handleMatrixDelete"
          >
            <span class="text-base">🔥</span><span>删除</span>
          </el-button>
        </div>

        <div class="mt-auto border-t border-slate-100 pt-3 flex flex-col gap-2">
          <el-button
              size="small"
              type="primary"
              plain
              class="w-full font-bold"
              :disabled="selectedUserIds.length > 0 || !currUser"
              @click="prepareTrackManage(currUser!)"
          >
            🏃 重组该生赛道矩阵
          </el-button>
        </div>
      </div>

    </div>

    <!-- 1. 修改学籍及一键重置密码弹窗 -->
    <el-dialog v-model="dlgUser" :title="`🔧 特权更正学生基本底册`" width="420px" destroy-on-close>
      <el-form :model="editingUser" label-width="80px" size="small" class="space-y-3">
        <el-form-item label="真实真名">
          <el-input v-model="editingUser.real_name" placeholder="更正姓名" />
        </el-form-item>
        <el-form-item label="班级名录">
          <el-select v-model="editingUser.class_name" placeholder="请选择或更正班级" class="w-full">
            <el-option v-for="c in standardClasses" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="标准学号">
          <el-input v-model="editingUser.student_id" placeholder="更正学号" />
        </el-form-item>
        <el-form-item label="用户身份">
          <el-select v-model="editingUser.user_type" placeholder="调整用户组权限" class="w-full">
            <el-option label="正式用户" :value="0" />
            <el-option label="Beta内测" :value="1" />
            <el-option label="管理员" :value="2" />
          </el-select>
        </el-form-item>
      </el-form>
      
      <div class="border-t border-slate-100 pt-4 mt-4">
        <el-button type="danger" size="default" class="w-full font-bold shadow-md shadow-red-50" @click="handleResetPassword">
          ⚠️ 一键重置登录密码 (全网踢下线重置)
        </el-button>
      </div>

      <template #footer>
        <el-button size="small" @click="dlgUser = false">取消</el-button>
        <el-button size="small" type="primary" @click="doSaveUserChanges">确认特权订正</el-button>
      </template>
    </el-dialog>

    <!-- 2. 发放奖励及撤回加分日志弹窗 -->
    <el-dialog v-model="dlgReward" :title="selectedUserIds.length > 0 ? `🎁 批量向 ${selectedUserIds.length} 名学生签发奖励` : `🎁 向【${currUser?.real_name || currUser?.nickname}】签发量化奖励`" width="650px" destroy-on-close>
      <div class="bg-slate-50 border border-slate-200 p-4 rounded-xl mb-4">
        <el-form :model="rewardForm" label-position="top" class="grid grid-cols-2 gap-4">
          <el-form-item label="量化加分指标类别" class="!mb-1">
            <el-select v-model="rewardForm.type" class="w-full" size="small">
              <el-option label="🌟 赋能成长积分 (EMP)" value="emp" />
              <el-option label="🏅 思想德育积分 (MORAL)" value="moral" />
              <el-option label="🪙 补签卡入账 (MAKEUP)" value="makeup" />
            </el-select>
          </el-form-item>
          <el-form-item label="增发额度" class="!mb-1">
            <el-input-number v-model="rewardForm.score" :min="1" :max="maxSingleAdd" class="!w-full" size="small" />
          </el-form-item>
          <el-form-item label="奖励公示事由 (必填)" class="col-span-2 !mb-1">
            <el-input v-model="rewardForm.reason" size="small" placeholder="请输入合理的公示奖励事由，如：参与晚会筹备" />
          </el-form-item>
        </el-form>
        <el-button type="warning" size="small" class="w-full font-bold mt-4 shadow-md shadow-orange-50" @click="doSubmitReward">
          立刻签发增分令
        </el-button>
      </div>

      <!-- 单人历史级联展示与撤销功能 -->
      <div v-if="selectedUserIds.length === 0" class="border-t border-slate-100 pt-3">
        <div class="text-xs font-bold text-slate-600 mb-2">🏅 历史获得奖励/加分细目明细：</div>
        <el-table :data="filteredRewardHistory" height="200px" stripe size="small" class="text-[11px]">
          <el-table-column prop="create_time" label="发生时间" width="150" />
          <el-table-column prop="reason" label="事由说明" min-width="180" />
          <el-table-column label="增数" width="80" align="center">
            <template #default="{row}">
              <span class="text-emerald-600 font-bold">+{{ row.score }}</span>
            </template>
          </el-table-column>
          <el-table-column label="动作" width="80" align="center">
            <template #default="{row}">
              <el-button size="small" type="danger" link @click="doRevokeReward(row.id)">撤回</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 3. 下发干预纪律惩戒/封禁弹窗 -->
    <el-dialog v-model="dlgPunish" :title="selectedUserIds.length > 0 ? `📉 批量向 ${selectedUserIds.length} 名学生下发惩戒` : `📉 对【${currUser?.real_name || currUser?.nickname}】下发干预纪律惩戒`" width="650px" destroy-on-close>
      <div class="bg-red-50/50 border border-red-100 p-4 rounded-xl mb-4">
        <el-form :model="punishForm" label-position="top" class="grid grid-cols-2 gap-4">
          <el-form-item label="纪律干预分类" class="!mb-1">
            <el-select v-model="punishForm.type" class="w-full" size="small">
              <el-option label="📉 扣减赋能成长分" value="emp" />
              <el-option label="📉 扣减思想德育分" value="moral" />
              <el-option label="🔒 一键执行安全封禁锁定" value="ban" />
            </el-select>
          </el-form-item>
          
          <el-form-item v-if="punishForm.type === 'ban'" label="封禁惩戒时长阶梯" class="!mb-1">
            <el-select v-model="punishForm.duration" class="w-full" size="small">
              <el-option label="5 分钟 (应急演练/临时熔断)" value="5m" />
              <el-option label="10 分钟" value="10m" />
              <el-option label="30 分钟" value="30m" />
              <el-option label="24 小时" value="24h" />
              <el-option label="永久物理锁死" value="forever" />
            </el-select>
          </el-form-item>
          <el-form-item v-else label="扣减分值" class="!mb-1">
            <el-input-number v-model="punishForm.score" :min="1" :max="maxSingleSub" class="!w-full" size="small" />
          </el-form-item>
          
          <el-form-item label="纪律处分惩戒原委 (必填)" class="col-span-2 !mb-1">
            <el-input v-model="punishForm.reason" size="small" placeholder="请输入违规说明作为存证，如：代人签到代打卡" />
          </el-form-item>
        </el-form>
        <el-button type="danger" size="small" class="w-full font-bold mt-4 shadow-md shadow-red-50" @click="doSubmitPunish">
          强制下发纪律干预
        </el-button>
      </div>

      <!-- 单人历史惩戒展现与撤回功能 -->
      <div v-if="selectedUserIds.length === 0" class="border-t border-slate-100 pt-3">
        <div class="text-xs font-bold text-red-600 mb-2">📉 历史处分惩戒底册记录：</div>
        <el-table :data="filteredPunishHistory" height="200px" stripe size="small" class="text-[11px]">
          <el-table-column prop="create_time" label="发生时间" width="150" />
          <el-table-column prop="reason" label="违规核定原委说明" min-width="180" />
          <el-table-column label="处罚特征" width="90" align="center">
            <template #default="{row}">
              <el-tag v-if="row.score_type === 'ban'" type="danger" size="small">安全封锁</el-tag>
              <span v-else class="text-red-500 font-bold">{{ row.score }}分</span>
            </template>
          </el-table-column>
          <el-table-column label="动作" width="80" align="center">
            <template #default="{row}">
              <el-button size="small" type="danger" link @click="doRevokePunish(row.id)">撤销</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 4. 个性化赛道分配调整弹窗 -->
    <el-dialog v-model="dlgTrack" :title="`🏃 【${currUser?.real_name || currUser?.nickname}】个性化赛道调整`" width="460px" destroy-on-close>
      <div class="text-xs font-bold text-slate-700 mb-2">当前运行中的打卡修行足迹：</div>
      <div class="grid grid-cols-2 gap-2 mb-3">
        <div v-for="t in currUser?.tracks" :key="t.id" class="p-2 bg-slate-50 border border-slate-200 rounded-lg text-center">
          <div class="text-xs font-bold text-slate-800">🏃 {{ t.name }}</div>
        </div>
        <div v-if="!currUser?.tracks?.length" class="col-span-2 text-center text-slate-400 text-xs py-2">暂未加入任何赛道</div>
      </div>
      <el-divider><span class="text-[10px] text-slate-400">可分配赛道池</span></el-divider>
      <el-checkbox-group v-model="selectedTrackIds" class="grid grid-cols-2 gap-2">
        <el-checkbox v-for="track in allAvailableTracks" :key="track.id" :label="track.id" border class="w-full !mr-0 !h-8 text-xs">
          {{ track.name }}
        </el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button size="small" @click="dlgTrack = false">取消</el-button>
        <el-button size="small" type="primary" @click="doSaveUserTracks">保存配置</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'
import { hasAction } from '@/utils/permission'

interface Track { id: number; name: string }
interface User { id: number; nickname: string; real_name: string; student_id: string; class_name: string; avatar: string; empowerment_score: number; moral_score: number; makeup_cards: number; account_status: number; current_device_id: string; user_type: number; tracks?: Track[] }
interface HistoryLog { id: number; create_time: string; reason: string; operator: string; score: number; score_type: string; action: string }
interface CheckinExportRecord { userId: number; user_id?: number; trackId: number; track_id?: number; checkinDate: string; checkin_date?: string; status: number; trackName?: string; track_name?: string }

const users = ref<User[]>([])
const searchQuery = ref('')
const currentClass = ref('all')
const isGlobalBanned = ref(false)
const currentSort = ref({ prop: '', order: '' })

const currUser = ref<User | null>(null)
const scoreLogs = ref<HistoryLog[]>([])
const allAvailableTracks = ref<Track[]>([])
const selectedTrackIds = ref<number[]>([])
const selectedUserIds = ref<number[]>([])
const standardClasses = ref<string[]>([])
const exportLoading = ref(false)

const dlgUser = ref(false)
const dlgReward = ref(false)
const dlgPunish = ref(false)
const dlgTrack = ref(false)

const editingUser = ref<Partial<User>>({})
const rewardForm = ref({ userId: 0, type: 'emp', score: 10, reason: '' })
const punishForm = ref({ userId: 0, type: 'emp', score: 10, duration: '5m', reason: '' })

const maxSingleAdd = ref(50)
const maxSingleSub = ref(50)

const fetchSystemLimits = async () => {
  try {
    const res: any = await request.get('/api/admin/settings/get')
    let d: any = null
    
    // 🛡️ 智能兼容防抖解包：完美穿透 Proxy 代理与多层嵌套数据体
    if (res.data) {
      if (res.data.max_single_add !== undefined || res.data.max_single_sub !== undefined) {
        d = res.data
      } else if (res.data.data) {
        d = res.data.data
      }
    }
    if (!d && (res.max_single_add !== undefined || res.max_single_sub !== undefined)) {
      d = res
    }

    if (d) {
      const maxAdd = d.max_single_add || d.maxSingleAdd
      if (maxAdd) maxSingleAdd.value = parseInt(String(maxAdd))
      
      const maxSub = d.max_single_sub || d.maxSingleSub
      if (maxSub) maxSingleSub.value = parseInt(String(maxSub))
    }
  } catch (error) {
    console.error('获取单次奖扣上限异常:', error)
  }
}

const token = localStorage.getItem('userToken') || localStorage.getItem('admin_token') || ''

// 智能动态生成顶置扁平班级池，不掺杂体验和黑号
const dynamicClassTabs = computed(() => {
  return [...new Set(users.value
      .filter(u => u.account_status !== 1 && u.account_status !== 2)
      .map(u => u.class_name)
      .filter(Boolean))].sort()
})

const handleSelectionChange = (val: User[]) => {
  selectedUserIds.value = val.map(u => u.id)
}

const handleRowClick = (row: User) => {
  currUser.value = row
}

const clearActiveSelection = () => {
  currUser.value = null
  selectedUserIds.value = []
}

const handleClassTabClick = (tab: string) => {
  currentClass.value = tab
  clearActiveSelection()
}

const getRowClassName = ({ row }: { row: User }) => {
  if (currUser.value && currUser.value.id === row.id) {
    return 'bg-indigo-50/50 font-bold border-l-4 border-l-indigo-600'
  }
  return ''
}

const handleSortChange = ({ prop, order }: { prop: string, order: string }) => {
  currentSort.value = { prop, order }
}

// 核心过滤调度属性：满足全部、体验、各班级、黑名单的多路全动态不冲突过滤
const pagedUsers = computed(() => {
  let filtered = [...users.value]

  if (currentClass.value === 'blacklist') {
    filtered = filtered.filter(u => u.account_status === 2)
  } else if (currentClass.value === 'experience') {
    filtered = filtered.filter(u => u.class_name === '体验账户' || u.account_status === 1)
  } else {
    filtered = filtered.filter(u => u.account_status !== 2 && u.account_status !== 1 && u.class_name !== '体验账户')
    if (currentClass.value !== 'all') {
      filtered = filtered.filter(u => u.class_name === currentClass.value)
    }
  }

  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    filtered = filtered.filter(u =>
        String(u.real_name || u.nickname || '').toLowerCase().includes(q) ||
        String(u.student_id || '').includes(q) ||
        String(u.class_name || '').toLowerCase().includes(q)
    )
  }

  if (currentSort.value.prop && currentSort.value.order) {
    filtered.sort((a, b) => {
      const valA = (a as any)[currentSort.value.prop] || 0
      const valB = (b as any)[currentSort.value.prop] || 0
      return currentSort.value.order === 'ascending' ? valA - valB : valB - valA
    })
  }
  return filtered
})

const loadUsers = async () => {
  try {
    const res: any = await request.get('/admin/users/list')
    users.value = res.data || []
  } catch {
    ElMessage.error('无法同步全院花名册')
  }
}

const loadTracks = async () => {
  try {
    const res: any = await request.get('/auth/tracks')
    allAvailableTracks.value = res.data || []
  } catch {}
}

const loadStandardClasses = async () => {
  try {
    const res: any = await request.get('/auth/classes')
    standardClasses.value = res.data || []
  } catch {}
}

const unwrapResponseData = (res: any) => {
  if (Array.isArray(res)) return res
  if (Array.isArray(res?.data)) return res.data
  if (Array.isArray(res?.data?.data)) return res.data.data
  return []
}

const xmlEscape = (value: any) => {
  return String(value ?? '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&apos;')
}

const normalizeDate = (record: CheckinExportRecord) => {
  return String(record.checkinDate || record.checkin_date || '').slice(0, 10)
}

const normalizeUserId = (record: CheckinExportRecord) => Number(record.userId ?? record.user_id ?? 0)

const normalizeTrackId = (record: CheckinExportRecord) => Number(record.trackId ?? record.track_id ?? 0)

const buildExcelXml = (rows: any[][]) => {
  const worksheetRows = rows.map(row => {
    const cells = row.map(cell => `<Cell><Data ss:Type="String">${xmlEscape(cell)}</Data></Cell>`).join('')
    return `<Row>${cells}</Row>`
  }).join('')

  return `<?xml version="1.0" encoding="UTF-8"?>
<?mso-application progid="Excel.Sheet"?>
<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:o="urn:schemas-microsoft-com:office:office"
 xmlns:x="urn:schemas-microsoft-com:office:excel"
 xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"
 xmlns:html="http://www.w3.org/TR/REC-html40">
 <Worksheet ss:Name="人员打卡数据">
  <Table>${worksheetRows}</Table>
 </Worksheet>
</Workbook>`
}

const downloadExcelXml = (rows: any[][]) => {
  const blob = new Blob([buildExcelXml(rows)], { type: 'application/vnd.ms-excel;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  const now = new Date()
  const stamp = `${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}${String(now.getDate()).padStart(2, '0')}_${String(now.getHours()).padStart(2, '0')}${String(now.getMinutes()).padStart(2, '0')}`
  link.href = url
  link.download = `人员打卡数据_${stamp}.xls`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

const handleExportExcel = async () => {
  exportLoading.value = true
  try {
    if (!users.value.length) {
      await loadUsers()
    }

    const recordsRes: any = await request.get('/admin/audit/list')
    const records = unwrapResponseData(recordsRes) as CheckinExportRecord[]
    const allDates = [...new Set(records.map(normalizeDate).filter(Boolean))].sort()
    const userRecords = new Map<number, CheckinExportRecord[]>()

    records.forEach(record => {
      const uid = normalizeUserId(record)
      if (!uid) return
      if (!userRecords.has(uid)) userRecords.set(uid, [])
      userRecords.get(uid)!.push(record)
    })

    const header = [
      '序号',
      '姓名',
      '班级',
      '长学号',
      '参与赛道数量',
      '赋能分',
      '德育分',
      '单赛道打卡次数与日期明细',
      '用户每日打卡数据'
    ]

    const dataRows = users.value.map((user, index) => {
      const tracks = user.tracks || []
      const trackNames = new Map<number, string>()
      tracks.forEach(track => trackNames.set(Number(track.id), track.name))

      const recordsOfUser = userRecords.get(Number(user.id)) || []
      recordsOfUser.forEach(record => {
        const tid = normalizeTrackId(record)
        const name = record.trackName || record.track_name
        if (tid && name && !trackNames.has(tid)) trackNames.set(tid, name)
      })

      const checkedByTrack = new Map<number, Set<string>>()
      recordsOfUser
          .filter(record => Number(record.status) === 1)
          .forEach(record => {
            const tid = normalizeTrackId(record)
            const date = normalizeDate(record)
            if (!tid || !date) return
            if (!checkedByTrack.has(tid)) checkedByTrack.set(tid, new Set())
            checkedByTrack.get(tid)!.add(date)
          })

      const trackDetails = [...trackNames.entries()].map(([trackId, trackName]) => {
        const checkedDates = [...(checkedByTrack.get(trackId) || new Set<string>())].sort()
        const missingDates = allDates.filter(date => !checkedDates.includes(date))
        return `${trackName}：${checkedDates.length}次；已打卡[${checkedDates.join('、') || '无'}]；未打卡[${missingDates.join('、') || '无'}]`
      }).join('\n')

      const dailyDetails = allDates.map(date => {
        const checkedCount = [...trackNames.keys()].filter(trackId => checkedByTrack.get(trackId)?.has(date)).length
        return `${date}：${checkedCount}/${trackNames.size || 0}`
      }).join('\n')

      return [
        index + 1,
        user.real_name || user.nickname || '',
        user.class_name || '',
        user.student_id || '',
        tracks.length,
        user.empowerment_score ?? 0,
        user.moral_score ?? 0,
        trackDetails || '无赛道数据',
        dailyDetails || '无打卡日期数据'
      ]
    })

    downloadExcelXml([header, ...dataRows])
    ElMessage.success(`已导出 ${dataRows.length} 名人员数据`)
  } catch (error) {
    console.error('导出人员打卡数据失败:', error)
    ElMessage.error('导出失败，请确认本地后端和数据库已启动')
  } finally {
    exportLoading.value = false
  }
}

const prepareTrackManage = (user: User) => {
  currUser.value = user
  selectedTrackIds.value = (user.tracks || []).map(t => t.id)
  if (!allAvailableTracks.value.length) loadTracks()
  dlgTrack.value = true
}

const doSaveUserTracks = async () => {
  if (!currUser.value) return
  try {
    await request.post('/admin/user/tracks/reset', { userId: currUser.value.id, trackIds: selectedTrackIds.value })
    ElMessage.success("赛道权限重组成功")
    dlgTrack.value = false
    loadUsers()
  } catch (e: any) {
    ElMessage.error(e.message || "变更失败")
  }
}

// ---------------- 🎨 2X3 大管理员控制矩阵事件 ----------------

// 1. 修改
const handleMatrixModify = () => {
  const target = currUser.value || users.value.find(u => u.id === selectedUserIds.value[0])
  if (!target) return
  editingUser.value = { ...target }
  if (!standardClasses.value.length) loadStandardClasses()
  dlgUser.value = true
}

const doSaveUserChanges = async () => {
  try {
    await request.put('/admin/user/update-archive', editingUser.value)
    ElMessage.success("底册学籍资料特权更正完毕")
    dlgUser.value = false
    loadUsers()
  } catch {
    ElMessage.error("订正失败，请检查参数")
  }
}

const handleResetPassword = () => {
  const targetId = editingUser.value.id
  if (!targetId) return
  ElMessageBox.confirm(
      `确定要对该账户进行【一键重设密码为 123456】且强制该生的全部微信设备强驱退登下线吗？`,
      '高危物理重置凭证安全拦截',
      { confirmButtonText: '确定重置并强退', type: 'warning' }
  ).then(async () => {
    try {
      const res: any = await request.post('/admin/user/reset-password', { userId: targetId })
      ElMessage.success(res.message || "密码重设指令已全网网关强制广播！")
      dlgUser.value = false
      loadUsers()
    } catch {}
  })
}

// 2. 奖励
const handleMatrixReward = () => {
  const target = currUser.value || users.value.find(u => u.id === selectedUserIds.value[0])
  if (!target) return
  rewardForm.value = { userId: target.id, type: 'emp', score: 10, reason: '' }
  
  if (selectedUserIds.value.length === 0) {
    // 单人模式下加载历史
    loadScoreLogs(target.id)
  }
  dlgReward.value = true
}

const loadScoreLogs = async (userId: number) => {
  try {
    const res: any = await request.get(`/admin/reward/history?userId=${userId}`)
    scoreLogs.value = res.data || []
  } catch {
    scoreLogs.value = []
  }
}

const filteredRewardHistory = computed(() => {
  return scoreLogs.value.filter(log => log.action === 'ADD')
})

const doSubmitReward = async () => {
  if (!rewardForm.value.reason) return ElMessage.warning("奖励事由公示不能为空")
  try {
    if (selectedUserIds.value.length > 0) {
      // 批量发送奖励 Batch Pipeline
      await Promise.all(selectedUserIds.value.map(id => 
        request.post('/admin/reward/send', { ...rewardForm.value, userId: id })
      ))
      ElMessage.success(`🎉 批量奖励发放成功，共 ${selectedUserIds.value.length} 人！`)
    } else {
      await request.post('/admin/reward/send', rewardForm.value)
      ElMessage.success("量化嘉奖分值已实时注入该生档案")
    }
    dlgReward.value = false
    clearActiveSelection()
    loadUsers()
  } catch (e: any) {
    ElMessage.error(e.message || "发放失败")
  }
}

const doRevokeReward = (logId: number) => {
  ElMessageBox.confirm(`撤回加分将从该生的总分/卡数中反向物理扣减，确定撤回吗？`, '物理撤回警告', { type: 'error' }).then(async () => {
    try {
      await request.post('/admin/reward/revoke', { logId })
      ElMessage.success("该加分荣誉记录已物理抹除，数值已反向扣除！")
      if (currUser.value) loadScoreLogs(currUser.value.id)
      loadUsers()
    } catch {}
  })
}

// 3. 处罚
const handleMatrixPunish = () => {
  const target = currUser.value || users.value.find(u => u.id === selectedUserIds.value[0])
  if (!target) return
  punishForm.value = { userId: target.id, type: 'emp', score: 10, duration: '5m', reason: '' }
  if (selectedUserIds.value.length === 0) {
    loadScoreLogs(target.id)
  }
  dlgPunish.value = true
}

const filteredPunishHistory = computed(() => {
  return scoreLogs.value.filter(log => log.action === 'SUB' || log.score_type === 'ban')
})

const doSubmitPunish = async () => {
  if (!punishForm.value.reason) return ElMessage.warning("惩戒原因必须填写作为存证")
  try {
    if (selectedUserIds.value.length > 0) {
      if (punishForm.value.type === 'ban') {
        // 批量临时封禁
        await request.post('/admin/user/ban-batch', { userIds: selectedUserIds.value })
        ElMessage.success(`🔒 批量临时熔断封禁已向 ${selectedUserIds.value.length} 人下发！`)
      } else {
        // 批量扣分
        await Promise.all(selectedUserIds.value.map(id =>
          request.post('/admin/punish/execute', { ...punishForm.value, userId: id })
        ))
        ElMessage.success(`📉 批量惩戒扣分成功，共 ${selectedUserIds.value.length} 人！`)
      }
    } else {
      await request.post('/admin/punish/execute', punishForm.value)
      ElMessage.success("纪律干预动作已被强制执行并记入底册！")
    }
    dlgPunish.value = false
    clearActiveSelection()
    loadUsers()
  } catch (e: any) {
    ElMessage.error(e.message || "惩戒执行被拒绝")
  }
}

const doRevokePunish = (logId: number) => {
  ElMessageBox.confirm(`撤销惩罚将把之前扣除的积分自动在 user 表中安全加回，并解锁可能存在的封锁状态，确定撤销吗？`, '撤销惩罚', { type: 'success' }).then(async () => {
    try {
      await request.post(`/admin/punish/revoke?logId=${logId}`)
      ElMessage.success("惩戒已撤销，积分及对应状态已全额复原！")
      if (currUser.value) loadScoreLogs(currUser.value.id)
      loadUsers()
    } catch {}
  })
}

// 4. 下线
const handleMatrixKick = () => {
  if (selectedUserIds.value.length > 0) {
    ElMessageBox.confirm(`确定要强制斩断选中的这 ${selectedUserIds.value.length} 名学生的所有在线连接吗？`, '批量作退下线', { type: 'warning' }).then(async () => {
      await request.post('/admin/user/kick-batch', { userIds: selectedUserIds.value })
      ElMessage.success("批量踢人网关指令抛出成功！")
      clearActiveSelection()
      loadUsers()
    })
  } else if (currUser.value) {
    ElMessageBox.confirm(`强制斩断【${currUser.value.real_name || currUser.value.nickname}】与系统的在线连接吗？`, '会话下线拦截', { type: 'warning' }).then(async () => {
      await request.post('/admin/user/kick', { userId: currUser.value!.id })
      ElMessage.success("断线踢人网关拦截信号抛出成功！")
      clearActiveSelection()
      loadUsers()
    })
  }
}

// 5. 拉黑 (封锁物理硬件，单人模式下操作)
const handleMatrixBlacklist = () => {
  if (!currUser.value) return
  const isBanned = currUser.value.account_status === 2
  const targetStatus = isBanned ? 0 : 2
  const actionText = isBanned ? '赦免解封并释放物理设备' : '设备级风控拉黑(封人先封机，永久阻断硬件)'

  ElMessageBox.confirm(`确定对该用户及当前绑定的物理手机设备指纹进行【${actionText}】吗？`, '设备风控硬件拉黑', { type: 'error' }).then(async () => {
    try {
      await request.post('/admin/user/blacklist', { userId: currUser.value!.id, status: targetStatus })
      ElMessage.success(`物理设备指纹拉黑指令已 ${isBanned ? '赦免解封' : '熔断永久锁死'} 成功！`)
      clearActiveSelection()
      loadUsers()
    } catch (e: any) {
      ElMessage.error(e.message || "设备拉黑执行失败")
    }
  })
}

// 6. 删除 (连根物理抹除六表)
const handleMatrixDelete = () => {
  const target = currUser.value || users.value.find(u => u.id === selectedUserIds.value[0])
  if (!target) return
  ElMessageBox.confirm(
      `警告：确定要一键抹除【${target.real_name || target.nickname}】在数据库所有六张表关联痕迹吗？此高危物理删除绝对不可恢复！`,
      '高危物理物理注销',
      { confirmButtonText: '确定连根物理抹除', type: 'error' }
  ).then(async () => {
    try {
      await request.delete('/admin/user/delete', { data: { userId: target.id } })
      ElMessage.success("该生全部档案及所有流水已被连根拔起彻底蒸发！")
      clearActiveSelection()
      loadUsers()
    } catch {}
  })
}

// ---------------- 👥 全局批量指挥防线 ----------------

// 批量踢人 (或一键全服踢人)
const handleBatchKick = () => {
  if (selectedUserIds.value.length > 0) {
    handleMatrixKick()
  } else {
    ElMessageBox.confirm(
        '⚠️ 全局应急下线：强制全网踢出全服所有在线的非管理员打卡人员？',
        '全局强制离线强退',
        { confirmButtonText: '确定强退全网', type: 'error' }
    ).then(async () => {
      await request.post('/admin/global/kick')
      ElMessage.warning('全网广播一键应急强退已瞬间送达下发！')
      loadUsers()
    })
  }
}

// 批量封禁 (或一键全服熔断)
const handleBatchBan = () => {
  if (selectedUserIds.value.length > 0) {
    handleMatrixPunish()
  } else {
    const target = !isGlobalBanned.value
    ElMessageBox.confirm(
        `紧急防线熔断：确定要对除管理员外的全服所有在线业务进行一键应急熔断封锁 5 分钟吗？`,
        '系统全局应急熔断保护',
        { confirmButtonText: '确定应急封禁', type: 'error' }
    ).then(async () => {
      await request.post('/admin/global/ban', { status: target })
      isGlobalBanned.value = target
      ElMessage.success(target ? '全局打卡网络已被系统休克隔离保护！' : '熔断已恢复')
      loadUsers()
    })
  }
}

// ---------------- 📂 本地头像直传上传 ----------------

const beforeAvatarUpload = (file: any) => {
  const isJPGorPNG = file.type === 'image/jpeg' || file.type === 'image/png' || file.type === 'image/webp'
  const isLt2M = file.size / 1024 / 1024 < 2

  if (!isJPGorPNG) {
    ElMessage.error('图片只能是 JPG/PNG/WEBP 格式!')
  }
  if (!isLt2M) {
    ElMessage.error('图片大小不能超过 2MB!')
  }
  return isJPGorPNG && isLt2M
}

const handleAvatarSuccess = async (res: any, row: User) => {
  if (res && res.imageUrl) {
    try {
      // 成功上传本地相对路径后，调用 updateAvatar 接口实时同步
      await request.post('/user/updateAvatar', { userId: row.id, avatarUrl: res.imageUrl })
      ElMessage.success("本地头像直传上传并实时对齐同步成功！")
      loadUsers()
    } catch {
      ElMessage.error("头像状态写入失败")
    }
  } else {
    ElMessage.error("头像文件上传保存失败")
  }
}

const handleAvatarSuccessWrapper = (row: User) => {
  return (res: any) => handleAvatarSuccess(res, row)
}

onMounted(() => {
  loadUsers()
  fetchSystemLimits()
})
</script>

<style scoped>
/* 强制消除Element自带的细微边距抖动，锁死全屏绝对不溢出滑动条 */
:deep(.el-table__body-wrapper) {
  will-change: transform;
}
</style>
