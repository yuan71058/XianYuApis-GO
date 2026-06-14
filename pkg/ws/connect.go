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
)

// Connect 建立 WebSocket 连接并完成初始化。
//
// 流程:
//  1. 建立 WebSocket 连接
//  2. 调用 api.GetToken() 获取 accessToken
//  3. 发送 /reg 注册消息（携带 token）
//  4. 发送 /r/SyncStatus/ackDiff 同步状态
//  5. 启动心跳 goroutine
func (ws *XianyuWS) Connect(ctx context.Context) error {
	hdr := http.Header{}
	hdr.Set("Host", "wss-goofish.dingtalk.com")
	hdr.Set("Connection", "Upgrade")
	hdr.Set("Pragma", "no-cache")
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	hdr.Set("Origin", "https://www.goofish.com")
	hdr.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	hdr.Set("Accept-Language", "zh-CN,zh;q=0.9")

	conn, _, err := websocket.Dial(ctx, wsBaseURL, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		return fmt.Errorf("ws: dial: %w", err)
	}
	ws.ws = conn

	// 初始化
	if err := ws.init(ctx); err != nil {
		conn.Close(websocket.StatusInternalError, "init failed")
		return fmt.Errorf("ws: init: %w", err)
	}

	// 启动心跳（后台 goroutine）
	go ws.heartbeat()

	ws.logger.Info("websocket connected")
	return nil
}

// init 完成 WebSocket 注册和状态同步。
func (ws *XianyuWS) init(ctx context.Context) error {
	// 1. 获取 Token
	token, err := ws.api.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("init: get token: %w", err)
	}

	// 2. 发送 /reg 注册消息
	regMsg := buildLWPMessage("/reg", nil, map[string]any{
		"cache-header": "app-key token ua wv",
		"app-key":      "444e9908a51d1cb236a27862abc769c9",
		"token":        token,
		"ua":           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 DingTalk(2.1.5) OS(Windows/10) Browser(Chrome/133.0.0.0) DingWeb/2.1.5 IMPaaS DingWeb/2.1.5",
		"dt":           "j",
		"wv":           "im:3,au:3,sy:6",
		"sync":         "0,0;0;0;",
		"did":          ws.deviceID,
		"mid":          util.GenerateMid(),
	})
	if err := ws.sendLWP(ctx, regMsg); err != nil {
		return fmt.Errorf("init: send reg: %w", err)
	}

	// 3. 发送 /r/SyncStatus/ackDiff 同步状态
	now := time.Now().UnixMilli()
	syncMsg := buildLWPMessage("/r/SyncStatus/ackDiff", nil, []map[string]any{
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
	})
	if err := ws.sendLWP(ctx, syncMsg); err != nil {
		return fmt.Errorf("init: send sync: %w", err)
	}

	ws.logger.Info("websocket initialized")
	return nil
}

// heartbeat 每 15 秒发送 LWP 心跳包。
func (ws *XianyuWS) heartbeat() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			msg := buildLWPMessage("/!", map[string]string{
				"mid": util.GenerateMid(),
			}, nil)
			if err := ws.sendLWP(ws.ctx, msg); err != nil {
				ws.logger.Warn("heartbeat send failed", zap.Error(err))
			}
		case <-ws.ctx.Done():
			ws.logger.Info("heartbeat stopped")
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
					ws.logger.Error("refresh token failed", zap.Error(err))
				} else {
					ws.logger.Debug("token refreshed")
				}
			case <-ws.ctx.Done():
				return
			}
		}
	}()
}

// buildLWPMessage 构建 LWP 协议消息体。
func buildLWPMessage(lwp string, headers any, body any) map[string]any {
	msg := map[string]any{"lwp": lwp}
	if headers != nil {
		msg["headers"] = headers
	}
	if body != nil {
		msg["body"] = body
	}
	return msg
}

// sendLWP 将 LWP 消息序列化为 JSON 并发送到 WebSocket。
func (ws *XianyuWS) sendLWP(ctx context.Context, msg map[string]any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("sendLWP: marshal: %w", err)
	}
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
