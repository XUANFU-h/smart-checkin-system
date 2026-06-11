package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== 1. 学生端：社区与帖子 ====================

// GetCommunityPosts 获取信息流 (支持点赞评论拉取)
func GetCommunityPosts(c *gin.Context) {
	channel := c.DefaultQuery("channel", "recommend")
	userID := c.MustGet("userId").(int64)

	var posts []map[string]interface{}

	// 联合查询出点赞数、评论数，并验证当前用户是否点赞
	postSql := `SELECT p.id, p.post_type as type, p.author_id as userId, p.author_name as userName, 
				p.author_avatar as avatarUrl, p.content, p.image_url as imageUrl, 
				p.is_pinned as isPinned, p.admin_comment as adminComment, 
				p.like_count as likeCount, p.comment_count as commentCount,
				(SELECT count(*) FROM community_like l WHERE l.post_id = p.id AND l.user_id = ?) > 0 as isLiked,
				p.create_time as createTime 
				FROM community_post p 
				WHERE p.channel = ? 
				ORDER BY p.is_pinned DESC, p.create_time DESC LIMIT 50`

	global.DB.Raw(postSql, userID, channel).Scan(&posts)

	common.Success(c, posts)
}

// PublishUserPost 用户自由发布动态 (附带 AI 风控)
type PublishPostReq struct {
	Content  string `json:"content"`
	ImageUrl string `json:"imageUrl"`
	TrackID  int    `json:"trackId"` // 强制要求绑定赛道
}

func PublishUserPost(c *gin.Context) {
	var req PublishPostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	userID := c.MustGet("userId").(int64)

	// 1. 校验用户是否真的报名了该赛道 (防黑客越权请求)
	var count int64
	global.DB.Model(&models.UserTrack{}).Where("user_id = ? AND track_id = ?", userID, req.TrackID).Count(&count)
	if count == 0 {
		common.Error(c, "越权拦截：您并未报名参与该赛道！")
		return
	}

	// 2. 查询用户当前封禁状态与违规次数
	var userStatus struct {
		Nickname           string
		Avatar             string
		PostViolationCount int
		IsPostBanned       int8
	}
	global.DB.Table("user").Select("nickname, avatar, post_violation_count, is_post_banned").Where("id = ?", userID).Scan(&userStatus)

	if userStatus.IsPostBanned == 1 {
		common.Error(c, "系统拦截：您的发帖功能已被永久封禁！")
		return
	}

	// 3. 校验每天一帖频控
	ctx := context.Background()
	today := time.Now().Format("2006-01-02")
	limitKey := fmt.Sprintf("post_limit:%d:%s", userID, today)
	if global.Redis.Exists(ctx, limitKey).Val() > 0 {
		common.Error(c, "今日发帖次数已用尽！")
		return
	}

	// 💡 4. DeepSeek AI 风控大脑介入审核 - 已按要求物理注释并默认放行
	isSafe, riskReason := true, "SAFE"
	if !isSafe {
		// 如果违规，记录违规次数
		newViolation := userStatus.PostViolationCount + 1
		banUpdate := map[string]interface{}{"post_violation_count": newViolation}

		// 累计 3 次直接永久禁言
		if newViolation >= 3 {
			banUpdate["is_post_banned"] = 1
		}
		global.DB.Table("user").Where("id = ?", userID).Updates(banUpdate)

		errMsg := fmt.Sprintf("AI检测到内容违规 (%s) 被拦截！", riskReason)
		if newViolation >= 3 {
			errMsg += "您已累计违规 3 次，发帖功能永久封禁！"
		}
		common.Error(c, errMsg)
		return
	}

	// 5. 安全合规，落库并锁定次数
	post := models.CommunityPost{
		PostType:     "USER",
		AuthorID:     userID,
		AuthorName:   userStatus.Nickname,
		AuthorAvatar: userStatus.Avatar,
		Content:      req.Content,
		ImageURL:     req.ImageUrl,
		TrackID:      req.TrackID,
		Channel:      fmt.Sprintf("%d", req.TrackID), // 自动将帖子路由到对应赛道频道
	}

	if err := global.DB.Create(&post).Error; err != nil {
		common.Error(c, "发布失败，数据库异常")
		return
	}

	global.Redis.Set(ctx, limitKey, 1, 24*time.Hour)
	common.Success(c, "动态发布成功")
}

// ==================== 2. 互动功能 (点赞与评论) ====================

type LikeReq struct {
	PostID int64  `json:"postId"`
	Action string `json:"action"` // "like" 或 "cancel"
}

func LikePost(c *gin.Context) {
	var req LikeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	userID := c.MustGet("userId").(int64)

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if req.Action == "like" {
			like := models.CommunityLike{PostID: req.PostID, UserID: userID}
			if err := tx.Create(&like).Error; err != nil {
				return err // 违反唯一索引说明已经点赞过
			}
			tx.Model(&models.CommunityPost{}).Where("id = ?", req.PostID).UpdateColumn("like_count", gorm.Expr("like_count + ?", 1))
		} else {
			res := tx.Where("post_id = ? AND user_id = ?", req.PostID, userID).Delete(&models.CommunityLike{})
			if res.RowsAffected > 0 {
				tx.Model(&models.CommunityPost{}).Where("id = ?", req.PostID).UpdateColumn("like_count", gorm.Expr("like_count - ?", 1))
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "操作频繁或状态异常")
		return
	}
	common.Success(c, "操作成功")
}

type CommentReq struct {
	PostID  int64  `json:"postId"`
	Content string `json:"content"`
}

func CommentPost(c *gin.Context) {
	var req CommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	if len(req.Content) == 0 || len([]rune(req.Content)) > 20 {
		common.Error(c, "评论必须在1到20字之间")
		return
	}

	userID := c.MustGet("userId").(int64)
	var user models.User
	global.DB.Select("nickname", "avatar").First(&user, userID)

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		comment := models.CommunityComment{
			PostID:     req.PostID,
			UserID:     userID,
			UserName:   user.Nickname,
			UserAvatar: user.Avatar,
			Content:    req.Content,
		}
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		return tx.Model(&models.CommunityPost{}).Where("id = ?", req.PostID).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	})

	if err != nil {
		common.Error(c, "评论失败")
		return
	}
	common.Success(c, "评论成功")
}

// ==================== 3. 全局强通知 ====================

func GetGlobalNotice(c *gin.Context) {
	var notice map[string]interface{}
	err := global.DB.Table("global_notice").Where("is_active = 1").Order("create_time desc").Limit(1).Take(&notice).Error
	if err != nil {
		common.Success(c, nil)
		return
	}
	common.Success(c, notice)
}

// GetOfficialAnnouncements 对应 /api/admin/community/official/list 或 /api/sub-admin/community/official/list (GET)
// 只读拉取官方置顶或普通动态通知，表不存在或空时安全返回空数组
func GetOfficialAnnouncements(c *gin.Context) {
	var results []map[string]interface{}
	// 安全校验：如果 community_post 表中定义了官方类型
	query := global.DB.Table("community_post").
		Where("post_type = ?", "OFFICIAL").
		Order("is_pinned desc, sort_order asc, create_time desc")

	if err := query.Limit(50).Scan(&results).Error; err != nil {
		// 防御性拦截：如果该表不存在或发生其他报错，安全返回空数组
		common.Success(c, []interface{}{})
		return
	}

	common.Success(c, results)
}

// 💡 物理安全自适应类型解析器
func parseToInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case string:
		var res int64
		_, _ = fmt.Sscanf(val, "%d", &res)
		return res
	default:
		return 0
	}
}

func parseToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		var res int
		_, _ = fmt.Sscanf(val, "%d", &res)
		return res
	default:
		return 0
	}
}

// ----------------------------------------------------
// 📢 管理端：社区推流与卡片管理控制器
// ----------------------------------------------------

// PushReq 推流请求载荷
type PushReq struct {
	RecordID     int64  `json:"recordId" binding:"required"`
	Channel      string `json:"channel" binding:"required"`
	AdminComment string `json:"adminComment"`
	AuthorAvatar string `json:"authorAvatar"`
}

// PushCommunityPost 打卡推流上线 (POST)
func PushCommunityPost(c *gin.Context) {
	var req PushReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误")
		return
	}

	// 1. 获取打卡记录
	var record models.CheckinRecord
	if err := global.DB.First(&record, req.RecordID).Error; err != nil {
		common.Error(c, "未找到对应的打卡记录")
		return
	}

	// 2. 获取打卡人姓名及头像
	var user models.User
	if err := global.DB.Select("nickname", "avatar").First(&user, record.UserID).Error; err != nil {
		common.Error(c, "未找到打卡人的学生档案")
		return
	}

	// 3. 开启事务落库
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 更新打卡记录
		if err := tx.Model(&record).Updates(map[string]interface{}{
			"is_pushed":    true,
			"push_channel": req.Channel,
		}).Error; err != nil {
			return err
		}

		// 处理马甲头像
		avatar := user.Avatar
		if req.AuthorAvatar != "" {
			avatar = req.AuthorAvatar
		}

		// 检查 community_post 表是否已有对应记录
		var post models.CommunityPost
		err := tx.Where("ref_record_id = ? AND post_type = 'CHECKIN'", record.ID).First(&post).Error
		if err == nil {
			// 更新
			return tx.Model(&post).Updates(map[string]interface{}{
				"author_name":   user.Nickname,
				"author_avatar": avatar,
				"content":       record.ContentText,
				"image_url":     record.ImageURL,
				"channel":       req.Channel,
				"track_id":      record.TrackID,
				"admin_comment": req.AdminComment,
			}).Error
		} else {
			// 新增
			newPost := models.CommunityPost{
				PostType:     "CHECKIN",
				RefRecordID:  &record.ID,
				AuthorID:     record.UserID,
				AuthorName:   user.Nickname,
				AuthorAvatar: avatar,
				Content:      record.ContentText,
				ImageURL:     record.ImageURL,
				Channel:      req.Channel,
				TrackID:      record.TrackID,
				AdminComment: req.AdminComment,
				IsPinned:     false,
				SortOrder:    0,
			}
			return tx.Create(&newPost).Error
		}
	})

	if err != nil {
		common.Error(c, "推流失败，数据库写入异常")
		return
	}

	common.Success(c, "推流成功")
}

// CancelPushCommunityPost 撤销打卡推流 (POST)
func CancelPushCommunityPost(c *gin.Context) {
	var req struct {
		RecordID int64 `json:"recordId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 更新打卡记录
		if err := tx.Model(&models.CheckinRecord{}).Where("id = ?", req.RecordID).Updates(map[string]interface{}{
			"is_pushed":    false,
			"push_channel": "",
			"is_pinned":    false,
		}).Error; err != nil {
			return err
		}

		// 2. 物理移出社区帖子表
		return tx.Where("ref_record_id = ? AND post_type = 'CHECKIN'", req.RecordID).Delete(&models.CommunityPost{}).Error
	})

	if err != nil {
		common.Error(c, "撤销推流失败，数据库更新异常")
		return
	}

	common.Success(c, "已撤销推流")
}

// ToggleOfficialPostPin 官方/推流帖子置顶切换 (POST)
func ToggleOfficialPostPin(c *gin.Context) {
	id := c.Query("id")
	isPinnedStr := c.Query("isPinned")
	if id == "" || isPinnedStr == "" {
		common.Error(c, "缺少必要参数")
		return
	}

	isPinned := isPinnedStr == "1"

	var post models.CommunityPost
	if err := global.DB.First(&post, id).Error; err != nil {
		common.Error(c, "未找到该动态记录")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 更新帖子置顶
		if err := tx.Model(&post).Update("is_pinned", isPinned).Error; err != nil {
			return err
		}

		// 2. 如果是打卡记录类型，同步更新 checkin_record 的 is_pinned
		if post.PostType == "CHECKIN" && post.RefRecordID != nil {
			return tx.Model(&models.CheckinRecord{}).Where("id = ?", *post.RefRecordID).Update("is_pinned", isPinned).Error
		}
		return nil
	})

	if err != nil {
		common.Error(c, "置顶配置保存失败")
		return
	}

	common.Success(c, "置顶状态切换成功")
}

// PublishOfficialPost 发布官方动态 (POST)
type PublishOfficialReq struct {
	AuthorName   string `json:"authorName" binding:"required"`
	AuthorAvatar string `json:"authorAvatar"`
	Topic        string `json:"topic" binding:"required"`
	Content      string `json:"content" binding:"required"`
	ImageUrl     string `json:"imageUrl"`
	Channel      string `json:"channel" binding:"required"`
	TrackID      *int   `json:"trackId"`
	IsPinned     int    `json:"isPinned"`
}

func PublishOfficialPost(c *gin.Context) {
	var req PublishOfficialReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误")
		return
	}

	trackID := 0
	if req.TrackID != nil {
		trackID = *req.TrackID
	}

	post := models.CommunityPost{
		PostType:     "OFFICIAL",
		AuthorID:     0, // 官方
		AuthorName:   req.AuthorName,
		AuthorAvatar: req.AuthorAvatar,
		Topic:        req.Topic,
		Content:      req.Content,
		ImageURL:     req.ImageUrl,
		Channel:      req.Channel,
		TrackID:      trackID,
		IsPinned:     req.IsPinned == 1,
		SortOrder:    0,
	}

	if err := global.DB.Create(&post).Error; err != nil {
		common.Error(c, "发布官方动态失败")
		return
	}

	common.Success(c, "官方动态发布成功")
}

// DeleteOfficialPost 物理删除官方动态 (DELETE)
func DeleteOfficialPost(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		common.Error(c, "缺失动态主键 ID")
		return
	}

	if err := global.DB.Where("id = ? AND post_type = 'OFFICIAL'", id).Delete(&models.CommunityPost{}).Error; err != nil {
		common.Error(c, "删除动态失败")
		return
	}

	common.Success(c, "动态删除成功")
}

// SortCommunityBoard 大盘卡片批量拖拽重排 (POST)
type BoardSortItem struct {
	ID        interface{} `json:"id"` // 可能是 string 或 number，自适应
	Type      string      `json:"type"`
	SortOrder int         `json:"sortOrder"`
	Channel   string      `json:"channel"`
	TrackID   interface{} `json:"trackId"`
}

func SortCommunityBoard(c *gin.Context) {
	var req []BoardSortItem
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req {
			// 解析 ID
			idVal := parseToInt64(item.ID)

			// 解析 trackId
			trackIDVal := parseToInt(item.TrackID)

			if item.Type == "official" {
				if err := tx.Model(&models.CommunityPost{}).Where("id = ? AND post_type = 'OFFICIAL'", idVal).Updates(map[string]interface{}{
					"sort_order": item.SortOrder,
					"channel":    item.Channel,
					"track_id":   trackIDVal,
				}).Error; err != nil {
					return err
				}
			} else if item.Type == "checkin" {
				if err := tx.Model(&models.CommunityPost{}).Where("ref_record_id = ? AND post_type = 'CHECKIN'", idVal).Updates(map[string]interface{}{
					"sort_order": item.SortOrder,
					"channel":    item.Channel,
					"track_id":   trackIDVal,
				}).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.CheckinRecord{}).Where("id = ?", idVal).Updates(map[string]interface{}{
					"push_channel": item.Channel,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "保存大盘排序失败")
		return
	}

	common.Success(c, "大盘排序成功同步")
}

// UpdateCommunityBoard 编辑大盘卡片参数 (POST)
type BoardEditReq struct {
	ID           interface{} `json:"id"`
	Type         string      `json:"type"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	ImageUrl     string      `json:"imageUrl"`
	AuthorAvatar string      `json:"authorAvatar"`
	IsAnonymous  int         `json:"isAnonymous"`
	IsPinned     int         `json:"isPinned"`
	Channel      string      `json:"channel"`
	TrackID      interface{} `json:"trackId"`
}

func UpdateCommunityBoard(c *gin.Context) {
	var req BoardEditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数格式错误")
		return
	}

	idVal := parseToInt64(req.ID)
	trackIDVal := parseToInt(req.TrackID)

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if req.Type == "official" {
			return tx.Model(&models.CommunityPost{}).Where("id = ? AND post_type = 'OFFICIAL'", idVal).Updates(map[string]interface{}{
				"topic":         req.Title,
				"content":       req.Content,
				"image_url":     req.ImageUrl,
				"author_avatar": req.AuthorAvatar,
				"channel":       req.Channel,
				"track_id":      trackIDVal,
				"is_pinned":     req.IsPinned == 1,
			}).Error
		} else if req.Type == "checkin" {
			// 更新大盘帖子
			if err := tx.Model(&models.CommunityPost{}).Where("ref_record_id = ? AND post_type = 'CHECKIN'", idVal).Updates(map[string]interface{}{
				"topic":         req.Title,
				"content":       req.Content,
				"image_url":     req.ImageUrl,
				"author_avatar": req.AuthorAvatar,
				"is_anonymous":  req.IsAnonymous == 1,
				"is_pinned":     req.IsPinned == 1,
				"channel":       req.Channel,
				"track_id":      trackIDVal,
			}).Error; err != nil {
				return err
			}
			// 同步更新打卡记录
			return tx.Model(&models.CheckinRecord{}).Where("id = ?", idVal).Updates(map[string]interface{}{
				"is_pushed":    true,
				"push_channel": req.Channel,
				"is_pinned":    req.IsPinned == 1,
			}).Error
		}
		return nil
	})

	if err != nil {
		common.Error(c, "更新卡片参数失败")
		return
	}

	common.Success(c, "推流卡片编辑保存成功")
}

// RemoveCommunityBoard 移出大盘卡片 (POST)
type RemoveBoardReq struct {
	ID   interface{} `json:"id"`
	Type string      `json:"type"`
}

func RemoveCommunityBoard(c *gin.Context) {
	var req RemoveBoardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误")
		return
	}

	idVal := parseToInt64(req.ID)

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if req.Type == "official" {
			return tx.Where("id = ? AND post_type = 'OFFICIAL'", idVal).Delete(&models.CommunityPost{}).Error
		} else if req.Type == "checkin" {
			if err := tx.Where("ref_record_id = ? AND post_type = 'CHECKIN'", idVal).Delete(&models.CommunityPost{}).Error; err != nil {
				return err
			}
			return tx.Model(&models.CheckinRecord{}).Where("id = ?", idVal).Updates(map[string]interface{}{
				"is_pushed":    false,
				"push_channel": "",
				"is_pinned":    false,
			}).Error
		}
		return nil
	})

	if err != nil {
		common.Error(c, "移出大盘失败")
		return
	}

	common.Success(c, "卡片已成功从社区大盘下线移出")
}

// ----------------------------------------------------
// 💰 管理端：悬赏任务榜控制器
// ----------------------------------------------------

// GetBountyTaskList 获取悬赏列表 (GET)
func GetBountyTaskList(c *gin.Context) {
	var list []models.BountyTask
	if err := global.DB.Where("status = 1").Order("is_pinned desc, sort_order asc, create_time desc").Find(&list).Error; err != nil {
		common.Success(c, []interface{}{})
		return
	}
	common.Success(c, list)
}

// SaveBountyTask 保存/更新悬赏 (POST)
type BountyTaskReq struct {
	ID         int    `json:"id"`
	Title      string `json:"title" binding:"required"`
	RewardText string `json:"rewardText" binding:"required"`
	EndTime    string `json:"endTime"`
	SortOrder  int    `json:"sortOrder"`
}

func SaveBountyTask(c *gin.Context) {
	var req BountyTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数填写不完整")
		return
	}

	var endTime *time.Time
	if req.EndTime != "" {
		// 支持标准 YYYY-MM-DD HH:mm:ss 格式转换
		t, err := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local)
		if err == nil {
			endTime = &t
		}
	}

	if req.ID > 0 {
		var bounty models.BountyTask
		if err := global.DB.First(&bounty, req.ID).Error; err == nil {
			bounty.Title = req.Title
			bounty.RewardText = req.RewardText
			bounty.EndTime = endTime
			bounty.SortOrder = req.SortOrder
			global.DB.Save(&bounty)
			common.Success(c, "悬赏保存成功")
			return
		}
	}

	newBounty := models.BountyTask{
		Title:      req.Title,
		RewardText: req.RewardText,
		EndTime:    endTime,
		SortOrder:  req.SortOrder,
		Status:     1,
		IsPinned:   false,
	}

	if err := global.DB.Create(&newBounty).Error; err != nil {
		common.Error(c, "悬赏添加失败")
		return
	}

	common.Success(c, "新悬赏发布成功")
}

// DeleteBountyTask 物理删除悬赏 (DELETE)
func DeleteBountyTask(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		common.Error(c, "缺少必要参数 ID")
		return
	}

	if err := global.DB.Where("id = ?", id).Delete(&models.BountyTask{}).Error; err != nil {
		common.Error(c, "删除悬赏失败")
		return
	}

	common.Success(c, "悬赏已成功下架删除")
}

// ToggleBountyTaskPin 悬赏置顶切换 (POST)
func ToggleBountyTaskPin(c *gin.Context) {
	id := c.Query("id")
	isPinnedStr := c.Query("isPinned")
	if id == "" || isPinnedStr == "" {
		common.Error(c, "缺少必要参数")
		return
	}

	isPinned := isPinnedStr == "1"

	if err := global.DB.Model(&models.BountyTask{}).Where("id = ?", id).Update("is_pinned", isPinned).Error; err != nil {
		common.Error(c, "置顶切换失败")
		return
	}

	common.Success(c, "置顶设置成功")
}

// UpdateBountyTaskSort 悬赏卡片批量重排排序 (POST)
type BountySortItem struct {
	ID        int `json:"id"`
	SortOrder int `json:"sortOrder"`
}

func UpdateBountyTaskSort(c *gin.Context) {
	var req []BountySortItem
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败")
		return
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req {
			if err := tx.Model(&models.BountyTask{}).Where("id = ?", item.ID).Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "悬赏排序保存失败")
		return
	}

	common.Success(c, "排序成功保存并同步")
}

// ----------------------------------------------------
// 📢 管理端：全局登录强弹窗通知控制器
// ----------------------------------------------------

// GetGlobalNoticeAdmin 获取全局强通知 (GET)
func GetGlobalNoticeAdmin(c *gin.Context) {
	var notice models.GlobalNotice
	// 获取最新的一条
	if err := global.DB.Order("create_time desc").Limit(1).Take(&notice).Error; err != nil {
		common.Success(c, nil)
		return
	}
	common.Success(c, notice)
}

// SaveGlobalNotice 保存全服强通知 (POST)
type GlobalNoticeReq struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	IsActive int    `json:"isActive"`
}

func SaveGlobalNotice(c *gin.Context) {
	var req GlobalNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "请确保标题与内容输入完整")
		return
	}

	// 将其他激活的通知全部置为非激活，维持当前的一条最新激活策略
	if req.IsActive == 1 {
		global.DB.Model(&models.GlobalNotice{}).Where("is_active = ?", true).Update("is_active", false)
	}

	notice := models.GlobalNotice{
		Title:    req.Title,
		Content:  req.Content,
		IsActive: req.IsActive == 1,
	}

	if err := global.DB.Create(&notice).Error; err != nil {
		common.Error(c, "保存弹窗广播失败")
		return
	}

	common.Success(c, notice)
}

// ToggleGlobalNotice 开启/关闭全局弹窗广播 (POST)
func ToggleGlobalNotice(c *gin.Context) {
	isActiveStr := c.Query("isActive")
	if isActiveStr == "" {
		common.Error(c, "缺少必要状态参数")
		return
	}

	isActive := isActiveStr == "true" || isActiveStr == "1"

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 将所有记录设为 false
		if err := tx.Model(&models.GlobalNotice{}).Where("1=1").Update("is_active", false).Error; err != nil {
			return err
		}

		// 2. 如果开启，将最近一条设为 true
		if isActive {
			var notice models.GlobalNotice
			if err := tx.Order("create_time desc").First(&notice).Error; err == nil {
				return tx.Model(&notice).Update("is_active", true).Error
			}
		}
		return nil
	})

	if err != nil {
		common.Error(c, "激活状态切换失败")
		return
	}

	common.Success(c, "广播开关更新完成")
}
