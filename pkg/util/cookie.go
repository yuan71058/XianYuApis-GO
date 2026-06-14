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

// GetCookieFromJar 从 *cookiejar.Jar 中查找指定域名和名称的 Cookie 值。
//
// 注意: 需要传入具体的 *cookiejar.Jar（而非 http.CookieJar 接口），
// 因为标准库的 CookieJar 接口不暴露遍历方法。
func GetCookieFromJar(jar http.CookieJar, domain, name string) string {
	u, _ := url.Parse("https://" + domain)
	for _, c := range jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// SetCookieToJar 向 CookieJar 中写入指定 Cookie。
func SetCookieToJar(jar http.CookieJar, domain, name, value, path string) {
	u, _ := url.Parse("https://" + domain)
	cookie := &http.Cookie{
		Name:   name,
		Value:  value,
		Domain: domain,
		Path:   path,
	}
	jar.SetCookies(u, []*http.Cookie{cookie})
}
