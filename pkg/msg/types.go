package msg

import (
	"fmt"
	"time"
)

// MessageType 消息类型枚举。
type MessageType int

const (
	// MessageTypeText 文字消息
	MessageTypeText MessageType = 1
	// MessageTypeImage 图片消息
	MessageTypeImage MessageType = 2
	// MessageTypeAudio 音频消息
	MessageTypeAudio MessageType = 26
)

func (mt MessageType) String() string {
	switch mt {
	case MessageTypeText:
		return "text"
	case MessageTypeImage:
		return "image"
	case MessageTypeAudio:
		return "audio"
	default:
		return fmt.Sprintf("unknown(%d)", mt)
	}
}

// Message 闲鱼消息结构体。
//
// 这是对外暴露的核心数据结构，所有消息回调函数均使用此类型。
type Message struct {
	SenderID       string      // 发送者用户 ID
	SenderName     string      // 发送者昵称
	Content        string      // 消息文本内容
	MessageType    MessageType // 消息类型
	ConversationID string      // 会话 ID
	ImageURL       string      // 图片 URL（图片消息时有效）
	ImageWidth     int         // 图片宽度
	ImageHeight    int         // 图片高度
	Timestamp      time.Time   // 消息时间
	Raw            any         // 原始消息数据（调试用）
}

// IsText 判断是否为文字消息。
func (m *Message) IsText() bool {
	return m.MessageType == MessageTypeText
}

// IsImage 判断是否为图片消息。
func (m *Message) IsImage() bool {
	return m.MessageType == MessageTypeImage
}
