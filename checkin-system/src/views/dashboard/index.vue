<template>
  <div class="p-6 bg-slate-50 min-h-screen flex flex-col gap-6 font-sans select-none" v-loading="loading">

    <!-- 头部标题栏 -->
    <div class="flex justify-between items-center pb-2 border-b border-slate-200">
      <h2 class="text-xl font-black text-slate-800 tracking-tight flex items-center gap-2">
        📊 宏观运营与遥测大盘
        <el-tag size="small" type="success" effect="dark" class="font-mono">43.143.215.16</el-tag>
      </h2>
      <el-button type="primary" size="small" class="font-bold shadow-sm" @click="refreshAllData">
        🔄 立即刷新大盘
      </el-button>
    </div>

    <!-- 1. 金融总账 KPI 卡片 (4 宏观绝对值) -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">

      <div class="bg-gradient-to-br from-blue-500 to-blue-700 rounded-2xl p-5 text-white shadow-md relative overflow-hidden hover:scale-[1.02] transition-all">
        <span class="text-[10px] font-bold opacity-75 uppercase tracking-widest block">累计发放赋能分</span>
        <span class="text-3xl font-black block mt-2">{{ ledger.totalIssuedEmpower }}</span>
        <span class="text-xs opacity-60 block mt-1">分</span>
        <span class="absolute bottom-1 right-3 text-5xl opacity-10 pointer-events-none">⚡</span>
      </div>

      <div class="bg-gradient-to-br from-rose-500 to-rose-700 rounded-2xl p-5 text-white shadow-md relative overflow-hidden hover:scale-[1.02] transition-all">
        <span class="text-[10px] font-bold opacity-75 uppercase tracking-widest block">累计消耗赋能分</span>
        <span class="text-3xl font-black block mt-2">{{ ledger.totalConsumedEmpower }}</span>
        <span class="text-xs opacity-60 block mt-1">分</span>
        <span class="absolute bottom-1 right-3 text-5xl opacity-10 pointer-events-none">🛍️</span>
      </div>

      <div class="bg-gradient-to-br from-emerald-500 to-emerald-700 rounded-2xl p-5 text-white shadow-md relative overflow-hidden hover:scale-[1.02] transition-all">
        <span class="text-[10px] font-bold opacity-75 uppercase tracking-widest block">累计发放德育分</span>
        <span class="text-3xl font-black block mt-2">{{ ledger.totalIssuedMoral }}</span>
        <span class="text-xs opacity-60 block mt-1">分</span>
        <span class="absolute bottom-1 right-3 text-5xl opacity-10 pointer-events-none">🌿</span>
      </div>

      <div class="bg-gradient-to-br from-amber-500 to-amber-700 rounded-2xl p-5 text-white shadow-md relative overflow-hidden hover:scale-[1.02] transition-all">
        <span class="text-[10px] font-bold opacity-75 uppercase tracking-widest block">全网在籍总积分池</span>
        <span class="text-3xl font-black block mt-2">{{ ledger.totalInStockPool }}</span>
        <span class="text-xs opacity-60 block mt-1">分</span>
        <span class="absolute bottom-1 right-3 text-5xl opacity-10 pointer-events-none">🏆</span>
      </div>

    </div>

    <!-- 2. 中段：赛道打卡态势柱状图（左 2/3）+ 硬件遥测（右 1/3）-->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">

      <!-- 左侧：ECharts 双向并列柱状图 -->
      <div class="bg-white rounded-3xl border border-slate-200/80 p-5 shadow-sm lg:col-span-2 min-h-[380px] flex flex-col">
        <div class="flex justify-between items-center border-b border-slate-100 pb-3 mb-3">
          <span class="text-sm font-black text-slate-800 flex items-center gap-1.5">🏁 四大赛道今日打卡态势</span>
          <el-tag size="small" type="info" effect="plain">今日实时</el-tag>
        </div>
        <!-- 非空拦截：数据未到之前不初始化 echarts -->
        <div id="trackAttendanceBarChart" class="w-full flex-1 min-h-[300px]" />
      </div>

      <!-- 右侧：物理硬件遥测（5 秒轮询） -->
      <div class="bg-white rounded-3xl border border-slate-200/80 p-5 shadow-sm min-h-[380px] flex flex-col gap-3">
        <div class="flex justify-between items-center border-b border-slate-100 pb-3">
          <span class="text-sm font-black text-slate-800 flex items-center gap-1.5">🖥️ 主机物理监测</span>
          <el-tag size="small" type="warning" effect="plain">5s 自动刷新</el-tag>
        </div>

        <div class="grid grid-cols-2 gap-4 flex-1 items-center justify-items-center py-4">
          <!-- CPU -->
          <div class="flex flex-col items-center gap-2">
            <el-progress type="circle" :percentage="monitor.cpuUsage" :width="82" :stroke-width="7" color="#3B82F6" />
            <span class="text-xs font-bold text-slate-500">CPU 负载率</span>
          </div>
          <!-- 内存 -->
          <div class="flex flex-col items-center gap-2">
            <el-progress type="circle" :percentage="monitor.memoryUsage" :width="82" :stroke-width="7" color="#8B5CF6" />
            <span class="text-xs font-bold text-slate-500">内存占用率</span>
          </div>
          <!-- 磁盘 -->
          <div class="flex flex-col items-center gap-2">
            <el-progress type="circle" :percentage="diskPercent" :width="82" :stroke-width="7" color="#10B981">
              <template #default>
                <span class="text-xs font-bold text-slate-700">{{ Math.round(monitor.diskFree) }}G</span>
              </template>
            </el-progress>
            <span class="text-xs font-bold text-slate-500">磁盘剩余</span>
          </div>
          <!-- 延迟 -->
          <div class="flex flex-col items-center gap-2">
            <el-progress type="circle" :percentage="pingPercent" :width="82" :stroke-width="7" color="#EF4444">
              <template #default>
                <span class="text-xs font-bold text-slate-700">{{ monitor.pingLatency }}ms</span>
              </template>
            </el-progress>
            <span class="text-xs font-bold text-slate-500">心跳延迟</span>
          </div>
        </div>
      </div>

    </div>

    <!-- 3. 底部：在籍成长荣誉龙虎榜 Top 10 -->
    <div class="bg-white rounded-3xl border border-slate-200/80 p-6 shadow-sm">
      <div class="flex justify-between items-center border-b border-slate-100 pb-3 mb-4">
        <span class="text-sm font-black text-slate-800 flex items-center gap-1.5">🏆 赋能成长荣誉龙虎榜 (Top 10)</span>
        <el-tag size="small" type="info" effect="plain" class="font-mono">Redis 10min 缓存</el-tag>
      </div>

      <el-table :data="leaderboard" size="small" border stripe>
        <el-table-column type="index" label="荣誉位" width="80" align="center">
          <template #default="{ $index }">
            <span
              class="text-sm font-black"
              :class="[$index === 0 ? 'text-amber-500' : $index === 1 ? 'text-slate-400' : $index === 2 ? 'text-amber-700' : 'text-slate-400/70']"
            >{{ $index + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="姓名" width="140">
          <template #default="{ row }">{{ row.nickname || row.real_name }}</template>
        </el-table-column>
        <el-table-column prop="student_id" label="学号" width="180" />
        <el-table-column prop="class_name" label="班级" show-overflow-tooltip />
        <el-table-column label="赋能积分" width="160" align="right">
          <template #default="{ row }">
            <span class="font-black text-blue-600">{{ row.empowerment_score }} 分</span>
          </template>
        </el-table-column>
      </el-table>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import request from '@/api/request.ts'
import * as echarts from 'echarts'

const loading = ref(false)

// ── 1. 金融总账 ──────────────────────────────────────────
const ledger = ref({
  totalIssuedEmpower: 0,
  totalConsumedEmpower: 0,
  totalIssuedMoral: 0,
  totalInStockPool: 0
})

// ── 2. 赛道打卡态势图 ────────────────────────────────────
const trackAttendanceList = ref<any[]>([])
let attendanceChart: echarts.ECharts | null = null

// ── 3. 硬件遥测探针 ──────────────────────────────────────
const monitor = ref({
  cpuUsage: 0,
  memoryUsage: 0,
  diskFree: 120,
  pingLatency: 0
})

// 磁盘剩余百分比（假设总容量 256G）
const diskPercent = computed(() => {
  const totalDisk = 256
  return Math.min(100, Math.max(0, Math.round((monitor.value.diskFree / totalDisk) * 100)))
})

// 延迟转百分比（100ms 封顶为 0%，越低越满）
const pingPercent = computed(() => {
  return Math.min(100, Math.max(0, 100 - Math.round(monitor.value.pingLatency)))
})

// ── 4. 排行榜 ────────────────────────────────────────────
const leaderboard = ref<any[]>([])

let monitorTimer: ReturnType<typeof setInterval> | null = null

// ═════════════════════════════════════════════════════════
//  全量刷新入口
// ═════════════════════════════════════════════════════════
const refreshAllData = async () => {
  loading.value = true
  try {
    await Promise.all([
      fetchLedgerAndTracks(),
      fetchMonitorStats(),
      fetchLeaderboard()
    ])
  } finally {
    loading.value = false
  }
}

// ── 获取金融总账 + 赛道打卡态势 ──────────────────────────
const fetchLedgerAndTracks = async () => {
  try {
    const res: any = await request.get('/admin/dashboard')
    const data = res.data ? res.data : res

    ledger.value = data.ledger ?? {
      totalIssuedEmpower: 0,
      totalConsumedEmpower: 0,
      totalIssuedMoral: 0,
      totalInStockPool: 0
    }

    trackAttendanceList.value = data.trackAttendance ?? []

    nextTick(() => {
      renderAttendanceChart()
    })
  } catch (e) {
    console.error('[Dashboard] 总账数据加载失败:', e)
  }
}

// ── ECharts 双向并列柱状图（严格非空拦截，防冷启动白屏） ──
const renderAttendanceChart = () => {
  // 💡 非空拦截：数据未到前直接 return 挂起
  if (!trackAttendanceList.value || trackAttendanceList.value.length === 0) return

  const el = document.getElementById('trackAttendanceBarChart')
  if (!el) return

  if (!attendanceChart) {
    attendanceChart = echarts.init(el)
  }

  const names = trackAttendanceList.value.map((t: any) => t.trackName)
  const totalUsers = trackAttendanceList.value.map((t: any) => t.totalUsers)
  const checkedIn = trackAttendanceList.value.map((t: any) => t.checkedIn)

  attendanceChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { bottom: 0, data: ['参与总人数', '今日成功打卡'] },
    grid: { left: '3%', right: '4%', bottom: '12%', top: '10%', containLabel: true },
    xAxis: {
      type: 'category',
      data: names,
      axisLabel: { color: '#64748b', fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { type: 'dashed' } }
    },
    series: [
      {
        name: '参与总人数',
        type: 'bar',
        barWidth: '25%',
        itemStyle: {
          borderRadius: [4, 4, 0, 0],
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#3B82F6' },
            { offset: 1, color: '#1D4ED8' }
          ])
        },
        data: totalUsers
      },
      {
        name: '今日成功打卡',
        type: 'bar',
        barWidth: '25%',
        itemStyle: {
          borderRadius: [4, 4, 0, 0],
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#10B981' },
            { offset: 1, color: '#047857' }
          ])
        },
        data: checkedIn
      }
    ]
  }, { notMerge: true })
}

// ── 物理硬件遥测接口 ─────────────────────────────────────
const fetchMonitorStats = async () => {
  try {
    const res: any = await request.get('/admin/telemetry/probe')
    const data = res.data ? res.data : res
    monitor.value = {
      cpuUsage: Math.round(data.cpuUsage ?? 0),
      memoryUsage: Math.round(data.memoryUsage ?? 0),
      diskFree: Math.round(data.diskFree ?? 120),
      pingLatency: data.pingLatency ?? 0
    }
  } catch (e) {
    console.error('[Dashboard] 物理探针数据调度失败:', e)
  }
}

// ── 排行榜加载 ───────────────────────────────────────────
const fetchLeaderboard = async () => {
  try {
    const res: any = await request.get('/admin/leaderboard')
    leaderboard.value = res.data ? res.data : res
  } catch (e) {
    console.error('[Dashboard] 排行榜加载失败:', e)
  }
}

// ═════════════════════════════════════════════════════════
//  生命周期
// ═════════════════════════════════════════════════════════
onMounted(() => {
  refreshAllData()

  // 5 秒轮询物理硬件探针
  monitorTimer = setInterval(() => {
    fetchMonitorStats()
  }, 5000)

  window.addEventListener('resize', () => {
    attendanceChart?.resize()
  })
})

onUnmounted(() => {
  if (monitorTimer) clearInterval(monitorTimer)
  attendanceChart?.dispose()
  attendanceChart = null
})
</script>

<style scoped>
:deep(.el-table) {
  --el-table-header-bg-color: #f8fafc;
  --el-table-header-text-color: #475569;
  font-size: 11px;
}
</style>