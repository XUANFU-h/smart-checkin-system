<template>
  <div class="p-6 max-w-[1600px] mx-auto bg-slate-50 min-h-screen flex flex-col gap-6 font-sans animate-fade-in">
    <!-- Header Banner -->
    <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between pb-6 border-b border-slate-200 gap-4">
      <div>
        <h1 class="text-2xl font-bold text-slate-900 flex items-center gap-2">
          📢 社区推流与动态管�?        </h1>
        <p class="text-sm text-slate-500 mt-1">
          拖拽调整小程序端显示顺序，管理推流内容、发布官方动态、配置悬赏榜单与全服通知�?        </p>
      </div>
      <div class="flex items-center gap-3 shrink-0">
        <el-button type="primary" plain class="font-bold shadow-sm" @click="refreshAllData">
          🔄 刷新数据大盘
        </el-button>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <div class="bg-white rounded-xl border border-slate-200 p-1 shadow-sm shrink-0">
      <el-tabs v-model="activeTab" class="community-manage-tabs px-4">
        <el-tab-pane label="🗂�?社区推流大盘 (Kanban)" name="board" />
        <el-tab-pane label="🎣 筛选用户推�? name="push" />
        <el-tab-pane label="📝 发布官方动�? name="publish" />
        <el-tab-pane v-if="isL1OrDev" label="💰 悬赏任务�? name="bounty" />
        <el-tab-pane v-if="isL1OrDev" label="📢 全局弹窗广播" name="notice" />
      </el-tabs>
    </div>

    <!-- Tab Content Area -->
    <div class="flex-1 flex flex-col min-h-0">
      
      <!-- Tab 1: Kanban Board -->
      <div v-if="activeTab === 'board'" class="flex-1 flex flex-col min-h-0">
        <div class="flex-1 flex gap-6 overflow-x-auto pb-4 items-start select-none custom-scrollbar">
          
          <!-- Recommendation Column -->
          <div 
            class="flex-shrink-0 w-80 bg-slate-100/80 rounded-2xl border flex flex-col max-h-[calc(100vh-280px)] shadow-sm transition-all duration-200"
            :class="[
              dragoverActiveColumn === 'recommend' ? 'border-dashed border-2 border-emerald-500 bg-emerald-50/40 shadow-md scale-[1.01]' : 'border-slate-200',
              isDraggingActive ? 'opacity-90' : ''
            ]"
            @dragover.prevent="onColumnDragOver('recommend')"
            @dragleave="onColumnDragLeave"
            @drop="onColumnDrop('recommend')"
          >
            <div class="p-4 flex justify-between items-center bg-rose-50 border-b border-rose-100 rounded-t-2xl shrink-0">
              <span class="font-bold text-rose-700 flex items-center gap-1.5 text-sm">
                <span class="text-base">🔥</span> 全局推荐�?              </span>
              <el-tag size="small" type="danger" effect="dark" class="font-bold font-mono rounded-full">
                {{ recommendItems.length }}
              </el-tag>
            </div>
            
            <div class="flex-1 p-3 overflow-y-auto flex flex-col gap-3 custom-scrollbar min-h-[150px]">
              <div v-if="recommendItems.length === 0" class="h-24 border-2 border-dashed border-slate-350 rounded-xl flex items-center justify-center text-slate-400 text-xs italic">
                拖拽卡片至此
              </div>
              
              <div
                v-for="(item, index) in recommendItems"
                :key="item.uid"
                class="bg-white border rounded-xl p-3 cursor-grab hover:shadow-md hover:-translate-y-0.5 transition-all duration-200 relative group"
                :class="[
                  item.isPinned ? 'border-l-4 border-l-rose-500 border-rose-100' : 'border-slate-200',
                  item.type === 'official' ? 'border-l-4 border-l-blue-500 border-blue-100 bg-blue-50/20' : '',
                  dragoverActiveIndex === index && dragoverActiveColumn === 'recommend' ? 'border-t-4 border-t-emerald-400' : ''
                ]"
                draggable="true"
                @dragstart="onCardDragStart(item)"
                @dragend="onCardDragEnd"
                @dragover.prevent="onCardDragOver('recommend', index)"
                @drop.stop="onCardDrop('recommend', index)"
                @click="openEditDialog(item)"
              >
                <div class="flex gap-1.5 mb-2 flex-wrap">
                  <el-tag v-if="item.isPinned" type="danger" size="small" effect="dark" class="font-bold text-[10px]">置顶</el-tag>
                  <el-tag v-if="item.type === 'official'" type="primary" size="small" effect="plain" class="font-bold text-[10px]">官方</el-tag>
                </div>
                
                <div class="flex justify-between items-start gap-2">
                  <div class="flex-1 min-w-0">
                    <h4 class="text-xs font-bold text-slate-800 line-clamp-2 leading-relaxed mb-2">{{ item.title || '【无标题�? }}</h4>
                    <div class="flex items-center gap-1.5 text-[11px] text-slate-500 font-medium">
                      <img v-if="item.authorAvatar" :src="formatImg(item.authorAvatar)" class="w-4 h-4 rounded-full object-cover border border-slate-100 shrink-0" />
                      <span v-else-if="item.isAnonymous" class="shrink-0">🕵�?/span>
                      <span v-else class="shrink-0">✍️</span>
                      <span class="truncate">{{ item.isAnonymous ? '匿名用户' : item.author }}</span>
                    </div>
                  </div>
                  
                  <el-image 
                    v-if="item.imageUrl" 
                    :src="formatImg(item.imageUrl.split(',')[0])" 
                    class="w-12 h-12 rounded-lg border border-slate-100 shrink-0 object-cover shadow-sm"
                    fit="cover" 
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Dynamic Tracks Columns -->
          <div 
            v-for="track in trackList"
            :key="track.id"
            class="flex-shrink-0 w-80 bg-slate-100/80 rounded-2xl border flex flex-col max-h-[calc(100vh-280px)] shadow-sm transition-all duration-200"
            :class="[
              dragoverActiveColumn === String(track.id) ? 'border-dashed border-2 border-amber-500 bg-amber-50/40 shadow-md scale-[1.01]' : 'border-slate-200',
              isDraggingActive ? 'opacity-90' : ''
            ]"
            @dragover.prevent="onColumnDragOver(String(track.id))"
            @dragleave="onColumnDragLeave"
            @drop="onColumnDrop(String(track.id))"
          >
            <div class="p-4 flex justify-between items-center bg-amber-50 border-b border-amber-100 rounded-t-2xl shrink-0">
              <span class="font-bold text-amber-700 flex items-center gap-1.5 text-sm truncate">
                <span class="text-base text-amber-500">�?/span> {{ track.name }}
              </span>
              <el-tag size="small" type="warning" effect="dark" class="font-bold font-mono rounded-full shrink-0">
                {{ getTrackItems(track.id).length }}
              </el-tag>
            </div>
            
            <div class="flex-1 p-3 overflow-y-auto flex flex-col gap-3 custom-scrollbar min-h-[150px]">
              <div v-if="getTrackItems(track.id).length === 0" class="h-24 border-2 border-dashed border-slate-350 rounded-xl flex items-center justify-center text-slate-400 text-xs italic">
                拖拽卡片至此
              </div>
              
              <div
                v-for="(item, index) in getTrackItems(track.id)"
                :key="item.uid"
                class="bg-white border rounded-xl p-3 cursor-grab hover:shadow-md hover:-translate-y-0.5 transition-all duration-200 relative group"
                :class="[
                  item.isPinned ? 'border-l-4 border-l-amber-500 border-amber-100' : 'border-slate-200',
                  item.type === 'official' ? 'border-l-4 border-l-blue-500 border-blue-100 bg-blue-50/20' : '',
                  dragoverActiveIndex === index && dragoverActiveColumn === String(track.id) ? 'border-t-4 border-t-amber-400' : ''
                ]"
                draggable="true"
                @dragstart="onCardDragStart(item)"
                @dragend="onCardDragEnd"
                @dragover.prevent="onCardDragOver(String(track.id), index)"
                @drop.stop="onCardDrop(String(track.id), index)"
                @click="openEditDialog(item)"
              >
                <div class="flex gap-1.5 mb-2 flex-wrap">
                  <el-tag v-if="item.isPinned" type="danger" size="small" effect="dark" class="font-bold text-[10px]">置顶</el-tag>
                  <el-tag v-if="item.type === 'official'" type="primary" size="small" effect="plain" class="font-bold text-[10px]">官方</el-tag>
                </div>
                
                <div class="flex justify-between items-start gap-2">
                  <div class="flex-1 min-w-0">
                    <h4 class="text-xs font-bold text-slate-800 line-clamp-2 leading-relaxed mb-2">{{ item.title || '【无标题�? }}</h4>
                    <div class="flex items-center gap-1.5 text-[11px] text-slate-500 font-medium">
                      <img v-if="item.authorAvatar" :src="formatImg(item.authorAvatar)" class="w-4 h-4 rounded-full object-cover border border-slate-100 shrink-0" />
                      <span v-else-if="item.isAnonymous" class="shrink-0">🕵�?/span>
                      <span v-else class="shrink-0">✍️</span>
                      <span class="truncate">{{ item.isAnonymous ? '匿名用户' : item.author }}</span>
                    </div>
                  </div>
                  
                  <el-image 
                    v-if="item.imageUrl" 
                    :src="formatImg(item.imageUrl.split(',')[0])" 
                    class="w-12 h-12 rounded-lg border border-slate-100 shrink-0 object-cover shadow-sm"
                    fit="cover" 
                  />
                </div>
              </div>
            </div>
          </div>
          
        </div>
      </div>

      <!-- Tab 2: Push Audit Table -->
      <div v-if="activeTab === 'push'" class="flex-1 bg-white rounded-2xl border border-slate-200 p-5 shadow-sm overflow-hidden flex flex-col min-h-[500px]">
        <el-table 
          v-loading="loading" 
          :data="posts" 
          style="width: 100%; height: 100%;" 
          stripe 
          class="text-slate-700"
        >
          <el-table-column prop="userName" label="作�? width="120">
            <template #default="{ row }">
              <span class="font-bold text-slate-800">{{ row.userName }}</span>
            </template>
          </el-table-column>
          
          <el-table-column prop="trackName" label="所属赛�? width="140">
            <template #default="{ row }">
              <el-tag size="small" type="info" class="font-semibold">{{ row.trackName }}</el-tag>
            </template>
          </el-table-column>
          
          <el-table-column label="打卡图文内容" min-width="300">
            <template #default="{ row }">
              <div class="flex items-start gap-3 py-1">
                <el-image
                  v-if="row.imageUrl"
                  :src="formatImg(row.imageUrl.split(',')[0])"
                  style="width: 50px; height: 50px;"
                  class="rounded-lg shrink-0 border border-slate-100 shadow-sm cursor-pointer hover:scale-105 transition-transform"
                  :preview-src-list="row.imageUrl.split(',').map((i: string) => formatImg(i))"
                  preview-teleported
                  fit="cover"
                />
                <div class="text-xs text-slate-650 leading-relaxed line-clamp-3 text-justify mt-0.5">
                  {{ row.contentText || '【无文字说明�? }}
                </div>
              </div>
            </template>
          </el-table-column>
          
          <el-table-column label="推流与置顶状�? width="180" align="center">
            <template #default="{ row }">
              <div class="flex flex-col gap-1 items-center">
                <el-tag v-if="isTrue(row.isPinned)" type="danger" effect="dark" size="small" class="font-bold">🔝 全局置顶</el-tag>
                <el-tag v-if="isTrue(row.isPushed) && row.pushChannel === 'recommend'" type="danger" effect="plain" size="small" class="font-bold">🔥 全局推荐</el-tag>
                <el-tag v-else-if="isTrue(row.isPushed) && row.pushChannel === 'track'" type="warning" effect="plain" size="small" class="font-bold">�?赛道专属</el-tag>
                <el-tag v-else type="info" effect="plain" size="small" class="font-medium">未推�?/el-tag>
              </div>
            </template>
          </el-table-column>
          
          <el-table-column label="操作" width="240" fixed="right" align="center">
            <template #default="{ row }">
              <div class="flex gap-2 justify-center">
                <el-button v-if="!isTrue(row.isPushed)" type="success" size="small" class="font-bold shadow-sm" @click="openPushDialog(row)">
                  🚀 去推�?                </el-button>
                <el-button v-if="isL1OrDev" type="primary" size="small" plain class="font-bold" @click="togglePostPin(row)">
                  {{ isTrue(row.isPinned) ? '取消置顶' : '设为置顶' }}
                </el-button>
                <el-button v-if="isTrue(row.isPushed)" type="danger" size="small" plain class="font-bold" @click="cancelPush(row.id)">
                  撤销
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Tab 3: Publish Official Posts -->
      <div v-if="activeTab === 'publish'" class="flex-1 flex flex-col lg:flex-row gap-6 min-h-0">
        <!-- New Post Creator Form -->
        <div class="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm lg:w-[480px] shrink-0 flex flex-col gap-4">
          <h3 class="text-sm font-bold text-slate-800 border-b pb-3 mb-2 flex items-center gap-1.5">
            �?撰写新官方动�?          </h3>
          
          <el-form label-position="top" class="space-y-4 flex-1 overflow-y-auto pr-1 custom-scrollbar">
            <div class="flex gap-4 items-center">
              <el-form-item label="官方发帖头像" class="!mb-0">
                <el-upload
                  class="avatar-uploader-box"
                  action="#"
                  :show-file-list="false"
                  :http-request="(opts: any) => uploadAvatar(opts, officialForm)"
                >
                  <img v-if="officialForm.authorAvatar" :src="formatImg(officialForm.authorAvatar)" class="w-12 h-12 rounded-full object-cover border shadow-sm" />
                  <div v-else class="w-12 h-12 rounded-full border-2 border-dashed border-slate-300 flex items-center justify-center text-slate-450 hover:border-blue-500 hover:text-blue-500 transition-colors cursor-pointer text-xl font-bold">+</div>
                </el-upload>
              </el-form-item>
              
              <el-form-item label="发帖身份展示名称" class="flex-1 !mb-0">
                <el-input v-model="officialForm.authorName" placeholder="例如：学风建设委员会" class="rounded-lg" />
              </el-form-item>
            </div>
            
            <!-- History Avatars Box -->
            <div v-if="avatarHistory.length > 0" class="bg-slate-50 border border-slate-100 rounded-xl p-3 space-y-1.5">
              <div class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">快捷历史头像�?/div>
              <div class="flex flex-wrap gap-2">
                <img 
                  v-for="av in avatarHistory" 
                  :key="av" 
                  :src="formatImg(av)" 
                  @click="officialForm.authorAvatar = av" 
                  class="w-7 h-7 rounded-full object-cover border border-slate-200 hover:scale-110 active:scale-95 transition-all cursor-pointer"
                  title="点击直接使用" 
                />
              </div>
            </div>
            
            <el-form-item label="动态主�?(必填)" required>
              <el-input v-model="officialForm.topic" placeholder="例如：墨香伴读打卡倡议�? />
            </el-form-item>
            
            <div class="grid grid-cols-2 gap-4">
              <el-form-item label="展示位置">
                <el-radio-group v-model="officialForm.channel" class="w-full flex gap-3">
                  <el-radio-button value="recommend" class="flex-1 text-center font-bold">🔥 全局</el-radio-button>
                  <el-radio-button value="track" class="flex-1 text-center font-bold">�?赛道</el-radio-button>
                </el-radio-group>
              </el-form-item>
              
              <el-form-item label="全局置顶">
                <div class="h-10 flex items-center">
                  <el-switch v-model="officialForm.isPinned" :active-value="1" :inactive-value="0" />
                </div>
              </el-form-item>
            </div>
            
            <transition name="el-zoom-in-top">
              <el-form-item label="关联目标赛道" v-if="officialForm.channel === 'track'" required>
                <el-select v-model="officialForm.trackId" placeholder="请选择关联赛道" class="w-full">
                  <el-option v-for="t in trackList" :key="t.id" :label="t.name" :value="t.id" />
                </el-select>
              </el-form-item>
            </transition>
            
            <el-form-item label="展示配图 (可�?">
              <el-upload
                class="image-uploader-box w-full"
                action="#"
                :show-file-list="false"
                :http-request="uploadNewPostImg"
              >
                <div v-if="officialForm.imageUrl" class="relative group w-full h-32 rounded-xl overflow-hidden border">
                  <img :src="formatImg(officialForm.imageUrl)" class="w-full h-full object-cover" />
                  <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white text-xs font-bold">点击替换图片</div>
                </div>
                <div v-else class="w-full h-28 border-2 border-dashed border-slate-300 bg-slate-50 hover:bg-slate-100/50 hover:border-blue-500 transition-all rounded-xl flex flex-col items-center justify-center gap-1.5 cursor-pointer text-slate-400">
                  <span class="text-2xl font-bold">+</span>
                  <span class="text-xs">上传官方精美配图</span>
                </div>
              </el-upload>
            </el-form-item>
            
            <el-form-item label="正文内容" required>
              <el-input type="textarea" :rows="5" v-model="officialForm.content" placeholder="输入你想发布的详细内�?.." />
            </el-form-item>
            
            <el-button type="primary" class="w-full h-10 font-bold shadow-md shadow-blue-50" @click="publishOfficialPost">
              🚀 立即发布并展示在看板
            </el-button>
          </el-form>
        </div>
        
        <!-- Published Posts History Audit -->
        <div class="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm flex-1 flex flex-col min-h-[500px]">
          <h3 class="text-sm font-bold text-slate-800 border-b pb-3 mb-4 flex items-center gap-1.5">
            📋 已发布的官方动态流�?          </h3>
          
          <div v-if="officialPosts.length === 0" class="flex-1 flex flex-col items-center justify-center text-slate-450 gap-2">
            <span class="text-4xl">📋</span>
            <p class="text-xs italic">暂无发布的官方动态记�?/p>
          </div>
          
          <div v-else class="flex-1 overflow-y-auto space-y-4 pr-1 custom-scrollbar">
            <div 
              v-for="post in officialPosts" 
              :key="post.id" 
              class="border border-slate-150 rounded-xl p-4 bg-slate-50/50 hover:border-blue-300 hover:shadow-md transition-all duration-200"
            >
              <div class="flex justify-between items-start gap-4 mb-2">
                <div class="flex items-center gap-2">
                  <el-tag v-if="isTrue(post.isPinned)" type="danger" size="small" effect="dark" class="font-bold">置顶</el-tag>
                  <span class="font-bold text-slate-800 text-sm">{{ post.topic }}</span>
                </div>
                
                <el-button 
                  type="danger" 
                  link 
                  class="font-bold text-xs hover:text-red-700 shrink-0" 
                  @click="deleteOfficialPost(post.id)"
                >
                  🗑�?物理删除
                </el-button>
              </div>
              
              <p class="text-xs text-slate-600 leading-relaxed text-justify mb-3 bg-white border border-slate-100 rounded-lg p-2.5">
                {{ post.content }}
              </p>
              
              <div class="flex flex-wrap justify-between items-center text-[10px] text-slate-400 font-semibold gap-2 border-t border-slate-100 pt-2">
                <div class="flex items-center gap-1.5">
                  <img v-if="post.authorAvatar" :src="formatImg(post.authorAvatar)" class="w-4 h-4 rounded-full object-cover shrink-0" />
                  <span>发布�? <b class="text-blue-600 font-bold">{{ post.authorName }}</b></span>
                  <span class="text-slate-300">|</span>
                  <span>展示版块: <b class="text-slate-600 font-bold">{{ post.channel === 'recommend' ? '全局推荐' : '赛道专属' }}</b></span>
                </div>
                
                <div class="font-mono text-slate-400">{{ post.createTime }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab 4: Bounty Task Board -->
      <div v-if="activeTab === 'bounty' && isL1OrDev" class="flex-1 bg-white rounded-2xl border border-slate-200 p-5 shadow-sm overflow-hidden flex flex-col min-h-[500px]">
        <div class="flex flex-wrap justify-between items-center border-b pb-4 mb-5 gap-3 shrink-0">
          <div class="flex items-center gap-2">
            <el-button type="warning" class="font-bold shadow-md shadow-amber-50" @click="openBountyDialog">
              💰 发布新悬�?            </el-button>
            <span class="text-xs text-slate-450 font-medium">
              💡 提示：按住左�?�?即可上下拖动排序。置顶悬赏不受排序影响，始终置顶�?            </span>
          </div>
        </div>
        
        <div class="flex-1 flex flex-col border border-slate-200 rounded-xl overflow-hidden min-h-0 bg-slate-50/50">
          <!-- Bounty Header -->
          <div class="flex items-center px-4 py-3 bg-slate-100 border-b border-slate-200 text-xs font-bold text-slate-450 tracking-wider">
            <div class="w-12 text-center">排序</div>
            <div class="flex-1 pl-4">任务描述</div>
            <div class="w-40 pl-4">奖励分�?文案</div>
            <div class="w-60 pl-4">倒计�?时限</div>
            <div class="w-48 text-center">系统指令</div>
          </div>
          
          <!-- Bounty List (Drag & Drop) -->
          <div class="flex-1 overflow-y-auto custom-scrollbar p-2 space-y-2 select-none min-h-0">
            <div v-if="bounties.length === 0" class="h-32 flex flex-col items-center justify-center text-slate-400 gap-2">
              <span class="text-3xl">💰</span>
              <p class="text-xs italic">当前暂无发布的悬赏榜�?/p>
            </div>
            
            <div 
              v-else
              v-for="(item, index) in bounties" 
              :key="item.id"
              class="flex items-center px-4 py-3.5 bg-white border border-slate-150 rounded-lg shadow-sm hover:shadow transition-all duration-200"
              :class="[
                item.isPinned ? 'border-l-4 border-l-rose-500 bg-rose-50/10 border-rose-100' : 'border-l border-l-slate-250',
                dragStartIndexBounty === index ? 'opacity-40 border-dashed border-blue-400 scale-[0.99]' : ''
              ]"
              draggable="true"
              @dragstart="onBountyDragStart(index)"
              @dragover.prevent
              @drop="onBountyDrop(index)"
            >
              <!-- Drag Handle -->
              <div class="w-12 flex justify-center text-slate-400 text-lg hover:text-blue-500 cursor-grab active:cursor-grabbing">
                �?              </div>
              
              <!-- Title -->
              <div class="flex-1 pl-4 font-bold text-slate-800 text-sm flex items-center gap-2 truncate">
                <el-tag v-if="item.isPinned" type="danger" size="small" effect="dark" class="font-bold text-[10px]">置顶</el-tag>
                <span class="truncate">{{ item.title }}</span>
              </div>
              
              <!-- Reward Text -->
              <div class="w-40 pl-4 text-emerald-600 font-extrabold text-sm truncate">
                {{ item.rewardText }}
              </div>
              
              <!-- Countdown Timers -->
              <div class="w-60 pl-4 text-xs font-semibold">
                <span v-if="!item.endTime" class="text-slate-400 italic">永久有效</span>
                <span 
                  v-else 
                  :class="item.countdown === '已结�? ? 'text-rose-500 font-bold' : 'text-amber-600 font-bold font-mono'"
                >
                  �?{{ item.countdown }}
                </span>
              </div>
              
              <!-- Actions -->
              <div class="w-48 flex justify-center gap-2">
                <el-button 
                  size="small" 
                  :type="item.isPinned ? 'info' : 'primary'" 
                  class="font-bold" 
                  @click="toggleBountyPin(item.id, item.isPinned)"
                >
                  {{ item.isPinned ? '取消置顶' : '置顶' }}
                </el-button>
                <el-button 
                  type="danger" 
                  size="small" 
                  plain 
                  class="font-bold" 
                  @click="deleteBounty(item.id)"
                >
                  下架
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab 5: Global Broadcast Notice -->
      <div v-if="activeTab === 'notice' && isL1OrDev" class="flex-1 flex justify-center items-start min-h-[500px]">
        <div class="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm w-full max-w-2xl flex flex-col gap-5">
          <div class="border-b pb-3 mb-1">
            <h3 class="text-sm font-bold text-slate-800">📢 全服强弹窗广播配�?/h3>
            <p class="text-xs text-slate-450 mt-1">设置后，所有用户一经登�?打开小程序即可首屏强弹出该标题与正文通知�?/p>
          </div>
          
          <el-form label-position="top" class="space-y-4">
            <el-form-item label="弹窗强推广播状�?>
              <div class="bg-slate-50 border border-slate-200 rounded-xl p-3.5 flex items-center justify-between w-full shadow-inner">
                <span class="text-xs font-semibold text-slate-600">全员登录首屏强制弹窗广播</span>
                <el-switch 
                  v-model="noticeForm.isActive" 
                  :active-value="1" 
                  :inactive-value="0" 
                  active-text="激活中 (全员上线强制弹出)" 
                  inactive-text="已静默下�? 
                  @change="toggleNotice"
                />
              </div>
            </el-form-item>
            
            <el-form-item label="广播醒目标题" required>
              <el-input v-model="noticeForm.title" placeholder="例如：学风建设月重要通知" class="rounded-lg" />
            </el-form-item>
            
            <el-form-item label="广播详细条款正文" required>
              <el-input type="textarea" :rows="8" v-model="noticeForm.content" placeholder="请输入通知正文详细条款..." class="rounded-lg" />
            </el-form-item>
            
            <el-button type="danger" class="w-full h-10 font-bold shadow-md shadow-rose-50" @click="saveNotice">
              💾 保存并分发全局强弹�?            </el-button>
          </el-form>
        </div>
      </div>
      
    </div>

    <!-- Dialog 1: Deep Card Editor -->
    <el-dialog v-model="editDialogVisible" title="🎨 深度编辑推流内容" width="620px" class="rounded-xl overflow-hidden custom-dialog" align-center>
      <el-form label-position="top" class="space-y-4">
        <el-form-item label="前端展示标题 (推流专用)">
          <el-input v-model="editForm.title" placeholder="为这段内容起个引人瞩目的新标�? />
        </el-form-item>
        
        <el-form-item label="发帖人马甲头�?(覆盖原头�?">
          <div class="flex flex-col gap-3">
            <div class="flex items-center gap-3">
              <el-upload
                class="avatar-uploader-box"
                action="#"
                :show-file-list="false"
                :http-request="(opts: any) => uploadAvatar(opts, editForm)"
              >
                <img v-if="editForm.authorAvatar" :src="formatImg(editForm.authorAvatar)" class="w-10 h-10 rounded-full object-cover border shadow-sm" />
                <div v-else class="w-10 h-10 rounded-full border-2 border-dashed border-slate-300 flex items-center justify-center text-slate-400 hover:border-blue-500 hover:text-blue-500 transition-colors cursor-pointer text-lg font-bold">+</div>
              </el-upload>
              <span class="text-xs text-slate-400">点击头像框可直接选择本地精美图片替换</span>
            </div>
            
            <div v-if="avatarHistory.length > 0" class="bg-slate-50 border border-slate-100 rounded-xl p-2.5 space-y-1.5">
              <div class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">历史头像库快捷选择</div>
              <div class="flex flex-wrap gap-2">
                <img 
                  v-for="av in avatarHistory" 
                  :key="av" 
                  :src="formatImg(av)" 
                  @click="editForm.authorAvatar = av" 
                  class="w-6.5 h-6.5 rounded-full object-cover border border-slate-200 hover:scale-110 transition-transform cursor-pointer"
                />
              </div>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="详细正文内容">
          <el-input type="textarea" :rows="4" v-model="editForm.content" placeholder="可在此直接微调错别字或调整排�? />
        </el-form-item>
        
        <el-form-item label="展示配图 (可自由替�?">
          <el-upload
            class="image-uploader-box w-full"
            action="#"
            :show-file-list="false"
            :http-request="uploadEditPostImg"
          >
            <div v-if="editForm.imageUrl" class="relative group w-full h-32 rounded-xl overflow-hidden border">
              <img :src="formatImg(editForm.imageUrl)" class="w-full h-full object-cover" />
              <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white text-xs font-bold">点击更换大图</div>
            </div>
            <div v-else class="w-full h-28 border-2 border-dashed border-slate-300 bg-slate-50 hover:bg-slate-100/50 hover:border-blue-500 transition-all rounded-xl flex flex-col items-center justify-center gap-1.5 cursor-pointer text-slate-400">
              <span class="text-2xl font-bold">+</span>
              <span class="text-xs">上传并附加图片展�?/span>
            </div>
          </el-upload>
        </el-form-item>
        
        <div class="grid grid-cols-2 gap-4 bg-slate-50 border border-slate-200 rounded-xl p-4 shadow-inner">
          <el-form-item label="隐藏姓名保护" class="!mb-0" v-if="editForm.type === 'checkin'">
            <div class="h-10 flex items-center">
              <el-switch v-model="editForm.isAnonymous" :active-value="1" :inactive-value="0" active-text="隐藏学号姓名(覆盖头像)" />
            </div>
          </el-form-item>
          
          <el-form-item label="全局推荐置顶" class="!mb-0">
            <div class="h-10 flex items-center">
              <el-switch v-model="editForm.isPinned" :active-value="1" :inactive-value="0" active-text="推荐位优先置顶展�? />
            </div>
          </el-form-item>
        </div>
        
        <el-form-item label="重新配置发布与展示赛�?>
          <div class="flex flex-col gap-2.5 w-full">
            <el-radio-group v-model="editForm.channel" class="flex gap-4">
              <el-radio value="recommend" label="recommend" border class="!mr-0 flex-1 text-center font-bold">🔥 全局推荐</el-radio>
              <el-radio value="track" label="track" border class="!mr-0 flex-1 text-center font-bold">�?专属赛道</el-radio>
            </el-radio-group>
            
            <transition name="el-zoom-in-top">
              <el-select v-if="editForm.channel === 'track'" v-model="editForm.trackId" placeholder="选择关联的具体赛�? class="w-full">
                <el-option v-for="t in trackList" :key="t.id" :label="t.name" :value="t.id" />
              </el-select>
            </transition>
          </div>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="flex justify-between items-center w-full pt-2">
          <el-button type="danger" plain class="font-bold shadow-sm" @click="removePush">
            🗑�?移出大盘 (撤销显示)
          </el-button>
          
          <div class="flex gap-2">
            <el-button @click="editDialogVisible = false">取消</el-button>
            <el-button type="primary" class="font-bold shadow-md shadow-blue-50" @click="saveEdit">
              💾 保存修改并同�?            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- Dialog 2: Push Action Editor -->
    <el-dialog v-model="pushDialogVisible" title="🚀 设置推流参数" width="560px" class="rounded-xl overflow-hidden custom-dialog" align-center>
      <div v-if="pushDialogData" class="mb-4 p-3 bg-slate-50 border border-slate-200 rounded-lg flex items-center justify-between text-xs font-semibold">
        <span class="text-slate-500">正在推送：</span>
        <div class="flex items-center gap-1">
          <span class="text-blue-600 font-extrabold">{{ pushDialogData.userName }}</span>
          <span class="text-slate-400 font-medium">�?/span>
          <span class="text-slate-700 font-extrabold">[{{ pushDialogData.trackName }}]</span>
          <span class="text-slate-400 font-medium">的优质打�?/span>
        </div>
      </div>
      
      <el-form label-position="top" class="space-y-4">
        <el-form-item label="自定义替换展示头�?(留空则用打卡人原头像)">
          <div class="flex flex-col gap-3">
            <div class="flex items-center gap-3">
              <el-upload
                class="avatar-uploader-box"
                action="#"
                :show-file-list="false"
                :http-request="(opts: any) => uploadAvatar(opts, pushForm)"
              >
                <img v-if="pushForm.authorAvatar" :src="formatImg(pushForm.authorAvatar)" class="w-10 h-10 rounded-full object-cover border shadow-sm" />
                <div v-else class="w-10 h-10 rounded-full border-2 border-dashed border-slate-300 flex items-center justify-center text-slate-400 hover:border-blue-500 hover:text-blue-500 transition-colors cursor-pointer text-lg font-bold">+</div>
              </el-upload>
              <span class="text-xs text-slate-400">上传后会在前台临时展示该马甲头像</span>
            </div>
            
            <div v-if="avatarHistory.length > 0" class="bg-slate-50 border border-slate-100 rounded-xl p-2.5 space-y-1.5">
              <div class="text-[10px] font-bold text-slate-400 uppercase tracking-wider">历史头像快捷选择</div>
              <div class="flex flex-wrap gap-2">
                <img 
                  v-for="av in avatarHistory" 
                  :key="av" 
                  :src="formatImg(av)" 
                  @click="pushForm.authorAvatar = av" 
                  class="w-6.5 h-6.5 rounded-full object-cover border border-slate-200 hover:scale-115 transition-transform cursor-pointer"
                />
              </div>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="推流目标分发位置" required>
          <el-radio-group v-model="pushForm.channel" class="w-full flex gap-3">
            <el-radio value="recommend" label="recommend" border class="!mr-0 flex-1 text-center font-bold">🔥 全局推荐�?/el-radio>
            <el-radio value="track" label="track" border class="!mr-0 flex-1 text-center font-bold">�?赛道专属�?/el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item label="官方权威置顶神评" required>
          <el-input type="textarea" :rows="3" v-model="pushForm.adminComment" placeholder="例如：打卡质量卓越，内容真实有条理，官方特推�? />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="flex justify-end gap-2 pt-2">
          <el-button @click="pushDialogVisible = false">取消</el-button>
          <el-button type="success" class="font-bold shadow-md shadow-emerald-50" @click="submitPush">
            确认推流上线
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Dialog 3: Bounty Publisher -->
    <el-dialog v-model="bountyDialogVisible" title="💰 发布新悬�? width="460px" class="rounded-xl overflow-hidden custom-dialog" align-center>
      <el-form label-position="top" class="space-y-4">
        <el-form-item label="任务描述 (必填)" required>
          <el-input v-model="bountyForm.title" placeholder="例如：今日打卡满 3 个赛�?/ 第一名打�? />
        </el-form-item>
        
        <el-form-item label="奖励文案 (必填)" required>
          <el-input v-model="bountyForm.rewardText" placeholder="例如：奖�?20 赋能积分 / 补签�?1 �? />
        </el-form-item>
        
        <el-form-item label="限时倒计时截止时�?(选填)">
          <el-date-picker 
            v-model="bountyForm.endTime" 
            type="datetime" 
            placeholder="留空则永久有效，不设定限�? 
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss" 
            class="!w-full" 
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="flex justify-end gap-2 pt-2">
          <el-button @click="bountyDialogVisible = false">取消</el-button>
          <el-button type="warning" class="font-bold shadow-md shadow-amber-50" @click="submitBounty">
            立刻发布上榜
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/api/request.ts'

// ==================== 📋 TypeScript Interfaces ====================
interface TrackItem {
  id: number | string
  name: string
  [key: string]: any
}

interface BoardItem {
  uid: string
  id: number | string
  type: 'checkin' | 'official'
  title: string
  content: string
  imageUrl?: string
  author: string
  authorAvatar?: string
  isAnonymous: number | boolean
  channel: 'recommend' | 'track'
  trackId: number | string | null
  isPinned: number | boolean
  sortOrder: number
}

interface PostItem {
  id: number | string
  userName: string
  trackName: string
  contentText: string
  imageUrl?: string
  isPinned: number | boolean
  isPushed: number | boolean
  pushChannel?: 'recommend' | 'track'
  pushTitle?: string
  pushContent?: string
  avatarUrl?: string
  pushAuthorAvatar?: string
  isAnonymous?: number | boolean
  trackId?: number | string
  status?: number
  aiComment?: string
  pushComment?: string
  sortOrder?: number | string
}

interface OfficialPostItem {
  id: number | string
  topic: string
  content: string
  imageUrl?: string
  authorName: string
  authorAvatar?: string
  channel: 'recommend' | 'track'
  trackId: number | string | null
  isPinned: number | boolean
  sortOrder: number
  createTime: string
}

interface OfficialForm {
  authorName: string
  authorAvatar: string
  topic: string
  content: string
  imageUrl: string
  channel: 'recommend' | 'track'
  trackId: number | string | null
  isPinned: number
}

interface PushForm {
  channel: 'recommend' | 'track'
  adminComment: string
  authorAvatar: string
}

interface BountyItem {
  id: number | string
  title: string
  rewardText: string
  endTime?: string
  isPinned: number | boolean
  sortOrder: number
  countdown?: string | null
}

interface BountyForm {
  title: string
  rewardText: string
  endTime: string
  sortOrder: number
}

interface NoticeForm {
  title: string
  content: string
  isActive: number
}

// ==================== 🛠�?State Variables ====================
const activeTab = ref<string>('board')
const loading = ref<boolean>(false)

// Safe Local Storage role mapping & permission checkers
const rawRole = localStorage.getItem('admin_role') || ''
const isL1OrDev = computed<boolean>(() => rawRole === 'DEV' || rawRole === 'ADMIN_L1')

// Avatar History Box
const avatarHistory = ref<string[]>(JSON.parse(localStorage.getItem('avatar_history') || '[]'))

// Dynamic Data Pools
const trackList = ref<TrackItem[]>([])
const boardData = ref<BoardItem[]>([])
const posts = ref<PostItem[]>([])
const officialPosts = ref<OfficialPostItem[]>([])
const bounties = ref<BountyItem[]>([])

// Drag Visual States
const draggedItem = ref<BoardItem | null>(null)
const isDraggingActive = ref<boolean>(false)
const dragoverActiveColumn = ref<string>('')
const dragoverActiveIndex = ref<number>(-1)
const dragStartIndexBounty = ref<number>(-1)

// Countdown timer reference
let bountyCountdownTimer: any = null

// Form Definitions & Dialog toggles
const officialForm = ref<OfficialForm>({
  authorName: localStorage.getItem('admin_name') || '官方小助�?,
  authorAvatar: '',
  topic: '',
  content: '',
  imageUrl: '',
  channel: 'recommend',
  trackId: null,
  isPinned: 0
})

const editDialogVisible = ref<boolean>(false)
const editForm = ref<BoardItem>({
  uid: '',
  id: '',
  type: 'checkin',
  title: '',
  content: '',
  imageUrl: '',
  author: '',
  authorAvatar: '',
  isAnonymous: 0,
  channel: 'recommend',
  trackId: null,
  isPinned: 0,
  sortOrder: 0
})

const pushDialogVisible = ref<boolean>(false)
const pushDialogData = ref<PostItem | null>(null)
const pushForm = ref<PushForm>({
  channel: 'recommend',
  adminComment: '打卡非常优秀，官方置顶特推！',
  authorAvatar: ''
})

const bountyDialogVisible = ref<boolean>(false)
const bountyForm = ref<BountyForm>({
  title: '',
  rewardText: '',
  endTime: '',
  sortOrder: 99
})

const noticeForm = ref<NoticeForm>({
  title: '',
  content: '',
  isActive: 0
})

// ==================== 🎨 Computed Values ====================
const recommendItems = computed<BoardItem[]>(() => {
  return boardData.value
    .filter(i => i.channel === 'recommend')
    .sort((a, b) => a.sortOrder - b.sortOrder)
})

const getTrackItems = (tId: number | string): BoardItem[] => {
  return boardData.value
    .filter(i => i.channel === 'track' && String(i.trackId) === String(tId))
    .sort((a, b) => a.sortOrder - b.sortOrder)
}

// ==================== 🛠�?Utility Functions ====================
const saveAvatarToHistory = (url: string) => {
  if (!url) return
  let history = [...avatarHistory.value]
  history = history.filter(item => item !== url) // 去重
  history.unshift(url) // 插到最前面
  if (history.length > 8) history = history.slice(0, 8) // 限存 8 �?  avatarHistory.value = history
  localStorage.setItem('avatar_history', JSON.stringify(history))
}

const isTrue = (val: any): boolean => {
  return val === 1 || val === '1' || val === true || String(val).toLowerCase() === 'true'
}

const formatImg = (url: string | undefined): string => {
  if (!url) return ''
  if (url.startsWith('http')) return url
  return 'http://127.0.0.1:8080' + url
}

// ==================== 🔒 Secure Axios Multi-part Uploads ====================
const uploadFile = async (file: File): Promise<string> => {
  const formData = new FormData()
  formData.append('file', file)
  
  // Directly call unified request, using skipEncrypt or skipping GCM because it's a FormData instance
  const res = await request.post('/api/checkin/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
  
  // Compatibility with double-structured JSON returns
  const data = res.data as any
  const url = data?.imageUrl || data?.data?.imageUrl
  if (!url) {
    throw new Error('返回的图�?URL 字段为空')
  }
  return url
}

// Avatar upload handlers
const uploadAvatar = async (options: any, targetForm: any) => {
  try {
    const imageUrl = await uploadFile(options.file)
    targetForm.authorAvatar = imageUrl
    saveAvatarToHistory(imageUrl)
    ElMessage.success("马甲头像已上传并被系统记�?)
  } catch (e) {
    console.error(e)
    ElMessage.error("图片上传鉴权失败，请重试")
  }
}

// Official post image handler
const uploadNewPostImg = async (options: any) => {
  try {
    const imageUrl = await uploadFile(options.file)
    officialForm.value.imageUrl = imageUrl
    ElMessage.success("官方宣传大图挂载成功")
  } catch (e) {
    console.error(e)
    ElMessage.error("图片上传失败")
  }
}

// Dialog editing image handler
const uploadEditPostImg = async (options: any) => {
  try {
    const imageUrl = await uploadFile(options.file)
    editForm.value.imageUrl = imageUrl
    ElMessage.success("宣传配图替换成功")
  } catch (e) {
    console.error(e)
    ElMessage.error("图片上传失败")
  }
}

// ==================== 🗂�?Data Loaders ====================
const refreshAllData = async () => {
  loading.value = true
  try {
    await Promise.all([
      fetchTracks(),
      fetchBoardData(),
      fetchPosts(),
      fetchOfficialPosts(),
      fetchBounties(),
      fetchNotice()
    ])
  } finally {
    loading.value = false
  }
}

const fetchTracks = async () => {
  try {
    const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
    const res = await request.get(`${apiPrefix}/track/list`)
    trackList.value = res.data?.data || []
  } catch (e) {
    console.error('获取赛道配置失败', e)
  }
}

const fetchBoardData = async () => {
  try {
    const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
    const [res1, res2] = await Promise.all([
      request.get(`${apiPrefix}/checkin/recent-list?t=${Date.now()}`),
      request.get(`${apiPrefix}/community/official/list?t=${Date.now()}`)
    ])
    
    // Map User Pushes
    const userPushes: BoardItem[] = (res1.data?.data || [])
      .filter((item: PostItem) => isTrue(item.isPushed) || isTrue(item.status === 1 && item.aiComment?.includes('推荐')))
      .map((item: PostItem) => ({
        uid: 'u_' + item.id,
        id: item.id,
        type: 'checkin',
        title: item.pushTitle || item.contentText?.substring(0, 20) || '【无文字说明�?,
        content: item.pushContent || item.contentText,
        imageUrl: item.imageUrl,
        author: item.userName,
        authorAvatar: item.pushAuthorAvatar || item.avatarUrl,
        isAnonymous: isTrue(item.isAnonymous) ? 1 : 0,
        channel: (item.pushChannel || (item.aiComment?.includes('全局') ? 'recommend' : 'track')) as 'recommend' | 'track',
        trackId: item.trackId ?? null,
        isPinned: isTrue(item.isPinned) ? 1 : 0,
        sortOrder: Number(item.sortOrder) || 0
      }))

    // Map Official announcements Pushes
    const officialPushes: BoardItem[] = (res2.data?.data || []).map((item: OfficialPostItem) => ({
      uid: 'o_' + item.id,
      id: item.id,
      type: 'official',
      title: item.topic,
      content: item.content,
      imageUrl: item.imageUrl,
      author: item.authorName,
      authorAvatar: item.authorAvatar,
      isAnonymous: 0,
      channel: item.channel as 'recommend' | 'track',
      trackId: item.trackId ?? null,
      isPinned: isTrue(item.isPinned) ? 1 : 0,
      sortOrder: Number(item.sortOrder) || 0
    }))

    boardData.value = [...userPushes, ...officialPushes]
  } catch (e) {
    ElMessage.error("获取看板与大盘数据失�?)
  }
}

const fetchPosts = async () => {
  try {
    const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
    const res = await request.get(`${apiPrefix}/checkin/recent-list?t=${Date.now()}`)
    posts.value = (res.data?.data || []).filter((item: PostItem) => item.status === 1)
  } catch (e) {
    console.error(e)
  }
}

const fetchOfficialPosts = async () => {
  try {
    const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
    const res = await request.get(`${apiPrefix}/community/official/list?t=${Date.now()}`)
    officialPosts.value = res.data?.data || []
  } catch (e) {
    console.error(e)
  }
}

// ==================== 🗂�?Kanban Drag & Drop System (TS Typed with visual hover indicators) ====================
const onCardDragStart = (item: BoardItem) => {
  draggedItem.value = item
  isDraggingActive.value = true
}

const onCardDragEnd = () => {
  draggedItem.value = null
  isDraggingActive.value = false
  dragoverActiveColumn.value = ''
  dragoverActiveIndex.value = -1
}

const onColumnDragOver = (colName: string) => {
  dragoverActiveColumn.value = colName
}

const onColumnDragLeave = () => {
  dragoverActiveColumn.value = ''
}

const onCardDragOver = (colName: string, index: number) => {
  dragoverActiveColumn.value = colName
  dragoverActiveIndex.value = index
}

const onColumnDrop = (targetChannel: 'recommend' | string) => {
  if (!draggedItem.value) return
  const isRec = targetChannel === 'recommend'
  const finalChannel = isRec ? 'recommend' : 'track'
  const finalTrackId = isRec ? null : targetChannel
  
  handleDropUpdate(finalChannel, finalTrackId, 9999)
}

const onCardDrop = (targetChannel: 'recommend' | string, targetIndex: number) => {
  if (!draggedItem.value) return
  const isRec = targetChannel === 'recommend'
  const finalChannel = isRec ? 'recommend' : 'track'
  const finalTrackId = isRec ? null : targetChannel
  
  handleDropUpdate(finalChannel, finalTrackId, targetIndex)
}

const handleDropUpdate = async (targetChannel: 'recommend' | 'track', targetTrackId: number | string | null, targetIndex: number) => {
  if (!draggedItem.value) return
  
  const currentDragged = draggedItem.value
  currentDragged.channel = targetChannel
  currentDragged.trackId = targetTrackId
  
  // Re-order locally for rapid layout feedback
  const filtered = boardData.value.filter(
    i => i.channel === targetChannel && 
    (targetChannel === 'recommend' ? true : String(i.trackId) === String(targetTrackId)) &&
    i.uid !== currentDragged.uid
  )
  
  // Insert at target index
  filtered.splice(targetIndex, 0, currentDragged)
  
  // Compute new sort order
  filtered.forEach((item, idx) => {
    item.sortOrder = idx + 1
  })
  
  const sortPayload = filtered.map(i => ({
    id: i.id,
    type: i.type,
    sortOrder: i.sortOrder,
    channel: i.channel,
    trackId: i.trackId
  }))
  
  try {
    const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
    await request.post(`${apiPrefix}/community/board/sort`, sortPayload)
    ElMessage.success("大盘排序成功同步至小程序前端�?)
  } catch (e) {
    ElMessage.error("保存排序发生错误，请刷新页面重新配置")
    fetchBoardData()
  } finally {
    onCardDragEnd()
  }
}

// ==================== 🎨 Dialog Editors & Detail Updaters ====================
const openEditDialog = (item: BoardItem) => {
  editForm.value = JSON.parse(JSON.stringify(item))
  editDialogVisible.value = true
}

const saveEdit = async () => {
  if (editForm.value.channel === 'track' && !editForm.value.trackId) {
    return ElMessage.warning("请选择要推送展示的赛道频道")
  }
  
  try {
    const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
    await request.post(`${apiPrefix}/community/board/update`, { ...editForm.value })
    ElMessage.success("推流参数保存成功，数据同步中")
    editDialogVisible.value = false
    await fetchBoardData()
    await fetchPosts()
    await fetchOfficialPosts()
  } catch (e) {
    ElMessage.error("编辑参数保存失败")
  }
}

const removePush = () => {
  ElMessageBox.confirm('这会将内容移出大盘，小程序端将不再显示，确定继续吗？', '移出大盘确认', {
    confirmButtonText: '确定移出',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      const apiPrefix = isL1OrDev.value ? '/api/admin' : '/api/sub-admin'
      await request.post(`${apiPrefix}/community/board/remove`, {
        id: editForm.value.id,
        type: editForm.value.type
      })
      ElMessage.success("大盘内容移出成功�?)
      editDialogVisible.value = false
      await fetchBoardData()
      await fetchPosts()
      await fetchOfficialPosts()
    } catch(e) {
      ElMessage.error("移出失败，请重试")
    }
  }).catch(() => {})
}

// ==================== 🎣 Post Audit Controllers ====================
const openPushDialog = (row: PostItem) => {
  pushDialogData.value = row
  pushForm.value.channel = row.pushChannel || 'recommend'
  pushForm.value.adminComment = row.pushComment || '打卡质量极高，内容严谨，特推推荐�?
  pushForm.value.authorAvatar = ''
  pushDialogVisible.value = true
}

const submitPush = async () => {
  if (!pushDialogData.value) return
  
  try {
    await request.post('/api/admin/community/push', {
      recordId: pushDialogData.value.id,
      channel: pushForm.value.channel,
      adminComment: pushForm.value.adminComment,
      authorAvatar: pushForm.value.authorAvatar
    })
    ElMessage.success("🎉 推流完成，内容已同步挂载大盘�?)
    pushDialogVisible.value = false
    await fetchPosts()
    await fetchBoardData()
  } catch (e) {
    ElMessage.error("推流配置上送失�?)
  }
}

const cancelPush = (id: number | string) => {
  ElMessageBox.confirm('确定要取消该用户打卡推流吗？', '取消推流', {
    confirmButtonText: '确认取消',
    cancelButtonText: '暂不处理',
    type: 'warning'
  }).then(async () => {
    try {
      await request.post('/api/admin/community/cancel-push', { recordId: id })
      ElMessage.success("已成功下线打卡推�?)
      await fetchPosts()
      await fetchBoardData()
    } catch (e) {
      ElMessage.error("取消动作失败，请重试")
    }
  }).catch(() => {})
}

const togglePostPin = async (row: PostItem) => {
  try {
    const targetPin = isTrue(row.isPinned) ? 0 : 1
    await request.post(`/api/admin/community/official/toggle-pin?id=${row.id}&isPinned=${targetPin}`)
    ElMessage.success(targetPin === 1 ? "该动态已配置全局置顶" : "已顺利取消该置顶")
    await fetchPosts()
  } catch (e) {
    ElMessage.error("置顶更改失败")
  }
}

// ==================== 📝 Official Posts Controllers ====================
const publishOfficialPost = async () => {
  if (!officialForm.value.topic.trim() || !officialForm.value.content.trim()) {
    return ElMessage.warning("请确保官方动态的主题与正文内容填写完�?)
  }
  if (officialForm.value.channel === 'track' && !officialForm.value.trackId) {
    return ElMessage.warning("在专属赛道广播下，请务必选择一个具体的关联赛道")
  }
  
  try {
    await request.post('/api/admin/community/official/publish', { ...officialForm.value })
    ElMessage.success("🎉 官方权威动态发布成功！")
    
    // Reset Form
    officialForm.value = {
      authorName: localStorage.getItem('admin_name') || '官方小助�?,
      authorAvatar: '',
      topic: '',
      content: '',
      imageUrl: '',
      channel: 'recommend',
      trackId: null,
      isPinned: 0
    }
    
    activeTab.value = 'board'
    await fetchBoardData()
    await fetchOfficialPosts()
  } catch (e) {
    ElMessage.error("发布官方动态失�?)
  }
}

const deleteOfficialPost = (id: number | string) => {
  ElMessageBox.confirm('物理删除此官方动态无法回滚，学生端大盘也将随之清除，确定删除�?, '高危物理删除', {
    confirmButtonText: '确定物理删除',
    cancelButtonText: '取消',
    type: 'error'
  }).then(async () => {
    try {
      await request.delete(`/api/admin/community/official/delete?id=${id}`)
      ElMessage.success("官方动态删除完�?)
      await fetchOfficialPosts()
      await fetchBoardData()
    } catch (e) {
      ElMessage.error("删除动作失败，请重试")
    }
  }).catch(() => {})
}

// ==================== 💰 Bounty Board Operations & Mem-Safe Timers ====================
const fetchBounties = async () => {
  if (!isL1OrDev.value) return
  try {
    const res = await request.get('/api/admin/community/bounty/list')
    bounties.value = res.data?.data || []
    calculateBountyCountdowns()
  } catch (e) {
    console.error(e)
  }
}

const calculateBountyCountdowns = () => {
  const now = Date.now()
  bounties.value.forEach(b => {
    if (b.endTime) {
      // Safe Date Parser across OS / Safaris
      const targetTime = new Date(b.endTime.replace(/-/g, '/')).getTime()
      const diff = targetTime - now
      
      if (diff > 0) {
        const hours = Math.floor(diff / (1000 * 60 * 60))
        const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
        const seconds = Math.floor((diff % (1000 * 60)) / 1000)
        b.countdown = `${hours}�?{minutes}�?{seconds}秒`
      } else {
        b.countdown = '已结�?
      }
    } else {
      b.countdown = null
    }
  })
}

const openBountyDialog = () => {
  bountyForm.value = {
    title: '',
    rewardText: '',
    endTime: '',
    sortOrder: 99
  }
  bountyDialogVisible.value = true
}

const submitBounty = async () => {
  if (!bountyForm.value.title.trim() || !bountyForm.value.rewardText.trim()) {
    return ElMessage.warning("任务描述与奖励描述不能为空！")
  }
  
  try {
    await request.post('/api/admin/community/bounty/save', bountyForm.value)
    ElMessage.success("🎉 新悬赏包成功挂载上榜�?)
    bountyDialogVisible.value = false
    await fetchBounties()
  } catch(e) {
    ElMessage.error("发布新悬赏失�?)
  }
}

const deleteBounty = async (id: number | string) => {
  ElMessageBox.confirm('确定要下架并删除这一条悬赏榜单吗�?, '下架悬赏', {
    confirmButtonText: '确定下架',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await request.delete(`/api/admin/community/bounty/delete?id=${id}`)
      ElMessage.success("悬赏已成功下�?)
      await fetchBounties()
    } catch (e) {
      ElMessage.error("下架失败")
    }
  }).catch(() => {})
}

const toggleBountyPin = async (id: number | string, currentPin: any) => {
  try {
    const isPinned = isTrue(currentPin) ? 0 : 1
    await request.post(`/api/admin/community/bounty/toggle-pin?id=${id}&isPinned=${isPinned}`)
    ElMessage.success(isPinned === 1 ? "悬赏成功置顶展示" : "已顺利取消置�?)
    await fetchBounties()
  } catch (e) {
    ElMessage.error("置顶切换失败")
  }
}

const onBountyDragStart = (index: number) => {
  dragStartIndexBounty.value = index
}

const onBountyDrop = async (dropIndex: number) => {
  if (dragStartIndexBounty.value === dropIndex || dragStartIndexBounty.value === -1) return
  
  // Swap elements
  const currentList = [...bounties.value]
  const dragged = currentList[dragStartIndexBounty.value]
  currentList.splice(dragStartIndexBounty.value, 1)
  currentList.splice(dropIndex, 0, dragged)
  
  // Map index
  const sortData = currentList.map((item, idx) => ({
    id: item.id,
    sortOrder: idx + 1
  }))
  
  try {
    await request.post('/api/admin/community/bounty/update-sort', sortData)
    ElMessage.success("重新调整悬赏排序大盘成功�?)
    await fetchBounties()
  } catch(e) {
    ElMessage.error("保存排序配置失败")
  } finally {
    dragStartIndexBounty.value = -1
  }
}

// ==================== 📢 Global Notices Controllers ====================
const fetchNotice = async () => {
  if (!isL1OrDev.value) return
  try {
    const res = await request.get('/api/admin/community/notice/get')
    if (res.data?.data) {
      noticeForm.value = res.data.data
    }
  } catch(e) {
    console.error(e)
  }
}

const saveNotice = async () => {
  if (!noticeForm.value.title.trim() || !noticeForm.value.content.trim()) {
    return ElMessage.warning("广播的标题与条款正文内容均不可留�?)
  }
  
  try {
    await request.post('/api/admin/community/notice/save', noticeForm.value)
    ElMessage.success("🎉 全局弹窗强推广播内容配置保存成功�?)
    await fetchNotice()
  } catch (e) {
    ElMessage.error("广播保存失败")
  }
}

const toggleNotice = async (val: any) => {
  try {
    const isActive = isTrue(val)
    await request.post(`/api/admin/community/notice/toggle?isActive=${isActive}`)
    ElMessage.success(isActive ? "⚠️ 全服登陆首屏强弹窗广播通道已开启！" : "广播强推静默关闭成功")
    await fetchNotice()
  } catch (e) {
    ElMessage.error("广播激活切换失�?)
  }
}

// ==================== 🔄 Lifecycles ====================
onMounted(async () => {
  await refreshAllData()
  
  // Set Safe Timer and safely dispose onUnmount
  bountyCountdownTimer = setInterval(calculateBountyCountdowns, 1000)
})

onUnmounted(() => {
  if (bountyCountdownTimer) {
    clearInterval(bountyCountdownTimer)
    bountyCountdownTimer = null
  }
})
</script>

<style scoped>
.community-manage-tabs :deep(.el-tabs__item) {
  color: #64748b;
  font-size: 14px;
  font-weight: 600;
  transition: all 0.3s;
}
.community-manage-tabs :deep(.el-tabs__item.is-active) {
  color: #3b82f6; /* Modern Blue 500 */
}
.community-manage-tabs :deep(.el-tabs__active-bar) {
  background-color: #3b82f6;
  height: 3px;
  border-radius: 9999px;
}
.community-manage-tabs :deep(.el-tabs__nav-wrap::after) {
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
  background-color: #eff6ff !important;
}
:deep(.el-table::before) {
  display: none;
}

.custom-dialog :deep(.el-dialog__header) {
  padding-bottom: 15px;
  border-bottom: 1px solid #f1f5f9;
  margin-right: 0;
}
.custom-dialog :deep(.el-dialog__body) {
  padding-top: 20px;
  padding-bottom: 20px;
}

.avatar-uploader-box {
  display: inline-block;
}

.image-uploader-box {
  display: inline-block;
}
</style>
