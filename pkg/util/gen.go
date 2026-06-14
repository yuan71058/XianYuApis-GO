package util

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"time"
)

const deviceCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenerateMid 生成消息 ID。
//
// 格式: "{0-999 随机数}{毫秒时间戳} 0"
// 示例: "7381748291023 0"
func GenerateMid() string {
	randNum := mathrand.Intn(1000)
	return fmt.Sprintf("%d%d 0", randNum, time.Now().UnixMilli())
}

// GenerateUUID 生成唯一请求标识。
//
// 格式: "-{毫秒时间戳}1"
// 注意: 这不是 RFC 4122 UUID，而是闲鱼协议约定的时间戳格式。
func GenerateUUID() string {
	return fmt.Sprintf("-%d1", time.Now().UnixMilli())
}

// GenerateDeviceID 基于用户 unb 生成设备 ID。
//
// 格式: "RFC4122-v4-UUID-用户unb"
// 使用 crypto/rand 确保随机性（原版 JS 使用 Math.random 可预测）。
func GenerateDeviceID(userID string) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// 回退到 time-based: 理论上不会发生
		return fmt.Sprintf("fallback-%s", userID)
	}

	// 构造 RFC 4122 v4 UUID 格式
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10xx

	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x-%s",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7],
		b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15],
		userID)
}
