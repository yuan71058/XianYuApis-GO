<div align="center">

# XianYuApis-GO

**闲鱼 API Go 语言实现**

从 [XianYuApis (Python)](https://github.com/cv-cat/XianYuApis) 迁移而来 

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

## ✨ 功能特性

| 模块 | 功能 | 说明 |
|:----:|------|------|
| 🔐 | **多种登录方式** | 扫码登录 / 手动 Cookie+Token 登录（推荐，避免风控） |
| 💬 | **WebSocket 实时消息** | 收发文字/图片，双心跳保活，Token 自动刷新 |
| 📦 | **多类型消息解析** | 文字 / 图片 / 商品卡片 / 转账 / 位置 / 语音，完整支持闲鱼消息协议 |
| 🛒 | **HTTP API** | 商品发布、详情查询、图片上传、Token 管理、自动确认发货 |
| 🔑 | **消息加解密** | Base64 + MessagePack 协议，自定义 msgpack 解码器支持整数键转字符串键 |
| 🔄 | **连接状态管理** | `IsAlive()` 查询连接状态，支持自动重连 |

---

## 📁 项目结构

```
xianyuapis/
├── cmd/demo/main.go            # 📌 详细调用示例（含完整中文注释）
├── config/                      # YAML 配置 + 环境变量
├── pkg/
│   ├── apis/                    # 🌐 HTTP API 封装
│   │   ├── api.go               #   XianyuAPI 核心 + mtop 请求封装
│   │   ├── login.go             #   GetToken / RefreshToken（含风控检测）
│   │   ├── qrcode.go            #   扫码登录完整流程
│   │   ├── cookies.go           #   BuildInitialCookies
│   │   ├── upload.go            #   UploadMedia 图片上传
│   │   └── product.go           #   GetItemInfo / PublishItem / ConfirmShipping
│   ├── ws/                      # 🔌 WebSocket 通信
│   │   ├── client.go            #   XianyuWS 结构体 + 构造函数 + IsAlive
│   │   ├── connect.go           #   Connect / ConnectWithToken / heartbeat / Token刷新
│   │   ├── receiver.go          #   Start / handleMessage / ACK / 多类型消息解析
│   │   ├── sender.go            #   SendText / SendImage
│   │   └── conversation.go      #   ListAllConversations / CreateChat
│   ├── msg/                     # 📨 消息类型定义
│   │   ├── types.go             #   Message 结构体 + MessageType 枚举
│   │   └── factory.go           #   NewTextMessage / NewImageMessage
│   ├── model/                   # 📊 数据模型
│   │   ├── product.go           #   Price / DeliverySettings / ImageInfo
│   │   └── login.go             #   QrcodeLoginConfig / QRCodeData
│   └── util/                    # 🔧 工具函数
│       ├── sign.go              #   GenerateSign（MD5 签名）
│       ├── decrypt.go           #   Decrypt（自定义 msgpack 解码器）
│       ├── gen.go               #   GenerateMid / UUID / DeviceID
│       ├── cookie.go            #   Cookie 解析/构建/Jar 操作
│       ├── tfstk.go             #   GenerateTFstk（Node.js 子进程）
│       └── version.go           #   版本常量
├── internal/
│   ├── lwp/                     # LWP 协议编解码
│   └── httpclient/              # HTTP 客户端工厂 + 中间件
├── assets/                      # JS 脚本（tfstk 生成）
├── API.md                       # 📖 详细 API 文档
├── Dockerfile
└── Makefile
```

---

## 🚀 快速开始

### 前置条件

- Go 1.22+
- （可选）Node.js — 扫码登录需要生成 tfstk

### 安装

```bash
git clone https://github.com/yuan71058/XianYuApis-GO.git
cd XianYuApis-GO/xianyuapis
go mod tidy
```

### 运行

```bash
# 编译运行
go build -o bin/demo.exe ./cmd/demo/
./bin/demo.exe

# 或直接运行
go run ./cmd/demo/
```

---

## 📖 详细 API 调用指南

> 完整可运行代码见 [cmd/demo/main.go](cmd/demo/main.go)，每个函数均有详细中文注释。
>
> 详细 API 文档见 [API.md](API.md)，包含所有函数的签名、参数、返回值、错误处理和示例。

### 1. 登录

#### 方式一：手动 Cookie + Token 登录（推荐，避免风控）

**步骤 1: 获取 Token**

浏览器打开 `https://www.goofish.com` 并登录，F12 → Console:

```javascript
// 第1步: 加载 MD5 库
var s=document.createElement('script');
s.src='https://cdn.bootcdn.net/ajax/libs/blueimp-md5/2.19.0/js/md5.min.js';
document.head.appendChild(s);
setTimeout(()=>console.log('MD5库加载完成'),1000)

// 第2步: 等待 "MD5库加载完成" 后，获取 Token（deviceId 替换为程序输出的值）
(async()=>{let t=Date.now(),tk=document.cookie.match(/_m_h5_tk=([^;]+)/)[1].split('_')[0],
d=JSON.stringify({appKey:'444e9908a51d1cb236a27862abc769c9',deviceId:'YOUR_DEVICE_ID'}),
sign=md5(tk+'&'+t+'&34839810&'+d);
let r=await fetch('https://h5api.m.goofish.com/h5/mtop.taobao.idlemessage.pc.login.token/1.0/?jsv=2.7.2&appKey=34839810&t='+t+'&sign='+sign+'&v=1.0&type=originaljson&dataType=json&timeout=20000&api=mtop.taobao.idlemessage.pc.login.token&sessionOption=AutoLoginOnly',
{method:'POST',headers:{'content-type':'application/x-www-form-urlencoded'},
body:'data='+encodeURIComponent(d),credentials:'include'});
let j=await r.json();console.log('TOKEN:',j.data?.accessToken)})()
```

**步骤 2: 获取 Cookie**

F12 → Network → 刷新页面 → 点第一个请求 → Request Headers 中复制完整 Cookie

> ⚠️ 必须从 Network 请求头复制，`document.cookie` 无法获取 HttpOnly 的 `cookie2`、`sgcookie`

```go
// Demo 中选择选项 2，输入 Token 和 Cookie 即可
```

#### 方式二：扫码登录

```go
api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
    PollInterval: 3 * time.Second,   // 轮询间隔
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
    "cookie2":   "xxxxxxxxxxxxxxxx",  // HttpOnly，必须从 Network 请求头复制
    "sgcookie":  "E100xxx...",         // HttpOnly，必须从 Network 请求头复制
}
api, err := apis.New(cookies, "")
```

---

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
// 返回商品详情 JSON，包含 title、price、picList 等
```

#### 2.3 图片上传

```go
result, err := api.UploadMedia(ctx, "./photo.png")
fmt.Printf("URL: %s, 尺寸: %dx%d\n", result.URL, result.Width, result.Height)
```

#### 2.4 商品发布

```go
result, err := api.PublishItem(ctx,
    []string{"./product1.jpg", "./product2.jpg"},  // 本地图片路径
    "九成新机械键盘 自用半年 功能完好",               // 商品描述
    &model.Price{CurrentPrice: 299.0, OriginalPrice: 599.0},  // 价格
    model.DeliverySettings{Choice: "包邮"},         // 配送方式
)
```

#### 2.5 自动确认发货

```go
// 适用于虚拟商品自动发货场景
result, err := api.ConfirmShipping(ctx, "订单ID")
// 成功时 ret 包含 "SUCCESS::调用成功"
```

---

### 3. WebSocket 实时通信

#### 3.1 创建客户端并连接

```go
// 从 API 客户端创建 WebSocket 客户端（共享登录态，推荐）
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

// 建立连接（支持手动传入 Token，避免风控）
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

#### 3.3 连接状态与重连

```go
// 检查连接是否存活
if !wsClient.IsAlive() {
    fmt.Println("连接断开，尝试重连...")
    wsClient.ConnectWithToken(ctx, "")
}
```

#### 3.4 优雅退出

```go
sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
defer wsClient.Stop()
```

---

### 4. 消息类型

闲鱼 WebSocket 推送消息支持以下类型:

| contentType | 类型 | Content 字段 | ImageURL 字段 |
|:-----------:|------|-------------|--------------|
| 1 | 💬 文字消息 | `text.text` 的值 | — |
| 2 | 🖼️ 图片消息 | — | `image.pics[].url` + 宽高 |
| 3 | 🎤 语音消息 | `[语音消息]` | — |
| 7 | 🛒 商品卡片 | `[我想要] 标题 价格 (id:xxx)` | 商品主图 URL |
| 17 | 💰 转账消息 | `[转账] ¥金额 (交易号:xxx)` | — |
| 30 | 📍 位置消息 | `[位置] 经度:xxx 纬度:xxx` | — |

**消息编码格式**:
- **格式1** (base64): `msg6_3["1"]` 为 base64 字符串 → 解码后 JSON（文字消息常用）
- **格式2** (直接 JSON): `msg6_3["5"]` 为直接 JSON 字符串（图片/卡片/转账/位置等常用）

---

### 5. Message 结构体

```go
type Message struct {
    SenderID       string      // 发送者用户 ID
    SenderName     string      // 发送者昵称
    Content        string      // 消息文本内容
    MessageType    MessageType // 消息类型 (1=文字, 2=图片, 26=音频)
    ConversationID string      // 会话 ID
    ImageURL       string      // 图片 URL（仅图片消息，多图逗号分隔）
    ImageWidth     int         // 首张图片宽度
    ImageHeight    int         // 首张图片高度
    Timestamp      time.Time   // 消息时间
    Raw            any         // 原始数据（调试用）
}
```

---

### 6. 工具函数

```go
// 签名生成: MD5(token + "&" + timestamp + "&" + appKey + "&" + data)
sign := util.GenerateSign(timestamp, token, data)

// 消息解密: base64 → 自定义 msgpack 解码 → JSON（整数键自动转字符串键）
decrypted, err := util.Decrypt(encryptedData)

// ID 生成
mid      := util.GenerateMid()              // 消息 ID
uuid     := util.GenerateUUID()             // 请求标识
deviceID := util.GenerateDeviceID("unb123") // 设备 ID

// Cookie 工具
cookies  := util.ParseCookies("key1=val1; key2=val2")  // 字符串 → map
cookieStr := util.BuildCookieString(cookies)            // map → 字符串
```

---

## ⚠️ 关键技术说明

### 风控规避

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| `FAIL_SYS_USER_VALIDATE` | 扫码登录触发风控 | 使用手动 Cookie+Token 方式 |
| `document.cookie` 缺少字段 | `cookie2`、`sgcookie` 是 HttpOnly | 从 F12→Network→请求头中复制 |
| 多次失败后仍被拦截 | 账号被限流 | 等待 5-10 分钟再重试 |
| Token 获取返回 `RGV587_ERROR::SM` | API 被限流 | 等待 30 秒冷却或使用手动 Token |

### Token 与 DeviceID 配对

> 浏览器获取 Token 时使用的 `deviceId` **必须**与 Go 的 `/reg` 请求中使用的 `deviceId` 一致。

不一致时返回: `401 device id or appkey is not equal`

Demo 中已自动处理此配对: 先在 Go 中生成 deviceID → 输出给用户 → 用户在浏览器脚本中使用同一 deviceID。

### WebSocket 连接注意事项

- **禁用压缩**: 服务端不协商 `permessage-deflate`，必须设置 `CompressionDisabled`
- **注册冷却**: `/reg` 后需等待 3 秒，避免 IM 流控错误 `400600001`
- **完整 Cookie**: 必须包含 HttpOnly 字段（`cookie2`、`sgcookie`），否则 mtop API 认证失败
- **退出顺序**: Ctrl+C 时先关闭 WebSocket 连接再取消 context，否则 Read 会阻塞 60 秒

### 双心跳机制

| 心跳类型 | 间隔 | 说明 |
|---------|------|------|
| LWP 心跳 | 15 秒 | 发送 `{"lwp":"/!"}` 消息 |
| WebSocket PING | 30 秒 | 发送 PING 帧（与 Python aiohttp `heartbeat=30` 对齐） |

---

## 🔄 与 Python 版本的对应关系

| Python 模块 | Go 包 | 说明 |
|------------|-------|------|
| `goofish_apis.py` | `pkg/apis/` | HTTP API 封装 |
| `goofish_live.py` | `pkg/ws/` | WebSocket 通信 |
| `utils/goofish_utils.py` | `pkg/util/` | 工具函数 |
| `utils/build_cookies.py` | `pkg/apis/cookies.go` | Cookie 构建 |
| `message/types.py` | `pkg/msg/` | 消息类型定义 |
| `push_message_parser.py` | `pkg/ws/receiver.go` | 消息解析 |
| `confirm_service.py` | `pkg/apis/product.go` | 自动确认发货 |

### Go 版本新增（原库没有）

| 功能 | 说明 |
|------|------|
| `ConfirmShipping()` | 自动确认发货（参考 xianyu-auto-reply） |
| `QrcodeLogin()` | 扫码登录完整流程 |
| `ConnectWithToken()` | 手动 Token 连接（避免风控） |
| `IsAlive()` | 连接状态查询（支持重连） |
| 多类型消息解析 | 商品卡片(7) / 转账(17) / 位置(30) |
| `StartTokenRefresher(interval)` | 可自定义间隔的 Token 刷新 |

---

## 📦 依赖

| 库 | 用途 |
|----|------|
| [nhooyr.io/websocket](https://github.com/coder/websocket) | WebSocket 客户端 |
| [skip2/go-qrcode](https://github.com/skip2/go-qrcode) | 终端二维码打印 |
| [go.uber.org/zap](https://go.uber.org/zap) | 结构化日志 |
| [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML 配置解析 |

---

## 📄 文档

- [API.md](API.md) — 完整 API 文档，包含所有函数的签名、参数、返回值、错误处理和示例

---

## 📜 License

MIT
