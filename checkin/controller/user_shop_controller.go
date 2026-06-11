package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/service"
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type ExchangeReq struct {
	ItemID int `json:"itemId"` // 💡 修复1：严格对齐前端驼峰命名
}

// GetShopItems 获取商城商品列表
func GetShopItems(c *gin.Context) {
	userIDRaw, _ := c.Get("userId")
	userID := userIDRaw.(int64)

	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		common.Error(c, "用户信息异常")
		return
	}

	var items []models.ShopItem
	// 🟢 UI级物理屏蔽：只展现处于激活状态的在售/售罄物资
	global.DB.Where("is_active = ? AND stock != 0", true).Order("emp_cost asc").Find(&items)

	var result []map[string]interface{}

	for _, item := range items {
		// 🟢 班级定向可见度隔离：如果不相符，物理隐形
		if item.TargetClass != "" && item.TargetClass != "全部" && item.TargetClass != user.ClassName {
			continue
		}

		var boughtCount int64
		global.DB.Table("user_purchase_record").Where("user_id = ? AND item_id = ?", userID, item.ID).Count(&boughtCount)

		result = append(result, map[string]interface{}{
			"id":           item.ID,
			"name":         item.Name,
			"reward_value": item.RewardValue,
			"type":         item.Type,
			"stock":        item.Stock,
			"emp_cost":     item.EmpCost,
			"start_time":   item.StartTime, // 💡 修复3：统一使用 start_time 映射限时逻辑
			"release_time": item.ReleaseTime,
			"end_time":     item.EndTime,
			"user_limit":   item.LimitPerUser,
			"user_bought":  boughtCount,
		})
	}

	common.Success(c, result)
}

// ExchangeItem 兑换商城物品
func ExchangeItem(c *gin.Context) {
	var req ExchangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析异常")
		return
	}

	userIDRaw, _ := c.Get("userId")
	userID := userIDRaw.(int64)
	ctx := context.Background()

	// 运维级高并发防刷锁
	lockKey := fmt.Sprintf("shop_exchange_lock:U%d", userID)
	acquired, _ := global.Redis.SetNX(ctx, lockKey, 1, 3*time.Second).Result()
	if !acquired {
		common.Error(c, "请求排队中，请勿连续点击")
		return
	}
	defer global.Redis.Del(ctx, lockKey)

	err := service.ExchangeItemExtended(userID, req.ItemID)
	if err != nil {
		common.Error(c, err.Error())
		return
	}

	common.Success(c, "兑换成功！")
}
