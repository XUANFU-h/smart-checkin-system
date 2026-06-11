package models

import "time"

// AdminUser 全局统一管理员表
type AdminUser struct {
	ID            int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username      string     `gorm:"size:64;not null;unique" json:"username"`
	Password      string     `gorm:"size:128;not null" json:"-"`
	RealName      string     `gorm:"size:64" json:"realName"`
	RoleLevel     int8       `gorm:"not null" json:"roleLevel"`
	Status        int8       `gorm:"default:0" json:"accountStatus"`
	StudentID     string     `gorm:"column:student_id;size:64;index" json:"studentId"`
	BindingSecret string     `gorm:"column:binding_secret;size:64" json:"bindingSecret"`
	BoundUserID   int64      `gorm:"column:bound_user_id;default:0" json:"boundUserId"`
	IsBound       bool       `gorm:"column:is_bound;default:false" json:"isBound"`
	LastLoginTime *time.Time `json:"lastLoginTime"`
	SecretExpireTime *time.Time `gorm:"column:secret_expire_time" json:"secretExpireTime"`
	CreateTime    time.Time  `gorm:"autoCreateTime" json:"createTime"`
}

func (AdminUser) TableName() string { return "admin_user" }

// SysAdminPermission 二级干事权限矩阵表
type SysAdminPermission struct {
	AdminID          int64 `gorm:"primaryKey"`
	CanAuditTrack    bool  `gorm:"default:false"`
	CanAuditAppeal   bool  `gorm:"default:false"`
	CanPushCommunity bool  `gorm:"default:false"`
	CanManageBounty  bool  `gorm:"default:false"`
	MinScore         int   `gorm:"default:5"`
	MaxScore         int   `gorm:"default:15"`
}

func (SysAdminPermission) TableName() string { return "sys_admin_permission" }



// SysDictionary 全局系统字典表
type SysDictionary struct {
	ID           int    `gorm:"primaryKey;autoIncrement"`
	DictType     string `gorm:"size:50;not null;index"`
	ParentID     int    `gorm:"default:0"`
	Name         string `gorm:"size:50;not null"`
	RequireImage bool   `gorm:"default:false"`
	SortOrder    int    `gorm:"default:0"`
}

func (SysDictionary) TableName() string { return "sys_dictionary" }

// SystemDynamicConfig 系统全局动态配置表
type SystemDynamicConfig struct {
	ID          int    `gorm:"primaryKey;autoIncrement"`
	ConfigKey   string `gorm:"size:50;not null;unique"`
	ConfigValue string `gorm:"type:text"`
	Description string `gorm:"size:255"`
}

func (SystemDynamicConfig) TableName() string { return "sys_config" }
