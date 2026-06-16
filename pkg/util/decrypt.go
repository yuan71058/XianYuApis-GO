package util

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Base64DecodeUTF8 将 Base64 字符串解码为 UTF-8 字符串。
func Base64DecodeUTF8(data string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			return r
		}
		return -1
	}, data)

	raw, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	return string(raw), nil
}

// Decrypt 解密闲鱼加密消息。
//
// 流程:
//  1. 清理输入中的非 Base64 字符
//  2. Base64 标准编码解码为原始字节
//  3. 自定义 MessagePack 反序列化（支持整数键，自动转为字符串键）
//  4. JSON 序列化输出字符串
func Decrypt(data string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			return r
		}
		return -1
	}, data)

	raw, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("decrypt: base64 decode: %w", err)
	}

	// 使用自定义 msgpack 解码器，支持整数键
	dec := newMsgpackDecoder(raw)
	result, err := dec.decode()
	if err != nil {
		return "", fmt.Errorf("decrypt: msgpack decode: %w", err)
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("decrypt: json marshal: %w", err)
	}

	return string(out), nil
}

// ==================== 自定义 msgpack 解码器 ====================
// 闲鱼的 msgpack 数据使用整数键（1, 2, 3...），
// 标准库 msgpack/v5 解码到 any 时默认用 map[string]any，遇到整数键会报错。
// 因此自行实现解码器，将整数键自动转为字符串键。

type msgpackDecoder struct {
	data []byte
	pos  int
}

func newMsgpackDecoder(data []byte) *msgpackDecoder {
	return &msgpackDecoder{data: data}
}

func (d *msgpackDecoder) decode() (any, error) {
	if d.pos >= len(d.data) {
		return nil, errors.New("unexpected end of data")
	}

	code := d.data[d.pos]
	d.pos++

	switch {
	case code <= 0x7f: // positive fixint
		return int64(code), nil
	case code >= 0x80 && code <= 0x8f: // fixmap
		return d.decodeMap(int(code & 0x0f))
	case code >= 0x90 && code <= 0x9f: // fixarray
		return d.decodeArray(int(code & 0x0f))
	case code >= 0xa0 && code <= 0xbf: // fixstr
		return d.decodeString(int(code & 0x1f))
	case code == 0xc0: // nil
		return nil, nil
	case code == 0xc2: // false
		return false, nil
	case code == 0xc3: // true
		return true, nil
	case code == 0xc4: // bin8
		n := int(d.data[d.pos])
		d.pos++
		return d.decodeBytes(n)
	case code == 0xc5: // bin16
		n := int(binary.BigEndian.Uint16(d.data[d.pos:]))
		d.pos += 2
		return d.decodeBytes(n)
	case code == 0xc6: // bin32
		n := int(binary.BigEndian.Uint32(d.data[d.pos:]))
		d.pos += 4
		return d.decodeBytes(n)
	case code == 0xca: // float32
		bits := binary.BigEndian.Uint32(d.data[d.pos:])
		d.pos += 4
		return float64(math.Float32frombits(bits)), nil
	case code == 0xcb: // float64
		bits := binary.BigEndian.Uint64(d.data[d.pos:])
		d.pos += 8
		return math.Float64frombits(bits), nil
	case code == 0xcc: // uint8
		v := int64(d.data[d.pos])
		d.pos++
		return v, nil
	case code == 0xcd: // uint16
		v := int64(binary.BigEndian.Uint16(d.data[d.pos:]))
		d.pos += 2
		return v, nil
	case code == 0xce: // uint32
		v := int64(binary.BigEndian.Uint32(d.data[d.pos:]))
		d.pos += 4
		return v, nil
	case code == 0xcf: // uint64
		v := int64(binary.BigEndian.Uint64(d.data[d.pos:]))
		d.pos += 8
		return v, nil
	case code == 0xd0: // int8
		v := int64(int8(d.data[d.pos]))
		d.pos++
		return v, nil
	case code == 0xd1: // int16
		v := int64(int16(binary.BigEndian.Uint16(d.data[d.pos:])))
		d.pos += 2
		return v, nil
	case code == 0xd2: // int32
		v := int64(int32(binary.BigEndian.Uint32(d.data[d.pos:])))
		d.pos += 4
		return v, nil
	case code == 0xd3: // int64
		v := int64(binary.BigEndian.Uint64(d.data[d.pos:]))
		d.pos += 8
		return v, nil
	case code == 0xd9: // str8
		n := int(d.data[d.pos])
		d.pos++
		return d.decodeString(n)
	case code == 0xda: // str16
		n := int(binary.BigEndian.Uint16(d.data[d.pos:]))
		d.pos += 2
		return d.decodeString(n)
	case code == 0xdb: // str32
		n := int(binary.BigEndian.Uint32(d.data[d.pos:]))
		d.pos += 4
		return d.decodeString(n)
	case code == 0xdc: // array16
		n := int(binary.BigEndian.Uint16(d.data[d.pos:]))
		d.pos += 2
		return d.decodeArray(n)
	case code == 0xdd: // array32
		n := int(binary.BigEndian.Uint32(d.data[d.pos:]))
		d.pos += 4
		return d.decodeArray(n)
	case code == 0xde: // map16
		n := int(binary.BigEndian.Uint16(d.data[d.pos:]))
		d.pos += 2
		return d.decodeMap(n)
	case code == 0xdf: // map32
		n := int(binary.BigEndian.Uint32(d.data[d.pos:]))
		d.pos += 4
		return d.decodeMap(n)
	case code >= 0xe0: // negative fixint
		return int64(int8(code)), nil
	default:
		return nil, fmt.Errorf("unsupported msgpack code: 0x%02x", code)
	}
}

func (d *msgpackDecoder) decodeMap(n int) (map[string]any, error) {
	m := make(map[string]any, n)
	for i := 0; i < n; i++ {
		key, err := d.decodeMapKey()
		if err != nil {
			return nil, err
		}
		value, err := d.decode()
		if err != nil {
			return nil, err
		}
		m[key] = value
	}
	return m, nil
}

func (d *msgpackDecoder) decodeMapKey() (string, error) {
	v, err := d.decode()
	if err != nil {
		return "", err
	}
	switch key := v.(type) {
	case string:
		return key, nil
	case int64:
		return fmt.Sprintf("%d", key), nil
	case float64:
		return fmt.Sprintf("%g", key), nil
	default:
		return fmt.Sprintf("%v", key), nil
	}
}

func (d *msgpackDecoder) decodeArray(n int) ([]any, error) {
	arr := make([]any, n)
	for i := 0; i < n; i++ {
		v, err := d.decode()
		if err != nil {
			return nil, err
		}
		arr[i] = v
	}
	return arr, nil
}

func (d *msgpackDecoder) decodeString(n int) (string, error) {
	if d.pos+n > len(d.data) {
		return "", errors.New("unexpected end of data")
	}
	s := string(d.data[d.pos : d.pos+n])
	d.pos += n
	return s, nil
}

func (d *msgpackDecoder) decodeBytes(n int) ([]byte, error) {
	if d.pos+n > len(d.data) {
		return nil, errors.New("unexpected end of data")
	}
	b := make([]byte, n)
	copy(b, d.data[d.pos:d.pos+n])
	d.pos += n
	return b, nil
}
