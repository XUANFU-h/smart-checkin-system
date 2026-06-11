package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/service"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetSubAdminList 一级管理员查看：获取所有二级干事及权限列表
func GetSubAdminList(c *gin.Context) {
	sql := `SELECT s.id, s.username, s.real_name as realName, s.status, 
			p.can_audit_track as canAuditTrack, p.can_audit_appeal as canAuditAppeal 
			FROM sys_admin_sub s LEFT JOIN sys_admin_permission p ON s.id = p.admin_id`

	var list []map[string]interface{}
	global.DB.Raw(sql).Scan(&list)
	common.Success(c, list)
}

// ToggleSubAdminStatus 一键封禁/解封干事账号
func ToggleSubAdminStatus(c *gin.Context) {
	type Req struct {
		ID     int64 `json:"id"`
		Status int   `json:"status"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	global.DB.Exec("UPDATE sys_admin_sub SET status = ? WHERE id = ?", req.Status, req.ID)
	msg := "账号已恢复"
	if req.Status == 1 {
		msg = "干事账号已被冻结下线"
	}
	common.Success(c, msg)
}

// GetBlindAuditList 生产级成品：随机拉取盲审试卷 (融合 Redis 5分钟独占锁机制)
func GetBlindAuditList(c *gin.Context) {
	// 从鉴权拦截器中提取登录的二级干事ID
	adminIDRaw, _ := c.Get("userId")
	adminID := adminIDRaw.(int64)

	ctx := context.Background()

	// 1. 从关系表中抓取当前干事被随机分派的赛道ID
	var trackIDs []int
	global.DB.Table("sys_admin_track_relation").Where("admin_id = ?", adminID).Pluck("track_id", &trackIDs)
	if len(trackIDs) == 0 {
		common.Success(c, []interface{}{}) // 没有被分配任何赛道
		return
	}

	// 2. 捞取状态为待审核(0)的记录 candidate 候选池
	var records []models.CheckinRecord
	global.DB.Where("track_id IN ? AND status = ?", trackIDs, 0).Order("create_time ASC").Limit(30).Find(&records)

	var finalAuditList []models.CheckinRecord

	// 3. 🛡️ 【排他性锁死过滤】：检查每条记录是否已被其他干事锁在工作台上
	for _, record := range records {
		lockKey := fmt.Sprintf("audit_lock:%d", record.ID)

		// 尝试抢占独占锁，租约时长 5 分钟
		acquired, _ := global.Redis.SetNX(ctx, lockKey, adminID, 5*time.Minute).Result()
		if acquired {
			// 成功扣下此记录，外派发给当前前端
			finalAuditList = append(finalAuditList, record)
			if len(finalAuditList) >= 5 { // 每次下发 5 条，防止单人一口气锁死过多任务
				break
			}
		}
	}

	common.Success(c, finalAuditList)
}

// SubmitAuditResult 生产级成品：干事人工纠偏打分提交
func SubmitAuditResult(c *gin.Context) {
	type AuditResultReq struct {
		RecordID  int64  `json:"record_id" binding:"required"`
		Score     int    `json:"score" binding:"required,gte=5,lte=20"` // 严格限制在配置区间
		Comment   string `json:"comment" binding:"required"`
		IsWarning bool   `json:"is_warning"` // 是否给予作弊警告
	}

	var req AuditResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析异常")
		return
	}

	adminIDRaw, _ := c.Get("userId")
	adminID := adminIDRaw.(int64)
	ctx := context.Background()

	// 1. 强安全审计：校验锁指纹，判断该卷子目前是不是你在锁定的，防越权提交
	lockKey := fmt.Sprintf("audit_lock:%d", req.RecordID)
	lockerID, err := global.Redis.Get(ctx, lockKey).Int64()
	if err != nil || lockerID != adminID {
		common.Error(c, "操作超时拦截：由于您超过 5 分钟未批阅，此打卡记录已被回收进入公共池，请重新拉取。")
		return
	}

	var record models.CheckinRecord
	if err := global.DB.First(&record, req.RecordID).Error; err != nil {
		common.Error(c, "阅卷对应的源打卡档案不存在")
		return
	}

	targetStatus := int8(2) // 默认改分后进入状态 2（已打分待清晨10点公布）
	if req.IsWarning {
		targetStatus = 3 // 判定为警告作弊
	}

	// 2. 核心事务校准落库
	originalAiScore := record.AiScore
	record.AiScore = req.Score
	record.AiComment = fmt.Sprintf("[人工批改] %s", req.Comment)
	record.Status = targetStatus

	if err := global.DB.Save(&record).Error; err != nil {
		common.Error(c, "人工核验数据写入数据库失败")
		return
	}

	// 3. 批阅完成，过河拆桥，立即物理释放 Redis 独占锁，供他人检索
	global.Redis.Del(ctx, lockKey)

	// 4. 🚨 写入全局高精度篡改追溯审计记录
	var studentUser models.User
	global.DB.Select("nickname").First(&studentUser, record.UserID)

	detailMsg := fmt.Sprintf("原AI判定分: %d分, 干事纠正为: %d分. 评语: %s", originalAiScore, req.Score, req.Comment)
	service.RecordAdminAuditLog(
		adminID,
		"二级换防干事",
		"干事人工盲审打分",
		strconv.FormatInt(req.RecordID, 10),
		studentUser.ClassName+"-"+studentUser.Nickname,
		req.Score-originalAiScore,
		req.Comment,
		detailMsg,
	)

	common.Success(c, "批阅打分成功，操作痕迹已被高精审计捕获归档！")
}
