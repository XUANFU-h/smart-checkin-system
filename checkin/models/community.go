package models

import "time"

// CommunityPost 大盘社区帖子统一表 (接管打卡推流、官方动态、用户发帖)
type CommunityPost struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	PostType     string    `gorm:"size:20;not null"` // OFFICIAL, CHECKIN, USER
	RefRecordID  *int64    // 关联的打卡记录ID(如果是推流，可为空)
	AuthorID     int64     `gorm:"not null"` // 记录发布者ID
	AuthorName   string    `gorm:"size:100;not null"`
	AuthorAvatar string    `gorm:"size:255"`
	IsAnonymous  bool      `gorm:"default:false"` // 👇 新增：是否匿名
	Topic        string    `gorm:"size:200"`
	Content      string    `gorm:"type:text;not null"`
	ImageURL     string    `gorm:"size:500"`
	Channel      string    `gorm:"size:50;default:'recommend'"`
	TrackID      int       `gorm:"default:0"` // 👇 新增：发帖时绑定的赛道
	AdminComment string    `gorm:"size:500"`  // 官方点评
	LikeCount    int       `gorm:"default:0"` // 👇 新增：点赞总数
	CommentCount int       `gorm:"default:0"` // 👇 新增：评论总数
	IsPinned     bool      `gorm:"default:false"`
	SortOrder    int       `gorm:"default:0"`
	CreateTime   time.Time `gorm:"autoCreateTime"`
}

func (CommunityPost) TableName() string { return "community_post" }

// 👇 新增：CommunityComment 帖子评论表
type CommunityComment struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	PostID     int64     `gorm:"not null;index"`
	UserID     int64     `gorm:"not null"`
	UserName   string    `gorm:"size:100"`
	UserAvatar string    `gorm:"size:255"`
	Content    string    `gorm:"size:100;not null"` // 限制20字以内
	Status     int8      `gorm:"default:1"`         // 1正常 0违规被隐藏
	CreateTime time.Time `gorm:"autoCreateTime"`
}

func (CommunityComment) TableName() string { return "community_comment" }

// 👇 新增：CommunityLike 帖子点赞记录表 (防重复点赞)
type CommunityLike struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	PostID     int64     `gorm:"not null;uniqueIndex:uk_post_user"`
	UserID     int64     `gorm:"not null;uniqueIndex:uk_post_user"`
	CreateTime time.Time `gorm:"autoCreateTime"`
}

func (CommunityLike) TableName() string { return "community_like" }

// BountyTask 悬赏任务表
type BountyTask struct {
	ID         int        `gorm:"primaryKey;autoIncrement"`
	Title      string     `gorm:"size:100;not null"`
	RewardText string     `gorm:"size:50;not null"`
	Status     int8       `gorm:"default:1"` // 1展示 0隐藏
	SortOrder  int        `gorm:"default:0"`
	EndTime    *time.Time // 可以为空
	IsPinned   bool       `gorm:"default:false"`
	CreateTime time.Time  `gorm:"autoCreateTime"`
}

func (BountyTask) TableName() string { return "bounty_task" }

// GlobalNotice 全局强弹窗通知表
type GlobalNotice struct {
	ID         int       `gorm:"primaryKey;autoIncrement"`
	Title      string    `gorm:"size:100;not null"`
	Content    string    `gorm:"type:text;not null"`
	IsActive   bool      `gorm:"default:false"`
	CreateTime time.Time `gorm:"autoCreateTime"`
}

func (GlobalNotice) TableName() string { return "global_notice" }
