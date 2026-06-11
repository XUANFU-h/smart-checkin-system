package models

/*
 ====================================================
 🚨 弃用警告 (DEPRECATED) 🚨
 学风指数站 (股票玩法) 核心数据模型已全面剥离！
 此文件内的所有数据表结构已进行静默封存。

 如果您的 IDE 报错，请继续按照下方的【第三步】
 去清理数据库建表文件。
 ====================================================
*/

/*
import "time"

// 学风指数站 独立资金账户表 (与打卡系统解耦)
type StockAccount struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null;uniqueIndex" json:"user_id"`
	TotalAssets  int       `gorm:"not null;default:0" json:"total_assets"`  // 总技能点资产
	FrozenAssets int       `gorm:"not null;default:0" json:"frozen_assets"` // T+1 冻结资产
	TradePwdHash string    `gorm:"size:255;not null" json:"-"`              // Argon2id 交易密码哈希
	Status       int8      `gorm:"default:1" json:"status"`                 // 1=正常, 0=已注销
	CreateTime   time.Time `gorm:"autoCreateTime" json:"create_time"`
	UpdateTime   time.Time `gorm:"autoUpdateTime" json:"update_time"`
}
func (StockAccount) TableName() string { return "stock_account" }

// StockItem 标的配置表
type StockItem struct {
	ID             int       `gorm:"primaryKey;autoIncrement" json:"id"`
	TrackID        int       `gorm:"not null;index:idx_track" json:"track_id"`
	Code           string    `gorm:"size:20;not null;unique" json:"code"`
	Name           string    `gorm:"size:50;not null" json:"name"`
	StockType      string    `gorm:"size:20;not null" json:"stock_type"`
	CurrentPrice   float64   `gorm:"type:decimal(10,2);default:100.00" json:"current_price"`
	MinFluctuation float64   `gorm:"type:decimal(5,2);not null" json:"min_fluctuation"`
	MaxFluctuation float64   `gorm:"type:decimal(5,2);not null" json:"max_fluctuation"`
	Status         int8      `gorm:"default:1" json:"status"`
	CreateTime     time.Time `gorm:"autoCreateTime" json:"create_time"`
}
func (StockItem) TableName() string { return "stock_item" }

// UserPosition 用户持仓表
type UserPosition struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null;uniqueIndex:uk_user_stock" json:"user_id"`
	StockID      int       `gorm:"not null;uniqueIndex:uk_user_stock" json:"stock_id"`
	TotalAmount  int       `gorm:"not null;default:0" json:"total_amount"`
	FrozenAmount int       `gorm:"not null;default:0" json:"frozen_amount"`
	AveragePrice float64   `gorm:"type:decimal(10,2);default:0" json:"average_price"`
	HoldCost     int       `gorm:"not null;default:0" json:"hold_cost"`
	CreateTime   time.Time `gorm:"autoCreateTime" json:"create_time"`
	UpdateTime   time.Time `gorm:"autoUpdateTime" json:"update_time"`
}
func (UserPosition) TableName() string { return "user_position" }

// StockTransaction 交易流水表
type StockTransaction struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"not null;index:idx_user_time" json:"user_id"`
	StockID     int       `gorm:"not null;index" json:"stock_id"`
	TradeType   string    `gorm:"size:20;not null" json:"trade_type"`
	Amount      int       `gorm:"not null" json:"amount"`
	TradePrice  float64   `gorm:"type:decimal(10,2);not null" json:"trade_price"`
	TotalPoints int       `gorm:"not null" json:"total_points"`
	TradeStatus int8      `gorm:"default:1" json:"trade_status"`
	Sm4Payload  string    `gorm:"type:text" json:"sm4_payload"`
	CreateTime  time.Time `gorm:"autoCreateTime" json:"create_time"`
}
func (StockTransaction) TableName() string { return "stock_transaction" }

// StockMarketHistory 每日行情历史表
type StockMarketHistory struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	StockID     int       `gorm:"not null;uniqueIndex:uk_stock_date" json:"stock_id"`
	RecordDate  string    `gorm:"type:date;not null;uniqueIndex:uk_stock_date" json:"record_date"`
	OpenPrice   float64   `gorm:"type:decimal(10,2);not null" json:"open_price"`
	ClosePrice  float64   `gorm:"type:decimal(10,2);not null" json:"close_price"`
	Fluctuation float64   `gorm:"type:decimal(5,4);not null" json:"fluctuation"`
	CheckinRate float64   `gorm:"type:decimal(5,4);not null" json:"checkin_rate"`
	CreateTime  time.Time `gorm:"autoCreateTime" json:"create_time"`
}
func (StockMarketHistory) TableName() string { return "stock_market_history" }
*/
