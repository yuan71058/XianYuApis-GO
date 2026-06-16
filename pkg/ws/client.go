// Package ws 封装闲鱼 WebSocket 实时通信。
package ws

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/msg"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

// MessageHandler 消息处理回调函数类型。
type MessageHandler func(m *msg.Message)

// XianyuWS 闲鱼 WebSocket 客户端。
//
// 与 Python 版 im_client.py GoofishImClient 对齐:
//   - 使用 nhooyr.io/websocket（支持 permessage-deflate 压缩，与 Python aiohttp 对齐）
//   - Connect: 建立 WebSocket + 启动 recvLoop + send-and-wait /reg + 启动心跳
//   - Start: 等待连接关闭
type XianyuWS struct {
	api      *apis.XianyuAPI       // HTTP API 实例
	ws       *websocket.Conn       // WebSocket 连接 (nhooyr.io/websocket)
	myID     string                // 当前用户 unb
	deviceID string                // 设备 ID
	ctx      context.Context       // 控制上下文
	cancel   context.CancelFunc    // 取消函数

	mu         sync.RWMutex   // 保护 msgHandler 和 connected
	msgHandler MessageHandler // 消息处理回调
	connected  bool           // WebSocket 连接是否存活
	logger     *zap.Logger    // 日志

	writeMu sync.Mutex // nhooyr.io/websocket 不支持并发写

	recvErrCh chan error // recvLoop 错误传播

	// 请求-响应配对（与 Python 版 _pending 对齐）
	pendingMu sync.Mutex
	pending   map[string]chan map[string]any // mid -> response channel
}

// parseURL 解析 URL 字符串。
func parseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

// New 创建 WebSocket 客户端。
func New(cookies map[string]string, deviceID string) (*XianyuWS, error) {
	ctx, cancel := context.WithCancel(context.Background())
	api, err := apis.New(cookies, deviceID)
	if err != nil {
		cancel()
		return nil, err
	}

	return &XianyuWS{
		api:       api,
		myID:      cookies["unb"],
		deviceID:  api.DeviceID(),
		ctx:       ctx,
		cancel:    cancel,
		logger:    zap.L(),
		recvErrCh: make(chan error, 1),
		pending:   make(map[string]chan map[string]any),
	}, nil
}

// NewWithAPI 使用已有的 API 实例创建 WebSocket 客户端（推荐）。
func NewWithAPI(api *apis.XianyuAPI) (*XianyuWS, error) {
	ctx, cancel := context.WithCancel(context.Background())

	var myID string
	for _, domain := range []string{
		"https://www.goofish.com",
		"https://goofish.com",
	} {
		u := parseURL(domain)
		for _, c := range api.Client().Jar.Cookies(u) {
			if c.Name == "unb" {
				myID = c.Value
				break
			}
		}
		if myID != "" {
			break
		}
	}

	if myID == "" {
		cancel()
		return nil, fmt.Errorf("ws: 'unb' cookie not found")
	}

	return &XianyuWS{
		api:       api,
		myID:      myID,
		deviceID:  api.DeviceID(),
		ctx:       ctx,
		cancel:    cancel,
		logger:    zap.L(),
		recvErrCh: make(chan error, 1),
		pending:   make(map[string]chan map[string]any),
	}, nil
}

// SetMessageHandler 设置消息处理回调。
func (ws *XianyuWS) SetMessageHandler(h MessageHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.msgHandler = h
}

// DeviceID 返回设备 ID。
func (ws *XianyuWS) DeviceID() string { return ws.deviceID }

// MyID 返回当前用户 unb ID。
func (ws *XianyuWS) MyID() string { return ws.myID }

// IsAlive 返回 WebSocket 连接是否存活。
//
// 与 Python 版 user_alive() 对齐:
// Python 版 user_alive() 是后台线程，每 600 秒刷新 Token 保持连接存活。
// Go 版本通过 StartTokenRefresher() 实现相同功能（每 10 分钟刷新），
// IsAlive() 提供连接状态查询，可用于判断是否需要重连。
func (ws *XianyuWS) IsAlive() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.connected && ws.ctx.Err() == nil
}

// Stop 停止 WebSocket 客户端。
func (ws *XianyuWS) Stop() {
	// 标记连接已断开
	ws.mu.Lock()
	ws.connected = false
	ws.mu.Unlock()

	// 先关闭 WebSocket 连接，使 recvLoop 中的 Read 立即返回
	if ws.ws != nil {
		ws.ws.Close(websocket.StatusNormalClosure, "client stop")
	}
	// 再取消 context，停止其他 goroutine
	ws.cancel()
}
