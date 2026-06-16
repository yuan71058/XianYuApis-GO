package util

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ParseCookies 将 Cookie 字符串解析为 map[string]string。
//
// 输入格式: "key1=value1; key2=value2; key3=value3"
// 对于值中包含 "=" 的情况，仅以第一个 "=" 作为分隔符。
func ParseCookies(cookieStr string) map[string]string {
	cookies := make(map[string]string)
	for _, item := range strings.Split(cookieStr, "; ") {
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			cookies[parts[0]] = parts[1]
		}
	}
	return cookies
}

// BuildCookieString 将 Cookie map 构建为字符串。
//
// 输出格式: "key1=value1; key2=value2"
func BuildCookieString(cookies map[string]string) string {
	var parts []string
	for k, v := range cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, "; ")
}

// GetCookieFromJar 从 CookieJar 中查找指定域名和名称的 Cookie 值。
//
// 重要: Go 的 cookiejar 遵循 RFC 6265，不接受以点开头的域名（如 .goofish.com）
// 作为 URL 主机名。必须使用合法主机名（如 www.goofish.com）查询。
// Domain 字段为 .goofish.com 的 Cookie 可以被 www.goofish.com 查询到。
func GetCookieFromJar(jar http.CookieJar, domain, name string) string {
	// 将 .goofish.com 转换为 www.goofish.com 用于查询
	host := domain
	if strings.HasPrefix(host, ".") {
		host = "www" + host
	}
	u, _ := url.Parse("https://" + host)
	for _, c := range jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// SetCookieToJar 向 CookieJar 中写入指定 Cookie。
//
// 重要: Go 的 cookiejar 遵循 RFC 6265，不接受以点开头的域名作为 URL 主机名。
// 但 Domain 字段可以设置为 .goofish.com，这样所有子域都能访问该 Cookie。
func SetCookieToJar(jar http.CookieJar, domain, name, value, path string) {
	// 将 .goofish.com 转换为 www.goofish.com 用于设置
	host := domain
	if strings.HasPrefix(host, ".") {
		host = "www" + host
	}
	u, _ := url.Parse("https://" + host)
	cookie := &http.Cookie{
		Name:   name,
		Value:  value,
		Domain: domain, // 保持原始 domain（如 .goofish.com），使所有子域可访问
		Path:   path,
	}
	jar.SetCookies(u, []*http.Cookie{cookie})
}
