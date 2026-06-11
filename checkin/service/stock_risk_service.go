package service

import (
	"checkin-go/global"
	"checkin-go/websocket"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// AsyncStockRiskCheck 异步调用 DeepSeek 检查股票操作风险
func AsyncStockRiskCheck(userID int64, tradeType string, amount int, rawPayload string) {
	// 开启异步 Goroutine，不阻塞用户的正常交易速度
	go func() {
		systemPrompt := `你是一个金融风控安全引擎。
正在分析用户的股票交易报文。请判断是否存在脚本外挂、逻辑漏洞利用或网络攻击特征。
如果行为合法，返回 "SAFE"；如果涉嫌技术攻击或异常高频刷单，返回 "BAN"。
请严格输出 JSON: {"status": "SAFE"或"BAN", "reason": "理由"}`

		userPrompt := fmt.Sprintf("交易动作: %s, 数量: %d, 原始解密报文: %s", tradeType, amount, rawPayload)

		// 👇 修复点 1：将两个 prompt 组合，并调用同包（deepseek_service.go）里的 callDeepSeek
		combinedPrompt := systemPrompt + "\n\n" + userPrompt
		result := callDeepSeek(combinedPrompt)

		var risk struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		}
		json.Unmarshal([]byte(result), &risk)

		if risk.Status == "BAN" {
			// ================= 风控封禁逻辑 =================
			banUntil := time.Now().Add(5 * time.Minute)
			global.DB.Exec("UPDATE stock_account SET status = 2, ban_until = ? WHERE user_id = ?", banUntil, userID)

			// 👇 修复点 2：使用新的 WebSocket 广播方法
			msgData := map[string]string{
				"title":   "系统交易风控警告",
				"content": fmt.Sprintf("检测到非法交易参数：%s。您的股票交易权限已被临时封禁 5 分钟！", risk.Reason),
			}
			websocket.Manager.BroadcastToUser(userID, websocket.MsgTypeSystemNotice, msgData)

			slog.Warn("DeepSeek 触发股票沙盒临时封禁", "UserID", userID, "Reason", risk.Reason)
		}
	}()
}
