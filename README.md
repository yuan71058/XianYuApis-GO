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
├── cmd/demo/main.go        # 入口示例
├── config/                  # YAML 配置 + 环境变量
├── pkg/
│   ├── apis/                # HTTP API 封装（登录、商品、上传）
│   ├── ws/                  # WebSocket 通信（连接、收发、心跳）
│   ├── msg/                 # 消息类型定义
│   ├── model/               # 数据模型
│   └── util/                # 签名、解密、ID 生成、Cookie 工具
├── internal/
│   ├── lwp/                 # LWP 协议编解码
│   └── httpclient/          # HTTP 客户端工厂
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

## 使用示例

### 扫码登录

```go
api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
    PollInterval: 3 * time.Second,
    Timeout:      120 * time.Second,
    ShowQR:       true,  // 终端打印二维码
})
```

### 已有 Cookie 登录

```go
cookies := map[string]string{
    "unb":       "xxx",
    "tracknick": "yyy",
    // ... 其他 Cookie
}
api, err := apis.New(cookies, "")
```

### WebSocket 消息收发

```go
client, _ := ws.New(cookies, api.DeviceID())

client.SetMessageHandler(func(m *msg.Message) {
    fmt.Printf("[%s] %s: %s\n", m.SenderName, m.SenderID, m.Content)
    // 回复
    client.SendText(context.Background(), m.ConversationID, m.SenderID, "收到")
})

client.StartTokenRefresher()
client.Connect(ctx)
client.Start()  // 阻塞
```

### 商品发布

```go
result, err := api.PublishItem(ctx,
    []string{"image1.png", "image2.jpg"},  // 图片路径
    "商品描述",
    &model.Price{CurrentPrice: 99.9, OriginalPrice: 199.9},
    model.DeliverySettings{Choice: "包邮"},
)
```

### 图片上传

```go
result, err := api.UploadMedia(ctx, "photo.png")
fmt.Println(result.URL, result.Width, result.Height)
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
