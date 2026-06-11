<template>
  <div class="p-6 max-w-[1600px] mx-auto bg-slate-50 min-h-screen flex flex-col gap-6 font-sans">
    <!-- 头部横幅：物理剔除冗余标题，仅保留关键上架操作 -->
    <div class="flex justify-end pb-4 border-b border-slate-200 gap-4 shrink-0">
      <div class="flex gap-3 shrink-0">
        <el-button type="primary" class="font-bold shadow-md shadow-blue-50" @click="prepareAdd">
          ➕ 上架道具
        </el-button>
      </div>
    </div>

    <!-- 模块一：货架商品管理 (主界面已不再区分页签) -->
    <div class="flex-1 flex flex-col gap-6">
      <!-- 正在销售 / 已售罄 / 已下架 三级分类导航 -->
      <div class="bg-white rounded-xl border border-slate-200 p-4 shadow-sm flex flex-wrap gap-4 items-center justify-between shrink-0">
        <el-radio-group v-model="filterStatus" class="flex gap-2">
          <!-- 🟢 正在销售 -->
          <el-radio-button value="selling">
            🟢 正在销售 ({{ sellingItems.length }})
          </el-radio-button>
          <!-- 🔴 已售罄 -->
          <el-radio-button value="soldout">
            🔴 已售罄 ({{ soldOutItems.length }})
          </el-radio-button>
          <!-- ⚪ 已下架礼包 -->
          <el-radio-button value="offshelf">
            ⚪ 已下架 ({{ offShelfItems.length }})
          </el-radio-button>
        </el-radio-group>
      </div>

      <!-- 缺省页 -->
      <div v-if="displayedItems.length === 0 && !loading" class="bg-white rounded-2xl border border-slate-200 p-16 text-center shadow-sm">
        <div class="text-6xl mb-4">🏪</div>
        <h3 class="text-lg font-bold text-slate-800">货架空空如也</h3>
        <p class="text-sm text-slate-400 mt-1 max-w-md mx-auto">当前分类下无任何道具。点击右上角“上架道具”按钮</p>
      </div>

      <!-- 商品流式货架卡片 -->
      <div v-else v-loading="loading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div 
          v-for="(item, i) in displayedItems" 
          :key="item.id" 
          class="bg-white rounded-2xl border border-slate-150 p-5 flex flex-col shadow-sm hover:-translate-y-1.5 hover:shadow-lg hover:border-emerald-200 transition-all duration-300 relative overflow-hidden"
          :class="!item.isActive ? 'border-dashed opacity-80' : ''"
        >
          <!-- 卡片头部标签 -->
          <div class="flex justify-between items-center pb-3 border-b border-dashed border-slate-200 mb-4">
            <span class="font-mono text-xs text-slate-400 font-bold bg-slate-50 border border-slate-100 px-2 py-0.5 rounded">NO.{{ String(i + 1).padStart(2, '0') }}</span>
            <el-tag 
              :type="item.type === 'moral' ? 'danger' : item.type === 'emp' ? 'success' : 'warning'"
              effect="dark" 
              size="small" 
              class="font-bold"
            >
              {{ item.type === 'moral' ? '德育分' : item.type === 'emp' ? '赋能分' : '补签卡' }}
            </el-tag>
          </div>

          <!-- 商品名称 -->
          <h3 class="text-lg font-black text-slate-800 mb-4 line-clamp-1 title-badge-adjust">{{ item.name }}</h3>

          <!-- 商品规格细则 -->
          <div class="flex flex-col gap-2 flex-1 mb-5">
            <div class="flex justify-between items-center bg-slate-50 border border-slate-100 px-3 py-2 rounded-lg text-xs">
              <span class="text-slate-400 font-semibold">标价售价</span>
              <b v-if="item.empCost === 0" class="text-emerald-600 font-black text-sm">🎁 免费发放</b>
              <b v-else class="text-amber-600 font-black text-sm">{{ item.empCost }} 赋能分</b>
            </div>
            <div class="flex justify-between items-center bg-slate-50 border border-slate-100 px-3 py-2 rounded-lg text-xs">
              <span class="text-slate-400 font-semibold">{{ item.type === 'card' ? '包含数量' : '内含分值' }}</span>
              <b class="text-slate-700 font-bold">{{ item.rewardValue }} {{ item.type === 'card' ? '张' : '分' }}</b>
            </div>
            <div class="flex justify-between items-center bg-slate-50 border border-slate-100 px-3 py-2 rounded-lg text-xs">
              <span class="text-slate-400 font-semibold">剩余库存</span>
              <b :class="item.stock === 0 ? 'text-rose-500 font-black' : 'text-slate-700 font-bold'">
                {{ item.stock === -1 ? '♾️ 无限供应' : item.stock + ' 件' }}
              </b>
            </div>
            <!-- 限购展示 -->
            <div class="flex justify-between items-center bg-slate-50 border border-slate-100 px-3 py-2 rounded-lg text-xs">
              <span class="text-slate-400 font-semibold">抢购限制</span>
              <b :class="item.limitType === 'none' ? 'text-slate-400 font-medium' : 'text-indigo-600 font-black'">
                {{ getLimitTypeText(item) }}
              </b>
            </div>
            <!-- 班级定向限制 -->
            <div class="flex justify-between items-center bg-slate-50 border border-slate-100 px-3 py-2 rounded-lg text-xs">
              <span class="text-slate-400 font-semibold">定向发放</span>
              <b :class="!item.targetClass || item.targetClass === '全部' ? 'text-slate-400 font-medium' : 'text-blue-600 font-black'">
                {{ !item.targetClass || item.targetClass === '全部' ? '面向全校公开' : `定向: ${item.targetClass}` }}
              </b>
            </div>
            <!-- 定时开抢 -->
            <div v-if="item.releaseTime" class="flex justify-between items-center bg-slate-50 border border-slate-100 px-3 py-2 rounded-lg text-xs">
              <span class="text-slate-400 font-semibold">限时售卖</span>
              <span class="text-blue-600 font-mono text-[11px] font-bold">{{ item.releaseTime }}</span>
            </div>
          </div>

          <!-- 操作按钮：三合一卡片控制 -->
          <div class="flex gap-2">
            <el-button size="small" type="primary" plain class="flex-1 font-bold !px-2" @click="prepareEdit(item)">
              编辑参数
            </el-button>
            <el-button size="small" type="success" plain class="flex-1 font-bold !px-2" @click="openExchangeStats(item)">
              兑换统计
            </el-button>
            <el-button 
              size="small" 
              :type="item.isActive ? 'danger' : 'success'" 
              plain 
              class="flex-1 font-bold !px-2" 
              @click="toggleShelf(item)"
            >
              {{ item.isActive ? '下架商品' : '重新上架' }}
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 上架新商品 / 编辑商品参数 弹窗 -->
    <el-dialog
      v-model="dlgVisible"
      :title="form.id ? '⚙️ 编辑物资商品参数' : '🏪 上架新物资商品'"
      width="560px"
      class="rounded-xl overflow-hidden custom-dialog"
      align-center
    >
      <el-form :model="form" label-position="top" class="space-y-4">
        <!-- 商品名称 -->
        <el-form-item label="商品物资名称 (必填)" required>
          <el-input v-model="form.name" placeholder="如：补签卡" class="rounded-lg" />
        </el-form-item>

        <!-- 商品分类 -->
        <el-form-item label="物资分类归属" required>
          <el-radio-group v-model="form.type" class="w-full flex gap-2">
            <el-radio-button value="emp" class="flex-1 text-center font-bold">赋能分</el-radio-button>
            <el-radio-button value="moral" class="flex-1 text-center font-bold">德育分</el-radio-button>
            <el-radio-button value="card" class="flex-1 text-center font-bold">补签卡</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 价格与价值配置 -->
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="兑换价格（消耗赋能分）" required>
            <el-input-number v-model="form.empCost" :min="0" class="!w-full" />
            <div class="text-[10px] text-slate-400 mt-1">设为 0 即代表该物资为免费赠赠送</div>
          </el-form-item>
          <el-form-item :label="form.type === 'card' ? '内含卡片数量 (张)' : '内含积分分值 (分)'" required>
            <el-input-number v-model="form.rewardValue" :min="1" class="!w-full" />
            <div class="text-[10px] text-slate-400 mt-1">兑换成功后学生资产将增加对应数额</div>
          </el-form-item>
        </div>

        <!-- 班级定向推送与上架状态 -->
        <div class="bg-blue-50/50 border border-blue-100 rounded-xl p-4 space-y-4">
          <div class="text-xs font-bold text-blue-700 flex items-center gap-1">
            <span>📢 班级定向派发与在售开关 (class_list 对接)</span>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <el-form-item label="定向限制班级" class="!mb-0">
              <el-select v-model="form.targetClass" placeholder="留空面向全校开放" class="w-full">
                <el-option label="📢 面向全校所有班级开放" value="" />
                <el-option 
                  v-for="cls in classOptions" 
                  :key="cls" 
                  :label="cls" 
                  :value="cls" 
                />
              </el-select>
            </el-form-item>
            <el-form-item label="是否激活上架" class="!mb-0" v-if="form.id">
              <el-switch 
                v-model="form.isActive" 
                active-text="上架销售" 
                inactive-text="暂时下架" 
                class="mt-2"
              />
            </el-form-item>
          </div>
        </div>

        <!-- 风控限制配置卡片 -->
        <div class="bg-indigo-50/50 border border-indigo-100 rounded-xl p-4 space-y-4">
          <div class="text-xs font-bold text-indigo-700 flex items-center gap-1">
            <span>🔒 抢购周期风控限购设计</span>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <el-form-item label="账号限制周期" class="!mb-0">
              <el-select v-model="form.limitType" class="w-full">
                <el-option label="🔓 不限制购买频次" value="none" />
                <el-option label="👤 终身账号限制领取" value="lifetime" />
                <el-option label="📅 每周账号限制领取" value="weekly" />
                <el-option label="☀️ 每日账号限制领取" value="daily" />
              </el-select>
            </el-form-item>
            <el-form-item label="限制购买数量 (件)" class="!mb-0" :disabled="form.limitType === 'none'">
              <el-input-number 
                v-model="form.limitCount" 
                :min="1" 
                :disabled="form.limitType === 'none'"
                class="!w-full" 
              />
            </el-form-item>
          </div>
        </div>

        <!-- 饥饿营销库存定时卡片 -->
        <div class="bg-amber-50/50 border border-amber-100 rounded-xl p-4 space-y-4">
          <div class="text-xs font-bold text-amber-700 flex items-center gap-1">
            <span>🔥 定时销售</span>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <el-form-item label="限量库存量 (件)" class="!mb-0" required>
              <el-input-number v-model="form.stock" :min="-1" class="!w-full" />
              <div class="text-[10px] text-amber-600 mt-1">填写 -1 代表商品无限供应</div>
            </el-form-item>
            <el-form-item label="定时售卖开启时间" class="!mb-0">
              <el-date-picker 
                v-model="form.releaseTime" 
                type="datetime"
                placeholder="留空即刻开抢"
                format="YYYY-MM-DD HH:mm:ss" 
                value-format="YYYY-MM-DD HH:mm:ss" 
                class="!w-full" 
              />
            </el-form-item>
          </div>
        </div>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-3 pt-2">
          <el-button @click="dlgVisible = false">取消</el-button>
          <el-button 
            type="primary" 
            :loading="submitLoading" 
            class="font-bold shadow-md shadow-blue-50"
            @click="doSave"
          >
            确认上架至货架
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 弹窗三：单个物资礼包兑换统计与秒级退回冲正面板 (已根据要求放大为宽 1100px 殿堂级弹窗，并只保留兑换成功与已退回) -->
    <el-dialog
      v-model="dlgStatsVisible"
      :title="`📊 【${statsItemName}】物资礼包兑换统计名册`"
      width="1100px"
      class="rounded-xl overflow-hidden custom-dialog"
      align-center
    >
      <div v-loading="statsLoading" class="flex flex-col gap-4">
        <!-- 缺省提示 -->
        <div v-if="statsLogs.length === 0 && !statsLoading" class="p-12 text-center text-slate-400 text-sm">
          暂无任何学生兑换此礼包商品的记录。
        </div>
        
        <!-- 名单列表 -->
        <el-table v-else :data="statsLogs" stripe height="450px" class="w-full text-slate-700">
          <el-table-column type="index" label="序列" width="75" align="center">
            <template #default="scope">
              <span class="font-mono text-xs text-slate-400 font-bold">#{{ scope.$index + 1 }}</span>
            </template>
          </el-table-column>
          
          <!-- 学生资料 -->
          <el-table-column label="兑换学生信息" min-width="220">
            <template #default="{ row }">
              <div class="flex items-center gap-3">
                <el-avatar :size="32" :src="row.avatar || '/logo.png'" class="border border-slate-100 shrink-0" />
                <div class="flex flex-col">
                  <span class="font-bold text-slate-800 text-sm">{{ row.name }}</span>
                  <span class="text-xs text-slate-400 font-mono mt-0.5">{{ row.studentId }} | {{ row.className }}</span>
                </div>
              </div>
            </template>
          </el-table-column>

          <!-- 物资大类 -->
          <el-table-column label="物资归属" width="120" align="center">
            <template #default="{ row }">
              <el-tag 
                :type="row.itemType === 'moral' ? 'danger' : row.itemType === 'emp' ? 'success' : 'warning'"
                size="small" 
                effect="plain"
                class="font-bold"
              >
                {{ row.itemType === 'moral' ? '德育分' : row.itemType === 'emp' ? '赋能分' : '补签卡' }}
              </el-tag>
            </template>
          </el-table-column>

          <!-- 消耗分值 -->
          <el-table-column label="消耗售价" width="130" align="center">
            <template #default="{ row }">
              <span class="font-bold text-amber-600">-{{ row.empCost }} 赋能分</span>
            </template>
          </el-table-column>

          <!-- 追授面额 -->
          <el-table-column label="追授面值" width="130" align="center">
            <template #default="{ row }">
              <span class="font-bold text-emerald-600">+{{ row.rewardValue }} {{ row.itemType === 'card' ? '张卡' : '分' }}</span>
            </template>
          </el-table-column>

          <!-- 状态：只支持 🟢 兑换成功 和 ↩️ 已退回 -->
          <el-table-column label="工单状态" width="130" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)" size="small" class="font-bold">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>

          <!-- 秒级时间 -->
          <el-table-column label="精确兑换发生时间" width="200" align="center">
            <template #default="{ row }">
              <span class="text-xs text-slate-400 font-mono">{{ row.exchangeTime }}</span>
            </template>
          </el-table-column>

          <!-- 退回操作 -->
          <el-table-column label="战术指令" width="120" align="center" fixed="right">
            <template #default="{ row }">
              <el-button 
                v-if="row.status === 1"
                size="small" 
                type="danger" 
                plain 
                class="font-bold" 
                @click="doRevokeExchangeInStats(row)"
              >
                退回撤销
              </el-button>
              <span v-else class="text-slate-400 text-xs italic">已退冲正</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'

// 货架商品强类型定义
interface ShopItem {
  id?: number | string
  name: string
  type: 'emp' | 'moral' | 'card'
  empCost: number
  rewardValue: number
  stock: number
  releaseTime?: string
  limitType: 'none' | 'lifetime' | 'weekly' | 'daily'
  limitCount: number
  isActive: boolean
  targetClass?: string
}

// 兑换审计强类型定义
interface ExchangeLog {
  id: number | string
  userId: number | string
  name: string
  studentId: string
  className: string
  avatar?: string
  itemId: number | string
  itemName: string
  itemType: 'emp' | 'moral' | 'card'
  empCost: number
  rewardValue: number
  status: 1 | 4 // 1: 兑换成功, 4: 已全额退款
  exchangeTime: string
  handlerName?: string
  handleTime?: string
}

// 原子化变量
const loading = ref<boolean>(false)
const submitLoading = ref<boolean>(false)
const shopItems = ref<ShopItem[]>([])
const classOptions = ref<string[]>([]) // 班级字典库

// 分流视图状态变量
const filterStatus = ref<'selling' | 'soldout' | 'offshelf'>('selling')

// 兑换统计子窗体控制变量
const dlgStatsVisible = ref<boolean>(false)
const statsLoading = ref<boolean>(false)
const statsItemName = ref<string>('')
const statsLogs = ref<ExchangeLog[]>([])

// 弹窗与表单初始化
const dlgVisible = ref<boolean>(false)
const form = ref<ShopItem>({
  name: '',
  type: 'emp',
  empCost: 0,
  rewardValue: 10,
  stock: -1,
  releaseTime: '',
  limitType: 'none',
  limitCount: 1,
  isActive: true,
  targetClass: ''
})

// ==================== 📊 前端轻量化数据透视看板计算 ====================
const sellingItems = computed(() => shopItems.value.filter(item => item.isActive && (item.stock > 0 || item.stock === -1)))
const soldOutItems = computed(() => shopItems.value.filter(item => item.isActive && item.stock === 0))
const offShelfItems = computed(() => shopItems.value.filter(item => !item.isActive))

const displayedItems = computed(() => {
  if (filterStatus.value === 'selling') return sellingItems.value
  if (filterStatus.value === 'soldout') return soldOutItems.value
  return offShelfItems.value
})

// ==================== 🛠️ 规格文本格式化 ====================
const getLimitTypeText = (item: ShopItem): string => {
  if (!item.limitType || item.limitType === 'none') return '不设限'
  const count = item.limitCount || 1
  if (item.limitType === 'lifetime') return `终身限购 ${count} 件`
  if (item.limitType === 'weekly') return `每周限购 ${count} 件`
  if (item.limitType === 'daily') return `每日限购 ${count} 件`
  return '不设限'
}

// 物理状态：只保留 兑换成功 和 已退冲正
const getStatusText = (status: number): string => {
  if (status === 1) return '🟢 兑换成功'
  if (status === 4) return '↩️ 已退冲正'
  return '未知'
}

const getStatusTagType = (status: number): 'success' | 'warning' | 'danger' | 'info' => {
  if (status === 1) return 'success'
  if (status === 4) return 'info'
  return 'info'
}

// ==================== 🏪 货架商品接口逻辑 ====================

// 1. 加载商品货架列表数据
const loadItems = async () => {
  loading.value = true
  try {
    const res: any = await request.get('/api/admin/shop/list')
    shopItems.value = res.data?.data || []
  } catch {
    console.warn('获取商品列表接口异常')
  } finally {
    loading.value = false
  }
}

// 2. 加载班级字典名册
const loadClasses = async () => {
  try {
    const res: any = await request.get('/api/auth/classes')
    classOptions.value = res.data || []
  } catch {
    console.warn('获取班级字典异常')
  }
}

// 3. 准备新增上架商品
const prepareAdd = () => {
  form.value = {
    name: '',
    type: 'emp',
    empCost: 0,
    rewardValue: 10,
    stock: -1,
    releaseTime: '',
    limitType: 'none',
    limitCount: 1,
    isActive: true,
    targetClass: ''
  }
  dlgVisible.value = true
}

// 4. 准备编辑商品
const prepareEdit = (item: ShopItem) => {
  form.value = {
    ...item,
    limitType: item.limitType || 'none',
    limitCount: item.limitCount || 1,
    isActive: item.isActive === undefined ? true : item.isActive,
    targetClass: item.targetClass || ''
  }
  dlgVisible.value = true
}

// 5. 上架/更新商品入库
const doSave = async () => {
  if (!form.value.name.trim()) return ElMessage.warning('商品名称不能为空！')

  submitLoading.value = true
  try {
    await request.post('/api/admin/shop/save', form.value)
    ElMessage.success('商品参数已更新生效')
    dlgVisible.value = false
    await loadItems()
  } catch {
    ElMessage.error('保存失败')
  } finally {
    submitLoading.value = false
  }
}

// 6. 逻辑下架与重新上架控制
const toggleShelf = async (item: ShopItem) => {
  const isShelving = !item.isActive
  const actionText = isShelving ? '重新上架' : '下架'
  try {
    if (!isShelving) {
      await ElMessageBox.confirm(`将强制逻辑下架商品【${item.name}】，学生端将无法再看到该礼包，是否继续？`, '逻辑下架商品', {
        type: 'warning',
        confirmButtonText: '确定下架',
        cancelButtonText: '保留',
        confirmButtonClass: 'el-button--danger'
      })
      await request.post('/api/admin/shop/delete', { id: item.id })
      ElMessage.success('商品已安全逻辑下架（存入下架大厅）')
    } else {
      await request.post('/api/admin/shop/save', {
        ...item,
        isActive: true
      })
      ElMessage.success('该礼包已成功重新上架激活！')
    }
    await loadItems()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error(`${actionText}商品异常`)
    }
  }
}

// ==================== 📜 兑换统计弹窗业务 ====================

// 1. 弹出特定商品的兑换详情名册 (大框体)
const openExchangeStats = async (item: ShopItem) => {
  statsItemName.value = item.name
  dlgStatsVisible.value = true
  await fetchStatsLogs()
}

// 2. 拉取此特定商品的每一位兑换账目
const fetchStatsLogs = async () => {
  statsLoading.value = true
  try {
    const res: any = await request.get('/api/admin/shop/exchange/list', {
      params: {
        itemName: statsItemName.value
      }
    })
    statsLogs.value = res.data?.data || []
  } catch {
    statsLogs.value = []
  } finally {
    statsLoading.value = false
  }
}

// 3. 执行单条兑换的“退回并撤销”大事务操作
const doRevokeExchangeInStats = (row: ExchangeLog) => {
  ElMessageBox.confirm(`高危操作警报：您确定强制撤回学生【${row.name}】的这笔积分物资兑换吗？\n系统将自动退还其 ${row.empCost} 赋能分，并强制安全收回当初自动派发给他的礼包资产奖励，同时自动回补 1 件商品库存并清洗其限购次数！`, '退款并资产撤销确认', {
    type: 'warning',
    confirmButtonText: '立即全额撤销并冲退',
    cancelButtonText: '取消',
    confirmButtonClass: 'el-button--danger'
  }).then(async () => {
    try {
      await request.post('/api/admin/shop/exchange/refund', { id: row.id })
      ElMessage.success('全额撤销冲退平账成功！积分已退，物资已被成功追收回！')
      await fetchStatsLogs()
      await loadItems() // 同步重新加载商品货架，刷新剩余库存
    } catch {
      ElMessage.error('冲正退回撤销事务执行失败')
    }
  }).catch(() => {})
}

// 初始化加载
onMounted(() => {
  loadItems()
  loadClasses()
})
</script>

<style scoped>
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
  background-color: #eff6ff !important;
}
:deep(.el-table::before) {
  display: none;
}

/* 弹窗样式优化 */
.custom-dialog :deep(.el-dialog__header) {
  padding-bottom: 15px;
  border-bottom: 1px solid #f1f5f9;
  margin-right: 0;
}
.custom-dialog :deep(.el-dialog__body) {
  padding-top: 20px;
  padding-bottom: 20px;
}

/* 按钮与卡片交互样式优化 */
.title-badge-adjust {
  line-height: 1.4;
}
</style>
