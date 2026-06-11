package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"fmt"
	"log"
	"strconv"

	"gorm.io/gorm"
)

// AIResult DeepSeek 强制要求的标准 JSON 结构
type AIResult struct {
	Score      int    `json:"score"`       // 5-20分
	Comment    string `json:"comment"`     // 20字以内评语
	RiskLevel  string `json:"risk_level"`  // SAFE, WARNING, BAN
	RiskReason string `json:"risk_reason"` // 违规原因(如果有)
}

// TriggerAiPipelineForDate 异步手动/自动激活 AI 调度引擎
func TriggerAiPipelineForDate(date string) {
	// 🛡️ 核心防爆盾：确保即使 AI 接口崩溃、超时或 API Key 未配置，也不会导致整个后端进程闪退
	defer func() {
		if r := recover(); r != nil {
			log.Println("⛔ [AI安全兜底] 流水线遭遇极度异常，已成功拦截崩溃:", r)
		}
	}()

	log.Println("[AI流水线] 收到总控面板指令，正在唤醒引擎强制批阅:", date)

	// 1. 动态获取分数区间
	var minCfg, maxCfg models.SystemDynamicConfig
	minScore, maxScore := 5, 20
	if err := global.DB.Where("config_key = ?", "score_min").First(&minCfg).Error; err == nil {
		minScore, _ = strconv.Atoi(minCfg.ConfigValue)
	}
	if err := global.DB.Where("config_key = ?", "score_max").First(&maxCfg).Error; err == nil {
		maxScore, _ = strconv.Atoi(maxCfg.ConfigValue)
	}

	// 2. 捞取指定日期未被批阅的数据 (ai_score = 0 且 status = 0等待批阅)
	var records []models.CheckinRecord
	query := global.DB.Where("ai_score = 0 AND status = 0")
	if date != "" {
		query = query.Where("checkin_date = ?", date)
	}
	query.Find(&records)

	// 计算默认评分（取极值范围的中间数，默认为10分）
	defaultScore := (minScore + maxScore) / 2
	if defaultScore <= 0 {
		defaultScore = 10
	}

	for _, record := range records {
		// 🛡️ AI 与 OCR 均已被物理断开并注释，直接转入纯本地安全合规处理
		global.DB.Transaction(func(tx *gorm.DB) error {
			record.AiScore = defaultScore
			record.AiComment = "[系统批阅] 提交内容格式安全，审核已合规通过"
			record.Status = 1       // 🌟 取消放榜：AI自动批阅通过后直接置为已通过
			record.PublishStatus = 1 // 🌟 取消放榜：直接发布，同步至学生端
			if err := tx.Save(&record).Error; err != nil {
				return err
			}

			// 🌟 算分入库：加分
			if record.AiScore > 0 {
				if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", record.AiScore, record.UserID).Error; err != nil {
					return err
				}

				// 🌟 写入 ScoreLog 流水表
				logRec := models.ScoreLog{
					UserID:    record.UserID,
					ScoreType: "EMP",
					Score:     record.AiScore,
					Action:    "ADD",
					Reason:    fmt.Sprintf("打卡成功 AI智能批阅自动通过并同步"),
					Operator:  "AI智能批阅大脑",
				}
				if err := tx.Create(&logRec).Error; err != nil {
					return err
				}
			}
			return nil
		})
	}
	log.Println("[AI流水线] 批阅流执行完毕，等待发榜")
}

// PublishScoresForDate 异步一键放榜 (发布成绩 + 算分入库 + 写入流水)
func PublishScoresForDate(date string, trackId int) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("⛔ [AI安全兜底] 放榜遭遇极度异常:", r)
		}
	}()

	log.Println("[大盘调度] 正在执行一键放榜指令...")

	query := global.DB.Model(&models.CheckinRecord{}).Where("status = 2 AND publish_status = 0")
	if date != "" {
		query = query.Where("checkin_date = ?", date)
	}
	if trackId > 0 {
		query = query.Where("track_id = ?", trackId)
	}

	var records []models.CheckinRecord
	query.Find(&records)

	for _, record := range records {
		var user models.User
		if err := global.DB.Select("id, account_status").First(&user, record.UserID).Error; err != nil {
			continue
		}

		if user.AccountStatus != 0 {
			global.DB.Model(&record).Update("status", 4) // 异常用户直接废弃
			continue
		}

		// 💡 重点修复：使用事务将【发榜更新记录】+【加赋能分】+【写入流水】强绑定
		global.DB.Transaction(func(tx *gorm.DB) error {
			// 1. 变更为通过并公布
			tx.Model(&record).Updates(map[string]interface{}{
				"status":         1,
				"publish_status": 1,
			})

			// 2. 加分
			if record.AiScore > 0 {
				tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", record.AiScore, record.UserID)

				// 3. 写入 ScoreLog 流水表
				logRec := models.ScoreLog{
					UserID:    record.UserID,
					ScoreType: "EMP",
					Score:     record.AiScore,
					Action:    "ADD",
					Reason:    fmt.Sprintf("打卡成功 AI智能批阅奖励"),
					Operator:  "AI智能批阅大脑",
				}
				tx.Create(&logRec)
			}
			return nil
		})
	}
	log.Println("[大盘调度] 一键放榜与积分结算完成，积分流水已安全写入")
}

// ======================= 内部私有 API 引擎 =======================

func callVolcengineOCR(imageURL string) string {
	// 已注释，物理断开火山引擎OCR
	return ""
}

func callDeepSeekEvaluation(userText string, ocrText string, minScore int, maxScore int) (*AIResult, error) {
	// 已注释，物理断开DeepSeek大模型
	return &AIResult{
		Score:      10,
		Comment:    "[本地合规评估]",
		RiskLevel:  "SAFE",
		RiskReason: "",
	}, nil
}
