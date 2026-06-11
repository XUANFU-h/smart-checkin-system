<template>
  <div class="p-6 max-w-[1600px] mx-auto bg-slate-50 min-h-screen flex flex-col gap-6 font-sans">
    
    <!-- 头部工具面板 -->
    <div class="bg-white/90 backdrop-blur-md rounded-2xl border border-slate-200/80 p-5 shadow-sm flex flex-wrap gap-4 items-center justify-between">
      <div class="flex items-center gap-3">
        <h1 class="text-xl font-black text-slate-800 tracking-tight flex items-center gap-2">
          {{ isDetailMode ? '🔍 赛道深度阅卷穿透中心' : '📊 赛道配置与打卡阅卷大盘' }}
        </h1>
        <el-tag v-if="isDetailMode" type="success" size="small" effect="dark" class="font-bold">
          {{ activeTrackName }}
        </el-tag>
      </div>

      <div class="flex items-center gap-4">
        <!-- 今日日期对齐 -->
        <div class="flex items-center gap-2">
          <span class="text-xs font-bold text-slate-500">考察日期</span>
          <el-date-picker 
            v-model="filterDate" 
            type="date" 
            placeholder="打卡日期" 
            format="YYYY-MM-DD" 
            value-format="YYYY-MM-DD" 
            @change="handleDateChange" 
            class="!w-40" 
            :clearable="false" 
          />
        </div>
        
        <!-- 大盘模式下显示管理配置入口 -->
        <el-button 
          v-if="!isDetailMode" 
          type="primary" 
          class="font-bold shadow-sm" 
          @click="openTrackConfigDrawer"
        >
          ⚙️ 赛道配置管理
        </el-button>

        <!-- 详情模式下显示返回按钮 -->
        <el-button 
          v-else 
          type="warning" 
          plain 
          class="font-bold" 
          @click="backToDashboard"
        >
          ⬅️ 返回大盘
        </el-button>
      </div>
    </div>

    <!-- 视图 A：2X2 响应式赛道仪表盘网格 -->
    <div v-if="!isDetailMode" class="grid grid-cols-1 xl:grid-cols-2 gap-6" v-loading="loading">
      <div 
        v-for="track in trackStatsList" 
        :key="track.trackId"
        @click="goToTrackDetail(track.trackId, track.trackName)"
        class="bg-white rounded-3xl border border-slate-200 p-6 shadow-sm flex flex-col gap-4 hover:shadow-xl hover:scale-[1.01] transition-all duration-300 cursor-pointer group relative overflow-hidden"
      >
        <!-- 装饰背景 -->
        <div class="absolute top-0 right-0 w-24 h-24 bg-gradient-to-bl from-blue-50/40 to-transparent rounded-bl-full pointer-events-none" />

        <div class="flex items-center justify-between">
          <div class="flex flex-col">
            <span class="text-base font-black text-slate-800 group-hover:text-blue-600 transition-colors">
              🏁 {{ track.trackName }}
            </span>
            <span class="text-[10px] text-slate-400 font-medium mt-0.5 line-clamp-1 max-w-[280px]">
              {{ track.description || '暂无描述' }}
            </span>
          </div>
          <el-tag size="small" type="info" class="font-bold">
            {{ track.requireType || 'image' }}
          </el-tag>
        </div>
        
        <!-- 双图形呈现引擎 -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 h-[180px] mt-2">
          <!-- 左侧：圆环图容器 -->
          <div :id="`doughnut-${track.trackId}`" class="w-full h-full" />
          <!-- 右侧：玫瑰图容器 -->
          <div :id="`rose-${track.trackId}`" class="w-full h-full" />
        </div>

        <div class="text-[10px] text-slate-400 font-bold text-center opacity-0 group-hover:opacity-100 transition-opacity mt-1">
          点击该卡片，进入【{{ track.trackName }}】深度穿透阅卷 🔍
        </div>
      </div>
    </div>

    <!-- 视图 B：深度穿透详情页面 -->
    <div v-else class="flex flex-col gap-6" v-loading="loading">
      <!-- 1. 遥测名单墙 (Telemetry Wall) -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- 左半侧：今日已提交名单 -->
        <div class="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm flex flex-col gap-3">
          <div class="flex items-center justify-between border-b border-slate-100 pb-3">
            <span class="text-sm font-black text-emerald-600 flex items-center gap-1.5">
              🟢 今日已提交人员名册 ({{ submissionStatus.submitted.length }}人)
            </span>
          </div>
          <el-table :data="submissionStatus.submitted" height="200" size="small" stripe>
            <el-table-column type="index" label="序号" width="55" align="center" />
            <el-table-column prop="realName" label="姓名" width="100">
              <template #default="{ row }">{{ row.nickname || row.realName }}</template>
            </el-table-column>
            <el-table-column prop="studentId" label="学号" width="130" />
            <el-table-column prop="className" label="所属班级" show-overflow-tooltip />
            <el-table-column prop="aiScore" label="打分" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.aiScore > 0 ? 'success' : 'info'" size="small" class="font-bold">
                  {{ row.aiScore > 0 ? `${row.aiScore} 分` : '未打分' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="createTime" label="打卡时间" width="160">
              <template #default="{ row }">{{ formatTime(row.createTime) }}</template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 右半侧：今日未提交名单 -->
        <div class="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm flex flex-col gap-3">
          <div class="flex items-center justify-between border-b border-slate-100 pb-3">
            <span class="text-sm font-black text-amber-600 flex items-center gap-1.5">
              🔴 今日未提交人员名单 ({{ submissionStatus.unsubmitted.length }}人)
            </span>
          </div>
          <el-table :data="submissionStatus.unsubmitted" height="200" size="small" stripe>
            <el-table-column type="index" label="序号" width="55" align="center" />
            <el-table-column prop="realName" label="姓名" width="100">
              <template #default="{ row }">{{ row.nickname || row.realName }}</template>
            </el-table-column>
            <el-table-column prop="studentId" label="学号" width="130" />
            <el-table-column prop="className" label="所属班级" show-overflow-tooltip />
            <el-table-column label="状态" width="100">
              <template #default>
                <el-tag type="danger" size="small" effect="plain" class="font-bold">未打卡</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- 2. 下方打卡足迹流筛选栏 -->
      <div class="bg-white rounded-2xl border border-slate-200 p-4 shadow-sm flex flex-wrap gap-4 items-center">
        <div class="flex items-center gap-2">
          <span class="text-xs font-bold text-slate-500">班级快速筛选</span>
          <el-select v-model="filterClass" placeholder="全系班级检索" clearable class="!w-48">
            <el-option v-for="c in classOptions" :key="c" :label="c" :value="c" />
          </el-select>
        </div>
        <el-button type="primary" plain size="small" @click="loadTrackDetailView" class="ml-auto">🔄 刷新阅卷列表</el-button>
      </div>

      <!-- 3. 全局打卡阅卷卡片大盘流 -->
      <div v-if="filteredRecords.length === 0" class="flex flex-col items-center justify-center py-24 bg-white rounded-2xl border border-slate-200 shadow-sm shrink-0">
        <el-empty description="该班级今日无任何打卡信息" />
      </div>

      <div v-else class="flex flex-col gap-8 shrink-0">
        <!-- 🎯 Section A: 待阅卷 / 未评分记录 -->
        <div class="flex flex-col gap-4">
          <h3 class="text-sm font-black text-amber-600 flex items-center gap-2 pb-2 border-b border-amber-100">
            📌 待阅卷 / 未评分记录 ({{ pendingRecords.length }} 份)
          </h3>
          <div v-if="pendingRecords.length === 0" class="py-12 text-center text-slate-400 bg-white rounded-2xl border border-slate-200 shadow-sm flex flex-col items-center justify-center">
            <span class="text-3xl mb-2">🎉</span>
            <p class="text-xs font-bold text-slate-500">今日暂无未阅卷记录，全部完成！</p>
          </div>
          <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            <div 
              v-for="row in pendingRecords" 
              :key="row.id" 
              class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden hover:shadow-lg transition-all duration-300 flex flex-col group hover:-translate-y-1"
            >
              <!-- 卡片头部：学生与学籍信息 -->
              <div class="p-4 border-b border-slate-100 bg-slate-50/50 flex items-center justify-between gap-2 shrink-0">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-8 h-8 rounded-full bg-slate-200 flex items-center justify-center font-bold text-slate-700 text-xs shrink-0">
                    {{ row.nickname ? row.nickname.substring(0, 1) : '学' }}
                  </div>
                  <div class="flex flex-col min-w-0">
                    <span class="font-bold text-slate-800 text-xs truncate">{{ row.nickname }}</span>
                    <span class="text-[10px] text-slate-400 font-mono truncate">{{ row.studentId || '未知学号' }}</span>
                  </div>
                </div>
                <span class="text-[10px] font-bold text-slate-500 shrink-0 truncate max-w-[120px]">
                  {{ row.className }}
                </span>
              </div>

              <!-- 打卡主体：图片/文档及留言 -->
              <div class="p-4 flex-1 flex flex-col gap-3">
                <div class="w-full aspect-[4/3] rounded-xl overflow-hidden bg-slate-100 border border-slate-100 shadow-inner relative group/img shrink-0">
                  <!-- Case A: Single Image -->
                  <template v-if="getFileList(row.imageUrl).length === 1 && !isDocumentFile(getFileList(row.imageUrl)[0])">
                    <el-image 
                      :src="getFileList(row.imageUrl)[0]" 
                      :preview-src-list="getFileList(row.imageUrl)" 
                      fit="cover" 
                      class="w-full h-full transition-transform duration-500 group-hover/img:scale-105" 
                      preview-teleported 
                    />
                  </template>
                  
                  <!-- Case B: Multiple Files / Documents -->
                  <template v-else-if="getFileList(row.imageUrl).length > 0">
                    <div class="w-full h-full overflow-y-auto p-3 flex flex-col gap-2 custom-scrollbar bg-slate-50/50">
                      <div v-for="(url, idx) in getFileList(row.imageUrl)" :key="idx" class="w-full shrink-0">
                        <!-- If it's a document -->
                        <div v-if="isDocumentFile(url)" class="flex items-center justify-between bg-white border border-slate-200 rounded-xl p-2.5 shadow-sm gap-2">
                          <span class="text-xl shrink-0">📄</span>
                          <span class="text-[10px] font-bold text-slate-700 truncate flex-1 min-w-0" :title="getFileName(url)">{{ getFileName(url) }}</span>
                          <el-link :href="url" target="_blank" type="primary" class="font-bold text-[10px] shrink-0 hover:underline" :underline="false">
                            📥 下载
                          </el-link>
                        </div>
                        <!-- If it's an image in a list -->
                        <div v-else class="w-full aspect-[16/9] rounded-xl overflow-hidden bg-slate-200 border border-slate-200 shadow-sm relative group/subimg">
                          <el-image 
                            :src="url" 
                            :preview-src-list="getFileList(row.imageUrl)" 
                            :initial-index="idx"
                            fit="cover" 
                            class="w-full h-full transition-transform duration-500 group-hover/subimg:scale-105" 
                            preview-teleported 
                          />
                        </div>
                      </div>
                    </div>
                  </template>

                  <!-- Case C: No Files -->
                  <template v-else>
                    <div class="w-full h-full flex flex-col items-center justify-center text-slate-300 gap-1">
                      <span class="text-2xl">🖼️</span>
                      <span class="text-[10px] font-bold">暂无凭证文件</span>
                    </div>
                  </template>
                </div>
                <div class="text-[11px] text-slate-600 bg-slate-50/50 p-2.5 rounded-xl border border-slate-100 leading-relaxed min-h-[50px] overflow-y-auto">
                  {{ row.contentText || '⚠️ 暂无留言描述' }}
                </div>
              </div>

              <!-- 导师评定栏 -->
              <div class="px-4 py-3 border-t border-slate-100 bg-slate-50/30 flex flex-col gap-2 shrink-0">
                <div class="flex items-center justify-between">
                  <span class="text-xs text-slate-500 font-bold">🎯 审定分值: 
                    <span v-if="row.status === 2" class="text-xs font-black text-rose-500 ml-1">0分 (已驳回)</span>
                    <span v-else-if="row.aiScore > 0" class="text-xs font-black text-emerald-600 ml-1">{{ row.aiScore }}分</span>
                    <span v-else class="text-xs font-bold text-slate-400 ml-1">未评分</span>
                  </span>
                  <el-tag :type="row.status === 2 ? 'danger' : (row.aiScore > 0 ? 'success' : 'warning')" size="small" :effect="row.status === 2 ? 'dark' : 'plain'" class="font-bold">
                    {{ row.status === 2 ? '已驳回' : (row.aiScore > 0 ? '已打分' : '待批阅') }}
                  </el-tag>
                </div>
                <div class="text-[10px] text-slate-400 line-clamp-2 italic bg-white p-2 rounded-lg border border-slate-100 min-h-[32px]">
                  " {{ row.adminComment || row.aiComment || '等待导师发布权威打分与评语...' }} "
                </div>
              </div>

              <!-- 操作按钮 -->
              <div class="p-3 border-t border-slate-100 bg-slate-50 flex items-center justify-between gap-2 shrink-0">
                <el-button size="small" type="danger" plain class="font-bold text-[10px]" @click="doDeleteRecord(row)">🗑️ 删除</el-button>
                <el-button size="small" :type="row.aiScore > 0 ? 'primary' : 'success'" class="font-bold text-[10px]" @click="openEvaluate(row)">
                  {{ row.aiScore > 0 ? '✏️ 修改打分' : '👨‍🏫 导师打分' }}
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <!-- ⚖️ 极具科技感的分界线 (Dashed Divider) -->
        <div class="relative py-4 shrink-0">
          <div class="absolute inset-0 flex items-center" aria-hidden="true">
            <div class="w-full border-t border-slate-200 border-dashed"></div>
          </div>
          <div class="relative flex justify-center text-xs">
            <span class="bg-slate-50 px-4 font-black text-slate-400 select-none tracking-wider">
              ⚖️ 上方未评分 Graded vs Ungraded 下方已评分 ⚖️
            </span>
          </div>
        </div>

        <!-- 🎯 Section B: 已审阅 / 已评分记录 -->
        <div class="flex flex-col gap-4">
          <h3 class="text-sm font-black text-emerald-600 flex items-center gap-2 pb-2 border-b border-emerald-100">
            ✅ 已审阅 / 已评分记录 ({{ processedRecords.length }} 份)
          </h3>
          <div v-if="processedRecords.length === 0" class="py-12 text-center text-slate-400 bg-white rounded-2xl border border-slate-200 shadow-sm flex flex-col items-center justify-center">
            <span class="text-3xl mb-2">⏳</span>
            <p class="text-xs font-bold text-slate-500">暂无已阅卷的数据名册</p>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            <div 
              v-for="row in processedRecords" 
              :key="row.id" 
              class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden hover:shadow-lg transition-all duration-300 flex flex-col group hover:-translate-y-1"
            >
              <!-- 卡片头部：学生与学籍信息 -->
              <div class="p-4 border-b border-slate-100 bg-slate-50/50 flex items-center justify-between gap-2 shrink-0">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-8 h-8 rounded-full bg-slate-200 flex items-center justify-center font-bold text-slate-700 text-xs shrink-0">
                    {{ row.nickname ? row.nickname.substring(0, 1) : '学' }}
                  </div>
                  <div class="flex flex-col min-w-0">
                    <span class="font-bold text-slate-800 text-xs truncate">{{ row.nickname }}</span>
                    <span class="text-[10px] text-slate-400 font-mono truncate">{{ row.studentId || '未知学号' }}</span>
                  </div>
                </div>
                <span class="text-[10px] font-bold text-slate-500 shrink-0 truncate max-w-[120px]">
                  {{ row.className }}
                </span>
              </div>

              <!-- 打卡主体：图片/文档及留言 -->
              <div class="p-4 flex-1 flex flex-col gap-3">
                <div class="w-full aspect-[4/3] rounded-xl overflow-hidden bg-slate-100 border border-slate-100 shadow-inner relative group/img shrink-0">
                  <!-- Case A: Single Image -->
                  <template v-if="getFileList(row.imageUrl).length === 1 && !isDocumentFile(getFileList(row.imageUrl)[0])">
                    <el-image 
                      :src="getFileList(row.imageUrl)[0]" 
                      :preview-src-list="getFileList(row.imageUrl)" 
                      fit="cover" 
                      class="w-full h-full transition-transform duration-500 group-hover/img:scale-105" 
                      preview-teleported 
                    />
                  </template>
                  
                  <!-- Case B: Multiple Files / Documents -->
                  <template v-else-if="getFileList(row.imageUrl).length > 0">
                    <div class="w-full h-full overflow-y-auto p-3 flex flex-col gap-2 custom-scrollbar bg-slate-50/50">
                      <div v-for="(url, idx) in getFileList(row.imageUrl)" :key="idx" class="w-full shrink-0">
                        <!-- If it's a document -->
                        <div v-if="isDocumentFile(url)" class="flex items-center justify-between bg-white border border-slate-200 rounded-xl p-2.5 shadow-sm gap-2">
                          <span class="text-xl shrink-0">📄</span>
                          <span class="text-[10px] font-bold text-slate-700 truncate flex-1 min-w-0" :title="getFileName(url)">{{ getFileName(url) }}</span>
                          <el-link :href="url" target="_blank" type="primary" class="font-bold text-[10px] shrink-0 hover:underline" :underline="false">
                            📥 下载
                          </el-link>
                        </div>
                        <!-- If it's an image in a list -->
                        <div v-else class="w-full aspect-[16/9] rounded-xl overflow-hidden bg-slate-200 border border-slate-200 shadow-sm relative group/subimg">
                          <el-image 
                            :src="url" 
                            :preview-src-list="getFileList(row.imageUrl)" 
                            :initial-index="idx"
                            fit="cover" 
                            class="w-full h-full transition-transform duration-500 group-hover/subimg:scale-105" 
                            preview-teleported 
                          />
                        </div>
                      </div>
                    </div>
                  </template>

                  <!-- Case C: No Files -->
                  <template v-else>
                    <div class="w-full h-full flex flex-col items-center justify-center text-slate-300 gap-1">
                      <span class="text-2xl">🖼️</span>
                      <span class="text-[10px] font-bold">暂无凭证文件</span>
                    </div>
                  </template>
                </div>
                <div class="text-[11px] text-slate-600 bg-slate-50/50 p-2.5 rounded-xl border border-slate-100 leading-relaxed min-h-[50px] overflow-y-auto">
                  {{ row.contentText || '⚠️ 暂无留言描述' }}
                </div>
              </div>

              <!-- 导师评定栏 -->
              <div class="px-4 py-3 border-t border-slate-100 bg-slate-50/30 flex flex-col gap-2 shrink-0">
                <div class="flex items-center justify-between">
                  <span class="text-xs text-slate-500 font-bold">🎯 审定分值: 
                    <span v-if="row.status === 2" class="text-xs font-black text-rose-500 ml-1">0分 (已驳回)</span>
                    <span v-else-if="row.aiScore > 0" class="text-xs font-black text-emerald-600 ml-1">{{ row.aiScore }}分</span>
                    <span v-else class="text-xs font-bold text-slate-400 ml-1">未评分</span>
                  </span>
                  <el-tag :type="row.status === 2 ? 'danger' : (row.aiScore > 0 ? 'success' : 'warning')" size="small" :effect="row.status === 2 ? 'dark' : 'plain'" class="font-bold">
                    {{ row.status === 2 ? '已驳回' : (row.aiScore > 0 ? '已打分' : '待批阅') }}
                  </el-tag>
                </div>
                <div class="text-[10px] text-slate-400 line-clamp-2 italic bg-white p-2 rounded-lg border border-slate-100 min-h-[32px]">
                  " {{ row.adminComment || row.aiComment || '等待导师发布权威打分与评语...' }} "
                </div>
              </div>

              <!-- 操作按钮 -->
              <div class="p-3 border-t border-slate-100 bg-slate-50 flex items-center justify-between gap-2 shrink-0">
                <el-button size="small" type="danger" plain class="font-bold text-[10px]" @click="doDeleteRecord(row)">🗑️ 删除</el-button>
                <el-button size="small" :type="row.aiScore > 0 ? 'primary' : 'success'" class="font-bold text-[10px]" @click="openEvaluate(row)">
                  {{ row.aiScore > 0 ? '✏️ 修改打分' : '👨‍🏫 导师打分' }}
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 弹窗 1：赛道配置管理弹窗 (Track CRU) -->
    <el-dialog v-model="dlgConfig" title="⚙️ 赛道原子级配置管理中枢" width="700px" rounded-2xl>
      <div class="flex flex-col gap-4">
        <div class="flex justify-between items-center pb-2 border-b border-slate-100">
          <span class="text-xs font-bold text-slate-500">赛道清单 (直接覆写物理表)</span>
          <el-button type="success" size="small" class="font-bold" @click="handleCreateTrack">➕ 增加新赛道</el-button>
        </div>
        
        <!-- 赛道列表展示 -->
        <el-table :data="trackOptions" size="small" border stripe max-height="250">
          <el-table-column prop="name" label="赛道名称" width="120" />
          <el-table-column prop="description" label="赛道描述" show-overflow-tooltip />
          <el-table-column prop="requireType" label="上传契约" width="160">
            <template #default="{ row }">
              <el-tag v-for="t in splitRequireType(row.requireType)" :key="t" size="small" class="mr-1 mt-1">{{ t }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'danger'" size="small">{{ row.isActive ? '活跃' : '停用' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="130" align="center">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="handleEditTrack(row)">编辑</el-button>
              <el-button type="danger" link size="small" @click="handleDeleteTrack(row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 嵌套赛道保存编辑表单 -->
        <el-card v-if="trackForm.visible" shadow="never" class="border border-slate-200 rounded-xl bg-slate-50/50 mt-2">
          <template #header>
            <div class="flex justify-between items-center">
              <span class="text-xs font-black">{{ trackForm.id ? '✏️ 编辑赛道属性' : '🌟 录入全新赛道' }}</span>
              <el-button type="info" link size="small" @click="trackForm.visible = false">收起</el-button>
            </div>
          </template>
          <el-form label-position="top" size="small">
            <el-form-item label="赛道名称" required>
              <el-input v-model="trackForm.name" placeholder="请输入赛道名（如：逐光运动）" />
            </el-form-item>
            <el-form-item label="赛道描述">
              <el-input type="textarea" v-model="trackForm.description" placeholder="请输入详细的赛道规则描述" />
            </el-form-item>
            <el-form-item label="多选上传契约对齐" required>
              <el-checkbox-group v-model="trackForm.requireTypes">
                <el-checkbox label="image">图文上传 (image)</el-checkbox>
                <el-checkbox label="wechat_sport">微信运动 (wechat_sport)</el-checkbox>
                <el-checkbox label="file">文档上传 (file)</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            <el-form-item label="是否激活运作">
              <el-switch v-model="trackForm.isActive" />
            </el-form-item>
            <div class="flex justify-end gap-2 mt-4">
              <el-button size="small" @click="trackForm.visible = false">取消</el-button>
              <el-button size="small" type="success" class="font-bold" @click="doSaveTrack">确认提交落库</el-button>
            </div>
          </el-form>
        </el-card>
      </div>
    </el-dialog>

    <!-- 弹窗 2：导师权威打分弹窗 -->
    <el-dialog v-model="dlgEval" title="👨‍🏫 导师权威批阅与内容订正" width="500px" rounded-2xl>
      <el-form label-position="top">
        <el-form-item label="重新核定分值">
          <el-input-number v-model="evalForm.score" :min="0" :max="100" class="!w-full" />
        </el-form-item>
        <el-form-item label="打卡留言文字校对与修正">
          <el-input type="textarea" v-model="evalForm.contentText" :rows="3" placeholder="在此可修正打卡学生的错别字..." />
        </el-form-item>
        <el-form-item label="导师亲笔评语">
          <el-input type="textarea" v-model="evalForm.comment" :rows="3" placeholder="请输入专业批注，此操作将即刻同步并发布..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlgEval = false">取消</el-button>
        <el-button type="danger" class="font-bold mr-2" @click="doRejectEvaluate">❌ 驳回打卡</el-button>
        <el-button type="success" class="font-bold" @click="doSaveEvaluate">签署批阅令</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'
import * as echarts from 'echarts'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const filterDate = ref('')
const filterClass = ref('')

// 视图模式判定：通过 URL 是否携带 track_id 判断
const isDetailMode = computed(() => !!route.query.track_id)
const activeTrackId = computed(() => Number(route.query.track_id || 0))
const activeTrackName = ref('')

// 数据集
const trackStatsList = ref<any[]>([])
const submissionStatus = ref<{ submitted: any[]; unsubmitted: any[] }>({ submitted: [], unsubmitted: [] })
const classOptions = ref<string[]>([])
const records = ref<any[]>([]) // 打卡足迹流
const trackOptions = ref<any[]>([]) // 全量赛道

// ECharts 实例池
const chartInstances = new Map<string, echarts.ECharts>()

// 弹窗表单状态
const dlgConfig = ref(false)
const trackForm = ref({
  visible: false,
  id: 0,
  name: '',
  description: '',
  requireTypes: [] as string[],
  isActive: true
})

const dlgEval = ref(false)
const evalForm = ref({ id: 0, score: 0, comment: '', contentText: '' })

// 实时班级检索过滤
const filteredRecords = computed(() => {
  if (!filterClass.value) return records.value
  return records.value.filter(r => r.className === filterClass.value)
})

// 🌟 将打卡足迹流物理割裂为：待批阅 (分数=0 且 status=0) 和 已阅卷 (分数>0 或 status=2)
const pendingRecords = computed(() => {
  return filteredRecords.value.filter(r => r.aiScore === 0 && r.status === 0)
})

const processedRecords = computed(() => {
  return filteredRecords.value.filter(r => r.aiScore > 0 || r.status === 2)
})

// 🌟 文件类型与下载预览极客适配组件函数
const isDocumentFile = (url: string) => {
  if (!url) return false
  const ext = url.split('.').pop()?.toLowerCase() || ''
  return ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'zip', 'rar', '7z', 'gz'].includes(ext)
}

const getFileName = (url: string) => {
  if (!url) return ''
  return decodeURIComponent(url.split('/').pop() || '')
}

const getFileList = (urlStr: string) => {
  if (!urlStr) return []
  return urlStr.split(',').filter(u => u.trim() !== '')
}

// 分离 RequireType 逗号串为数组以生成标签
const splitRequireType = (val: string) => {
  if (!val) return []
  return val.split(',')
}

// 侦听路由变化，以平滑切换视图
watch(() => route.query.track_id, (newTrackId) => {
  disposeAllCharts()
  if (newTrackId) {
    loadTrackDetailView()
  } else {
    loadDashboardView()
  }
})

// === 加载全量班级字典 (异步拉取在籍班级，杜绝白屏) ===
const loadClassOptions = async () => {
  try {
    const res: any = await request.get('/auth/classes')
    classOptions.value = res.data || []
  } catch (err) {
    console.error('加载在籍班级字典失败:', err)
  }
}

// === 大盘视图数据引擎 ===
const loadDashboardView = async () => {
  loading.value = true
  try {
    const res: any = await request.get(`/admin/track/stats?date=${filterDate.value}`)
    trackStatsList.value = res.data || []
    
    // 渲染所有 ECharts 图形容器
    nextTick(() => {
      initAllCharts()
    })
  } finally {
    loading.value = false
  }
}

// 初始化 2X2 所有 ECharts 图形
const initAllCharts = () => {
  disposeAllCharts()
  trackStatsList.value.forEach(t => {
    // 1. 初始化圆环图 (左侧)
    const dEl = document.getElementById(`doughnut-${t.trackId}`)
    if (dEl) {
      const dChart = echarts.init(dEl)
      dChart.setOption({
        tooltip: { trigger: 'item', formatter: '{b}: {c}人 ({d}%)' },
        series: [{
          name: '打卡状态',
          type: 'pie',
          radius: ['55%', '75%'],
          avoidLabelOverlap: false,
          label: {
            show: true,
            position: 'center',
            formatter: () => `参与\n${t.totalParticipants}人`,
            fontSize: 11,
            fontWeight: 'bold',
            color: '#475569'
          },
          labelLine: { show: false },
          data: [
            { value: t.submittedCount, name: '已打卡', itemStyle: { color: '#10B981' } },
            { value: t.unsubmittedCount, name: '未打卡', itemStyle: { color: '#EF4444' } }
          ]
        }]
      })
      chartInstances.set(`doughnut-${t.trackId}`, dChart)
    }

    // 2. 初始化南丁格尔玫瑰图 (右侧)
    const rEl = document.getElementById(`rose-${t.trackId}`)
    if (rEl) {
      const rChart = echarts.init(rEl)
      rChart.setOption({
        tooltip: { trigger: 'item', formatter: '{b}: {c}人' },
        series: [{
          name: '打卡质量',
          type: 'pie',
          radius: ['15%', '75%'],
          roseType: 'radius',
          label: { show: false },
          itemStyle: { borderRadius: 4 },
          data: [
            { value: t.roseData.score_gt_20, name: '20分以上', itemStyle: { color: '#1D4ED8' } },
            { value: t.roseData.score_15_20, name: '15-20分', itemStyle: { color: '#3B82F6' } },
            { value: t.roseData.score_10_15, name: '10-15分', itemStyle: { color: '#60A5FA' } },
            { value: t.roseData.score_5_10, name: '5-10分', itemStyle: { color: '#93C5FD' } },
            { value: t.roseData.score_lt_5, name: '5分以下', itemStyle: { color: '#CBD5E1' } },
            { value: t.roseData.passed, name: '已通过', itemStyle: { color: '#10B981' } },
            { value: t.roseData.rejected, name: '已驳回', itemStyle: { color: '#EF4444' } }
          ]
        }]
      })
      chartInstances.set(`rose-${t.trackId}`, rChart)
    }
  })
}

// 销毁并释放 ECharts 实例，防止堆内存泄露
const disposeAllCharts = () => {
  chartInstances.forEach(c => c.dispose())
  chartInstances.clear()
}

// === 详情视图数据引擎 ===
const loadTrackDetailView = async () => {
  loading.value = true
  try {
    // 1. 获取名单墙
    const statRes: any = await request.get(`/admin/track/submission-status?track_id=${activeTrackId.value}&date=${filterDate.value}`)
    submissionStatus.value = statRes.data || { submitted: [], unsubmitted: [] }

    // 2. 获取打卡足迹流数据 (全量展示，无视放榜 publish_status 拦截)
    const listRes: any = await request.get(`/admin/audit/list?date=${filterDate.value}&trackId=${activeTrackId.value}`)
    records.value = listRes.data || []
    
    // 3. 同时尝试补充赛道名称，防止越权或空路由跳转返回显示错误
    if (trackOptions.value.length === 0) {
      await loadTracksData()
    }
    const curTrack = trackOptions.value.find(t => t.id === activeTrackId.value)
    if (curTrack) {
      activeTrackName.value = curTrack.name
    }
  } finally {
    loading.value = false
  }
}

// === 路由与交互控制 ===
const handleDateChange = () => {
  if (isDetailMode.value) {
    loadTrackDetailView()
  } else {
    loadDashboardView()
  }
}

const goToTrackDetail = (trackId: number, name: string) => {
  activeTrackName.value = name
  router.push({ path: '/admin/audit', query: { track_id: trackId } })
}

const backToDashboard = () => {
  router.push({ path: '/admin/audit' })
}

// === 赛道管理数据拉取 ===
const loadTracksData = async () => {
  try {
    const res: any = await request.get('/auth/tracks')
    trackOptions.value = res.data || []
  } catch (err) {
    console.error('拉取赛道信息失败:', err)
  }
}

const openTrackConfigDrawer = async () => {
  dlgConfig.value = true
  await loadTracksData()
}

const handleCreateTrack = () => {
  trackForm.value = {
    visible: true,
    id: 0,
    name: '',
    description: '',
    requireTypes: ['image'],
    isActive: true
  }
}

const handleEditTrack = (row: any) => {
  trackForm.value = {
    visible: true,
    id: row.id,
    name: row.name,
    description: row.description,
    requireTypes: splitRequireType(row.requireType),
    isActive: row.isActive
  }
}

const doSaveTrack = async () => {
  if (!trackForm.value.name) return ElMessage.warning('请输入赛道名')
  if (trackForm.value.requireTypes.length === 0) return ElMessage.warning('请至少勾选一种上传契约方式')

  try {
    await request.post('/admin/track/save', {
      id: trackForm.value.id,
      name: trackForm.value.name,
      description: trackForm.value.description,
      requireTypes: trackForm.value.requireTypes,
      isActive: trackForm.value.isActive
    })
    ElMessage.success('赛道配置已更新')
    trackForm.value.visible = false
    await openTrackConfigDrawer() // 刷新配置中枢
    await loadDashboardView() // 刷新大盘
  } catch {}
}

const handleDeleteTrack = (id: number) => {
  ElMessageBox.confirm('废弃此赛道吗？该赛道下的全量报名与打卡数据将可能会失去关联关系，请务必谨慎！', '危险警示', { type: 'warning' }).then(async () => {
    await request.post('/admin/track/delete', { id })
    ElMessage.success('赛道已成功废弃移除')
    await openTrackConfigDrawer()
    await loadDashboardView()
  }).catch(() => {})
}

// === 导师打分与批阅逻辑 ===
const openEvaluate = (row: any) => {
  evalForm.value = {
    id: row.id,
    score: row.aiScore || 0,
    comment: row.adminComment || row.aiComment || '',
    contentText: row.contentText || ''
  }
  dlgEval.value = true
}

const doSaveEvaluate = async () => {
  try {
    await request.post('/admin/evaluate', {
      recordId: evalForm.value.id,
      score: evalForm.value.score,
      comment: evalForm.value.comment,
      contentText: evalForm.value.contentText,
      isReject: false
    })
    ElMessage.success('导师人工打分评语已落库，即刻秒级放榜对齐同步')
    dlgEval.value = false
    await loadTrackDetailView()
  } catch {}
}

const doRejectEvaluate = async () => {
  if (!evalForm.value.comment.trim()) {
    ElMessage.warning('驳回打卡必须填写导师亲笔评语（说明驳回原因）！')
    return
  }
  try {
    await request.post('/admin/evaluate', {
      recordId: evalForm.value.id,
      score: 0,
      comment: evalForm.value.comment,
      contentText: evalForm.value.contentText,
      isReject: true
    })
    ElMessage.error('该打卡记录已被驳回')
    dlgEval.value = false
    await loadTrackDetailView()
  } catch {}
}

const doDeleteRecord = (row: any) => {
  ElMessageBox.confirm('物理销毁这名同学的这条打卡数据吗？此行为不可逆！', '极度高危警示', { type: 'error' }).then(async () => {
    await request.post('/admin/audit/delete', { recordId: row.id })
    ElMessage.success('打卡流水已被安全销毁抹除')
    await loadTrackDetailView()
  }).catch(() => {})
}

// 基础辅助函数
const formatTime = (timeStr: string) => {
  if (!timeStr) return ''
  const d = new Date(timeStr)
  return `${d.getMonth() + 1}-${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

onMounted(async () => {
  // 静态刷新初始化系统当前日期 YYYY-MM-DD
  const today = new Date()
  filterDate.value = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
  
  // 异步加载在籍全量班级名册字典
  await loadClassOptions()

  if (isDetailMode.value) {
    await loadTrackDetailView()
  } else {
    await loadDashboardView()
  }

  // 监听窗口缩放事件以重置 ECharts 大小
  window.addEventListener('resize', () => {
    chartInstances.forEach(c => c.resize())
  })
})

onUnmounted(() => {
  disposeAllCharts()
})
</script>

<style scoped>
:deep(.el-table) {
  --el-table-header-bg-color: #f8fafc;
  --el-table-header-text-color: #475569;
  font-size: 11px;
}
</style>