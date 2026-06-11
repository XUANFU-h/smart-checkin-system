package middleware

import (
	"checkin-go/common"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// GlobalErrorMiddleware 全局错误捕获 (防 Panic)
func GlobalErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("系统致命错误", "err", err)
				common.Error(c, "系统服务异常，请稍后再试")
				c.Abort()
			}
		}()
		c.Next()
	}
}
