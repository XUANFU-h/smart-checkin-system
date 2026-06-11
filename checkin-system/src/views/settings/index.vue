<template>
  <div class="p-6 max-w-[1600px] mx-auto bg-slate-50 min-h-screen flex flex-col gap-6 font-sans">
    <!-- Header Banner -->
    <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between pb-6 border-b border-slate-200 gap-4">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 flex items-center gap-2">
          ⚙️ 系统核心配置中心
        </h1>
        <p class="text-sm text-slate-500 mt-1">动态调整小程序端运行参数、系部管理员凭证与申诉分类大厅引擎。</p>
      </div>
    </div>

    <!-- Main Layout -->
    <el-row :gutter="25" class="flex-1 min-h-0">
      <!-- Left Column: Settings and Password -->
      <el-col :span="11" class="h-full flex flex-col gap-6 overflow-y-auto pr-1">
        <!-- Panel 1: Core Parameters -->
        <div class="bg-white/80 backdrop-blur-md border border-slate-200/80 rounded-2xl p-6 shadow-sm hover:shadow-md transition-all">
          <h3 class="text-lg font-bold text-emerald-600 border-b border-slate-100 pb-3 mb-5 flex items-center gap-2">
            ⏰ 核心运行参数控制
          </h3>
          <el-form label-position="left" label-width="160px" v-loading="settingsLoading">
            <el-form-item label="每日打卡截止时间">
              <el-time-picker
                v-model="sysSettings.deadline"
                format="HH:mm"
                value-format="HH:mm"
                class="!w-full"
                placeholder="选择截止时间"
              />
            </el-form-item>
            <el-form-item label="修改赛道截止时间">
              <el-date-picker
                v-model="sysSettings.trackEditDeadline"
                type="datetime"
                format="YYYY-MM-DD HH:mm:ss"
                value-format="YYYY-MM-DD HH:mm:ss"
                placeholder="超过此时小程序端将拦截修改"
                class="!w-full"
              />
            </el-form-item>
            <el-form-item label="德育分最高上限">
              <el-input-number
                v-model="sysSettings.maxMoralScore"
                :min="1"
                class="!w-full"
                placeholder="每日德育分上限值"
              />
            </el-form-item>
            <el-form-item label="赋能分最高上限">
              <el-input-number
                v-model="sysSettings.maxEmpScore"
                :min="1"
                class="!w-full"
                placeholder="成长赋能分上限值"
              />
            </el-form-item>
            <el-form-item label="单次最大加分上限">
              <el-input-number
                v-model="sysSettings.maxSingleAdd"
                :min="1"
                class="!w-full"
                placeholder="单次最大加分限制"
              />
            </el-form-item>
            <el-form-item label="单次最大减分上限">
              <el-input-number
                v-model="sysSettings.maxSingleSub"
                :min="1"
                class="!w-full"
                placeholder="单次最大减分限制"
              />
            </el-form-item>

            <div class="mt-6">
              <el-button
                type="primary"
                color="#07c160"
                size="large"
                class="w-full !rounded-xl font-bold shadow-md shadow-emerald-50 hover:scale-[1.01] transition-transform"
                :loading="saveSettingsLoading"
                @click="saveSettings"
              >
                💾 一键保存所有核心参数
              </el-button>
            </div>
          </el-form>
        </div>

        <!-- Panel 2: Admin Auth -->
        <div class="bg-white/80 backdrop-blur-md border border-slate-200/80 rounded-2xl p-6 shadow-sm hover:shadow-md transition-all">
          <h3 class="text-lg font-bold text-amber-600 border-b border-slate-100 pb-3 mb-5 flex items-center gap-2">
            🛡️ 系部管理员权限控制
          </h3>
          <el-form label-position="top">
            <el-form-item label="当前账号标识">
              <el-tag type="warning" effect="dark" size="large" class="font-bold px-3 py-1 rounded-lg">
                👤 {{ currentAdminName }}
              </el-tag>
            </el-form-item>
            <el-form-item label="重置本人密码" required>
              <el-input
                v-model="passwordForm.newPass"
                type="password"
                show-password
                placeholder="输入新安全密码 (至少6位)"
                class="!rounded-lg"
              />
            </el-form-item>
            <el-form-item label="确认新密码" required>
              <el-input
                v-model="passwordForm.confirmPass"
                type="password"
                show-password
                placeholder="再次输入新密码以确认"
                class="!rounded-lg"
              />
            </el-form-item>
            <div class="mt-4">
              <el-button
                type="warning"
                size="large"
                class="w-full !rounded-xl font-bold shadow-md shadow-amber-50 hover:scale-[1.01] transition-transform"
                :loading="savePasswordLoading"
                @click="updatePassword"
              >
                🔑 确认修改我的密码
              </el-button>
            </div>
          </el-form>
        </div>
      </el-col>

      <!-- Right Column: Appeal Categories Engine -->
      <el-col :span="13" class="h-full">
        <div class="bg-white/80 backdrop-blur-md border border-slate-200/80 rounded-2xl p-6 shadow-sm flex flex-col h-full min-h-[500px]">
          <div class="flex justify-between items-center border-b border-slate-100 pb-4 mb-4 shrink-0">
            <h3 class="text-lg font-bold text-blue-600 flex items-center gap-2">
              🗂️ 申诉大厅分类引擎
            </h3>
            <el-button
              size="small"
              type="primary"
              plain
              class="font-bold shadow-sm"
              @click="addCategory(0)"
            >
              ➕ 新增主类别
            </el-button>
          </div>

          <!-- Category Tree Scroll Area -->
          <div class="flex-1 overflow-y-auto pr-2 custom-scrollbar" v-loading="categoriesLoading">
            <el-tree
              :data="categoryTree"
              node-key="id"
              default-expand-all
              :expand-on-click-node="false"
              class="custom-tree"
            >
              <template #default="{ node, data }">
                <div class="custom-tree-node py-1 flex-1 flex items-center justify-between">
                  <span class="flex items-center gap-2">
                    <!-- Text color changes: parent blue, children slate-700 (fully legible contrast) -->
                    <b :style="{ color: data.parentId === 0 ? '#2563eb' : '#334155' }" class="text-sm font-semibold">
                      {{ node.label }}
                    </b>
                    <el-tag
                      size="small"
                      type="danger"
                      effect="dark"
                      v-if="data.requireImage"
                      class="ml-2 font-bold px-1.5 py-0.5 rounded-md text-[10px]"
                    >
                      必须传图
                    </el-tag>
                  </span>
                  <span class="tree-actions opacity-0 group-hover:opacity-100 transition-opacity flex gap-2">
                    <el-button
                      link
                      type="primary"
                      size="small"
                      class="font-bold hover:underline"
                      @click.stop="addCategory(data.id)"
                    >
                      ➕ 细分
                    </el-button>
                    <el-button
                      link
                      type="warning"
                      size="small"
                      class="font-bold hover:underline"
                      @click.stop="editCategory(data)"
                    >
                      📝 编辑
                    </el-button>
                    <el-button
                      link
                      type="danger"
                      size="small"
                      class="font-bold hover:underline"
                      @click.stop="deleteCategory(data.id)"
                    >
                      🗑️ 删除
                    </el-button>
                  </span>
                </div>
              </template>
            </el-tree>

            <!-- Empty State -->
            <div v-if="categoryTree.length === 0 && !categoriesLoading" class="py-12 text-center text-slate-400">
              <div class="text-4xl mb-2">📁</div>
              <p class="text-sm">暂无申诉分类，请点击上方按钮新增主分类</p>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- Appeal Category Config Dialog -->
    <el-dialog
      v-model="dialogCategory"
      title="⚙️ 申诉分类引擎配置"
      width="440px"
      class="rounded-xl overflow-hidden custom-dialog"
      align-center
    >
      <el-form label-width="120px" label-position="left" class="space-y-4">
        <el-form-item label="分类名称" required>
          <el-input
            v-model="catForm.name"
            placeholder="例如：账号异常、赛道错漏"
            class="rounded-lg"
          />
        </el-form-item>
        <el-form-item label="必须附带截图">
          <el-switch
            v-model="catForm.requireImage"
            :active-value="1"
            :inactive-value="0"
            active-text="强制上传"
            inactive-text="选填存证"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="flex justify-end gap-3 pt-2">
          <el-button @click="dialogCategory = false">取 消</el-button>
          <el-button
            type="primary"
            class="font-bold shadow-md shadow-blue-50"
            :loading="saveCategoryLoading"
            @click="saveCategory"
          >
            确 认
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'

// System settings interface definitions
interface SysSettings {
  deadline: string
  maxMoralScore: number
  maxEmpScore: number
  maxSingleAdd: number
  maxSingleSub: number
  trackEditDeadline: string
}

interface PasswordForm {
  newPass: string
  confirmPass: string
}

// Category API raw node format
interface ApiCategory {
  id: number
  parentId: number
  name: string
  requireImage: number
}

// Tree structured format
interface CategoryNode {
  id: number | null
  parentId: number
  label: string
  requireImage: number
  children?: CategoryNode[]
}

interface CategoryForm {
  id: number | null
  parentId: number
  name: string
  requireImage: number
}

// State hooks
const currentAdminName = ref<string>(localStorage.getItem('admin_name') || '系部管理员')

const settingsLoading = ref<boolean>(false)
const saveSettingsLoading = ref<boolean>(false)
const savePasswordLoading = ref<boolean>(false)
const categoriesLoading = ref<boolean>(false)
const saveCategoryLoading = ref<boolean>(false)

const sysSettings = ref<SysSettings>({
  deadline: '22:00',
  maxMoralScore: 60,
  maxEmpScore: 500,
  maxSingleAdd: 50,
  maxSingleSub: 50,
  trackEditDeadline: '2026-05-31 00:00:00'
})

const passwordForm = ref<PasswordForm>({
  newPass: '',
  confirmPass: ''
})

// Category state
const categoryTree = ref<CategoryNode[]>([])
const dialogCategory = ref<boolean>(false)
const catForm = ref<CategoryForm>({
  id: null,
  parentId: 0,
  name: '',
  requireImage: 0
})

// Load current core settings from API
const fetchSettings = async () => {
  settingsLoading.value = true
  try {
    const res: any = await request.get('/api/admin/settings/get')
    let d: any = null
    
    // 🛡️ 战术容错解包：兼容 Proxy 代理返回、原始 Axios 结构、及原生字段载荷
    if (res.data) {
      if (res.data.deadline !== undefined || res.data.max_moral_score !== undefined || res.data.max_emp_score !== undefined) {
        d = res.data
      } else if (res.data.data) {
        d = res.data.data
      }
    }
    if (!d && (res.deadline !== undefined || res.max_moral_score !== undefined)) {
      d = res
    }

    if (d) {
      if (d.deadline) sysSettings.value.deadline = d.deadline
      
      const maxMoral = d.max_moral_score || d.maxMoralScore
      if (maxMoral) sysSettings.value.maxMoralScore = parseInt(String(maxMoral))
      
      const maxEmp = d.max_emp_score || d.maxEmpScore
      if (maxEmp) sysSettings.value.maxEmpScore = parseInt(String(maxEmp))
      
      const maxAdd = d.max_single_add || d.maxSingleAdd
      if (maxAdd) sysSettings.value.maxSingleAdd = parseInt(String(maxAdd))
      
      const maxSub = d.max_single_sub || d.maxSingleSub
      if (maxSub) sysSettings.value.maxSingleSub = parseInt(String(maxSub))
      
      const trackEdit = d.track_edit_deadline || d.trackEditDeadline
      if (trackEdit) sysSettings.value.trackEditDeadline = trackEdit
    }
  } catch (error) {
    console.error('获取配置异常:', error)
    // Fallback internally if backend mock node is not fully initialized
    fillMockSettings()
  } finally {
    settingsLoading.value = false
  }
}

// Fallback loader to protect demonstration flows
const fillMockSettings = () => {
  sysSettings.value = {
    deadline: '22:00',
    maxMoralScore: 100,
    maxEmpScore: 1000,
    maxSingleAdd: 100,
    maxSingleSub: 100,
    trackEditDeadline: '2026-05-31 23:59:59'
  }
}

// Save core run parameters
const saveSettings = async () => {
  saveSettingsLoading.value = true
  try {
    await request.post('/api/admin/settings/save', {
      deadline: sysSettings.value.deadline,
      max_moral_score: String(sysSettings.value.maxMoralScore),
      max_emp_score: String(sysSettings.value.maxEmpScore),
      max_single_add: String(sysSettings.value.maxSingleAdd),
      max_single_sub: String(sysSettings.value.maxSingleSub),
      track_edit_deadline: sysSettings.value.trackEditDeadline
    })
    ElMessage.success("🎉 系统核心配置已实时生效")
  } catch (error) {
    ElMessage.error("保存失败，请检查网络或后端通信")
  } finally {
    saveSettingsLoading.value = false
  }
}

// Update password with security rules
const updatePassword = async () => {
  if (!passwordForm.value.newPass) {
    ElMessage.warning("密码不能为空")
    return
  }
  if (passwordForm.value.newPass.length < 6) {
    ElMessage.warning("新安全密码长度必须至少为 6 位！")
    return
  }
  if (passwordForm.value.newPass !== passwordForm.value.confirmPass) {
    ElMessage.error("两次输入密码不一致！")
    return
  }

  savePasswordLoading.value = true
  try {
    await request.post('/api/admin/auth/reset-self', {
      newPassword: passwordForm.value.newPass,
      adminId: Number(localStorage.getItem('admin_id') || 0)
    })
    ElMessage.success("密码修改成功，下次登录即生效")
    passwordForm.value = { newPass: '', confirmPass: '' }
  } catch (e) {
    ElMessage.error("密码修改失败，请重试")
  } finally {
    savePasswordLoading.value = false
  }
}

// Retrieve category tree values
const fetchCategories = async () => {
  categoriesLoading.value = true
  try {
    const res: any = await request.get('/api/admin/appeal/categories')
    let list: any = null
    
    // 🛡️ 智能兼容解包：判断 Proxy 代理下的各种数据包返回形式
    if (Array.isArray(res)) {
      list = res
    } else if (res.data && Array.isArray(res.data)) {
      list = res.data
    } else if (res.data && res.data.data && Array.isArray(res.data.data)) {
      list = res.data.data
    }
    
    if (list) {
      categoryTree.value = buildTree(list, 0)
    } else {
      categoryTree.value = []
    }
  } catch (e) {
    console.error('获取申诉分类异常:', e)
    categoryTree.value = []
  } finally {
    categoriesLoading.value = false
  }
}

// Tree building recursive mapping logic
const buildTree = (data: ApiCategory[], parentId: number): CategoryNode[] => {
  const tree: CategoryNode[] = []
  data.forEach((item: ApiCategory) => {
    if (item.parentId === parentId) {
      const children = buildTree(data, item.id)
      tree.push({
        id: item.id,
        parentId: item.parentId,
        label: item.name,
        requireImage: item.requireImage,
        children: children.length ? children : undefined
      })
    }
  })
  return tree
}

// Operations hooks
const addCategory = (parentId: number) => {
  catForm.value = {
    id: null,
    parentId,
    name: '',
    requireImage: 0
  }
  dialogCategory.value = true
}

const editCategory = (data: CategoryNode) => {
  catForm.value = {
    id: data.id,
    parentId: data.parentId,
    name: data.label,
    requireImage: data.requireImage
  }
  dialogCategory.value = true
}

const saveCategory = async () => {
  if (!catForm.value.name.trim()) {
    ElMessage.warning("分类名称不能为空")
    return
  }

  saveCategoryLoading.value = true
  try {
    await request.post('/api/admin/appeal/category/save', catForm.value)
    ElMessage.success("🎉 申诉分类引擎保存成功")
    dialogCategory.value = false
    await fetchCategories()
  } catch (error) {
    ElMessage.error("分类保存失败，请检查数据完整性")
  } finally {
    saveCategoryLoading.value = false
  }
}

const deleteCategory = async (id: number) => {
  ElMessageBox.confirm(
    '您确定要删除该分类及其下属所有子级细分类别吗？此操作将立即修改前端结构且不可逆！',
    '危险警告',
    {
      confirmButtonText: '确定删除',
      cancelButtonText: '取消',
      type: 'error',
      buttonSize: 'default'
    }
  ).then(async () => {
    try {
      await request.delete(`/api/admin/appeal/category/delete?id=${id}`)
      ElMessage.success("🗑️ 类别及其下属子项已被彻底抹除")
      await fetchCategories()
    } catch (error) {
      ElMessage.error("删除分类失败")
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchSettings()
  fetchCategories()
})
</script>

<style scoped>
.custom-tree :deep(.el-tree-node) {
  position: relative;
}
.custom-tree :deep(.el-tree-node__content) {
  height: 42px !important;
  border-radius: 8px;
  margin-bottom: 4px;
  padding-left: 8px !important;
  transition: all 0.25s ease;
}
.custom-tree :deep(.el-tree-node__content:hover) {
  background-color: #f1f5f9 !important;
}

.custom-tree-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.tree-actions {
  opacity: 0;
  transition: opacity 0.2s ease-in-out;
  padding-right: 8px;
}

:deep(.el-tree-node__content:hover) .tree-actions {
  opacity: 1;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 99px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
</style>
