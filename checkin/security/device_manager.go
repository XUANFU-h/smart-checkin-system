package security

import (
	"checkin-go/global"
	"checkin-go/websocket"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9" // 👇 修复点：引入 redis 官方包以使用 redis.Z
)

const MaxDevicesPerUser = 3

// RecordDeviceLogin 用户登录成功后调用此方法，记录设备并执行 3 设备互踢逻辑
func RecordDeviceLogin(userID int64, deviceID string, token string) {
	ctx := context.Background()

	// 1. 记录当前设备的 Token (Hash 结构: deviceId -> token)
	hashKey := fmt.Sprintf("user_devices_token:%d", userID)
	global.Redis.HSet(ctx, hashKey, deviceID, token)
	global.Redis.Expire(ctx, hashKey, 7*24*time.Hour) // 跟随 Token 过期时间

	// 2. 维护一个有序集合 (ZSet)，记录登录时间排序，用于找出最老的设备
	zsetKey := fmt.Sprintf("user_devices_sort:%d", userID)
	nowScore := float64(time.Now().UnixMilli())

	// 👇 修复点：使用 redis.Z
	global.Redis.ZAdd(ctx, zsetKey, redis.Z{Score: nowScore, Member: deviceID})
	global.Redis.Expire(ctx, zsetKey, 7*24*time.Hour)

	// 3. 检查设备总数是否超过 3 台
	deviceCount := global.Redis.ZCard(ctx, zsetKey).Val()
	if deviceCount > MaxDevicesPerUser {
		// 找出最老的那台设备 (Score 最小的 1 个)
		oldestDevices, err := global.Redis.ZRange(ctx, zsetKey, 0, 0).Result()
		if err == nil && len(oldestDevices) > 0 {
			oldestDeviceID := oldestDevices[0]

			// A. 从 Redis 中删除最老设备的记录，使其 Token 失效
			global.Redis.ZRem(ctx, zsetKey, oldestDeviceID)
			global.Redis.HDel(ctx, hashKey, oldestDeviceID)

			// B. 触发 WebSocket 实时踢下线通知
			kickMsg := "您的账号已在其他新设备登录。为保障安全，此设备已被强制下线。限制最高同时在线 3 台设备。"
			websocket.Manager.SendToSpecificDevice(userID, oldestDeviceID, websocket.MsgTypeKickOut, kickMsg)
		}
	}
}

// IsDeviceTokenValid 在 JWT 中间件中调用，判断该请求的设备 Token 是否仍然有效
func IsDeviceTokenValid(userID int64, deviceID string, clientToken string) bool {
	ctx := context.Background()
	hashKey := fmt.Sprintf("user_devices_token:%d", userID)

	// 从 Redis 获取该设备最新绑定的 Token
	savedToken, err := global.Redis.HGet(ctx, hashKey, deviceID).Result()
	if err != nil {
		return false // 设备不存在或已过期
	}

	// 如果传上来的 token 和 Redis 里存的不一致，说明该设备上之前的登录状态已经被覆盖
	return savedToken == clientToken
}
