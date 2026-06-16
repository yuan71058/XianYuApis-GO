package util

import (
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
// 格式: "UUID-v4-格式-用户unb"
// 与 Python 版 generate_device_id 对齐，使用与 Python 版相同的字符集。
func GenerateDeviceID(userID string) string {
	chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	result := make([]byte, 36+1+len(userID))

	for i := 0; i < 36; i++ {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			result[i] = '-'
		case i == 14:
			result[i] = '4' // UUID v4
		case i == 19:
			randVal := mathrand.Intn(16)
			result[i] = chars[(randVal&0x3)|0x8]
		default:
			randVal := mathrand.Intn(len(chars))
			result[i] = chars[randVal]
		}
	}

	result[36] = '-'
	copy(result[37:], userID)
	return string(result)
}
