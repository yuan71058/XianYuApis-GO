// Package openapi 封装闲鱼开放平台 API（https://open.goofish.pro）。
//
// 与 pkg/apis（Cookie 模拟登录）不同，本包使用官方开放平台 app_key/app_secret 签名认证，
// 适用于已申请闲鱼开放平台资质的商家。
//
// 翻译自 Python SDK: https://github.com/XIE7654/goofish_api
//
// 快速开始:
//
//	client := openapi.New("your_app_key", "your_app_secret")
//	// 查询授权店铺
//	resp, _ := client.User.GetAuthorizeList(ctx)
//	// 查询商品列表
//	resp, _ := client.Good.GetProductList(ctx, &openapi.GetProductListRequest{PageNo: 1, PageSize: 50})
package openapi

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// openapiDomain 闲鱼开放平台域名
const openapiDomain = "https://open.goofish.pro"

// GoofishClient 闲鱼开放平台客户端。
//
// 内部管理 app_key/app_secret 签名认证，包含 User/Good/Order/Other 四个 API 子模块。
// 线程安全：同一实例可被多个 goroutine 并发使用。
type GoofishClient struct {
	appKey    string // 应用 Key（开放平台申请）
	appSecret string // 应用 Secret（开放平台申请）
	sellerID  string // 卖家 ID（授权后获得，可选）
	debug     bool   // 调试模式
	http      *http.Client

	User  *UserService  // 用户模块
	Good  *GoodService  // 商品模块
	Order *OrderService // 订单模块
	Other *OtherService // 其他模块
}

// New 创建闲鱼开放平台客户端。
//
// 参数:
//   - appKey: 应用 Key
//   - appSecret: 应用 Secret
//
// 示例:
//
//	client := openapi.New("your_app_key", "your_app_secret")
func New(appKey, appSecret string) *GoofishClient {
	c := &GoofishClient{
		appKey:    appKey,
		appSecret: appSecret,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
	c.User = &UserService{client: c}
	c.Good = &GoodService{client: c}
	c.Order = &OrderService{client: c}
	c.Other = &OtherService{client: c}
	return c
}

// WithSellerID 设置卖家 ID（授权后获得）。
// 已授权卖家调用 API 时签名需包含 seller_id。
func (c *GoofishClient) WithSellerID(sellerID string) *GoofishClient {
	c.sellerID = sellerID
	return c
}

// WithDebug 启用/禁用调试模式。
func (c *GoofishClient) WithDebug(debug bool) *GoofishClient {
	c.debug = debug
	return c
}

// getSign 生成请求签名。
//
// 签名算法（与 Python 版一致）:
//  1. body_md5 = MD5(body_json)
//  2. 若有 seller_id: sign_str = app_key + "," + body_md5 + "," + timestamp + "," + seller_id + "," + app_secret
//     若无 seller_id: sign_str = app_key + "," + body_md5 + "," + timestamp + "," + app_secret
//  3. sign = MD5(sign_str)
func (c *GoofishClient) getSign(bodyJSON string, timestamp int64) string {
	// 1. 计算 body 的 MD5
	bodyMD5 := md5Hex(bodyJSON)

	// 2. 拼接签名字符串
	var signStr string
	if c.sellerID != "" {
		signStr = fmt.Sprintf("%s,%s,%d,%s,%s", c.appKey, bodyMD5, timestamp, c.sellerID, c.appSecret)
	} else {
		signStr = fmt.Sprintf("%s,%s,%d,%s", c.appKey, bodyMD5, timestamp, c.appSecret)
	}

	// 3. 计算 MD5 签名
	return md5Hex(signStr)
}

// md5Hex 计算字符串的 MD5 哈希，返回 32 位小写十六进制。
func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// request 发送 API 请求。
//
// 内部流程:
//  1. 将 data 序列化为紧凑 JSON（无空格，与 Python json.dumps(separators=(",", ":")) 一致）
//  2. 移除值为 nil 的字段
//  3. 生成签名
//  4. 构建 URL: {domain}{path}?appid={app_key}&timestamp={timestamp}&sign={sign}[&seller_id={seller_id}]
//  5. POST 请求并返回响应
//
// 参数:
//   - ctx: 请求上下文
//   - path: API 路径（如 "/api/open/user/authorize/list"）
//   - data: 请求体数据（会被序列化为 JSON）
func (c *GoofishClient) request(ctx context.Context, path string, data any) (map[string]any, error) {
	// 序列化为紧凑 JSON（无多余空格，签名关键）
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("openapi: marshal request data: %w", err)
	}

	// 生成签名
	timestamp := time.Now().Unix()
	sign := c.getSign(string(body), timestamp)

	// 构建 URL
	url := fmt.Sprintf("%s%s?appid=%s&timestamp=%d&sign=%s", openapiDomain, path, c.appKey, timestamp, sign)
	if c.sellerID != "" {
		url += "&seller_id=" + c.sellerID
	}

	// 构建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openapi: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	// 发送请求
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openapi: do request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openapi: read response: %w", err)
	}

	if c.debug {
		fmt.Printf("[openapi debug] %s %s\n  request: %s\n  response: %s\n",
			http.MethodPost, url, string(body), string(respBody))
	}

	// 解析响应 JSON
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("openapi: unmarshal response: %w", err)
	}

	return result, nil
}
