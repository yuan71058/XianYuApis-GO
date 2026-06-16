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
	client       *http.Client // 带 CookieJar 的 HTTP 客户端（用于 Cookie 管理）
	noJarClient  *http.Client // 无 Jar 的 HTTP 客户端（用于发送请求，Cookie 手动设置）
	deviceID     string       // 设备 ID
	baseURL      string       // mtop 基础地址
	uploadURL    string       // 媒体上传地址
	passportURL  string       // 通行证地址
	logger       *zap.Logger  // 日志记录器
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
	// 关键: 必须使用合法 URL 设置 Cookie（RFC 6265）
	// .goofish.com 不是合法 URL，必须用 https://www.goofish.com
	// Cookie 的 Domain 设为 .goofish.com，这样所有子域都能访问
	for name, value := range cookies {
		cookie := &http.Cookie{
			Name:     name,
			Value:    value,
			Domain:   ".goofish.com",
			Path:     "/",
			HttpOnly: false,
		}
		// 使用合法 URL 设置 Cookie
		u, _ := url.Parse("https://www.goofish.com")
		jar.SetCookies(u, []*http.Cookie{cookie})
	}

	return &XianyuAPI{
		client:      client,
		noJarClient: &http.Client{Timeout: 30 * time.Second}, // 无 Jar，Cookie 手动设置
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

// CookieString 返回 .goofish.com 域下所有 Cookie 的字符串表示。
// 用于 WebSocket 连接时设置 Cookie header。
//
// 与 Python 版 get_session_cookies_str(session) 对齐:
// 获取所有域下的所有 Cookie，拼接为 "key=value; key2=value2" 格式。
func (api *XianyuAPI) CookieString() string {
	var cookies []*http.Cookie
	seen := make(map[string]bool)

	// 遍历所有可能的域名，收集所有 Cookie
	for _, domain := range []string{
		"https://www.goofish.com",
		"https://goofish.com",
		"https://h5api.m.goofish.com",
		"https://passport.goofish.com",
	} {
		u, _ := url.Parse(domain)
		for _, c := range api.client.Jar.Cookies(u) {
			if !seen[c.Name] {
				seen[c.Name] = true
				cookies = append(cookies, c)
			}
		}
	}

	if len(cookies) == 0 {
		return ""
	}
	var parts []string
	for _, c := range cookies {
		parts = append(parts, c.Name+"="+c.Value)
	}
	return joinStrings(parts, "; ")
}

// joinStrings 用 sep 连接字符串切片。
func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}

// tokenFromCookie 从 CookieJar 中提取 _m_h5_tk 的 token 部分（下划线前）。
func (api *XianyuAPI) tokenFromCookie() string {
	for _, domain := range []string{
		"https://www.goofish.com",
		"https://h5api.m.goofish.com",
	} {
		u, _ := url.Parse(domain)
		for _, c := range api.client.Jar.Cookies(u) {
			if c.Name == "_m_h5_tk" {
				parts := bytes.SplitN([]byte(c.Value), []byte("_"), 2)
				if len(parts) > 0 {
					return string(parts[0])
				}
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
// 与 Python 版 _fetch_im_token_from_api 对齐:
// 手动设置 Cookie header（而非依赖 CookieJar），确保发送完整 Cookie。
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
	// 手动设置 Cookie header（与 Python 版一致，不依赖 CookieJar）
	// CookieJar 可能不发送 HttpOnly 的 cookie2/sgcookie
	req.Header.Set("Cookie", api.CookieString())

	// 设置 URL 参数
	req.URL.RawQuery = params.Encode()

	// 使用无 Jar 的 client 发送请求，避免 CookieJar 覆盖手动设置的 Cookie header
	resp, err := api.noJarClient.Do(req)
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

// RefreshMtopToken 刷新 _m_h5_tk Cookie。
//
// 通过调用一个简单的 mtop 接口触发服务端重新签发 _m_h5_tk。
// 当验证码完成后，原有的 _m_h5_tk 可能已失效，需要刷新。
func (api *XianyuAPI) RefreshMtopToken(ctx context.Context) {
	// 使用空签名调用 mtop 接口，服务端会返回新的 _m_h5_tk
	params := url.Values{
		"jsv":           {jsv},
		"appKey":        {appKey},
		"t":             {fmt.Sprintf("%d", time.Now().UnixMilli())},
		"sign":          {""},
		"v":             {"1.0"},
		"type":          {"originaljson"},
		"dataType":      {dataType},
		"timeout":       {mtopTimout},
		"api":           {"mtop.taobao.idlehome.home.webpc.feed"},
		"sessionOption": {sessionOpt},
	}

	reqURL := fmt.Sprintf("%s/h5/mtop.taobao.idlehome.home.webpc.feed/1.0/?%s", api.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader([]byte("data=%7B%7D")))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Origin", "https://www.goofish.com")
	req.Header.Set("Referer", "https://www.goofish.com/")
	req.Header.Set("Cookie", api.CookieString())

	resp, err := api.noJarClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()

	// 检查 _m_h5_tk 是否已更新
	if newToken := api.tokenFromCookie(); newToken != "" {
		api.logger.Info("mtop token refreshed", zap.String("token", newToken[:10]+"..."))
	}
}
