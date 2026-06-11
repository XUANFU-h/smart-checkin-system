import request from './request'

// ==================== 🧠 1. 赛道配置与 AI 调度中心 ====================
export const getAiSettings = () => request.get('/admin/settings/get')
export const getTrackOptions = () => request.get('/auth/tracks')
export const fetchAuditList = (params: { date: string; trackId?: number }) =>
    request.get('/admin/audit/list', { params })
export const toggleAiEngine = (data: { ai_ocr_enabled?: string; ai_deepseek_enabled?: string }) =>
    request.post('/admin/settings/save', data)
export const triggerAiPipeline = (data: { id?: number; mode: string; date?: string }) =>
    request.post('/admin/ai/trigger', data)
export const publishScores = (data: { date: string; trackId?: number }) =>
    request.post('/admin/audit/publish', data)

// ==================== 📊 2. 数据与监控看板 ====================
export const getDashboardData = () => request.get('/admin/dashboard')

// ==================== ⚙️ 3. 系统核心配置 ====================
export const getSysSettings = () => request.get('/admin/settings/get')
export const saveSysSettings = (data: any) => request.post('/admin/settings/save', data)
export const resetAdminPassword = (data: { newPassword: string; adminId: number }) =>
    request.post('/admin/auth/reset-self', data)

// ==================== ⚖️ 4. 申诉审核大厅 ====================
export const getAppealCategories = () => request.get('/ticket/categories') // 对齐后端 ticket 分组
export const saveAppealCategory = (data: any) => request.post('/admin/appeal/category/save', data)
export const deleteAppealCategory = (id: number) => request.post('/admin/appeal/category/delete', { id })
export const getAppealList = (params: { status: number; keyword?: string; date?: string }) =>
    request.get('/admin/appeal/list', { params })
export const processAppeal = (data: any) => request.post('/admin/appeal/process', data)
export const deleteAppeal = (id: number) => request.post('/admin/appeal/delete', { id })

// ==================== 🪙 5. 学风驿站商城 ====================
export const getShopList = () => request.get('/shop/list')
export const saveShopItem = (data: any) => request.post('/admin/mall/item', data) // 精准对齐后端 adminGroup /mall/item
export const appendShopStock = (data: any) => request.post('/admin/mall/append-stock', data)
export const deleteShopItem = (data: any) => request.post('/admin/mall/delete', data)
export const getExchangeList = (params: { keyword?: string; statuses?: string }) =>
    request.get('/admin/sub-admin/blind-list', { params })

// ==================== 📢 6. 社区管理与推流 ====================
export const uploadCommunityFile = (formData: FormData) =>
    request.post('/checkin/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
export const getCommunityTracks = () => request.get('/admin/sub-admin/blind-list')
export const getRecentCheckins = () => request.get('/admin/checkin/recent-list')
export const getOfficialList = () => request.get('/admin/community/official/list')
export const saveBoardSort = (data: any[]) => request.post('/admin/community/board/sort', data)
export const updateBoardItem = (data: any) => request.post('/admin/community/board/update', data)
export const removeBoardItem = (data: any) => request.post('/admin/community/board/remove', data)
export const pushToCheckinBoard = (data: any) => request.post('/admin/community/push', data)
export const cancelCheckinPush = (data: any) => request.post('/admin/community/cancel-push', data)
export const toggleOfficialPin = (params: { id: number; isPinned: number }) =>
    request.post('/admin/community/official/toggle-pin', null, { params })
export const publishOfficialNotice = (data: any) => request.post('/admin/community/official/publish', data)
export const deleteOfficialNotice = (id: number) => request.delete(`/admin/community/official/delete?id=${id}`)

// ==================== 🪙 7. 悬赏榜单与首屏弹窗 ====================
export const getBountyList = () => request.get('/admin/community/bounty/list')
export const saveBountyItem = (data: any) => request.post('/admin/community/bounty/save', data)
export const deleteBountyItem = (id: number) => request.delete(`/admin/community/bounty/delete?id=${id}`)
export const toggleBountyPin = (params: { id: number; isPinned: number }) =>
    request.post('/admin/community/bounty/toggle-pin', null, { params })
export const updateBountySort = (data: any[]) => request.post('/admin/community/bounty/update-sort', data)
export const getPopupNotice = () => request.get('/admin/community/notice/get')
export const savePopupNotice = (data: any) => request.post('/admin/community/notice/save', data)
export const togglePopupNotice = (params: { isActive: boolean }) =>
    request.post('/admin/community/notice/toggle', null, { params })