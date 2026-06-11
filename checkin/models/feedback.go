package models

import "time"

// BugReport Bug反馈表 (AppealRecord已经在ticket.go中，这里不再重复定义)
type BugReport struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"not null" json:"user_id"`
	Description string    `gorm:"type:text;not null" json:"description"`
	ImageUrls   string    `gorm:"type:text" json:"image_urls"` // 存放阿里云OSS的图片链接，多个用逗号隔开
	Status      int8      `gorm:"default:0" json:"status"`     // 0=未修复, 1=已修复
	CreateTime  time.Time `gorm:"autoCreateTime" json:"create_time"`
}

func (BugReport) TableName() string { return "bug_report" }
