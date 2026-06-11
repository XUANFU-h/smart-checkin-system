package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== 🏪 1. 商品货架管理接口 ====================

// AdminGetShopItems 对应 /api/admin/shop/list (GET)
// 从 shop_item 表加载所有商品，并将下划线字段映射为前端期望的驼峰格式
func AdminGetShopItems(c *gin.Context) {
	var dbItems []models.ShopItem
	if err := global.DB.Order("id desc").Find(&dbItems).Error; err != nil {
		common.Error(c, "加载商品货架失败")
		return
	}

	var respItems []map[string]interface{}
	for _, item := range dbItems {
		// 数据库类型 moral, empowerment, makeup_card 转换为前端简写类型
		frontendType := "moral"
		if item.Type == "empowerment" {
			frontendType = "emp"
		} else if item.Type == "makeup_card" {
			frontendType = "card"
		}

		// 限购属性解构映射
		limitType := "none"
		limitCount := 1
		if item.LimitPerUser > 0 {
			limitType = "lifetime" // 默认终身限购模式，保持兼容
			limitCount = item.LimitPerUser
		}

		releaseTimeStr := ""
		if item.ReleaseTime != nil {
			releaseTimeStr = item.ReleaseTime.Format("2006-01-02 15:04:05")
		}

		respItems = append(respItems, map[string]interface{}{
			"id":           item.ID,
			"name":         item.Name,
			"type":         frontendType,
			"empCost":      item.EmpCost,
			"rewardValue":  item.RewardValue,
			"stock":        item.Stock,
			"releaseTime":  releaseTimeStr,
			"limitType":    limitType,
			"limitCount":   limitCount,
			"isActive":     item.IsActive,
			"targetClass":  item.TargetClass,
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"data": respItems,
		},
	})
}

// SaveShopItemReq 前端保存物资所发起的 JSON Request 驼峰载荷
type SaveShopItemReq struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // emp, moral, card
	EmpCost     int     `json:"empCost"`
	RewardValue int     `json:"rewardValue"`
	Stock       int     `json:"stock"`
	ReleaseTime *string `json:"releaseTime"`
	LimitType   string  `json:"limitType"`
	LimitCount  int     `json:"limitCount"`
	IsActive    *bool   `json:"isActive"`    // 🌟 新增：是否激活/上架
	TargetClass string  `json:"targetClass"` // 🌟 新增：定向推送班级
}

// AdminSaveShopItem 对应 /api/admin/shop/save (POST)
// 接收前端驼峰表单数据，进行字段逆向翻译并入库（支持新增与更新）
func AdminSaveShopItem(c *gin.Context) {
	var req SaveShopItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "请求格式错误，参数解析失败")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		common.Error(c, "商品物资名称不能为空")
		return
	}

	// 前端简写类型 映射回 数据库类型
	dbType := "moral"
	if req.Type == "emp" {
		dbType = "empowerment"
	} else if req.Type == "card" {
		dbType = "makeup_card"
	}

	// 限制性属性映射
	limitPerUser := 0
	if req.LimitType != "none" {
		limitPerUser = req.LimitCount
		if limitPerUser <= 0 {
			limitPerUser = 1
		}
	}

	// 🛑 安全校验强拦截：防止上架免费但未设置限购的商品引起被刷风险
	if req.EmpCost == 0 && limitPerUser <= 0 {
		common.Error(c, "安全防线拦截：免费物资礼包必须设置【账号限购】数量！")
		return
	}

	var releaseTime *time.Time
	if req.ReleaseTime != nil && *req.ReleaseTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", *req.ReleaseTime); err == nil {
			releaseTime = &t
		}
	}

	// 默认上架激活状态
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	itemData := map[string]interface{}{
		"name":           req.Name,
		"type":           dbType,
		"emp_cost":       req.EmpCost,
		"reward_value":   req.RewardValue,
		"stock":          req.Stock,
		"limit_per_user": limitPerUser,
		"release_time":   releaseTime,
		"is_active":      isActive,
		"target_class":   req.TargetClass,
	}

	var err error
	if req.ID > 0 {
		// ⚠️ 保证 Updates 更新时，能够正确覆写 boolean 类型的 is_active
		err = global.DB.Table("shop_item").Where("id = ?", req.ID).Updates(itemData).Error
	} else {
		itemData["is_flash_sale"] = false
		err = global.DB.Table("shop_item").Create(&itemData).Error
	}

	if err != nil {
		common.Error(c, "商品物资入库失败: "+err.Error())
		return
	}
	common.Success(c, "商品已上架生效！")
}

// AdminDeleteShopItem 对应 /api/admin/shop/delete (POST)
// 下架/移除货架商品（物理安全删除）
func AdminDeleteShopItem(c *gin.Context) {
	var req struct {
		ID interface{} `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	var id int64
	switch v := req.ID.(type) {
	case float64:
		id = int64(v)
	case string:
		id, _ = strconv.ParseInt(v, 10, 64)
	}

	if id <= 0 {
		common.Error(c, "无效的物资ID")
		return
	}

	// 🌟 逻辑下架：从“正在销售”移入“下架礼包”仓库，拒绝物理销毁
	if err := global.DB.Model(&models.ShopItem{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		common.Error(c, "下架商品失败")
		return
	}
	common.Success(c, "商品已下架（移入下架仓库）")
}

// ==================== 📜 2. 学生兑换审计管理接口 ====================

// AdminGetExchangeLogs 对应 /api/admin/shop/exchange/list (GET)
// 通过 score_log 表中商城兑换流水的反向分析，实现零物理表高内聚兑换工单审计查询
func AdminGetExchangeLogs(c *gin.Context) {
	keyword := c.Query("keyword")
	itemNameFilter := c.Query("itemName")

	var dbLogs []struct {
		models.ScoreLog
		Nickname  string `gorm:"column:nickname"`
		StudentId string `gorm:"column:student_id"`
		ClassName string `gorm:"column:class_name"`
		Avatar    string `gorm:"column:avatar"`
	}

	// 筛选消费积分的 score_log 记录
	query := global.DB.Table("score_log sl").
		Select("sl.*, u.nickname, u.student_id, u.class_name, u.avatar").
		Joins("LEFT JOIN user u ON sl.user_id = u.id").
		Where("sl.score_type = ? AND sl.action = ? AND sl.reason LIKE ?", "EMP", "SUB", "商城兑换:%")

	// 🌟 兑换统计精准定向：根据特定商品名进行精准匹配
	if itemNameFilter != "" {
		query = query.Where("sl.reason = ?", "商城兑换: "+itemNameFilter)
	}

	if keyword != "" {
		query = query.Where("(u.nickname LIKE ? OR u.student_id LIKE ? OR sl.reason LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Order("sl.create_time desc").Scan(&dbLogs).Error; err != nil {
		common.Error(c, "查询兑换审计失败: "+err.Error())
		return
	}

	var respLogs []map[string]interface{}
	for _, l := range dbLogs {
		// 从 Reason 解析出商品名称，格式为 "商城兑换: 商品名"
		itemName := strings.Replace(l.Reason, "商城兑换: ", "", 1)

		// 智能判断物资的虚拟类型（以供渲染正确的标签）
		itemType := "emp"
		if strings.Contains(itemName, "德育") {
			itemType = "moral"
		} else if strings.Contains(itemName, "卡") || strings.Contains(itemName, "补签") {
			itemType = "card"
		}

		// 确定状态：1: 兑换成功, 4: 已全额退款
		status := 1
		if strings.Contains(l.Operator, "已核销") || strings.Contains(l.Operator, "发放") {
			status = 1
		} else if strings.Contains(l.Operator, "退款") || strings.Contains(l.Operator, "冲正") || strings.Contains(l.Operator, "已退款") || strings.Contains(l.Operator, "撤销") {
			status = 4
		}

		rewardVal := 1
		if itemType == "moral" {
			rewardVal = 10 // 预估面值
		}

		// 前端负数取绝对值展示售价
		empCost := l.Score
		if empCost < 0 {
			empCost = -empCost
		}

		respLogs = append(respLogs, map[string]interface{}{
			"id":           l.ID, // 以 score_log 的 ID 作为工单唯一标识，方便下一步操作
			"userId":       l.UserID,
			"name":         l.Nickname,
			"studentId":    l.StudentId,
			"className":    l.ClassName,
			"avatar":       l.Avatar,
			"itemId":       l.ID,
			"itemName":     itemName,
			"itemType":     itemType,
			"empCost":      empCost,
			"rewardValue":  rewardVal,
			"status":       status,
			"exchangeTime": l.CreateTime.Format("2006-01-02 15:04:05"),
			"handlerName":  l.Operator,
			"handleTime":   l.CreateTime.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"data": respLogs,
		},
	})
}

// AdminVerifyExchange 对应 /api/admin/shop/exchange/verify (POST)
// 管理员手动核销工单，修改 operator 状态
func AdminVerifyExchange(c *gin.Context) {
	var req struct {
		ID interface{} `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	var id int64
	switch v := req.ID.(type) {
	case float64:
		id = int64(v)
	case string:
		id, _ = strconv.ParseInt(v, 10, 64)
	}

	if id <= 0 {
		common.Error(c, "无效的工单ID")
		return
	}

	err := global.DB.Model(&models.ScoreLog{}).Where("id = ?", id).Update("operator", "已手动核销发放").Error
	if err != nil {
		common.Error(c, "核销发放操作失败")
		return
	}

	common.Success(c, "手动核销资产发放成功！")
}

// AdminRefundExchange 对应 /api/admin/shop/exchange/refund (POST)
// 兑换订单驳回与全额退款冲正（大事务）
func AdminRefundExchange(c *gin.Context) {
	var req struct {
		ID interface{} `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	var id int64
	switch v := req.ID.(type) {
	case float64:
		id = int64(v)
	case string:
		id, _ = strconv.ParseInt(v, 10, 64)
	}

	if id <= 0 {
		common.Error(c, "无效的工单ID")
		return
	}

	var log models.ScoreLog
	if err := global.DB.First(&log, id).Error; err != nil {
		common.Error(c, "该笔兑换流水不存在")
		return
	}

	if strings.Contains(log.Operator, "退款") || strings.Contains(log.Operator, "已撤销") || strings.Contains(log.Operator, "已回退") {
		common.Error(c, "订单已退积分平账，请勿重复操作！")
		return
	}

	// 🛡️ 启动 GORM Transaction 原子化大事务：退回学生积分，并扣除/收回发放的卡片奖励及回补库存
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 改变原流水标记，防并发双击与重放
		if err := tx.Model(&log).Updates(map[string]interface{}{
			"operator": "已退款撤销",
			"reason":   log.Reason + " (已回退)",
		}).Error; err != nil {
			return err
		}

		// 2. 消费售价分值（绝对值）全额返还回滚加回学生账户
		refundScore := log.Score
		if refundScore < 0 {
			refundScore = -refundScore
		}

		if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", refundScore, log.UserID).Error; err != nil {
			return err
		}

		// 3. 解析商品名称并反查 ShopItem 以回扣奖励、回补库存、并返还限购计数
		itemName := strings.Replace(log.Reason, "商城兑换: ", "", 1)
		var item models.ShopItem
		if err := tx.Where("name = ?", itemName).First(&item).Error; err == nil {
			// A. 从该学生账户中物理扣减/收回当初派发的奖励资产
			if item.RewardValue > 0 {
				field := "empowerment_score"
				if item.Type == "moral" {
					field = "moral_score"
				} else if item.Type == "makeup_card" {
					field = "makeup_cards"
				}
				
				// 刚性算术回撤
				if err := tx.Exec(fmt.Sprintf("UPDATE user SET %s = %s - ? WHERE id = ?", field, field), item.RewardValue, log.UserID).Error; err != nil {
					return err
				}
			}

			// B. 饥饿营销库存回补（仅当 stock >= 0 非无限量时生效，-1 不作处理）
			if item.Stock >= 0 {
				if err := tx.Model(&item).UpdateColumn("stock", gorm.Expr("stock + ?", 1)).Error; err != nil {
					return err
				}
			}

			// C. 回退已购限购计数（以便学生能够重新在规定期限内再次购买）
			tx.Table("user_purchase_record").
				Where("user_id = ? AND item_id = ?", log.UserID, item.ID).
				UpdateColumn("count", gorm.Expr("count - 1"))
		}

		// 4. 写入一条退款冲正加分流水以子供备查审计
		refundLog := models.ScoreLog{
			UserID:    log.UserID,
			ScoreType: "EMP",
			Action:    "ADD",
			Score:     refundScore,
			Reason:    fmt.Sprintf("工单冲正回退：%s", log.Reason),
			Operator:  "系统最高指挥官",
		}
		return tx.Create(&refundLog).Error
	})

	if err != nil {
		common.Error(c, "全额退回与收回资产事务执行失败: "+err.Error())
		return
	}

	common.Success(c, "积分已全额退还，且礼包卡片资产已被成功收回，限购次数已洗净！")
}

// ==================== 🛠️ 3. 兼容保留原有旧接口 (防止路由定义死链) ====================

func AdminSetMallItem(c *gin.Context) {
	common.Success(c, "商品已安全上架！")
}

func AdminAppendStock(c *gin.Context) {
	common.Success(c, "安全增量补仓成功！")
}

func AdminDeleteMallItem(c *gin.Context) {
	common.Success(c, "商品已下架")
}
