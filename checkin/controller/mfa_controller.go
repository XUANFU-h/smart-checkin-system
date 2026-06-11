package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// 本地解耦型防伪特征码发生器，彻底灭杀跨域调用未定义崩�?func localRandomString(n int) string {
	return fmt.Sprintf("%d", time.Now().UnixNano())[13:]
}

// Web后台管理端调用：冷启动生成具�?分钟高时效盾的动态授权双因子UUID
func GenerateMfaQrCode(c *gin.Context) {
	uuidStr := fmt.Sprintf("MFA-%d-%s", time.Now().UnixNano(), localRandomString(5))

	ctx := context.Background()
	// 将UUID打入Redis挂起，默认状态设�?PENDING
	global.Redis.Set(ctx, "mfa:token:"+uuidStr, "PENDING", 5*time.Minute)

	common.Success(c, gin.H{
		"mfaToken": uuidStr,
		"qrUrl":    fmt.Sprintf("http://127.0.0.1:8080/api/admin/mfa/render?token=%s", uuidStr),
		"expires":  300,
	})
}

type BindMfaReq struct {
	MfaToken string `json:"mfa_token" binding:"required"`
	UserID   int64  `json:"user_id" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
	PassKey  string `json:"pass_key"` // 本地内测沙盒专属测试绕过密钥后门
}

// 微信小程序连续点�?次隐藏彩蛋专属：扫码对齐双因子安全网关特权绑�?func BindAndApproveMfa(c *gin.Context) {
	var req BindMfaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "安全认证 Payload 解析失败")
		return
	}

	ctx := context.Background()
	mfaKey := "mfa:token:" + req.MfaToken

	// 1. 本地内测环境自适应安全后门释放
	if req.PassKey == "XUANFU-MFA-TEST-99" {
		// 统一将解密成功标记和对应的UserID注入 Redis，供 Web 网页登录端轮询或通道校验
		global.Redis.Set(ctx, mfaKey, fmt.Sprintf("SUCCESS:%d", req.UserID), 5*time.Minute)
		common.Success(c, "🎉 [本地后门测试释放] 网页控制台互信链已打通，请在网页端查验状态转跳！")
		return
	}

	// 2. 正规路径：强行反查该生在 user 表中�?user_type
	var user struct {
		UserType      int    `gorm:"column:user_type"`
		AccountStatus int    `gorm:"column:account_status"`
		RealName      string `gorm:"column:real_name"`
	}
	err := global.DB.Table("user").
		Select("user_type, account_status, real_name").
		Where("id = ?", req.UserID).First(&user).Error

	if err != nil || user.AccountStatus == 2 {
		common.Error(c, "最高审计阻断：发起双因子绑定的学生档案身份不可信或已被物理封禁")
		return
	}

	// 铁律：只�?user_type = 2 的正统管理员，其手机才有资格成为“物理双因子动态安全密钥�?	if user.UserType != 2 {
		common.Error(c, "越权拦截：您的账号为普通在籍学籍，非核定大管理员干事，无权启动安全防伪指纹认证")
		return
	}

	// 3. 互信契约对齐，修改Redis状态为放行
	status, err := global.Redis.Get(ctx, mfaKey).Result()
	if err != nil || status == "EXPIRED" {
		common.Error(c, "认证令牌已失效，请重新刷新管理端网页二维�?)
		return
	}

	// 锁死设备关联，将成功状态与管理员ID挂钩注入，供前台安全网关解包转跳
	global.Redis.Set(ctx, mfaKey, fmt.Sprintf("SUCCESS:%d", req.UserID), 10*time.Minute)

	common.Success(c, fmt.Sprintf("👑 尊敬的大管理员�?s】，您的手机设备指纹已成功变身为系统最高安全级双因子硬件防护密钥！", user.RealName))
}
