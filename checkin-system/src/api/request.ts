/**
 * 智能自适应通信网关 — request.ts
 *
 * 策略（资深架构级 Web 通信中枢）：
 * ✅ 高智能环境感知：开发自适应 8080 端口，线上自动切换 8080 或 生产网关
 * ✅ 自动清洗防呆：完美咬合后端 Gin 的 /api 顶级路由分组
 * ✅ 开放接口物理降维：完美兼容并剥离后端 captcha 开放回执，规避 Proxy 逻辑踩踏
 * ✅ 403 物理风控感知：一旦捕获封号或强退拦截，立刻死锁屏幕强制驱逐，绝不留反抗缝隙
 * ✅ 自动附加 Bearer 认证 Token 与 Device-Id 设备安全防伪指纹
 */

import axios, {
    type AxiosInstance,
    type InternalAxiosRequestConfig,
    type AxiosResponse
} from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

// 开发模式自动锁死对接 8080 端口，生产模式对接线上网关
const isDev =
    window.location.hostname === 'localhost' ||
    window.location.hostname === '127.0.0.1'

// 将函数重构为标准变量获取，完美对齐环境自适应
const getBaseURL = (): string => {
    const hostname = window.location.hostname
    const protocol = window.location.protocol

    // 1. 本地沙盒开发环境 (localhost 或 127.0.0.1)
    if (isDev) {
        return 'http://127.0.0.1:8080/api'
    }

    // 2. 服务器生产内测环境 (如果显式使用 8080 端口)
    if (window.location.port === '8080') {
        return `${protocol}//${hostname}:8080/api`
    }

    // 3. 终极默认线上生产网关自适应适配
    let apiHost = window.location.host
    if (hostname.startsWith('admin.')) {
        apiHost = hostname.replace(/^admin\./, 'api.')
    }
    return `${protocol}//${apiHost}/api`
}

// 生成或获取设备唯一指纹，确保 system 防护屏障安全
function getDeviceId(): string {
    let deviceId = localStorage.getItem('admin_device_id')
    if (!deviceId) {
        deviceId =
            'ADMIN-' +
            Math.random().toString(36).substring(2, 11) +
            Date.now().toString(36).toUpperCase()
        localStorage.setItem('admin_device_id', deviceId)
    }
    return deviceId
}

// 清除过期管理员账户鉴权状态并驱逐回登录界面
function clearAuthAndRedirect(): void {
    const keys = ['admin_token', 'admin_id', 'admin_name', 'admin_role', 'l2_token', 'userToken']
    // 💡 保留设备 ID 不清除，确保该设备处于被拉黑的硬件防伪盾持续生效状态
    keys.forEach(k => localStorage.removeItem(k))
    window.location.href = '/login'
}

// 建立 Axios 高可用网络请求实例
const request: AxiosInstance = axios.create({
    baseURL: getBaseURL(),
    timeout: 15000,
    headers: { 'Content-Type': 'application/json' }
})

// 请求拦截器：注入认证凭据与安全指纹
request.interceptors.request.use(
    (config: InternalAxiosRequestConfig) => {
        // 智能防呆过滤：裁剪抹平可能出现的重复 /api
        if (config.url && config.url.startsWith('/api/')) {
            config.url = config.url.substring(4)
        } else if (config.url && config.url.startsWith('/api')) {
            config.url = config.url.substring(4)
        }

        // 确保路径开头有标准斜杠
        if (config.url && !config.url.startsWith('/')) {
            config.url = '/' + config.url
        }

        const token = localStorage.getItem('admin_token') || localStorage.getItem('userToken') || localStorage.getItem('l2_token')
        if (token) {
            config.headers['Authorization'] = token.startsWith('Bearer ')
                ? token
                : `Bearer ${token}`
        }

        config.headers['X-Device-Id'] = getDeviceId()
        return config
    },
    (error: any) => Promise.reject(error)
)

// 响应拦截器：纯净 JSON 返回值解包 + 高度兼容性 Proxy
request.interceptors.response.use(
    (response: AxiosResponse) => {
        const res = response.data
        const url = response.config.url || ''

        // 💡 VIP 开放接口解包绿色通道：
        if ((url.includes('captcha') || url.includes('generate')) && res && res.data) {
            return JSON.parse(JSON.stringify(res.data))
        }

        // 业务响应状态码检测 (200 / 0 / 成功标签 为物理成功指标)
        if (res.code === 200 || res.code === 0 || res.success === true) {
            if (response.data && typeof response.data === 'object') {
                response.data.success = true
            }

            // 登录回执单独原样放行，防止登录凭证内部多层对象遭遇过度解包
            if (url.includes('login')) {
                return res
            }

            // 💯 完美重构：完美留存老版Proxy拦截逻辑，并强力修复 data 层级反查踩空报错
            return new Proxy(res, {
                get(target, prop, receiver) {
                    // 💡 防御核心：当内页长篇代码尝试去获取 .data 节点时，如果 target 内部本身就是数据（例如数组），
                    // 或者 target 是包装对象且包含真实的 data 节点，我们进行智能解包或自适应定向返回，决不返还 undefined。
                    if (prop === 'data') {
                        if (target && target.data !== undefined) {
                            return target.data
                        }
                        return target
                    }
                    if (prop === '__axios_status') return response.status
                    if (prop === '__axios_headers') return response.headers

                    return Reflect.get(target, prop, receiver)
                }
            }) as any
        } else {
            ElMessage.error(res.msg || res.message || '业务逻辑判定拦截')
            return Promise.reject(new Error(res.msg || res.message || 'Error'))
        }
    },
    (error: any) => {
        if (error.response) {
            const { status, data } = error.response as { status: number; data: Record<string, string> }
            const errMsg: string = data?.msg || data?.message || ''

            switch (status) {
                case 401:
                    ElMessage.error('登录凭证过期，请重新验证管理员身份')
                    clearAuthAndRedirect()
                    break

                case 403:
                    ElMessageBox.alert(
                        `⚠️ ${errMsg || '安全审计风控拦截：当前管理权已被系统超级管理员依法物理隔离封禁！'}`,
                        '最高级别风控处决强退警报',
                        {
                            confirmButtonText: '接受处罚并强制登出',
                            type: 'error',
                            showClose: false,
                            callback: () => {
                                clearAuthAndRedirect()
                            }
                        }
                    )
                    break

                case 404:
                    ElMessage.error(`网关定位异常 (404)：请确保后端 Go 服务已成功开启且路由匹配`)
                    break

                case 500:
                    ElMessage.error(errMsg || '后端引擎发生运行时未知恐慌 (500)')
                    break

                default:
                    ElMessage.error(errMsg || `网关通信阻断 (${status})`)
            }
        } else {
            ElMessage.error('网络物理阻断，请确保本地 Go 引擎已在 8080 端口成功运行')
        }

        return Promise.reject(error)
    }
)

export default request
