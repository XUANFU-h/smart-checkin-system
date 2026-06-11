package initialize

import (
	"checkin-go/global"
	"checkin-go/models"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() {
	// 直接使用已在 config.go 中装载好的规范化 URL 路径
	dsn := global.Config.Spring.Datasource.Url

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		slog.Error("MySQL连接失败", "err", err)
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	global.DB = db

	// 自动迁移全量持久层核心表模型结构
	err = global.DB.AutoMigrate(
		&models.User{},
		&models.ClassList{}, // 💡 正式将标准班级字典表纳入自动同步底座
		&models.CheckinRecord{},
		&models.Track{},
		&models.UserTrack{},
		&models.ScoreLog{},
		&models.ShopItem{},
		&models.UserPurchaseRecord{},
		&models.UserLottoEntry{},
		&models.LottoSetting{},
		&models.LottoPoolLog{}, // 💡 新增大乐透收支变动日志表
		&models.AppealRecord{},
		&models.UserMakeupLog{},
		&models.SysAdminMenuPerm{},
		&models.SysAdminActionPerm{},
		&models.SysAdminTrackRelation{},
		&models.AppealCategory{},
		&models.AdminUser{},
		&models.CommunityPost{},
		&models.CommunityComment{},
		&models.CommunityLike{},
		&models.BountyTask{},
		&models.GlobalNotice{},
	)
	if err != nil {
		slog.Error("数据库自动迁移建表失败", "err", err)
	} else {
		slog.Info("数据库自动迁移完成：全量安全核心数据表已就绪！")
	}

	slog.Info("🛡️ 专属特权鉴权体系就绪。全栈身份及注册机制全面转向动态解耦模型。")
	slog.Info("✅ MySQL数据库连接成功")
}
