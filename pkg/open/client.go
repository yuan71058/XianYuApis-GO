package open

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
)

// 默认域名（正式环境）。
const defaultDomain = "https://open.goofish.pro"

// Client 闲鱼开放平台客户端。
//
// 对应 Python 版 BaseClient + GoofishClient，使用 app_key + app_secret 鉴权。
// 内部封装签名计算、请求构建、响应解析，各功能模块通过组合方式挂载：
//
//   - User  : 店铺授权管理
//   - Good  : 商品 CRUD、类目、属性
//   - Order : 订单查询、发货、卡密
//   - Other : 快递公司查询
//
// 线程安全：同一 Client 实例可被多个 goroutine 并发调用。
type Client struct {
	appKey     string       // 应用 Key
	appSecret  string       // 应用 Secret
	sellerID   string       // 商家 ID（商务对接模式，可选）
	domain     string       // API 域名
	httpClient *http.Client // HTTP 客户端
	debug      bool         // 调试模式

	User  *UserService  // 用户模块
	Good  *GoodService  // 商品模块
	Order *OrderService // 订单模块
	Other *OtherService // 其他模块
}

// Option 客户端配置选项。
type Option func(*Client)

// WithSellerID 设置商家 ID（商务对接模式）。
func WithSellerID(sellerID string) Option {
	return func(c *Client) { c.sellerID = sellerID }
}

// WithDomain 设置 API 域名（默认 https://open.goofish.pro）。
func WithDomain(domain string) Option {
	return func(c *Client) { c.domain = strings.TrimRight(domain, "/") }
}

// WithHTTPClient 设置自定义 HTTP 客户端。
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithDebug 启用调试模式（打印请求和响应）。
func WithDebug(debug bool) Option {
	return func(c *Client) { c.debug = debug }
}

// NewClient 创建闲鱼开放平台客户端。
//
// 参数：
//   - appKey:    应用 Key（开放平台申请）
//   - appSecret: 应用 Secret（开放平台申请）
//   - opts:      可选配置（商家 ID、域名、HTTP 客户端、调试模式）
//
// 示例：
//
//	// 自研模式
//	client := open.NewClient("your_app_key", "your_app_secret")
//
//	// 商务对接模式
//	client := open.NewClient("app_key", "app_secret", open.WithSellerID("seller123"))
func NewClient(appKey, appSecret string, opts ...Option) *Client {
	c := &Client{
		appKey:     appKey,
		appSecret:  appSecret,
		domain:     defaultDomain,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}

	// 挂载功能模块
	c.User = &UserService{client: c}
	c.Good = &GoodService{client: c}
	c.Order = &OrderService{client: c}
	c.Other = &OtherService{client: c}

	return c
}

// sign 计算请求签名。
//
// 签名算法（与 Python 版 get_sign 对齐）：
//  1. body_md5 = MD5(body_json)
//  2. 自研模式：sign_str = app_key + "," + body_md5 + "," + timestamp + "," + app_secret
//     商务模式：sign_str = app_key + "," + body_md5 + "," + timestamp + "," + seller_id + "," + app_secret
//  3. sign = MD5(sign_str)
func (c *Client) sign(bodyJSON string, timestamp int64) string {
	// 1. body MD5
	h := md5.Sum([]byte(bodyJSON))
	bodyMD5 := hex.EncodeToString(h[:])

	// 2. 拼接签名串
	var signStr string
	if c.sellerID != "" {
		signStr = fmt.Sprintf("%s,%s,%d,%s,%s", c.appKey, bodyMD5, timestamp, c.sellerID, c.appSecret)
	} else {
		signStr = fmt.Sprintf("%s,%s,%d,%s", c.appKey, bodyMD5, timestamp, c.appSecret)
	}

	// 3. 计算 MD5
	h2 := md5.Sum([]byte(signStr))
	return hex.EncodeToString(h2[:])
}

// request 执行 API 请求并返回解析后的响应。
//
// 对应 Python 版 BaseClient.request，关键点：
//   - 时间戳为秒级
//   - JSON 序列化使用紧凑格式（separators=(",", ":"），否则签名不匹配
//   - 递归移除值为 nil 的字段
//   - POST 请求，body 为 JSON 字符串
func (c *Client) request(ctx context.Context, path string, data any) (*ApiResponse, error) {
	// 1. 序列化 body（紧凑 JSON + 移除 nil 字段）
	cleaned := removeNilValues(data)
	bodyBytes, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("open: marshal request body: %w", err)
	}
	bodyJSON := string(bodyBytes)

	// 2. 计算签名
	timestamp := time.Now().Unix()
	sign := c.sign(bodyJSON, timestamp)

	// 3. 构建完整 URL
	reqURL := fmt.Sprintf("%s%s?appid=%s&timestamp=%d&sign=%s", c.domain, path, c.appKey, timestamp, sign)
	if c.sellerID != "" {
		reqURL += "&seller_id=" + c.sellerID
	}

	// 4. 构建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("open: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	// 5. 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("open: read response: %w", err)
	}

	// 6. 解析响应
	var apiResp ApiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("open: decode response: %w (body: %s)", err, string(respBody))
	}

	return &apiResp, nil
}

// removeNilValues 递归移除 map 中值为 nil 的字段。
//
// 对应 Python 版 remove_null_values，避免发送 null 字段给服务端。
// 注意：
//   - 0、false、空字符串不会被移除（仅移除 nil）
//   - 使用反射检测 nil 指针（Go 接口持有 nil 指针时 == nil 为 false 的陷阱）
func removeNilValues(v any) any {
	// 处理 nil 接口和 nil 指针
	if v == nil || isNilPointer(v) {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, item := range val {
			cleaned := removeNilValues(item)
			if cleaned == nil {
				continue
			}
			result[k] = cleaned
		}
		return result
	case []any:
		result := make([]any, 0, len(val))
		for _, item := range val {
			cleaned := removeNilValues(item)
			if cleaned == nil {
				continue
			}
			result = append(result, cleaned)
		}
		return result
	default:
		return v
	}
}

// isNilPointer 通过反射检测 v 是否为 nil 指针。
//
// 解决 Go 接口持有 nil 指针时 `interface == nil` 为 false 的问题：
//
//	var p *SpBizType = nil
//	var i any = p
//	i == nil        // false（接口有类型信息）
//	isNilPointer(i) // true
func isNilPointer(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	}
	return false
}
