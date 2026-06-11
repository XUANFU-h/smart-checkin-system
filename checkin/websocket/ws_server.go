package websocket

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 定义实时消息的类型枚举 (✅ 已补齐配合前端和审批流的所有指令)
const (
	MsgTypeKickOut       = "KICK_OUT"       // 强制下线 (封号/顶号)
	MsgTypeSystemNotice  = "SYSTEM_NOTICE"  // 站内信/全局通知
	MsgTypeCheckinResult = "CHECKIN_RESULT" // 实时打卡/申诉结果反馈
	MsgTypeAuthApproved  = "AUTH_APPROVED"  // 工单(重置密码/体验账户)审批通过，前端自动跳转
	MsgTypeRestore       = "RESTORE"        // 解除封禁恢复正常通知
	MsgTypeRecordDeleted = "RECORD_DELETED" // 打卡记录被删除通知
	MsgTypeNotify        = "NOTIFY"         // 奖励弹窗通知
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// Client 封装单个 WebSocket 连接
type Client struct {
	UserID   int64
	DeviceID string
	Conn     *websocket.Conn
}

// ClientManager 维护所有在线用户的多端连接
type ClientManager struct {
	// 结构: UserID -> DeviceID -> Client
	Clients map[int64]map[string]*Client
	mu      sync.RWMutex
}

var Manager = &ClientManager{
	Clients: make(map[int64]map[string]*Client),
}

// WsHandler 处理前端发起的 ws://.../api/ws 请求
func WsHandler(c *gin.Context) {
	// 从 JWT 中间件获取 userID，同时需要前端在建立 ws时传入 deviceId
	userIDRaw, exists := c.Get("userId")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userID := userIDRaw.(int64)

	deviceID := c.Query("deviceId")
	if deviceID == "" {
		devRaw, devExists := c.Get("deviceId")
		if devExists {
			deviceID = devRaw.(string)
		} else {
			deviceID = "unknown_device"
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("WebSocket升级失败", "err", err)
		return
	}

	client := &Client{
		UserID:   userID,
		DeviceID: deviceID,
		Conn:     conn,
	}

	// 存入全局 Manager
	Manager.mu.Lock()
	if Manager.Clients[userID] == nil {
		Manager.Clients[userID] = make(map[string]*Client)
	}
	Manager.Clients[userID][deviceID] = client
	Manager.mu.Unlock()
	slog.Info("用户设备上线", "userID", userID, "deviceID", deviceID)

	// 监听断开
	defer func() {
		Manager.mu.Lock()
		if _, ok := Manager.Clients[userID]; ok {
			delete(Manager.Clients[userID], deviceID)
			if len(Manager.Clients[userID]) == 0 {
				delete(Manager.Clients, userID)
			}
		}
		Manager.mu.Unlock()
		conn.Close()
		slog.Info("用户设备离线", "userID", userID, "deviceID", deviceID)
	}()

	// 保持连接常驻
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// ================== 发送消息的 API ==================

// BroadcastToUser 发送消息给该用户的所有在线设备 (例如：收到了一封站内信)
func (m *ClientManager) BroadcastToUser(userID int64, msgType string, data interface{}) {
	m.mu.RLock()
	devices, ok := m.Clients[userID]
	m.mu.RUnlock()

	if !ok {
		return // 用户不在线
	}

	for _, client := range devices {
		client.Conn.WriteJSON(map[string]interface{}{
			"type": msgType,
			"data": data,
		})
	}
}

// SendToSpecificDevice 只发送给该用户的某一台指定设备 (例如：异地登录，把旧设备踢下线)
func (m *ClientManager) SendToSpecificDevice(userID int64, deviceID string, msgType string, data interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if devices, ok := m.Clients[userID]; ok {
		if client, ok2 := devices[deviceID]; ok2 {
			client.Conn.WriteJSON(map[string]interface{}{
				"type": msgType,
				"data": data,
			})
			if msgType == MsgTypeKickOut {
				client.Conn.Close()
			}
		}
	}
}

// 💡 🌟 全量补齐加固：面向全局管理员大盘广播的公开安全管道函数！
// 本地加锁，完美规避跨包访问限制，实现数据大屏 3 秒无感数据刷新
func (m *ClientManager) BroadcastToAll(msgType string, data interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, devices := range m.Clients {
		for _, client := range devices {
			_ = client.Conn.WriteJSON(map[string]interface{}{
				"type": msgType,
				"data": data,
			})
		}
	}
}
