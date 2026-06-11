package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ExchangeItemExtended(userID int64, itemID int) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		var item models.ShopItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, itemID).Error; err != nil {
			return errors.New("商品已下架或不存在")
		}

		// 🛡️ 刚性安全拦截：逻辑下架商品不可兑换
		if !item.IsActive {
			return errors.New("该礼包商品已被管理员暂时下架")
		}

		// 🛡️ 刚性安全拦截：班级定向礼包特定权限判定
		if item.TargetClass != "" && item.TargetClass != "全部" {
			var user models.User
			if err := tx.First(&user, userID).Error; err != nil {
				return errors.New("用户信息异常")
			}
			if user.ClassName != item.TargetClass {
				return errors.New("该礼包仅面向特定班级定向派发，您暂无兑换权限！")
			}
		}

		now := time.Now()
		if item.StartTime != nil && now.Before(*item.StartTime) {
			return errors.New("该礼包活动尚未开始")
		}
		if item.EndTime != nil && now.After(*item.EndTime) {
			return errors.New("该礼包活动已结束")
		}

		// 💡 修复2：严谨的首次购买与复购逻辑分支
		if item.LimitPerUser > 0 {
			var record models.UserPurchaseRecord
			err := tx.Where("user_id = ? AND item_id = ?", userID, itemID).First(&record).Error

			if err != nil && err != gorm.ErrRecordNotFound {
				return errors.New("限购状态查询失败")
			}

			if err == nil && record.Count >= item.LimitPerUser {
				return fmt.Errorf("该礼包每人限领 %d 个，您已达到上限", item.LimitPerUser)
			}

			if err == gorm.ErrRecordNotFound {
				record = models.UserPurchaseRecord{UserID: userID, ItemID: itemID, Count: 1}
				tx.Create(&record)
			} else {
				record.Count++
				tx.Save(&record)
			}
		}

		if item.Stock == 0 {
			return errors.New("商品已被抢空")
		}

		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return errors.New("用户信息异常")
		}
		if user.EmpowermentScore < item.EmpCost {
			return errors.New("赋能分余额不足")
		}

		// 1. 扣除兑换售价
		user.EmpowermentScore -= item.EmpCost

		// 2. 即时充值并授信对应资产类型变动
		var addLog *models.ScoreLog
		if item.RewardValue > 0 {
			if item.Type == "empowerment" {
				user.EmpowermentScore += item.RewardValue
				addLog = &models.ScoreLog{
					UserID:    userID,
					ScoreType: "EMP",
					Action:    "ADD",
					Score:     item.RewardValue,
					Reason:    fmt.Sprintf("兑换礼包追授: %s", item.Name),
					Operator:  "系统自动",
				}
			} else if item.Type == "moral" {
				user.MoralScore += item.RewardValue
				addLog = &models.ScoreLog{
					UserID:    userID,
					ScoreType: "MORAL",
					Action:    "ADD",
					Score:     item.RewardValue,
					Reason:    fmt.Sprintf("兑换礼包追授: %s", item.Name),
					Operator:  "系统自动",
				}
			} else if item.Type == "makeup_card" {
				user.MakeupCards += item.RewardValue
				addLog = &models.ScoreLog{
					UserID:    userID,
					ScoreType: "card",
					Action:    "ADD",
					Score:     item.RewardValue,
					Reason:    fmt.Sprintf("兑换礼包追授: %s", item.Name),
					Operator:  "系统自动",
				}
			}
		}

		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// 3. 扣减商品库存
		if item.Stock > 0 {
			item.Stock -= 1
			if err := tx.Save(&item).Error; err != nil {
				return err
			}
		}

		// 4. 记录扣减积分流水
		subLog := models.ScoreLog{
			UserID:    userID,
			ScoreType: "EMP",
			Action:    "SUB",
			Score:     item.EmpCost,
			Reason:    fmt.Sprintf("商城兑换: %s", item.Name),
			Operator:  "系统自动",
		}
		if err := tx.Create(&subLog).Error; err != nil {
			return err
		}

		// 5. 记录新增资产追授变动流水
		if addLog != nil {
			if err := tx.Create(addLog).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
