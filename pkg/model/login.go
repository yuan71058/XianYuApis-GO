package model

import "time"

// QrcodeLoginConfig 扫码登录配置。
type QrcodeLoginConfig struct {
	PollInterval time.Duration // 轮询间隔，默认 3 秒
	Timeout      time.Duration // 超时时间，默认 120 秒
	ShowQR       bool          // 是否在终端打印二维码
}

// QRCodeData 二维码数据结构。
type QRCodeData struct {
	CodeContent string // 二维码 URL
	T           string // 时间戳标识
	Ck          string // Cookie 标识
}
