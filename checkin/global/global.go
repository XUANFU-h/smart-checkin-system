package global

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ConfigStruct 统一的全局配置结构体 (使用 mapstructure 完美对齐 application.yml)
type ConfigStruct struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`

	JWT struct {
		Secret string `mapstructure:"secret"`
		Expire int    `mapstructure:"expire"`
	} `mapstructure:"jwt"`

	// 兼容旧有数据源配置
	Spring struct {
		Datasource struct {
			Url      string `mapstructure:"url"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
		} `mapstructure:"datasource"`

		Data struct {
			Redis struct {
				Host     string `mapstructure:"host"`
				Port     int    `mapstructure:"port"`
				Password string `mapstructure:"password"`
				Database int    `mapstructure:"database"`
			} `mapstructure:"redis"`
		} `mapstructure:"data"`
	} `mapstructure:"spring"`

	Upload struct {
		MaxFileSize    string `mapstructure:"max-file-size"`
		MaxRequestSize string `mapstructure:"max-request-size"`
	} `mapstructure:"upload"`

	// 💡 黄金对齐：将 Wx 全量统一为 WeChat，完美解决路由与解密引擎的报错！
	WeChat struct {
		Miniapp struct {
			Appid  string `mapstructure:"appid"`
			Secret string `mapstructure:"secret"`
		} `mapstructure:"miniapp"`
	} `mapstructure:"wx"`

	Aliyun struct {
		OSS struct {
			Endpoint        string `mapstructure:"endpoint"`
			AccessKeyID     string `mapstructure:"access-key-id"`
			AccessKeySecret string `mapstructure:"access-key-secret"`
			BucketName      string `mapstructure:"bucket-name"`
		} `mapstructure:"oss"`
	} `mapstructure:"aliyun"`

	AI struct {
		Volcengine struct {
			AccessKey string `mapstructure:"access-key"`
			SecretKey string `mapstructure:"secret-key"`
		} `mapstructure:"volcengine"`
		OCR struct {
			Endpoint string `mapstructure:"endpoint"`
		} `mapstructure:"ocr"`
		DeepSeek struct {
			APIKey  string `mapstructure:"api-key"`
			BaseURL string `mapstructure:"base-url"`
			Model   string `mapstructure:"model"`
		} `mapstructure:"deepseek"`
	} `mapstructure:"ai"`
}

// 全局静态变量指针
var (
	DB     *gorm.DB
	Redis  *redis.Client
	Config ConfigStruct // 💡 Viper 解析时请直接 Bind 到此变量上
)
