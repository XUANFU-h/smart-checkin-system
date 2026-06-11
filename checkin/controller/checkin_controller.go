package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var userLocks sync.Map

// CheckDeadline 内部公用方法：检查是否已过今晚的打卡截止时间
func checkDeadline() (bool, string) {
	deadline := "22:00" // 默认22:00
	var cfg models.SystemDynamicConfig
	if err := global.DB.Where("config_key = ?", "checkin_deadline").First(&cfg).Error; err == nil {
		deadline = cfg.ConfigValue
	}

	currentTime := time.Now().Format("15:04")
	return currentTime >= deadline, deadline
}

// SubmitCheckin 对应 /api/checkin/submit
func SubmitCheckin(c *gin.Context) {
	var record models.CheckinRecord
	if err := c.ShouldBindJSON(&record); err != nil {
		common.Error(c, "请求格式或签名错误，拒绝访问！")
		return
	}

	// 🛡️ 网络安全加固：强制使用JWT令牌解出的真实用户ID，杜绝越权伪造轰炸
	userID := c.MustGet("userId").(int64)
	record.UserID = userID

	if strings.TrimSpace(record.ContentText) == "" && record.ImageURL == "" {
		common.Error(c, "打卡内容与图片不能同时为空")
		return
	}

	// 3. 并发锁控制
	lockInterface, _ := userLocks.LoadOrStore(record.UserID, &sync.Mutex{})
	lock := lockInterface.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	// 4. 防并发连点 (3秒内是否有过提交)
	var recentSubmits int64
	global.DB.Model(&models.CheckinRecord{}).
		Where("user_id = ? AND create_time >= DATE_SUB(NOW(), INTERVAL 3 SECOND)", record.UserID).
		Count(&recentSubmits)

	if recentSubmits > 0 {
		common.Error(c, "提交过于频繁！系统已自动拦截多设备并发操作。")
		return
	}

	// ⏰ 核心安全逻辑：区分【当日正常打卡】与【历史卡片补签】
	today := time.Now().Format("2006-01-02")
	if record.CheckinDate == "" {
		record.CheckinDate = today
	}

	if record.CheckinDate == today {
		// 【当日正常打卡流程】受22:00死线严格限制
		isLate, deadlineTime := checkDeadline()
		if isLate {
			common.Error(c, "今日打卡通道已于 "+deadlineTime+" 关闭，目前系统正由管理员进行全量批阅，请明日再来！")
			return
		}
	} else {
		// 【历史缺卡补签流程】不受当日死线限制，但必须先使用补签卡开通指定日期的全赛道通行权限
		if record.CheckinDate > today {
			common.Error(c, "非法的打卡日期，无法预测未来打卡！")
			return
		}
		var makeupLogCount int64
		global.DB.Table("user_makeup_log").
			Where("user_id = ? AND makeup_date = ?", record.UserID, record.CheckinDate).
			Count(&makeupLogCount)

		if makeupLogCount == 0 {
			common.Error(c, "打卡通道已关闭！如需补签历史卡片，请在“我的”页面先消耗一张补签卡开通历史权限。")
			return
		}
	}

	// 5. 校验今天该日期下该赛道是否已打过卡
	var trackDailyCount int64
	global.DB.Model(&models.CheckinRecord{}).
		Where("user_id = ? AND track_id = ? AND checkin_date = ?", record.UserID, record.TrackID, record.CheckinDate).
		Count(&trackDailyCount)
	if trackDailyCount > 0 {
		common.Error(c, "您在所选的日期已经在这条赛道打过卡了！请勿重复提交。")
		return
	}

	// 6. AI 实时风控
	// 🛑 静默熔断：由于取消了 AI 大模型的额度消耗，将风控校验暂时注释，默认放行。
	// if !service.CheckPayloadSafe(record.ContentText) {
	// 	record.Status = 4
	// 	record.AiComment = "系统安全官检测到您的提交内容存在恶意乱码或注入风险，已自动隔离。"
	// }

	// 7. 落库
	if err := global.DB.Create(&record).Error; err != nil {
		common.Error(c, "保存打卡记录失败")
		return
	}

	common.Success(c, map[string]interface{}{
		"message": "打卡提交成功！已并入核验流水线。",
	})
}

// UpdateCheckin 修改打卡 (时间截止前)
func UpdateCheckin(c *gin.Context) {
	var req models.CheckinRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	isLate, _ := checkDeadline()
	if isLate {
		common.Error(c, "打卡通道已关闭，无法修改当日记录！")
		return
	}

	userID := c.MustGet("userId").(int64)
	global.DB.Model(&models.CheckinRecord{}).
		Where("id = ? AND user_id = ? AND checkin_date = ?", req.ID, userID, time.Now().Format("2006-01-02")).
		Updates(map[string]interface{}{
			"content_text": req.ContentText,
			"image_url":    req.ImageURL,
			"status":       0,
		})

	common.Success(c, "修改成功！")
}

// ActivateMakeupCard 消耗1张补签卡，解锁历史某一天全赛道补签通行权限
func ActivateMakeupCard(c *gin.Context) {
	var req struct {
		Date       string `json:"date"`       // 💡 对齐微信小程序发送的 JSON 字段 "date"
		TargetDate string `json:"targetDate"` // 兼容历史格式
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "请求数据结构错误")
		return
	}

	// 💡 自动取值对齐，杜绝因参数名不一致造成的“未指定目标解锁日期”报错
	if req.TargetDate == "" {
		req.TargetDate = req.Date
	}

	if req.TargetDate == "" {
		common.Error(c, "未指定目标解锁日期")
		return
	}

	userID := c.MustGet("userId").(int64)
	today := time.Now().Format("2006-01-02")

	if req.TargetDate >= today {
		common.Error(c, "补签卡只能用于激活今天之前的历史缺卡日期！")
		return
	}

	// ⚡ 强一致性本地事务启动：严格对抗并发连击、脚本重放
	tx := global.DB.Begin()

	// 1. 悲观锁锁定学生核心账户行
	var user models.User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", userID).First(&user).Error; err != nil {
		tx.Rollback()
		common.Error(c, "账户核验失败")
		return
	}

	if user.MakeupCards <= 0 {
		tx.Rollback()
		common.Error(c, "您当前手中没有可用的补签卡，请前往学风指数商城兑换！")
		return
	}

	// 2. 尝试抢占插入补签白名单日志（依靠唯一索引 uk_user_makeup_date 实现原子幂等，拒绝重复刷卡）
	makeupLog := models.UserMakeupLog{
		UserID:     userID,
		MakeupDate: req.TargetDate,
	}
	if err := tx.Create(&makeupLog).Error; err != nil {
		tx.Rollback()
		common.Error(c, "该历史缺卡日期已激活补签权限，无需重复消耗卡片！")
		return
	}

	// 3. 安全扣除1张卡片
	if err := tx.Model(&user).Update("makeup_cards", user.MakeupCards-1).Error; err != nil {
		tx.Rollback()
		common.Error(c, "扣除补签卡数量失败，系统回滚")
		return
	}

	tx.Commit()
	common.Success(c, "补签卡激活成功！已为您开通了 "+req.TargetDate+" 全天所有赛道的历史补签放行权限。")
}

// GetMyTrackRecords 对应体验版/线上版小程序首页日历及列表同步接口 /api/checkin/my
func GetMyTrackRecords(c *gin.Context) {
	userID := c.MustGet("userId").(int64) // 🛡️ 强制在网关层采用解密后的真实令牌ID隔离越权

	trackID := c.Query("trackId")
	if trackID == "" {
		common.Error(c, "缺少赛道参数")
		return
	}

	var records []models.CheckinRecord
	// 💡 生产级自愈核心：主页日历及列表需要实时渲染学生各赛道的当日完成度（变绿/变黄），因此不应过滤 publish_status = 1
	err := global.DB.Where("user_id = ? AND track_id = ?", userID, trackID).
		Order("create_time DESC").
		Find(&records).Error

	if err != nil {
		common.Error(c, "同步打卡历史失败")
		return
	}

	common.Success(c, records)
}

// GetCheckinRecordDetail 学生拉取单条打卡回显详情
func GetCheckinRecordDetail(c *gin.Context) {
	idStr := c.Query("id")
	var recordID int64
	if idStr != "" {
		fmt.Sscanf(idStr, "%d", &recordID)
	}

	if recordID == 0 {
		common.Error(c, "缺少记录标识 id")
		return
	}

	userID := c.MustGet("userId").(int64)

	var record models.CheckinRecord
	if err := global.DB.First(&record, recordID).Error; err != nil {
		common.Error(c, "记录不存在或已被撤销")
		return
	}

	// 横向越权核验：仅允许本人查询
	if record.UserID != userID {
		common.Error(c, "安全阻断：无权读取他人打卡记录详情！")
		return
	}

	common.Success(c, record)
}
