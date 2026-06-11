package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"time"

	"github.com/gin-gonic/gin"
)

// 💡 补充消息结构体（可以直接放在这里，也可以移到 models/system_message.go）
type SystemMessage struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"index" json:"userId"`
	Type       string    `gorm:"size:50" json:"type"` // system, appeal, bug, checkin
	Title      string    `gorm:"size:100" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	IsRead     int       `gorm:"default:0" json:"isRead"`
	CreateTime time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (SystemMessage) TableName() string { return "system_message" }

func GetUnreadMessageCount(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	var count int64
	global.DB.Table("system_message").Where("user_id = ? AND is_read = 0", userID).Count(&count)
	common.Success(c, gin.H{"count": count})
}

func GetUserMessages(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	var messages []SystemMessage
	global.DB.Where("user_id = ?", userID).Order("create_time desc").Find(&messages)
	common.Success(c, messages)
}

func ReadAllMessages(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	global.DB.Table("system_message").Where("user_id = ? AND is_read = 0", userID).Update("is_read", 1)
	common.Success(c, "全部标为已读")
}
