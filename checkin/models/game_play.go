package models

import "time"

// UserLottoEntry 大乐透/老虎机 投注凭证表 (去除了仅限周末的限制，转为按期数/周数结算)
type UserLottoEntry struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"index" json:"user_id"`
	LottoNumber string    `gorm:"size:20;not null" json:"lotto_number"` // 锁定的数字，如 "88-88-88"
	WeekNum     int       `gorm:"index" json:"week_num"`                // 期数/周数
	IsWinner    bool      `gorm:"default:false" json:"is_winner"`
	PrizeInfo   string    `gorm:"size:255" json:"prize_info"` // 中奖/补偿奖品描述
	IsLocked    bool      `gorm:"default:false;column:is_locked" json:"is_locked"`
	SpinCount   int       `gorm:"default:0;column:spin_count" json:"spin_count"`
	CreateTime  time.Time `gorm:"autoCreateTime" json:"create_time"`
}

func (UserLottoEntry) TableName() string { return "user_lotto_entry" }

// LottoSetting 彩票全局配置与“内幕”设置
type LottoSetting struct {
	WeekNum       int    `gorm:"primaryKey" json:"week_num"`    // 哪一周/期
	WinningNumber string `gorm:"size:20" json:"winning_number"` // 中奖数字 (内定大奖)
	PrizeType     string `gorm:"size:20" json:"prize_type"`     // 奖励类型
	PrizeValue    int    `json:"prize_value"`
	IsDrawn       bool   `gorm:"default:false" json:"is_drawn"` // 是否已开奖
}

func (LottoSetting) TableName() string { return "lotto_setting" }

// LottoPoolLog 德育分/赋能分 双奖池变动历史日志表
type LottoPoolLog struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	WeekNum     int       `gorm:"index;column:week_num" json:"week_num"`
	PoolType    string    `gorm:"size:20;column:pool_type" json:"pool_type"` // "empowerment" or "moral"
	Amount      int       `gorm:"column:amount" json:"amount"`
	LogType     string    `gorm:"size:20;column:log_type" json:"log_type"` // "user_entry" or "admin_inject" or "admin_deduct" or "user_draw"
	UserID      int64     `gorm:"column:user_id;default:0" json:"user_id"`
	Description string    `gorm:"size:255;column:description" json:"description"`
	CreateTime  time.Time `gorm:"autoCreateTime;column:create_time" json:"create_time"`
}

func (LottoPoolLog) TableName() string { return "lotto_pool_log" }
