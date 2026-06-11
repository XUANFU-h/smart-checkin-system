package service

import (
	"checkin-go/global"
	"checkin-go/models"
	"checkin-go/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type WxSessionResp struct {
	Openid     string `json:"openid"`
	SessionKey string `json:"session_key"`
	Unionid    string `json:"unionid"`
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
}

func WxLogin(code string) (map[string]interface{}, error) {
	// 💡 修复：对接全新的全局层级
	appid := global.Config.WeChat.Miniapp.Appid
	secret := global.Config.WeChat.Miniapp.Secret
	if appid == "" || secret == "" {
		return nil, errors.New("服务器微信小程序配置(Appid/Secret)缺失")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", appid, secret, code)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.New("请求微信服务器失败")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var wxResp WxSessionResp
	if err := json.Unmarshal(body, &wxResp); err != nil {
		return nil, errors.New("解析微信响应失败")
	}

	if wxResp.Errcode != 0 || wxResp.Openid == "" {
		return nil, fmt.Errorf("微信登录失败: %s", wxResp.Errmsg)
	}

	var user models.User
	err = global.DB.Where("openid = ?", wxResp.Openid).First(&user).Error

	if err != nil {
		return map[string]interface{}{
			"action": "REQUIRE_BIND",
			"openid": wxResp.Openid,
		}, nil
	}

	if user.AccountStatus == 2 {
		return nil, errors.New("【风控拦截】您的账号存在违规行为已被封禁，请联系辅导员处理")
	}

	token, err := utils.GenerateToken(user.ID, "STUDENT")
	if err != nil {
		return nil, errors.New("系统异常：生成令牌失败")
	}

	return map[string]interface{}{
		"action": "LOGIN_SUCCESS",
		"token":  token,
		"userId": user.ID,
		"avatar": user.Avatar,
	}, nil
}

// GetOpenIDByCode 静默交换微信 code 得到 OpenID
func GetOpenIDByCode(code string) (string, error) {
	appid := global.Config.WeChat.Miniapp.Appid
	secret := global.Config.WeChat.Miniapp.Secret
	if appid == "" || secret == "" {
		return "", errors.New("服务器微信小程序配置(Appid/Secret)缺失")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", appid, secret, code)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.New("请求微信服务器失败")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var wxResp struct {
		Openid  string `json:"openid"`
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &wxResp); err != nil {
		return "", errors.New("解析微信响应失败")
	}

	if wxResp.Errcode != 0 || wxResp.Openid == "" {
		return "", fmt.Errorf("微信接口调用失败: %s", wxResp.Errmsg)
	}

	return wxResp.Openid, nil
}
