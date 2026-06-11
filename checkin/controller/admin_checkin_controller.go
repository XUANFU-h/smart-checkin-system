package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/service"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditRecordResp struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	TrackID       int       `json:"trackId"`
	ContentText   string    `json:"contentText"`
	ImageURL      string    `json:"imageUrl"`
	Status        int8      `json:"status"`
	AiScore       int       `json:"aiScore"`
	AiComment     string    `json:"aiComment"`
	AdminComment  string    `json:"adminComment"`
	PublishStatus int8      `json:"publishStatus"`
	CheckinDate   string    `json:"checkinDate"`
	CreateTime    time.Time `json:"createTime"`
	Nickname      string    `json:"nickname"`
	StudentID     string    `json:"studentId"`
	ClassName     string    `json:"className"`
	TrackName     string    `json:"trackName"`
}

type ManualEvalReq struct {
	RecordID    int64  `json:"recordId"`
	NewScore    int    `json:"score"`
	Comment     string `json:"comment"`
	ContentText string `json:"contentText"`
	IsReject    bool   `json:"isReject"` // 🌟 新增：是否为驳回操作
}

// ManualEvaluate 人工批阅/覆盖分数
func ManualEvaluate(c *gin.Context) {
	var req ManualEvalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	var record models.CheckinRecord
	if err := global.DB.First(&record, req.RecordID).Error; err != nil {
		common.Error(c, "打卡记录不存在")
		return
	}

	// 💡 智能调分差额机制：
	// 如果是驳回操作，目标得分强制为 0
	targetScore := req.NewScore
	if req.IsReject {
		targetScore = 0
	}

	var scoreDiff int
	if record.PublishStatus == 1 {
		scoreDiff = targetScore - record.AiScore
	} else {
		scoreDiff = targetScore
	}

	adminName := "系统最高指挥官" // 后续可从 JWT 获取当前管理员名称

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 更新记录
		record.AiScore = targetScore
		record.AdminComment = req.Comment
		
		if req.IsReject {
			record.Status = 2 // 2=已驳回
		} else {
			record.Status = 1 // 1=已通过
		}
		
		record.PublishStatus = 1 // 🌟 取消放榜逻辑：导师打分后即刻放榜同步给学生
		if req.ContentText != "" {
			record.ContentText = req.ContentText
		}
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		// 2. 更新用户总分
		if scoreDiff != 0 {
			if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", scoreDiff, record.UserID).Error; err != nil {
				return err
			}

			// 3. 💡 增加积分流水记录
			actionLabel := "ADD"
			reasonLabel := "人工批阅打分"
			if req.IsReject {
				actionLabel = "SUB"
				reasonLabel = "人工批阅驳回清分"
			}

			log := models.ScoreLog{
				UserID:    record.UserID,
				ScoreType: "EMP",
				Score:     scoreDiff,
				Action:    actionLabel,
				Reason:    fmt.Sprintf("导师%s，评语变动说明: %s", reasonLabel, req.Comment),
				Operator:  adminName,
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "数据库操作失败: "+err.Error())
		return
	}

	common.Success(c, "人工批阅修改成功")
}

// GetAuditList 获取赛道审批大盘数据
func GetAuditList(c *gin.Context) {
	date := c.Query("date")
	trackId := c.Query("trackId")

	// 🌟 对齐字段：Select 补全 class_name 班级字段，并对齐前端 camelCase 预期的 JSON 标签
	query := global.DB.Table("checkin_record cr").
		Select("cr.id, cr.user_id, cr.track_id, cr.content_text, cr.image_url, cr.status, cr.ai_score, cr.ai_comment, cr.admin_comment, cr.publish_status, cr.checkin_date, cr.create_time, u.nickname, u.student_id, u.class_name, t.name as track_name").
		Joins("LEFT JOIN user u ON cr.user_id = u.id").
		Joins("LEFT JOIN track t ON cr.track_id = t.id").
		Where("cr.status != 4")

	if date != "" {
		query = query.Where("cr.checkin_date = ?", date)
	}
	if trackId != "" && trackId != "0" {
		query = query.Where("cr.track_id = ?", trackId)
	}

	var results []AuditRecordResp
	if err := query.Order("cr.create_time DESC").Find(&results).Error; err != nil {
		common.Error(c, "获取大盘数据失败")
		return
	}
	common.Success(c, results)
}

// TriggerAiPipeline 强制调度启动后台 AI 流水线
func TriggerAiPipeline(c *gin.Context) {
	var req struct {
		Date string `json:"date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}
	go service.TriggerAiPipelineForDate(req.Date)
	common.Success(c, "AI 流水线唤醒指令已下发")
}

// PublishScores 一键放榜
func PublishScores(c *gin.Context) {
	var req struct {
		Date    string `json:"date"`
		TrackID int    `json:"trackId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}
	go service.PublishScoresForDate(req.Date, req.TrackID)
	common.Success(c, "成绩放榜指令已下发至消息队列")
}

type DeleteCheckinReq struct {
	RecordID int64 `json:"recordId" binding:"required"`
}

// DeleteCheckinRecord 导师大盘删除打卡流水 (物理删除，支持清理孤儿数据)
func DeleteCheckinRecord(c *gin.Context) {
	var req DeleteCheckinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	// 开启物理删除事务
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 先查找该记录
		var record models.CheckinRecord
		if err := tx.First(&record, req.RecordID).Error; err != nil {
			return err
		}

		// 2. 检查此记录是否已通过并计分（Status == 1 且 PublishStatus == 1），如果计分了，且用户还在，则扣除其 empowerment_score 分数！
		if record.Status == 1 && record.PublishStatus == 1 && record.AiScore > 0 {
			var userCount int64
			tx.Table("user").Where("id = ?", record.UserID).Count(&userCount)
			if userCount > 0 {
				// 扣除积分
				if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score - ? WHERE id = ?", record.AiScore, record.UserID).Error; err != nil {
					return err
				}

				// 物理擦除分值记录，以保持总账本绝对干净！
				tx.Where("user_id = ? AND score_type = 'EMP' AND score = ? AND reason LIKE '%打卡%'", record.UserID, record.AiScore).Delete(&models.ScoreLog{})
			}
		}

		// 3. 彻底从 checkin_record 表中删除
		if err := tx.Delete(&record).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		common.Error(c, "打卡流水销毁失败: "+err.Error())
		return
	}

	common.Success(c, "该打卡流水已被物理销毁抹除")
}

// DeleteCheckinRecordSelf 学生自删打卡流水 (物理删除，安全隔离)
func DeleteCheckinRecordSelf(c *gin.Context) {
	idStr := c.Query("id")
	var recordID int64
	if idStr != "" {
		fmt.Sscanf(idStr, "%d", &recordID)
	}
	if recordID == 0 {
		var req struct {
			RecordID int64 `json:"recordId"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			recordID = req.RecordID
		}
	}

	if recordID == 0 {
		common.Error(c, "操作失败：缺少打卡记录 id")
		return
	}

	userID := c.MustGet("userId").(int64)

	// 开启物理删除事务
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 先查找该记录
		var record models.CheckinRecord
		if err := tx.First(&record, recordID).Error; err != nil {
			return err
		}

		// 横向越权检验：只允许删除本人的记录！
		if record.UserID != userID {
			return fmt.Errorf("安全阻断：无权删除他人的打卡记录！")
		}

		// 2. 检查此记录是否已通过并计分（Status == 1 且 PublishStatus == 1），如果计分了，且用户还在，则扣除其 empowerment_score 分数！
		if record.Status == 1 && record.PublishStatus == 1 && record.AiScore > 0 {
			var userCount int64
			tx.Table("user").Where("id = ?", record.UserID).Count(&userCount)
			if userCount > 0 {
				// 扣除积分
				if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score - ? WHERE id = ?", record.AiScore, record.UserID).Error; err != nil {
					return err
				}

				// 物理擦除分值记录，以保持总账本绝对干净！
				tx.Where("user_id = ? AND score_type = 'EMP' AND score = ? AND reason LIKE '%打卡%'", record.UserID, record.AiScore).Delete(&models.ScoreLog{})
			}
		}

		// 3. 彻底从 checkin_record 表中删除
		if err := tx.Delete(&record).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		common.Error(c, "打卡记录删除失败: "+err.Error())
		return
	}

	common.Success(c, "打卡记录已成功撤销并抹除")
}

// ================= 🏆 真实奖惩中心 =================

// SendReward 发放奖励/惩罚扣分
func SendReward(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "请求格式错误")
		return
	}

	userID := int64(req["userId"].(float64))
	reqType := req["type"].(string) // "emp", "moral" 或 "makeup"
	reason := req["reason"].(string)

	score := 0
	if s, ok := req["score"].(float64); ok {
		score = int(s)
	} else if v, ok := req["value"].(float64); ok {
		score = int(v)
	}

	if score == 0 {
		common.Error(c, "分数不能为0")
		return
	}

	// 🚀 刚性拦截：验证签发奖励是否击穿系统设定的单次加分上限
	var maxSingleAddStr string
	var maxSingleAdd int = 100 // 默认兜底
	if err := global.DB.Table("sys_config").Where("config_key = ?", "max_single_add").Pluck("config_value", &maxSingleAddStr).Error; err == nil && maxSingleAddStr != "" {
		fmt.Sscanf(maxSingleAddStr, "%d", &maxSingleAdd)
	}
	if score > maxSingleAdd {
		common.Error(c, fmt.Sprintf("签发奖励值已击穿系统设定的单次加分上限（单次最大可加 %d 分）", maxSingleAdd))
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 确定积分字段与类型对齐
		field := "empowerment_score"
		dbType := reqType
		if reqType == "moral" || reqType == "MORAL" {
			field = "moral_score"
			dbType = "MORAL"
		} else if reqType == "makeup" || reqType == "MAKEUP" || reqType == "card" {
			field = "makeup_cards"
			dbType = "card" // 对齐 score_log 中的 card 历史查询键
		} else {
			dbType = "EMP"
		}

		// 2. 记录加分/增卡流水
		log := models.ScoreLog{
			UserID:    userID,
			ScoreType: dbType,
			Score:     score,
			Action:    "ADD",
			Reason:    reason,
			Operator:  "系统最高指挥官",
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		// 3. 更新用户数据
		return tx.Exec("UPDATE user SET "+field+" = "+field+" + ? WHERE id = ?", score, userID).Error
	})

	if err != nil {
		common.Error(c, "奖惩操作失败: "+err.Error())
		return
	}

	common.Success(c, "奖励加分令已成功签发并记入大底账！")
}

// GetRewardHistory 获取某人的奖惩流水
func GetRewardHistory(c *gin.Context) {
	userID := c.Query("userId")
	var logs []map[string]interface{}
	// 保持原逻辑，确保score_type覆盖全面
	global.DB.Table("score_log").
		Where("user_id = ? AND score_type IN ('emp', 'moral', 'EMP', 'MORAL', 'card', 'ban')", userID).
		Order("id DESC").Find(&logs)
	common.Success(c, logs)
}

// RevokeReward 撤销某次奖惩记录，并反向扣减用户积分/卡数
func RevokeReward(c *gin.Context) {
	var req struct {
		LogID int64 `json:"logId"`
	}
	// 兼容 JSON 和 URL query
	if err := c.ShouldBindJSON(&req); err != nil || req.LogID == 0 {
		logIdStr := c.Query("logId")
		if logIdStr != "" {
			fmt.Sscanf(logIdStr, "%d", &req.LogID)
		}
	}

	if req.LogID == 0 {
		common.Error(c, "参数解析失败，LogID 缺失")
		return
	}

	var scoreLog models.ScoreLog
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 查找流水记录
		if err := tx.First(&scoreLog, req.LogID).Error; err != nil {
			return err
		}

		// 2. 只有加分（ADD）记录才能被物理撤销扣减
		if scoreLog.Action == "ADD" && scoreLog.Score > 0 {
			field := "empowerment_score"
			if scoreLog.ScoreType == "MORAL" || scoreLog.ScoreType == "moral" {
				field = "moral_score"
			} else if scoreLog.ScoreType == "card" || scoreLog.ScoreType == "makeup" {
				field = "makeup_cards"
			}
			// 反向更新用户分值
			if err := tx.Exec(fmt.Sprintf("UPDATE user SET %s = %s - ? WHERE id = ?", field, field), scoreLog.Score, scoreLog.UserID).Error; err != nil {
				return err
			}
		}

		// 3. 物理擦除流水记录
		return tx.Delete(&scoreLog).Error
	})

	if err != nil {
		common.Error(c, "物理撤回操作失败: "+err.Error())
		return
	}

	common.Success(c, "该荣誉嘉奖记录已被物理撤销，用户总分/卡数已执行反向扣减对齐！")
}

// GetRecentCheckinRecords 对应 /api/admin/checkin/recent-list 或 /api/sub-admin/checkin/recent-list (GET)
// 只读拉取最近审核通过的 50 条打卡，供前端社区大盘流展示
func GetRecentCheckinRecords(c *gin.Context) {
	var results []map[string]interface{}
	query := global.DB.Table("checkin_record cr").
		Select("cr.*, u.nickname as user_name, u.avatar as avatar_url, t.name as track_name").
		Joins("LEFT JOIN user u ON cr.user_id = u.id").
		Joins("LEFT JOIN track t ON cr.track_id = t.id").
		Where("cr.status = 1") // 已经核销放榜公布的

	if err := query.Order("cr.create_time desc").Limit(50).Scan(&results).Error; err != nil {
		common.Error(c, "获取最近打卡记录失败: "+err.Error())
		return
	}

	common.Success(c, results)
}
