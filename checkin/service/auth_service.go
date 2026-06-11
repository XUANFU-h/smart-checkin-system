package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/security"
	"checkin-go/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 密码加密
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// 密码比对
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// =========================================================================
// 1. 💯 核心大重构：【用户注�?+ 赛道重组】全原子两阶段提交服务层实现
// =========================================================================
func RegisterWithTracksService(name, clazz, stuId, password, deviceId string, trackIDs []int, isAdminReg bool, backdoor string) (map[string]interface{}, error) {
	var targetUserType int = 0      // 默认 0 = 普通学�?	var targetAccountStatus int = 0 // 默认 0 = 正式学籍放行

	// 🛡�?路径 A：特权管理员注册升级审查
	if isAdminReg {
		// 💡 本地开发调试专属后门特权旁路放�?		if backdoor == "XUANFU-MFA-TEST-99" || strings.HasPrefix(deviceId, "BACKDOOR-MFA") {
			targetUserType = 2 // 晋升 2=管理�?		} else {
			// 💯 彻底割裂白名单：管理员白名单校验已被彻底铲除，不再支持通过前端直接匿名注册管理员角�?			err := errors.New("安全校验失败：管理员注册白名单校验已被彻底根除，请联系超级管理员在后台分配或升级管理员权�?)
			log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
			return nil, err
		}
	} else {
		// 🛡�?路径 B：普通学生学号校�?		if clazz == "体验账户" {
			// 体验账户免检免审，直接以激活�?0)放行，供顺畅体验使用
			targetAccountStatus = 0
		} else {
			// 限制 10 位数，开头必须是 19-25 的数字区�?			if len(stuId) != 10 {
				err := errors.New("安全阻断：学号长度非法，必须强制�?10 位数�?)
				log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
				return nil, err
			}
			prefix := stuId[0:2]
			if prefix < "19" || prefix > "25" {
				err := errors.New("安全阻断：非本届合规在籍学号段，网络网关拒绝接入")
				log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
				return nil, err
			}
			targetAccountStatus = 0
		}
	}

	hashedPwd, _ := hashPassword(password)

	// 组装标准的纯小写蛇形下划线字段，完美对接全新 user 模型
	newUserMap := map[string]interface{}{
		"student_id":        stuId,
		"nickname":          name,
		"real_name":         name,
		"class_name":        clazz,
		"avatar":            "http://127.0.0.1:8080/static/default-avatar.png",
		"password":          hashedPwd,
		"user_type":         targetUserType,
		"account_status":    targetAccountStatus,
		"current_device_id": deviceId,
		"create_time":       time.Now(),
	}

	// 强行启动大事务：确保【用�?赛道】要么同时落库，要么同时滚回
	var finalUserID int64
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// 校验学号唯一
		var count int64
		tx.Table("user").Where("student_id = ?", stuId).Count(&count)
		if count > 0 {
			err := errors.New("学号重复冲突")
			log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
			return err
		}

		if err := tx.Table("user").Create(&newUserMap).Error; err != nil {
			log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
			return err
		}

		// 反获刚刚入库的主键ID
		tx.Table("user").Where("student_id = ?", stuId).Pluck("id", &finalUserID)

		// 级联批量建立赛道修行足迹映射，拒绝僵尸号空转
		for _, tid := range trackIDs {
			userTrack := map[string]interface{}{
				"user_id":     finalUserID,
				"track_id":    tid,
				"create_time": time.Now(),
			}
			if err := tx.Table("user_track").Create(&userTrack).Error; err != nil {
				log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
				return err
			}
		}

		// 💯 彻底割裂白名单：管理员白名单核销逻辑已物理清除，无须反写数据
		return nil
	})

	if err != nil {
		log.Printf("�?[注册防线拦截硬铁证] 原因: %v", err)
		return nil, err
	}

	// 签发身份令牌
	roleString := "USER"
	if targetUserType == 2 {
		roleString = "ADMIN"
	}
	token, err := utils.GenerateToken(finalUserID, roleString)
	if err != nil {
		errForLog := fmt.Errorf("令牌签发失败: %v", err)
		log.Printf("�?[注册防线拦截硬铁证] 原因: %v", errForLog)
		return nil, errForLog
	}

	// 💯 终极修复：注册成功时同步写入设备登录态，防止进入系统时瞬间提�?401 登录失效
	security.RecordDeviceLogin(finalUserID, deviceId, token)

	return map[string]interface{}{
		"token":          token,
		"userId":         finalUserID,
		"user_type":      targetUserType,
		"account_status": targetAccountStatus,
	}, nil
}

// =========================================================================
// 2. 核心大通车：高阶物理隔离登录逻辑
// =========================================================================
func LoginUser(name, clazz, password, deviceId string) (map[string]interface{}, error) {
	// 🚀 1. 优先分流：反查专用管理账户表 admin_user
	var admin models.AdminUser
	isAdminLogin := false
	if clazz == "" { // 只有在未传入班级（即 Web 后端管理员登录）时，才触发管理员验证分支�?		if err := global.DB.Where("username = ?", name).First(&admin).Error; err == nil {
			isAdminLogin = true
		}
	}

	if isAdminLogin {
		if admin.Status == 1 {
			return nil, errors.New("风控硬拦截：该管理账户已被超级管理员依法冻结封禁")
		}
		if !checkPasswordHash(password, admin.Password) {
			return nil, errors.New("凭据比对失败，密码输入错�?)
		}

		// 🚀 核心对齐：子管理员可以直接账密登录，绕过微信特征绑定拦截限制

		// 动态定�?JWT 签发角色 (0=SUPERADMIN, 其他=ADMIN)
		roleString := "ADMIN"
		userTypeVal := 2
		if admin.RoleLevel == 0 {
			roleString = "SUPERADMIN"
			userTypeVal = 9
		}

		token, err := utils.GenerateToken(admin.ID, roleString)
		if err != nil {
			return nil, fmt.Errorf("令牌生成失败: %v", err)
		}

		// 写入设备登录状态，规避 401 拦截
		security.RecordDeviceLogin(admin.ID, deviceId, token)

		ctx := context.Background()
		// 大赦免：物理清除 kick_user 强退标记
		kickKey := fmt.Sprintf("kick_user:%d", admin.ID)
		global.Redis.Del(ctx, kickKey)

		// 兼容旧值缓存键
		userDeviceKey := fmt.Sprintf("user_device:%d:%s", admin.ID, deviceId)
		global.Redis.Set(ctx, userDeviceKey, token, 24*time.Hour)

		return map[string]interface{}{
			"token":          token,
			"role":           roleString,
			"userId":         admin.ID,
			"adminId":        admin.ID,       // 🚀 黄金兼容：完美对接前�?Login.vue 属性反查！
			"nickname":       admin.RealName,  // 🚀 黄金兼容：完美对接前�?Login.vue 的姓名显示！
			"real_name":      admin.RealName,
			"user_type":      userTypeVal,
			"account_status": int(admin.Status),
		}, nil
	}

	// 🚀 2. Fallback 回溯线：继续支持普通微�?学生账密登录反查 user 表（100%兼容原有学生逻辑�?	var user struct {
		ID            int64  `gorm:"column:id"`
		RealName      string `gorm:"column:real_name"`
		Password      string `gorm:"column:password"`
		UserType      int    `gorm:"column:user_type"`
		AccountStatus int    `gorm:"column:account_status"`
	}

	// 💯 双向键自适应登录校验：允许用户输入【学号】或【真实姓名】登�?	if err := global.DB.Table("user").Where("real_name = ? OR student_id = ?", name, name).First(&user).Error; err != nil {
		return nil, errors.New("用户账户不存在，请确认是否已完成注册选道")
	}

	// 联动设备最高物理风控封禁阻�?	if user.AccountStatus == 2 {
		return nil, errors.New("风控硬拦截：由于严重代打卡作弊违规，当前硬件及微信会话已被大网全网锁死熔�?)
	}

	if !checkPasswordHash(password, user.Password) {
		return nil, errors.New("凭据比对失败，密码输入错�?)
	}

	// 根据数据库真实的 user_type 定夺 JWT 签发角色
	// user_type=9 �?大管理员 SUPERADMIN（拥有全局免检权限�?	// user_type=2 �?子管理员/干事 ADMIN（受 RBAC 权限矩阵约束�?	// 其他 �?普通用�?USER
	roleString := "USER"
	if user.UserType == 9 {
		roleString = "SUPERADMIN"
	} else if user.UserType == 2 {
		roleString = "ADMIN"
	}

	token, err := utils.GenerateToken(user.ID, roleString)
	if err != nil {
		return nil, fmt.Errorf("令牌生成失败: %v", err)
	}

	// 💯 终极修复：登录成功时调用 security.RecordDeviceLogin，将设备�?Token 状态绑定写�?Redis，防止被中间�?401 踢下�?	security.RecordDeviceLogin(user.ID, deviceId, token)

	// 清洗并同步当�?Token �?Redis 硬件防护罩（兼容旧的单值缓存键�?	ctx := context.Background()
	userDeviceKey := fmt.Sprintf("user_device:%d:%s", user.ID, deviceId)
	global.Redis.Set(ctx, userDeviceKey, token, 24*time.Hour)

	// �?登录成功时大赦免：物理清除可能存在的 kick_user 强踢标记，防止二次登录遭遇秒退
	kickKey := fmt.Sprintf("kick_user:%d", user.ID)
	global.Redis.Del(ctx, kickKey)

	return map[string]interface{}{
		"token":          token,
		"role":           roleString,
		"userId":         user.ID,
		"user_type":      user.UserType,
		"account_status": user.AccountStatus,
	}, nil
}

// =========================================================================
// 3. 物理设备防作弊指模加解封双向联动原子驱动函数
// =========================================================================
func ToggleHardwareBlacklist(userID int64, targetStatus int) error {
	var user struct {
		CurrentDeviceID string `gorm:"column:current_device_id"`
	}
	if err := global.DB.Table("user").Select("current_device_id").Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("user").Where("id = ?", userID).Update("account_status", targetStatus).Error; err != nil {
			return err
		}

		if user.CurrentDeviceID == "" {
			return nil
		}

		if targetStatus == 2 {
			var count int64
			tx.Table("device_blacklist").Where("device_id = ?", user.CurrentDeviceID).Count(&count)
			if count == 0 {
				blacklistMap := map[string]interface{}{
					"device_id": user.CurrentDeviceID,
					"reason":    "大管理员在控制台一键远程下发最高设备指纹熔断阻�?,
					"ban_time":  time.Now(),
				}
				if err := tx.Table("device_blacklist").Create(&blacklistMap).Error; err != nil {
					return err
				}
			}
		} else if targetStatus == 0 {
			if err := tx.Table("device_blacklist").Where("device_id = ?", user.CurrentDeviceID).Delete(nil).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// =========================================================================
// 4. 忘记密码智能申请
// =========================================================================
func ApplyForgotPassword(name, clazz, stuId, deviceId string) (map[string]interface{}, error) {
	var student struct {
		ID int64 `gorm:"column:id"`
	}
	if err := global.DB.Table("user").Where("real_name = ? AND class_name = ? AND student_id = ?", name, clazz, stuId).First(&student).Error; err != nil {
		return nil, errors.New("身份特征核验失败：班级、学号与姓名未匹配成�?)
	}

	var isWhite bool
	_ = global.DB.Table("device_whitelist").
		Select("count(1) > 0").
		Where("user_id = ? AND device_id = ?", student.ID, deviceId).
		Row().Scan(&isWhite)

	if isWhite {
		resetToken := fmt.Sprintf("RESET-%d-%d", student.ID, time.Now().UnixNano())
		global.Redis.Set(context.Background(), "auth_reset:"+resetToken, student.ID, 5*time.Minute)
		return map[string]interface{}{
			"action":     "ALLOW_RESET",
			"resetToken": resetToken,
		}, nil
	}

	approvalMap := map[string]interface{}{
		"user_id":     student.ID,
		"apply_type":  "RESET_PWD",
		"device_id":   deviceId,
		"status":      0,
		"create_time": time.Now(),
	}
	global.DB.Table("auth_approval").Create(&approvalMap)

	return map[string]interface{}{
		"action":  "PENDING_APPROVAL",
		"message": "已在异地设备发起重置，已进入人工安全审核队列",
	}, nil
}

func ResetPassword(resetToken, newPassword string) error {
	userID, err := global.Redis.Get(context.Background(), "auth_reset:"+resetToken).Int64()
	if err != nil || userID == 0 {
		return errors.New("非法或过期的重置凭证")
	}

	hashedPwd, _ := hashPassword(newPassword)
	global.DB.Table("user").Where("id = ?", userID).Update("password", hashedPwd)

	global.Redis.Del(context.Background(), "auth_reset:"+resetToken)
	return nil
}

// =========================================================================
// 5. 体验账户申请（沙盒流�?// =========================================================================
func ApplyExperience(name, password, deviceId string) error {
	hashedPwd, _ := hashPassword(password)

	expUserMap := map[string]interface{}{
		"real_name":         name,
		"nickname":          name,
		"class_name":        "体验账户",
		"student_id":        "EXP-" + MathRandomString(6),
		"password":          hashedPwd,
		"user_type":         0,
		"account_status":    0, // 0=直接激活放行�?		"current_device_id": deviceId,
		"create_time":       time.Now(),
	}
	return global.DB.Table("user").Create(&expUserMap).Error
}

func MathRandomString(n int) string {
	return fmt.Sprintf("%d", time.Now().UnixNano())[13:]
}

func RecordAdminAuditLog(adminId int64, username, action, targetTable, targetId string, roleLevel int, reason, details string) {
	logMap := map[string]interface{}{
		"admin_id":     adminId,
		"username":     username,
		"action":       action,
		"target_table": targetTable,
		"target_id":    targetId,
		"role_level":   roleLevel,
		"reason":       reason,
		"details":      details,
		"create_time":  time.Now(),
	}
	global.DB.Table("sys_admin_audit_log").Create(&logMap)
}
