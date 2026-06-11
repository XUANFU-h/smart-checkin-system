package controller

import (
	"checkin-go/common"
	"checkin-go/global"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DecryptStepsReq struct {
	Code          string `json:"code" binding:"required"`
	EncryptedData string `json:"encryptedData" binding:"required"`
	IV            string `json:"iv" binding:"required"`
}

type WeChatSessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type WeRunData struct {
	StepInfoList []struct {
		Timestamp int64 `json:"timestamp"`
		Step      int   `json:"step"`
	} `json:"stepInfoList"`
}

func DecryptWechatSteps(c *gin.Context) {
	var req DecryptStepsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Error(c, "参数解析失败：缺失核心加密要素")
		return
	}

	// 💡 修复：正确对接 WeChat.Miniapp 层级
	appID := global.Config.WeChat.Miniapp.Appid
	appSecret := global.Config.WeChat.Miniapp.Secret
	if appID == "" || appSecret == "" {
		common.Error(c, "系统缺少微信开放平台配置，解密引擎无法启动")
		return
	}

	wxURL := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", appID, appSecret, req.Code)

	resp, err := http.Get(wxURL)
	if err != nil {
		common.Error(c, "连接微信风控服务器失败")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var wxResp WeChatSessionResponse
	if err := json.Unmarshal(body, &wxResp); err != nil {
		common.Error(c, "解析微信风控回执失败")
		return
	}

	if wxResp.ErrCode != 0 {
		common.Error(c, fmt.Sprintf("微信鉴权拦截: %s", wxResp.ErrMsg))
		return
	}

	decryptedBytes, err := decryptWeChatData(req.EncryptedData, wxResp.SessionKey, req.IV)
	if err != nil {
		common.Error(c, "数据解密破译失败，可能数据已被篡改")
		return
	}

	var runData WeRunData
	if err := json.Unmarshal(decryptedBytes, &runData); err != nil {
		common.Error(c, "步数报文解析异常")
		return
	}

	common.Success(c, runData)
}

// ==================== 底层密码学解密引擎 ====================
func decryptWeChatData(encryptedData, sessionKey, iv string) ([]byte, error) {
	aesKey, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return nil, err
	}

	aesIV, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, err
	}

	aesCipherText, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	if len(aesCipherText) < aes.BlockSize {
		return nil, errors.New("加密数据块过短")
	}
	if len(aesCipherText)%aes.BlockSize != 0 {
		return nil, errors.New("加密数据非标准对齐")
	}

	mode := cipher.NewCBCDecrypter(block, aesIV)
	mode.CryptBlocks(aesCipherText, aesCipherText)

	return pkcs7Unpad(aesCipherText)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("数据为空，解除填充失败")
	}
	unpadding := int(data[length-1])
	if unpadding > length {
		return nil, errors.New("非法的解密填充位")
	}
	return data[:(length - unpadding)], nil
}
