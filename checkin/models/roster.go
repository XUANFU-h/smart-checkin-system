package models

// DeviceBlacklist 设备黑名单 (封禁设备防小号)
type DeviceBlacklist struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	DeviceID string `gorm:"size:255;uniqueIndex"`
	Reason   string `gorm:"size:255"`
}

func (DeviceBlacklist) TableName() string { return "device_blacklist" }
