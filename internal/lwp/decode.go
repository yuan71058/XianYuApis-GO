package lwp

import "encoding/json"

// DecodeMessage 将 JSON bytes 解码为消息结构体。
func DecodeMessage(data []byte) (map[string]any, error) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// EncodeMessage 将消息结构体编码为 JSON bytes。
func EncodeMessage(msg map[string]any) ([]byte, error) {
	return json.Marshal(msg)
}
