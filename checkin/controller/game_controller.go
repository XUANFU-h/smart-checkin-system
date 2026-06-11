package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== 1. 学生端：打卡后【千人千面】智能抽奖 ====================

func DrawLotteryResult(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	today := time.Now().Format("2006-01-02")
	ctx := context.Background()

	// 1. 高并发防重锁
	lockKey := fmt.Sprintf("lottery_lock:%d", userID)
	acquired, _ := global.Redis.SetNX(ctx, lockKey, 1, 2*time.Second).Result()
	if !acquired {
		common.Error(c, "手速太快啦，机器还在运转")
		return
	}
	defer global.Redis.Del(ctx, lockKey)

	// 2. 检查今日是否已抽取
	dailyLimitKey := fmt.Sprintf("lottery_done:%d:%s", userID, today)
	if global.Redis.Exists(ctx, dailyLimitKey).Val() > 0 {
		common.Error(c, "您今日的抽奖机会已用完，明天打卡后再来吧！")
		return
	}

	// 3. 检查今日是否已真实打卡
	var checkinCount int64
	global.DB.Model(&models.CheckinRecord{}).Where("user_id = ? AND checkin_date = ?", userID, today).Count(&checkinCount)
	if checkinCount == 0 {
		common.Error(c, "系统未检测到您今日的打卡记录，无法开启奖励盲盒！")
		return
	}

	// 4. 💡 核心隐形风控引擎：按劳分配的概率计算
	var trackCount int64
	global.DB.Model(&models.UserTrack{}).Where("user_id = ?", userID).Count(&trackCount)

	rand.Seed(time.Now().UnixNano())
	roll := rand.Intn(100) // 掷骰子 0-99

	score := 0
	if trackCount >= 3 {
		// 👑 高活跃用户 (3条及以上赛道)：绝对不会谢谢参与！
		// 70% 概率爆出大奖(10-20分)，30% 概率小奖(1-9分)
		if roll < 70 {
			score = rand.Intn(11) + 10 // [0-10] + 10 = 10~20
		} else {
			score = rand.Intn(9) + 1 // [0-8] + 1 = 1~9
		}
	} else {
		// 🚶 基础活跃用户 (1-2条赛道)
		// 40% 谢谢参与，50% 小奖(1-5分)，10% 惊喜(6-10分)
		if roll < 40 {
			score = 0
		} else if roll < 90 {
			score = rand.Intn(5) + 1 // 1~5
		} else {
			score = rand.Intn(5) + 6 // 6~10
		}
	}

	// 5. 数据落库与资金变动
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if score > 0 {
			if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", score, userID).Error; err != nil {
				return err
			}
			log := models.ScoreLog{
				UserID:    userID,
				ScoreType: "EMP",
				Score:     score,
				Action:    "ADD",
				Reason:    "打卡成功幸运盲盒结算",
				Operator:  "系统算力分配",
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}
		global.Redis.Set(ctx, dailyLimitKey, 1, 24*time.Hour)
		return nil
	})

	if err != nil {
		common.Error(c, "发奖模块异常，请联系系统管理员")
		return
	}

	// 6. 构造返回报文
	isReward := score > 0
	prizeName := "谢谢参与"
	if isReward {
		prizeName = fmt.Sprintf("%d 赋能分", score)
	}

	common.Success(c, gin.H{
		"isReward":  isReward,
		"prizeName": prizeName,
	})
}

// ==================== 2. 创意工坊：老虎机大乐透 ====================

func maskName(name string) string {
	runes := []rune(name)
	if len(runes) <= 1 {
		return string(runes) + "*"
	}
	return string(runes[0]) + "*" + string(runes[len(runes)-1])
}

func checkFiveDayStreak(userID int64) bool {
	today := time.Now()
	for i := 0; i < 5; i++ {
		targetDate := today.AddDate(0, 0, -i).Format("2006-01-02")
		var count int64
		global.DB.Model(&models.CheckinRecord{}).Where("user_id = ? AND checkin_date = ? AND status = 1", userID, targetDate).Count(&count)
		if count == 0 {
			return false
		}
	}
	return true
}

func checkTodayAllTracksCheckedIn(userID int64) (isEligible bool, trackCount int64, todayCheckedInCount int64) {
	// 1. 获取用户参与的赛道数量
	global.DB.Model(&models.UserTrack{}).Where("user_id = ?", userID).Count(&trackCount)
	if trackCount == 0 {
		return false, 0, 0
	}

	// 2. 获取今日已打卡（待审或通过）的去重赛道数量
	today := time.Now().Format("2006-01-02")
	global.DB.Model(&models.CheckinRecord{}).
		Where("user_id = ? AND checkin_date = ? AND status IN (0, 1)", userID, today).
		Distinct("track_id").
		Count(&todayCheckedInCount)

	isEligible = todayCheckedInCount >= trackCount
	return isEligible, trackCount, todayCheckedInCount
}

// GetLottoActiveStatus 获取大乐透全局开启状态 (open/closed)
func GetLottoActiveStatus() string {
	var cfg models.SystemDynamicConfig
	err := global.DB.Where("config_key = ?", "lotto_active_status").First(&cfg).Error
	if err == nil {
		return cfg.ConfigValue
	}
	return "closed" // 默认未开启
}

func GetActiveWeekNum() int {
	var cfg models.SystemDynamicConfig
	err := global.DB.Where("config_key = ?", "lotto_current_period").First(&cfg).Error
	if err == nil {
		var period int
		if _, err := fmt.Sscanf(cfg.ConfigValue, "%d", &period); err == nil {
			return period
		}
	}
	// 如果配置缺失，初始化为第 1 期开启并自动落库
	defaultPeriod := 1

	cfg = models.SystemDynamicConfig{
		ConfigKey:   "lotto_current_period",
		ConfigValue: fmt.Sprintf("%d", defaultPeriod),
		Description: "大乐透当前管理员开启的活跃期数",
	}
	global.DB.Create(&cfg)
	return defaultPeriod
}

func GetLottoSpinCost() int {
	var cfg models.SystemDynamicConfig
	err := global.DB.Where("config_key = ?", "lotto_spin_cost").First(&cfg).Error
	if err == nil {
		var cost int
		if _, err := fmt.Sscanf(cfg.ConfigValue, "%d", &cost); err == nil {
			return cost
		}
	}
	return 100 // Default cost fallback
}

func GetSlotStatus(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	weekNum := GetActiveWeekNum()

	drawTimeStr := "每周六 10:00"
	var cfg models.SystemDynamicConfig
	if err := global.DB.Where("config_key = ?", "lotto_draw_time").First(&cfg).Error; err == nil {
		drawTimeStr = cfg.ConfigValue
	}

	var entry models.UserLottoEntry
	isLocked := false
	spinCount := 0
	var currentNumbers []string
	if err := global.DB.Where("user_id = ? AND week_num = ?", userID, weekNum).First(&entry).Error; err == nil {
		isLocked = entry.IsLocked
		spinCount = entry.SpinCount
		currentNumbers = strings.Split(entry.LottoNumber, "-")
	}

	remainSpins := 10 - spinCount
	if remainSpins < 0 {
		remainSpins = 0
	}

	isFreeEligible, trackCount, todayCheckedInCount := checkTodayAllTracksCheckedIn(userID)

	var records []models.UserLottoEntry
	global.DB.Where("prize_info != ''").Order("id DESC").Limit(10).Find(&records)

	luckyRecords := make([]map[string]interface{}, 0)
	for _, rec := range records {
		var u models.User
		global.DB.Select("real_name", "nickname").First(&u, rec.UserID)
		name := u.RealName
		if name == "" {
			name = u.Nickname
		}
		if name == "" {
			name = "神秘人"
		}

		luckyRecords = append(luckyRecords, map[string]interface{}{
			"id":       rec.ID,
			"userName": maskName(name),
			"numbers":  rec.LottoNumber,
			"prize":    rec.PrizeInfo,
		})
	}

	// 💯 实时从数据库统计当前周的投注人数和总奖池积分，彻底告别前端任何假数据或本地计算！
	var allEntries []models.UserLottoEntry
	global.DB.Where("week_num = ?", weekNum).Find(&allEntries)
	totalPaidUser := 0
	for _, ent := range allEntries {
		// 检查该用户是否有支付记录（通过ScoreLog查询是否被扣除过赋能分）
		var count int64
		global.DB.Model(&models.ScoreLog{}).
			Where("user_id = ? AND reason = ? AND create_time >= ?", ent.UserID, "大乐透摇号入场券", ent.CreateTime.Add(-24*time.Hour)).
			Count(&count)
		if count > 0 {
			totalPaidUser++
		}
	}
	var totalPool int64
	global.DB.Model(&models.LottoPoolLog{}).Where("week_num = ? AND pool_type = ?", weekNum, "empowerment").Select("COALESCE(SUM(amount), 0)").Scan(&totalPool)

	var totalMoralPool int64
	global.DB.Model(&models.LottoPoolLog{}).Where("week_num = ? AND pool_type = ?", weekNum, "moral").Select("COALESCE(SUM(amount), 0)").Scan(&totalMoralPool)

	workshopActiveStatus := "open" // 默认开启
	var workshopCfg models.SystemDynamicConfig
	if err := global.DB.Where("config_key = ?", "workshop_active_status").First(&workshopCfg).Error; err == nil {
		workshopActiveStatus = workshopCfg.ConfigValue
	}

	common.Success(c, gin.H{
		"remainSpins":         remainSpins,
		"isLocked":            isLocked,
		"currentNumbers":      currentNumbers,
		"hasFiveDayStreak":    isFreeEligible, // 💡 完美向下兼容旧小程序的免费资格字段
		"isFreeEligible":      isFreeEligible,
		"trackCount":          trackCount,
		"todayCheckedInCount": todayCheckedInCount,
		"drawTimeStr":         drawTimeStr,
		"isDrawTime":          time.Now().Weekday() == time.Saturday && time.Now().Hour() >= 22,
		"luckyRecords":        luckyRecords,
		"weekNum":             weekNum,   // 💡 真实当前期数/周数
		"totalPool":           totalPool, // 💡 真实当前总奖池积分
		"totalMoralPool":      totalMoralPool,
		"spinCost":            GetLottoSpinCost(), // 动态积分划扣值
		"activeStatus":        GetLottoActiveStatus(), // 💡 管理员手动大乐透开启状态 (open/closed)
		"workshopActiveStatus": workshopActiveStatus, // 💡 创意工坊开启状态 (open/closed)
	})
}

type SpinReq struct {
	PayType string `json:"payType"` // "free" or "points"
}

func SpinSlotMachine(c *gin.Context) {
	if GetLottoActiveStatus() != "open" {
		common.Error(c, "大中枢提示：当前大乐透活动未开启，请等待管理员手动开启新一期！")
		return
	}

	userID := c.MustGet("userId").(int64)
	weekNum := GetActiveWeekNum()

	var entry models.UserLottoEntry
	hasEntry := false
	if err := global.DB.Where("user_id = ? AND week_num = ?", userID, weekNum).First(&entry).Error; err == nil {
		hasEntry = true
		if entry.IsLocked {
			common.Error(c, "您的本期大乐透凭证已锁死封存，无法继续刷新！")
			return
		}
		if entry.SpinCount >= 10 {
			common.Error(c, "您本期的 10 次刷新额度已用尽！")
			return
		}
	}

	var req SpinReq
	c.ShouldBindJSON(&req)

	isFreeEligible, _, _ := checkTodayAllTracksCheckedIn(userID)

	// 只有在第一次入场刷新时，执行资格和划扣校验
	if !hasEntry {
		if req.PayType == "free" && !isFreeEligible {
			common.Error(c, "您今日打卡尚未全部完成，无法免费参与！")
			return
		}
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 只有在第一次入场，且非免费参与时，根据动态划扣分值扣除
		if !hasEntry && (req.PayType == "points" || (req.PayType == "" && !isFreeEligible)) {
			var user models.User
			tx.Select("empowerment_score").First(&user, userID)
			cost := GetLottoSpinCost()
			if user.EmpowermentScore < cost {
				return fmt.Errorf("您的赋能分不足 %d，无法付费刷新数字", cost)
			}
			if err := tx.Model(&models.User{}).Where("id = ?", userID).UpdateColumn("empowerment_score", gorm.Expr("empowerment_score - ?", cost)).Error; err != nil {
				return err
			}
			log := models.ScoreLog{UserID: userID, ScoreType: "EMP", Score: cost, Action: "SUB", Reason: "大乐透摇号入场券", Operator: "系统扣除"}
			tx.Create(&log)

			poolLog := models.LottoPoolLog{
				WeekNum:     weekNum,
				PoolType:    "empowerment",
				Amount:      cost,
				LogType:     "user_entry",
				UserID:      userID,
				Description: "学生入场摇号积分扣除",
			}
			tx.Create(&poolLog)
		}

		rand.Seed(time.Now().UnixNano())
		n1 := fmt.Sprintf("%02d", rand.Intn(100))
		n2 := fmt.Sprintf("%02d", rand.Intn(100))
		n3 := fmt.Sprintf("%02d", rand.Intn(100))
		numStr := fmt.Sprintf("%s-%s-%s", n1, n2, n3)

		jackpots := map[string]bool{
			"00-00-00": true, "66-66-66": true, "88-88-88": true, "99-99-99": true, "12-34-56": true,
		}

		if jackpots[numStr] {
			if rand.Intn(100) < 85 {
				n1 = fmt.Sprintf("%02d", rand.Intn(100))
				numStr = fmt.Sprintf("%s-%s-%s", n1, n2, n3)
			}
		}

		isInstantWin := false
		if jackpots[numStr] {
			isInstantWin = true
			tx.Model(&models.User{}).Where("id = ?", userID).UpdateColumn("empowerment_score", gorm.Expr("empowerment_score + ?", 200))
			log := models.ScoreLog{UserID: userID, ScoreType: "EMP", Score: 200, Action: "ADD", Reason: fmt.Sprintf("大乐透内幕靓号 [%s] 即时大奖", numStr), Operator: "创意系统"}
			tx.Create(&log)
		}

		if hasEntry {
			entry.LottoNumber = numStr
			entry.SpinCount += 1
			if entry.SpinCount >= 10 {
				entry.IsLocked = true
			}
			if err := tx.Save(&entry).Error; err != nil {
				return err
			}
		} else {
			entry = models.UserLottoEntry{
				UserID:      userID,
				LottoNumber: numStr,
				WeekNum:     weekNum,
				SpinCount:   1,
				IsLocked:    false,
			}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
		}

		common.Success(c, gin.H{
			"numbers":     []string{n1, n2, n3},
			"isEasterEgg": isInstantWin,
			"remainSpins": 10 - entry.SpinCount,
			"spinCount":   entry.SpinCount,
			"isLocked":    entry.IsLocked,
		})

		return nil
	})

	if err != nil {
		common.Error(c, err.Error())
		return
	}
}

func LockSlotNumbers(c *gin.Context) {
	if GetLottoActiveStatus() != "open" {
		common.Error(c, "大中枢提示：当前大乐透活动未开启，请等待管理员手动开启新一期！")
		return
	}

	userID := c.MustGet("userId").(int64)
	weekNum := GetActiveWeekNum()

	var existing models.UserLottoEntry
	if err := global.DB.Where("user_id = ? AND week_num = ?", userID, weekNum).First(&existing).Error; err != nil {
		common.Error(c, "您尚未参与本期大乐透刷新，无法锁定！")
		return
	}

	if existing.IsLocked {
		common.Error(c, "您的号码已经处于锁定状态，不可重复操作！")
		return
	}

	existing.IsLocked = true
	if err := global.DB.Save(&existing).Error; err != nil {
		common.Error(c, "数据库锁定封存失败")
		return
	}
	common.Success(c, "凭证已永久锁定封存入库！")
}

// ==================== 3. 创意工坊：五子棋 ====================

func GomokuWinReward(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	ctx := context.Background()
	today := time.Now().Format("2006-01-02")

	limitKey := fmt.Sprintf("gomoku_win_limit:%d:%s", userID, today)
	count, _ := global.Redis.Get(ctx, limitKey).Int()
	if count >= 3 {
		common.Error(c, "今日五子棋胜场奖励已达上限（3次），请明天再来挑战！")
		return
	}

	global.DB.Transaction(func(tx *gorm.DB) error {
		tx.Model(&models.User{}).Where("id = ?", userID).UpdateColumn("empowerment_score", gorm.Expr("empowerment_score + ?", 10))
		log := models.ScoreLog{UserID: userID, ScoreType: "EMP", Score: 10, Action: "ADD", Reason: "五子棋人机对战获胜", Operator: "系统发奖"}
		return tx.Create(&log).Error
	})

	global.Redis.Incr(ctx, limitKey)
	global.Redis.Expire(ctx, limitKey, 24*time.Hour)

	common.Success(c, "10 赋能分奖励发放成功！")
}

type WinnerItem struct {
	UserID     int64  `json:"userId"`
	PrizeLevel string `json:"prizeLevel"` // "level1", "level2", "level3"
	PrizeLabel string `json:"prizeLabel"`
	EmpPower   int    `json:"empPower"`
	MoralPower int    `json:"moralPower"`
}

type LottoUnifiedReq struct {
	WeekNum       int          `json:"weekNum"`
	IsDrawn       bool         `json:"isDrawn"`
	Winners       []WinnerItem `json:"winners"`
	DrawTimeStr   string       `json:"drawTimeStr"`
	WinningNumber string       `json:"winning_number"`
	PrizeType     string       `json:"prize_type"`
	PrizeValue    int          `json:"prize_value"`
}

func AdminSetLottoSetting(c *gin.Context) {
	var req LottoUnifiedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	// 1. 如果是保存系统时间设置
	if req.DrawTimeStr != "" {
		var cfg models.SystemDynamicConfig
		err := global.DB.Where("config_key = ?", "lotto_draw_time").First(&cfg).Error
		if err != nil {
			cfg = models.SystemDynamicConfig{
				ConfigKey:   "lotto_draw_time",
				ConfigValue: req.DrawTimeStr,
				Description: "大乐透/老虎机自动开奖时间",
			}
			global.DB.Create(&cfg)
		} else {
			cfg.ConfigValue = req.DrawTimeStr
			global.DB.Save(&cfg)
		}
		common.Success(c, "开奖时间策略保存成功")
		return
	}

	// 2. 如果是传统的老版本单纯配置保存 (Winners 为空且没有 DrawTimeStr)
	if len(req.Winners) == 0 {
		var setting models.LottoSetting
		setting.WeekNum = req.WeekNum
		setting.WinningNumber = req.WinningNumber
		setting.PrizeType = req.PrizeType
		setting.PrizeValue = req.PrizeValue
		setting.IsDrawn = req.IsDrawn
		global.DB.Save(&setting)
		common.Success(c, "配置已保存")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 参与奖（阳光普照奖）分值固定为 50 赋能分
		consolationEmp := 50
		consolationMoral := 0

		// 2. 计算本次开奖需要划扣的总额 (💡 仅中奖人，参与奖积分由系统直接发放而不从奖池扣除！)
		requiredEmp := 0
		requiredMoral := 0
		for _, w := range req.Winners {
			requiredEmp += w.EmpPower
			requiredMoral += w.MoralPower
		}

		// 获取所有参与者，用于未中奖者的阳光普照发放
		winnerMap := make(map[int64]WinnerItem)
		for _, w := range req.Winners {
			winnerMap[w.UserID] = w
		}

		var allEntries []models.UserLottoEntry
		tx.Where("week_num = ?", req.WeekNum).Find(&allEntries)

		// 3. 强安全防线：核对德育分与赋能分大奖池余额
		var empSum int64
		tx.Model(&models.LottoPoolLog{}).Where("week_num = ? AND pool_type = ?", req.WeekNum, "empowerment").Select("COALESCE(SUM(amount), 0)").Scan(&empSum)
		
		var moralSum int64
		tx.Model(&models.LottoPoolLog{}).Where("week_num = ? AND pool_type = ?", req.WeekNum, "moral").Select("COALESCE(SUM(amount), 0)").Scan(&moralSum)

		if empSum < int64(requiredEmp) {
			return fmt.Errorf("开奖失败：本期大乐透共需发放 %d 赋能分，当前赋能分奖池余额仅剩 %d 分，余额不足！请手动注入分值后再试！", requiredEmp, empSum)
		}
		if moralSum < int64(requiredMoral) {
			return fmt.Errorf("开奖失败：本期大乐透共需发放 %d 德育分，当前德育分奖池余额仅剩 %d 分，余额不足！请手动注入分值后再试！", requiredMoral, moralSum)
		}

		// 4. 保存/更新开奖记录
		var setting models.LottoSetting
		tx.Where("week_num = ?", req.WeekNum).FirstOrCreate(&setting)
		setting.IsDrawn = true
		
		// 默认以第一个赢家的号码或设定号码作为本周开奖号
		var firstEntry models.UserLottoEntry
		tx.Where("user_id = ? AND week_num = ?", req.Winners[0].UserID, req.WeekNum).First(&firstEntry)
		if firstEntry.LottoNumber != "" {
			setting.WinningNumber = firstEntry.LottoNumber
		} else {
			setting.WinningNumber = "88-88-88"
		}
		tx.Save(&setting)

		// 同步将大乐透活动状态标记为 closed (已圆满开奖，等待管理员手动开启新的一期)
		tx.Exec("INSERT INTO sys_config (config_key, config_value, description) VALUES ('lotto_active_status', 'closed', '状态') ON DUPLICATE KEY UPDATE config_value = 'closed'")

		// 5. 依次向中奖人发放奖励
		for _, w := range req.Winners {
			prizeInfo := fmt.Sprintf("%s (%d 赋能分)", w.PrizeLabel, w.EmpPower)
			if w.MoralPower > 0 {
				if w.EmpPower > 0 {
					prizeInfo = fmt.Sprintf("%s (%d 赋能分 + %d 德育分)", w.PrizeLabel, w.EmpPower, w.MoralPower)
				} else {
					prizeInfo = fmt.Sprintf("%s (%d 德育分)", w.PrizeLabel, w.MoralPower)
				}
			}

			// 给赢家入库
			tx.Table("user_lotto_entry").Where("user_id = ? AND week_num = ?", w.UserID, req.WeekNum).
				Updates(map[string]interface{}{
					"is_winner":  true,
					"prize_info": prizeInfo,
				})
			
			if w.EmpPower > 0 {
				tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", w.EmpPower, w.UserID)
				log := models.ScoreLog{UserID: w.UserID, ScoreType: "EMP", Score: w.EmpPower, Action: "ADD", Reason: "学风大乐透活动幸运中大奖(赋能部分)", Operator: "大中枢系统开奖"}
				tx.Create(&log)
			}
			if w.MoralPower > 0 {
				tx.Exec("UPDATE user SET moral_score = moral_score + ? WHERE id = ?", w.MoralPower, w.UserID)
				log := models.ScoreLog{UserID: w.UserID, ScoreType: "MOR", Score: w.MoralPower, Action: "ADD", Reason: "学风大乐透活动幸运中大奖(德育部分)", Operator: "大中枢系统开奖"}
				tx.Create(&log)
			}
		}

		// 6. 阳光普照奖：本期其他有参与但是未中大奖的同学核发补偿
		for _, entry := range allEntries {
			if _, isWinner := winnerMap[entry.UserID]; !isWinner {
				prizeInfo := fmt.Sprintf("参与奖 (%d 赋能分)", consolationEmp)
				if consolationMoral > 0 {
					if consolationEmp > 0 {
						prizeInfo = fmt.Sprintf("参与奖 (%d 赋能分 + %d 德育分)", consolationEmp, consolationMoral)
					} else {
						prizeInfo = fmt.Sprintf("参与奖 (%d 德育分)", consolationMoral)
					}
				} else if consolationEmp == 0 {
					prizeInfo = "参与勉励"
				}

				tx.Table("user_lotto_entry").Where("id = ?", entry.ID).
					Updates(map[string]interface{}{
						"is_winner":  false,
						"prize_info": prizeInfo,
					})

				if consolationEmp > 0 {
					tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", consolationEmp, entry.UserID)
					log := models.ScoreLog{UserID: entry.UserID, ScoreType: "EMP", Score: consolationEmp, Action: "ADD", Reason: "学风大乐透阳光普照奖(自适分配)", Operator: "大中枢系统开奖"}
					tx.Create(&log)
				}
				if consolationMoral > 0 {
					tx.Exec("UPDATE user SET moral_score = moral_score + ? WHERE id = ?", consolationMoral, entry.UserID)
					log := models.ScoreLog{UserID: entry.UserID, ScoreType: "MOR", Score: consolationMoral, Action: "ADD", Reason: "学风大乐透阳光普照奖(自适分配)", Operator: "大中枢系统开奖"}
					tx.Create(&log)
				}
			}
		}

		// 7. 🚀 扣减并在变动历史表写入派奖的流出变动流水，保持奖池财务平衡！
		if requiredEmp > 0 {
			empPayoutLog := models.LottoPoolLog{
				WeekNum:     req.WeekNum,
				PoolType:    "empowerment",
				Amount:      -requiredEmp,
				LogType:     "user_draw",
				Description: fmt.Sprintf("大乐透开奖派发：%d 赋能分", requiredEmp),
			}
			tx.Create(&empPayoutLog)
		}
		if requiredMoral > 0 {
			moralPayoutLog := models.LottoPoolLog{
				WeekNum:     req.WeekNum,
				PoolType:    "moral",
				Amount:      -requiredMoral,
				LogType:     "user_draw",
				Description: fmt.Sprintf("大乐透开奖派发：%d 德育分", requiredMoral),
			}
			tx.Create(&moralPayoutLog)
		}

		return nil
	})

	if err != nil {
		common.Error(c, "结算失败: "+err.Error())
		return
	}

	common.Success(c, "全服大乐透手动结算派奖大功告成！")
}

// GetLottoDrawResult 学生获取本周大乐透开奖结果
func GetLottoDrawResult(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	year, week := time.Now().ISOWeek()
	weekNum := year*100 + week

	var setting models.LottoSetting
	if err := global.DB.Where("week_num = ? AND is_drawn = 1", weekNum).First(&setting).Error; err != nil {
		common.Error(c, "本期大乐透尚未揭晓，请耐心等待管理员开奖！")
		return
	}

	var entry models.UserLottoEntry
	hasEntry := true
	if err := global.DB.Where("user_id = ? AND week_num = ?", userID, weekNum).First(&entry).Error; err != nil {
		hasEntry = false
	}

	winningNumbers := strings.Split(setting.WinningNumber, "-")
	if len(winningNumbers) < 3 {
		winningNumbers = []string{"88", "88", "88"} // 降级兜底
	}

	prizeName := ""
	if hasEntry {
		prizeName = entry.PrizeInfo
	}

	common.Success(c, gin.H{
		"winningNumbers": winningNumbers,
		"prizeName":      prizeName,
		"hasEntry":       hasEntry,
	})
}

// GetLottoParticipants 管理端拉取本期参与摇号的学生名单
func GetLottoParticipants(c *gin.Context) {
	weekNum := GetActiveWeekNum()

	var list []struct {
		ID                int64  `json:"id"`
		UserID            int64  `json:"userId"`
		RealName          string `json:"realName"`
		Nickname          string `json:"nickname"`
		StudentID         string `json:"studentId"`
		ClassName         string `json:"className"`
		LottoNumber       string `json:"lottoNumber"`
		IsStreak          bool   `json:"isStreak"`
		TrackCount        int    `json:"trackCount"`
		TotalCheckinCount int    `json:"totalCheckinCount"`
	}

	sql := `SELECT e.id, e.user_id, u.real_name, u.nickname, u.student_id, u.class_name as class_name, e.lotto_number,
	        (SELECT COUNT(1) >= 5 FROM checkin_record cr WHERE cr.user_id = e.user_id AND cr.status = 1 AND cr.checkin_date >= DATE_SUB(CURDATE(), INTERVAL 5 DAY)) as is_streak,
	        (SELECT COUNT(1) FROM user_track ut WHERE ut.user_id = e.user_id) as track_count,
	        (SELECT COUNT(1) FROM checkin_record cr WHERE cr.user_id = e.user_id AND cr.status = 1) as total_checkin_count
			FROM user_lotto_entry e JOIN user u ON e.user_id = u.id
			WHERE e.week_num = ? ORDER BY e.id DESC`

	global.DB.Raw(sql, weekNum).Scan(&list)
	common.Success(c, list)
}

// AdminClearCurrentLottoParticipants 一键清空本期所有参玩学生名单
func AdminClearCurrentLottoParticipants(c *gin.Context) {
	weekNum := GetActiveWeekNum()

	// 物理清空当前活跃期数的所有参与记录
	err := global.DB.Where("week_num = ?", weekNum).Delete(&models.UserLottoEntry{}).Error
	if err != nil {
		common.Error(c, "一键清空参玩人员失败: "+err.Error())
		return
	}

	common.Success(c, "本期参玩学生序列已成功全部清空")
}

// AdminGetLottoPoolLogs 获取德育分/赋能分奖池的收支变动账本
func AdminGetLottoPoolLogs(c *gin.Context) {
	weekNumStr := c.Query("weekNum")
	poolType := c.Query("poolType") // "empowerment" or "moral"
	
	var list []models.LottoPoolLog
	db := global.DB.Where("pool_type = ?", poolType)
	if weekNumStr != "" {
		db = db.Where("week_num = ?", weekNumStr)
	}
	db.Order("id DESC").Find(&list)
	common.Success(c, list)
}

// AdminInjectLottoPool 管理员手动注水/扣减奖池积分
func AdminInjectLottoPool(c *gin.Context) {
	var req struct {
		WeekNum     int    `json:"weekNum"`
		PoolType    string `json:"poolType"`    // "empowerment" or "moral"
		Amount      int    `json:"amount"`      // 可正可负
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	log := models.LottoPoolLog{
		WeekNum:     req.WeekNum,
		PoolType:    req.PoolType,
		Amount:      req.Amount,
		LogType:     "admin_inject",
		Description: req.Description,
	}
	if req.Amount < 0 {
		log.LogType = "admin_deduct"
	}

	if err := global.DB.Create(&log).Error; err != nil {
		common.Error(c, "注入失败: "+err.Error())
		return
	}
	common.Success(c, "资金变动注入成功")
}

// AdminStartNextLottoPeriod 手动开启下一期活动 (POST)
func AdminStartNextLottoPeriod(c *gin.Context) {
	currentPeriod := GetActiveWeekNum()
	nextPeriod := currentPeriod + 1

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 更新活跃期数配置
		tx.Exec("INSERT INTO sys_config (config_key, config_value, description) VALUES ('lotto_current_period', ?, '大乐透期数') ON DUPLICATE KEY UPDATE config_value = ?", nextPeriod, nextPeriod)
		
		// 2. 将状态设置为 open
		tx.Exec("INSERT INTO sys_config (config_key, config_value, description) VALUES ('lotto_active_status', 'open', '大乐透状态') ON DUPLICATE KEY UPDATE config_value = 'open'")

		// 3. 在 lotto_setting 中创建一条未开奖的全新纪录
		var setting models.LottoSetting
		err := tx.Where("week_num = ?", nextPeriod).First(&setting).Error
		if err != nil {
			setting = models.LottoSetting{
				WeekNum: nextPeriod,
				IsDrawn: false,
			}
			return tx.Create(&setting).Error
		}
		setting.IsDrawn = false
		return tx.Save(&setting).Error
	})

	if err != nil {
		common.Error(c, "开启新的一期大乐透活动失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{
		"newPeriod": nextPeriod,
	})
}

// GetLottoPrizeConfigs 获取自适应的奖项分值配置
func GetLottoPrizeConfigs(c *gin.Context) {
	var cfg models.SystemDynamicConfig
	err := global.DB.Where("config_key = ?", "lotto_prize_configs").First(&cfg).Error
	if err != nil {
		defaultConfigs := `[
			{"level": "level1", "label": "🥇 一等奖", "rewardType": "empowerment", "empPower": 500, "moralPower": 0, "maxWinners": 2, "color": "text-red-500"},
			{"level": "level2", "label": "🥈 二等奖", "rewardType": "empowerment", "empPower": 200, "moralPower": 0, "maxWinners": 5, "color": "text-orange-500"},
			{"level": "level3", "label": "🥉 三等奖", "rewardType": "empowerment", "empPower": 50, "moralPower": 0, "maxWinners": 15, "color": "text-amber-500"},
			{"level": "level4", "label": "🎁 参与奖", "rewardType": "empowerment", "empPower": 10, "moralPower": 0, "maxWinners": 50, "color": "text-slate-500"}
		]`
		common.Success(c, defaultConfigs)
		return
	}
	common.Success(c, cfg.ConfigValue)
}

// SaveLottoPrizeConfigs 保存自适应的奖项分值配置
func SaveLottoPrizeConfigs(c *gin.Context) {
	var req struct {
		ConfigValue string `json:"configValue"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	var cfg models.SystemDynamicConfig
	err := global.DB.Where("config_key = ?", "lotto_prize_configs").First(&cfg).Error
	if err != nil {
		cfg = models.SystemDynamicConfig{
			ConfigKey:   "lotto_prize_configs",
			ConfigValue: req.ConfigValue,
			Description: "大乐透/老虎机奖项分值与类型配置",
		}
		global.DB.Create(&cfg)
	} else {
		cfg.ConfigValue = req.ConfigValue
		global.DB.Save(&cfg)
	}
	common.Success(c, "大中枢奖项分值比重已动态持久化保存！")
}

// AdminRejectLottoParticipant 一键驳回大乐透参与并原路退回退款大事务
func AdminRejectLottoParticipant(c *gin.Context) {
	var req struct {
		WeekNum int   `json:"weekNum"`
		UserID  int64 `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var entry models.UserLottoEntry
		// 行锁锁定
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ? AND week_num = ?", req.UserID, req.WeekNum).
			First(&entry).Error; err != nil {
			return errors.New("未找到该用户的投注凭证记录")
		}

		if entry.IsLocked && entry.SpinCount == 0 {
			return errors.New("该用户凭证状态异常")
		}

		// A. 判定是否为积分付费参与的 (查询该用户本期在LottoPoolLog中是否存在扣除记录)
		var paidLog models.LottoPoolLog
		err := tx.Where("user_id = ? AND week_num = ? AND pool_type = ? AND log_type = ?", req.UserID, req.WeekNum, "empowerment", "user_entry").First(&paidLog).Error
		if err == nil {
			refundAmount := paidLog.Amount // 实际扣除的分值
			if refundAmount > 0 {
				// 1. 原路退还赋能分
				if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", refundAmount, req.UserID).Error; err != nil {
					return err
				}
				scoreLog := models.ScoreLog{
					UserID:    req.UserID,
					ScoreType: "EMP",
					Score:     refundAmount,
					Action:    "ADD",
					Reason:    "大乐透参与被管理员驳回退还",
					Operator:  "管理员驳回",
				}
				if err := tx.Create(&scoreLog).Error; err != nil {
					return err
				}

				// 2. 扣除大池对应的赋能分记录
				poolLog := models.LottoPoolLog{
					WeekNum:     req.WeekNum,
					PoolType:    "empowerment",
					Amount:      -refundAmount,
					LogType:     "admin_deduct",
					UserID:      req.UserID,
					Description: fmt.Sprintf("管理员驳回，退回%d入场赋能分", refundAmount),
				}
				if err := tx.Create(&poolLog).Error; err != nil {
					return err
				}
			}
		}

		// B. 物理删除投注凭证以退回打卡免票次数
		if err := tx.Delete(&entry).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		common.Error(c, "驳回失败: "+err.Error())
		return
	}
	common.Success(c, "该用户投注已被一键驳回，入场资产与打卡机会已原路奉还！")
}

// AdminDraftLottoDraw 草拟现场开奖名单（含有后端秘密加权算法，打卡勤奋、赛道多的概率更高）
func AdminDraftLottoDraw(c *gin.Context) {
	var req struct {
		WeekNum int `json:"weekNum"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	// 1. 读取系统持久化配置以确定一二三等奖的名额上限
	level1Max := 2
	level2Max := 5
	level3Max := 15

	var prizeCfg models.SystemDynamicConfig
	if err := global.DB.Where("config_key = ?", "lotto_prize_configs").First(&prizeCfg).Error; err == nil {
		type LocalPrizeConfig struct {
			Level      string `json:"level"`
			MaxWinners int    `json:"maxWinners"`
		}
		var localPrizeConfigs []LocalPrizeConfig
		if err := json.Unmarshal([]byte(prizeCfg.ConfigValue), &localPrizeConfigs); err == nil {
			for _, item := range localPrizeConfigs {
				if item.Level == "level1" {
					level1Max = item.MaxWinners
				} else if item.Level == "level2" {
					level2Max = item.MaxWinners
				} else if item.Level == "level3" {
					level3Max = item.MaxWinners
				}
			}
		}
	}

	// 2. 拉取本周所有的有效参与锁定者
	var allEntries []models.UserLottoEntry
	global.DB.Where("week_num = ? AND is_locked = 1", req.WeekNum).Find(&allEntries)
	if len(allEntries) == 0 {
		common.Success(c, []interface{}{})
		return
	}

	// 3. 构建候选人切片并注入权重属性
	type Candidate struct {
		Entry    models.UserLottoEntry
		RealName string
		Nickname string
		Class    string
		Weight   int
	}

	var candidates []Candidate
	for _, e := range allEntries {
		var u models.User
		global.DB.Select("real_name", "nickname", "class_name").First(&u, e.UserID)

		// 统计赛道数与总打卡数
		var trackCount int64
		global.DB.Model(&models.UserTrack{}).Where("user_id = ?", e.UserID).Count(&trackCount)

		var checkinCount int64
		global.DB.Model(&models.CheckinRecord{}).Where("user_id = ? AND status = 1", e.UserID).Count(&checkinCount)

		// 检索该生本期是否为付费入场 (从大池流水中检查)
		var paidCount int64
		global.DB.Model(&models.LottoPoolLog{}).
			Where("user_id = ? AND week_num = ? AND pool_type = ? AND log_type = ?", e.UserID, req.WeekNum, "empowerment", "user_entry").
			Count(&paidCount)
		
		isPaid := paidCount > 0

		// 🎯 秘密内幕核心：计算加权值
		weight := 10 // 积分付费买票的基础权重
		if !isPaid {
			// 打卡免费入场的拥有更高的中奖权重
			weight = 50 + int(trackCount)*20 + int(checkinCount)*5
		}

		name := u.RealName
		if name == "" {
			name = u.Nickname
		}
		if name == "" {
			name = "神秘同学"
		}

		candidates = append(candidates, Candidate{
			Entry:    e,
			RealName: name,
			Nickname: u.Nickname,
			Class:    u.ClassName,
			Weight:   weight,
		})
	}

	// 顺序加权抽选闭包函数 (防重复中奖)
	drawWinners := func(count int, prizeLevel string, prizeLabel string) []WinnerItem {
		var winners []WinnerItem
		rand.Seed(time.Now().UnixNano())

		for i := 0; i < count; i++ {
			if len(candidates) == 0 {
				break
			}

			totalWeight := 0
			for _, c := range candidates {
				totalWeight += c.Weight
			}

			if totalWeight <= 0 {
				break
			}

			rVal := rand.Intn(totalWeight)
			currSum := 0
			selectedIdx := -1

			for idx, c := range candidates {
				currSum += c.Weight
				if currSum > rVal {
					selectedIdx = idx
					break
				}
			}

			if selectedIdx != -1 {
				sel := candidates[selectedIdx]
				winners = append(winners, WinnerItem{
					UserID:     sel.Entry.UserID,
					PrizeLevel: prizeLevel,
					PrizeLabel: prizeLabel,
				})

				// 从候选名单切片中移除
				candidates = append(candidates[:selectedIdx], candidates[selectedIdx+1:]...)
			}
		}
		return winners
	}

	// 依次盲抽 Level 1, 2, 3 中奖者
	winnersLevel1 := drawWinners(level1Max, "level1", "🥇 一等奖")
	winnersLevel2 := drawWinners(level2Max, "level2", "🥈 二等奖")
	winnersLevel3 := drawWinners(level3Max, "level3", "🥉 三等奖")

	// 组装并返回列表给前端草拟面板
	var draftWinners []map[string]interface{}
	addDraft := func(items []WinnerItem) {
		for _, it := range items {
			var u models.User
			global.DB.Select("real_name", "nickname", "class_name").First(&u, it.UserID)
			name := u.RealName
			if name == "" {
				name = u.Nickname
			}
			draftWinners = append(draftWinners, map[string]interface{}{
				"userId":     it.UserID,
				"name":       name,
				"className":  u.ClassName,
				"prizeLevel": it.PrizeLevel,
				"prizeLabel": it.PrizeLabel,
			})
		}
	}

	addDraft(winnersLevel1)
	addDraft(winnersLevel2)
	addDraft(winnersLevel3)

	common.Success(c, draftWinners)
}

// AdminRejectLottoParticipantBatch 批量多选驳回学生资格并原路退款大事务
func AdminRejectLottoParticipantBatch(c *gin.Context) {
	var req struct {
		WeekNum int     `json:"weekNum"`
		UserIDs []int64 `json:"userIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, uID := range req.UserIDs {
			var entry models.UserLottoEntry
			// 行锁物理排他定位
			if err := tx.Set("gorm:query_option", "FOR UPDATE").
				Where("user_id = ? AND week_num = ?", uID, req.WeekNum).
				First(&entry).Error; err != nil {
				continue // 已经删除的直接跳过
			}

			// A. 原路返还积分付费参与的用户
			var paidLog models.LottoPoolLog
			err := tx.Where("user_id = ? AND week_num = ? AND pool_type = ? AND log_type = ?", uID, req.WeekNum, "empowerment", "user_entry").First(&paidLog).Error
			if err == nil {
				refundAmount := paidLog.Amount // 实际扣除的分值
				if refundAmount > 0 {
					if err := tx.Exec("UPDATE user SET empowerment_score = empowerment_score + ? WHERE id = ?", refundAmount, uID).Error; err != nil {
						return err
					}
					scoreLog := models.ScoreLog{
						UserID:    uID,
						ScoreType: "EMP",
						Score:     refundAmount,
						Action:    "ADD",
						Reason:    "大乐透参与被管理员批量驳回退还",
						Operator:  "管理员批量驳回",
					}
					if err := tx.Create(&scoreLog).Error; err != nil {
						return err
					}

					// 扣除大池对应的赋能分记录
					poolLog := models.LottoPoolLog{
						WeekNum:     req.WeekNum,
						PoolType:    "empowerment",
						Amount:      -refundAmount,
						LogType:     "admin_deduct",
						UserID:      uID,
						Description: fmt.Sprintf("管理员批量驳回，退回%d入场赋能分", refundAmount),
					}
					if err := tx.Create(&poolLog).Error; err != nil {
						return err
					}
				}
			}

			// B. 物理销毁投注凭证，在微信小程序重置状态
			if err := tx.Delete(&entry).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "批量驳回失败: "+err.Error())
		return
	}
	common.Success(c, "选定学生的投注凭证已被物理裁撤，资产/打卡机会已批量退回！")
}

// GetLottoSpinCostApi 查询当前付费积分设定
func GetLottoSpinCostApi(c *gin.Context) {
	common.Success(c, GetLottoSpinCost())
}

// SaveLottoSpinCostApi 修改并保存付费积分设定
func SaveLottoSpinCostApi(c *gin.Context) {
	var req struct {
		Cost int `json:"cost"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}
	if req.Cost < 0 {
		common.Error(c, "参与积分不可为负数")
		return
	}

	var cfg models.SystemDynamicConfig
	err := global.DB.Where("config_key = ?", "lotto_spin_cost").First(&cfg).Error
	if err != nil {
		cfg = models.SystemDynamicConfig{
			ConfigKey:   "lotto_spin_cost",
			ConfigValue: fmt.Sprintf("%d", req.Cost),
			Description: "大乐透/老虎机付费参与所需消耗的赋能分分值",
		}
		global.DB.Create(&cfg)
	} else {
		cfg.ConfigValue = fmt.Sprintf("%d", req.Cost)
		global.DB.Save(&cfg)
	}
	common.Success(c, "付费入场积分消耗额度更新成功！")
}

// GetLottoHistoryRecords 获取历史大乐透已开奖红榜归档
func GetLottoHistoryRecords(c *gin.Context) {
	var entries []models.UserLottoEntry
	// 仅拉取一二三等奖的往期赢家
	global.DB.Where("is_winner = 1 AND week_num < ? AND prize_info NOT LIKE ?", GetActiveWeekNum(), "%参与奖%").
		Order("week_num DESC, id DESC").Find(&entries)

	type HistoryItem struct {
		WeekNum   int    `json:"weekNum"`
		UserName  string `json:"userName"`
		ClassName string `json:"className"`
		PrizeInfo string `json:"prizeInfo"`
		Numbers   string `json:"numbers"`
	}

	var result []HistoryItem
	for _, e := range entries {
		var u models.User
		global.DB.Select("real_name", "nickname", "class_name").First(&u, e.UserID)
		name := u.RealName
		if name == "" {
			name = u.Nickname
		}
		if name == "" {
			name = "神秘人"
		}
		result = append(result, HistoryItem{
			WeekNum:   e.WeekNum,
			UserName:  maskName(name),
			ClassName: u.ClassName,
			PrizeInfo: e.PrizeInfo,
			Numbers:   e.LottoNumber,
		})
	}
	common.Success(c, result)
}

// GetWorkshopActiveStatusApi 获取创意工坊整体开启状态 (GET)
func GetWorkshopActiveStatusApi(c *gin.Context) {
	status := "open"
	var cfg models.SystemDynamicConfig
	if err := global.DB.Where("config_key = ?", "workshop_active_status").First(&cfg).Error; err == nil {
		status = cfg.ConfigValue
	}
	common.Success(c, status)
}

// ToggleWorkshopActiveStatusApi 切换并保存创意工坊全局开启状态 (POST)
func ToggleWorkshopActiveStatusApi(c *gin.Context) {
	var req struct {
		Status string `json:"status"` // "open" or "closed"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	if req.Status != "open" && req.Status != "closed" {
		common.Error(c, "非法的开启状态")
		return
	}

	// 💡 物理更新 sys_config 表，保障前后台强一致同步
	err := global.DB.Exec("INSERT INTO sys_config (config_key, config_value, description) VALUES ('workshop_active_status', ?, '创意工坊开启状态') ON DUPLICATE KEY UPDATE config_value = ?", req.Status, req.Status).Error
	if err != nil {
		common.Error(c, "更新创意工坊状态失败: "+err.Error())
		return
	}

	common.Success(c, "创意工坊状态更新成功！")
}

