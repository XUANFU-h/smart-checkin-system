package service

// volcengine_ocr_service.go - 已物理断开 AI 连接，桩函数留档

// OcrResult 火山引擎 OCR 响应结构
type OcrResult struct {
	Code int `json:"code"`
	Data struct {
		Text string `json:"text"`
	} `json:"data"`
}

// ExtractTextFromImage 调用火山引擎通用 OCR - 已安全物理注释并关闭外部接口
func ExtractTextFromImage(imageUrl string) (string, error) {
	return "", nil
}
