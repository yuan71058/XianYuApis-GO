# XianYuApis-GO

闲鱼 API Go 语言实现，从 [XianYuApis (Python)](https://github.com/cv-cat/XianYuApis) 迁移而来，参考 [xianyu-auto-reply](https://github.com/zhinianboke/xianyu-auto-reply) 修复消息解析。

## 功能

- **扫码登录** — 终端二维码 / URL 两种方式
- **手动 Cookie+Token 登录** — 避免风控，从浏览器手动获取（推荐）
- **WebSocket 实时消息** — 收发文字/图片，心跳保活，Token 自动刷新
- **多类型消息解析** — 文字/图片/商品卡片/转账/位置/语音，完整支持闲鱼消息协议
- **HTTP API** — 商品发布、详情查询、图片上传、Token 管理、自动确认发货
- **消息加解密** — Base64 + MessagePack 协议，自定义 msgpack 解码器支持整数键转字符串键

## 项目结构

```
xianyuapis/
├── cmd/demo/main.go        # 详细调用示例（含完整注释）
├── config/                  # YAML 配置 + 环境变量
├── pkg/
│   ├── apis/                # HTTP API 封装（登录、商品、上传、确认发货）
│   │   ├── api.go           #   XianyuAPI 核心 + mtop 请求封装
│   │   ├── login.go         #   GetToken / RefreshToken（含风控检测）
│   │   ├── qrcode.go        #   扫码登录完整流程
│   │   ├── cookies.go       #   BuildInitialCookies
│   │   ├── upload.go        #   UploadMedia 图片上传
│   │   └── product.go       #   GetItemInfo / PublishItem / ConfirmShipping
│   ├── ws/                  # WebSocket 通信（连接、收发、心跳）
│   │   ├── client.go        #   XianyuWS 结构体 + 构造函数
│   │   ├── connect.go       #   Connect / ConnectWithToken / init / heartbeat
│   │   ├── receiver.go      #   Start / handleMessage / ACK / 消息解析
│   │   ├── sender.go        #   SendText / SendImage
│   │   └── conversation.go  #   ListAllConversations / CreateChat
│   ├── msg/                 # 消息类型定义
│   │   ├── types.go         #   Message 结构体 + MessageType 枚举
│   │   └── factory.go       #   NewTextMessage / NewImageMessage
│   ├── model/               # 数据模型
│   │   ├── product.go       #   Price / DeliverySettings / ImageInfo
│   │   └── login.go         #   QrcodeLoginConfig / QRCodeData
│   └── util/                # 签名、解密、ID 生成、Cookie 工具
│       ├── sign.go          #   GenerateSign（MD5 签名）
│       ├── decrypt.go       #   Decrypt（自定义 msgpack 解码器，整数键→字符串键）
│       ├── gen.go           #   GenerateMid / UUID / DeviceID
│       ├── cookie.go        #   Cookie 解析/构建/Jar 操作
│       ├── tfstk.go         #   GenerateTFstk（Node.js 子进程）
│       └── version.go       #   版本常量
├── internal/
│   ├── lwp/                 # LWP 协议编解码
│   └── httpclient/          # HTTP 客户端工厂 + 中间件
├── assets/                  # JS 脚本（tfstk 生成）
├── Dockerfile
└── Makefile
```

## 快速开始

### 前置条件

- Go 1.22+

### 安装

```bash
git clone https://github.com/yuan71058/XianYuApis-GO.git
cd XianYuApis-GO/xianyuapis
go mod tidy
```

### 运行

```bash
go build -o bin/demo.exe ./cmd/demo/
./bin/demo.exe
```

或直接运行：

```bash
go run ./cmd/demo/
```

## 详细 API 调用指南

> 完整可运行代码见 [cmd/demo/main.go](cmd/demo/main.go)，每个函数均有详细中文注释。

### 1. 登录

#### 方式一：手动 Cookie + Token 登录（推荐，避免风控）

1. 浏览器打开 `https://www.goofish.com` 并登录
2. F12 → Console → 先加载 MD5 库，再获取 Token
3. F12 → Network → 复制完整 Cookie

```go
api, accessToken, err := manualCookieAndTokenLogin()
// 内部流程: 输入unb → 生成deviceID → 浏览器获取Token → 输入Cookie
```

#### 方式二：扫码登录

自动构建初始 Cookie → 生成二维码 → 轮询扫码状态 → 完成登录。

```go
api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
    PollInterval: 3 * time.Second,  // 轮询间隔
    Timeout:      120 * time.Second, // 超时时间
    ShowQR:       true,              // 终端打印二维码
})
```

#### 方式三：已有 Cookie 字典

```go
cookies := map[string]string{
    "unb":       "123456789",
    "tracknick": "我的昵称",
    "_m_h5_tk":  "abcdef_1718000000000",
    "cookie2":   "xxxxxxxxxxxxxxxx",  // HttpOnly，必须从Network请求头复制
    "sgcookie":  "E100xxx...",         // HttpOnly，必须从Network请求头复制
}
api, err := apis.New(cookies, "")
```

### 2. HTTP API

#### 2.1 Token 管理

```go
ctx := context.Background()

// 获取 WebSocket 连接所需的 accessToken
token, err := api.GetToken(ctx)

// 刷新登录态（WebSocket 客户端已内置自动刷新）
err = api.RefreshToken(ctx)
```

#### 2.2 商品查询

```go
itemInfo, err := api.GetItemInfo(ctx, "891198795482")
```

#### 2.3 图片上传

```go
result, err := api.UploadMedia(ctx, "./photo.png")
fmt.Println(result.URL, result.Width, result.Height)
```

#### 2.4 商品发布

```go
result, err := api.PublishItem(ctx,
    []string{"./product1.jpg", "./product2.jpg"},
    "九成新机械键盘 自用半年 功能完好",
    &model.Price{CurrentPrice: 299.0, OriginalPrice: 599.0},
    model.DeliverySettings{Choice: "包邮"},
)
```

#### 2.5 自动确认发货

```go
// 调用 mtop.taobao.idle.logistic.consign.dummy 确认订单发货
// 适用于虚拟商品自动发货场景
result, err := api.ConfirmShipping(ctx, "订单ID")
// 成功时 ret 包含 "SUCCESS::调用成功"
```

### 3. WebSocket 实时通信

#### 3.1 创建客户端并连接

```go
// 从 API 客户端创建 WebSocket 客户端（共享登录态）
wsClient, err := ws.NewWithAPI(api)

// 设置消息处理回调
wsClient.SetMessageHandler(func(m *msg.Message) {
    switch {
    case m.IsText():
        fmt.Printf("[文字] %s(%s): %s\n", m.SenderName, m.SenderID, m.Content)
    case m.IsImage():
        fmt.Printf("[图片] %s(%s): %s (%dx%d)\n",
            m.SenderName, m.SenderID, m.ImageURL, m.ImageWidth, m.ImageHeight)
    }
})

// 启动 Token 自动刷新（默认 10 分钟，可自定义间隔）
wsClient.StartTokenRefresher()
wsClient.StartTokenRefresher(5 * time.Minute) // 自定义 5 分钟

// 建立连接（支持手动传入 Token）
wsClient.ConnectWithToken(ctx, accessToken)

// 启动消息接收循环（阻塞）
wsClient.Start()
```

#### 3.2 发送消息

```go
// 发送文字消息
wsClient.SendText(ctx, "conversation_id", "receiver_id", "你好")

// 发送图片消息（需先上传获取 URL）
uploadResult, _ := api.UploadMedia(ctx, "./photo.png")
wsClient.SendImage(ctx, "conversation_id", "receiver_id",
    uploadResult.URL, uploadResult.Width, uploadResult.Height)
```

#### 3.3 优雅退出

```go
sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
defer wsClient.Stop()
```

### 4. 消息类型

闲鱼 WebSocket 推送消息支持以下类型:

| contentType | 类型 | 解析内容 |
|:-----------:|------|---------|
| 1 | 文字消息 | `text.text` 字段 |
| 2 | 图片消息 | `image.pics[].url` + 宽高 |
| 3 | 语音消息 | `[语音消息]` |
| 7 | 商品卡片 | `[我想要] 标题 价格 (id:xxx)` |
| 17 | 转账消息 | `[转账] ¥金额 (交易号:xxx)` |
| 30 | 位置消息 | `[位置] 经度:xxx 纬度:xxx` |

消息内容有两种编码格式:
- **格式1**: `msg6_3["1"]` 为 base64 字符串（文字消息常用）
- **格式2**: `msg6_3["5"]` 为直接 JSON 字符串（图片/卡片/转账/位置等常用）

### 5. Message 结构体

```go
type Message struct {
    SenderID       string      // 发送者用户 ID
    SenderName     string      // 发送者昵称
    Content        string      // 消息文本内容
    MessageType    MessageType // 消息类型 (1=文字, 2=图片, 26=音频)
    ConversationID string      // 会话 ID
    ImageURL       string      // 图片 URL（仅图片消息）
    ImageWidth     int         // 图片宽度
    ImageHeight    int         // 图片高度
    Timestamp      time.Time   // 消息时间
    Raw            any         // 原始数据（调试用）
}
```

### 6. 工具函数

#### 6.1 签名生成

```go
// 闲鱼 mtop API 签名: MD5(token + "&" + timestamp + "&" + appKey + "&" + data)
sign := util.GenerateSign(timestamp, token, data)
```

#### 6.2 消息解密

```go
// 自定义 msgpack 解码器，自动将整数键转为字符串键（与 Python 版对齐）
decrypted, err := util.Decrypt(encryptedData)
```

#### 6.3 ID 生成

```go
mid      := util.GenerateMid()              // 消息 ID
uuid     := util.GenerateUUID()             // 请求标识
deviceID := util.GenerateDeviceID("unb123") // 设备 ID
```

## 关键技术说明

### 风控规避

- 扫码登录可能触发风控 `FAIL_SYS_USER_VALIDATE`，推荐使用手动 Cookie+Token 方式
- `document.cookie` 无法获取 HttpOnly 的 `cookie2`、`sgcookie`，必须从 F12→Network→请求头中复制
- 多次失败后需等待 5-10 分钟再重试

### Token 与 DeviceID 配对

浏览器获取 Token 时使用的 `deviceId` 必须与 Go 的 `/reg` 请求中使用的 `deviceId` 一致，否则返回 401 `device id or appkey is not equal`。Demo 中已自动处理此配对。

### WebSocket 连接注意事项

- 必须禁用压缩（`CompressionDisabled`），服务端不协商 `permessage-deflate`
- `/reg` 注册后需等待 3 秒冷却，避免 IM 流控错误 400600001
- Cookie 必须包含 HttpOnly 字段（`cookie2`、`sgcookie`），否则 mtop API 认证失败
- Ctrl+C 退出时先关闭 WebSocket 连接再取消 context

## 与 Python 版本的对应关系

| Python | Go |
|--------|-----|
| `goofish_apis.py` | `pkg/apis/` |
| `goofish_live.py` | `pkg/ws/` |
| `utils/goofish_utils.py` | `pkg/util/` |
| `utils/build_cookies.py` | `pkg/apis/cookies.go` |
| `message/types.py` | `pkg/msg/` |
| `push_message_parser.py` | `pkg/ws/receiver.go` |
| `confirm_service.py` | `pkg/apis/product.go` (ConfirmShipping) |

## 依赖

| 库 | 用途 |
|----|------|
| [nhooyr.io/websocket](https://github.com/coder/websocket) | WebSocket 客户端 |
| [skip2/go-qrcode](https://github.com/skip2/go-qrcode) | 终端二维码打印 |
| [go.uber.org/zap](https://go.uber.org/zap) | 结构化日志 |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML 配置解析 |

## License

MIT
