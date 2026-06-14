package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
				T           string `json:"t"`
				Ck          string `json:"ck"`
			} `json:"data"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &model.QRCodeData{
		CodeContent: result.Content.Data.CodeContent,
		T:           result.Content.Data.T,
		Ck:          result.Content.Data.Ck,
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
func extractGoofishCookies(client *http.Client) map[string]string {
	cookies := make(map[string]string)
	if jar, ok := client.Jar.(*cookiejar.Jar); ok {
		for _, domain := range []string{".goofish.com", ".mmstat.com"} {
			u, _ := url.Parse("https://" + domain)
			for _, c := range jar.Cookies(u) {
				cookies[c.Name] = c.Value
			}
		}
	}
	return cookies
}

// printQR 在终端使用 Unicode 块字符打印二维码。
func printQR(data string) {
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		fmt.Printf("[qrcode] generate failed: %v\n", err)
		return
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
