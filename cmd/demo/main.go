package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/msg"
	"github.com/cv-cat/xianyuapis/pkg/ws"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 方式一: 扫码登录（自动获取 Cookie）
	api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
		PollInterval: 3 * time.Second,
		Timeout:      120 * time.Second,
		ShowQR:       true,
	})
	if err != nil {
		logger.Fatal("qrcode login failed", zap.Error(err))
	}

	// 方式二: 已有 Cookie 字典
	// cookies := map[string]string{"unb": "xxx", "tracknick": "yyy", ...}
	// api, err := apis.New(cookies, "")

	// 提取 cookies 用于创建 WebSocket 客户端
	cookies := make(map[string]string)
	if jar, ok := api.Client().Jar.(interface {
		Cookies(u *url.URL) []*http.Cookie
	}); ok {
		for _, domain := range []string{".goofish.com"} {
			u, _ := url.Parse("https://" + domain)
			for _, c := range jar.Cookies(u) {
				cookies[c.Name] = c.Value
			}
		}
	}

	// 创建 WebSocket 客户端
	client, err := ws.New(cookies, api.DeviceID())
	if err != nil {
		logger.Fatal("create ws client", zap.Error(err))
	}

	// 设置消息处理器
	client.SetMessageHandler(func(m *msg.Message) {
		fmt.Printf("[%s] %s: %s\n", m.SenderName, m.SenderID, m.Content)

		// === AI 接入点 ===
		// reply := callAI(m.Content)
		// client.SendText(context.Background(), m.ConversationID, m.SenderID, reply)
		// =================

		// 默认 echo 回复
		client.SendText(context.Background(), m.ConversationID, m.SenderID,
			fmt.Sprintf("%s 说了: %s", m.SenderName, m.Content))
	})

	// 启动 Token 刷新
	client.StartTokenRefresher()

	// 建立连接
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		logger.Fatal("connect failed", zap.Error(err))
	}

	logger.Info("xianyuapis started, Ctrl+C to stop")

	// 启动消息接收（阻塞）
	if err := client.Start(); err != nil {
		if err != context.Canceled {
			logger.Fatal("ws error", zap.Error(err))
		}
	}

	logger.Info("shutting down")
}
