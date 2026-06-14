// Package ws 封装闲鱼 WebSocket 实时通信，包括连接管理、消息收发、心跳保活等。
package ws

import (
	"context"
	"sync"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/msg"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

// MessageHandler 消息处理回调函数类型。
// 当收到闲鱼用户消息时，该函数被调用。
type MessageHandler func(m *msg.Message)

// XianyuWS 闲鱼 WebSocket 客户端。
//
// 该结构体管理闲鱼 WebSocket 长连接，负责:
//   - 连接建立和初始化（Token 获取、注册、状态同步）
//   - 心跳保活（每 15 秒发送心跳包）
//   - 消息接收和处理
//   - Token 自动刷新（每 10 分钟）
type XianyuWS struct {
	api      *apis.XianyuAPI    // HTTP API 实例
	ws       *websocket.Conn    // WebSocket 连接
	myID     string             // 当前用户 unb
	deviceID string             // 设备 ID
	ctx      context.Context    // 控制上下文
	cancel   context.CancelFunc // 取消函数

	mu         sync.RWMutex   // 保护 msgHandler
	msgHandler MessageHandler // 消息处理回调
	logger     *zap.Logger    // 日志
}

// New 创建 WebSocket 客户端。
//
// 参数:
//   - cookies:  登录后的 Cookie 字典，必须包含 "unb" 字段
//   - deviceID: 可选的设备 ID，为空时自动从 unb 生成
//
// 返回值:
//   - *XianyuWS: 初始化的 WebSocket 客户端实例
//   - error: 创建失败时的错误
func New(cookies map[string]string, deviceID string) (*XianyuWS, error) {
	ctx, cancel := context.WithCancel(context.Background())
	api, err := apis.New(cookies, deviceID)
	if err != nil {
		cancel()
		return nil, err
	}

	return &XianyuWS{
		api:      api,
		myID:     cookies["unb"],
		deviceID: api.DeviceID(),
		ctx:      ctx,
		cancel:   cancel,
		logger:   zap.L(),
	}, nil
}

// SetMessageHandler 设置消息处理回调。
//
// 该回调在收到用户消息时被调用。回调函数应在合理时间内返回，
// 长时间阻塞会影响后续消息的处理。如需耗时处理，请在回调内启动 goroutine。
func (ws *XianyuWS) SetMessageHandler(h MessageHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.msgHandler = h
}

// DeviceID 返回当前客户端的设备 ID。
func (ws *XianyuWS) DeviceID() string {
	return ws.deviceID
}

// MyID 返回当前用户的 unb ID。
func (ws *XianyuWS) MyID() string {
	return ws.myID
}

// Stop 停止 WebSocket 客户端。
func (ws *XianyuWS) Stop() {
	ws.cancel()
	if ws.ws != nil {
		ws.ws.Close(websocket.StatusNormalClosure, "client stop")
	}
}
