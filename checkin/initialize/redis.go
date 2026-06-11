package initialize

import (
	"checkin-go/global"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	// 💡 修复：接入全新的 Spring.Data.Redis 结构，注意 Port 现在是 int 类型
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", global.Config.Spring.Data.Redis.Host, global.Config.Spring.Data.Redis.Port),
		Password: global.Config.Spring.Data.Redis.Password,
		DB:       global.Config.Spring.Data.Redis.Database,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		slog.Error("Redis连接失败", "err", err)
		panic(err)
	}

	global.Redis = RedisClient
	slog.Info("✅ Redis连接成功")
}
