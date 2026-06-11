package main

import (
	"checkin-go/cron"
	"checkin-go/global"
	"checkin-go/initialize"
	"checkin-go/router"
	"checkin-go/service"
	"fmt"
	"log/slog"
)

func main() {
	initialize.InitConfig()
	initialize.InitDB()
	initialize.InitRedis()

	// 启动核心异步引擎
	service.StartGameCron()

	// 🚀 终极点火：启动数据大盘 Linux 物理硬件实时抓取与 WebSocket 广播双流引擎！
	service.StartDashboardCronEngine()
	slog.Info("💡 [监控大盘中枢] 服务器实时遥测与无感刷新协程启动成功")

	// 启动 AI 流水线 Cron 中枢
	cron.SetupCronJobs()

	// 架构优化：应用层保持绝对轻量。由网络基础设施（TLS 1.3）承担传输层高密安全，
	// 并在内部逻辑路由层前置注册了轻量化 JWT + 接口数字签名防护锁。
	r := router.SetupRouter()

	slog.Info("路由初始化完成")

	addr := fmt.Sprintf(":%d", global.Config.Server.Port)
	slog.Info("服务器启动成功", "addr", "http://localhost"+addr)

	if err := r.Run(addr); err != nil {
		slog.Error("服务器启动失败", "err", err)
		panic(err)
	}
}
