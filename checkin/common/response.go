package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 200,
		Msg:  "成功",
		Data: data,
	})
}

func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{ // HTTP状态码保持200，业务状态码标识错误
		Code: 500, // 或其他业务错误码
		Msg:  msg,
		Data: nil,
	})
}
