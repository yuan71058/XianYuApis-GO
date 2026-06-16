package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

const (
	wsBaseURL            = "wss://wss-goofish.dingtalk.com/"
	heartbeatInterval    = 15 * time.Second
	tokenRefreshInterval = 10 * time.Minute
	regTimeout           = 5 * time.Second  // /reg 响应超时
	regCooldown          = 3 * time.Second   // 注册后冷却
	wsPingInterval       = 30 * time.Second  // WebSocket PING 间隔（与 Python 版 heartbeat=30 对齐）
)

// Connect 建立 WebSocket 连接并完成初始化。
//
// 与 Python 版 im_client.py connect() 完全对齐:
//  1. 获取 Token
//  2. 建立 WebSocket 连接（启用压缩，与 Python aiohttp 对齐）
//  3. 启动 recvLoop + PING goroutine（先于 /reg）
//  4. 发送 /reg 并等待响应（send-and-wait 模式）
//  5. 发送 /r/SyncStatus/ackDiff
//  6. 冷却 3 秒
//  7. 启动心跳 goroutine
func (ws *XianyuWS) Connect(ctx context.Context) error {
	return ws.ConnectWithToken(ctx, "")
}

// ConnectWithToken 使用指定的 accessToken 建立 WebSocket 连接。
// 如果 token 为空，则自动调用 api.GetToken() 获取。
func (ws *XianyuWS) ConnectWithToken(ctx context.Context, token string) error {
	// 1. 获取 Token
	if token == "" {
		var err error
		token, err = ws.api.GetToken(ctx)
		if err != nil {
			return fmt.Errorf("ws: get token: %w", err)
		}
	}
	ws.logger.Info("ws: token obtained", zap.Int("tokenLen", len(token)), zap.String("token", token))

	// 2. 建立 WebSocket 连接
	cookieStr := ws.api.CookieString()
	hdr := http.Header{}
	hdr.Set("Cookie", cookieStr)
	hdr.Set("Host", "wss-goofish.dingtalk.com")
	hdr.Set("Connection", "Upgrade")
	hdr.Set("Pragma", "no-cache")
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	hdr.Set("Origin", "https://www.goofish.com")
	hdr.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	hdr.Set("Accept-Language", "zh-CN,zh;q=0.9")

	ws.logger.Info("ws: dialing...",
		zap.String("url", wsBaseURL),
		zap.Int("cookieLen", len(cookieStr)),
		zap.String("deviceID", ws.deviceID),
		zap.String("myID", ws.myID),
	)

	conn, resp, err := websocket.Dial(ctx, wsBaseURL, &websocket.DialOptions{
		HTTPHeader: hdr,
		// 禁用压缩: Python aiohttp ws_connect 默认不发送 Sec-WebSocket-Extensions
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		if resp != nil {
			ws.logger.Error("ws: dial failed",
				zap.Int("status", resp.StatusCode),
				zap.String("statusText", resp.Status),
			)
		}
		return fmt.Errorf("ws: dial: %w", err)
	}
	ws.ws = conn

	if resp != nil {
		ws.logger.Info("ws: upgrade response",
			zap.Int("status", resp.StatusCode),
			zap.String("proto", resp.Proto),
		)
		if ext := resp.Header.Get("Sec-WebSocket-Extensions"); ext != "" {
			ws.logger.Info("ws: negotiated extensions", zap.String("ext", ext))
		}
	}

	// 3. 启动 recvLoop（先于 /reg，与 Python 版 asyncio.create_task(self._recv_loop()) 对齐）
	go ws.recvLoop()

	// 启动 WebSocket PING goroutine（与 Python 版 aiohttp heartbeat=30 对齐）
	go ws.pingLoop()

	// 4. 发送 /reg 并等待响应（与 Python 版 await self._send_and_wait(reg_mid, reg_msg, timeout=5) 对齐）
	regMid := util.GenerateMid()
	regMsg := map[string]any{
		"lwp": "/reg",
		"headers": map[string]string{
			"cache-header": "app-key token ua wv",
			"app-key":      "444e9908a51d1cb236a27862abc769c9",
			"token":        token,
			"ua":           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 DingTalk(2.1.5) OS(Windows/10) Browser(Chrome/146.0.0.0) DingWeb/2.1.5 IMPaaS DingWeb/2.1.5",
			"dt":           "j",
			"wv":           "im:3,au:3,sy:6",
			"sync":         "0,0;0;0;",
			"did":          ws.deviceID,
			"mid":          regMid,
		},
	}

	ws.logger.Info("ws: sending /reg...", zap.String("mid", regMid))
	regResp, err := ws.sendAndWait(ctx, regMid, regMsg, regTimeout)
	if err != nil {
		// Python 版: /reg 超时不 Fatal，继续尝试发送 ackDiff
		ws.logger.Warn("ws: /reg wait failed (continuing anyway)", zap.Error(err))
	} else {
		regCode, _ := regResp["code"].(float64)
		ws.logger.Info("ws: /reg response", zap.Float64("code", regCode))
	}

	// 5. 发送 /r/SyncStatus/ackDiff（与 Python 版 _send_raw(ack_msg) 对齐）
	now := time.Now().UnixMilli()
	ackMid := util.GenerateMid()
	syncMsg := map[string]any{
		"lwp":     "/r/SyncStatus/ackDiff",
		"headers": map[string]string{"mid": ackMid},
		"body": []map[string]any{
			{
				"pipeline":    "sync",
				"tooLong2Tag": "PNM,1",
				"channel":     "sync",
				"topic":       "sync",
				"highPts":     0,
				"pts":         now * 1000,
				"seq":         0,
				"timestamp":   now,
			},
		},
	}
	if err := ws.sendLWP(ctx, syncMsg); err != nil {
		conn.Close(websocket.StatusInternalError, "sync failed")
		return fmt.Errorf("ws: sync: %w", err)
	}

	// 6. 冷却 3 秒（与 Python 版 await asyncio.sleep(3) 对齐）
	time.Sleep(regCooldown)

	// 7. 启动心跳 goroutine（与 Python 版 asyncio.create_task(self._heartbeat_loop()) 对齐）
	go ws.heartbeat()

	ws.logger.Info("ws: connected and initialized")
	fmt.Println("[WS] 连接初始化完成，开始监听消息")

	// 标记连接存活
	ws.mu.Lock()
	ws.connected = true
	ws.mu.Unlock()

	return nil
}

// sendAndWait 发送 LWP 消息并等待匹配 mid 的响应。
//
// 与 Python 版 _send_and_wait(mid, msg, timeout) 完全对齐。
func (ws *XianyuWS) sendAndWait(ctx context.Context, mid string, msg map[string]any, timeout time.Duration) (map[string]any, error) {
	ch := make(chan map[string]any, 1)

	ws.pendingMu.Lock()
	ws.pending[mid] = ch
	ws.pendingMu.Unlock()

	defer func() {
		ws.pendingMu.Lock()
		delete(ws.pending, mid)
		ws.pendingMu.Unlock()
	}()

	if err := ws.sendLWP(ctx, msg); err != nil {
		return nil, fmt.Errorf("sendAndWait: send: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("sendAndWait: timeout (%v) for %s", timeout, msg["lwp"])
	case <-ctx.Done():
		return nil, fmt.Errorf("sendAndWait: context cancelled: %w", ctx.Err())
	}
}

// pingLoop 定期发送 WebSocket PING 帧。
//
// 与 Python 版 aiohttp heartbeat=30 对齐:
// aiohttp 的 ws_connect(heartbeat=30) 会每 30 秒发送 WebSocket PING 帧。
// nhooyr.io/websocket 的 Ping() 方法发送 PING 并等待 PONG。
func (ws *XianyuWS) pingLoop() {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ws.ctx, 10*time.Second)
			err := ws.ws.Ping(pingCtx)
			cancel()
			if err != nil {
				ws.logger.Warn("ws: ping failed", zap.Error(err))
				return
			}
		case <-ws.ctx.Done():
			return
		}
	}
}

// heartbeat 每 15 秒发送 LWP 心跳包。
//
// 与 Python 版 _heartbeat_loop() 对齐。
func (ws *XianyuWS) heartbeat() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			msg := map[string]any{
				"lwp":     "/!",
				"headers": map[string]string{"mid": util.GenerateMid()},
			}
			if err := ws.sendLWP(ws.ctx, msg); err != nil {
				ws.logger.Warn("ws: heartbeat send failed", zap.Error(err))
			}
		case <-ws.ctx.Done():
			return
		}
	}
}

// StartTokenRefresher 启动后台 Token 刷新 goroutine。
func (ws *XianyuWS) StartTokenRefresher() {
	go func() {
		ticker := time.NewTicker(tokenRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := ws.api.RefreshToken(ws.ctx); err != nil {
					ws.logger.Error("ws: refresh token failed", zap.Error(err))
				}
			case <-ws.ctx.Done():
				return
			}
		}
	}()
}

// sendLWP 将 LWP 消息序列化为 JSON 并发送到 WebSocket。
func (ws *XianyuWS) sendLWP(ctx context.Context, msg map[string]any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("sendLWP: marshal: %w", err)
	}
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()
	return ws.ws.Write(ctx, websocket.MessageText, data)
}

// mustMarshal 将任意值序列化为 JSON bytes，失败时 panic。
func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
