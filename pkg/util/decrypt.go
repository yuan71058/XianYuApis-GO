package util

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
)

// Decrypt 解密闲鱼加密消息。
//
// 流程:
//  1. 清理输入中的非 Base64 字符（等效于原版 JS rW 函数）
//  2. Base64 标准编码解码为原始字节
//  3. MessagePack 反序列化为 Go interface{} 树
//  4. JSON 序列化输出字符串
//
// 该函数与原版 goofish_js_version_2.js 中的 decrypt() 完全等价。
// 解密后的 JSON 可通过键路径 "1" → "10" → "reminderTitle" 等提取消息字段。
//
// 参数:
//   - data: 加密消息字符串（Base64 编码）
//
// 返回值:
//   - string: 解密后的 JSON 字符串
//   - error: 解密过程中遇到的错误
func Decrypt(data string) (string, error) {
	// Step 1: 清理非 Base64 字符（与 JS rW 函数等价）
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			return r
		}
		return -1
	}, data)

	// Step 2: Base64 解码
	raw, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("decrypt: base64 decode: %w", err)
	}

	// Step 3: MessagePack 反序列化
	// msgpack/v5 默认将 map 解码为 map[string]interface{}
	var result any
	dec := msgpack.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&result); err != nil {
		return "", fmt.Errorf("decrypt: msgpack decode: %w", err)
	}

	// Step 4: JSON 序列化（等价于 JS JSON.stringify）
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("decrypt: json marshal: %w", err)
	}

	return string(out), nil
}
