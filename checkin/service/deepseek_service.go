package service

// deepseek_service.go - 已物理断开 AI 连接，桩函数留档

// CalculateDynamicRate 动态计算技能点与赋能分的兑换倍率 - 已安全退化为纯本地基础汇率，无需AI计算
func CalculateDynamicRate(baseRate float64, marketTrend string) float64 {
	return baseRate
}

// CheckPayloadSafe 白天实时风控 - 已退化为默认通过
func CheckPayloadSafe(content string) bool {
	return true
}

// EvaluateCheckin 夜间打分引擎 - 已退化为默认通过与默认分数评估
func EvaluateCheckin(ocrText string, minScore, maxScore int) (score int, comment string) {
	return (minScore + maxScore) / 2, "[本地评估] 格式合规安全"
}

// DeepSeekEvaluatePost 发帖风控 - 已退化为默认安全放行
func DeepSeekEvaluatePost(content, imageUrl string) (bool, string) {
	return true, "SAFE"
}

func callDeepSeek(prompt string) string {
	// 物理屏蔽 DeepSeek 大模型外部调用，直接返回空值
	return ""
}
