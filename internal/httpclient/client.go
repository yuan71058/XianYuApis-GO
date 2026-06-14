package httpclient

import (
	"net/http"
	"net/http/cookiejar"
	"time"
)

// NewClient 创建带 CookieJar 的 HTTP 客户端。
func NewClient(timeout time.Duration) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Jar:     jar,
		Timeout: timeout,
	}, nil
}
