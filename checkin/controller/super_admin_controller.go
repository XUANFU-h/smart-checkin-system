package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────────────────────
// SuperAdmin 控制中枢：名册联查、权限覆写、封禁、强踢
// 彻底消灭 N+1 查询，单 JOIN SQL 一次性返回完整权限快照
// ─────────────────────────────────────────────────────────────────────────────

// SuperGetSubAdminList 零延迟子管理员名册：LEFT JOIN 聚合一次通车
func SuperGetSubAdminList(c *gin.Context) {
	sql := `
		SELECT
			a.id,
			a.username        AS username,
			a.student_id      AS studentId,
			a.binding_secret  AS bindingSecret,
			a.is_bound        AS isBound,
			a.status          AS accountStatus,
			COALESCE(GROUP_CONCAT(DISTINCT mp.menu_key  ORDER BY mp.menu_key  SEPARATOR ','), '') AS menuKeys,
			COALESCE(GROUP_CONCAT(DISTINCT tr.track_id  ORDER BY tr.track_id  SEPARATOR ','), '') AS trackIds
		FROM admin_user a
		LEFT JOIN sys_admin_menu_perm      mp ON mp.admin_id = a.id
		LEFT JOIN sys_admin_track_relation tr ON tr.admin_id = a.id
		WHERE a.role_level IN (1, 2)
		GROUP BY a.id, a.username, a.student_id, a.binding_secret, a.is_bound, a.status
		ORDER BY a.id ASC
	`
	var list []map[string]interface{}
	global.DB.Raw(sql).Scan(&list)
	common.Success(c, list)
}

// GenerateRandomSecret 随机生成 8 位排除易混淆字符的微信绑定密钥
func GenerateRandomSecret(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除 0, O, 1, I 等易错字
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// SuperSaveMenuPerms 大事务全量覆写或保存/修改子管理员账号大网关
func SuperSaveMenuPerms(c *gin.Context) {
	type Req struct {
		Action    string   `json:"action"` // "save" 或 "sync_matrix" (默认为 "sync_matrix" 或 空)
		AdminID   int64    `json:"adminId"`
		MenuKeys  []string `json:"menuKeys"`
		StudentID string   `json:"student_id"`
		Username  string   `json:"username"`
		Password  string   `json:"password"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误")
		return
	}

	// 📡 分流一：物理覆盖写入 (添加与修改子管理员账号)
	if req.Action == "save" {
		username := req.Username
		studentID := req.StudentID

		if username == "" {
			common.Error(c, "用户名（真实中文姓名）不能为空")
			return
		}
		if len(studentID) != 10 {
			common.Error(c, "学号格式非法：学号必须强制为 10 位数！")
			return
		}

		var admin models.AdminUser
		err := global.DB.Table("admin_user").Where("username = ?", username).First(&admin).Error

		if err != nil {
			// 账户不存在 ➔ 新增管理员账号 (默认定级为 2=SUB二级干事, 默认密码 123456, 随机特征密钥)
			pass := req.Password
			if pass == "" {
				pass = "123456"
			}
			bytes, errHash := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			if errHash != nil {
				common.Error(c, "底层加密哈希失败")
				return
			}
			hashedPassword := string(bytes)
			bindingSecret := GenerateRandomSecret(8)

			errInsert := global.DB.Exec(`
				INSERT INTO admin_user (username, password, real_name, role_level, status, student_id, binding_secret, is_bound, bound_user_id, create_time) 
				VALUES (?, ?, ?, 2, 0, ?, ?, 0, 0, NOW())
			`, username, hashedPassword, username, studentID, bindingSecret).Error

			if errInsert != nil {
				common.Error(c, "空降新管理员账号写入失败: "+errInsert.Error())
				return
			}
			common.Success(c, map[string]interface{}{
				"message":       "新子管理员账号已成功空降 `admin_user` 表！",
				"bindingSecret": bindingSecret,
			})
			return
		} else {
			// 账户已存在 ➔ 更新基本属性，密码如果非空则覆写
			updates := map[string]interface{}{
				"role_level": 2,
				"status":     0,
				"student_id": studentID,
			}
			if req.Password != "" {
				bytes, errHash := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
				if errHash == nil {
					updates["password"] = string(bytes)
				}
			}
			errUpdate := global.DB.Table("admin_user").Where("username = ?", username).Updates(updates).Error
			if errUpdate != nil {
				common.Error(c, "更正子管理员账号失败: "+errUpdate.Error())
				return
			}
			common.Success(c, map[string]interface{}{
				"message":       "子管理员资料与凭证已成功更正对齐！",
				"bindingSecret": admin.BindingSecret,
			})
			return
		}
	}

	// 📡 分流二：同步复选框矩阵流程 (原有大事务授权逻辑)
	if req.AdminID == 0 {
		common.Error(c, "参数错误：需要 adminId")
		return
	}

	// 确认目标是子管理员
	var count int64
	global.DB.Table("admin_user").Where("id = ? AND role_level IN (1, 2)", req.AdminID).Count(&count)
	if count == 0 {
		common.Error(c, "目标账号不存在或不是子管理员")
		return
	}

	db := global.DB.Begin()
	if db.Error != nil {
		common.Error(c, "事务开启失败")
		return
	}

	// Step 1: 全量清除旧菜单与操作权限
	if err := db.Exec("DELETE FROM sys_admin_menu_perm WHERE admin_id = ?", req.AdminID).Error; err != nil {
		db.Rollback()
		common.Error(c, "清除旧菜单权限失败: "+err.Error())
		return
	}
	if err := db.Exec("DELETE FROM sys_admin_action_perm WHERE admin_id = ?", req.AdminID).Error; err != nil {
		db.Rollback()
		common.Error(c, "清除旧操作权限失败: "+err.Error())
		return
	}

	// Step 2: 批量插入新菜单权限
	if len(req.MenuKeys) > 0 {
		placeholders := make([]string, len(req.MenuKeys))
		args := make([]interface{}, 0, len(req.MenuKeys)*2)
		for i, key := range req.MenuKeys {
			placeholders[i] = "(?, ?)"
			args = append(args, req.AdminID, key)
		}
		insertSQL := "INSERT IGNORE INTO sys_admin_menu_perm (admin_id, menu_key) VALUES " + strings.Join(placeholders, ",")
		if err := db.Exec(insertSQL, args...).Error; err != nil {
			db.Rollback()
			common.Error(c, "写入菜单权限失败: "+err.Error())
			return
		}

		// Step 3: 根据勾选的菜单自动推导并批量插入对应的细粒度操作功能权限
		var actionKeys []string
		for _, key := range req.MenuKeys {
			if key == "students" {
				actionKeys = append(actionKeys, "user_rename", "user_ban", "user_kick", "user_delete", "score_modify")
			} else if key == "audit" {
				actionKeys = append(actionKeys, "record_delete")
			}
		}

		if len(actionKeys) > 0 {
			actPlaceholders := make([]string, len(actionKeys))
			actArgs := make([]interface{}, 0, len(actionKeys)*2)
			for i, key := range actionKeys {
				actPlaceholders[i] = "(?, ?)"
				actArgs = append(actArgs, req.AdminID, key)
			}
			actInsertSQL := "INSERT IGNORE INTO sys_admin_action_perm (admin_id, action_key) VALUES " + strings.Join(actPlaceholders, ",")
			if err := db.Exec(actInsertSQL, actArgs...).Error; err != nil {
				db.Rollback()
				common.Error(c, "写入功能特权失败: "+actInnerErr(err))
				return
			}
		}
	}

	if err := db.Commit().Error; err != nil {
		common.Error(c, "事务提交失败: "+err.Error())
		return
	}

	common.Success(c, "子管理员菜单及操作权限已全量覆写落库")
}

// 辅助错误解析
func actInnerErr(err error) string {
	return err.Error()
}

// SuperKickSubAdmin 物理强踢：DEL Redis 中所有该干事设备 Token，1 秒内强制 401 退出
func SuperKickSubAdmin(c *gin.Context) {
	type Req struct {
		AdminID int64 `json:"adminId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	ctx := context.Background()
	var allKeys []string

	// 扫描 user_device:{id}:* 格式的键
	for _, pattern := range []string{
		fmt.Sprintf("user_device:%d:*", req.AdminID),
		fmt.Sprintf("device_token:%d:*", req.AdminID),
	} {
		var cursor uint64
		for {
			batch, nextCursor, err := global.Redis.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			allKeys = append(allKeys, batch...)
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}

	if len(allKeys) > 0 {
		global.Redis.Del(ctx, allKeys...)
	}

	// 写入踢人标记，JWT 中间件可即时感知
	kickKey := fmt.Sprintf("kick_user:%d", req.AdminID)
	global.Redis.Set(ctx, kickKey, "1", 0)

	common.Success(c, fmt.Sprintf("干事 %d 已被物理强踢下线，共清除 %d 个 Token", req.AdminID, len(allKeys)))
}

// SuperToggleSubAdminStatus 封禁/解封子管理员账号（即时联动 Redis）
func SuperToggleSubAdminStatus(c *gin.Context) {
	type Req struct {
		AdminID int64 `json:"adminId" binding:"required"`
		Status  int8  `json:"status"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数错误")
		return
	}

	if err := global.DB.Table("admin_user").Where("id = ? AND role_level IN (1, 2)", req.AdminID).
		Update("status", req.Status).Error; err != nil {
		common.Error(c, "状态更新失败: "+err.Error())
		return
	}

	ctx := context.Background()
	banKey := fmt.Sprintf("ban_user:%d", req.AdminID)

	if req.Status == 1 {
		global.Redis.Set(ctx, banKey, "账号已被大管理员冻结", 0)
		common.Success(c, "干事账号已封禁，即时生效")
	} else {
		global.Redis.Del(ctx, banKey)
		common.Success(c, "干事账号已恢复，即时解封")
	}
}

// GetMyPermissions 子管理员拉取自己的权限字典（单 JOIN，零 N+1）
func GetMyPermissions(c *gin.Context) {
	userID := c.MustGet("userId").(int64)
	userRole, _ := c.Get("userRole")

	// 大管理员拥有全部权限，直接返回全集，免查数据库
	if userRole == "SUPERADMIN" {
		common.Success(c, gin.H{
			"menus":        []string{"dashboard", "students", "audit", "appeal", "community", "lotto", "shop", "settings", "users"},
			"actions":      []string{"user_delete", "user_ban", "user_kick", "user_rename", "record_delete", "score_modify"},
			"trackIds":     []int{},
			"isSuperAdmin": true,
		})
		return
	}

	// 子管理员：单 SQL 三表 LEFT JOIN 聚合返回权限快照
	type PermSnap struct {
		MenuKeys   string `gorm:"column:menuKeys"`
		ActionKeys string `gorm:"column:actionKeys"`
		TrackIds   string `gorm:"column:trackIds"`
	}
	var snap PermSnap

	sql := `
		SELECT
			COALESCE(GROUP_CONCAT(DISTINCT mp.menu_key   ORDER BY mp.menu_key   SEPARATOR ','), '') AS menuKeys,
			COALESCE(GROUP_CONCAT(DISTINCT ap.action_key ORDER BY ap.action_key SEPARATOR ','), '') AS actionKeys,
			COALESCE(GROUP_CONCAT(DISTINCT tr.track_id   ORDER BY tr.track_id   SEPARATOR ','), '') AS trackIds
		FROM (SELECT ? AS admin_id) base
		LEFT JOIN sys_admin_menu_perm      mp ON mp.admin_id = base.admin_id
		LEFT JOIN sys_admin_action_perm    ap ON ap.admin_id = base.admin_id
		LEFT JOIN sys_admin_track_relation tr ON tr.admin_id = base.admin_id
	`
	global.DB.Raw(sql, userID).Scan(&snap)

	parseCSV := func(s string) []string {
		if s == "" {
			return []string{}
		}
		return strings.Split(s, ",")
	}

	parseIntCSV := func(s string) []int {
		if s == "" {
			return []int{}
		}
		parts := strings.Split(s, ",")
		result := make([]int, 0, len(parts))
		for _, p := range parts {
			var n int
			fmt.Sscanf(strings.TrimSpace(p), "%d", &n)
			result = append(result, n)
		}
		return result
	}

	common.Success(c, gin.H{
		"menus":        parseCSV(snap.MenuKeys),
		"actions":      parseCSV(snap.ActionKeys),
		"trackIds":     parseIntCSV(snap.TrackIds),
		"isSuperAdmin": false,
	})
}

// AdminResetSelfPassword 子管理员/大管理员修改自身密码
func AdminResetSelfPassword(c *gin.Context) {
	type Req struct {
		NewPassword string `json:"newPassword" binding:"required"`
		AdminID     int64  `json:"adminId"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析异常: "+err.Error())
		return
	}

	// 优先以 token 中的实际 logged-in userID 为准，横向越权彻底杜绝
	userID := c.MustGet("userId").(int64)
	if req.AdminID == 0 {
		req.AdminID = userID
	}

	// 仅允许大管理员修改他人密码，子管理员只允许修改自身密码
	userRole, _ := c.Get("userRole")
	if req.AdminID != userID && userRole != "SUPERADMIN" {
		common.Error(c, "安全阻断：子管理员无权越权更改他人账密！")
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		common.Error(c, "安全密码哈希散列失败")
		return
	}
	hashedPassword := string(bytes)

	if err := global.DB.Table("admin_user").Where("id = ?", req.AdminID).Update("password", hashedPassword).Error; err != nil {
		common.Error(c, "安全密码落库失败: "+err.Error())
		return
	}

	common.Success(c, "安全密码已成功更正，下次登录即生效")
}

// SuperRefreshSecret 大管理员强制或者定时给子管理员生成并刷新 10分钟时效的动态绑定密钥
func SuperRefreshSecret(c *gin.Context) {
	type Req struct {
		AdminID int64 `json:"adminId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "解析参数失败")
		return
	}

	var admin models.AdminUser
	if err := global.DB.First(&admin, req.AdminID).Error; err != nil {
		common.Error(c, "未找到该管理员账号记录")
		return
	}

	if admin.IsBound {
		common.Error(c, "该管理员账号已成功绑定微信物理安全盾，无法再刷新特征密钥")
		return
	}

	// 生成 10分钟高时效动态绑定密钥
	newSecret := GenerateRandomSecret(8)
	expireTime := time.Now().Add(10 * time.Minute)

	if err := global.DB.Model(&admin).Updates(map[string]interface{}{
		"binding_secret":     newSecret,
		"secret_expire_time": &expireTime,
	}).Error; err != nil {
		common.Error(c, "密钥更新落库失败: "+err.Error())
		return
	}

	common.Success(c, map[string]interface{}{
		"bindingSecret":    newSecret,
		"secretExpireTime": expireTime,
	})
}

// SuperDeleteSubAdmin 彻底物理抹除注销子管理员，并自动解绑和还原对应的学生身份为普通学生
func SuperDeleteSubAdmin(c *gin.Context) {
	type Req struct {
		AdminID int64 `json:"adminId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "解析参数失败")
		return
	}

	var admin models.AdminUser
	if err := global.DB.First(&admin, req.AdminID).Error; err != nil {
		common.Error(c, "未找到该管理员账号")
		return
	}

	if admin.RoleLevel == 0 {
		common.Error(c, "禁止操作：超级管理员根系统账号受核心安全盾保护，无法物理注销！")
		return
	}

	// 开启大事务进行物理退权及级联删除
	errTx := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 如果已绑定了微信，还原对应学生用户表角色为普通用户 (user_type = 0)
		if admin.IsBound && admin.BoundUserID > 0 {
			if err := tx.Table("user").Where("id = ?", admin.BoundUserID).Update("user_type", 0).Error; err != nil {
				return err
			}
		}

		// 2. 清除该干事的全部菜单、操作、赛道绑定特权
		if err := tx.Exec("DELETE FROM sys_admin_menu_perm WHERE admin_id = ?", admin.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM sys_admin_action_perm WHERE admin_id = ?", admin.ID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM sys_admin_track_relation WHERE admin_id = ?", admin.ID).Error; err != nil {
			return err
		}

		// 3. 物理注销删除 admin_user 账号
		if err := tx.Delete(&admin).Error; err != nil {
			return err
		}

		return nil
	})

	if errTx != nil {
		common.Error(c, "子管理员物理注销失败: "+errTx.Error())
		return
	}

	// 4. 清理 Redis 活跃设备与强退会话
	ctx := context.Background()
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", admin.ID))
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", admin.ID))
	keys, _ := global.Redis.Keys(ctx, fmt.Sprintf("user_device:%d:*", admin.ID)).Result()
	if len(keys) > 0 {
		global.Redis.Del(ctx, keys...)
	}
	global.Redis.Set(ctx, fmt.Sprintf("kick_user:%d", admin.ID), "FORCE_LOGOUT_BY_SUPER_DELETE", 24*time.Hour)

	common.Success(c, "子管理员账户及绑定微信物理安全契约已彻底清空注销，其学生权限已还原！")
}

// SuperUnbindWechat 强制解除子管理员微信双因子物理绑定，使其下次登录必须重新生成密钥并绑定
func SuperUnbindWechat(c *gin.Context) {
	type Req struct {
		AdminID int64 `json:"adminId" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "解析参数失败")
		return
	}

	var admin models.AdminUser
	if err := global.DB.First(&admin, req.AdminID).Error; err != nil {
		common.Error(c, "未找到该管理员账号")
		return
	}

	if !admin.IsBound {
		common.Error(c, "该管理员账号当前未绑定任何微信安全物理盾，无需解除绑定")
		return
	}

	// 开启大事务重置绑定状态，并物理退权学生账户为 0
	errTx := global.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 将学生用户表的 user_type 还原为普通用户 0
		if admin.BoundUserID > 0 {
			if err := tx.Table("user").Where("id = ?", admin.BoundUserID).Update("user_type", 0).Error; err != nil {
				return err
			}
		}

		// 2. 清空 admin_user 的绑定列和动态密钥
		if err := tx.Model(&admin).Updates(map[string]interface{}{
			"is_bound":           false,
			"bound_user_id":      0,
			"binding_secret":     "",
			"secret_expire_time": nil,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if errTx != nil {
		common.Error(c, "强制解除微信物理安全绑定失败: "+errTx.Error())
		return
	}

	// 3. 强退该干事在 Web 端管理后台的所有在线 Session 与会话
	ctx := context.Background()
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_token:%d", admin.ID))
	global.Redis.Del(ctx, fmt.Sprintf("user_devices_sort:%d", admin.ID))
	keys, _ := global.Redis.Keys(ctx, fmt.Sprintf("user_device:%d:*", admin.ID)).Result()
	if len(keys) > 0 {
		global.Redis.Del(ctx, keys...)
	}
	global.Redis.Set(ctx, fmt.Sprintf("kick_user:%d", admin.ID), "FORCE_LOGOUT_BY_UNBIND_WECHAT", 24*time.Hour)

	common.Success(c, "该管理员微信物理绑定契约已成功解除！其 Web 端所有在线终端已被物理强退下机，下次登录必须重新对齐密钥绑定！")
}

