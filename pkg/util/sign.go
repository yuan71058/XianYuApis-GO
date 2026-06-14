// Package util 提供闲鱼 API 所需的工具函数，包括签名、解密、ID 生成等。
//
// 所有函数均为纯函数或无副作用的生成函数，可被外部项目直接 import 复用。
package util

import (
	"crypto/md5"
	"fmt"
)

// GenerateSign 生成闲鱼 API 签名。
//
// 签名公式: MD5(token + "&" + timestamp + "&" + appKey + "&" + data)
// 其中 appKey = "34839810" 为闲鱼平台固定标识。
//
// 参数:
//   - timestamp: 毫秒级时间戳字符串
//   - token:     从 Cookie _m_h5_tk 中提取的 token 部分（下划线前）
//   - data:      请求体 JSON 字符串
//
// 返回值: 32 位小写十六进制签名串
func GenerateSign(timestamp, token, data string) string {
	msg := fmt.Sprintf("%s&%s&%s&%s", token, timestamp, "34839810", data)
	sum := md5.Sum([]byte(msg))
	return fmt.Sprintf("%x", sum)
}
