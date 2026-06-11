package models

import "time"

// AppealRecord 统一工单表 (申诉/反馈/建议)
type AppealRecord struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64      `gorm:"not null;index" json:"userId"`
	AppealType   string     `gorm:"size:100;default:'其他'" json:"appealType"`
	TrackName    string     `gorm:"size:50" json:"trackName"`         // 预留：关联赛道
	AbnormalDate string     `gorm:"size:20" json:"abnormalDate"`      // 预留：异常日期
	Reason       string     `gorm:"type:text;not null" json:"reason"` // 申诉/Bug详情
	ImageURL     string     `gorm:"type:text" json:"imageUrl"`
	Status       int8       `gorm:"default:0" json:"status"` // 0=待审核 1=通过/采纳 2=驳回/忽略
	Reply        string     `gorm:"size:500" json:"reply"`   // 官方回复
	HandlerName  string     `gorm:"size:50" json:"handlerName"`
	RewardText   string     `gorm:"size:100" json:"rewardText"` // 奖励描述
	HandleTime   *time.Time `json:"handleTime"`
	CreateTime   time.Time  `gorm:"autoCreateTime" json:"createTime"`
}

func (AppealRecord) TableName() string { return "appeal_record" }

// AppealCategory 动态工单分类表 (对接管理后台)
type AppealCategory struct {
	ID           int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"size:50;not null" json:"name"`
	ParentID     int    `gorm:"not null;default:0" json:"parentId"`
	RequireImage int8   `gorm:"not null;default:0" json:"requireImage"` // 1=必填截图
	SortOrder    int    `gorm:"default:0" json:"sortOrder"`
}

func (AppealCategory) TableName() string { return "appeal_category" }
