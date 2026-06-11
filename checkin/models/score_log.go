package models

import "time"

// ScoreLog 积分变动流水档案表 (赋能分/德育分)
type ScoreLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null;index" json:"userId"`
	ScoreType  string    `gorm:"size:20;not null" json:"type"`            // 积分类型: EMP (赋能分), MORAL (德育分), EXCHANGE(商城兑换)
	Score      int       `gorm:"not null" json:"score"`                   // 变动分值 (带正负号，如 1 或 -1)
	Action     string    `gorm:"size:20" json:"action"`                   // 动作: add / sub
	Reason     string    `gorm:"size:255" json:"reason"`                  // 变动事由 (如：打卡奖励、商城兑换、违规扣分)
	Operator   string    `gorm:"size:100;default:'系统自动'" json:"operator"` // 操作人 (真实姓名或系统自动)
	CreateTime time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (ScoreLog) TableName() string { return "score_log" }
