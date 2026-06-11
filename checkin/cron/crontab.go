package cron

import (
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/service"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// SetupCronJobs 挂载所有定时任务
func SetupCronJobs() {
	shanghaiTime, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		shanghaiTime = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	c := cron.New(cron.WithLocation(shanghaiTime))

	morningCron := "0 10 * * *"
	var cfgMorning models.SystemDynamicConfig

	if err := global.DB.Where("config_key = ?", "cron_morning").First(&cfgMorning).Error; err == nil {
		morningCron = cfgMorning.ConfigValue
	}

	// 注册晨间流水线 (统一发榜与加分)
	c.AddFunc(morningCron, func() {
		// 💡 修复：调用全新升级后的放榜函数，传入空字符串和0代表全量放榜
		service.PublishScoresForDate("", 0)
	})

	// (预留) 赛道干事凌晨洗牌算法
	c.AddFunc("0 2 * * *", func() {
		log.Println("执行二级干事赛道负责权盲洗牌...")
		// service.ExecSubAdminTrackShuffle()
	})

	c.Start()
	log.Println("定时任务引擎启动成功，[注意：AI夜间批阅已转为后台手动/大盘调度触发]...")
}
