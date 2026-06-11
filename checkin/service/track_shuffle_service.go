package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"log/slog"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// ExecSubAdminTrackShuffle 凌晨 2:00 执行的干事交叉洗牌算法
func ExecSubAdminTrackShuffle() {
	slog.Info("[洗牌引擎] 开始执行二级干事赛道负责权强制打乱...")

	// 1. 获取所有正常状态的二级干事 (RoleLevel = 2)
	var subAdmins []models.AdminUser
	global.DB.Where("role_level = 2 AND status = 0").Find(&subAdmins)
	if len(subAdmins) == 0 {
		return
	}

	// 2. 获取所有开放的赛道
	var tracks []models.Track
	global.DB.Where("is_active = 1").Find(&tracks)
	if len(tracks) == 0 {
		return
	}

	// 3. 将赛道 ID 打乱
	trackIDs := make([]int, len(tracks))
	for i, t := range tracks {
		trackIDs[i] = t.ID
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(trackIDs), func(i, j int) { trackIDs[i], trackIDs[j] = trackIDs[j], trackIDs[i] })

	// 4. 开启事务，清空旧关系，平均分配新关系
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		tx.Exec("DELETE FROM sys_admin_track_relation") // 清空昨天的负责关系

		// 轮询分配：将打乱后的赛道均分给各个干事
		for i, trackID := range trackIDs {
			admin := subAdmins[i%len(subAdmins)]
			relation := models.SysAdminTrackRelation{
				AdminID: admin.ID,
				TrackID: int64(trackID),
			}
			if err := tx.Create(&relation).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		slog.Error("[洗牌引擎] 洗牌失败", "err", err)
	} else {
		slog.Info("[洗牌引擎] 盲选洗牌完毕，已切断潜在作弊利益链")
		// 写系统级审计日志
		RecordAdminAuditLog(0, "SYSTEM_CRON", "赛道干事洗牌", "ALL", "ALL", 0, "每日防作弊换防", "洗牌算法执行成功")
	}
}
