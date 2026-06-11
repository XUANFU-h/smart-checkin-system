package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"checkin-go/models"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// GetTrackList 专门供新用户注册时展示可用赛道
func GetTrackList(c *gin.Context) {
	var tracks []models.Track
	if err := global.DB.Find(&tracks).Error; err != nil {
		common.Error(c, "拉取赛道列表失败")
		return
	}
	common.Success(c, tracks)
}

// UploadImage 基础图片上传接驳口 (供申诉等模块使用)
func UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		common.Error(c, "文件接收失败")
		return
	}

	// 这里使用时间戳命名并存入本地，后续可按需替换为真实的 OSS 逻辑
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(file.Filename))
	savePath := "./uploads/" + fileName

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		common.Error(c, "文件保存失败")
		return
	}

	// 返回前端可访问的相对路径 (需配合静态资源服务)
	common.Success(c, gin.H{"imageUrl": "/uploads/" + fileName})
}
