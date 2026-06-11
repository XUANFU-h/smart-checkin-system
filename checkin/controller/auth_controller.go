package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/security"
	"checkin-go/service"
	"checkin-go/utils"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 💡 本地内建的高时效轻量随机序列发生器
func getCaptchaRandomSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())[13:]
}

// 💯 生产级闭环：纯原生轻量化动态算术 SVG 验证码引擎（绝不依赖第三方库、100%本地计算输出）
func generateSvgCaptcha(a, b int) (string, string) {
	// 随机决定是加法还是减法
	rand.Seed(time.Now().UnixNano())
	operator := "+"
	answer := a + b
	if rand.Intn(2) == 0 && a >= b {
		operator = "-"
		answer = a - b
	}

	// 动态绘制具有一定防机器扫描干扰的 SVG 矢量图形流 (优化宽高比，完美解决文本截断问题)
	svgTemplate := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="150" height="40" viewBox="0 0 150 40">
			<rect width="100%%" height="100%%" fill="#1e272e"/>
			<line x1="10" y1="15" x2="140" y2="25" stroke="#341f97" stroke-width="2"/>
			<line x1="15" y1="30" x2="135" y2="10" stroke="#10ac84" stroke-width="1.5"/>
			<text x="50%%" y="60%%" fill="#2ed573" font-size="16" font-family="monospace" font-weight="bold" text-anchor="middle" letter-spacing="2">
				%d %s %d = ?
			</text>
		</svg>`, a, operator, b,
	)

	// 将生成的动态 SVG 内容转换为标准的 Base64 图形数据流（前端 <image> 标签可完美无缝直连直刷）
	base64Data := base64.StdEncoding.EncodeToString([]byte(svgTemplate))
	return fmt.Sprintf("data:image/svg+xml;base64,%s", base64Data), fmt.Sprintf("%d", answer)
}

// -------------------------------------------------------------------------
// 1. 统一登录及凭证验证域
// -------------------------------------------------------------------------

type LoginReq struct {
	Name          string `json:"name" binding:"required"`
	Class         string `json:"class"`          // 兼容 class
	Clazz         string `json:"clazz"`          // 兼容 clazz (小程序原版传递此项)
	Password      string `json:"password" binding:"required"`
	DeviceID      string `json:"device_id"`      // 兼容 snake_case
	DeviceIDCamel string `json:"deviceId"`       // 兼容 camelCase (小程序 login.js 传递此项)
	CaptchaId     string `json:"captchaId" binding:"required"`     // 💯 补齐强效约束
	CaptchaAnswer string `json:"captchaAnswer" binding:"required"` // 💯 补齐强效约束
}

// Login 统管登录入口
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "登录参数解析失败，请检查验证码或账密是否填全")
		return
	}

	// 🔒 验证码核验网关
	ctx := context.Background()
	captchaKey := "captcha:" + req.CaptchaId
	realAnswer, err := global.Redis.Get(ctx, captchaKey).Result()
	if err != nil || realAnswer == "" {
		common.Error(c, "验证码已过期，请点击图形刷新")
		return
	}

	if req.CaptchaAnswer != realAnswer {
		common.Error(c, "安全核算失败：算术得数输入错误，防刷盾拦截！")
		return
	}

	// 销毁验证码，防止重放攻击
	global.Redis.Del(ctx, captchaKey)

	// 💡 自适应分流网页与移动端不同参数命名空间
	finalClass := req.Class
	if finalClass == "" {
		finalClass = req.Clazz
	}

	finalDeviceID := req.DeviceID
	if finalDeviceID == "" {
		finalDeviceID = req.DeviceIDCamel
	}
	if finalDeviceID == "" {
		finalDeviceID = c.GetHeader("X-Device-Id") // 🚀 双保险：自动向上回溯捕获 HTTP 头部携带的设备指纹
	}
	if finalDeviceID == "" {
		finalDeviceID = "WEB-ADMIN-CONSOLE-DASHBOARD"
	}

	resData, err := service.LoginUser(req.Name, finalClass, req.Password, finalDeviceID)
	if err != nil {
		common.Error(c, "风控盾阻断: "+err.Error())
		return
	}

	common.Success(c, resData)
}

type WxLoginReq struct {
	Code     string `json:"code" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
}

// WxLogin 微信小程序静默登录网关
func WxLogin(c *gin.Context) {
	var req WxLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "微信凭证缺失")
		return
	}
	common.Error(c, "微信通道正处于本地沙盒模拟环境，请使用标准实名账密登录")
}

// -------------------------------------------------------------------------
// 2. 二合一原子注册与字典服务域
// -------------------------------------------------------------------------

type UnifiedRegisterReq struct {
	Name          string `json:"name" binding:"required"`
	Class         string `json:"class" binding:"required"`
	StudentID     string `json:"student_id" binding:"required"`
	Password      string `json:"password" binding:"required"`
	DeviceID      string `json:"device_id" binding:"required"`
	TrackIDs      []int  `json:"track_ids" binding:"required"`
	IsAdminReg    bool   `json:"is_admin_reg"`
	Backdoor      string `json:"backdoor"`
	CaptchaId     string `json:"captchaId" binding:"required"`     // 💯 补齐强效约束
	CaptchaAnswer string `json:"captchaAnswer" binding:"required"` // 💯 补齐强效约束
}

// RegisterWithTracks 全栈二合一：【注册账号 + 绑定赛道】原子两阶段提交中央路由接口
func RegisterWithTracks(c *gin.Context) {
	var req UnifiedRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("⚠️ [注册控制器参数拦截] Gin反序列化失败原因: %v", err)
		common.Error(c, "参数解析失败，注册表单项残缺")
		return
	}

	// 🔒 验证码核验网关
	ctx := context.Background()
	captchaKey := "captcha:" + req.CaptchaId
	realAnswer, err := global.Redis.Get(ctx, captchaKey).Result()
	if err != nil || realAnswer == "" {
		common.Error(c, "防刷审计失败：注册算术验证码错误或已失效")
		return
	}

	if req.CaptchaAnswer != realAnswer {
		common.Error(c, "防刷审计失败：注册算术验证码错误或已失效")
		return
	}

	// 销毁验证码，防止重放攻击
	global.Redis.Del(ctx, captchaKey)

	resData, err := service.RegisterWithTracksService(
		req.Name,
		req.Class,
		req.StudentID,
		req.Password,
		req.DeviceID,
		req.TrackIDs,
		req.IsAdminReg,
		req.Backdoor,
	)

	if err != nil {
		common.Error(c, "安全网关回滚拦截: "+err.Error())
		return
	}

	common.Success(c, resData)
}

// GetDistinctClasses 获取标准班级字典
func GetDistinctClasses(c *gin.Context) {
	var list []string
	global.DB.Table("class_list").Order("id ASC").Pluck("class_name", &list)
	common.Success(c, list)
}

// -------------------------------------------------------------------------
// 3. 安全找回凭证与验证码降维域
// -------------------------------------------------------------------------

type ForgotReq struct {
	Name          string `json:"name" binding:"required"`
	Class         string `json:"class" binding:"required"`
	StudentID     string `json:"student_id" binding:"required"`
	DeviceID      string `json:"device_id" binding:"required"`
	CaptchaId     string `json:"captchaId" binding:"required"`     // 💯 彻底对齐前端找回密码页
	CaptchaAnswer string `json:"captchaAnswer" binding:"required"` // 💯 彻底对齐前端找回密码页
}

func ForgotPasswordApply(c *gin.Context) {
	var req ForgotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "申请资料不完整，验证码未填或参数错误")
		return
	}

	// 🔒 💯 补齐找回密码防刷网关：不给恶意爆破留一丝后门
	ctx := context.Background()
	captchaKey := "captcha:" + req.CaptchaId
	realAnswer, err := global.Redis.Get(ctx, captchaKey).Result()
	if err != nil || req.CaptchaAnswer != realAnswer {
		common.Error(c, "验证码校验失败：算术得数输入错误或已失效")
		return
	}
	global.Redis.Del(ctx, captchaKey)

	res, err := service.ApplyForgotPassword(req.Name, req.Class, req.StudentID, req.DeviceID)
	if err != nil {
		common.Error(c, err.Error())
		return
	}
	common.Success(c, res)
}

type ResetReq struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func ForgotPasswordReset(c *gin.Context) {
	var req ResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "重置凭证缺失")
		return
	}

	err := service.ResetPassword(req.ResetToken, req.NewPassword)
	if err != nil {
		common.Error(c, err.Error())
		return
	}
	common.Success(c, "密码重置成功")
}

// -------------------------------------------------------------------------
// 4. 沙盒与开放回执支持域
// -------------------------------------------------------------------------

func ExperienceApply(c *gin.Context) {
	type ExpReq struct {
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required"`
		DeviceID string `json:"device_id" binding:"required"`
	}
	var req ExpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "资料未填全")
		return
	}

	err := service.ApplyExperience(req.Name, req.Password, req.DeviceID)
	if err != nil {
		common.Error(c, "申请录入失败")
		return
	}
	common.Success(c, "体验账户已安全进入沙盒挂起序列")
}

// GenerateCaptcha 自动发放具有高性能算术图文混淆的真实防伪盾凭条
func GenerateCaptcha(c *gin.Context) {
	rand.Seed(time.Now().UnixNano())
	num1 := rand.Intn(9) + 1 // 1 ~ 10
	num2 := rand.Intn(9) + 1

	// 调用原生图形引擎，生成实时的 SVG 图像和对应算术答案
	captchaImageBase64, realAnswer := generateSvgCaptcha(num1, num2)

	captchaID := fmt.Sprintf("CAP-%d-%s", time.Now().UnixNano(), getCaptchaRandomSuffix())

	// 💯 真实入库绑定：将当前验证码 ID 与真答案打入 Redis，维持 5 分钟安全审计盾
	ctx := context.Background()
	global.Redis.Set(ctx, "captcha:"+captchaID, realAnswer, 5*time.Minute)

	// 回归标准响应层级
	common.Success(c, gin.H{
		"captchaId":    captchaID,
		"captchaImage": captchaImageBase64, // 抛出 Base64 流数据
	})
}

// =========================================================================
// 5. 🚀 微信小程序互信绑定与扫码登录高阶安全 API 组
// =========================================================================

// BindWechatMfaRequest 绑定微信 MFA 请求体
type BindWechatMfaRequest struct {
	UUID          string `json:"uuid" binding:"required"`
	BindingSecret string `json:"binding_secret" binding:"required"`
}

// BindWechatMfa 微信端核验并绑定
func BindWechatMfa(c *gin.Context) {
	var req BindWechatMfaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误: "+err.Error())
		return
	}

	studentIDVal := c.MustGet("userId").(int64)
	var student models.User
	if err := global.DB.First(&student, studentIDVal).Error; err != nil {
		common.Error(c, "未找到微信小程序用户数据")
		return
	}

	// 🔍 核实管理员名册：学生学号必须与 admin_user.student_id 匹配
	var admin models.AdminUser
	if err := global.DB.Where("student_id = ?", student.StudentID).First(&admin).Error; err != nil {
		common.Error(c, "未在管理员名册中检索到您的学号，请联系超级管理员配置")
		return
	}

	// 🔍 核实姓名一致性：微信实名与管理员名册必须完全一致
	if admin.Username != student.RealName {
		common.Error(c, "绑定失败：您的微信实名与管理员名册中的姓名不匹配！")
		return
	}

	// 🔍 核实动态密钥
	if strings.ToUpper(req.BindingSecret) != strings.ToUpper(admin.BindingSecret) {
		common.Error(c, "微信特征绑定安全密钥错误！")
		return
	}

	// 🔍 核实动态密钥的时效生命周期 (10分钟时效校验)
	if admin.SecretExpireTime == nil || admin.SecretExpireTime.Before(time.Now()) {
		common.Error(c, "绑定安全特征密钥已超时过期，请联系超级管理员在控制台重新刷新生成！")
		return
	}

	// 🔍 核实重复绑定
	if admin.IsBound {
		common.Error(c, "该管理员账户已完成绑定，无需重复绑定！")
		return
	}

	// ⚡ 事务原子落库：同步更改 admin_user 绑定信息与学生 user_type 身份，同时物理销毁动态特征密钥（阅后即焚）
	errTx := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&admin).Updates(map[string]interface{}{
			"is_bound":           true,
			"bound_user_id":      student.ID,
			"binding_secret":     "", // 阅后即焚：成功绑定后永久销毁此特征绑定密钥
			"secret_expire_time": nil,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&student).Update("user_type", 2).Error; err != nil {
			return err
		}
		return nil
	})

	if errTx != nil {
		common.Error(c, "互信契约写入数据库失败: "+errTx.Error())
		return
	}

	// 签发管理员专属 token，缓存进 Redis 供 Web 端轮询自动登入
	roleString := "ADMIN"
	if admin.RoleLevel == 0 {
		roleString = "SUPERADMIN"
	}
	token, err := utils.GenerateToken(admin.ID, roleString)
	if err != nil {
		common.Error(c, "管理员会话令牌生成失败: "+err.Error())
		return
	}

	ctx := context.Background()
	global.Redis.Set(ctx, "admin_bind_status:"+req.UUID, fmt.Sprintf("BOUND:%s:%d:%s", token, admin.ID, admin.Username), 5*time.Minute)

	common.Success(c, "物理双因子特征绑定绑定成功！")
}

// CheckBindStatus 网页端轮询微信绑定状态
func CheckBindStatus(c *gin.Context) {
	uuidStr := c.Query("uuid")
	if uuidStr == "" {
		common.Error(c, "缺少会话特征参数")
		return
	}

	ctx := context.Background()
	status, err := global.Redis.Get(ctx, "admin_bind_status:"+uuidStr).Result()
	if err != nil || status == "" {
		// 校验绑定 session 是否过期
		bindSession, errSession := global.Redis.Get(ctx, "admin_bind:"+uuidStr).Result()
		if errSession != nil || bindSession == "" {
			common.Success(c, gin.H{"status": "EXPIRED"})
			return
		}
		common.Success(c, gin.H{"status": "PENDING"})
		return
	}

	parts := strings.Split(status, ":")
	if len(parts) >= 4 && parts[0] == "BOUND" {
		token := parts[1]
		var adminID int64
		fmt.Sscanf(parts[2], "%d", &adminID)
		username := parts[3]

		// 录入设备登录态，防止 401 瞬间强退
		security.RecordDeviceLogin(adminID, "WEB-ADMIN-CONSOLE-DASHBOARD", token)

		common.Success(c, gin.H{
			"status":   "BOUND",
			"token":    token,
			"adminId":  adminID,
			"nickname": username,
			"role":     "ADMIN",
		})
		return
	}

	common.Success(c, gin.H{"status": "PENDING"})
}

// GenerateLoginQr 生成 UUID 登录二维码
func GenerateLoginQr(c *gin.Context) {
	uuidStr := uuid.New().String()
	ctx := context.Background()
	global.Redis.Set(ctx, "admin_login:"+uuidStr, "PENDING", 5*time.Minute)

	// 使用免费的轻量二维码 API，完美规避 Node 画布打包难题
	qrUrl := "https://api.qrserver.com/v1/create-qr-code/?size=250x250&data=admin-login:" + uuidStr

	common.Success(c, gin.H{
		"uuid":  uuidStr,
		"qrUrl": qrUrl,
	})
}

// ConfirmLoginReq 授权登录请求体
type ConfirmLoginReq struct {
	UUID string `json:"uuid" binding:"required"`
}

// ConfirmQrLogin 微信小程序端授权登录
func ConfirmQrLogin(c *gin.Context) {
	var req ConfirmLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析错误: "+err.Error())
		return
	}

	studentIDVal := c.MustGet("userId").(int64)
	var student models.User
	if err := global.DB.First(&student, studentIDVal).Error; err != nil {
		common.Error(c, "未找到微信小程序用户数据")
		return
	}

	// 🔍 反查微信绑定的管理员
	var admin models.AdminUser
	if err := global.DB.Where("bound_user_id = ? AND is_bound = ?", student.ID, true).First(&admin).Error; err != nil {
		common.Error(c, "授权拒绝：该微信账号尚未绑定任何管理员账户，请先使用密码登录并绑定微信")
		return
	}

	if admin.Status == 1 {
		common.Error(c, "风控硬拦截：您的管理账户已被大超冻结封禁")
		return
	}

	roleString := "ADMIN"
	if admin.RoleLevel == 0 {
		roleString = "SUPERADMIN"
	}
	token, err := utils.GenerateToken(admin.ID, roleString)
	if err != nil {
		common.Error(c, "身份令牌生成失败")
		return
	}

	ctx := context.Background()
	global.Redis.Set(ctx, "admin_login_status:"+req.UUID, fmt.Sprintf("APPROVED:%s:%d:%s:%s", token, admin.ID, admin.Username, roleString), 5*time.Minute)

	common.Success(c, "授权成功！网页端即将自动跳转进入系统。")
}

// CheckLoginQrStatus 网页端轮询二维码扫码登录状态
func CheckLoginQrStatus(c *gin.Context) {
	uuidStr := c.Query("uuid")
	if uuidStr == "" {
		common.Error(c, "缺少会话特征参数")
		return
	}

	ctx := context.Background()
	status, err := global.Redis.Get(ctx, "admin_login_status:"+uuidStr).Result()
	if err != nil || status == "" {
		loginSession, errSession := global.Redis.Get(ctx, "admin_login:"+uuidStr).Result()
		if errSession != nil || loginSession == "" {
			common.Success(c, gin.H{"status": "EXPIRED"})
			return
		}
		common.Success(c, gin.H{"status": "PENDING"})
		return
	}

	parts := strings.Split(status, ":")
	if len(parts) >= 5 && parts[0] == "APPROVED" {
		token := parts[1]
		var adminID int64
		fmt.Sscanf(parts[2], "%d", &adminID)
		username := parts[3]
		role := parts[4]

		security.RecordDeviceLogin(adminID, "WEB-ADMIN-CONSOLE-DASHBOARD", token)

		common.Success(c, gin.H{
			"status":   "APPROVED",
			"token":    token,
			"adminId":  adminID,
			"nickname": username,
			"role":     role,
		})
		return
	}

	common.Success(c, gin.H{"status": "PENDING"})
}

// RemoteLogoutAdmin 子管理员微信端一键切断 Web 登录 Session 强驱下机
func RemoteLogoutAdmin(c *gin.Context) {
	studentIDVal := c.MustGet("userId").(int64)
	var student models.User
	if err := global.DB.First(&student, studentIDVal).Error; err != nil {
		common.Error(c, "未找到微信小程序用户数据")
		return
	}

	// 🔍 反查微信绑定的管理员
	var admin models.AdminUser
	if err := global.DB.Where("bound_user_id = ? AND is_bound = ?", student.ID, true).First(&admin).Error; err != nil {
		common.Error(c, "核实失败：该微信尚未绑定任何管理员账户")
		return
	}

	ctx := context.Background()
	var allKeys []string
	for _, pattern := range []string{
		fmt.Sprintf("user_device:%d:*", admin.ID),
		fmt.Sprintf("device_token:%d:*", admin.ID),
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

	// 写入强退标记，JWT 中间件会强制其 401 秒退
	kickKey := fmt.Sprintf("kick_user:%d", admin.ID)
	global.Redis.Set(ctx, kickKey, "1", 24*time.Hour)

	common.Success(c, "您的 Web 管理控制台所有会话已全部被物理切断，成功强驱下机！")
}

