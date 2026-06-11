package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/service"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetUserInfo(c *gin.Context) {
	userIDRaw, exists := c.Get("userId")
	if !exists {
		common.Error(c, "登录状态异常")
		return
	}
	userID := userIDRaw.(int64)

	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		common.Error(c, "用户不存在")
		return
	}

	user.OpenID = ""

	// 💡 实时查询该用户已激活的所有云端补签历史日期，打包下发
	var makeupLogs []models.UserMakeupLog
	var unlockedDates []string = []string{}
	if err := global.DB.Where("user_id = ?", userID).Find(&makeupLogs).Error; err == nil {
		for _, log := range makeupLogs {
			unlockedDates = append(unlockedDates, log.MakeupDate)
		}
	}
	user.UnlockedMakeupDays = unlockedDates

	common.Success(c, user)
}

func GetUserCheckinHistory(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	var records []models.CheckinRecord
	global.DB.Where("user_id = ?", userID).Order("create_time desc").Find(&records)
	common.Success(c, records)
}

func GetScoreInfo(c *gin.Context) {
	userID := c.MustGet("userId").(int64)

	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		common.Error(c, "用户状态异常")
		return
	}

	var empLogs []models.ScoreLog
	global.DB.Where("user_id = ? AND score_type IN ('EMP', 'EXCHANGE')", userID).Order("create_time desc").Find(&empLogs)

	var moralLogs []models.ScoreLog
	global.DB.Where("user_id = ? AND score_type = ?", userID, "MORAL").Order("create_time desc").Find(&moralLogs)

	common.Success(c, gin.H{
		"userType":         user.UserType,
		"studentId":        user.StudentID,
		"trackModifyCount": user.TrackModifyCount,
		"empowermentScore": user.EmpowermentScore,
		"moralScore":       user.MoralScore,
		"empLogs":          empLogs,
		"moralLogs":        moralLogs,
	})
}

func GetDashboardStats(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	var empowerSum int
	global.DB.Model(&models.CheckinRecord{}).
		Where("user_id = ? AND checkin_date = ? AND status = 1", userID, yesterday).
		Select("COALESCE(SUM(ai_score), 0)").Scan(&empowerSum)

	var moralSum int = 0
	global.DB.Model(&models.ScoreLog{}).
		Where("user_id = ? AND score_type = 'MORAL' AND DATE(create_time) = ?", userID, yesterday).
		Select("COALESCE(SUM(score), 0)").Scan(&moralSum)

	ctx := context.Background()
	today := time.Now().Format("2006-01-02")
	limitKey := fmt.Sprintf("post_limit:%d:%s", userID, today)
	remainingCount := 1
	if global.Redis.Exists(ctx, limitKey).Val() > 0 {
		remainingCount = 0
	}

	var todayCheckinCount int64
	global.DB.Model(&models.CheckinRecord{}).
		Where("user_id = ? AND checkin_date = ?", userID, today).Count(&todayCheckinCount)

	todayProgress := "未打卡"
	if todayCheckinCount > 0 {
		todayProgress = "已完成"
	}

	var currentUser models.User
	global.DB.Select("empowerment_score").Where("id = ?", userID).First(&currentUser)

	var higherCount int64
	global.DB.Model(&models.User{}).
		Where("empowerment_score > ? AND account_status = 0 AND user_type = 0", currentUser.EmpowermentScore).
		Count(&higherCount)
	currentRank := higherCount + 1

	common.Success(c, gin.H{
		"yesterdayEmpowerment": empowerSum,
		"yesterdayMoral":       moralSum,
		"remainingPostCount":   remainingCount,
		"todayProgress":        todayProgress,
		"currentRank":          currentRank,
	})
}


type UpdateAvatarReq struct {
	AvatarUrl string `json:"avatar_url"`
}

func UpdateUserAvatar(c *gin.Context) {
	userID := c.MustGet("userId").(int64)

	var req UpdateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		var fallback map[string]string
		c.ShouldBindJSON(&fallback)
		req.AvatarUrl = fallback["avatar"]
	}
	if req.AvatarUrl == "" {
		common.Error(c, "未提供新头像链接")
		return
	}

	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		common.Error(c, "用户身份异常")
		return
	}

	oldAvatar := user.Avatar
	if err := global.DB.Model(&user).Update("avatar", req.AvatarUrl).Error; err != nil {
		common.Error(c, "头像数据库同步失败")
		return
	}

	if oldAvatar != "" && strings.Contains(oldAvatar, global.Config.Aliyun.OSS.Endpoint) {
		go func(targetUrl string) {
			ossCfg := global.Config.Aliyun.OSS
			client, err := oss.New(ossCfg.Endpoint, ossCfg.AccessKeyID, ossCfg.AccessKeySecret)
			if err != nil {
				return
			}
			bucket, err := client.Bucket(ossCfg.BucketName)
			if err != nil {
				return
			}
			prefix := fmt.Sprintf("https://%s.%s/", ossCfg.BucketName, ossCfg.Endpoint)
			objectKey := strings.TrimPrefix(targetUrl, prefix)
			if objectKey != targetUrl {
				_ = bucket.DeleteObject(objectKey)
			}
		}(oldAvatar)
	}
	common.Success(c, "头像更新成功")
}

func UpdateUserProfile(c *gin.Context) {
	common.Error(c, "系统已开启强实名底库强制校验，禁止二次修改身份信息，如有错漏请联系系部教务处")
}

func Logout(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	global.DB.Model(&models.User{}).Where("id = ?", userID).Update("current_device_id", "")
	common.Success(c, "设备已安全解绑")
}

func RequestCancelAccount(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	global.DB.Model(&models.User{}).Where("id = ?", userID).Update("account_status", 1)
	common.Success(c, "账户已安全进入静默挂起期")
}

type ChangeTracksReq struct {
	UserID   int64 `json:"userId"` // 💡 完美对齐管理端
	TrackIds []int `json:"trackIds"`
}

func ChangeUserTracks(c *gin.Context) {
	var req ChangeTracksReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.TrackIds) < 2 {
		common.Error(c, "请至少选择2个赛道")
		return
	}

	// 如果没有传特权参数，则降级为当前登录的学生JWT
	targetUserID := req.UserID
	if targetUserID == 0 {
		targetUserID = c.MustGet("userId").(int64)
	}

	var user models.User
	if err := global.DB.First(&user, targetUserID).Error; err != nil {
		common.Error(c, "对应档案目标不存在")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if req.UserID == 0 { // 说明是学生自己改，扣减额度
			if user.TrackModifyCount <= 0 {
				return fmt.Errorf("修改机会已用尽，无法再次调整赛道")
			}
			if err := tx.Model(&user).UpdateColumn("track_modify_count", gorm.Expr("track_modify_count - 1")).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ?", targetUserID).Delete(&models.UserTrack{}).Error; err != nil {
			return err
		}
		for _, tid := range req.TrackIds {
			if err := tx.Create(&models.UserTrack{UserID: targetUserID, TrackID: tid}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, err.Error())
		return
	}
	common.Success(c, "赛道重组成功，已重新配置！")
}

// 💯 彻底根治白屏：全量向 Web 管理端暴露蛇形下划线原生对象，并深度反补级联賽道
func GetAdminUserList(c *gin.Context) {
	var list []map[string]interface{}
	if err := global.DB.Table("user").Order("id DESC").Find(&list).Error; err != nil {
		common.Error(c, "获取人员大名单失败")
		return
	}

	// 高级反补：级联取出每个人报名的赛道，挂载到 user.tracks 下供 Vue 看板渲染
	for _, u := range list {
		uid := u["id"].(int64)
		var tracks []map[string]interface{}
		querySql := `SELECT t.id, t.name FROM track t INNER JOIN user_track ut ON t.id = ut.track_id WHERE ut.user_id = ?`
		global.DB.Raw(querySql, uid).Scan(&tracks)
		u["tracks"] = tracks
	}

	common.Success(c, list)
}

// 👑 特权指令1：大管理员专享特权资料订正通道
type AdminUpdateReq struct {
	ID        int64  `json:"id"`
	RealName  string `json:"real_name"`
	ClassName string `json:"class_name"`
	StudentID string `json:"student_id"`
}

func AdminUpdateStudentInfo(c *gin.Context) {
	var req AdminUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		common.Error(c, "更正请求参数残缺")
		return
	}

	updateFields := map[string]interface{}{
		"real_name":  req.RealName,
		"nickname":   req.RealName,
		"class_name": req.ClassName,
		"student_id": req.StudentID,
	}

	if err := global.DB.Table("user").Where("id = ?", req.ID).Updates(updateFields).Error; err != nil {
		common.Error(c, "学籍基础档案更正落库遭遇失败")
		return
	}
	common.Success(c, "特权底册资料已订正成功")
}

// 👑 特权指令2：体验账户沙盒隔离专区——待审列表拉取
func GetPendingExperienceList(c *gin.Context) {
	var pendingUsers []map[string]interface{}
	// 💡 架构升级：拉取所有处于待审状态 (account_status = 1) 的普通用户进行放行裁决！
	err := global.DB.Table("user").
		Where("account_status = 1").
		Order("id DESC").Find(&pendingUsers).Error

	if err != nil {
		common.Error(c, "沙盒待审池数据同步失败")
		return
	}
	common.Success(c, pendingUsers)
}

// 👑 特权指令3：体验账户沙盒隔离专区——特权裁决决断处理
type ProcessExpReq struct {
	UserID int64  `json:"userId"`
	Action string `json:"action"` // 'pass' = 一键核准放行, 'reject' = 直接拒绝抹除
}

func ProcessExperienceApproval(c *gin.Context) {
	var req ProcessExpReq
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		common.Error(c, "解析裁决指令失败")
		return
	}

	if req.Action == "pass" {
		var targetUser models.User
		if err := global.DB.Table("user").Where("id = ?", req.UserID).First(&targetUser).Error; err != nil {
			common.Error(c, "未找到该待批用户学籍档案")
			return
		}

		updates := map[string]interface{}{
			"account_status": 0, // 💡 彻底放行：必须更新为 0 才能成功登录系统！
		}
		// 只有体验账户才更新班级为“体验放行班”，正规学生（如网络C225）保持原班级不变！
		if targetUser.ClassName == "体验账户" {
			updates["class_name"] = "体验放行班"
		}

		if err := global.DB.Table("user").Where("id = ?", req.UserID).Updates(updates).Error; err != nil {
			common.Error(c, "核准放行通信失败: "+err.Error())
			return
		}
		common.Success(c, "账户特权核准放行成功，已放开登录限制")
	} else {
		// 拒绝则直接物理蒸发抹除
		global.DB.Table("user").Where("id = ?", req.UserID).Delete(&models.User{})
		common.Success(c, "沙盒垃圾申请记录已物理抹除干净")
	}
}

// BindWechatOpenID 微信小程序静默绑定用户的真实 OpenID
func BindWechatOpenID(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "解析绑定参数失败: "+err.Error())
		return
	}

	userID := c.MustGet("userId").(int64)

	openid, err := service.GetOpenIDByCode(req.Code)
	if err != nil {
		common.Error(c, "获取微信 OpenID 失败: "+err.Error())
		return
	}

	// 执行绑定逻辑
	if err := global.DB.Table("user").Where("id = ?", userID).Update("openid", openid).Error; err != nil {
		common.Error(c, "绑定微信 OpenID 失败: "+err.Error())
		return
	}

	common.Success(c, "微信静默绑定完成")
}
