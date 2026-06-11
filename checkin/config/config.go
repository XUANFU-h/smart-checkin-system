package config

// ServerConfig 完整映射 application.yml 或 config.yaml
type ServerConfig struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`

	JWT struct {
		Secret string `mapstructure:"secret"`
		Expire int    `mapstructure:"expire"`
	} `mapstructure:"jwt"`

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

	Wx struct {
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
			Endpoint  string `mapstructure:"endpoint"` // 补充 OCR Endpoint
		} `mapstructure:"volcengine"`
		DeepSeek struct {
			APIKey  string `mapstructure:"api-key"`
			BaseURL string `mapstructure:"base-url"`
			Model   string `mapstructure:"model"`
		} `mapstructure:"deepseek"`
	} `mapstructure:"ai"`
}
