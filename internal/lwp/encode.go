package lwp

import "encoding/json"

// Encode 将 LWP 消息编码为 JSON bytes。
func Encode(msg map[string]any) ([]byte, error) {
	return json.Marshal(msg)
}

// Decode 将 JSON bytes 解码为 LWP 消息。
func Decode(data []byte) (map[string]any, error) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}
