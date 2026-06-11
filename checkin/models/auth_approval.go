package models

import "time"

// AuthApproval 授权与体验账户审批队列表
type AuthApproval struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null;index" json:"userId"`
	ApplyType  string    `gorm:"size:50;not null" json:"applyType"` // RESET_PWD=重置密码, EXPERIENCE=体验账户
	Status     int8      `gorm:"default:0" json:"status"`           // 0=待审批, 1=同意, 2=拒绝
	DeviceID   string    `gorm:"size:255" json:"deviceId"`
	CreateTime time.Time `gorm:"autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"autoUpdateTime" json:"updateTime"`
}

func (AuthApproval) TableName() string { return "auth_approval" }
