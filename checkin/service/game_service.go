package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// StartGameCron 启动周末大乐透开奖调度器
func StartGameCron() {
	shanghaiTime, _ := time.LoadLocation("Asia/Shanghai")
	c := cron.New(cron.WithSeconds(), cron.WithLocation(shanghaiTime))

	// 每周六晚上 22:00:00 准时开奖
	c.AddFunc("0 0 22 * * 6", ProcessWeeklyLotto)

	c.Start()
	slog.Info("🎰 大乐透自动开奖中心已启动，等待周六调度...")
}

// ProcessWeeklyLotto 开奖并自动发奖逻辑 (包含阳光普照补偿)
func ProcessWeeklyLotto() {
	slog.Info("开始执行本周大乐透开奖流程...")
	year, week := time.Now().ISOWeek()
	weekNum := year*100 + week

	global.DB.Transaction(func(tx *gorm.DB) error {
		var setting models.LottoSetting
		err := tx.First(&setting, weekNum).Error

		// 如果管理员没有在后台设置内幕，系统自动随机生成一个阳光大奖
		if err != nil {
			n1 := fmt.Sprintf("%02d", rand.Intn(100))
			n2 := fmt.Sprintf("%02d", rand.Intn(100))
			n3 := fmt.Sprintf("%02d", rand.Intn(100))

			setting = models.LottoSetting{
				WeekNum:       weekNum,
				WinningNumber: fmt.Sprintf("%s-%s-%s", n1, n2, n3),
				PrizeType:     "empowerment",
				PrizeValue:    1000, // 默认大奖1000赋能分
				IsDrawn:       true,
			}
			tx.Create(&setting)
		} else {
			setting.IsDrawn = true
			tx.Save(&setting)
		}

		slog.Info("🎯 本周中奖号码揭晓", "Number", setting.WinningNumber, "Prize", setting.PrizeValue)

		// 捞取本周所有下注凭证
		var entries []models.UserLottoEntry
		tx.Where("week_num = ?", weekNum).Find(&entries)

		for _, entry := range entries {
			if entry.LottoNumber == setting.WinningNumber {
				// 命中大奖
				entry.IsWinner = true
				entry.PrizeInfo = fmt.Sprintf("恭喜中得头奖: %d 赋能分", setting.PrizeValue)
				tx.Save(&entry)

				tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", setting.PrizeValue, entry.UserID)

				log := models.ScoreLog{UserID: entry.UserID, ScoreType: "EMP", Score: setting.PrizeValue, Action: "ADD", Reason: "学风大乐透头奖", Operator: "系统开奖"}
				tx.Create(&log)
			} else {
				// 未中大奖：阳光普照补偿 50 赋能分
				entry.IsWinner = false
				entry.PrizeInfo = "未中奖，获得补偿: 50 赋能分"
				tx.Save(&entry)

				tx.Exec("UPDATE user SET empowerment_score = empowerment_score + 50 WHERE id = ?", entry.UserID)

				log := models.ScoreLog{UserID: entry.UserID, ScoreType: "EMP", Score: 50, Action: "ADD", Reason: "学风大乐透阳光普照补偿", Operator: "系统开奖"}
				tx.Create(&log)
			}
		}

		slog.Info("💰 本周奖品及补偿派发完毕", "总参与人数", len(entries))
		return nil
	})
}
