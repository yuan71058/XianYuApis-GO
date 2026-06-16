# XianYuApis-GO API 文档

> 包路径前缀: `github.com/cv-cat/xianyuapis`

---

## 目录

- [apis — HTTP API 封装](#apis--http-api-封装)
- [ws — WebSocket 实时通信](#ws--websocket-实时通信)
- [msg — 消息类型定义](#msg--消息类型定义)
- [model — 数据模型](#model--数据模型)
- [util — 工具函数](#util--工具函数)

---

## apis — HTTP API 封装

`import "github.com/cv-cat/xianyuapis/pkg/apis"`

### XianyuAPI 结构体

闲鱼 HTTP API 封装，内部管理 Cookie、签名、请求构建。

```go
type XianyuAPI struct { /* 内部字段 */ }
```

---

### New

```go
func New(cookies map[string]string, deviceID string) (*XianyuAPI, error)
```

创建闲鱼 API 实例。

| 参数 | 类型 | 说明 |
|------|------|------|
| cookies | `map[string]string` | 登录后的 Cookie 字典，必须包含 `unb` |
| deviceID | `string` | 设备 ID，为空则自动从 unb 生成 |

**返回**: `*XianyuAPI` 实例 + 错误信息

**示例**:
```go
cookies := map[string]string{"unb": "123456789", "_m_h5_tk": "abc_123"}
api, err := apis.New(cookies, "")
```

---

### DeviceID

```go
func (api *XianyuAPI) DeviceID() string
```

返回当前 API 实例的设备 ID。

---

### Client

```go
func (api *XianyuAPI) Client() *http.Client
```

返回内部 HTTP 客户端（用于 WebSocket 层获取 Cookie）。

---

### CookieString

```go
func (api *XianyuAPI) CookieString() string
```

返回 `.goofish.com` 域下所有 Cookie 的字符串表示，格式: `key1=val1; key2=val2`。

---

### GetToken

```go
func (api *XianyuAPI) GetToken(ctx context.Context) (string, error)
```

获取 WebSocket 连接所需的 accessToken。

内部自动处理风控检测（`FAIL_SYS_USER_VALIDATE` / `RGV587_ERROR::SM`），最多重试 3 次，每次等待 30 秒冷却。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |

**返回**: accessToken 字符串 + 错误信息

---

### RefreshToken

```go
func (api *XianyuAPI) RefreshToken(ctx context.Context) error
```

刷新登录态。建议每 10 分钟调用一次，WebSocket 客户端已内置自动刷新。

---

### RefreshMtopToken

```go
func (api *XianyuAPI) RefreshMtopToken(ctx context.Context)
```

刷新 `_m_h5_tk` Cookie。通过调用 mtop 接口触发服务端重新签发。

---

### QrcodeLogin

```go
func QrcodeLogin(cfg QrcodeLoginConfig) (*XianyuAPI, error)
```

扫码登录完整流程。自动构建初始 Cookie → 生成二维码 → 轮询扫码状态 → 完成登录。

| 参数 | 类型 | 说明 |
|------|------|------|
| cfg.PollInterval | `time.Duration` | 轮询间隔，默认 3s |
| cfg.Timeout | `time.Duration` | 超时时间，默认 120s |
| cfg.ShowQR | `bool` | 是否在终端打印二维码 |

**返回**: 登录成功的 `*XianyuAPI` 实例 + 错误信息

---

### BuildInitialCookies

```go
func BuildInitialCookies() (*http.Client, map[string]string, error)
```

构建初始 Cookie（含 tfstk），用于扫码登录前置步骤。

---

### GetItemInfo

```go
func (api *XianyuAPI) GetItemInfo(ctx context.Context, itemID string) (map[string]any, error)
```

获取闲鱼商品详情。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| itemID | `string` | 闲鱼商品 ID |

**返回**: 商品详情 JSON + 错误信息

---

### GetPublicChannel

```go
func (api *XianyuAPI) GetPublicChannel(ctx context.Context, title string, images []model.ImageInfo) (map[string]any, error)
```

获取商品推荐标签和分类建议，用于商品发布时自动推荐属性。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| title | `string` | 商品标题/描述 |
| images | `[]model.ImageInfo` | 已上传的图片信息列表 |

**返回**: 推荐结果 JSON（包含 cardList 和 categoryPredictResult）+ 错误信息

---

### GetDefaultLocation

```go
func (api *XianyuAPI) GetDefaultLocation(ctx context.Context) (map[string]any, error)
```

获取默认地理位置信息，用于商品发布时填写发货地址。

**返回**: 位置信息 JSON（包含 area、city、prov、gps 等字段）+ 错误信息

---

### PublishItem

```go
func (api *XianyuAPI) PublishItem(ctx context.Context, images []string, desc string, price *model.Price, ds model.DeliverySettings) (map[string]any, error)
```

发布闲鱼商品。内部自动执行: 上传图片 → 获取推荐标签 → 获取默认地址 → 提交发布。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| images | `[]string` | 本地图片文件路径列表（最多 9 张） |
| desc | `string` | 商品标题和描述 |
| price | `*model.Price` | 价格信息，nil 使用默认定价 |
| ds | `model.DeliverySettings` | 配送设置 |

**返回**: 发布结果 JSON + 错误信息

---

### UploadMedia

```go
func (api *XianyuAPI) UploadMedia(ctx context.Context, filePath string) (*UploadMediaResult, error)
```

上传媒体文件（图片）到闲鱼服务器。支持 PNG/JPG/JPEG/GIF 格式。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| filePath | `string` | 本地文件路径 |

**返回**:

```go
type UploadMediaResult struct {
    URL    string // 图片访问 URL
    Width  int    // 图片宽度
    Height int    // 图片高度
}
```

---

### ConfirmShipping

```go
func (api *XianyuAPI) ConfirmShipping(ctx context.Context, orderID string) (map[string]any, error)
```

自动确认发货。调用 `mtop.taobao.idle.logistic.consign.dummy` API，适用于虚拟商品自动发货场景。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| orderID | `string` | 订单 ID |

**返回**: 确认结果 JSON（成功时 `ret` 包含 `SUCCESS::调用成功`）+ 错误信息

---

## ws — WebSocket 实时通信

`import "github.com/cv-cat/xianyuapis/pkg/ws"`

### XianyuWS 结构体

闲鱼 WebSocket 客户端，封装连接、注册、心跳、消息收发。

```go
type XianyuWS struct { /* 内部字段 */ }
```

---

### New

```go
func New(cookies map[string]string, deviceID string) (*XianyuWS, error)
```

创建 WebSocket 客户端。内部自动创建 `XianyuAPI` 实例。

| 参数 | 类型 | 说明 |
|------|------|------|
| cookies | `map[string]string` | Cookie 字典，必须包含 `unb` |
| deviceID | `string` | 设备 ID |

**返回**: `*XianyuWS` 实例 + 错误信息

---

### NewWithAPI

```go
func NewWithAPI(api *apis.XianyuAPI) (*XianyuWS, error)
```

使用已有的 API 实例创建 WebSocket 客户端（推荐，共享登录态）。

| 参数 | 类型 | 说明 |
|------|------|------|
| api | `*apis.XianyuAPI` | 已初始化的 API 实例 |

**返回**: `*XianyuWS` 实例 + 错误信息

---

### Connect

```go
func (ws *XianyuWS) Connect(ctx context.Context) error
```

建立 WebSocket 连接。自动获取 Token → 连接 → 注册 → 同步状态 → 启动心跳。

等同于 `ConnectWithToken(ctx, "")`。

---

### ConnectWithToken

```go
func (ws *XianyuWS) ConnectWithToken(ctx context.Context, token string) error
```

使用指定的 accessToken 建立 WebSocket 连接。如果 token 为空，则自动调用 `api.GetToken()` 获取。

完整流程:
1. 获取 Token（如为空）
2. 建立 WebSocket 连接
3. 启动 recvLoop（消息接收循环）
4. 发送 `/reg` 注册并等待响应
5. 发送 `/r/SyncStatus/ackDiff` 同步状态
6. 冷却 3 秒
7. 启动心跳 goroutine

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 控制上下文 |
| token | `string` | accessToken，为空则自动获取 |

**返回**: 连接错误信息

---

### Start

```go
func (ws *XianyuWS) Start() error
```

启动消息接收循环（阻塞）。在 `Connect` / `ConnectWithToken` 之后调用。

**返回**: 连接关闭时的错误信息，`context.Canceled` 表示正常退出

---

### Stop

```go
func (ws *XianyuWS) Stop()
```

停止 WebSocket 客户端。先关闭 WebSocket 连接，再取消 context。

---

### IsAlive

```go
func (ws *XianyuWS) IsAlive() bool
```

返回 WebSocket 连接是否存活。可用于判断是否需要重连。

---

### SetMessageHandler

```go
func (ws *XianyuWS) SetMessageHandler(h MessageHandler)
```

设置消息处理回调函数。

| 参数 | 类型 | 说明 |
|------|------|------|
| h | `MessageHandler` | 消息回调函数，签名为 `func(m *msg.Message)` |

---

### StartTokenRefresher

```go
func (ws *XianyuWS) StartTokenRefresher(interval ...time.Duration)
```

启动后台 Token 刷新 goroutine。与 Python 版 `user_alive()` 对齐。

| 参数 | 类型 | 说明 |
|------|------|------|
| interval | `...time.Duration` | 可选，刷新间隔。0 或不传使用默认值 10 分钟 |

**示例**:
```go
// 默认 10 分钟刷新
wsClient.StartTokenRefresher()

// 自定义 5 分钟刷新
wsClient.StartTokenRefresher(5 * time.Minute)
```

---

### SendText

```go
func (ws *XianyuWS) SendText(ctx context.Context, cid, toID, text string) error
```

发送文字消息。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| cid | `string` | 会话 ID（不含 `@goofish` 后缀） |
| toID | `string` | 接收方用户 ID |
| text | `string` | 消息文本内容 |

**返回**: 发送错误信息

---

### SendImage

```go
func (ws *XianyuWS) SendImage(ctx context.Context, cid, toID, imageURL string, width, height int) error
```

发送图片消息。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| cid | `string` | 会话 ID（不含 `@goofish` 后缀） |
| toID | `string` | 接收方用户 ID |
| imageURL | `string` | 图片 URL（需先通过 `UploadMedia` 上传获取） |
| width | `int` | 图片宽度 |
| height | `int` | 图片高度 |

**返回**: 发送错误信息

---

### ListAllConversations

```go
func (ws *XianyuWS) ListAllConversations(ctx context.Context, cid string) ([]*ConversationMessage, error)
```

获取与指定用户的全部历史聊天记录。建立临时连接，获取完自动关闭。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| cid | `string` | 会话 ID |

**返回**: 历史消息列表 + 错误信息

```go
type ConversationMessage struct {
    SenderID   string `json:"send_user_id"`
    SenderName string `json:"send_user_name"`
    Message    any    `json:"message"`
}
```

---

### CreateChat

```go
func (ws *XianyuWS) CreateChat(ctx context.Context, toID, itemID string) error
```

创建新会话，可关联商品 ID。

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| toID | `string` | 目标用户 ID |
| itemID | `string` | 关联商品 ID |

**返回**: 创建错误信息

---

### DeviceID / MyID

```go
func (ws *XianyuWS) DeviceID() string
func (ws *XianyuWS) MyID() string
```

返回设备 ID / 当前用户 unb ID。

---

## msg — 消息类型定义

`import "github.com/cv-cat/xianyuapis/pkg/msg"`

### MessageType 枚举

```go
type MessageType int

const (
    MessageTypeText  MessageType = 1  // 文字消息
    MessageTypeImage MessageType = 2  // 图片消息
    MessageTypeAudio MessageType = 26 // 音频消息
)
```

---

### Message 结构体

```go
type Message struct {
    SenderID       string      // 发送者用户 ID
    SenderName     string      // 发送者昵称
    Content        string      // 消息文本内容
    MessageType    MessageType // 消息类型
    ConversationID string      // 会话 ID
    ImageURL       string      // 图片 URL（仅图片消息）
    ImageWidth     int         // 图片宽度
    ImageHeight    int         // 图片高度
    Timestamp      time.Time   // 消息时间
    Raw            any         // 原始消息数据（调试用）
}
```

---

### Message 方法

| 方法 | 签名 | 说明 |
|------|------|------|
| IsText | `func (m *Message) IsText() bool` | 判断是否为文字消息 |
| IsImage | `func (m *Message) IsImage() bool` | 判断是否为图片消息 |
| String | `func (mt MessageType) String() string` | 消息类型转字符串 |

---

### NewTextMessage

```go
func NewTextMessage(senderID, senderName, content, conversationID string) *Message
```

创建文字消息。

---

### NewImageMessage

```go
func NewImageMessage(senderID, senderName, imageURL, conversationID string, width, height int) *Message
```

创建图片消息。

---

### 消息解析支持的 contentType

| contentType | 类型 | 解析内容 | 数据来源 |
|:-----------:|------|---------|---------|
| 1 | 文字消息 | `text.text` 字段 | 格式1/2 |
| 2 | 图片消息 | `image.pics[].url` + 宽高 | 格式2 |
| 3 | 语音消息 | `[语音消息]` | 格式1/2 |
| 7 | 商品卡片 | `[我想要] 标题 价格 (id:xxx)` | 格式2 |
| 17 | 转账消息 | `[转账] ¥金额 (交易号:xxx)` | 格式2 |
| 30 | 位置消息 | `[位置] 经度:xxx 纬度:xxx` | 格式2 |

**编码格式说明**:
- **格式1**: `msg6_3["1"]` 为 base64 字符串，解码后得到 JSON（文字消息常用）
- **格式2**: `msg6_3["5"]` 为直接 JSON 字符串（图片/卡片/转账/位置等常用）

---

## model — 数据模型

`import "github.com/cv-cat/xianyuapis/pkg/model"`

### Price

```go
type Price struct {
    CurrentPrice  float64 // 当前售价
    OriginalPrice float64 // 原价
}
```

---

### DeliverySettings

```go
type DeliverySettings struct {
    Choice        string  // 配送方式: "包邮" | "按距离计费" | "一口价" | "无需邮寄"
    PostPrice     float64 // 运费（仅 "一口价" 时有效）
    CanSelfPickup bool    // 是否支持自提
}
```

---

### ImageInfo

```go
type ImageInfo struct {
    URL    string // 图片 URL
    Width  int    // 图片宽度
    Height int    // 图片高度
}
```

---

### QrcodeLoginConfig

```go
type QrcodeLoginConfig struct {
    PollInterval time.Duration // 轮询间隔，默认 3s
    Timeout      time.Duration // 超时时间，默认 120s
    ShowQR       bool          // 是否在终端打印二维码
}
```

---

## util — 工具函数

`import "github.com/cv-cat/xianyuapis/pkg/util"`

### GenerateSign

```go
func GenerateSign(timestamp, token, data string) string
```

生成闲鱼 mtop API 签名。公式: `MD5(token + "&" + timestamp + "&" + appKey + "&" + data)`，appKey 固定值 `34839810`。

| 参数 | 类型 | 说明 |
|------|------|------|
| timestamp | `string` | 毫秒级时间戳 |
| token | `string` | 从 `_m_h5_tk` Cookie 下划线前提取 |
| data | `string` | 请求数据 JSON 字符串 |

**返回**: 32 位 MD5 签名字符串

---

### Base64DecodeUTF8

```go
func Base64DecodeUTF8(data string) (string, error)
```

将 Base64 字符串解码为 UTF-8 字符串。

---

### Decrypt

```go
func Decrypt(data string) (string, error)
```

解密闲鱼加密消息。使用自定义 msgpack 解码器，自动将整数键转为字符串键（与 Python 版 `MessagePackDecoder` 对齐）。

解码流程: `base64 → 自定义 msgpack 解码 → JSON 字符串`

---

### GenerateMid

```go
func GenerateMid() string
```

生成消息 ID，格式: `"毫秒时间戳 0"`，如 `"7381748291023 0"`。

---

### GenerateUUID

```go
func GenerateUUID() string
```

生成请求标识 UUID，格式: `"-毫秒时间戳随机数"`。

---

### GenerateDeviceID

```go
func GenerateDeviceID(userID string) string
```

生成设备 ID。格式: `8-4-4-4-12位随机字符-userID`，字符集: `0-9A-Za-z`。

| 参数 | 类型 | 说明 |
|------|------|------|
| userID | `string` | 用户 ID（unb） |

**返回**: 设备 ID 字符串

---

### GenerateTFstk

```go
func GenerateTFstk(scriptPath string) (string, error)
```

通过 Node.js 子进程生成 tfstk 值，用于构建初始 Cookie。

---

### ParseCookies

```go
func ParseCookies(cookieStr string) map[string]string
```

解析 Cookie 字符串为 map。输入格式: `"key1=val1; key2=val2"`。

---

### BuildCookieString

```go
func BuildCookieString(cookies map[string]string) string
```

将 Cookie map 构建为字符串，格式: `"key1=val1; key2=val2"`。

---

### GetCookieFromJar

```go
func GetCookieFromJar(jar http.CookieJar, domain, name string) string
```

从 CookieJar 中读取指定域名的 Cookie 值。自动将 `.goofish.com` 转为 `www.goofish.com` 查询。

---

### SetCookieToJar

```go
func SetCookieToJar(jar http.CookieJar, domain, name, value, path string)
```

向 CookieJar 写入指定域名的 Cookie。自动将 `.goofish.com` 转为 `www.goofish.com` 设置。

---

## 常量

| 常量 | 值 | 说明 |
|------|-----|------|
| `wsBaseURL` | `wss://wss-goofish.dingtalk.com/ws` | WebSocket 连接地址 |
| `appKey` | `34839810` | mtop API 固定 appKey |
| `UA` | `Mozilla/5.0 ...` | 统一 User-Agent |
| `regTimeout` | `5s` | /reg 注册超时时间 |
| `regCooldown` | `3s` | /reg 注册后冷却时间 |
| `heartbeatInterval` | `15s` | 心跳间隔 |
| `tokenRefreshInterval` | `10m` | Token 刷新间隔 |
