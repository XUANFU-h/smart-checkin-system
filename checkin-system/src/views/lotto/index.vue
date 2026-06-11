<template>
  <div class="p-6 max-w-[1600px] mx-auto bg-slate-50 min-h-screen">
    <!-- Header Controls -->
    <div class="flex flex-col xl:flex-row xl:items-center xl:justify-between pb-6 border-b border-slate-200 mb-6 gap-4">
      <div>
        <h1 class="text-2xl font-bold text-slate-900">大乐透开奖控制中心</h1>
        <p class="text-sm text-slate-500 mt-1 flex items-center gap-2">
          <el-tag :type="activeStatus === 'open' ? 'success' : 'danger'" size="small" effect="dark" class="font-bold">
            {{ activeStatus === 'open' ? '🟢 参与中 (可刷新/锁定)' : '🔴 已结束 (等待管理员开启下一期)' }}
          </el-tag>
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-4">
        <!-- 🪙 赋能分活动总奖池 (Empowerment Prize Pool) -->
        <div 
          class="bg-gradient-to-br from-amber-500 via-amber-600 to-yellow-500 text-white shadow-md hover:shadow-xl hover:shadow-amber-500/20 transition-all duration-300 transform hover:-translate-y-0.5 cursor-pointer border border-amber-400/30 rounded-xl px-5 py-2.5 text-center relative overflow-hidden min-w-[160px]"
          @click="openPoolLedger('empowerment')"
        >
          <div class="text-xs text-amber-100 font-bold tracking-wider flex items-center justify-center gap-1">
            <span>🪙 赋能分活动总奖池</span>
            <el-icon class="animate-pulse"><InfoFilled /></el-icon>
          </div>
          <div class="text-2xl font-black mt-1 drop-shadow-md">
            {{ totalPoolScore }} <span class="text-xs font-normal opacity-90">分</span>
          </div>
          <div class="absolute right-[-10px] bottom-[-10px] opacity-10 text-5xl font-bold font-sans select-none">EMP</div>
        </div>

        <!-- 🏆 德育分活动总奖池 (Moral Prize Pool) -->
        <div 
          class="bg-gradient-to-br from-violet-600 via-indigo-600 to-purple-500 text-white shadow-md hover:shadow-xl hover:shadow-indigo-500/20 transition-all duration-300 transform hover:-translate-y-0.5 cursor-pointer border border-indigo-400/30 rounded-xl px-5 py-2.5 text-center relative overflow-hidden min-w-[160px]"
          @click="openPoolLedger('moral')"
        >
          <div class="text-xs text-indigo-100 font-bold tracking-wider flex items-center justify-center gap-1">
            <span>🏆 德育分活动总奖池</span>
            <el-icon class="animate-pulse"><InfoFilled /></el-icon>
          </div>
          <div class="text-2xl font-black mt-1 drop-shadow-md">
            {{ totalMoralPoolScore }} <span class="text-xs font-normal opacity-90">分</span>
          </div>
          <div class="absolute right-[-10px] bottom-[-10px] opacity-10 text-5xl font-bold font-sans select-none">MOR</div>
        </div>

        <div class="bg-emerald-50 border border-emerald-200 rounded-xl px-4 py-2.5 text-center shadow-sm min-w-[120px]">
          <div class="text-xs text-emerald-600 font-bold">本期已锁定人数</div>
          <div class="text-2xl font-black text-emerald-700 mt-0.5">{{ entryList.length }} <span class="text-xs font-normal">人</span></div>
        </div>

        <!-- 🪙 付费参与消耗积分设定 -->
        <div class="bg-white border border-slate-200 rounded-xl p-2 flex items-center gap-2 shadow-sm">
          <span class="text-xs text-slate-500 font-bold">🪙 参与积分消耗</span>
          <el-input-number v-model="spinCostConfig" :min="0" :max="5000" size="small" class="!w-24" />
          <el-button type="success" size="small" plain @click="saveSpinCostConfig">保存设定</el-button>
        </div>

        <div class="bg-white border border-slate-200 rounded-xl p-2 flex items-center gap-2 shadow-sm">
          <span class="text-xs font-bold text-slate-500">自动开奖时间</span>
          <el-input
              v-model="configForm.drawTime"
              placeholder="每周六 10:00"
              size="default"
              class="!w-44"
          />
          <el-button type="primary" plain @click="saveTimeConfig">保存时间</el-button>
        </div>

        <!-- 🎮 创意工坊开启/锁定开关 -->
        <div class="bg-white border border-slate-200 rounded-xl p-2 h-10 flex items-center gap-3 shadow-sm px-4">
          <span class="text-xs font-bold text-slate-600">🎮 创意工坊通道</span>
          <el-switch
              v-model="workshopEnabled"
              active-text="开放中"
              inactive-text="已锁定"
              inline-prompt
              active-color="#10B981"
              inactive-color="#EF4444"
              @change="handleToggleWorkshop"
          />
        </div>

        <el-button 
          v-if="activeStatus !== 'open'"
          type="success" 
          class="font-bold h-11 px-6 shadow-md shadow-emerald-100" 
          @click="handleStartNextPeriod"
        >
          🛠️ 开启第 {{ currentWeekNum + 1 }} 期活动
        </el-button>

        <el-button 
          v-if="activeStatus === 'open'"
          type="danger" 
          class="font-bold h-11 px-6 shadow-md shadow-red-100" 
          @click="handleManualDraw"
        >
          🚀 现场全服开奖
        </el-button>
      </div>
    </div>

    <!-- Collapsible 往期中奖历史红榜归档 -->
    <el-collapse class="mb-6 rounded-xl border border-slate-200 overflow-hidden shadow-sm">
      <el-collapse-item name="history">
        <template #title>
          <div class="px-4 font-bold text-slate-800 flex items-center gap-2 text-sm">
            <span>📅 往期大乐透开奖红榜归档 (点击展开历史记录)</span>
            <el-tag size="small" type="success">{{ historyList.length }} 条记录</el-tag>
          </div>
        </template>
        <div class="px-4 pb-4">
          <el-table :data="historyList" height="280px" stripe class="w-full">

            <el-table-column prop="userName" label="中奖同学" width="120" align="center">
              <template #default="{row}">
                <span class="font-bold text-slate-800">{{ row.userName }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="className" label="所属班级" width="160" show-overflow-tooltip />
            <el-table-column prop="numbers" label="🔒 锁定的中奖号码" width="160" align="center">
              <template #default="{row}">
                <span class="bg-slate-100 text-slate-700 px-2 py-1 rounded font-mono font-bold tracking-wider text-xs">
                  {{ row.numbers }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="prizeInfo" label="获得的奖品与奖励额度" min-width="250" show-overflow-tooltip>
              <template #default="{row}">
                <span class="font-bold text-amber-600">{{ row.prizeInfo }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-collapse-item>
    </el-collapse>

    <!-- Main Workspace Layout -->
    <div class="flex flex-col lg:flex-row gap-6 items-start">
      <!-- Left Table List -->
      <div class="w-full lg:w-[65%] bg-white rounded-xl border border-slate-200 p-4 shadow-sm h-[calc(100vh-220px)] flex flex-col">
        <div class="flex justify-between items-center mb-4">
          <div class="flex items-center gap-2">
            <span class="font-bold text-slate-800">本期参玩学生序列</span>
            <el-tag size="small" type="info">已选中 {{ selectedUserIds.length }} 人</el-tag>
          </div>
          <div class="flex gap-2">
            <!-- 🧹 一键清空参玩列表 -->
            <el-button 
              type="warning" 
              size="small" 
              class="font-bold" 
              @click="handleClearAllParticipants"
            >
              🧹 一键清空参玩列表
            </el-button>
            <!-- 批量移除驳回退款按钮 -->
            <el-button 
              type="danger" 
              size="small" 
              class="font-bold" 
              :disabled="selectedUserIds.length === 0" 
              @click="handleBatchReject"
            >
              🔥 批量驳回退款 (已选 {{ selectedUserIds.length }} 人)
            </el-button>
            <el-input v-model="searchQuery" placeholder="搜姓名/学号..." size="small" class="!w-48" clearable />
          </div>
        </div>

        <el-table
            :data="filteredEntries"
            height="100%"
            @selection-change="handleSelectionChange"
            class="w-full text-slate-700"
        >
          <el-table-column type="selection" width="50" align="center" />
          <el-table-column type="index" label="序号" width="60" align="center" />
          <el-table-column prop="realName" label="学生姓名" width="100">
            <template #default="{row}">
              <span class="font-bold text-slate-800">{{ row.realName || row.nickname }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="studentId" label="学号" width="130" />
          <el-table-column prop="className" label="所属班级" width="120" show-overflow-tooltip />
          <el-table-column prop="lottoNumber" label="🔒 锁定的老虎机号" min-width="130" align="center">
            <template #default="{row}">
              <span class="bg-slate-100 text-slate-700 px-2 py-1 rounded font-mono font-bold tracking-wider text-sm">
                {{ row.lottoNumber || '00-00-00' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="参与方式" width="160" align="center">
            <template #default="{row}">
              <el-tag :type="row.isStreak ? 'success' : 'warning'" size="default" class="font-bold">
                {{ row.isStreak ? '通过打卡之后参与' : '通过消耗积分参与' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="trackCount" label="参加赛道数量" width="110" align="center">
            <template #default="{row}">
              <el-tag size="small" type="info">{{ row.trackCount || 0 }} 条</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="totalCheckinCount" label="累计打卡次数" width="110" align="center">
            <template #default="{row}">
              <el-tag size="small" type="success" effect="plain">{{ row.totalCheckinCount || 0 }} 次</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="center" fixed="right">
            <template #default="{row}">
              <el-button 
                size="small" 
                type="danger" 
                link
                class="font-bold hover:text-red-700 transition-colors"
                @click="rejectParticipant(row)"
              >
                驳回
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Right Prize configurations Sidebar -->
      <div class="w-full lg:w-[35%] space-y-6 shrink-0">
        <!-- 各级奖项中奖名额与分值类型配置 -->
        <div class="bg-white rounded-xl border border-slate-200 p-5 shadow-sm">
          <div class="flex justify-between items-center mb-4">
            <h3 class="font-bold text-slate-800 flex items-center gap-1.5">
              ⚙️ 各级奖项中奖名额与价值配准
            </h3>
            <el-button type="success" size="small" class="font-bold" @click="savePrizeConfigs">
              💾 保存配置
            </el-button>
          </div>
          <div class="space-y-4 max-h-[520px] overflow-y-auto pr-1">
            <div v-for="prize in prizeConfigs" :key="prize.level" class="p-3.5 bg-slate-50 border border-slate-200 rounded-xl transition-all duration-200 hover:border-slate-300">
              <div class="flex justify-between items-center mb-2.5">
                <span class="text-sm font-bold" :class="prize.color">{{ prize.label }}</span>
                <span class="text-xs text-slate-400" v-if="prize.level !== 'level4'">名额: {{ prize.maxWinners }}人</span>
                <span class="text-xs text-indigo-600 font-bold" v-else>普照奖人人有份</span>
              </div>
              
              <div class="space-y-2 text-xs">
                <!-- 一二三等奖可调整名额和类型 -->
                <block v-if="prize.level !== 'level4'">
                  <div class="grid grid-cols-2 gap-2">
                    <div>
                      <label class="text-[10px] text-slate-400 block mb-1">规划中奖名额(人)</label>
                      <el-input-number v-model="prize.maxWinners" :min="0" :max="500" size="small" class="!w-full" />
                    </div>
                    <div>
                      <label class="text-[10px] text-slate-400 block mb-1">奖励发放类型</label>
                      <el-select v-model="prize.rewardType" size="small" class="w-full">
                        <el-option label="仅发放赋能分" value="empowerment" />
                        <el-option label="仅发放德育分" value="moral" />
                        <el-option label="双向混合赠送" value="mixed" />
                      </el-select>
                    </div>
                  </div>

                  <div v-if="prize.rewardType === 'empowerment'" class="space-y-1">
                    <label class="text-[10px] text-slate-400 block">单人发放赋能分分值</label>
                    <el-input-number v-model="prize.empPower" :min="0" :max="10000" size="small" class="!w-full" />
                  </div>
                  <div v-else-if="prize.rewardType === 'moral'" class="space-y-1">
                    <label class="text-[10px] text-slate-400 block">单人发放德育分分值</label>
                    <el-input-number v-model="prize.moralPower" :min="0" :max="10000" size="small" class="!w-full" />
                  </div>
                  <div v-else-if="prize.rewardType === 'mixed'" class="grid grid-cols-2 gap-2">
                    <div>
                      <label class="text-[10px] text-slate-400 block mb-1">赋能分值</label>
                      <el-input-number v-model="prize.empPower" :min="0" :max="10000" size="small" class="!w-full" />
                    </div>
                    <div>
                      <label class="text-[10px] text-slate-400 block mb-1">德育分值</label>
                      <el-input-number v-model="prize.moralPower" :min="0" :max="10000" size="small" class="!w-full" />
                    </div>
                  </div>
                </block>

                <!-- 参与奖人人有份提示 -->
                <block v-else>
                  <div class="bg-indigo-50 border border-indigo-100 rounded-lg p-2.5 text-indigo-700 text-xs mt-1">
                    💡 参与奖（阳光普照奖）固定为 <strong>50 赋能分</strong>（已在系统后端写死安全逻辑）。参与但未获得一二三等奖的所有人将在开奖结算时全员物理派发！无需在此处单独配置或现场摇号抽选。
                  </div>
                </block>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 💰 奖池明细账本与管理干预 Dialog -->
    <el-dialog
      v-model="showPoolLedgerDlg"
      :title="activePoolType === 'empowerment' ? '🪙 赋能分活动奖池收支账本与注入面板' : '🏆 德育分活动奖池收支账本与注入面板'"
      width="750px"
      destroy-on-close
      class="rounded-xl overflow-hidden premium-dialog"
    >
      <el-tabs type="border-card" class="rounded-lg shadow-sm">
        <el-tab-pane label="📊 奖池收支明细账本">
          <div v-loading="loadingLedger" class="min-h-[300px]">
            <el-table :data="ledgerLogs" height="350px" stripe class="w-full">
              <el-table-column prop="id" label="ID" width="70" align="center" />
              <el-table-column label="发生时间" width="165" align="center">
                <template #default="{row}">
                  <span class="font-mono text-slate-500 text-xs">{{ formatTime(row.create_time) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="变动分值" width="120" align="center">
                <template #default="{row}">
                  <span :class="row.amount >= 0 ? 'text-emerald-600 font-bold' : 'text-rose-600 font-bold'">
                    {{ row.amount >= 0 ? '+' : '' }}{{ row.amount }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column label="类型" width="110" align="center">
                <template #default="{row}">
                  <el-tag :type="getLogTagType(row.log_type)" size="small">
                    {{ getLogTypeName(row.log_type) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="description" label="备注说明" min-width="180" show-overflow-tooltip />
            </el-table>
          </div>
        </el-tab-pane>
        <el-tab-pane label="⚡ 注入与干预控制台">
          <div class="p-4 space-y-4">
            <div class="bg-slate-50 border border-slate-200 rounded-lg p-4 text-sm text-slate-600">
              <p class="font-bold text-slate-800 mb-1">💡 双向奖池干预策略说明：</p>
              <p>1. 您可以手动向奖池中<strong>注入分值</strong>（正数）或者从奖池中<strong>划扣扣除</strong>（负数）。</p>
              <p>2. 所有变动都会生成不可篡改的原子变动流水账本，供班级或全校监督审计。</p>
            </div>
            <el-form :model="injectForm" label-position="top">
              <el-form-item label="干预发生分值（正数为注入，负数为抽水扣除）" required>
                <el-input-number v-model="injectForm.amount" :min="-100000" :max="100000" class="!w-full" />
              </el-form-item>
              <el-form-item label="审计原由/备注说明" required>
                <el-input 
                  v-model="injectForm.description" 
                  type="textarea" 
                  :rows="3" 
                  placeholder="例如：大乐透首发官方保底注入，或者：第202601期德育保底分" 
                />
              </el-form-item>
              <el-form-item class="!mb-0">
                <el-button 
                  type="primary" 
                  class="w-full font-bold h-10 tracking-wider" 
                  @click="submitPoolInjection"
                >
                  🚀 确认执行干预并写入账本
                </el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <!-- 🎰 现场开奖 Draft 草拟名单预览与刷新大弹窗 -->
    <el-dialog
      v-model="showDraftDrawDlg"
      title="🎉 现场大乐透开奖令草拟名单预览"
      width="780px"
      destroy-on-close
      class="rounded-xl overflow-hidden premium-dialog"
    >
      <div v-loading="loadingDraft" class="p-4 space-y-4">
        <!-- 头部说明及刷新键 -->
        <div class="flex justify-between items-center bg-indigo-50 border border-indigo-100 rounded-xl p-4 text-indigo-900 text-sm">
          <div class="flex-1 pr-6">
            <p class="font-bold flex items-center gap-1"><el-icon><InfoFilled /></el-icon> 🎲 加权内幕抽签已完成：</p>
            <p class="text-xs text-indigo-700 mt-1">此名单是由后端<b>隐藏权重算法</b>自动筛选草拟。通过打卡方式参与的，且<b>打卡次数高、参与赛道多</b>的学生权重更高、中大奖概率呈数倍级增幅！参与奖（阳光普照固定 50 赋能分）在此名单之外自动核发。</p>
          </div>
          <el-button type="success" class="font-bold" size="default" @click="fetchDraftDrawList">
            🔄 重新摇号/刷新名单
          </el-button>
        </div>

        <!-- 拟定赢家列表 -->
        <el-table :data="draftWinners" height="360px" stripe class="w-full text-slate-800">
          <el-table-column label="拟中奖档" width="130" align="center">
            <template #default="{row}">
              <span class="font-bold" :class="getPrizeLevelColor(row.prizeLevel)">{{ row.prizeLabel }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="中奖同学姓名" width="130" align="center">
            <template #default="{row}">
              <span class="font-bold">{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="className" label="所属班级" min-width="180" show-overflow-tooltip />
          <el-table-column label="规划分值" width="220" align="center">
            <template #default="{row}">
              <el-tag type="info" size="small" class="font-bold">
                {{ getPrizeConfigSummary(row.prizeLevel) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>

        <!-- 结算按钮 -->
        <div class="flex justify-end gap-3 pt-4 border-t border-slate-200 mt-2">
          <el-button @click="showDraftDrawDlg = false">暂不派奖</el-button>
          <el-button type="danger" class="font-bold h-10 px-6 shadow-md shadow-red-100" @click="confirmDraftAndDrawFinal">
            🚀 确认名单并现场派发奖励
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import request from '@/api/request.ts'

// 强类型声明接口
interface LottoEntry {
  id: number;
  userId: number;
  realName: string;
  nickname: string;
  studentId: string;
  className: string;
  lottoNumber: string;
  isStreak: boolean;
  trackCount: number;
  totalCheckinCount: number;
}
interface PrizeConfig {
  level: string;
  label: string;
  rewardType: 'empowerment' | 'moral' | 'mixed';
  empPower: number;
  moralPower: number;
  maxWinners: number;
  drawnWinners: number[];
  color: string;
}

const currentWeekNum = ref(0)
const activeStatus = ref('closed')
const workshopEnabled = ref(true) // 🎮 创意工坊开启状态
const totalPoolScore = ref(0)
const totalMoralPoolScore = ref(0)
const searchQuery = ref('')
const entryList = ref<LottoEntry[]>([])
const selectedUserIds = ref<number[]>([])

const configForm = ref({ drawTime: '' })
const spinCostConfig = ref(100) // 后台随时配置

// 双向奖池与干预状态
const showPoolLedgerDlg = ref(false)
const activePoolType = ref<'empowerment' | 'moral'>('empowerment')
const ledgerLogs = ref<any[]>([])
const loadingLedger = ref(false)
const injectForm = ref({
  amount: 0,
  description: ''
})

// 🎲 现场开奖 Draft 弹出与刷新名单相关状态
const showDraftDrawDlg = ref(false)
const draftWinners = ref<any[]>([])
const loadingDraft = ref(false)

// 往期历史红榜
const historyList = ref<any[]>([])

// 默认奖项等级与名额调控字典
const prizeConfigs = ref<PrizeConfig[]>([
  { level: 'level1', label: '🥇 一等奖', rewardType: 'empowerment', empPower: 500, moralPower: 0, maxWinners: 2, drawnWinners: [], color: 'text-red-500' },
  { level: 'level2', label: '🥈 二等奖', rewardType: 'empowerment', empPower: 200, moralPower: 0, maxWinners: 5, drawnWinners: [], color: 'text-orange-500' },
  { level: 'level3', label: '🥉 三等奖', rewardType: 'empowerment', empPower: 50, moralPower: 0, maxWinners: 15, drawnWinners: [], color: 'text-amber-500' },
  { level: 'level4', label: '🎁 参与奖', rewardType: 'empowerment', empPower: 50, moralPower: 0, maxWinners: 9999, drawnWinners: [], color: 'text-slate-500' }
])

const handleSelectionChange = (val: LottoEntry[]) => {
  selectedUserIds.value = val.map(item => item.userId)
}

// 基于前端内存的高阶名单检索器
const filteredEntries = computed(() => {
  let res = [...entryList.value]
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    res = res.filter(item =>
        String(item.realName || item.nickname).toLowerCase().includes(q) ||
        String(item.studentId).includes(q)
    )
  }
  return res
})

// 🚀 加载控制中心面板
const loadLottoControlPanel = async () => {
  try {
    const res: any = await request.get('/game/slot/status')
    if (res && res.data) {
      configForm.value.drawTime = res.data.drawTimeStr || ''
      currentWeekNum.value = res.data.weekNum || 0
      totalPoolScore.value = res.data.totalPool || 0
      totalMoralPoolScore.value = res.data.totalMoralPool || 0
      activeStatus.value = res.data.activeStatus || 'closed' // 💡 获取并更新活动状态
      workshopEnabled.value = res.data.workshopActiveStatus !== 'closed' // 💡 创意工坊开启状态
      
      // 加载自适应配置
      const configRes: any = await request.get('/admin/game/lotto/prize-configs')
      if (configRes && configRes.data) {
        const parsed = typeof configRes.data === 'string' ? JSON.parse(configRes.data) : configRes.data
        if (Array.isArray(parsed)) {
          prizeConfigs.value = parsed.map(p => {
            const existing = prizeConfigs.value.find(old => old.level === p.level)
            return {
              ...p,
              drawnWinners: existing ? existing.drawnWinners : []
            }
          })
        }
      }

      // 获取动态积分扣减额度设定
      const costRes: any = await request.get('/admin/game/lotto/spin-cost')
      if (costRes && costRes.data !== undefined) {
        spinCostConfig.value = Number(costRes.data)
      }

      // 获取当前参与锁定名录
      const participantsRes: any = await request.get('/admin/game/lotto/participants')
      if (participantsRes && participantsRes.data) {
        entryList.value = participantsRes.data.map((r: any) => ({
          id: r.id,
          userId: r.userId,
          realName: r.realName || r.nickname || '未知同学',
          nickname: r.nickname || '未知',
          studentId: r.studentId || '',
          className: r.className || '普通班级',
          lottoNumber: r.lottoNumber || '00-00-00',
          isStreak: r.isStreak || false,
          trackCount: r.trackCount || 0,
          totalCheckinCount: r.totalCheckinCount || 0
        }))
      } else {
        entryList.value = []
      }

      // 获取往期中奖历史红榜记录
      const historyRes: any = await request.get('/game/lotto/history-records')
      if (historyRes && historyRes.data) {
        historyList.value = historyRes.data
      }
    }
  } catch {
    ElMessage.error('控制中枢核心数据载入发生未知异常')
  }
}

// 手动开启下一期大乐透活动
const handleStartNextPeriod = () => {
  ElMessageBox.confirm(
    `确认要手动开启第 ${currentWeekNum.value + 1} 期大乐透活动吗？开启后，微信小程序端玩法将重新对全体学生开放，大家可以参与刷新与锁定号码！`,
    '手动开启下一期活动',
    {
      confirmButtonText: '立即开启',
      cancelButtonText: '取消',
      type: 'success'
    }
  ).then(async () => {
    try {
      await request.post('/admin/game/lotto/start-next-period')
      ElMessage.success(`🎉 第 ${currentWeekNum.value + 1} 期大乐透活动已成功开启！`)
      await loadLottoControlPanel()
    } catch {}
  })
}

// 保存自动开奖时间
const saveTimeConfig = async () => {
  try {
    await request.post('/admin/game/lotto-setting', { drawTimeStr: configForm.value.drawTime })
    ElMessage.success('系统时钟自动开奖拦截策略已更新生效')
  } catch {}
}

// 🪙 保存动态付费积分扣减设定
const saveSpinCostConfig = async () => {
  try {
    await request.post('/admin/game/lotto/spin-cost', { cost: spinCostConfig.value })
    ElMessage.success(`学生参与消耗赋能分额度已成功调整为 ${spinCostConfig.value} 分！`)
    loadLottoControlPanel()
  } catch {}
}

// 💰 打开指定奖池明细账本
const openPoolLedger = async (type: 'empowerment' | 'moral') => {
  activePoolType.value = type
  showPoolLedgerDlg.value = true
  injectForm.value.amount = 0
  injectForm.value.description = ''
  await loadPoolLogs()
}

// 加载账本收支明细
const loadPoolLogs = async () => {
  loadingLedger.value = true
  try {
    const res: any = await request.get('/admin/game/lotto/pool-logs', {
      params: {
        weekNum: currentWeekNum.value,
        poolType: activePoolType.value
      }
    })
    if (res && res.data) {
      ledgerLogs.value = res.data
    }
  } catch {
    ElMessage.error('收支变动明细账本加载失败')
  } finally {
    loadingLedger.value = false
  }
}

// 管理员向奖池注入/扣减积分
const submitPoolInjection = async () => {
  if (!injectForm.value.amount) {
    return ElMessage.warning('请输入变动积分数值（正数为注入，负数为抽水）')
  }
  if (!injectForm.value.description.trim()) {
    return ElMessage.warning('请输入这次干预支出的原由与备注说明')
  }
  try {
    await request.post('/admin/game/lotto/pool-inject', {
      weekNum: currentWeekNum.value,
      poolType: activePoolType.value,
      amount: injectForm.value.amount,
      description: injectForm.value.description
    })
    ElMessage.success('资金变动干预指令执行成功！已成功载入账本。')
    await loadPoolLogs()
    await loadLottoControlPanel()
  } catch {}
}

// 保存奖配置
const savePrizeConfigs = async () => {
  try {
    const dataToSave = prizeConfigs.value.map(p => ({
      level: p.level,
      label: p.label,
      rewardType: p.rewardType,
      empPower: p.level === 'level4' ? 50 : (p.empPower || 0), // 参与奖写死50赋能
      moralPower: p.level === 'level4' ? 0 : (p.moralPower || 0),
      maxWinners: p.maxWinners,
      color: p.color
    }))
    await request.post('/admin/game/lotto/prize-configs', {
      configValue: JSON.stringify(dataToSave)
    })
    ElMessage.success('各级奖项中奖名额与分值发放类型已成功保存并立即生效！')
    await loadLottoControlPanel()
  } catch {}
}

// 单行一键驳回投注资格并退回款项 (积分或打卡)
const rejectParticipant = (row: LottoEntry) => {
  ElMessageBox.confirm(
    `确定要驳回【${row.realName || row.nickname}】的参玩资格并启动原路退款/重置打卡退回大事务吗？`,
    '投注资格裁撤与极速原路退回',
    {
      confirmButtonText: '确定驳回',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await request.post('/admin/game/lotto/reject', {
        weekNum: currentWeekNum.value,
        userId: row.userId
      })
      ElMessage.success(`已成功驳回并退回该同学投注！积分/打卡已原路恢复！`)
      await loadLottoControlPanel()
    } catch {}
  })
}

// 🧹 一键清空本期参玩学生序列
const handleClearAllParticipants = () => {
  ElMessageBox.confirm(
    '您确定要一键清空本期所有参与大乐透摇号的学生名单吗？清空后此操作不可逆，学生需重新进入小程序选择号码！',
    '严苛警告',
    {
      confirmButtonText: '确定全部清空',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    try {
      await request.post('/admin/game/lotto/clear-participants')
      ElMessage.success('🧹 本期参玩学生序列已安全清空！')
      await loadLottoControlPanel()
    } catch (e: any) {
      ElMessage.error('清空失败: ' + (e.message || '网络异常'))
    }
  })
}

// 🎮 切换创意工坊整体开放状态
const handleToggleWorkshop = async (val: boolean) => {
  const statusStr = val ? 'open' : 'closed'
  try {
    await request.post('/admin/game/workshop/active-status', { status: statusStr })
    ElMessage.success(`🎉 创意工坊状态已成功更新为：${val ? '🟢 开放中' : '🔴 已锁定'}`)
    await loadLottoControlPanel()
  } catch (e: any) {
    workshopEnabled.value = !val // 失败则恢复原状态
    ElMessage.error('更新创意工坊状态失败: ' + (e.message || '网络异常'))
  }
}

// 批量多选驳回与退款功能
const handleBatchReject = () => {
  if (selectedUserIds.value.length === 0) return
  
  ElMessageBox.confirm(
    `警报：确定要批量驳回这 ${selectedUserIds.value.length} 名学生的本期大乐透参与资格吗？后端将执行高并发原子大事务，对积分用户原路批量退还积分，打卡用户批量释放重新打卡！`,
    '批量资格裁撤与原路批量退款',
    {
      confirmButtonText: '确定批量驳回',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--danger',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await request.post('/admin/game/lotto/reject-batch', {
        weekNum: currentWeekNum.value,
        userIds: selectedUserIds.value
      })
      ElMessage.success('批量驳回并退款指令执行大获成功！')
      selectedUserIds.value = []
      await loadLottoControlPanel()
    } catch {}
  })
}

// ==================== 🎲 现场开奖 Draft 草拟算法与弹窗 ====================

// 点击“现场全服开奖”按钮：不直接结算，而是打开草拟结果 Dialog，展示加权算法中奖名单
const handleManualDraw = () => {
  if (entryList.value.length === 0) {
    return ElMessage.warning('本期暂无学生参与锁定老虎机数字，无法草拟开奖名单')
  }
  showDraftDrawDlg.value = true
  fetchDraftDrawList()
}

// 发起请求，索取后端由“内幕隐藏加权算法”草拟的中奖名单
const fetchDraftDrawList = async () => {
  loadingDraft.value = true
  try {
    const res: any = await request.post('/admin/game/lotto/draft-draw', {
      weekNum: currentWeekNum.value
    })
    if (res && res.data) {
      draftWinners.value = res.data
      if (draftWinners.value.length === 0) {
        ElMessage.warning('未能抽选出符合名额配置的中奖候选人，请检查锁定名单人数')
      }
    }
  } catch {
    showDraftDrawDlg.value = false
  } finally {
    loadingDraft.value = false
  }
}

// 确认此名单并下发派发结算大事务
const confirmDraftAndDrawFinal = () => {
  if (draftWinners.value.length === 0) {
    return ElMessage.warning('名单为空，无法执行开奖！')
  }

  ElMessageBox.confirm(
    `核对：本次开奖将对已公示的一二三等奖名单进行最终物理派发。参与奖（人人有份）将同时自动核发 50 赋能分。本操作扣除大池积分，写入账单且不可撤销，确定开奖吗？`,
    '最高最终结算派奖发令',
    {
      confirmButtonText: '确定现场全服开奖',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--success',
      type: 'success'
    }
  ).then(async () => {
    try {
      // 组装符合后端 WinnerItem 类型的发奖报文
      const winnersPayload = draftWinners.value.map(w => {
        const cfg = prizeConfigs.value.find(p => p.level === w.prizeLevel)
        return {
          userId: w.userId,
          prizeLevel: w.prizeLevel,
          prizeLabel: w.prizeLabel,
          empPower: cfg ? cfg.empPower : 0,
          moralPower: cfg ? cfg.moralPower : 0
        }
      })

      await request.post('/admin/game/lotto-setting', {
        weekNum: currentWeekNum.value,
        isDrawn: true,
        winners: winnersPayload
      })

      ElMessage({
        message: '🚀 开奖大功告成！双奖池扣减明细已完美记账，期数已自动进位至下一期！',
        type: 'success',
        duration: 6000
      })

      showDraftDrawDlg.value = false
      draftWinners.value = []
      await loadLottoControlPanel()
    } catch {}
  })
}

// 样式颜色映射助手
const getPrizeLevelColor = (level: string) => {
  switch (level) {
    case 'level1': return 'text-red-600'
    case 'level2': return 'text-orange-500'
    case 'level3': return 'text-amber-500'
    default: return 'text-slate-600'
  }
}

// 描述分值明细显示
const getPrizeConfigSummary = (level: string) => {
  const cfg = prizeConfigs.value.find(p => p.level === level)
  if (!cfg) return '-'
  if (cfg.rewardType === 'empowerment') {
    return `${cfg.empPower} 赋能分`
  } else if (cfg.rewardType === 'moral') {
    return `${cfg.moralPower} 德育分`
  } else {
    return `${cfg.empPower}赋能 + ${cfg.moralPower}德育`
  }
}

// 格式化时间
const formatTime = (timeStr: string) => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', { hour12: false })
}

// 账本状态Tag
const getLogTagType = (type: string) => {
  switch (type) {
    case 'user_entry': return 'warning'
    case 'user_draw': return 'success'
    case 'admin_inject': return 'primary'
    case 'admin_deduct': return 'danger'
    default: return 'info'
  }
}

const getLogTypeName = (type: string) => {
  switch (type) {
    case 'user_entry': return '学生参与'
    case 'user_draw': return '派奖支出'
    case 'admin_inject': return '管理员注入'
    case 'admin_deduct': return '扣减干预'
    default: return '未知变动'
  }
}

onMounted(() => {
  loadLottoControlPanel()
})
</script>

<style scoped>
.premium-dialog :deep(.el-tabs--border-card) {
  border: none;
  background: transparent;
}
.premium-dialog :deep(.el-tabs__header) {
  background-color: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}
.premium-dialog :deep(.el-tabs__content) {
  padding: 16px;
}
</style>