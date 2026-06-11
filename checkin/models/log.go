package models

import "time"

// ShopItem 积分商城商品表 (增强版：支持限购、限时秒杀)
type ShopItem struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string     `gorm:"size:100;not null" json:"name"`
	Type         string     `gorm:"size:20;default:'moral'" json:"type"` // moral, empowerment, makeup_card
	EmpCost      int        `gorm:"not null" json:"emp_cost"`
	RewardValue  int        `gorm:"not null" json:"reward_value"`
	Stock        int        `gorm:"default:-1" json:"stock"`            // -1为无限
	LimitPerUser int        `gorm:"default:0" json:"limit_per_user"`    // 每人限领/限购数量，0为不限
	StartTime    *time.Time `json:"start_time"`                         // 限时上架开始时间
	EndTime      *time.Time `json:"end_time"`                           // 限时上架结束时间
	IsFlashSale  bool       `gorm:"default:false" json:"is_flash_sale"` // 是否为秒杀礼包
	ReleaseTime  *time.Time `json:"release_time"`
	IsActive     bool       `gorm:"default:true;column:is_active" json:"is_active"`
	TargetClass  string     `gorm:"size:100;default:'';column:target_class" json:"target_class"`
}

func (ShopItem) TableName() string { return "shop_item" }

// UserPurchaseRecord 用户购买/领取记录表 (防刷核心)
type UserPurchaseRecord struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	UserID     int64     `gorm:"uniqueIndex:uk_user_item"`
	ItemID     int       `gorm:"uniqueIndex:uk_user_item"`
	Count      int       `gorm:"default:0"` // 已购买次数
	UpdateTime time.Time `gorm:"autoUpdateTime"`
}

func (UserPurchaseRecord) TableName() string { return "user_purchase_record" }

// SysAuditLog 管理员操作审计日志表 (防背锅利器)
type SysAuditLog struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	AdminID        int64  `gorm:"not null"`
	AdminName      string `gorm:"size:64;not null"`
	ActionType     string `gorm:"size:64;not null"`
	TargetID       string `gorm:"size:64"`
	TargetUserName string `gorm:"size:255"`
	ScoreChange    int    `gorm:"default:0"`
	Reason         string `gorm:"type:text"`
	Detail         string `gorm:"type:text"`
	RecordID       *int64
	CreateTime     time.Time `gorm:"autoCreateTime"`
}

func (SysAuditLog) TableName() string { return "sys_audit_log" }

// UserMessage 站内信系统表
type UserMessage struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	UserID     int64     `gorm:"not null;index"`
	Title      string    `gorm:"size:100;not null"`
	Content    string    `gorm:"type:text;not null"`
	Type       string    `gorm:"size:50;default:'system'"`
	IsRead     bool      `gorm:"default:false;index"`
	CreateTime time.Time `gorm:"autoCreateTime"`
}

func (UserMessage) TableName() string { return "user_message" }

// WxMsgLog 微信订阅消息发送记录表
type WxMsgLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	UserID      int64     `gorm:"not null"`
	Openid      string    `gorm:"size:100;not null"`
	TemplateID  string    `gorm:"size:100;not null"`
	MsgType     string    `gorm:"size:50"`
	SendContent string    `gorm:"type:longtext"`
	Status      int8      `gorm:"not null"` // 0=失败, 1=成功
	ErrorMsg    string    `gorm:"size:255"`
	CreateTime  time.Time `gorm:"autoCreateTime"`
}

func (WxMsgLog) TableName() string { return "wx_msg_log" }
