package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/websocket"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetAllConfigs(c *gin.Context) {
	var configs []map[string]interface{}
	global.DB.Table("sys_config").Find(&configs)
	common.Success(c, configs)
}

type UpdateConfigReq struct {
	ConfigKey   string `json:"config_key" binding:"required"`
	ConfigValue string `json:"config_value" binding:"required"`
}

func UpdateSystemConfig(c *gin.Context) {
	var req UpdateConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数校验失败")
		return
	}

	global.DB.Exec("INSERT INTO sys_config (config_key, config_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE config_value = ?", req.ConfigKey, req.ConfigValue, req.ConfigValue)
	common.Success(c, "配置修改已实时生效！")
}

func GetSystemSettings(c *gin.Context) {
	var configs []map[string]interface{}
	global.DB.Table("sys_config").Find(&configs)

	settings := make(map[string]interface{})
	for _, v := range configs {
		if key, ok := v["config_key"].(string); ok {
			settings[key] = v["config_value"]
			
			// 🛡️ 战术映射防御：为前端及微信小程序端各类驼峰/蛇形字段期待提供双向无损支持
			if key == "checkin_deadline" {
				settings["deadline"] = v["config_value"]
			}
			if key == "track_edit_deadline" {
				settings["trackEditDeadline"] = v["config_value"]
			}
			if key == "max_moral_score" {
				settings["maxMoralScore"] = v["config_value"]
			}
			if key == "max_emp_score" {
				settings["maxEmpScore"] = v["config_value"]
			}
			if key == "max_single_add" {
				settings["maxSingleAdd"] = v["config_value"]
			}
			if key == "max_single_sub" {
				settings["maxSingleSub"] = v["config_value"]
			}
		}
	}
	common.Success(c, settings)
}

func SaveSystemSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	// 🛡️ 双重拦截映射：支持前端发送驼峰、蛇形、或是微信小程序端原装物理键名
	keyMapping := map[string]string{
		"deadline":            "checkin_deadline", // 🌟 刚性拦截：对齐 checkin_deadline
		"checkin_deadline":    "checkin_deadline",
		"trackEditDeadline":   "track_edit_deadline",
		"track_edit_deadline": "track_edit_deadline",
		"maxMoralScore":       "max_moral_score",
		"max_moral_score":     "max_moral_score",
		"maxEmpScore":         "max_emp_score",
		"max_emp_score":       "max_emp_score",
		"maxSingleAdd":        "max_single_add",
		"max_single_add":      "max_single_add",
		"maxSingleSub":        "max_single_sub",
		"max_single_sub":      "max_single_sub",
	}

	for k, v := range req {
		dbKey := k
		if mappedKey, exists := keyMapping[k]; exists {
			dbKey = mappedKey
		}
		global.DB.Exec("INSERT INTO sys_config (config_key, config_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE config_value = ?", dbKey, v, v)
	}

	global.Redis.Del(context.Background(), "system_settings_cache", "sys_configs")
	common.Success(c, "全局安全配置已更新并清空旧缓存！")
}

func SaveAppealCategory(c *gin.Context) {
	var req models.AppealCategory
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "分类参数解析失败: "+err.Error())
		return
	}

	if req.ID > 0 {
		// 修改逻辑：使用强模型反写
		if err := global.DB.Model(&models.AppealCategory{}).Where("id = ?", req.ID).Updates(&req).Error; err != nil {
			common.Error(c, "分类更正失败: "+err.Error())
			return
		}
	} else {
		// 新建逻辑
		req.ID = 0
		if err := global.DB.Create(&req).Error; err != nil {
			common.Error(c, "分类添加失败: "+err.Error())
			return
		}
	}
	common.Success(c, "分类已实时同步至数据库")
}

func DeleteAppealCategory(c *gin.Context) {
	// ⚡ 完美兼容：优先读取 URL Query 传参 `?id=xxx`，如为空再 fallback 绑定 JSON
	idStr := c.Query("id")
	var id int
	if idStr != "" {
		fmt.Sscanf(idStr, "%d", &id)
	}
	if id == 0 {
		var req struct {
			ID int `json:"id"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			id = req.ID
		}
	}

	if id == 0 {
		common.Error(c, "操作失败：缺少分类唯一主键标识 id")
		return
	}

	// 联级彻底抹除该类别及其下级所有细分分类
	if err := global.DB.Table("appeal_category").Where("id = ? OR parent_id = ?", id, id).Delete(nil).Error; err != nil {
		common.Error(c, "删除分类失败: "+err.Error())
		return
	}

	common.Success(c, "类别及其下属子树已被彻底抹除")
}

type ResetPasswordPayload struct {
	NewPassword string `json:"newPassword"`
	AdminID     string `json:"adminId"`
	NewUsername string `json:"newUsername"`
}

func ResetPassword(c *gin.Context) {
	var req ResetPasswordPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	if req.NewUsername != "" {
		global.DB.Exec("INSERT INTO sys_config (config_key, config_value) VALUES ('admin_username', ?) ON DUPLICATE KEY UPDATE config_value = ?", req.NewUsername, req.NewUsername)
	}

	if req.NewPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			common.Error(c, "密码加固加密失败")
			return
		}
		global.DB.Exec("INSERT INTO sys_config (config_key, config_value) VALUES ('admin_password', ?) ON DUPLICATE KEY UPDATE config_value = ?", string(hashedPassword), string(hashedPassword))
	}

	common.Success(c, "安全凭证全量加盐加密覆写完成，已实时落库")
}

// ================= 🛡️ 真实风控与人员管理核心引擎 =================

func KickUser(c *gin.Context) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	userIdStr := c.Query("userId")
	if userIdStr != "" {
		fmt.Sscanf(userIdStr, "%d", &req.UserID)
	} else {
		c.ShouldBindJSON(&req)
	}

	if req.UserID == 0 {
		common.Error(c, "解析下线指令参数失败")
		return
	}

	ctx := context.Background()
	// 1. 清空 Redis 会话令牌
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", req.UserID))
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", req.UserID))
	keys, _ := global.Redis.Keys(ctx, fmt.Sprintf("user_device:%d:*", req.UserID)).Result()
	if len(keys) > 0 {
		global.Redis.Del(ctx, keys...)
	}

	// 2. 广播 WS 下线消息
	websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeKickOut, map[string]interface{}{
		"kickType": "FORCE_LOGOUT",
		"reason":   "您的账号已被管理员强制下线，请重新登录。",
	})

	common.Success(c, "强制下机指令已执行，该设备已被踢出！")
}

func ToggleBlacklist(c *gin.Context) {
	var req struct {
		UserID int64 `json:"userId"`
		Status int   `json:"status"` // 2 = 拉黑设备, 0 = 恢复
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		common.Error(c, "参数解析失败")
		return
	}

	var user models.User
	if err := global.DB.First(&user, req.UserID).Error; err != nil {
		common.Error(c, "未找到该用户")
		return
	}

	ctx := context.Background()
	if req.Status == 2 {
		if user.CurrentDeviceID == "" {
			common.Error(c, "该用户尚未产生物理设备绑定线索，无法执行设备拉黑")
			return
		}
		// 1. 物理拉黑当前设备
		global.Redis.Set(ctx, "ban_device:"+user.CurrentDeviceID, "BANNED", 3650*24*time.Hour)

		// 2. 更新数据库状态为封禁，并剥离设备绑定
		global.DB.Table("user").Where("id = ?", req.UserID).Updates(map[string]interface{}{
			"account_status":    2,
			"current_device_id": "",
		})

		// 3. 清理 Redis 会话
		global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", req.UserID))
		global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", req.UserID))

		// 4. WebSocket 广播强驱隔离
		websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeKickOut, map[string]interface{}{
			"banType": 4, // 硬件拉黑
			"reason":  "由于设备涉嫌违规代打卡作弊，当前设备已被物理列入系统黑名单锁死。",
		})

		common.Success(c, "该设备已被物理拉黑，永久禁止登录系统！")
	} else {
		// 恢复解封
		global.DB.Table("user").Where("id = ?", req.UserID).Update("account_status", 0)
		common.Success(c, "设备及账号封禁已恢复解封！")
	}
}

func RevokePunish(c *gin.Context) {
	logId := c.Query("logId")
	var scoreLog models.ScoreLog

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&scoreLog, logId).Error; err != nil {
			return err
		}

		if scoreLog.Action == "SUB" || scoreLog.Score < 0 {
			absScore := int(math.Abs(float64(scoreLog.Score)))
			field := "empowerment_score"
			if scoreLog.ScoreType == "MORAL" || scoreLog.ScoreType == "moral" {
				field = "moral_score"
			}
			if scoreLog.ScoreType != "ban" {
				if err := tx.Exec(fmt.Sprintf("UPDATE user SET %s = %s + ? WHERE id = ?", field, field), absScore, scoreLog.UserID).Error; err != nil {
					return err
				}
			}
		}

		return tx.Delete(&scoreLog).Error
	})

	if err != nil {
		common.Error(c, "惩戒撤回事务失败: "+err.Error())
		return
	}

	userIdStr := fmt.Sprintf("%d", scoreLog.UserID)
	ctx := context.Background()
	global.Redis.Del(ctx, "ban_user:"+userIdStr)
	global.Redis.Del(ctx, "ban_status:"+userIdStr)

	global.DB.Table("user").Where("id = ?", scoreLog.UserID).Update("account_status", 0)
	websocket.Manager.BroadcastToUser(scoreLog.UserID, websocket.MsgTypeRestore, nil)

	common.Success(c, "该纪律干预惩戒已成功物理撤销，数值与系统权限已全额恢复对齐！")
}

func DeleteUser(c *gin.Context) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	userIdStr := c.Query("userId")
	if userIdStr != "" {
		fmt.Sscanf(userIdStr, "%d", &req.UserID)
	} else {
		c.ShouldBindJSON(&req)
	}

	if req.UserID == 0 {
		common.Error(c, "解析删除指令参数失败")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		tx.Table("user").Where("id = ?", req.UserID).Delete(nil)
		tx.Table("user_track").Where("user_id = ?", req.UserID).Delete(nil)
		tx.Table("checkin_record").Where("user_id = ?", req.UserID).Delete(nil)
		tx.Table("score_log").Where("user_id = ?", req.UserID).Delete(nil)
		tx.Table("appeal_record").Where("user_id = ?", req.UserID).Delete(nil)
		tx.Table("user_purchase_record").Where("user_id = ?", req.UserID).Delete(nil)
		return nil
	})

	if err != nil {
		common.Error(c, "级联物理抹除大事务失败: "+err.Error())
		return
	}

	ctx := context.Background()
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", req.UserID))
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", req.UserID))
	websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeKickOut, map[string]interface{}{
		"kickType": "FORCE_LOGOUT",
		"reason":   "您的账号已被管理员从系统中彻底物理抹除注销。",
	})

	common.Success(c, "该生账户及关联打卡流水底账已被彻底连根物理抹除蒸发！")
}

func RejectCancel(c *gin.Context) {
	userId := c.Query("userId")
	global.DB.Model(&models.User{}).Where("id = ?", userId).Update("account_status", 0)
	common.Success(c, "已驳回注销申请，账号恢复可用状态！")
}

func AdminResetPassword(c *gin.Context) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		common.Error(c, "更正密码参数残缺")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		common.Error(c, "新凭证加密哈希失败")
		return
	}

	if err := global.DB.Table("user").Where("id = ?", req.UserID).Update("password", string(hashedPassword)).Error; err != nil {
		common.Error(c, "密码更正保存数据库失败")
		return
	}

	ctx := context.Background()
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", req.UserID))
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", req.UserID))
	keys, _ := global.Redis.Keys(ctx, fmt.Sprintf("user_device:%d:*", req.UserID)).Result()
	if len(keys) > 0 {
		global.Redis.Del(ctx, keys...)
	}

	resetToken := fmt.Sprintf("admin_reset_%d_%d", req.UserID, time.Now().UnixNano())
	global.Redis.Set(ctx, "auth_reset:"+resetToken, req.UserID, 15*time.Minute)

	websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeKickOut, map[string]interface{}{
		"kickType":   "RESET_PASSWORD",
		"resetToken": resetToken,
		"reason":     "您的登录密码已被管理员重设，初始密码为 123456，请立即设置属于您的安全密码！",
	})

	common.Success(c, "密码已强置为 123456，且相关在线终端已被强退引导至重设界面！")
}

type PunishReq struct {
	UserID   int64  `json:"userId"`
	Type     string `json:"type"` // "emp", "moral", "ban"
	Score    int    `json:"score"`
	Duration string `json:"duration"` // "5m", "10m", "30m", "24h", "forever"
	Reason   string `json:"reason"`
}

func ExecutePunish(c *gin.Context) {
	var req PunishReq
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		common.Error(c, "指令解析失败，数据格式错误")
		return
	}

	if req.Reason == "" {
		common.Error(c, "干预纪律惩戒原因必须填写以资存证")
		return
	}

	ctx := context.Background()

	if req.Type == "ban" {
		var expireTime time.Duration
		switch req.Duration {
		case "5m":
			expireTime = 5 * time.Minute
		case "10m":
			expireTime = 10 * time.Minute
		case "30m":
			expireTime = 30 * time.Minute
		case "24h":
			expireTime = 24 * time.Hour
		case "forever", "perm", "":
			expireTime = 3650 * 24 * time.Hour
		default:
			expireTime = 3650 * 24 * time.Hour
		}

		if err := global.DB.Table("user").Where("id = ?", req.UserID).Update("account_status", 2).Error; err != nil {
			common.Error(c, "封禁失败：数据库状态同步失败")
			return
		}

		global.DB.Table("score_log").Create(map[string]interface{}{
			"user_id":    req.UserID,
			"score_type": "ban",
			"score":      0,
			"action":     "SUB",
			"reason":     req.Reason,
			"operator":   "系统最高指挥官",
		})

		banUserKey := fmt.Sprintf("ban_user:%d", req.UserID)
		banStatusKey := fmt.Sprintf("ban_status:%d", req.UserID)
		global.Redis.Set(ctx, banUserKey, req.Reason, expireTime)
		global.Redis.Set(ctx, banStatusKey, 1, expireTime)

		websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeKickOut, map[string]interface{}{
			"banType":    2,
			"banEndTime": time.Now().Add(expireTime).UnixMilli(),
			"reason":     req.Reason,
		})

		common.Success(c, "纪律封禁锁定令下发成功！目标用户已被强制封锁！")
		return
	}

	if req.Score <= 0 {
		common.Error(c, "扣除分值必须大于0")
		return
	}

	// 🚀 刚性拦截：验证惩戒扣分是否击穿系统设定的单次扣分上限
	var maxSingleSubStr string
	var maxSingleSub int = 100 // 默认兜底
	if err := global.DB.Table("sys_config").Where("config_key = ?", "max_single_sub").Pluck("config_value", &maxSingleSubStr).Error; err == nil && maxSingleSubStr != "" {
		fmt.Sscanf(maxSingleSubStr, "%d", &maxSingleSub)
	}
	if req.Score > maxSingleSub {
		common.Error(c, fmt.Sprintf("惩戒分值已击穿系统设定的扣分上限（单次最大可扣 %d 分）", maxSingleSub))
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		log := models.ScoreLog{
			UserID:    req.UserID,
			ScoreType: req.Type,
			Score:     -req.Score,
			Action:    "SUB",
			Reason:    req.Reason,
			Operator:  "系统最高指挥官",
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		field := "empowerment_score"
		if req.Type == "moral" || req.Type == "MORAL" {
			field = "moral_score"
		}
		return tx.Exec(fmt.Sprintf("UPDATE user SET %s = %s - ? WHERE id = ?", field, field), req.Score, req.UserID).Error
	})

	if err != nil {
		common.Error(c, "扣分操作执行失败: "+err.Error())
		return
	}

	common.Success(c, "扣分惩戒令已强制注入执行！")
}

func KickUserBatch(c *gin.Context) {
	var req struct {
		UserIDs []int64 `json:"userIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UserIDs) == 0 {
		common.Error(c, "解析批量踢除参数失败")
		return
	}

	ctx := context.Background()
	for _, userID := range req.UserIDs {
		global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", userID))
		global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", userID))
		keys, _ := global.Redis.Keys(ctx, fmt.Sprintf("user_device:%d:*", userID)).Result()
		if len(keys) > 0 {
			global.Redis.Del(ctx, keys...)
		}

		websocket.Manager.BroadcastToUser(userID, websocket.MsgTypeKickOut, map[string]interface{}{
			"kickType": "FORCE_LOGOUT",
			"reason":   "您的会话已被管理员批量强制踢下线。",
		})
	}
	common.Success(c, "批量强制作退下线已全部注入完成！")
}

func BanUserBatch(c *gin.Context) {
	var req struct {
		UserIDs []int64 `json:"userIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.UserIDs) == 0 {
		common.Error(c, "解析批量封禁参数失败")
		return
	}

	duration := 5 * time.Minute
	ctx := context.Background()
	endTime := time.Now().Add(duration).UnixMilli()

	for _, userID := range req.UserIDs {
		global.DB.Table("user").Where("id = ?", userID).Update("account_status", 2)
		global.Redis.Set(ctx, fmt.Sprintf("ban_user:%d", userID), "管理员批量临时防线熔断", duration)
		global.Redis.Set(ctx, fmt.Sprintf("ban_status:%d", userID), 1, duration)

		websocket.Manager.BroadcastToUser(userID, websocket.MsgTypeKickOut, map[string]interface{}{
			"banType":    2,
			"banEndTime": endTime,
			"reason":     "管理员一键安全干预，您的账号已被临时熔断隔离 5 分钟！",
		})
	}
	common.Success(c, "选中的人群临时熔断防御已执行！")
}

func GlobalKick(c *gin.Context) {
	var allUsers []models.User
	global.DB.Where("user_type != 2").Find(&allUsers)

	ctx := context.Background()
	for _, u := range allUsers {
		global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", u.ID))
		global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", u.ID))
		keys, _ := global.Redis.Keys(ctx, fmt.Sprintf("user_device:%d:*", u.ID)).Result()
		if len(keys) > 0 {
			global.Redis.Del(ctx, keys...)
		}

		websocket.Manager.BroadcastToUser(u.ID, websocket.MsgTypeKickOut, map[string]interface{}{
			"kickType": "FORCE_LOGOUT",
			"reason":   "全服紧急应急疏散：由于系统进行安全维护，您已被管理员强制下机作退，请稍后重新进入系统！",
		})
	}
	common.Success(c, "全网紧急下线强退信号已向全体非管理员广播送达！")
}

func GlobalBan(c *gin.Context) {
	var req struct {
		Status bool `json:"status"`
	}
	c.ShouldBindJSON(&req)

	var allUsers []models.User
	global.DB.Where("user_type != 2").Find(&allUsers)

	ctx := context.Background()
	duration := 5 * time.Minute
	endTime := time.Now().Add(duration).UnixMilli()

	for _, u := range allUsers {
		if req.Status {
			global.DB.Table("user").Where("id = ?", u.ID).Update("account_status", 2)
			global.Redis.Set(ctx, fmt.Sprintf("ban_user:%d", u.ID), "系统一键应急全局休克隔离", duration)
			global.Redis.Set(ctx, fmt.Sprintf("ban_status:%d", u.ID), 1, duration)

			websocket.Manager.BroadcastToUser(u.ID, websocket.MsgTypeKickOut, map[string]interface{}{
				"banType":    2,
				"banEndTime": endTime,
				"reason":     "系统全局临时应急隔离防线启动，全服非管理员用户熔断 5 分钟！",
			})
		} else {
			global.DB.Table("user").Where("id = ?", u.ID).Update("account_status", 0)
			global.Redis.Del(ctx, fmt.Sprintf("ban_user:%d", u.ID))
			global.Redis.Del(ctx, fmt.Sprintf("ban_status:%d", u.ID))
			websocket.Manager.BroadcastToUser(u.ID, websocket.MsgTypeRestore, nil)
		}
	}

	common.Success(c, "全服一键业务熔断/恢复隔离机制指令已触发成功！")
}

// 👑 [EMERGENCY CONTROL CENTER] 独立特权带外原子管理网关核心实现 (新增)
type EmergencyControlReq struct {
	Action    string `json:"action" binding:"required"`
	SecureKey string `json:"secure_key" binding:"required"`
	StudentID string `json:"student_id"`
	Password  string `json:"password"`
	RealName  string `json:"real_name"`
	ClassName string `json:"class_name"`
}

func EmergencyCommanderCenter(c *gin.Context) {
	var req EmergencyControlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "请求载荷参数解析失败")
		return
	}

	// 🔒 全局防伪锁安全暗号审计
	if req.SecureKey != "XUANFU-MFA-TEST-99" {
		common.Error(c, "安全核对失败：全局安全防护锁暗号不正确！")
		return
	}

	// 📡 分流一：获取大管理员名册列表
	if req.Action == "list" {
		var admins []struct {
			StudentID string `json:"student_id" gorm:"column:student_id"`
			RealName  string `json:"real_name" gorm:"column:real_name"`
			ClassName string `json:"class_name" gorm:"column:class_name"`
		}
		if err := global.DB.Table("user").Where("user_type = 2 AND account_status = 0").Find(&admins).Error; err != nil {
			common.Error(c, "数据库名册检索遭遇故障")
			return
		}
		common.Success(c, admins)
		return
	}

	// 📡 分流二：物理覆盖写入 (添加与修改)
	if req.Action == "save" {
		if req.StudentID == "" || req.RealName == "" {
			common.Error(c, "账号(学号)与真实姓名不能为空")
			return
		}

		var err error
		var hashedPassword string

		// 若携带了密码参数，激活高并发特征 BCrypt 哈希引擎加固
		if req.Password != "" {
			bytes, errHash := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if errHash != nil {
				common.Error(c, "底层加密引擎哈希算力异常")
				return
			}
			hashedPassword = string(bytes)
		}

		if hashedPassword != "" {
			// 包含密码重置/更正的全原子插入更新
			err = global.DB.Exec(`
				INSERT INTO user (student_id, password, real_name, nickname, class_name, user_type, account_status, create_time) 
				VALUES (?, ?, ?, ?, ?, 2, 0, NOW()) 
				ON DUPLICATE KEY UPDATE 
					password = ?, real_name = ?, nickname = ?, class_name = ?, user_type = 2, account_status = 0
			`, req.StudentID, hashedPassword, req.RealName, req.RealName, req.ClassName, hashedPassword, req.RealName, req.RealName, req.ClassName).Error
		} else {
			// 仅更正基本学籍属性的覆盖写入
			err = global.DB.Exec(`
				INSERT INTO user (student_id, password, real_name, nickname, class_name, user_type, account_status, create_time) 
				VALUES (?, 'NO_PWD_OVERRIDE_INIT', ?, ?, ?, 2, 0, NOW()) 
				ON DUPLICATE KEY UPDATE 
					real_name = ?, nickname = ?, class_name = ?, user_type = 2, account_status = 0
			`, req.StudentID, req.RealName, req.RealName, req.ClassName, req.RealName, req.RealName, req.ClassName).Error
		}

		if err != nil {
			common.Error(c, "保存大事务回滚，数据库拒绝落库: "+err.Error())
			return
		}
		common.Success(c, "大管理员特权账户数据已成功安全落库！")
		return
	}

	// 📡 分流三：特权权限抹除 (安全降权机制)
	if req.Action == "delete" {
		if req.StudentID == "" {
			common.Error(c, "特权降级必须指定明确的目标学号")
			return
		}

		// 物理降权：直接将 user_type 斩断归零，移出指挥官序列
		err := global.DB.Table("user").Where("student_id = ?", req.StudentID).Updates(map[string]interface{}{
			"user_type": 0,
		}).Error

		if err != nil {
			common.Error(c, "撤销指挥官权限大事务失败: "+err.Error())
			return
		}
		common.Success(c, "目标账户的大管理员特权已被强制剥离拔除！")
		return
	}

	common.Error(c, "未知的战术 Action 执行指令")
}
