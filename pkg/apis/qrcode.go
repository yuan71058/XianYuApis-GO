package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/model"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"github.com/skip2/go-qrcode"
)

// QrcodeLoginConfig 扫码登录配置。
type QrcodeLoginConfig = model.QrcodeLoginConfig

// QrcodeLogin 完整的扫码登录流程。
//
// 登录步骤:
//  1. BuildInitialCookies()          → 获取基础 Cookie
//  2. 加载 passport.goofish.com/mini_login.htm → 获取 XSRF-TOKEN
//  3. POST /qrcode/generate.do       → 生成二维码 URL
//  4. 终端打印二维码 (可选)
//  5. 轮询 /qrcode/query.do          → 等待用户扫码
//  6. POST /login_token/login.do     → 完成登录
//  7. 刷新 mtop Cookie               → 更新 _m_h5_tk
//  8. 返回已登录的 XianyuAPI 实例
func QrcodeLogin(cfg QrcodeLoginConfig) (*XianyuAPI, error) {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 3 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}

	// Step 1: 构建初始 Cookie
	client, initialCookies, err := BuildInitialCookies()
	if err != nil {
		return nil, fmt.Errorf("qrcode: build initial cookies: %w", err)
	}

	// Step 2: 加载 passport 登录页面，获取 XSRF-TOKEN
	csrfToken, err := loadPassportPage(client)
	if err != nil {
		return nil, fmt.Errorf("qrcode: load passport: %w", err)
	}

	// Step 3: 生成二维码
	qrData, err := generateQRCode(client, csrfToken, initialCookies["cookie2"])
	if err != nil {
		return nil, fmt.Errorf("qrcode: generate QR: %w", err)
	}

	if cfg.ShowQR {
		printQR(qrData.CodeContent)
	}
	fmt.Printf("[qrcode_login] QR URL: %s\n", qrData.CodeContent)
	fmt.Printf("[qrcode_login] 请使用闲鱼 APP 扫码登录\n")

	// Step 4: 轮询扫码状态
	loginToken, err := pollQRStatus(client, qrData, csrfToken, initialCookies["cookie2"], cfg.PollInterval, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("qrcode: poll status: %w", err)
	}

	// Step 5: 完成登录
	if loginToken != "" {
		cna := initialCookies["cna"]
		if err := completeLogin(client, loginToken, cna); err != nil {
			return nil, fmt.Errorf("qrcode: complete login: %w", err)
		}
	}

	// Step 6: 刷新 mtop Cookie
	refreshMtopCookies(client)

	// Step 7: 提取 .goofish.com 域下的 Cookie 并组装 XianyuAPI
	finalCookies := extractGoofishCookies(client)
	if unb, ok := finalCookies["unb"]; !ok || unb == "" {
		return nil, fmt.Errorf("qrcode: unb cookie not found after login")
	}
	deviceID := util.GenerateDeviceID(finalCookies["unb"])

	return New(finalCookies, deviceID)
}

// loadPassportPage 访问 passport 登录页面，获取 XSRF-TOKEN 等 Cookie。
func loadPassportPage(client *http.Client) (string, error) {
	params := url.Values{
		"lang":            {"zh_cn"},
		"appName":         {"xianyu"},
		"appEntrance":     {"web"},
		"styleType":       {"vertical"},
		"bizParams":       {""},
		"notLoadSsoView":  {"false"},
		"notKeepLogin":    {"false"},
		"isMobile":        {"false"},
		"qrCodeFirst":     {"false"},
		"stie":            {"77"},
		"rnd":             {"0.6842814084442211"},
	}

	resp, err := client.Get("https://passport.goofish.com/mini_login.htm?" + params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 从响应 Cookie 中获取 XSRF-TOKEN
	if jar, ok := client.Jar.(*cookiejar.Jar); ok {
		u, _ := url.Parse("https://passport.goofish.com")
		for _, c := range jar.Cookies(u) {
			if c.Name == "XSRF-TOKEN" {
				return c.Value, nil
			}
		}
	}
	return "", fmt.Errorf("qrcode: XSRF-TOKEN not found")
}

// generateQRCode 请求生成二维码，返回二维码数据。
func generateQRCode(client *http.Client, csrfToken, cookie2 string) (*model.QRCodeData, error) {
	params := url.Values{
		"appName":     {"xianyu"},
		"fromSite":    {"77"},
		"appEntrance": {"web"},
		"_csrf_token": {csrfToken},
		"umidToken":   {""},
		"hsiz":        {cookie2},
		"bizParams":   {"taobaoBizLoginFrom=web&renderRefer=" + url.QueryEscape("https://www.goofish.com/")},
		"mainPage":    {"false"},
		"isMobile":    {"false"},
		"lang":        {"zh_CN"},
		"returnUrl":   {""},
		"umidTag":     {"SERVER"},
	}

	req, err := http.NewRequest(http.MethodGet,
		"https://passport.goofish.com/newlogin/qrcode/generate.do?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Referer", "https://passport.goofish.com/mini_login.htm")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Content struct {
			Data struct {
				CodeContent string `json:"codeContent"`
				T           any    `json:"t"`
				Ck          any    `json:"ck"`
			} `json:"data"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &model.QRCodeData{
		CodeContent: result.Content.Data.CodeContent,
		T:           anyToString(result.Content.Data.T),
		Ck:          anyToString(result.Content.Data.Ck),
	}, nil
}

// pollQRStatus 轮询二维码扫码状态。
func pollQRStatus(client *http.Client, qrData *model.QRCodeData,
	csrfToken, cookie2 string, interval, timeout time.Duration,
) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		body := url.Values{
			"appName":     {"xianyu"},
			"fromSite":    {"77"},
			"appEntrance": {"web"},
			"_csrf_token": {csrfToken},
			"umidToken":   {""},
			"hsiz":        {cookie2},
			"bizParams":   {"taobaoBizLoginFrom=web&renderRefer=" + url.QueryEscape("https://www.goofish.com/")},
			"mainPage":    {"false"},
			"isMobile":    {"false"},
			"lang":        {"zh_CN"},
			"returnUrl":   {""},
			"umidTag":     {"SERVER"},
			"navlanguage": {"en"},
			"navUserAgent": {UA},
			"navPlatform": {"Win32"},
			"isIframe":    {"true"},
			"documentReferer": {"https://www.goofish.com/"},
			"defaultView": {"sms"},
			"t":           {qrData.T},
			"ck":          {qrData.Ck},
		}

		req, err := http.NewRequest(http.MethodPost,
			"https://passport.goofish.com/newlogin/qrcode/query.do?appName=xianyu&fromSite=77",
			bytes.NewReader([]byte(body.Encode())))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", UA)
		req.Header.Set("Origin", "https://passport.goofish.com")
		req.Header.Set("Referer", "https://passport.goofish.com/mini_login.htm")

		resp, err := client.Do(req)
		if err == nil {
			var result struct {
				Content struct {
					Data struct {
						QRCodeStatus string `json:"qrCodeStatus"`
						Token        string `json:"token"`
						LgToken      string `json:"lgToken"`
					} `json:"data"`
				} `json:"content"`
			}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			status := result.Content.Data.QRCodeStatus
			remaining := int(time.Until(deadline).Seconds())
			switch status {
			case "NEW":
				fmt.Printf("[qrcode] [NEW] Waiting for scan (%ds left)\n", remaining)
			case "SCANNED":
				fmt.Printf("[qrcode] [SCANNED] Scanned, confirm on phone (%ds left)\n", remaining)
			case "CONFIRMED":
				token := result.Content.Data.Token
				if token == "" {
					token = result.Content.Data.LgToken
				}
				fmt.Println("[qrcode] [CONFIRMED] Login confirmed")
				return token, nil
			case "EXPIRED":
				return "", fmt.Errorf("qrcode: QR expired")
			}
		}

		time.Sleep(interval)
	}
	return "", fmt.Errorf("qrcode: scan timeout")
}

// completeLogin 使用 loginToken 完成登录流程。
func completeLogin(client *http.Client, token, cna string) error {
	params := url.Values{
		"token":   {token},
		"subFlow": {"DIALOG_CHECK_LOGIN_RPC"},
		"nextCode": {"0018"},
		"bizScene": {"qrcode"},
		"confirm": {"true"},
	}

	body := url.Values{"deviceId": {cna}}
	req, err := http.NewRequest(http.MethodPost,
		"https://passport.goofish.com/login_token/login.do?"+params.Encode(),
		bytes.NewReader([]byte(body.Encode())))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Origin", "https://passport.goofish.com")
	req.Header.Set("Referer", "https://passport.goofish.com/mini_login.htm")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("[qrcode_login] login_token response status: %d\n", resp.StatusCode)
	return nil
}

// refreshMtopCookies 登录后刷新 mtop 域下的 Cookie（_m_h5_tk 会变）。
func refreshMtopCookies(client *http.Client) {
	params := url.Values{
		"jsv":           {"2.7.2"},
		"appKey":        {"34839810"},
		"t":             {fmt.Sprintf("%d", time.Now().UnixMilli())},
		"sign":          {""},
		"v":             {"1.0"},
		"type":          {"originaljson"},
		"dataType":      {"json"},
		"timeout":       {"20000"},
		"api":           {"mtop.idle.web.user.page.nav"},
		"sessionOption": {"AutoLoginOnly"},
		"spm_cnt":       {"a21ybx.home.0.0"},
	}

	req, _ := http.NewRequest(http.MethodPost,
		"https://h5api.m.goofish.com/h5/mtop.idle.web.user.page.nav/1.0/?"+params.Encode(),
		bytes.NewReader([]byte("data=%7B%7D")))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Origin", "https://www.goofish.com")
	req.Header.Set("Referer", "https://www.goofish.com/")

	client.Do(req)
}

// extractGoofishCookies 从 CookieJar 中提取 .goofish.com 域下的所有 Cookie。
//
// 重要: Go 的 cookiejar 遵循 RFC 6265，不接受以点开头的域名（如 .goofish.com）
// 作为 URL 主机名。必须使用合法主机名（如 www.goofish.com）查询。
// Domain 字段为 .goofish.com 的 Cookie 可以被 www.goofish.com 查询到。
func extractGoofishCookies(client *http.Client) map[string]string {
	cookies := make(map[string]string)
	if jar, ok := client.Jar.(*cookiejar.Jar); ok {
		for _, domain := range []string{".goofish.com", ".mmstat.com"} {
			// 将 .goofish.com 转换为 www.goofish.com 用于查询
			host := domain
			if strings.HasPrefix(host, ".") {
				host = "www" + host
			}
			u, _ := url.Parse("https://" + host)
			for _, c := range jar.Cookies(u) {
				cookies[c.Name] = c.Value
			}
		}
	}
	return cookies
}

// printQR 在终端打印二维码，并保存为 PNG 图片文件。
func printQR(data string) {
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		fmt.Printf("[qrcode] generate failed: %v\n", err)
		return
	}

	// 保存为 PNG 图片文件，方便无法看清终端二维码的用户
	pngPath := "qrcode.png"
	if err := qr.WriteFile(256, pngPath); err != nil {
		fmt.Printf("[qrcode] save png failed: %v\n", err)
	} else {
		absPath, _ := absPath(pngPath)
		fmt.Printf("[qrcode] 二维码已保存到: %s\n", absPath)
		fmt.Printf("[qrcode] 请用闲鱼 APP 扫描该图片，或直接打开上面的 URL\n")
	}

	// 使用半块字符 (▀▄█) 绘制接近正方形的二维码
	matrix := qr.Bitmap()
	rows := len(matrix)
	if rows == 0 {
		return
	}
	cols := len(matrix[0])
	for r := 0; r < rows; r += 2 {
		line := ""
		for c := 0; c < cols; c++ {
			top := matrix[r][c]
			bot := false
			if r+1 < rows {
				bot = matrix[r+1][c]
			}
			if top && bot {
				line += "█"
			} else if top && !bot {
				line += "▀"
			} else if !top && bot {
				line += "▄"
			} else {
				line += " "
			}
		}
		fmt.Println(line)
	}
}

// anyToString 将 any 类型安全转为 string（兼容 string 和 float64 两种 JSON 类型）。
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%.0f", s)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// absPath 返回文件的绝对路径。
func absPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p, err
	}
	return abs, nil
}

// ==================== 异步扫码登录（供 Wails 等桌面应用集成） ====================
// 按 DESIGN.md 4.1 扫码登录设计：拆分为 生成二维码 → 轮询状态 → 完成登录 三步
// 需求理解：1. 阻塞流程拆分为异步步骤 2. 保存中间 HTTP 会话 3. 前端可展示二维码并轮询

// QrcodeSession 扫码登录会话，保存中间状态以便分步骤调用。
// 生命周期：Generate → PollOnce(循环) → Complete
type QrcodeSession struct {
	Client     *http.Client       // 带 CookieJar 的 HTTP 客户端
	CSRFToken  string             // XSRF-TOKEN
	Cookie2    string             // cookie2 值
	Cna        string             // cna 值
	QRData     *model.QRCodeData  // 二维码数据
	LoginToken string             // 登录令牌（CONFIRMED 状态时填充）
}

// QrcodeGenerateAsync 异步扫码登录第一步：生成二维码。
// 调用 BuildInitialCookies + loadPassportPage + generateQRCode，
// 返回包含二维码 URL 和中间 HTTP 会话的 QrcodeSession。
func QrcodeGenerateAsync() (*QrcodeSession, error) {
	// Step 1: 构建初始 Cookie
	client, initialCookies, err := BuildInitialCookies()
	if err != nil {
		return nil, fmt.Errorf("qrcode: build initial cookies: %w", err)
	}

	// Step 2: 加载 passport 登录页面，获取 XSRF-TOKEN
	csrfToken, err := loadPassportPage(client)
	if err != nil {
		return nil, fmt.Errorf("qrcode: load passport: %w", err)
	}

	// Step 3: 生成二维码
	qrData, err := generateQRCode(client, csrfToken, initialCookies["cookie2"])
	if err != nil {
		return nil, fmt.Errorf("qrcode: generate QR: %w", err)
	}

	return &QrcodeSession{
		Client:    client,
		CSRFToken: csrfToken,
		Cookie2:   initialCookies["cookie2"],
		Cna:       initialCookies["cna"],
		QRData:    qrData,
	}, nil
}

// GetQRCodeURL 返回二维码 URL（前端用此 URL 生成二维码图片）。
func (s *QrcodeSession) GetQRCodeURL() string {
	if s.QRData == nil {
		return ""
	}
	return s.QRData.CodeContent
}

// PollOnce 单次轮询扫码状态（不阻塞）。
// 返回值:
//   - status: "NEW" / "SCANNED" / "CONFIRMED" / "EXPIRED" / "ERROR"
//   - err: 网络错误或二维码过期
//
// 当 status == "CONFIRMED" 时，LoginToken 已保存到 session 中，可调用 Complete()。
func (s *QrcodeSession) PollOnce() (string, error) {
	if s.QRData == nil {
		return "ERROR", fmt.Errorf("qrcode: session has no QR data")
	}

	body := url.Values{
		"appName":     {"xianyu"},
		"fromSite":    {"77"},
		"appEntrance": {"web"},
		"_csrf_token": {s.CSRFToken},
		"umidToken":   {""},
		"hsiz":        {s.Cookie2},
		"bizParams":   {"taobaoBizLoginFrom=web&renderRefer=" + url.QueryEscape("https://www.goofish.com/")},
		"mainPage":    {"false"},
		"isMobile":    {"false"},
		"lang":        {"zh_CN"},
		"returnUrl":   {""},
		"umidTag":     {"SERVER"},
		"navlanguage": {"en"},
		"navUserAgent": {UA},
		"navPlatform":  {"Win32"},
		"isIframe":     {"true"},
		"documentReferer": {"https://www.goofish.com/"},
		"defaultView": {"sms"},
		"t":           {s.QRData.T},
		"ck":          {s.QRData.Ck},
	}

	req, err := http.NewRequest(http.MethodPost,
		"https://passport.goofish.com/newlogin/qrcode/query.do?appName=xianyu&fromSite=77",
		bytes.NewReader([]byte(body.Encode())))
	if err != nil {
		return "ERROR", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Origin", "https://passport.goofish.com")
	req.Header.Set("Referer", "https://passport.goofish.com/mini_login.htm")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "ERROR", err
	}
	defer resp.Body.Close()

	// 读取完整响应体用于调试
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "ERROR", err
	}
	fmt.Printf("[qrcode_poll] response body: %s\n", string(respBody))

	// 解析完整响应结构（兼容多种字段名）
	var rawResult struct {
		Content struct {
			Data struct {
				QRCodeStatus string `json:"qrCodeStatus"`
				Token        string `json:"token"`
				LgToken      string `json:"lgToken"`
				St           string `json:"st"`
				StEx         string `json:"stEx"`
			} `json:"data"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &rawResult); err != nil {
		return "ERROR", err
	}

	status := rawResult.Content.Data.QRCodeStatus
	fmt.Printf("[qrcode_poll] status=%s, token=%q, lgToken=%q, st=%q\n",
		status, rawResult.Content.Data.Token, rawResult.Content.Data.LgToken, rawResult.Content.Data.St)

	switch status {
	case "NEW":
		return "NEW", nil
	case "SCANNED":
		return "SCANNED", nil
	case "CONFIRMED":
		// 尝试多个字段获取 token：token > lgToken > st > stEx
		token := rawResult.Content.Data.Token
		if token == "" {
			token = rawResult.Content.Data.LgToken
		}
		if token == "" {
			token = rawResult.Content.Data.St
		}
		if token == "" {
			token = rawResult.Content.Data.StEx
		}
		if token == "" {
			// token 为空，返回错误而不是静默继续
			return "ERROR", fmt.Errorf("qrcode: CONFIRMED but no token found in response (token/lgToken/st/stEx all empty)")
		}
		s.LoginToken = token
		return "CONFIRMED", nil
	case "EXPIRED":
		return "EXPIRED", fmt.Errorf("qrcode: QR expired")
	default:
		return "NEW", nil // 未知状态当作 NEW 处理
	}
}

// Complete 完成登录流程，返回 .goofish.com 域下的 Cookie 字典。
// 必须在 PollOnce 返回 "CONFIRMED" 后调用。
// 返回的 Cookie 字典可直接传给 New() 创建 XianyuAPI 实例。
func (s *QrcodeSession) Complete() (map[string]string, error) {
	if s.LoginToken == "" {
		return nil, fmt.Errorf("qrcode: no login token, please poll until CONFIRMED")
	}

	// Step 5: 完成登录
	if err := completeLogin(s.Client, s.LoginToken, s.Cna); err != nil {
		return nil, fmt.Errorf("qrcode: complete login: %w", err)
	}

	// Step 6: 刷新 mtop Cookie
	refreshMtopCookies(s.Client)

	// Step 7: 提取 .goofish.com 域下的 Cookie
	finalCookies := extractGoofishCookies(s.Client)
	if unb, ok := finalCookies["unb"]; !ok || unb == "" {
		return nil, fmt.Errorf("qrcode: unb cookie not found after login")
	}

	return finalCookies, nil
}
