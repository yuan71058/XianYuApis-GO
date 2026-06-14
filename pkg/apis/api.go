// Package apis 封装闲鱼 HTTP API，包括登录、Token 管理、商品操作、媒体上传等。
package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/util"
	"go.uber.org/zap"
)

// UA 统一的 User-Agent，用于所有 HTTP 请求。
const UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"

const (
	appKey     = "34839810"
	jsv        = "2.7.2"
	dataType   = "json"
	mtopTimout = "20000"
	sessionOpt = "AutoLoginOnly"
)

// XianyuAPI 闲鱼 HTTP API 封装。
//
// 该结构体封装了所有闲鱼 HTTP 接口，包括 sign 签名、请求构建、
// 响应解析。内部使用带 CookieJar 的 http.Client 自动管理会话。
//
// 线程安全: 同一 XianyuAPI 实例可被多个 goroutine 并发调用。
type XianyuAPI struct {
	client      *http.Client // 带 CookieJar 的 HTTP 客户端
	deviceID    string       // 设备 ID
	baseURL     string       // mtop 基础地址
	uploadURL   string       // 媒体上传地址
	passportURL string       // 通行证地址
	logger      *zap.Logger  // 日志记录器
}

// New 创建闲鱼 API 实例。
//
// 参数:
//   - cookies: 登录后的 Cookie 字典，至少包含 "unb" 字段
//   - deviceID: 可选的设备 ID。若为空则自动从 unb 生成
//
// 返回值:
//   - *XianyuAPI: 初始化的 API 实例
//   - error: 创建失败时的错误（如 unb 字段缺失）
func New(cookies map[string]string, deviceID string) (*XianyuAPI, error) {
	unb, ok := cookies["unb"]
	if !ok || unb == "" {
		return nil, fmt.Errorf("apis: cookies must contain 'unb' field")
	}

	if deviceID == "" {
		deviceID = util.GenerateDeviceID(unb)
	}

	// 创建带 CookieJar 的 HTTP 客户端
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("apis: create cookiejar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	// 将传入的 cookies 写入 CookieJar
	for name, value := range cookies {
		u, _ := url.Parse("https://.goofish.com")
		cookie := &http.Cookie{
			Name:     name,
			Value:    value,
			Domain:   ".goofish.com",
			Path:     "/",
			HttpOnly: false,
		}
		jar.SetCookies(u, []*http.Cookie{cookie})
	}

	return &XianyuAPI{
		client:      client,
		deviceID:    deviceID,
		baseURL:     "https://h5api.m.goofish.com",
		uploadURL:   "https://stream-upload.goofish.com",
		passportURL: "https://passport.goofish.com",
		logger:      zap.L(),
	}, nil
}

// DeviceID 返回当前 API 实例的设备 ID。
func (api *XianyuAPI) DeviceID() string {
	return api.deviceID
}

// Client 返回内部 HTTP 客户端（用于 WebSocket 层获取 Cookie）。
func (api *XianyuAPI) Client() *http.Client {
	return api.client
}

// tokenFromCookie 从 CookieJar 中提取 _m_h5_tk 的 token 部分（下划线前）。
func (api *XianyuAPI) tokenFromCookie() string {
	u, _ := url.Parse("https://.goofish.com")
	for _, c := range api.client.Jar.Cookies(u) {
		if c.Name == "_m_h5_tk" {
			parts := bytes.SplitN([]byte(c.Value), []byte("_"), 2)
			if len(parts) > 0 {
				return string(parts[0])
			}
		}
	}
	return ""
}

// mtopParams 构建 mtop API 的通用 URL 参数。
func (api *XianyuAPI) mtopParams(apiName, version string, extra map[string]string) url.Values {
	params := url.Values{
		"jsv":           {jsv},
		"appKey":        {appKey},
		"t":             {fmt.Sprintf("%d", time.Now().UnixMilli())},
		"v":             {version},
		"type":          {"originaljson"},
		"accountSite":   {"xianyu"},
		"dataType":      {dataType},
		"timeout":       {mtopTimout},
		"api":           {apiName},
		"sessionOption": {sessionOpt},
	}
	// 合并额外参数
	for k, v := range extra {
		params[k] = []string{v}
	}
	return params
}

// doMtopRequest 执行 mtop API 请求并返回解析后的 JSON。
//
// 内部流程:
//  1. 构建签名 (sign = MD5(token + "&" + t + "&" + appKey + "&" + data))
//  2. 设置签名到 URL 参数
//  3. 发送 POST 请求
//  4. 解析响应 JSON
//  5. 合并响应中的新 Cookie
func (api *XianyuAPI) doMtopRequest(ctx context.Context, apiName, version, data string, extraParams url.Values) (map[string]any, error) {
	params := api.mtopParams(apiName, version, nil)
	for k, v := range extraParams {
		params[k] = v
	}

	// 获取 token 并生成签名
	token := api.tokenFromCookie()
	if token == "" {
		return nil, fmt.Errorf("apis: _m_h5_tk cookie not found")
	}
	params.Set("sign", util.GenerateSign(params.Get("t"), token, data))

	// 构建请求 URL
	reqURL := fmt.Sprintf("%s/h5/%s/%s/", api.baseURL, apiName, version)

	// 构建请求体
	body := url.Values{"data": {data}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBufferString(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("apis: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.goofish.com")
	req.Header.Set("Referer", "https://www.goofish.com/")

	// 设置 URL 参数
	req.URL.RawQuery = params.Encode()

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apis: do request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("apis: decode response: %w", err)
	}

	return result, nil
}
