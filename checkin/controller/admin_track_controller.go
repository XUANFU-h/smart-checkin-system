package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// TrackStatsResp 赛道图表透视结构体
type TrackStatsResp struct {
	TrackID           int              `json:"trackId"`
	TrackName         string           `json:"trackName"`
	Description       string           `json:"description"`
	RequireType       string           `json:"requireType"`
	TotalParticipants int64            `json:"totalParticipants"`
	SubmittedCount    int64            `json:"submittedCount"`
	UnsubmittedCount  int64            `json:"unsubmittedCount"`
	RoseData          map[string]int64 `json:"roseData"`
}

// GetTrackStats 提供大盘 2X2 静态刷新的 ECharts 统计数据
func GetTrackStats(c *gin.Context) {
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	var tracks []models.Track
	if err := global.DB.Find(&tracks).Error; err != nil {
		common.Error(c, "获取赛道失败: "+err.Error())
		return
	}

	var stats []TrackStatsResp = []TrackStatsResp{}
	for _, t := range tracks {
		var total int64
		global.DB.Model(&models.UserTrack{}).Where("track_id = ?", t.ID).Count(&total)

		var submitted int64
		global.DB.Model(&models.CheckinRecord{}).Where("track_id = ? AND checkin_date = ?", t.ID, date).Count(&submitted)

		// 质量区间扇区统计
		var records []models.CheckinRecord
		global.DB.Where("track_id = ? AND checkin_date = ?", t.ID, date).Find(&records)

		rose := map[string]int64{
			"score_gt_20": 0,
			"score_15_20": 0,
			"score_10_15": 0,
			"score_5_10":  0,
			"score_lt_5":  0,
			"passed":      0,
			"rejected":    0,
		}

		for _, r := range records {
			if r.Status == 2 {
				rose["rejected"]++
			} else if r.AiScore > 20 {
				rose["score_gt_20"]++
			} else if r.AiScore >= 15 && r.AiScore <= 20 {
				rose["score_15_20"]++
			} else if r.AiScore >= 10 && r.AiScore < 15 {
				rose["score_10_15"]++
			} else if r.AiScore >= 5 && r.AiScore < 10 {
				rose["score_5_10"]++
			} else if r.Status == 1 {
				rose["passed"]++
			} else {
				rose["score_lt_5"]++
			}
		}

		stats = append(stats, TrackStatsResp{
			TrackID:           t.ID,
			TrackName:         t.Name,
			Description:       t.Description,
			RequireType:       t.RequireType,
			TotalParticipants: total,
			SubmittedCount:    submitted,
			UnsubmittedCount:  total - submitted,
			RoseData:          rose,
		})
	}
	common.Success(c, stats)
}

// GetTrackSubmissionStatus 获取名单墙名单
func GetTrackSubmissionStatus(c *gin.Context) {
	trackID := c.Query("track_id")
	date := c.Query("date")
	if trackID == "" || date == "" {
		common.Error(c, "缺少必要参数 track_id 或 date")
		return
	}

	// 1. 已提交名单 (INNER JOIN)
	var submitted []map[string]interface{} = []map[string]interface{}{}
	global.DB.Table("checkin_record cr").
		Select("u.id, u.nickname, u.real_name, u.student_id AS studentId, u.class_name AS className, cr.create_time AS createTime, cr.ai_score AS aiScore").
		Joins("INNER JOIN user u ON cr.user_id = u.id").
		Where("cr.track_id = ? AND cr.checkin_date = ?", trackID, date).
		Order("cr.create_time ASC").
		Scan(&submitted)

	// 2. 未提交名单 (大事务左外连接反查排除，排除管理员并确保账号状态合规)
	var unsubmitted []map[string]interface{} = []map[string]interface{}{}
	global.DB.Table("user_track ut").
		Select("u.id, u.nickname, u.real_name, u.student_id AS studentId, u.class_name AS className").
		Joins("INNER JOIN user u ON ut.user_id = u.id").
		Joins("LEFT JOIN checkin_record cr ON ut.user_id = cr.user_id AND ut.track_id = cr.track_id AND cr.checkin_date = ?", date).
		Where("ut.track_id = ? AND cr.id IS NULL AND u.user_type != 1 AND u.account_status = 0", trackID).
		Scan(&unsubmitted)

	common.Success(c, gin.H{
		"submitted":   submitted,
		"unsubmitted": unsubmitted,
	})
}

// TrackSaveReq 赛道保存请求体
type TrackSaveReq struct {
	ID           int      `json:"id"`
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	RequireTypes []string `json:"requireTypes"` // 接收多选上传契约数组
	IsActive     bool     `json:"isActive"`
}

// SaveTrack 赛道增加/修改 (原子中枢配置)
func SaveTrack(c *gin.Context) {
	var req TrackSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败: "+err.Error())
		return
	}

	requireStr := strings.Join(req.RequireTypes, ",") // 自动转义为逗号分隔的 varchar 字符串

	var track models.Track
	if req.ID > 0 {
		if err := global.DB.First(&track, req.ID).Error; err != nil {
			common.Error(c, "赛道不存在")
			return
		}
	}

	track.Name = req.Name
	track.Description = req.Description
	track.RequireType = requireStr
	track.IsActive = req.IsActive

	if err := global.DB.Save(&track).Error; err != nil {
		common.Error(c, "保存赛道失败: "+err.Error())
		return
	}
	common.Success(c, "赛道配置已更新")
}

// DeleteTrack 废弃赛道级联删除
func DeleteTrack(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	if err := global.DB.Delete(&models.Track{}, req.ID).Error; err != nil {
		common.Error(c, "删除赛道失败: "+err.Error())
		return
	}
	common.Success(c, "赛道已被彻底废弃并移除")
}
