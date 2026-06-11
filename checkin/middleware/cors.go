package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware 跨域中间件（完美兼容本地带外钥匙 null 原点放行版）
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 💯 终极消灭 CORS 拦截：当本地双击 index.html 产生 "null" 时，
		// 必须精准镜像返回 "null" 或者当前的真实原点，绝不能漏掉，否则浏览器直接物理拦截
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")

		// 💡 核心对齐：本地 file:// 协议跨网轰炸时，如果 Origin 为 "null"，
		// 浏览器禁止 Credentials 设为 true。此处针对本地控制台自动分流自适应，防止浏览器报错
		if origin == "null" {
			c.Header("Access-Control-Allow-Credentials", "false")
		} else {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 放行所有 OPTIONS 预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
