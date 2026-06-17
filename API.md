# XianYuApis-GO API 文档

> 包路径前缀: `github.com/cv-cat/xianyuapis`
>
> 所有 HTTP API 方法均为线程安全，可在多个 goroutine 中并发调用同一 `XianyuAPI` 实例。

---

## 目录

- [apis — HTTP API 封装](#apis--http-api-封装)
  - [XianyuAPI 结构体](#xianyuapi-结构体)
  - [New](#new)
  - [DeviceID](#deviceid)
  - [Client](#client)
  - [CookieString](#cookiestring)
  - [GetToken](#gettoken)
  - [RefreshToken](#refreshtoken)
  - [RefreshMtopToken](#refreshmtopken)
  - [QrcodeLogin](#qrcodelogin)
  - [BuildInitialCookies](#buildinitialcookies)
  - [GetItemInfo](#getiteminfo)
  - [GetPublicChannel](#getpublicchannel)
  - [GetDefaultLocation](#getdefaultlocation)
  - [PublishItem](#publishitem)
  - [UploadMedia](#uploadmedia)
  - [ConfirmShipping](#confirmshipping)
- [ws — WebSocket 实时通信](#ws--websocket-实时通信)
  - [XianyuWS 结构体](#xianyuws-结构体)
  - [New](#new-1)
  - [NewWithAPI](#newwithapi)
  - [Connect](#connect)
  - [ConnectWithToken](#connectwithtoken)
  - [Start](#start)
  - [Stop](#stop)
  - [IsAlive](#isalive)
  - [SetMessageHandler](#setmessagehandler)
  - [StartTokenRefresher](#starttokenrefresher)
  - [SendText](#sendtext)
  - [SendImage](#sendimage)
  - [ListAllConversations](#listallconversations)
  - [CreateChat](#createchat)
  - [DeviceID / MyID](#deviceid--myid)
- [msg — 消息类型定义](#msg--消息类型定义)
  - [MessageType 枚举](#messagetype-枚举)
  - [Message 结构体](#message-结构体)
  - [Message 方法](#message-方法)
  - [NewTextMessage](#newtextmessage)
  - [NewImageMessage](#newimagemessage)
  - [消息解析支持的 contentType](#消息解析支持的-contenttype)
- [model — 数据模型](#model--数据模型)
  - [Price](#price)
  - [DeliverySettings](#deliverysettings)
  - [ImageInfo](#imageinfo)
  - [QrcodeLoginConfig](#qrcodeloginconfig)
- [util — 工具函数](#util--工具函数)
  - [GenerateSign](#generatesign)
  - [Base64DecodeUTF8](#base64decodeutf8)
  - [Decrypt](#decrypt)
  - [GenerateMid](#generatemid)
  - [GenerateUUID](#generateuuid)
  - [GenerateDeviceID](#generatedeviceid)
  - [GenerateTFstk](#generatetfstk)
  - [ParseCookies](#parsecookies)
  - [BuildCookieString](#buildcookiestring)
  - [GetCookieFromJar](#getcookiefromjar)
  - [SetCookieToJar](#setcookietojar)
- [常量](#常量)
- [错误处理](#错误处理)
- [完整示例](#完整示例)

---

## apis — HTTP API 封装

`import "github.com/cv-cat/xianyuapis/pkg/apis"`

### XianyuAPI 结构体

闲鱼 HTTP API 封装，内部管理 Cookie、签名、请求构建。

```go
type XianyuAPI struct { /* 内部字段 */ }
```

**内部机制**:
- 使用双 HTTP 客户端架构:
  - `client`: 带 CookieJar 的客户端，用于 Cookie 管理（查询/存储）
  - `noJarClient`: 无 Jar 的客户端，用于发送 mtop 请求（手动设置 Cookie header，避免 CookieJar 覆盖 HttpOnly 字段）
- 所有 mtop 请求通过 `doMtopRequest` 统一处理: 自动签名、设置 header、解析响应
- Cookie 的 Domain 设为 `.goofish.com`，使用 `https://www.goofish.com` 作为合法 URL 设置（RFC 6265）

**线程安全**: 同一 `XianyuAPI` 实例可被多个 goroutine 并发调用。

---

### New

```go
func New(cookies map[string]string, deviceID string) (*XianyuAPI, error)
```

创建闲鱼 API 实例。

| 参数 | 类型 | 说明 |
|------|------|------|
| cookies | `map[string]string` | 登录后的 Cookie 字典，**必须包含 `unb`** |
| deviceID | `string` | 设备 ID，为空则自动从 unb 生成 |

**返回**: `*XianyuAPI` 实例 + 错误信息

**错误情况**:
- `cookies must contain 'unb' field` — Cookie 字典缺少 `unb` 字段
- `create cookiejar: ...` — CookieJar 创建失败（极少见）

**注意事项**:
- Cookie 字典中 `cookie2` 和 `sgcookie` 是 HttpOnly 字段，无法通过 `document.cookie` 获取，必须从浏览器 F12→Network→请求头中复制
- 如果 deviceID 为空，会调用 `util.GenerateDeviceID(unb)` 自动生成

**示例**:
```go
// 方式1: 最简创建（自动生成 deviceID）
cookies := map[string]string{
    "unb":       "1088389759",
    "_m_h5_tk":  "cf64b07dc62279e55bc0a030719d59a9_1781616565787",
    "cookie2":   "283d60b38b047c43ca907174a44d2561",
    "sgcookie":  "E100KNAU1mcNdjYjV5hi0ZvDrh0j2WiQ6QuK5C8IAslxsySi...",
}
api, err := apis.New(cookies, "")

// 方式2: 指定 deviceID（需与浏览器获取 Token 时使用的 deviceID 一致）
api, err := apis.New(cookies, "kev7fdfD-OLm7-4SK8-BBzg-Fbvxobzh0ggq-1088389759")
```

---

### DeviceID

```go
func (api *XianyuAPI) DeviceID() string
```

返回当前 API 实例的设备 ID。

**示例**:
```go
fmt.Println(api.DeviceID()) // "MxELdSFn-GrTE-4XT1-Bgh2-gajPurajZ4re-1088389759"
```

---

### Client

```go
func (api *XianyuAPI) Client() *http.Client
```

返回内部带 CookieJar 的 HTTP 客户端。主要用于 WebSocket 层通过 CookieJar 查询 Cookie。

---

### CookieString

```go
func (api *XianyuAPI) CookieString() string
```

返回 `.goofish.com` 域下所有 Cookie 的字符串表示，格式: `key1=val1; key2=val2`。

**内部机制**:
- 遍历 `www.goofish.com`、`goofish.com`、`h5api.m.goofish.com`、`passport.goofish.com` 四个域名
- 去重（同名 Cookie 只保留第一个）
- 拼接为 `name=value; name2=value2` 格式

**用途**: WebSocket 连接时设置 `Cookie` header、mtop 请求时手动设置 Cookie

**示例**:
```go
cookieStr := api.CookieString()
fmt.Println(cookieStr) // "tracknick=%E7%BD%91...; unb=1088389759; cookie2=283d60b..."
```

---

### GetToken

```go
func (api *XianyuAPI) GetToken(ctx context.Context) (string, error)
```

获取 WebSocket 连接所需的 accessToken。

**对应 API**: `mtop.taobao.idlemessage.pc.login.token/1.0`

**内部机制**:
1. 构建 `{"appKey":"444e9908a51d1cb236a27862abc769c9","deviceId":"xxx"}` 作为请求数据
2. 自动从 CookieJar 中提取 `_m_h5_tk` 计算 MD5 签名
3. 最多重试 3 次:
   - 请求失败: 等待递增时间后重试
   - 风控验证码 (`FAIL_SYS_USER_VALIDATE` / `RGV587_ERROR::SM`): 等待 30 秒冷却 + 刷新 `_m_h5_tk` 后重试
   - 令牌过期: 等待 500ms 后重试
4. 从响应 `data.accessToken` 提取 Token

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文，可用于超时控制 |

**返回**: accessToken 字符串（格式: `oauth_k1:xxxxx==`）+ 错误信息

**错误情况**:
- `_m_h5_tk cookie not found` — Cookie 中缺少 `_m_h5_tk`，需先登录
- `token not found in response: ...` — 响应中无 accessToken，可能 Cookie 已过期
- `get token failed after 3 retries` — 3 次重试均失败，通常因风控拦截

**注意事项**:
- Token 与 DeviceID 配对: 获取 Token 时使用的 `deviceId` 必须与 WebSocket `/reg` 中使用的 `deviceId` 一致
- 如果触发风控，建议使用手动 Cookie+Token 方式绕过（见 demo 示例）

**示例**:
```go
token, err := api.GetToken(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Println(token) // "oauth_k1:6SX/feDQW01ApS469NEyBlDRh/HzAQlTM4L/0QTNk0ep..."
```

---

### RefreshToken

```go
func (api *XianyuAPI) RefreshToken(ctx context.Context) error
```

刷新登录态。调用 `mtop.taobao.idlemessage.pc.loginuser.get/1.0`，维持 WebSocket 连接不掉线。

**对应 API**: `mtop.taobao.idlemessage.pc.loginuser.get/1.0`

**与 Python 版对应**: `user_alive()` 中的 Token 刷新逻辑

**使用建议**: 通过 `ws.StartTokenRefresher()` 自动定期调用，无需手动调用

**错误情况**:
- `_m_h5_tk cookie not found` — Cookie 已失效
- 网络错误等

**示例**:
```go
// 手动刷新（通常不需要，StartTokenRefresher 会自动处理）
err := api.RefreshToken(ctx)
```

---

### RefreshMtopToken

```go
func (api *XianyuAPI) RefreshMtopToken(ctx context.Context)
```

刷新 `_m_h5_tk` Cookie。通过调用 `mtop.taobao.idlehome.home.webpc.feed/1.0` 触发服务端重新签发。

**使用场景**: 验证码完成后，原有的 `_m_h5_tk` 可能已失效，需刷新后才能继续调用 mtop API

**注意**: 该方法不返回错误，刷新失败仅记录日志

---

### QrcodeLogin

```go
func QrcodeLogin(cfg QrcodeLoginConfig) (*XianyuAPI, error)
```

扫码登录完整流程。

**完整流程**:
1. `BuildInitialCookies()` — 构建初始 Cookie（含 tfstk）
2. 生成二维码 URL 并展示
3. 轮询扫码状态（`mtop.taobao.h5.mtaobao.login.qr.check/1.0`）
4. 扫码成功后获取登录 Cookie（`mtop.taobao.h5.mtaobao.login.qr.h5login/1.0`）
5. 返回已登录的 `XianyuAPI` 实例

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| cfg.PollInterval | `time.Duration` | `3s` | 轮询扫码状态间隔 |
| cfg.Timeout | `time.Duration` | `120s` | 扫码超时时间 |
| cfg.ShowQR | `bool` | `false` | 是否在终端打印二维码 |

**返回**: 登录成功的 `*XianyuAPI` 实例 + 错误信息

**错误情况**:
- `qr login timed out` — 扫码超时
- `qr login cancelled` — 用户取消扫码
- `build initial cookies: ...` — 初始 Cookie 构建失败

**注意事项**:
- 扫码登录可能触发风控 `FAIL_SYS_USER_VALIDATE`，建议使用手动 Cookie+Token 方式
- 需要 Node.js 环境生成 tfstk（`BuildInitialCookies` 依赖）

**示例**:
```go
api, err := apis.QrcodeLogin(apis.QrcodeLoginConfig{
    PollInterval: 3 * time.Second,
    Timeout:      120 * time.Second,
    ShowQR:       true,
})
```

---

### BuildInitialCookies

```go
func BuildInitialCookies() (*http.Client, map[string]string, error)
```

构建初始 Cookie（含 tfstk），用于扫码登录前置步骤。

**返回**:
- `*http.Client`: 已设置 Cookie 的 HTTP 客户端
- `map[string]string`: Cookie 字典
- `error`: tfstk 生成失败时的错误

**注意**: 需要 Node.js 环境执行 `assets/tfstk.js` 生成 tfstk 值

---

### GetItemInfo

```go
func (api *XianyuAPI) GetItemInfo(ctx context.Context, itemID string) (map[string]any, error)
```

获取闲鱼商品详情。

**对应 API**: `mtop.taobao.idle.pc.detail/1.0`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| itemID | `string` | 闲鱼商品 ID（纯数字，如 `"1058221493160"`） |

**返回**: 商品详情 JSON + 错误信息

**返回值结构**（关键字段）:
```json
{
  "ret": ["SUCCESS::调用成功"],
  "data": {
    "itemDO": {
      "itemId": "1058221493160",
      "title": "商品标题",
      "desc": "商品描述",
      "price": "99.00",
      "originalPrice": "199.00",
      "picList": [...],
      "sellerUserId": "1088389759",
      "sellerNickName": "卖家昵称"
    }
  }
}
```

**示例**:
```go
info, err := api.GetItemInfo(ctx, "1058221493160")
if err != nil {
    log.Fatal(err)
}
data := info["data"].(map[string]any)
itemDO := data["itemDO"].(map[string]any)
fmt.Println(itemDO["title"], itemDO["price"])
```

---

### GetPublicChannel

```go
func (api *XianyuAPI) GetPublicChannel(ctx context.Context, title string, images []model.ImageInfo) (map[string]any, error)
```

获取商品推荐标签和分类建议，用于商品发布时自动推荐属性。

**对应 API**: `mtop.taobao.idle.kgraph.property.recommend/2.0`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| title | `string` | 商品标题/描述 |
| images | `[]model.ImageInfo` | 已上传的图片信息列表（需先调用 `UploadMedia` 获取） |

**返回**: 推荐结果 JSON + 错误信息

**返回值结构**（关键字段）:
```json
{
  "data": {
    "cardList": [
      {
        "cardData": {
          "propertyName": "品牌",
          "propertyId": "123",
          "valuesList": [
            {"catName": "Apple", "isClicked": true, "channelCatId": "456", "tbCatId": "789"}
          ]
        }
      }
    ],
    "categoryPredictResult": {
      "catId": "50012345",
      "catName": "手机",
      "channelCatId": "12345",
      "tbCatId": "1512"
    }
  }
}
```

---

### GetDefaultLocation

```go
func (api *XianyuAPI) GetDefaultLocation(ctx context.Context) (map[string]any, error)
```

获取默认地理位置信息，用于商品发布时填写发货地址。

**对应 API**: `mtop.taobao.idle.local.poi.get/1.0`

**返回**: 位置信息 JSON + 错误信息

**返回值结构**（关键字段）:
```json
{
  "data": {
    "commonAddresses": [
      {
        "area": "余杭区",
        "city": "杭州市",
        "prov": "浙江省",
        "divisionId": "330110",
        "longitude": 120.0,
        "latitude": 30.0,
        "poiId": "B123456",
        "poi": "某某小区"
      }
    ]
  }
}
```

---

### PublishItem

```go
func (api *XianyuAPI) PublishItem(ctx context.Context, images []string, desc string, price *model.Price, ds model.DeliverySettings) (map[string]any, error)
```

发布闲鱼商品。内部自动执行完整发布流程。

**对应 API**: `mtop.idle.pc.idleitem.publish/1.0`

**完整流程**:
1. 逐张上传图片（调用 `UploadMedia`）
2. 获取推荐标签和分类（调用 `GetPublicChannel`）
3. 获取默认地址（调用 `GetDefaultLocation`）
4. 构建发布请求体并提交

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| images | `[]string` | 本地图片文件路径列表（最多 9 张，支持 PNG/JPG/JPEG/GIF） |
| desc | `string` | 商品标题和描述 |
| price | `*model.Price` | 价格信息，`nil` 使用系统默认定价 |
| ds | `model.DeliverySettings` | 配送设置 |

**返回**: 发布结果 JSON + 错误信息

**错误情况**:
- `upload image xxx: ...` — 图片上传失败（文件不存在、格式不支持等）
- `get public channel: ...` — 推荐标签获取失败
- `get default location: ...` — 地址获取失败

**示例**:
```go
result, err := api.PublishItem(ctx,
    []string{"./product1.jpg", "./product2.jpg"},
    "九成新机械键盘 自用半年 功能完好",
    &model.Price{CurrentPrice: 299.0, OriginalPrice: 599.0},
    model.DeliverySettings{Choice: "包邮"},
)
```

---

### UploadMedia

```go
func (api *XianyuAPI) UploadMedia(ctx context.Context, filePath string) (*UploadMediaResult, error)
```

上传媒体文件（图片）到闲鱼服务器。

**上传地址**: `https://stream-upload.goofish.com/api/upload.api`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| filePath | `string` | 本地文件路径（支持 PNG/JPG/JPEG/GIF） |

**返回**:

```go
type UploadMediaResult struct {
    URL    string // 图片访问 URL（如 https://img.alicdn.com/imgextra/...）
    Width  int    // 图片宽度（像素）
    Height int    // 图片高度（像素）
}
```

**错误情况**:
- `open file xxx: ...` — 文件不存在或无权限
- `stat file: ...` — 文件信息读取失败
- `do upload request: ...` — 网络请求失败
- `decode upload response: ...` — 响应解析失败

**示例**:
```go
result, err := api.UploadMedia(ctx, "./photo.png")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("URL: %s, 尺寸: %dx%d\n", result.URL, result.Width, result.Height)
```

---

### ConfirmShipping

```go
func (api *XianyuAPI) ConfirmShipping(ctx context.Context, orderID string) (map[string]any, error)
```

自动确认发货。适用于虚拟商品自动发货场景，确认后买家款项将到账。

**对应 API**: `mtop.taobao.idle.logistic.consign.dummy/1.0`

**与 Python 版对应**: `ConfirmShippingService.auto_confirm`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| orderID | `string` | 订单 ID |

**返回**: 确认结果 JSON + 错误信息

**成功响应**:
```json
{
  "ret": ["SUCCESS::调用成功"],
  "data": { ... }
}
```

**错误情况**:
- `confirm shipping: ...` — 请求失败（订单不存在、无权限等）

**示例**:
```go
result, err := api.ConfirmShipping(ctx, "202606162300136230143293342")
if err != nil {
    log.Fatal(err)
}
// 检查是否成功
if ret, ok := result["ret"].([]any); ok && len(ret) > 0 {
    if retStr, ok := ret[0].(string); ok && strings.Contains(retStr, "SUCCESS") {
        fmt.Println("确认发货成功")
    }
}
```

---

## ws — WebSocket 实时通信

`import "github.com/cv-cat/xianyuapis/pkg/ws"`

### XianyuWS 结构体

闲鱼 WebSocket 客户端，封装连接、注册、心跳、消息收发。

```go
type XianyuWS struct { /* 内部字段 */ }
```

**内部机制**:
- 使用 `nhooyr.io/websocket` 库（禁用压缩，服务端不协商 `permessage-deflate`）
- 双心跳机制:
  - LWP 心跳: 每 15 秒发送 `{"lwp":"/!"}` 消息
  - WebSocket PING: 每 30 秒发送 PING 帧（与 Python aiohttp `heartbeat=30` 对齐）
- 请求-响应配对: `pending map[string]chan map[string]any`，通过 mid 匹配
- 写入互斥: `writeMu sync.Mutex` 保护并发写（nhooyr.io/websocket 不支持并发写）
- 连接状态: `connected bool`，由 `IsAlive()` 查询

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

使用已有的 API 实例创建 WebSocket 客户端（**推荐**，共享登录态和 Cookie）。

| 参数 | 类型 | 说明 |
|------|------|------|
| api | `*apis.XianyuAPI` | 已初始化的 API 实例 |

**返回**: `*XianyuWS` 实例 + 错误信息

**示例**:
```go
api, _ := apis.New(cookies, "")
wsClient, _ := ws.NewWithAPI(api)
```

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

使用指定的 accessToken 建立 WebSocket 连接。

**完整流程**:
1. 获取 Token（如为空则调用 `api.GetToken()`）
2. 建立 WebSocket 连接到 `wss://wss-goofish.dingtalk.com/`
3. 启动 `recvLoop` goroutine（消息接收循环，先于 /reg）
4. 启动 `pingLoop` goroutine（WebSocket PING，每 30 秒）
5. 发送 `/reg` 注册并等待响应（5 秒超时）
6. 发送 `/r/SyncStatus/ackDiff` 同步状态
7. 冷却 3 秒（防止 IM 流控错误 400600001）
8. 启动 `heartbeat` goroutine（LWP 心跳，每 15 秒）
9. 标记 `connected = true`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 控制上下文 |
| token | `string` | accessToken，为空则自动获取 |

**返回**: 连接错误信息

**错误情况**:
- `ws: get token: ...` — Token 获取失败
- `ws: dial: ...` — WebSocket 连接失败（网络问题、Cookie 过期等）
- `ws: sync: ...` — 同步状态发送失败

**注意事项**:
- Token 中的 `deviceId` 必须与 `XianyuAPI` 的 `deviceID` 一致
- `/reg` 超时不 Fatal，会继续尝试后续步骤
- 连接后必须调用 `Start()` 进入消息接收循环

**示例**:
```go
// 自动获取 Token
err := wsClient.ConnectWithToken(ctx, "")

// 手动传入 Token（避免风控）
err := wsClient.ConnectWithToken(ctx, "oauth_k1:6SX/feDQW01ApS469NEyBlDRh/HzAQlTM4L/0QTNk0ep...")
```

---

### Start

```go
func (ws *XianyuWS) Start() error
```

启动消息接收循环（**阻塞**）。在 `Connect` / `ConnectWithToken` 之后调用。

**返回**: 连接关闭时的错误信息
- `context.Canceled` — 正常退出（调用了 `Stop()` 或 context 被取消）
- 其他错误 — 连接异常断开

**内部机制**:
- 等待 `recvLoop` goroutine 退出
- `recvLoop` 中每次 Read 设置 60 秒超时，确保 `Stop()` 后能快速退出

**示例**:
```go
err := wsClient.Start()
if err != nil && err != context.Canceled {
    log.Printf("连接异常断开: %v", err)
}
```

---

### Stop

```go
func (ws *XianyuWS) Stop()
```

停止 WebSocket 客户端。

**执行顺序**（重要）:
1. 标记 `connected = false`
2. 关闭 WebSocket 连接（使 `recvLoop` 中的 `Read` 立即返回）
3. 取消 context（停止心跳、Token 刷新等 goroutine）

**注意**: 必须先关闭连接再取消 context，否则 `Read` 会阻塞直到 60 秒超时

---

### IsAlive

```go
func (ws *XianyuWS) IsAlive() bool
```

返回 WebSocket 连接是否存活。线程安全。

**状态变更时机**:
- `true`: `ConnectWithToken` 成功后
- `false`: `recvLoop` 读取出错 / `Stop()` 被调用

**用途**: 判断是否需要重连

**示例**:
```go
if !wsClient.IsAlive() {
    fmt.Println("连接断开，尝试重连...")
    wsClient.ConnectWithToken(ctx, "")
}
```

---

### SetMessageHandler

```go
func (ws *XianyuWS) SetMessageHandler(h MessageHandler)
```

设置消息处理回调函数。线程安全。

| 参数 | 类型 | 说明 |
|------|------|------|
| h | `MessageHandler` | 消息回调函数，签名为 `func(m *msg.Message)` |

**回调函数签名**:
```go
type MessageHandler func(m *msg.Message)
```

**示例**:
```go
wsClient.SetMessageHandler(func(m *msg.Message) {
    switch {
    case m.IsText():
        fmt.Printf("[文字] %s(%s): %s\n", m.SenderName, m.SenderID, m.Content)
    case m.IsImage():
        fmt.Printf("[图片] %s(%s): %s (%dx%d)\n",
            m.SenderName, m.SenderID, m.ImageURL, m.ImageWidth, m.ImageHeight)
    }
})
```

---

### StartTokenRefresher

```go
func (ws *XianyuWS) StartTokenRefresher(interval ...time.Duration)
```

启动后台 Token 刷新 goroutine。与 Python 版 `user_alive()` 对齐。

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| interval | `...time.Duration` | `10m` | 刷新间隔，0 或不传使用默认值 |

**内部机制**:
- 使用 `time.Ticker` 定期调用 `api.RefreshToken()`
- 刷新失败仅记录日志，不中断
- context 取消时自动退出

**示例**:
```go
// 默认 10 分钟刷新
wsClient.StartTokenRefresher()

// 自定义 5 分钟刷新
wsClient.StartTokenRefresher(5 * time.Minute)

// 自定义 30 分钟刷新
wsClient.StartTokenRefresher(30 * time.Minute)
```

---

### SendText

```go
func (ws *XianyuWS) SendText(ctx context.Context, cid, toID, text string) error
```

发送文字消息。

**对应 LWP**: `/r/MessageSend/sendByReceiverScope`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| cid | `string` | 会话 ID（**不含** `@goofish` 后缀，函数内部自动添加） |
| toID | `string` | 接收方用户 ID（**不含** `@goofish` 后缀） |
| text | `string` | 消息文本内容 |

**返回**: 发送错误信息

**内部机制**:
1. 构建 `{"contentType":1,"text":{"text":"xxx"}}`
2. JSON 序列化 → Base64 编码
3. 封装为 LWP 消息发送

**示例**:
```go
err := wsClient.SendText(ctx, "conversation_id", "2218356024224", "你好，请问还在吗？")
```

---

### SendImage

```go
func (ws *XianyuWS) SendImage(ctx context.Context, cid, toID, imageURL string, width, height int) error
```

发送图片消息。

**对应 LWP**: `/r/MessageSend/sendByReceiverScope`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| cid | `string` | 会话 ID（**不含** `@goofish` 后缀） |
| toID | `string` | 接收方用户 ID（**不含** `@goofish` 后缀） |
| imageURL | `string` | 图片 URL（需先通过 `UploadMedia` 上传获取） |
| width | `int` | 图片宽度（像素） |
| height | `int` | 图片高度（像素） |

**返回**: 发送错误信息

**示例**:
```go
// 先上传图片
uploadResult, _ := api.UploadMedia(ctx, "./photo.png")

// 再发送图片消息
err := wsClient.SendImage(ctx, "conversation_id", "2218356024224",
    uploadResult.URL, uploadResult.Width, uploadResult.Height)
```

---

### ListAllConversations

```go
func (ws *XianyuWS) ListAllConversations(ctx context.Context, cid string) ([]*ConversationMessage, error)
```

获取与指定用户的全部历史聊天记录。

**对应 LWP**: `/r/MessageManager/listUserMessages`

**内部机制**:
- 建立临时 WebSocket 连接（不影响主连接）
- 分页获取，每页 20 条，自动翻页直到 `hasMore=0`
- 获取完毕后自动关闭临时连接

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| cid | `string` | 会话 ID |

**返回**: 历史消息列表 + 错误信息

```go
type ConversationMessage struct {
    SenderID   string `json:"send_user_id"`   // 发送者用户 ID
    SenderName string `json:"send_user_name"` // 发送者昵称
    Message    any    `json:"message"`         // 解码后的消息内容（map[string]any）
}
```

**示例**:
```go
messages, err := wsClient.ListAllConversations(ctx, "conversation_id")
for _, msg := range messages {
    fmt.Printf("%s(%s): %v\n", msg.SenderName, msg.SenderID, msg.Message)
}
```

---

### CreateChat

```go
func (ws *XianyuWS) CreateChat(ctx context.Context, toID, itemID string) error
```

创建新会话，可关联商品 ID。

**对应 LWP**: `/r/SingleChatConversation/create`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| toID | `string` | 目标用户 ID（**不含** `@goofish` 后缀） |
| itemID | `string` | 关联商品 ID，为空使用默认值 |

**返回**: 创建错误信息

**示例**:
```go
// 创建关联商品的会话
err := wsClient.CreateChat(ctx, "2218356024224", "1058221493160")

// 创建不关联商品的会话
err := wsClient.CreateChat(ctx, "2218356024224", "")
```

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
    SenderID       string      // 发送者用户 ID（如 "2218356024224"）
    SenderName     string      // 发送者昵称（如 "冬浩网创"）
    Content        string      // 消息文本内容（文字/卡片/转账/位置均填充此字段）
    MessageType    MessageType // 消息类型枚举
    ConversationID string      // 会话 ID（不含 @goofish 后缀）
    ImageURL       string      // 图片 URL（仅图片消息，多图用逗号分隔）
    ImageWidth     int         // 首张图片宽度（像素）
    ImageHeight    int         // 首张图片高度（像素）
    Timestamp      time.Time   // 消息时间
    Raw            any         // 原始消息数据（调试用，类型为 map[string]any）
}
```

---

### Message 方法

| 方法 | 签名 | 说明 |
|------|------|------|
| IsText | `func (m *Message) IsText() bool` | 判断是否为文字消息（`MessageType == 1`） |
| IsImage | `func (m *Message) IsImage() bool` | 判断是否为图片消息（`MessageType == 2`） |
| String | `func (mt MessageType) String() string` | 消息类型转字符串（`"text"` / `"image"` / `"audio"` / `"unknown"`） |

---

### NewTextMessage

```go
func NewTextMessage(senderID, senderName, content, conversationID string) *Message
```

创建文字消息。`MessageType` 自动设为 `MessageTypeText`。

| 参数 | 类型 | 说明 |
|------|------|------|
| senderID | `string` | 发送者用户 ID |
| senderName | `string` | 发送者昵称 |
| content | `string` | 消息文本内容 |
| conversationID | `string` | 会话 ID |

---

### NewImageMessage

```go
func NewImageMessage(senderID, senderName, imageURL, conversationID string, width, height int) *Message
```

创建图片消息。`MessageType` 自动设为 `MessageTypeImage`。

| 参数 | 类型 | 说明 |
|------|------|------|
| senderID | `string` | 发送者用户 ID |
| senderName | `string` | 发送者昵称 |
| imageURL | `string` | 图片 URL |
| conversationID | `string` | 会话 ID |
| width | `int` | 图片宽度 |
| height | `int` | 图片高度 |

---

### 消息解析支持的 contentType

闲鱼 WebSocket 推送消息的 `contentType` 字段决定消息类型:

| contentType | 类型 | Content 字段内容 | ImageURL 字段 |
|:-----------:|------|-----------------|--------------|
| 1 | 文字消息 | `text.text` 的值 | 空 |
| 2 | 图片消息 | 空 | `image.pics[].url`，多图逗号分隔 |
| 3 | 语音消息 | `[语音消息]` | 空 |
| 7 | 商品卡片 | `[我想要] 标题 价格 (id:xxx)` | 商品主图 URL |
| 17 | 转账消息 | `[转账] ¥金额 (交易号:xxx)` | 空 |
| 30 | 位置消息 | `[位置] 经度:xxx 纬度:xxx` | 空 |

**编码格式说明**:

闲鱼消息内容有两种编码格式，`decodeContent` 自动识别:

- **格式1** (base64): `msg["1"]["6"]["3"]["1"]` 为 base64 字符串，解码后得到 JSON
  - 文字消息常用此格式
  - 解码后结构: `{"contentType":1,"text":{"text":"xxx"}}`

- **格式2** (直接 JSON): `msg["1"]["6"]["3"]["5"]` 为直接 JSON 字符串
  - 图片/卡片/转账/位置等常用此格式
  - 结构: `{"contentType":2,"image":{"pics":[...]}}`

- `msg["1"]["6"]["3"]["4"]` 为 contentType 整数值（格式2的辅助字段）
- `msg["1"]["6"]["3"]["2"]` 为 reminderContent（消息摘要，如 `[图片]`、`[尴尬]`）

**消息解析流程**:
1. 从 `msg["1"]["10"]` 提取 senderID、senderName、reminderContent
2. 优先尝试 `msg["1"]["6"]["3"]["5"]` 直接 JSON 解析
3. 回退到 `msg["1"]["6"]["3"]["1"]` base64 解码
4. 如果都解析不出内容，使用 reminderContent 作为文本

---

## model — 数据模型

`import "github.com/cv-cat/xianyuapis/pkg/model"`

### Price

```go
type Price struct {
    CurrentPrice  float64 // 当前售价（单位: 元，如 299.0）
    OriginalPrice float64 // 原价（单位: 元，如 599.0）
}
```

**注意**: 内部发布时会自动转换为分（`priceInCent` / `origPriceInCent`），0 值字段不发送

---

### DeliverySettings

```go
type DeliverySettings struct {
    Choice        string  // 配送方式
    PostPrice     float64 // 运费（仅 "一口价" 时有效，单位: 元）
    CanSelfPickup bool    // 是否支持自提
}
```

**Choice 可选值**:

| 值 | 说明 | 对应 API 参数 |
|----|------|-------------|
| `"包邮"` | 卖家承担运费 | `canFreeShipping=true, supportFreight=true` |
| `"按距离计费"` | 按距离计算运费 | `supportFreight=true, templateId="-100"` |
| `"一口价"` | 固定运费 | `supportFreight=true, templateId="0", postPriceInCent=xxx` |
| `"无需邮寄"` | 虚拟商品 | `templateId="0"` |

---

### ImageInfo

```go
type ImageInfo struct {
    URL    string // 图片 URL（上传后获得）
    Width  int    // 图片宽度（像素）
    Height int    // 图片高度（像素）
}
```

---

### QrcodeLoginConfig

```go
type QrcodeLoginConfig struct {
    PollInterval time.Duration // 轮询扫码状态间隔，默认 3s
    Timeout      time.Duration // 扫码超时时间，默认 120s
    ShowQR       bool          // 是否在终端打印二维码
}
```

---

## util — 工具函数

`import "github.com/cv-cat/xianyuapis/pkg/util"`

> 所有函数均为纯函数或无副作用的生成函数，可被外部项目直接 import 复用。

### GenerateSign

```go
func GenerateSign(timestamp, token, data string) string
```

生成闲鱼 mtop API 签名。

**签名公式**: `MD5(token + "&" + timestamp + "&" + appKey + "&" + data)`

其中 `appKey = "34839810"` 为闲鱼平台固定标识。

| 参数 | 类型 | 说明 |
|------|------|------|
| timestamp | `string` | 毫秒级时间戳（如 `"1718000000000"`） |
| token | `string` | 从 `_m_h5_tk` Cookie 下划线前提取（如 `"cf64b07dc62279e55bc0a030719d59a9"`） |
| data | `string` | 请求数据 JSON 字符串 |

**返回**: 32 位小写十六进制 MD5 签名字符串

**示例**:
```go
sign := util.GenerateSign("1718000000000", "cf64b07dc62279e55bc0a030719d59a9", `{"appKey":"444e9908a51d1cb236a27862abc769c9"}`)
fmt.Println(sign) // "a1b2c3d4e5f6..."
```

---

### Base64DecodeUTF8

```go
func Base64DecodeUTF8(data string) (string, error)
```

将 Base64 字符串解码为 UTF-8 字符串。

**内部机制**:
1. 清理输入中的非 Base64 字符（只保留 `A-Za-z0-9+/=`）
2. 标准解码

| 参数 | 类型 | 说明 |
|------|------|------|
| data | `string` | Base64 编码的字符串 |

**返回**: 解码后的 UTF-8 字符串 + 错误信息

---

### Decrypt

```go
func Decrypt(data string) (string, error)
```

解密闲鱼加密消息。

**解码流程**: `base64 → 自定义 msgpack 解码 → JSON 字符串`

**自定义 msgpack 解码器**:
- 闲鱼的 msgpack 数据使用整数键（1, 2, 3...）
- 标准 `msgpack/v5` 库解码到 `map[string]any` 时遇到整数键会报错 `invalid code=1`
- 自定义解码器自动将整数键转为字符串键（如 `1 → "1"`），与 Python 版 `MessagePackDecoder` 对齐

**支持的 msgpack 类型**:

| 类型 | 代码范围 | Go 类型 |
|------|---------|---------|
| positive fixint | `0x00-0x7f` | `int64` |
| negative fixint | `0xe0-0xff` | `int64` |
| fixmap | `0x80-0x8f` | `map[string]any` |
| fixarray | `0x90-0x9f` | `[]any` |
| fixstr | `0xa0-0xbf` | `string` |
| nil | `0xc0` | `nil` |
| bool | `0xc2/0xc3` | `bool` |
| float32/64 | `0xca/0xcb` | `float64` |
| uint/int 8-64 | `0xcc-0xd3` | `int64` |
| str 8/16/32 | `0xd9-0xdb` | `string` |
| bin 8/16/32 | `0xc4-0xc6` | `[]byte` |
| array 16/32 | `0xdc-0xdd` | `[]any` |
| map 16/32 | `0xde-0xdf` | `map[string]any` |

| 参数 | 类型 | 说明 |
|------|------|------|
| data | `string` | Base64 编码的加密消息 |

**返回**: JSON 字符串 + 错误信息

---

### GenerateMid

```go
func GenerateMid() string
```

生成消息 ID。

**格式**: `"毫秒时间戳 0"`

**示例**: `"7381748291023 0"`

---

### GenerateUUID

```go
func GenerateUUID() string
```

生成请求标识 UUID。

**格式**: `"-毫秒时间戳随机数"`

**示例**: `"-1718000000000123456"`

---

### GenerateDeviceID

```go
func GenerateDeviceID(userID string) string
```

生成设备 ID。

**格式**: `8-4-4-4-12位随机字符-userID`

**字符集**: `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz`

| 参数 | 类型 | 说明 |
|------|------|------|
| userID | `string` | 用户 ID（unb） |

**返回**: 设备 ID 字符串

**示例**:
```go
deviceID := util.GenerateDeviceID("1088389759")
// "MxELdSFn-GrTE-4XT1-Bgh2-gajPurajZ4re-1088389759"
```

**重要**: 浏览器获取 Token 时必须使用相同的 deviceID，否则 WebSocket `/reg` 返回 401

---

### GenerateTFstk

```go
func GenerateTFstk(scriptPath string) (string, error)
```

通过 Node.js 子进程生成 tfstk 值，用于构建初始 Cookie。

**前置条件**: 需要安装 Node.js

| 参数 | 类型 | 说明 |
|------|------|------|
| scriptPath | `string` | `tfstk.js` 脚本路径 |

**返回**: tfstk 字符串 + 错误信息

---

### ParseCookies

```go
func ParseCookies(cookieStr string) map[string]string
```

解析 Cookie 字符串为 map。

**输入格式**: `"key1=val1; key2=val2"`（浏览器 F12→Network→请求头中的 Cookie 值）

**示例**:
```go
cookies := util.ParseCookies("unb=1088389759; cookie2=283d60b38b047c43ca907174a44d2561")
// map[string]string{"unb": "1088389759", "cookie2": "283d60b38b047c43ca907174a44d2561"}
```

---

### BuildCookieString

```go
func BuildCookieString(cookies map[string]string) string
```

将 Cookie map 构建为字符串。

**输出格式**: `"key1=val1; key2=val2"`

---

### GetCookieFromJar

```go
func GetCookieFromJar(jar http.CookieJar, domain, name string) string
```

从 CookieJar 中读取指定域名的 Cookie 值。

**域名转换**: 自动将 `.goofish.com` 转为 `www.goofish.com` 查询（RFC 6265 规则）

| 参数 | 类型 | 说明 |
|------|------|------|
| jar | `http.CookieJar` | CookieJar 实例 |
| domain | `string` | 域名（如 `".goofish.com"` 或 `"www.goofish.com"`） |
| name | `string` | Cookie 名称 |

**返回**: Cookie 值，不存在返回空字符串

---

### SetCookieToJar

```go
func SetCookieToJar(jar http.CookieJar, domain, name, value, path string)
```

向 CookieJar 写入指定域名的 Cookie。

**域名转换**: 自动将 `.goofish.com` 转为 `www.goofish.com` 设置

| 参数 | 类型 | 说明 |
|------|------|------|
| jar | `http.CookieJar` | CookieJar 实例 |
| domain | `string` | 域名 |
| name | `string` | Cookie 名称 |
| value | `string` | Cookie 值 |
| path | `string` | Cookie 路径 |

---

## 常量

| 常量 | 值 | 说明 |
|------|-----|------|
| `wsBaseURL` | `wss://wss-goofish.dingtalk.com/` | WebSocket 连接地址 |
| `appKey` | `34839810` | mtop API 固定 appKey（签名用） |
| `msgAppKey` | `444e9908a51d1cb236a27862abc769c9` | 消息服务 appKey（Token 获取和 /reg 用） |
| `UA` | `Mozilla/5.0 (Windows NT 10.0; Win64; x64) ...` | 统一 User-Agent |
| `regTimeout` | `5s` | /reg 注册响应超时时间 |
| `regCooldown` | `3s` | /reg 注册后冷却时间（防止流控 400600001） |
| `heartbeatInterval` | `15s` | LWP 心跳间隔 |
| `wsPingInterval` | `30s` | WebSocket PING 间隔（与 Python aiohttp heartbeat=30 对齐） |
| `tokenRefreshInterval` | `10m` | Token 刷新默认间隔 |

---

## 错误处理

### 常见错误及解决方案

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| `cookies must contain 'unb' field` | Cookie 字典缺少 `unb` | 确保从浏览器完整复制 Cookie |
| `_m_h5_tk cookie not found` | Cookie 中缺少 `_m_h5_tk` | 重新登录获取 Cookie |
| `FAIL_SYS_USER_VALIDATE` / `RGV587_ERROR::SM` | 触发风控验证码 | 使用手动 Cookie+Token 方式绕过 |
| `device id or appkey is not equal` | Token 和 /reg 的 deviceID 不一致 | 确保浏览器获取 Token 时使用相同 deviceID |
| `ws: dial: ...` | WebSocket 连接失败 | 检查网络、Cookie 是否过期 |
| `invalid code=1 decoding string/bytes length` | msgpack 整数键问题 | 已通过自定义解码器解决 |
| `confirm shipping: ...` | 确认发货失败 | 检查订单 ID 是否正确、是否有权限 |

### 风控规避建议

1. **优先使用手动 Cookie+Token 方式**，避免扫码登录触发风控
2. Cookie 必须包含 HttpOnly 字段（`cookie2`、`sgcookie`），从 F12→Network→请求头中复制
3. 多次失败后等待 5-10 分钟再重试
4. Token 获取后尽快使用，避免过期

---

## 完整示例

### 手动 Cookie+Token 登录 + WebSocket 消息收发

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/cv-cat/xianyuapis/pkg/apis"
    "github.com/cv-cat/xianyuapis/pkg/msg"
    "github.com/cv-cat/xianyuapis/pkg/util"
    "github.com/cv-cat/xianyuapis/pkg/ws"
)

func main() {
    // 1. 解析手动输入的 Cookie
    cookieStr := "t=ec6cd229...; unb=1088389759; cookie2=283d60b38b...; sgcookie=E100KNAU1m..."
    cookies := util.ParseCookies(cookieStr)

    // 2. 创建 API 实例
    api, err := apis.New(cookies, "")
    if err != nil {
        panic(err)
    }

    // 3. 创建 WebSocket 客户端（共享 API 实例）
    wsClient, err := ws.NewWithAPI(api)
    if err != nil {
        panic(err)
    }

    // 4. 设置消息处理回调
    wsClient.SetMessageHandler(func(m *msg.Message) {
        switch {
        case m.IsText():
            fmt.Printf("[文字] %s(%s): %s\n", m.SenderName, m.SenderID, m.Content)
            // 自动回复
            wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, "收到: "+m.Content)
        case m.IsImage():
            fmt.Printf("[图片] %s(%s): %s (%dx%d)\n",
                m.SenderName, m.SenderID, m.ImageURL, m.ImageWidth, m.ImageHeight)
        }
    })

    // 5. 启动 Token 自动刷新（自定义 5 分钟间隔）
    wsClient.StartTokenRefresher(5 * time.Minute)

    // 6. 使用手动 Token 连接（避免风控）
    accessToken := "oauth_k1:6SX/feDQW01ApS469NEyBlDRh/HzAQlTM4L/0QTNk0ep..."
    if err := wsClient.ConnectWithToken(context.Background(), accessToken); err != nil {
        panic(err)
    }

    // 7. 优雅退出
    sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    defer wsClient.Stop()

    // 8. 阻塞等待退出信号
    <-sigCtx.Done()
    fmt.Println("正在退出...")
}
```

### 商品发布

```go
// 上传图片 + 发布商品
result, err := api.PublishItem(ctx,
    []string{"./product1.jpg", "./product2.jpg"},
    "九成新机械键盘 自用半年 功能完好",
    &model.Price{CurrentPrice: 299.0, OriginalPrice: 599.0},
    model.DeliverySettings{Choice: "包邮"},
)
```

### 自动确认发货

```go
// 收到转账消息后自动确认发货
result, err := api.ConfirmShipping(ctx, orderID)
if ret, ok := result["ret"].([]any); ok && len(ret) > 0 {
    if retStr, ok := ret[0].(string); ok && strings.Contains(retStr, "SUCCESS") {
        fmt.Println("确认发货成功，款项将到账")
    }
}
```

---

## open — 闲鱼开放平台 API SDK

`import "github.com/cv-cat/xianyuapis/pkg/open"`

> 源自 [XIE7654/goofish_api](https://github.com/XIE7654/goofish_api)，基于 app_key + app_secret 鉴权。
> 与 `pkg/apis`（逆向 API，Cookie 鉴权）互补，适用于商家商品/订单管理。

### Client 结构体

闲鱼开放平台客户端，封装签名计算、请求构建、响应解析。

```go
type Client struct {
    User  *UserService
    Good  *GoodService
    Order *OrderService
    Other *OtherService
}
```

**内部机制**:
- 签名算法: `MD5(app_key + "," + body_md5 + "," + timestamp + "," + [seller_id + ","] + app_secret)`
- 时间戳: 秒级
- JSON 序列化: 紧凑格式（无空格），否则签名不匹配
- 自动移除 nil 字段（递归，含 nil 指针检测）

**线程安全**: 同一 Client 实例可被多个 goroutine 并发调用。

---

### NewClient

```go
func NewClient(appKey, appSecret string, opts ...Option) *Client
```

创建开放平台客户端。

| 参数 | 类型 | 说明 |
|------|------|------|
| appKey | `string` | 应用 Key（开放平台申请） |
| appSecret | `string` | 应用 Secret |
| opts | `...Option` | 可选配置 |

**可选配置**:

| 选项 | 说明 |
|------|------|
| `WithSellerID(id)` | 设置商家 ID（商务对接模式） |
| `WithDomain(url)` | 自定义 API 域名（默认 `https://open.goofish.pro`） |
| `WithHTTPClient(c)` | 自定义 HTTP 客户端 |
| `WithDebug(true)` | 启用调试模式 |

**示例**:
```go
client := open.NewClient("app_key", "app_secret")
client := open.NewClient("app_key", "app_secret", open.WithSellerID("seller123"))
```

---

### ApiResponse

```go
type ApiResponse struct {
    Code    int             `json:"code"`    // 错误码，0 表示成功
    Message string          `json:"message"` // 提示信息
    Data    json.RawMessage `json:"data"`    // 原始业务数据
    Success bool            `json:"success"` // 是否成功
}
```

| 方法 | 签名 | 说明 |
|------|------|------|
| IsSuccess | `func (r *ApiResponse) IsSuccess() bool` | 判断是否成功（code == 0） |
| UnmarshalData | `func (r *ApiResponse) UnmarshalData(v any) error` | 解析 data 字段到结构体 |
| String | `func (r *ApiResponse) String() string` | 简洁字符串表示 |

---

### UserService — 用户模块

#### GetAuthorizeList

```go
func (s *UserService) GetAuthorizeList(ctx context.Context) (*ApiResponse, error)
```

查询已授权的闲鱼店铺列表。

**对应 API**: `POST /api/open/user/authorize/list`

---

### GoodService — 商品模块

#### GetProductCategoryList

```go
func (s *GoodService) GetProductCategoryList(ctx context.Context, itemBizType ItemBizType, spBizType *SpBizType, flashSaleType *FlashSaleType) (*ApiResponse, error)
```

查询商品类目。**对应 API**: `POST /api/open/product/category/list`

#### GetProductPvList

```go
func (s *GoodService) GetProductPvList(ctx context.Context, itemBizType ItemBizType, spBizType SpBizType, channelCatID string, subPropertyID *string) (*ApiResponse, error)
```

查询商品属性。**对应 API**: `POST /api/open/product/pv/list`

#### GetProductList

```go
func (s *GoodService) GetProductList(ctx context.Context, req *GetProductListRequest) (*ApiResponse, error)
```

查询商品列表。**对应 API**: `POST /api/open/product/list`

#### GetProductDetail

```go
func (s *GoodService) GetProductDetail(ctx context.Context, productID int64) (*ApiResponse, error)
```

查询商品详情。**对应 API**: `POST /api/open/product/detail`

#### GetProductSkuList

```go
func (s *GoodService) GetProductSkuList(ctx context.Context, productIDs []int64) (*ApiResponse, error)
```

查询商品规格。**对应 API**: `POST /api/open/product/sku/list`

#### CreateProduct

```go
func (s *GoodService) CreateProduct(ctx context.Context, productData any) (*ApiResponse, error)
```

创建商品（单个）。**对应 API**: `POST /api/open/product/create`

#### ProductBatchCreate

```go
func (s *GoodService) ProductBatchCreate(ctx context.Context, productList []any) (*ApiResponse, error)
```

批量创建商品（每批最多 50 个）。**对应 API**: `POST /api/open/product/batchCreate`

#### ProductPublish

```go
func (s *GoodService) ProductPublish(ctx context.Context, req *ProductPublishRequest) (*ApiResponse, error)
```

上架商品（异步）。**对应 API**: `POST /api/open/product/publish`

#### ProductDownShelf

```go
func (s *GoodService) ProductDownShelf(ctx context.Context, productID int64) (*ApiResponse, error)
```

下架商品。**对应 API**: `POST /api/open/product/downShelf`

#### ProductEdit

```go
func (s *GoodService) ProductEdit(ctx context.Context, productData any) (*ApiResponse, error)
```

编辑商品。**对应 API**: `POST /api/open/product/edit`

#### ProductEditStock

```go
func (s *GoodService) ProductEditStock(ctx context.Context, req *ProductEditStockRequest) (*ApiResponse, error)
```

编辑商品库存和价格。**对应 API**: `POST /api/open/product/edit/stock`

#### ProductDelete

```go
func (s *GoodService) ProductDelete(ctx context.Context, productID int64) (*ApiResponse, error)
```

删除商品。**对应 API**: `POST /api/open/product/delete`

---

### OrderService — 订单模块

#### GetOrderList

```go
func (s *OrderService) GetOrderList(ctx context.Context, req *GetOrderListRequest) (*ApiResponse, error)
```

查询订单列表。**对应 API**: `POST /api/open/order/list`

#### GetOrderDetail

```go
func (s *OrderService) GetOrderDetail(ctx context.Context, orderNo string) (*ApiResponse, error)
```

查询订单详情。**对应 API**: `POST /api/open/order/detail`

#### KamOrderList

```go
func (s *OrderService) KamOrderList(ctx context.Context, orderNo string) (*ApiResponse, error)
```

查询订单卡密列表。**对应 API**: `POST /api/open/order/kam/list`

#### OrderShip

```go
func (s *OrderService) OrderShip(ctx context.Context, req *OrderShipRequest) (*ApiResponse, error)
```

订单物流发货。**对应 API**: `POST /api/open/order/ship`

**必填字段**: OrderNo、WaybillNo、ExpressName、ExpressCode
**条件必填**: ShipDistrictID 或 (ShipProvName + ShipCityName + ShipAreaName)

---

### OtherService — 其他模块

#### GetExpressCompanies

```go
func (s *OtherService) GetExpressCompanies(ctx context.Context) (*ApiResponse, error)
```

获取快递公司列表。**对应 API**: `POST /api/open/express/companies`

---

### 枚举类型

| 枚举 | 类型 | 说明 |
|------|------|------|
| `ItemBizType` | `int` | 商品类型（Common=2, Inspected=0, InspectionBao=10, ...） |
| `SpBizType` | `int` | 行业类型（Mobile=1, Trend=2, HomeAppliance=3, ...） |
| `FlashSaleType` | `int` | 闲鱼特卖类型（LiQi=1, GuPin=2, ...） |
| `ProductStatus` | `int` | 商品状态（Status21=21, ...） |
| `SaleStatus` | `int` | 销售状态（OnSale=2, OffSale=3, ...） |
| `OrderStatus` | `int` | 订单状态（PendingPayment=11, PendingShipment=12, ...） |
| `RefundStatus` | `int` | 退款状态（Success=5, Rejected=6, ...） |

---

## search — 商品搜索爬虫

`import "github.com/cv-cat/xianyuapis/pkg/search"`

> 源自 [XIE7654/goofish_api/spider](https://github.com/XIE7654/goofish_api/tree/main/spider)，基于 Cookie 鉴权。
> 使用 `mtop.taobao.idlemtopsearch.pc.search` API，依赖 `pkg/apis.XianyuAPI`。

### Crawler 结构体

```go
type Crawler struct { /* 内部字段 */ }
```

### New

```go
func New(api *apis.XianyuAPI) *Crawler
```

创建搜索爬虫实例。

| 参数 | 类型 | 说明 |
|------|------|------|
| api | `*apis.XianyuAPI` | 已登录的 API 实例（需包含有效 Cookie） |

### Search

```go
func (c *Crawler) Search(ctx context.Context, req *Request) ([]*Item, error)
```

执行关键词搜索，返回单页结果。

**对应 API**: `mtop.taobao.idlemtopsearch.pc.search/1.0`

| 参数 | 类型 | 说明 |
|------|------|------|
| ctx | `context.Context` | 请求上下文 |
| req | `*Request` | 搜索请求参数 |

**Request 结构体**:

```go
type Request struct {
    Keyword     string // 搜索关键词（必需）
    PageNumber  int    // 页码，从 1 开始
    RowsPerPage int    // 每页数量，默认 30
    FromFilter  bool   // 是否来自筛选
    SortValue   string // 排序值（可选）
    SortField   string // 排序字段（可选）
}
```

### SearchAll

```go
func (c *Crawler) SearchAll(ctx context.Context, req *Request, maxPages int) ([]*Item, error)
```

搜索所有页（最多 maxPages 页），聚合结果。

### Item 结构体

```go
type Item struct {
    UserName  string // 卖家用户名
    Area      string // 地区
    SoldPrice string // 售价
    Title     string // 商品标题
    DetailURL string // 详情页 URL
    ItemID    string // 商品 ID
    CreatedAt string // 采集时间
}
```

**示例**:

```go
api, _ := apis.New(cookies, "")
crawler := search.New(api)

items, err := crawler.Search(ctx, &search.Request{
    Keyword:     "机械键盘",
    PageNumber:  1,
    RowsPerPage: 30,
})
for _, item := range items {
    fmt.Printf("%s | ¥%s | %s\n", item.Title, item.SoldPrice, item.DetailURL)
}
```
