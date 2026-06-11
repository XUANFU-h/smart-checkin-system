package service

/*
 ====================================================
 🚨 弃用警告 (DEPRECATED) 🚨
 学风指数站 (股票玩法) 账户资产服务已全面剥离！
 配合数据库模型的下线，此文件内的所有逻辑已静默封存。
 ====================================================
*/

/*
import (
	"checkin-go/global"
	"checkin-go/models"
	"errors"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SignStockAgreement 用户签约并设置6位交易密码
func SignStockAgreement(userID int64, tradePwd string) error {
	if len(tradePwd) != 6 {
		return errors.New("交易密码必须为6位数字")
	}

	hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(tradePwd), bcrypt.DefaultCost)
	hashedPwd := string(hashedBytes)

	account := models.StockAccount{
		UserID:       userID,
		TotalAssets:  0,
		FrozenAssets: 0,
		TradePwdHash: hashedPwd,
		Status:       1,
	}

	return global.DB.Create(&account).Error
}

// TransferToStock 赋能分转入股票系统 (实时到账)
func TransferToStock(userID int64, empAmount int64) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		var conf models.SysConfig
		tx.First(&conf, "config_key = 'STOCK_EXCHANGE_RATE'")
		rate, _ := strconv.ParseInt(conf.ConfigValue, 10, 64)

		var user models.User
		tx.First(&user, userID)
		if int64(user.EmpowermentScore) < empAmount {
			return errors.New("赋能分余额不足")
		}

		stockAmount := empAmount * rate

		tx.Exec("UPDATE user SET empowerment_score = empowerment_score - ? WHERE id = ?", empAmount, userID)
		tx.Exec("UPDATE stock_account SET total_assets = total_assets + ?, frozen_assets = frozen_assets + ? WHERE user_id = ?", stockAmount, stockAmount, userID)

		return nil
	})
}

// TransferToEmpowerment 股票积分提现至赋能分 (T+1 限制)
func TransferToEmpowerment(userID int64, stockAmount int64) error {
	now := time.Now()
	if now.Hour() < 12 {
		return errors.New("系统结算中，请于每日 12:00 之后进行积分提现操作")
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		var account models.StockAccount
		tx.First(&account, "user_id = ?", userID)

		if account.Status != 1 {
			return errors.New("您的股票账户状态异常或已被临时封禁，无法提现")
		}

		available := int64(account.TotalAssets - account.FrozenAssets)
		if available < stockAmount {
			return errors.New("可提现股票积分不足！(今日交易所得/转入的资金需等待明日中午12:00结算后方可提现)")
		}

		var conf models.SysConfig
		tx.First(&conf, "config_key = 'STOCK_EXCHANGE_RATE'")
		rate, _ := strconv.ParseInt(conf.ConfigValue, 10, 64)

		empAmount := int(stockAmount / rate)

		tx.Exec("UPDATE stock_account SET total_assets = total_assets - ? WHERE user_id = ?", stockAmount, userID)
		tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", empAmount, userID)

		return nil
	})
}

// CancelStockAccountAndLiquidate 注销并清算
func CancelStockAccountAndLiquidate(userID int64) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		var account models.StockAccount
		if err := tx.First(&account, "user_id = ?", userID).Error; err != nil {
			return errors.New("股票账户不存在")
		}

		if account.Status == 0 {
			return errors.New("该账户已被注销，请勿重复操作")
		}

		var conf models.SysConfig
		tx.First(&conf, "config_key = 'STOCK_EXCHANGE_RATE'")
		rate, _ := strconv.ParseInt(conf.ConfigValue, 10, 64)
		if rate <= 0 {
			rate = 1
		}

		empAmount := int(float64(account.TotalAssets) / float64(rate) * 1.05)

		tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", empAmount, userID)
		tx.Exec("UPDATE stock_account SET status = 0, total_assets = 0, frozen_assets = 0 WHERE user_id = ?", userID)

		log := models.ScoreLog{
			UserID:    userID,
			ScoreType: "EMPOWERMENT",
			Action:    "ADD",
			Score:     empAmount,
			Reason:    "股票账户注销清算，包含 5% 红利",
			Operator:  "系统清算",
		}
		tx.Create(&log)

		return nil
	})
}
*/
