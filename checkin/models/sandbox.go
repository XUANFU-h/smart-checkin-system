package models

// SysConfig 全局动态配置表
type SysConfig struct {
	ConfigKey   string `gorm:"primaryKey;size:50" json:"config_key"`
	ConfigValue string `gorm:"size:255;not null" json:"config_value"`
	Remark      string `gorm:"size:100" json:"remark"`
}

func (SysConfig) TableName() string { return "sys_config" }
