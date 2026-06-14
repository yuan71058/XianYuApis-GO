// XianYuApis-GO 详细调用示例
//
// 本文件覆盖所有核心功能的调用方式，包括：
//   1. 登录（扫码 / Cookie）
//   2. HTTP API（Token 管理、商品查询、商品发布、图片上传）
//   3. WebSocket 实时通信（连接、收发消息、心跳、历史记录）
//   4. 工具函数（签名、解密、ID 生成、Cookie 解析）
//   5. 配置管理
//
// 运行方式: go run ./cmd/demo/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/msg"
	"github.com/cv-cat/xianyuapis/pkg/model"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"github.com/cv-cat/xianyuapis/pkg/ws"
	"go.uber.org/zap"
)

func main() {
	// ================================================================
	// 0. 初始化日志
	// ================================================================
	// zap 是高性能结构化日志库，NewProduction 输出 JSON 格式
	// 开发阶段可用 NewDevelopment() 输出更友好的控制台格式
	logger, _ := zap.NewProduction()
	defer logger.Sync() // 程序退出前刷新缓冲区

	// ================================================================
	// 1. 登录 — 两种方式任选其一
	// ================================================================

	// ---- 方式一: 扫码登录（推荐，全自动获取 Cookie） ----
	//
	// 流程: 构建初始 Cookie → 访问 passport 获取 XSRF-TOKEN
	//       → 生成二维码 → 终端打印 → 轮询扫码状态 → 完成登录
	//
	// QrcodeLoginConfig 参数说明:
	//   PollInterval: 轮询扫码状态的间隔，默认 3 秒
	//   Timeout:      扫码超时时间，默认 120 秒
	//   ShowQR:       是否在终端用 Unicode 块字符打印二维码
	//                 设为 false 则仅打印 URL，需手动打开
	api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
		PollInterval: 3 * time.Second,  // 每 3 秒查询一次扫码状态
		Timeout:      120 * time.Second, // 2 分钟未扫码则超时
		ShowQR:       true,              // 在终端打印二维码
	})
	if err != nil {
		logger.Fatal("扫码登录失败", zap.Error(err))
	}
	logger.Info("扫码登录成功", zap.String("deviceID", api.DeviceID()))

	// ---- 方式二: 已有 Cookie 字典（适合从浏览器手动复制） ----
	//
	// Cookie 获取方法:
	//   1. 浏览器打开 https://www.goofish.com 并登录
	//   2. F12 → Application → Cookies → .goofish.com
	//   3. 复制以下关键字段: unb, tracknick, _m_h5_tk, cookie2, cna 等
	//
	// 注意: cookies 中必须包含 "unb" 字段（用户 ID），否则 New() 会报错
	//
	// cookies := map[string]string{
	//     "unb":       "123456789",                    // 用户 ID（必填）
	//     "tracknick": "我的昵称",                       // 用户昵称
	//     "_m_h5_tk":  "abcdef1234_1718000000000",     // mtop 签名 token
	//     "cookie2":   "xxxxxxxxxxxxxxxx",             // 会话标识
	//     "cna":       "yyyyyyyyyyyyyyyy",             // 设备标识
	//     // ... 其他 .goofish.com 域下的 Cookie
	// }
	// api, err := apis.New(cookies, "")
	// if err != nil {
	//     logger.Fatal("创建 API 实例失败", zap.Error(err))
	// }

	// ---- 方式三: 从 Cookie 字符串解析（浏览器 DevTools 复制） ----
	//
	// 浏览器中复制的 Cookie 格式通常是: "key1=val1; key2=val2; key3=val3"
	// 可用 util.ParseCookies() 解析为 map
	//
	// cookieStr := "unb=123456789; tracknick=我的昵称; _m_h5_tk=abcdef_1718000000; cookie2=xxx; cna=yyy"
	// cookies := util.ParseCookies(cookieStr)
	// api, err := apis.New(cookies, "")

	// ================================================================
	// 2. HTTP API 调用示例
	// ================================================================

	ctx := context.Background()

	// ---- 2.1 获取 WebSocket Token ----
	//
	// GetToken 调用 mtop.taobao.idlemessage.pc.login.token 接口
	// 返回的 accessToken 用于 WebSocket /reg 注册
	// 内部自动处理"令牌过期"重试（最多 3 次）
	token, err := api.GetToken(ctx)
	if err != nil {
		logger.Error("获取 Token 失败", zap.Error(err))
	} else {
		logger.Info("获取 Token 成功", zap.String("token", token))
	}

	// ---- 2.2 刷新登录态 ----
	//
	// RefreshToken 调用 mtop.taobao.idlemessage.pc.loginuser.get 接口
	// 用于维持长期运行的 WebSocket 连接不掉线
	// 建议每 10 分钟调用一次（WebSocket 客户端已内置自动刷新）
	if err := api.RefreshToken(ctx); err != nil {
		logger.Error("刷新 Token 失败", zap.Error(err))
	}

	// ---- 2.3 查询商品详情 ----
	//
	// GetItemInfo 调用 mtop.taobao.idle.pc.detail 接口
	// itemID 可从闲鱼商品页面 URL 中获取
	// 例如: https://www.goofish.com/item?id=891198795482 中的 891198795482
	itemInfo, err := api.GetItemInfo(ctx, "891198795482")
	if err != nil {
		logger.Error("查询商品失败", zap.Error(err))
	} else {
		// itemInfo 是 map[string]any 类型，包含完整的商品详情 JSON
		data, _ := json.MarshalIndent(itemInfo, "", "  ")
		logger.Info("商品详情", zap.String("data", string(data)))
	}

	// ---- 2.4 上传图片 ----
	//
	// UploadMedia 将本地图片上传到闲鱼 CDN
	// 支持 PNG/JPG/JPEG/GIF 格式
	// 返回的 URL 可用于发送图片消息或发布商品
	uploadResult, err := api.UploadMedia(ctx, "./test_image.png")
	if err != nil {
		logger.Error("上传图片失败", zap.Error(err))
	} else {
		logger.Info("上传图片成功",
			zap.String("url", uploadResult.URL),
			zap.Int("width", uploadResult.Width),
			zap.Int("height", uploadResult.Height),
		)
	}

	// ---- 2.5 发布商品 ----
	//
	// PublishItem 是完整的商品发布流程，内部自动执行:
	//   Step 1: 逐张上传图片 → 获取 URL 和尺寸
	//   Step 2: 获取推荐标签和分类建议 → 自动填充商品属性
	//   Step 3: 获取默认地理位置 → 填写发货地址
	//   Step 4: 构建发布请求体 → 提交发布
	//
	// 参数说明:
	//   images: 本地图片路径列表（最多 9 张）
	//   desc:   商品标题和描述（闲鱼中标题即描述）
	//   price:  价格信息，nil 表示使用系统默认定价
	//   ds:     配送设置
	publishResult, err := api.PublishItem(ctx,
		[]string{"./product1.jpg", "./product2.jpg"}, // 图片路径
		"九成新机械键盘 自用半年 功能完好",               // 商品描述
		&model.Price{                                // 价格
			CurrentPrice:  299.0, // 售价 299 元
			OriginalPrice: 599.0, // 原价 599 元
		},
		model.DeliverySettings{ // 配送设置
			Choice:        "包邮",    // 配送方式: "包邮" | "按距离计费" | "一口价" | "无需邮寄"
			PostPrice:     0,       // 运费（仅 "一口价" 时有效）
			CanSelfPickup: false,   // 是否支持自提
		},
	)
	if err != nil {
		logger.Error("发布商品失败", zap.Error(err))
	} else {
		data, _ := json.MarshalIndent(publishResult, "", "  ")
		logger.Info("发布商品成功", zap.String("result", string(data)))
	}

	// ================================================================
	// 3. WebSocket 实时通信
	// ================================================================

	// ---- 3.1 从 API 客户端提取 Cookie（创建 WS 客户端需要） ----
	//
	// WebSocket 客户端需要与 HTTP API 共享同一套 Cookie
	// 从 api.Client().Jar 中提取 .goofish.com 域下的所有 Cookie
	cookies := extractCookiesFromAPI(api)

	// ---- 3.2 创建 WebSocket 客户端 ----
	//
	// ws.New 内部会创建一个新的 apis.XianyuAPI 实例
	// cookies 必须包含 "unb" 字段
	// deviceID 传空字符串则自动从 unb 生成
	wsClient, err := ws.New(cookies, api.DeviceID())
	if err != nil {
		logger.Fatal("创建 WebSocket 客户端失败", zap.Error(err))
	}

	// ---- 3.3 设置消息处理回调 ----
	//
	// 回调函数在收到用户消息时被调用
	// 注意: 回调应尽快返回，耗时操作请启动 goroutine
	wsClient.SetMessageHandler(func(m *msg.Message) {
		// m 是 *msg.Message 类型，包含以下字段:
		//   SenderID       string      // 发送者用户 ID
		//   SenderName     string      // 发送者昵称
		//   Content        string      // 消息文本内容
		//   MessageType    MessageType // 消息类型 (1=文字, 2=图片, 26=音频)
		//   ConversationID string      // 会话 ID（回复消息时需要）
		//   ImageURL       string      // 图片 URL（仅图片消息）
		//   ImageWidth     int         // 图片宽度
		//   ImageHeight    int         // 图片高度
		//   Timestamp      time.Time   // 消息时间
		//   Raw            any         // 原始数据（调试用）

		switch {
		case m.IsText():
			// 文字消息处理
			fmt.Printf("[文字消息] %s(%s): %s\n", m.SenderName, m.SenderID, m.Content)

			// === AI 接入点 ===
			// 在此处接入你的 AI 服务，示例:
			// reply := callYourAI(m.Content)
			// wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, reply)

			// 默认 echo 回复
			reply := fmt.Sprintf("你好 %s，你说了: %s", m.SenderName, m.Content)
			if err := wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, reply); err != nil {
				logger.Error("发送文字失败", zap.Error(err))
			}

		case m.IsImage():
			// 图片消息处理
			fmt.Printf("[图片消息] %s(%s): %s (%dx%d)\n",
				m.SenderName, m.SenderID, m.ImageURL, m.ImageWidth, m.ImageHeight)

		default:
			// 其他类型消息（如音频）
			fmt.Printf("[%s消息] %s(%s)\n", m.MessageType, m.SenderName, m.SenderID)
		}
	})

	// ---- 3.4 启动 Token 自动刷新 ----
	//
	// 每 10 分钟自动调用 api.RefreshToken()
	// 防止长时间运行导致登录态过期
	// 必须在 Connect() 之前调用
	wsClient.StartTokenRefresher()

	// ---- 3.5 建立 WebSocket 连接 ----
	//
	// Connect 内部流程:
	//   1. 建立 WebSocket 连接到 wss://wss-goofish.dingtalk.com/
	//   2. 调用 api.GetToken() 获取 accessToken
	//   3. 发送 /reg 注册消息（携带 token、设备 ID 等）
	//   4. 发送 /r/SyncStatus/ackDiff 同步状态
	//   5. 启动心跳 goroutine（每 15 秒发送 /! 心跳包）
	//
	// 使用 signal.NotifyContext 监听 Ctrl+C 信号，实现优雅退出
	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := wsClient.Connect(sigCtx); err != nil {
		logger.Fatal("WebSocket 连接失败", zap.Error(err))
	}
	logger.Info("WebSocket 连接成功")

	// ---- 3.6 发送文字消息 ----
	//
	// 参数:
	//   cid:   会话 ID（从收到的消息中获取，不含 @goofish 后缀）
	//   toID:  接收方用户 ID
	//   text:  消息内容
	err = wsClient.SendText(sigCtx, "conversation_id_here", "receiver_user_id", "你好，这是测试消息")
	if err != nil {
		logger.Error("发送文字消息失败", zap.Error(err))
	}

	// ---- 3.7 发送图片消息 ----
	//
	// 图片需先通过 api.UploadMedia() 上传获取 URL
	// 然后用 URL + 宽高发送
	if uploadResult != nil {
		err = wsClient.SendImage(sigCtx,
			"conversation_id_here",          // 会话 ID
			"receiver_user_id",              // 接收方 ID
			uploadResult.URL,                // 图片 URL（上传后获取）
			uploadResult.Width,              // 图片宽度
			uploadResult.Height,             // 图片高度
		)
		if err != nil {
			logger.Error("发送图片消息失败", zap.Error(err))
		}
	}

	// ---- 3.8 创建新会话 ----
	//
	// 与指定用户创建聊天会话，可关联商品 ID
	// itemID 为空时使用默认值
	err = wsClient.CreateChat(sigCtx, "target_user_id", "891198795482")
	if err != nil {
		logger.Error("创建会话失败", zap.Error(err))
	}

	// ---- 3.9 获取历史聊天记录 ----
	//
	// ListAllConversations 会建立一个新的临时 WebSocket 连接
	// 获取完历史消息后自动关闭连接
	// 返回按时间正序排列的消息列表
	history, err := wsClient.ListAllConversations(sigCtx, "conversation_id_here")
	if err != nil {
		logger.Error("获取历史记录失败", zap.Error(err))
	} else {
		for i, h := range history {
			fmt.Printf("[历史%d] %s(%s): %v\n", i+1, h.SenderName, h.SenderID, h.Message)
		}
	}

	// ---- 3.10 启动消息接收循环（阻塞） ----
	//
	// Start() 阻塞运行，持续读取 WebSocket 消息
	// 退出条件: 连接关闭 / context 取消 / 致命错误
	// 收到消息后自动调用 SetMessageHandler 设置的回调函数
	logger.Info("开始监听消息，按 Ctrl+C 退出")
	if err := wsClient.Start(); err != nil {
		if err != context.Canceled {
			logger.Fatal("WebSocket 错误", zap.Error(err))
		}
	}

	// ---- 3.11 优雅停止 ----
	//
	// Stop() 会:
	//   1. 取消内部 context（停止心跳、Token 刷新等 goroutine）
	//   2. 发送 WebSocket 关闭帧
	wsClient.Stop()
	logger.Info("程序退出")
}

// ================================================================
// 辅助函数
// ================================================================

// extractCookiesFromAPI 从 API 客户端的 CookieJar 中提取所有 Cookie。
//
// WebSocket 客户端需要独立的 Cookie 副本，因此需要从 HTTP API 客户端中提取。
func extractCookiesFromAPI(api *apis.XianyuAPI) map[string]string {
	cookies := make(map[string]string)
	jar := api.Client().Jar

	// 遍历 .goofish.com 域下的所有 Cookie
	u, _ := url.Parse("https://.goofish.com")
	for _, c := range jar.Cookies(u) {
		cookies[c.Name] = c.Value
	}

	// 也收集 .mmstat.com 域下的 cna（设备标识）
	u2, _ := url.Parse("https://.mmstat.com")
	for _, c := range jar.Cookies(u2) {
		if c.Name == "cna" {
			cookies["cna"] = c.Value
		}
	}

	return cookies
}

// ================================================================
// 以下为独立功能调用示例（不在 main 中直接运行，仅作参考）
// ================================================================

// demoUtilSign 演示签名生成。
//
// 闲鱼 mtop API 的签名公式:
//
//	sign = MD5(token + "&" + timestamp + "&" + appKey + "&" + data)
//
// 其中:
//   - token:    从 Cookie _m_h5_tk 中提取，下划线前的部分
//   - timestamp: 毫秒级时间戳
//   - appKey:   固定值 "34839810"
//   - data:     请求体 JSON 字符串
func demoUtilSign() {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	token := "abcdef1234567890" // 从 _m_h5_tk Cookie 中提取
	data := `{"itemId":"891198795482"}`

	sign := util.GenerateSign(timestamp, token, data)
	fmt.Printf("签名结果: %s\n", sign)
	// 输出类似: 签名结果: a1b2c3d4e5f6...
}

// demoUtilDecrypt 演示消息解密。
//
// 闲鱼 WebSocket 推送的加密消息格式:
//   Base64(MessagePack(原始数据))
//
// 解密流程:
//   1. 清理非 Base64 字符
//   2. Base64 解码为原始字节
//   3. MessagePack 反序列化为 map
//   4. JSON 序列化为字符串
//
// 解密后的 JSON 结构:
//
//	{
//	  "1": {
//	    "2": "conversation_id@goofish",  // 会话 ID
//	    "10": {
//	      "reminderTitle": "发送者昵称",     // 发送者名称
//	      "senderUserId": "123456789",    // 发送者 ID
//	      "reminderContent": "消息内容"     // 消息文本
//	    }
//	  }
//	}
func demoUtilDecrypt() {
	// 模拟加密消息（实际从 WebSocket 推送中获取）
	encryptedData := "kqC1Nc0+DjK2..."

	decrypted, err := util.Decrypt(encryptedData)
	if err != nil {
		fmt.Printf("解密失败: %v\n", err)
		return
	}
	fmt.Printf("解密结果: %s\n", decrypted)
}

// demoUtilID 演示 ID 生成。
func demoUtilID() {
	// GenerateMid: 生成消息 ID
	// 格式: "{0-999随机数}{毫秒时间戳} 0"
	// 示例: "7381748291023 0"
	mid := util.GenerateMid()
	fmt.Printf("消息 ID: %s\n", mid)

	// GenerateUUID: 生成请求唯一标识
	// 格式: "-{毫秒时间戳}1"
	// 注意: 不是 RFC 4122 UUID，而是闲鱼协议约定格式
	uuid := util.GenerateUUID()
	fmt.Printf("请求 UUID: %s\n", uuid)

	// GenerateDeviceID: 基于用户 ID 生成设备 ID
	// 格式: "RFC4122-v4-UUID-用户unb"
	// 使用 crypto/rand 确保随机性
	deviceID := util.GenerateDeviceID("123456789")
	fmt.Printf("设备 ID: %s\n", deviceID)
}

// demoUtilCookie 演示 Cookie 解析与构建。
func demoUtilCookie() {
	// ParseCookies: 从浏览器复制的 Cookie 字符串解析为 map
	cookieStr := "unb=123456789; tracknick=我的昵称; _m_h5_tk=abcdef_1718000000; cookie2=xxx"
	cookies := util.ParseCookies(cookieStr)
	fmt.Printf("解析结果: %v\n", cookies)
	// 输出: map[unb:123456789 tracknick:我的昵称 _m_h5_tk:abcdef_1718000000 cookie2:xxx]

	// BuildCookieString: 将 map 构建回 Cookie 字符串
	// 注意: map 遍历顺序不确定，输出顺序可能不同
	cookieStr2 := util.BuildCookieString(cookies)
	fmt.Printf("构建结果: %s\n", cookieStr2)

	// 从 Cookie 字符串中提取单个值
	// 例如提取 _m_h5_tk 中下划线前的 token 部分
	h5tk, ok := cookies["_m_h5_tk"]
	if ok {
		parts := strings.SplitN(h5tk, "_", 2)
		token := parts[0]
		fmt.Printf("H5 Token: %s\n", token)
	}
}

// demoUtilTFstk 演示 tfstk 生成。
//
// tfstk 是阿里系追踪 Cookie，通过 Node.js 子进程生成。
// 需要本机安装 Node.js，且 assets/gen_tfstk.js 文件存在。
func demoUtilTFstk() {
	tfstk, err := util.GenerateTFstk("assets/gen_tfstk.js")
	if err != nil {
		fmt.Printf("生成 tfstk 失败: %v\n", err)
		return
	}
	fmt.Printf("tfstk: %s\n", tfstk)
}

// demoConfig 演示配置管理。
//
// 配置优先级: 环境变量 > 配置文件 > 默认值
func demoConfig() {
	// 加载配置文件（如果存在）
	// cfg, err := config.Load("config/example.yaml")
	// if err != nil {
	//     log.Fatal(err)
	// }

	// 环境变量覆盖:
	//   XIANYU_COOKIE_UNB=123456789  → 覆盖 cookies.unb
	//   XIANYU_LOG_LEVEL=debug       → 覆盖 log.level

	// 默认配置:
	//   log.level: info
	//   log.json: true
	//   ws.heartbeat: 15s
	//   ws.token_refresh: 10m
	fmt.Println("配置管理示例 — 参见 config/config.go")
}
