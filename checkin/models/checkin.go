package models

import "time"

// Track 赛道配置表
type Track struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Type        string    `gorm:"size:20;default:'mixed'" json:"type"`
	RequireType string    `gorm:"size:50;default:'image'" json:"requireType"`
	MinScore    int       `gorm:"default:0" json:"minScore"`
	MaxScore    int       `gorm:"default:10" json:"maxScore"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	CreateTime  time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (Track) TableName() string { return "track" }

// UserTrack 学生赛道报名表
type UserTrack struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null;uniqueIndex:uk_user_track" json:"userId"`
	TrackID    int       `gorm:"not null;uniqueIndex:uk_user_track" json:"trackId"`
	CreateTime time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (UserTrack) TableName() string { return "user_track" }

// CheckinRecord 极简打卡记录表 (瘦身后的超快版本)
type CheckinRecord struct {
	ID            int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64  `gorm:"not null;index" json:"userId"`
	TrackID       int    `gorm:"not null;index" json:"trackId"`
	ContentText   string `gorm:"type:text" json:"contentText"`
	ImageURL      string `gorm:"size:255" json:"imageUrl"`
	ImageMd5      string `gorm:"size:32" json:"imageMd5"`
	Status        int8   `gorm:"default:0" json:"status"` // 0=待审, 1=通过, 2=驳回, 3=警告, 4=风控拦截
	AiScore       int    `gorm:"default:0" json:"aiScore"`
	AiComment     string `gorm:"size:500" json:"aiComment"`
	AdminComment  string `gorm:"size:500" json:"adminComment"` // ✅ 修复：新增导师独立人工评语字段
	OcrText       string `gorm:"type:text" json:"ocrText"`
	PublishStatus int8   `gorm:"default:0" json:"publishStatus"` // 0=未放榜隐藏, 1=已放榜可查阅
	IsPushed      bool   `gorm:"default:false" json:"isPushed"`  // 是否推流到社区
	PushChannel   string `gorm:"size:50" json:"pushChannel"`     // 推流频道
	IsPinned      bool   `gorm:"default:false" json:"isPinned"`  // 在社区是否置顶

	// 核心修复：修改为高兼容性的 varchar(20)，彻底粉碎自 Java 迁移而来的 parseTime=True 类型扫描死锁
	CheckinDate string `gorm:"type:varchar(20);not null;index" json:"checkinDate"`

	CreateTime time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (CheckinRecord) TableName() string { return "checkin_record" }

// CheckinApprovalQueue 打卡修改/删除审批流队列表
type CheckinApprovalQueue struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RecordID     int64     `gorm:"not null" json:"recordId"`
	SubAdminID   int64     `gorm:"not null" json:"subAdminId"`
	SubAdminName string    `gorm:"size:64;not null" json:"subAdminName"`
	ActionType   string    `gorm:"size:20;not null" json:"actionType"` // DELETE / EDIT
	NewContent   string    `gorm:"type:text" json:"newContent"`
	Status       int       `gorm:"default:0" json:"status"` // 0=待审批, 1=已同意, 2=已驳回
	CreateTime   time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (CheckinApprovalQueue) TableName() string { return "checkin_approval_queue" }

// UserMakeupLog 生产级历史补签天数解锁记录表
type UserMakeupLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"not null;uniqueIndex:uk_user_makeup_date" json:"userId"`
	MakeupDate string    `gorm:"type:date;not null;uniqueIndex:uk_user_makeup_date" json:"makeupDate"`
	CreateTime time.Time `gorm:"autoCreateTime" json:"createTime"`
}

func (UserMakeupLog) TableName() string { return "user_makeup_log" }
