package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/websocket"
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type BanUserReq struct {
	UserID    int64  `json:"userId"`
	Action    string `json:"action"`
	DeviceID  string `json:"deviceId"`
	Duration  string `json:"duration"`
	BanDevice bool   `json:"banDevice"`
	Reason    string `json:"reason"`
}

func AdminBanUser(c *gin.Context) {
	var req BanUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "指令解析失败，数据格式错误")
		return
	}

	ctx := context.Background()

	if req.Action == "unban" {
		if err := global.DB.Table("user").Where("id = ?", req.UserID).Update("account_status", 0).Error; err != nil {
			common.Error(c, "解封失败：数据库状态同步失败")
			return
		}
		global.Redis.Del(ctx, fmt.Sprintf("ban_status:%d", req.UserID))
		global.Redis.Del(ctx, fmt.Sprintf("ban_user:%d", req.UserID))

		// 💡 修复：向流水表记录解封事件
		global.DB.Table("score_log").Create(map[string]interface{}{
			"user_id":    req.UserID,
			"score_type": "unban",
			"score":      0,
			"reason":     "账号已解封",
			"operator":   "系统最高指挥官",
		})

		common.Success(c, "账号已恢复正常状态！")
		return
	}

	var expireTime time.Duration
	switch req.Duration {
	case "1m":
		expireTime = 1 * time.Minute
	case "5m":
		expireTime = 5 * time.Minute
	case "10m":
		expireTime = 10 * time.Minute
	case "30m":
		expireTime = 30 * time.Minute
	case "1h":
		expireTime = 1 * time.Hour
	case "12h":
		expireTime = 12 * time.Hour
	case "24h":
		expireTime = 24 * time.Hour
	case "forever", "perm", "":
		expireTime = 0
	default:
		expireTime = 0
	}

	if err := global.DB.Table("user").Where("id = ?", req.UserID).Update("account_status", 2).Error; err != nil {
		common.Error(c, "封禁失败：数据库状态同步失败")
		return
	}

	// 💡 修复：向流水表插入一条封禁事件，这样前端列表就能查到了！
	global.DB.Table("score_log").Create(map[string]interface{}{
		"user_id":    req.UserID,
		"score_type": "ban",
		"score":      0,
		"reason":     req.Reason,
		"operator":   "系统最高指挥官",
	})

	banUserKey := fmt.Sprintf("ban_user:%d", req.UserID)
	banStatusKey := fmt.Sprintf("ban_status:%d", req.UserID)
	if expireTime > 0 {
		global.Redis.Set(ctx, banUserKey, req.Reason, expireTime)
		global.Redis.Set(ctx, banStatusKey, 1, expireTime)
	} else {
		global.Redis.Set(ctx, banUserKey, req.Reason, 3650*24*time.Hour)
		global.Redis.Set(ctx, banStatusKey, 1, 3650*24*time.Hour)
	}

	if req.BanDevice && req.DeviceID != "" {
		banDeviceKey := fmt.Sprintf("ban_device:%s", req.DeviceID)
		global.Redis.Set(ctx, banDeviceKey, "该设备已被系统拉黑", 3650*24*time.Hour)
	}

	kickMsg := "您的账号已被强制下线。"
	if req.Reason != "" {
		kickMsg += fmt.Sprintf("原因：%s。", req.Reason)
	}
	websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeKickOut, kickMsg)

	common.Success(c, "封禁执行成功！目标已被实时踢出。")
}

type SendSiteMessageReq struct {
	UserID  int64  `json:"user_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func AdminSendSiteMessage(c *gin.Context) {
	var req SendSiteMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	// 💡 修复致命漏洞：必须先将站内信落库，防止离线消息丢失！
	msg := map[string]interface{}{
		"user_id": req.UserID,
		"type":    "system",
		"title":   req.Title,
		"content": req.Content,
		"is_read": 0,
	}
	// 假设你的系统表名就是 system_message
	if err := global.DB.Table("system_message").Create(msg).Error; err != nil {
		common.Error(c, "站内信落库失败")
		return
	}

	// 落库成功后，再触发 Websocket 实时送达弹窗
	noticeData := map[string]string{
		"title":   req.Title,
		"content": req.Content,
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	}
	websocket.Manager.BroadcastToUser(req.UserID, websocket.MsgTypeSystemNotice, noticeData)
	common.Success(c, "站内信已持久化存储并实时投递！")
}
