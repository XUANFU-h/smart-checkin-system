package utils

import (
	"checkin-go/global"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 放在 utils 里，供 middleware 和 service 共同调用
type CustomClaims struct {
	UserID int64  `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成标准的 JWT 令牌 (HS256)
func GenerateToken(userID int64, role string) (string, error) {
	// 💡 防线 1：秘钥空值兜底，防止因为环境变量未读取到导致签名失败
	secret := global.Config.JWT.Secret
	if secret == "" {
		secret = "smart_checkin_fallback_secret_2026_xuanfu"
	}

	// 💡 防线 2：过期时间 0 值兜底，彻底粉碎“上线即死亡”的 401 拦截
	expire := global.Config.JWT.Expire
	if expire <= 0 {
		expire = 720 // 如果读取配置失败，强行兜底 30 天有效期
	}

	claims := CustomClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expire) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "smart-checkin-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析并校验令牌 (顺手补齐工具函数，方便其他模块使用)
func ParseToken(tokenStr string) (*CustomClaims, error) {
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	secret := global.Config.JWT.Secret
	if secret == "" {
		secret = "smart_checkin_fallback_secret_2026_xuanfu"
	}

	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("令牌签名异常或已过期")
}
