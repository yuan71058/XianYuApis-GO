# XianYuApis-GO

闲鱼 API Go 语言实现，从 [XianYuApis (Python)](https://github.com/cv-cat/XianYuApis) 迁移而来。

## 功能

- **扫码登录** — 终端二维码 / URL 两种方式
- **WebSocket 实时消息** — 收发文字/图片，心跳保活，Token 自动刷新
- **HTTP API** — 商品发布、详情查询、图片上传、Token 管理
- **消息加解密** — Base64 + MessagePack 协议，与原版 JS/Python 完全等价

## 项目结构

```
xianyuapis/
├── cmd/demo/main.go        # 详细调用示例（含完整注释）
├── config/                  # YAML 配置 + 环境变量
├── pkg/
│   ├── apis/                # HTTP API 封装（登录、商品、上传）
│   │   ├── api.go           #   XianyuAPI 核心 + mtop 请求封装
│   │   ├── login.go         #   GetToken / RefreshToken
│   │   ├── qrcode.go        #   扫码登录完整流程
│   │   ├── cookies.go       #   BuildInitialCookies
│   │   ├── upload.go        #   UploadMedia 图片上传
│   │   └── product.go       #   GetItemInfo / PublishItem
│   ├── ws/                  # WebSocket 通信（连接、收发、心跳）
│   │   ├── client.go        #   XianyuWS 结构体 + 构造函数
│   │   ├── connect.go       #   Connect / init / heartbeat
│   │   ├── receiver.go      #   Start / handleMessage / ACK
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
│       ├── decrypt.go       #   Decrypt（Base64 + MessagePack 反序列化）
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
- Node.js（tfstk 生成需要）

### 安装

```bash
git clone https://github.com/yuan71058/XianYuApis-GO.git
cd XianYuApis-GO/xianyuapis
make init
```

### 运行

```bash
# 扫码登录 + WebSocket 消息监听
make build
./bin/xianyuapis
```

或直接运行：

```bash
go run ./cmd/demo/
```

## 详细 API 调用指南

> 完整可运行代码见 [cmd/demo/main.go](cmd/demo/main.go)，每个函数均有详细中文注释。

### 1. 登录

#### 方式一：扫码登录（推荐）

自动构建初始 Cookie → 生成二维码 → 轮询扫码状态 → 完成登录。

```go
api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
    PollInterval: 3 * time.Second,  // 轮询间隔
    Timeout:      120 * time.Second, // 超时时间
    ShowQR:       true,              // 终端打印二维码
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("登录成功，设备ID:", api.DeviceID())
```

#### 方式二：已有 Cookie 字典

从浏览器 F12 → Application → Cookies → .goofish.com 复制关键字段。

```go
cookies := map[string]string{
    "unb":       "123456789",                // 用户 ID（必填）
    "tracknick": "我的昵称",                   // 用户昵称
    "_m_h5_tk":  "abcdef_1718000000000",     // mtop 签名 token
    "cookie2":   "xxxxxxxxxxxxxxxx",         // 会话标识
    "cna":       "yyyyyyyyyyyyyyyy",         // 设备标识
}
api, err := apis.New(cookies, "")  // deviceID 为空则自动生成
```

#### 方式三：从 Cookie 字符串解析

```go
// 浏览器复制的格式: "key1=val1; key2=val2; key3=val3"
cookieStr := "unb=123456789; tracknick=我的昵称; _m_h5_tk=abcdef_1718000000"
cookies := util.ParseCookies(cookieStr)
api, err := apis.New(cookies, "")
```

### 2. HTTP API

#### 2.1 Token 管理

```go
ctx := context.Background()

// 获取 WebSocket 连接所需的 accessToken
// 内部自动处理"令牌过期"重试（最多 3 次）
token, err := api.GetToken(ctx)

// 刷新登录态（建议每 10 分钟调用一次，WebSocket 客户端已内置自动刷新）
err = api.RefreshToken(ctx)
```

#### 2.2 商品查询

```go
// itemID 从闲鱼商品页面 URL 获取
// 例如 https://www.goofish.com/item?id=891198795482
itemInfo, err := api.GetItemInfo(ctx, "891198795482")
// 返回 map[string]any，包含完整商品详情 JSON
```

#### 2.3 图片上传

```go
// 支持 PNG/JPG/JPEG/GIF，返回 CDN URL 和尺寸
result, err := api.UploadMedia(ctx, "./photo.png")
fmt.Println(result.URL, result.Width, result.Height)
```

#### 2.4 商品发布

```go
// PublishItem 内部自动执行: 上传图片 → 获取推荐标签 → 获取默认地址 → 提交发布
result, err := api.PublishItem(ctx,
    []string{"./product1.jpg", "./product2.jpg"},  // 图片路径（最多 9 张）
    "九成新机械键盘 自用半年 功能完好",                // 商品描述
    &model.Price{                                 // 价格，nil 使用默认定价
        CurrentPrice:  299.0,
        OriginalPrice: 599.0,
    },
    model.DeliverySettings{                       // 配送设置
        Choice:        "包邮",    // "包邮" | "按距离计费" | "一口价" | "无需邮寄"
        PostPrice:     0,       // 运费（仅 "一口价" 时有效）
        CanSelfPickup: false,   // 是否支持自提
    },
)
```

### 3. WebSocket 实时通信

#### 3.1 创建客户端并连接

```go
// 从 API 客户端提取 Cookie（WebSocket 需要共享登录态）
cookies := extractCookiesFromAPI(api)

// 创建 WebSocket 客户端
wsClient, err := ws.New(cookies, api.DeviceID())

// 设置消息处理回调
wsClient.SetMessageHandler(func(m *msg.Message) {
    switch {
    case m.IsText():
        fmt.Printf("[文字] %s: %s\n", m.SenderName, m.Content)
    case m.IsImage():
        fmt.Printf("[图片] %s: %s (%dx%d)\n",
            m.SenderName, m.ImageURL, m.ImageWidth, m.ImageHeight)
    }
})

// 启动 Token 自动刷新（每 10 分钟）
wsClient.StartTokenRefresher()

// 建立连接（内部: 连接 → 获取Token → 注册 → 同步状态 → 启动心跳）
wsClient.Connect(ctx)

// 启动消息接收循环（阻塞）
wsClient.Start()
```

#### 3.2 发送消息

```go
// 发送文字消息
// cid: 会话 ID（从收到的消息中获取，不含 @goofish 后缀）
// toID: 接收方用户 ID
wsClient.SendText(ctx, "conversation_id", "receiver_id", "你好")

// 发送图片消息（需先上传获取 URL）
uploadResult, _ := api.UploadMedia(ctx, "./photo.png")
wsClient.SendImage(ctx, "conversation_id", "receiver_id",
    uploadResult.URL, uploadResult.Width, uploadResult.Height)
```

#### 3.3 会话管理

```go
// 创建新会话（可关联商品 ID）
wsClient.CreateChat(ctx, "target_user_id", "891198795482")

// 获取历史聊天记录（建立临时连接，获取完自动关闭）
history, err := wsClient.ListAllConversations(ctx, "conversation_id")
for _, h := range history {
    fmt.Printf("%s(%s): %v\n", h.SenderName, h.SenderID, h.Message)
}
```

#### 3.4 优雅退出

```go
// 监听 Ctrl+C 信号
sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()

// Stop() 取消内部 context + 发送 WebSocket 关闭帧
defer wsClient.Stop()
```

### 4. 工具函数

#### 4.1 签名生成

```go
// 闲鱼 mtop API 签名公式: MD5(token + "&" + timestamp + "&" + appKey + "&" + data)
// appKey 固定值 "34839810"
timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
token := "abcdef1234567890"  // 从 _m_h5_tk Cookie 下划线前提取
data := `{"itemId":"891198795482"}`
sign := util.GenerateSign(timestamp, token, data)
```

#### 4.2 消息解密

```go
// 闲鱼加密消息格式: Base64(MessagePack(原始数据))
// 解密后 JSON 结构: {"1": {"2": "cid@goofish", "10": {"reminderTitle": ..., "senderUserId": ...}}}
decrypted, err := util.Decrypt(encryptedData)
```

#### 4.3 ID 生成

```go
mid      := util.GenerateMid()              // 消息 ID: "7381748291023 0"
uuid     := util.GenerateUUID()             // 请求标识: "-17180000001231"
deviceID := util.GenerateDeviceID("unb123") // 设备 ID: "a1b2c3d4-...-unb123"
```

#### 4.4 Cookie 工具

```go
// 解析 Cookie 字符串为 map
cookies := util.ParseCookies("unb=123; tracknick=昵称; _m_h5_tk=abc_123")

// 构建 Cookie 字符串
str := util.BuildCookieString(cookies)

// 从 CookieJar 读取/写入
val  := util.GetCookieFromJar(jar, ".goofish.com", "unb")
util.SetCookieToJar(jar, ".goofish.com", "cna", "xxx", "/")
```

### 5. Message 结构体字段

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

## 配置

复制 `config/example.yaml` 并修改：

```yaml
cookies: {}
log:
  level: info
  json: true
ws:
  heartbeat: 15s
  token_refresh: 10m
```

环境变量覆盖：

| 变量 | 说明 |
|------|------|
| `XIANYU_COOKIE_UNB` | 用户 unb（覆盖配置文件） |
| `XIANYU_LOG_LEVEL` | 日志级别 |

## Docker

```bash
docker build -t xianyuapis .
docker run -it xianyuapis
```

## 与 Python 版本的对应关系

| Python | Go |
|--------|-----|
| `goofish_apis.py` | `pkg/apis/` |
| `goofish_live.py` | `pkg/ws/` |
| `utils/goofish_utils.py` | `pkg/util/` |
| `utils/build_cookies.py` | `pkg/apis/cookies.go` |
| `message/types.py` | `pkg/msg/` |

## 依赖

| 库 | 用途 |
|----|------|
| [nhooyr.io/websocket](https://github.com/coder/websocket) | WebSocket 客户端 |
| [vmihailenco/msgpack/v5](https://github.com/vmihailenco/msgpack) | MessagePack 解码 |
| [skip2/go-qrcode](https://github.com/skip2/go-qrcode) | 终端二维码打印 |
| [go.uber.org/zap](https://go.uber.org/zap) | 结构化日志 |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML 配置解析 |

## License

MIT
