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

// ==================== 1. 学生端：动态分类与工单提交 ====================

// GetAppealCategories 动态拉取后台配置的申诉/Bug分类
func GetAppealCategories(c *gin.Context) {
	var categories []models.AppealCategory
	if err := global.DB.Order("sort_order asc, id asc").Find(&categories).Error; err != nil {
		common.Error(c, "获取申诉分类失败: "+err.Error())
		return
	}
	if categories == nil {
		categories = []models.AppealCategory{}
	}
	common.Success(c, categories)
}

type SubmitAppealReq struct {
	ID         int64  `json:"id"`
	AppealType string `json:"appealType"`
	Reason     string `json:"reason"`
	ImageURL   string `json:"imageUrl"`
}

// SubmitAppeal 统一的工单提交/修改入口
func SubmitAppeal(c *gin.Context) {
	var req SubmitAppealReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析异常")
		return
	}

	userID := c.MustGet("userId").(int64)

	if req.ID > 0 {
		// 修改逻辑
		var existing models.AppealRecord
		if err := global.DB.Where("id = ? AND user_id = ? AND status = 0", req.ID, userID).First(&existing).Error; err != nil {
			common.Error(c, "工单不存在或正在处理中，无法修改")
			return
		}
		global.DB.Model(&existing).Updates(map[string]interface{}{
			"reason":    req.Reason,
			"image_url": req.ImageURL,
		})
		common.Success(c, "工单内容已更新")
	} else {
		// 新建逻辑
		record := models.AppealRecord{
			UserID:     userID,
			AppealType: req.AppealType,
			Reason:     req.Reason,
			ImageURL:   req.ImageURL,
			Status:     0,
		}
		if err := global.DB.Create(&record).Error; err != nil {
			common.Error(c, "工单提交失败")
			return
		}
		common.Success(c, "工单已成功提交，请留意历史大厅进度！")
	}
}

// GetAppealList 拉取个人历史工单 (修复表名脱节Bug)
func GetAppealList(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	var appeals []models.AppealRecord
	global.DB.Where("user_id = ?", userID).Order("id desc").Find(&appeals)

	var results []map[string]interface{}
	for _, app := range appeals {
		handleTimeStr := ""
		if app.HandleTime != nil {
			handleTimeStr = app.HandleTime.Format("2006-01-02 15:04")
		}
		results = append(results, map[string]interface{}{
			"id":          app.ID,
			"appealType":  app.AppealType,
			"reason":      app.Reason,
			"imageUrl":    app.ImageURL,
			"status":      app.Status,
			"reply":       app.Reply,
			"handlerName": app.HandlerName,
			"rewardText":  app.RewardText,
			"appealTime":  app.CreateTime.Format("2006-01-02 15:04"),
			"handleTime":  handleTimeStr,
		})
	}
	common.Success(c, results)
}

// RevokeAppeal 撤销/物理删除未处理的工单
func RevokeAppeal(c *gin.Context) {
	var req struct {
		ID int64 `json:"id"`
	}
	c.ShouldBindJSON(&req)
	userID := c.MustGet("userId").(int64)

	res := global.DB.Where("id = ? AND user_id = ? AND status = 0", req.ID, userID).Delete(&models.AppealRecord{})
	if res.RowsAffected == 0 {
		common.Error(c, "撤销失败，工单可能已处于处理阶段")
		return
	}
	common.Success(c, "工单已物理注销")
}

// ==================== 2. 管理员处理端 ====================

type ProcessTicketReq struct {
	AppealID    int64  `json:"appeal_id" binding:"required"`
	Status      int8   `json:"status" binding:"required"`
	Reply       string `json:"reply" binding:"required"`
	RewardScore int    `json:"reward_score"`
}

// ProcessAppeal 统一工单裁决与发奖
func ProcessAppeal(c *gin.Context) {
	var req ProcessTicketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "请求参数校验失败")
		return
	}

	var appeal models.AppealRecord
	if err := global.DB.First(&appeal, req.AppealID).Error; err != nil {
		common.Error(c, "目标工单不存在")
		return
	}

	var user models.User
	global.DB.Select("id, openid, nickname, class_name").First(&user, appeal.UserID)

	adminName := "系部总指挥官" // 未来从 Admin JWT 解析

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		appeal.Status = req.Status
		appeal.Reply = req.Reply
		appeal.HandlerName = adminName

		now := time.Now()
		appeal.HandleTime = &now

		if req.RewardScore > 0 {
			appeal.RewardText = fmt.Sprintf("%d 赋能分", req.RewardScore)
		}
		if err := tx.Save(&appeal).Error; err != nil {
			return err
		}

		// 如果通过采纳且带有奖励分数，安全写入流水
		if req.RewardScore > 0 && req.Status == 1 {
			if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", req.RewardScore, user.ID).Error; err != nil {
				return err
			}

			sl := models.ScoreLog{
				UserID:    user.ID,
				ScoreType: "EMP", // 💡 严格遵守统一日志类型
				Action:    "ADD",
				Score:     req.RewardScore,
				Reason:    "工单处理追加奖励补偿",
				Operator:  adminName,
			}
			if err := tx.Create(&sl).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "处理工单事务回滚失败")
		return
	}

	statusStr := "已驳回/忽略"
	if req.Status == 1 {
		statusStr = "已同意/采纳"
	}

	if user.OpenID != "" {
		go service.NotifyAppealResult(user.ID, user.OpenID, statusStr, req.Reply)
	}

	common.Success(c, "工单处理成功，回执与奖励已实时清算！")
}

// GetAdminAppealList 获取全站所有工单记录
func GetAdminAppealList(c *gin.Context) {
	var appeals []models.AppealRecord
	if err := global.DB.Order("id DESC").Find(&appeals).Error; err != nil {
		common.Error(c, "获取工单大厅数据失败")
		return
	}
	common.Success(c, appeals)
}

// DeleteAppeal 彻底抹除工单记录（物理删除）
func DeleteAppeal(c *gin.Context) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}
	if err := global.DB.Exec("DELETE FROM appeal_record WHERE id = ?", req.ID).Error; err != nil {
		common.Error(c, "工单抹除失败")
		return
	}
	common.Success(c, "工单已物理抹除")
}
