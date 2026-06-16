// XianYuApis-GO 调用示例
//
// 核心流程: 登录 → WebSocket 连接 → 监听消息
//
// 运行方式: go run ./cmd/demo/
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/msg"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"github.com/cv-cat/xianyuapis/pkg/ws"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("====================================")
	fmt.Println("  闲鱼 WebSocket Demo")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("选择登录方式:")
	fmt.Println("  1. 扫码登录（自动）")
	fmt.Println("  2. 手动输入 Cookie + Token（推荐，避免风控）")
	fmt.Println()
	fmt.Print("请输入选择 (1/2): ")

	var choice string
	fmt.Scanln(&choice)

	var api *apis.XianyuAPI
	var accessToken string
	var err error

	switch choice {
	case "2":
		api, accessToken, err = manualCookieAndTokenLogin()
	default:
		api, err = qrcodeLogin()
	}

	if err != nil {
		logger.Fatal("登录失败", zap.Error(err))
	}

	logger.Info("登录成功",
		zap.String("deviceID", api.DeviceID()),
		zap.String("cookiePreview", previewCookie(api)),
	)

	// 创建 WebSocket 客户端
	wsClient, err := ws.NewWithAPI(api)
	if err != nil {
		logger.Fatal("创建 WebSocket 客户端失败", zap.Error(err))
	}

	// 设置消息处理回调
	wsClient.SetMessageHandler(func(m *msg.Message) {
		switch {
		case m.IsText():
			fmt.Printf("[文字消息] %s(%s): %s\n", m.SenderName, m.SenderID, m.Content)
			reply := fmt.Sprintf("你好 %s，你说了: %s", m.SenderName, m.Content)
			if err := wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, reply); err != nil {
				logger.Error("发送文字失败", zap.Error(err))
			}
		case m.IsImage():
			fmt.Printf("[图片消息] %s(%s): %s (%dx%d)\n",
				m.SenderName, m.SenderID, m.ImageURL, m.ImageWidth, m.ImageHeight)
			reply := fmt.Sprintf("收到图片: %s", m.ImageURL)
			if err := wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, reply); err != nil {
				logger.Error("发送文字失败", zap.Error(err))
			}
		default:
			fmt.Printf("[%v消息] %s(%s)\n", m.MessageType, m.SenderName, m.SenderID)
		}
	})

	wsClient.StartTokenRefresher()

	// 建立 WebSocket 连接
	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("\n正在连接 WebSocket...")
	if err := wsClient.ConnectWithToken(sigCtx, accessToken); err != nil {
		logger.Fatal("WebSocket 连接失败", zap.Error(err))
	}
	logger.Info("WebSocket 连接成功")

	// 监听消息
	fmt.Println("开始监听消息，按 Ctrl+C 退出")
	if err := wsClient.Start(); err != nil {
		if err != context.Canceled {
			logger.Fatal("WebSocket 错误", zap.Error(err))
		}
	}

	wsClient.Stop()
	logger.Info("程序退出")
}

// qrcodeLogin 扫码登录
func qrcodeLogin() (*apis.XianyuAPI, error) {
	return apis.QrcodeLogin(apis.QrcodeLoginConfig{
		PollInterval: 3 * time.Second,
		Timeout:      120 * time.Second,
		ShowQR:       true,
	})
}

// manualCookieAndTokenLogin 手动输入 Cookie 和 Token 登录
//
// 步骤:
//  1. 浏览器打开 https://www.goofish.com 并登录
//  2. F12 → Console → 粘贴以下代码并回车:
//
//     fetch('https://h5api.m.goofish.com/h5/mtop.taobao.idlemessage.pc.login.token/1.0/?jsv=2.7.2&appKey=34839810&t='+Date.now()+'&sign=&v=1.0&type=originaljson&dataType=json&timeout=20000&api=mtop.taobao.idlemessage.pc.login.token&sessionOption=AutoLoginOnly', {method:'POST', headers:{'content-type':'application/x-www-form-urlencoded'}, body:'data=%7B%22appKey%22%3A%22444e9908a51d1cb236a27862abc769c9%22%2C%22deviceId%22%3A%22test-device-id%22%7D', credentials:'include'}).then(r=>r.json()).then(d=>console.log('TOKEN:', d.data?.accessToken))
//
//  3. 复制输出的 TOKEN 值
//  4. 从 F12 → Network → 请求头中复制完整 Cookie
func manualCookieAndTokenLogin() (*apis.XianyuAPI, string, error) {
	fmt.Println()
	fmt.Println("====================================")
	fmt.Println("  手动 Cookie + Token 登录说明:")
	fmt.Println()

	// 先生成 deviceID，让用户在浏览器脚本中使用同一个
	// 从 Cookie 中提取 unb 来生成 deviceID
	fmt.Print("请输入你的闲鱼用户ID (unb): ")
	var unb string
	fmt.Scanln(&unb)
	if unb == "" {
		return nil, "", fmt.Errorf("unb 不能为空")
	}

	deviceID := util.GenerateDeviceID(unb)
	fmt.Printf("\n  你的 DeviceID: %s\n\n", deviceID)

	fmt.Println("  步骤 1: 获取 Token")
	fmt.Println("  浏览器打开 https://www.goofish.com 并登录")
	fmt.Println("  F12 → Console → 先粘贴第1行加载MD5库，再粘贴第2行获取Token:")
	fmt.Println()
	fmt.Println("  第1行（加载MD5库）:")
	fmt.Println(`  var s=document.createElement('script');s.src='https://cdn.bootcdn.net/ajax/libs/blueimp-md5/2.19.0/js/md5.min.js';document.head.appendChild(s);setTimeout(()=>console.log('MD5库加载完成'),1000)`)
	fmt.Println()
	fmt.Println("  等待输出 'MD5库加载完成' 后，粘贴第2行:")
	fmt.Println()
	fmt.Printf("  (async()=>{let t=Date.now(),tk=document.cookie.match(/_m_h5_tk=([^;]+)/)[1].split('_')[0],d=JSON.stringify({appKey:'444e9908a51d1cb236a27862abc769c9',deviceId:'%s'}),sign=md5(tk+'&'+t+'&34839810&'+d);let r=await fetch('https://h5api.m.goofish.com/h5/mtop.taobao.idlemessage.pc.login.token/1.0/?jsv=2.7.2&appKey=34839810&t='+t+'&sign='+sign+'&v=1.0&type=originaljson&dataType=json&timeout=20000&api=mtop.taobao.idlemessage.pc.login.token&sessionOption=AutoLoginOnly',{method:'POST',headers:{'content-type':'application/x-www-form-urlencoded','origin':'https://www.goofish.com','referer':'https://www.goofish.com/'},body:'data='+encodeURIComponent(d),credentials:'include'});let j=await r.json();console.log('TOKEN:',j.data?.accessToken);console.log('FULL:',JSON.stringify(j))})()\n", deviceID)
	fmt.Println()
	fmt.Println("  复制 TOKEN: 后面的值")
	fmt.Println()
	fmt.Println("  步骤 2: 获取 Cookie")
	fmt.Println("  F12 → Network → 刷新页面 → 点第一个请求")
	fmt.Println("  在 Request Headers 中复制 Cookie 值")
	fmt.Println("====================================")
	fmt.Println()

	fmt.Print("请粘贴 Token: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("读取 Token 失败")
	}
	token := strings.TrimSpace(scanner.Text())

	if token == "" {
		return nil, "", fmt.Errorf("Token 不能为空")
	}

	fmt.Print("请粘贴 Cookie 字符串: ")
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("读取 Cookie 失败")
	}
	cookieStr := strings.TrimSpace(scanner.Text())

	if cookieStr == "" {
		return nil, "", fmt.Errorf("Cookie 不能为空")
	}

	cookies := util.ParseCookies(cookieStr)
	if _, ok := cookies["unb"]; !ok {
		return nil, "", fmt.Errorf("Cookie 中缺少 unb 字段，请确保已登录")
	}

	fmt.Printf("解析到 %d 个 Cookie 字段，unb=%s\n", len(cookies), cookies["unb"])

	api, err := apis.New(cookies, deviceID)
	if err != nil {
		return nil, "", fmt.Errorf("创建 API 实例失败: %w", err)
	}

	return api, token, nil
}

// previewCookie 生成 Cookie 预览字符串
func previewCookie(api *apis.XianyuAPI) string {
	cookieStr := api.CookieString()
	if cookieStr == "" {
		return "(empty)"
	}
	parts := strings.Split(cookieStr, "; ")
	var preview []string
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			v := kv[1]
			if len(v) > 10 {
				v = v[:10] + "..."
			}
			preview = append(preview, kv[0]+"="+v)
		}
	}
	return strings.Join(preview, "; ")
}
