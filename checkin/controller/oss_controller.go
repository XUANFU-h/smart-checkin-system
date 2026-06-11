package controller

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"checkin-go/common"
	"checkin-go/global"

	"github.com/gin-gonic/gin"
)

// PolicyToken 响应给前端的签名结构体
type PolicyToken struct {
	AccessKeyId string `json:"OSSAccessKeyId"` // 💡 修复：完美匹配前端所需的 OSSAccessKeyId
	Host        string `json:"host"`
	Policy      string `json:"policy"`
	Signature   string `json:"signature"`
	Expire      int64  `json:"expire"`
	Dir         string `json:"dir"`
}

// GetOssSignature 获取阿里云OSS前端直传签名 (对应 /api/oss/signature)
func GetOssSignature(c *gin.Context) {
	ossCfg := global.Config.Aliyun.OSS

	// 1. 组装前端直传的 Host 地址
	// 例如：https://smart-checkin.oss-cn-beijing.aliyuncs.com
	host := fmt.Sprintf("https://%s.%s", ossCfg.BucketName, ossCfg.Endpoint)

	// 2. 指定上传目录（这里我们按日期分文件夹，比如: checkin/2026-05-15/）
	uploadDir := fmt.Sprintf("checkin/%s/", time.Now().Format("2006-01-02"))

	// 3. 设置签名的过期时间（限制前端只能在 30 分钟内上传，极其安全）
	expireTime := time.Now().Add(30 * time.Minute)
	expireUnix := expireTime.Unix()

	// 阿里云要求的 UTC 格式时间
	expireStr := expireTime.UTC().Format("2006-01-02T15:04:05Z")

	// 4. 构建 Policy 策略字典
	policyMap := map[string]interface{}{
		"expiration": expireStr,
		"conditions": []interface{}{
			// 👇 修复点：将 []string 改为 []interface{} 兼容数字和字符串
			[]interface{}{"content-length-range", 0, 104857600}, // 限制最大上传 100MB
			[]interface{}{"starts-with", "$key", uploadDir},     // 限制只能传到我们指定的目录下
		},
	}

	// 5. 对 Policy 进行 JSON 序列化并进行 Base64 编码
	policyBytes, _ := json.Marshal(policyMap)
	policyBase64 := base64.StdEncoding.EncodeToString(policyBytes)

	// 6. 使用 HmacSHA1 算法和你的 Aliyun_Secret 进行数字签名
	mac := hmac.New(sha1.New, []byte(ossCfg.AccessKeySecret))
	mac.Write([]byte(policyBase64))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// 7. 组装响应给前端的 Token
	token := PolicyToken{
		AccessKeyId: ossCfg.AccessKeyID,
		Host:        host,
		Policy:      policyBase64,
		Signature:   signature,
		Expire:      expireUnix,
		Dir:         uploadDir,
	}

	common.Success(c, token)
}
