package service

/*
 ====================================================
 🚨 弃用警告 (DEPRECATED) 🚨
 学风指数站 (股票玩法) 交易引擎已全面剥离！
 此文件内的核心撮合逻辑与持仓计算已静默封存。
 ====================================================
*/

/*
import (
	"checkin-go/global"
	"checkin-go/models"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ExecuteTrade 股票买卖核心逻辑引擎
func ExecuteTrade(userID int64, stockID int, tradeType string, amount int, tradePassword string) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 账户校验
		var account models.StockAccount
		if err := tx.First(&account, "user_id = ?", userID).Error; err != nil {
			return errors.New("您尚未开通股票账户")
		}

		if account.Status != 1 {
			return errors.New("您的账户状态异常或已被封禁，禁止交易")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(account.TradePwdHash), []byte(tradePassword)); err != nil {
			return errors.New("交易密码错误")
		}

		// 2. 标的物校验
		var stock models.StockItem
		if err := tx.First(&stock, stockID).Error; err != nil {
			return errors.New("该交易标的不存在")
		}
		if stock.Status != 1 {
			return errors.New("该标的目前处于停牌状态")
		}

		tradePrice := stock.CurrentPrice
		totalPoints := int(float64(amount) * tradePrice)

		// 3. 买入/卖出 执行路由
		if tradeType == "BUY" {
			available := account.TotalAssets - account.FrozenAssets
			if available < totalPoints {
				return errors.New("可用资金不足")
			}

			if err := tx.Exec("UPDATE stock_account SET total_assets = total_assets - ? WHERE user_id = ?", totalPoints, userID).Error; err != nil {
				return err
			}

			var pos models.UserPosition
			err := tx.Where("user_id = ? AND stock_id = ?", userID, stockID).First(&pos).Error
			if err != nil {
				pos = models.UserPosition{
					UserID:       userID,
					StockID:      stockID,
					TotalAmount:  amount,
					FrozenAmount: amount,
					AveragePrice: tradePrice,
					HoldCost:     totalPoints,
				}
				if err := tx.Create(&pos).Error; err != nil {
					return err
				}
			} else {
				newTotal := pos.TotalAmount + amount
				newCost := pos.HoldCost + totalPoints
				newAvg := float64(newCost) / float64(newTotal)
				if err := tx.Model(&pos).Updates(map[string]interface{}{
					"total_amount":  newTotal,
					"frozen_amount": pos.FrozenAmount + amount,
					"hold_cost":     newCost,
					"average_price": newAvg,
				}).Error; err != nil {
					return err
				}
			}
		} else if tradeType == "SELL" {
			var pos models.UserPosition
			if err := tx.Where("user_id = ? AND stock_id = ?", userID, stockID).First(&pos).Error; err != nil {
				return errors.New("您未持有该标的")
			}

			availablePos := pos.TotalAmount - pos.FrozenAmount
			if availablePos < amount {
				return errors.New("可卖份额不足 (今日买入需明日解冻)")
			}

			if pos.TotalAmount == amount {
				if err := tx.Delete(&pos).Error; err != nil {
					return err
				}
			} else {
				costReduction := int(float64(pos.HoldCost) * (float64(amount) / float64(pos.TotalAmount)))
				if err := tx.Model(&pos).Updates(map[string]interface{}{
					"total_amount": pos.TotalAmount - amount,
					"hold_cost":    pos.HoldCost - costReduction,
				}).Error; err != nil {
					return err
				}
			}

			if err := tx.Exec("UPDATE stock_account SET total_assets = total_assets + ?, frozen_assets = frozen_assets + ? WHERE user_id = ?", totalPoints, totalPoints, userID).Error; err != nil {
				return err
			}
		} else {
			return errors.New("未知的交易类型")
		}

		// 4. 写入交易流水账单
		logRecord := models.StockTransaction{
			UserID:      userID,
			StockID:     stockID,
			TradeType:   tradeType,
			Amount:      amount,
			TradePrice:  tradePrice,
			TotalPoints: totalPoints,
			TradeStatus: 1,
			Sm4Payload:  "ENCRYPTED_SIGNATURE",
		}
		if err := tx.Create(&logRecord).Error; err != nil {
			return err
		}

		return nil
	})
}
*/
