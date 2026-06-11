package service

import (
	"bytes"
	"checkin-go/global"
	"checkin-go/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func SendWechatTemplateMsg(userID int64, openid, templateID string, page string, data map[string]interface{}) error {
	ctx := context.Background()

	tokenKey := "wx:access_token"
	accessToken, err := global.Redis.Get(ctx, tokenKey).Result()
	if err != nil || accessToken == "" {
		accessToken, err = fetchWxAccessToken()
		if err != nil {
			return err
		}
		global.Redis.Set(ctx, tokenKey, accessToken, 7000*time.Second)
	}

	payload := map[string]interface{}{
		"touser":            openid,
		"template_id":       templateID,
		"page":              page,
		"data":              data,
		"miniprogram_state": "formal",
		"lang":              "zh_CN",
	}

	jsonBytes, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s", accessToken)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var wxResp struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
	}
	json.Unmarshal(body, &wxResp)

	status := int8(1)
	if wxResp.Errcode != 0 {
		status = 0
		slog.Error("微信消息发送失败", "errcode", wxResp.Errcode, "errmsg", wxResp.Errmsg)
	}

	global.DB.Create(&models.WxMsgLog{
		UserID:      userID,
		Openid:      openid,
		TemplateID:  templateID,
		SendContent: string(jsonBytes),
		Status:      status,
		ErrorMsg:    wxResp.Errmsg,
	})

	if status == 0 {
		return fmt.Errorf("推送失败: %s", wxResp.Errmsg)
	}
	return nil
}

func fetchWxAccessToken() (string, error) {
	// 💡 修复：对接全新的全局层级
	appid := global.Config.WeChat.Miniapp.Appid
	secret := global.Config.WeChat.Miniapp.Secret
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appid, secret)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res struct {
		AccessToken string `json:"access_token"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	json.Unmarshal(body, &res)

	if res.Errcode != 0 {
		return "", fmt.Errorf("获取AccessToken失败: %s", res.Errmsg)
	}
	return res.AccessToken, nil
}

// NotifyAppealResult 申诉处理结果通知
func NotifyAppealResult(userID int64, openid, statusStr, reply string) {
	templateID := "aYdt-BAZoGs2BpiL24iHMtkHgPKc9zSOBiMeggYf5h8"
	data := map[string]interface{}{
		"phrase1": map[string]string{"value": statusStr},
		"thing2":  map[string]string{"value": reply},
		"time3":   map[string]string{"value": time.Now().Format("2006-01-02 15:04:05")},
	}
	SendWechatTemplateMsg(userID, openid, templateID, "pages/appealList/appealList", data)
}

// NotifyBugReward Bug反馈奖励通知
func NotifyBugReward(userID int64, openid, rewardText, reply string) {
	templateID := "xtJrOeId4e__GV_IUX0KS_rDDYVJWPwG76pWL60La58"
	data := map[string]interface{}{
		"thing1": map[string]string{"value": "系统Bug反馈奖励"},
		"thing2": map[string]string{"value": rewardText},
		"thing3": map[string]string{"value": reply},
	}
	SendWechatTemplateMsg(userID, openid, templateID, "pages/user/bug-list", data)
}

// NotifyCheckinResult 每日早晨打卡结果通知
func NotifyCheckinResult(userID int64, openid, trackName, resultStr, scoreInfo string) {
	templateID := "CSTAhTuY1-qngCvB_HDsFxHROca9UuEKd1wYfnVq-Fs"
	data := map[string]interface{}{
		"thing1":  map[string]string{"value": trackName},
		"phrase2": map[string]string{"value": resultStr},
		"thing3":  map[string]string{"value": scoreInfo},
		"time4":   map[string]string{"value": time.Now().Format("2006-01-02 15:04:05")},
	}
	SendWechatTemplateMsg(userID, openid, templateID, "pages/home/home", data)
}
