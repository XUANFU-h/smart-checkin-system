package initialize

import (
	"checkin-go/global"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// 辅助函数：读取环境变量，带默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// InitConfig 加载配置文件到全局变量
func InitConfig() {
	// 加载本地.env文件（本地开发用）
	err := godotenv.Load()
	if err != nil {
		log.Println("未找到本地.env文件，将使用系统环境变量")
	}

	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))

	serverPort, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if serverPort == 0 {
		serverPort = 8080
	}
	global.Config.Server.Port = serverPort

	global.Config.JWT.Secret = getEnv("JWT_SECRET", "smart_checkin_super_secret_key_2026") // 💡 给予默认签名密钥防止为空闪退
	global.Config.JWT.Expire = 720

	// ================= 💡 核心修复：智能拼装本地数据库 DSN 字符串 =================
	dbUser := getEnv("DB_USER", "root")
	dbPwd := getEnv("DB_PASSWORD", "")
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "smart_checkin")

	// 优先读取完整 DB_URL，若无则使用上面读取自 .env 的各个散装变量进行完美动态拼装！
	defaultDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPwd, dbHost, dbPort, dbName)
	global.Config.Spring.Datasource.Url = getEnv("DB_URL", defaultDSN)
	global.Config.Spring.Datasource.Username = dbUser
	global.Config.Spring.Datasource.Password = dbPwd

	// 适配 Spring.Data.Redis 架构
	global.Config.Spring.Data.Redis.Host = getEnv("REDIS_HOST", "127.0.0.1")
	global.Config.Spring.Data.Redis.Port = redisPort
	global.Config.Spring.Data.Redis.Password = getEnv("REDIS_PASSWORD", "")
	global.Config.Spring.Data.Redis.Database = 0

	// 适配 WeChat.Miniapp 架构
	global.Config.WeChat.Miniapp.Appid = getEnv("WX_APPID", "touristappid")
	global.Config.WeChat.Miniapp.Secret = getEnv("WX_SECRET", "")

	// 适配 AI 架构
	global.Config.AI.DeepSeek.APIKey = getEnv("DEEPSEEK_API_KEY", "")
	global.Config.AI.DeepSeek.BaseURL = getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
	global.Config.AI.DeepSeek.Model = getEnv("DEEPSEEK_MODEL", "deepseek-chat")

	// 适配火山引擎 OCR
	global.Config.AI.Volcengine.AccessKey = getEnv("VOLC_OCR_AK", "")
	global.Config.AI.Volcengine.SecretKey = getEnv("VOLC_OCR_SK", "")
	global.Config.AI.OCR.Endpoint = getEnv("VOLC_OCR_ENDPOINT", "https://visual-api.volcengineapi.com")

	// 适配 Aliyun OSS 架构
	global.Config.Aliyun.OSS.Endpoint = getEnv("OSS_ENDPOINT", "")
	global.Config.Aliyun.OSS.BucketName = getEnv("OSS_BUCKET", "")
	global.Config.Aliyun.OSS.AccessKeyID = getEnv("OSS_ACCESS_KEY_ID", "")
	global.Config.Aliyun.OSS.AccessKeySecret = getEnv("OSS_ACCESS_KEY_SECRET", "")

	// 检查必填配置
	if global.Config.JWT.Secret == "" {
		log.Fatal("必填配置项缺失：JWT_SECRET 不能为空")
	}

	log.Println("✅ 配置文件动态组装成功")
}
