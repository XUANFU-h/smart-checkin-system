package models

import "time"

// SysAdminMenuPerm 子管理员页面菜单权限表
type SysAdminMenuPerm struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID   int64     `gorm:"not null;index:uk_admin_menu,unique" json:"adminId"`
	MenuKey   string    `gorm:"size:50;not null;index:uk_admin_menu,unique" json:"menuKey"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (SysAdminMenuPerm) TableName() string { return "sys_admin_menu_perm" }

// SysAdminActionPerm 子管理员功能按钮权限表
type SysAdminActionPerm struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AdminID   int64     `gorm:"not null;index:uk_admin_action,unique" json:"adminId"`
	ActionKey string    `gorm:"size:80;not null;index:uk_admin_action,unique" json:"actionKey"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (SysAdminActionPerm) TableName() string { return "sys_admin_action_perm" }

// SysAdminTrackRelation 子管理员赛道隔离表
type SysAdminTrackRelation struct {
	AdminID   int64     `gorm:"primaryKey;not null;index:uk_admin_track,unique" json:"adminId"`
	TrackID   int64     `gorm:"primaryKey;not null;index:uk_admin_track,unique" json:"trackId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (SysAdminTrackRelation) TableName() string { return "sys_admin_track_relation" }
