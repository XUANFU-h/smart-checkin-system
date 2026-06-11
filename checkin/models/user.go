package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// User 核心用户表
type User struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	OpenID   string `gorm:"size:100;uniqueIndex:uk_openid" json:"openid"`
	Nickname string `gorm:"size:50" json:"nickname"`
	Avatar   string `gorm:"size:255" json:"avatar"`

	RealName string `gorm:"size:255" json:"realName"`
	// 💡 优化：为 StudentID 增加索引，保障频繁的学号查询性能
	StudentID string `gorm:"size:255;index" json:"studentId"`
	Password  string `gorm:"size:255" json:"-"`

	ClassName        string    `gorm:"size:50" json:"className"`
	EmpowermentScore int       `gorm:"default:0" json:"empowermentScore"`
	MoralScore       int       `gorm:"default:0" json:"moralScore"`
	MakeupCards      int       `gorm:"default:0" json:"makeupCards"`
	UserType         int8      `gorm:"default:0" json:"userType"`
	TrackModifyCount int       `gorm:"default:1" json:"trackModifyCount"`
	CurrentDeviceID  string    `gorm:"size:255" json:"currentDeviceId"`
	AccountStatus    int8      `gorm:"default:0" json:"accountStatus"`
	CreateTime       time.Time `gorm:"autoCreateTime" json:"createTime"`
	UpdateTime       time.Time `gorm:"autoUpdateTime" json:"updateTime"`
	UnlockedMakeupDays []string  `gorm:"-" json:"unlockedMakeupDays"`
}

func (User) TableName() string { return "user" }

func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	// 如果你是基于 H5 虚拟注册，这里逻辑是安全的
	if u.OpenID == "" {
		u.OpenID = fmt.Sprintf("h5_virtual_%d", time.Now().UnixNano())
	}
	return nil
}
