import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

// 💡 引入大一统主壳布局组件，让侧边栏和主页内容完美共存
const Layout = () => import('@/layout/index.vue')

const routes: Array<RouteRecordRaw> = [
    {
        path: '/',
        redirect: '/login'
    },
    {
        path: '/login',
        name: 'Login',
        component: () => import('@/views/auth/Login.vue'),
        meta: { title: '安全登录 - 智能打卡系统', requiresAuth: false }
    },
    // 🚀 将平铺路由全部重构为 Layout 的子路由，让侧边栏导航在这些页面持久常驻！
    {
        path: '/admin',
        component: Layout,
        redirect: '/admin/dashboard', // 💡 完美顺应心愿：进来后默认直接去主页（数据看板）
        children: [
            {
                path: 'dashboard',
                name: 'DataMonitor',
                component: () => import('@/views/dashboard/index.vue'),
                meta: { title: '数据与监控看板 - 智能打卡系统', requiresAuth: true }
            },

            {
                path: 'students',
                name: 'StudentUserManage',
                component: () => import('@/views/student/index.vue'),
                meta: { title: '学生人员档案库 - 智能打卡系统', requiresAuth: true }
            },
            {
                path: 'lotto',
                name: 'CreativeLottoManage',
                component: () => import('@/views/lotto/index.vue'),
                meta: { title: '大乐透创意开奖大中枢 - 智能打卡系统', requiresAuth: true }
            },
            {
                path: 'audit',
                name: 'CheckinAuditCenter',
                component: () => import('@/views/audit/index.vue'),
                meta: { title: '赛道审批与 AI 调度中心 - 智能打卡系统', requiresAuth: true }
            },
            {
                path: 'community',
                name: 'CommunityManage',
                component: () => import('@/views/community/index.vue'),
                meta: { title: '社区推流与动态管理 - 智能打卡系统', requiresAuth: true }
            },
            {
                path: 'settings',
                name: 'SystemSettings',
                component: () => import('@/views/settings/index.vue'),
                meta: { title: '系统核心配置中心 - 智能打卡系统', requiresAuth: true }
            },
            {
                path: 'appeal',
                name: 'AppealHall',
                component: () => import('@/views/appeal/index.vue'),
                meta: { title: '申诉审核大厅 - 智能打卡系统', requiresAuth: true }
            },
            {
                path: 'shop',
                name: 'AcademicShop',
                component: () => import('@/views/shop/index.vue'),
                meta: { title: '学风驿站商城 - 智能打卡系统', requiresAuth: true }
            }
        ]
    },
    {
        path: '/:pathMatch(.*)*',
        name: 'NotFound',
        component: () => import('@/views/error/index.vue'),
        meta: { title: '404 - 页面未找到', requiresAuth: false }
    }
]

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes
})

router.beforeEach((to, _from, next) => {
    if (to.meta.title) {
        document.title = to.meta.title as string
    }

    const token = localStorage.getItem('userToken') || localStorage.getItem('admin_token')

    if (to.meta.requiresAuth && !token) {
        next({ path: '/login', query: { redirect: to.fullPath } })
    } else if (to.path === '/login' && token) {
        // 💡 登录成功后，防止卡在登录页，默认直奔管理主页（数据监控看板）
        next('/admin/dashboard')
    } else {
        next()
    }
})

export default router