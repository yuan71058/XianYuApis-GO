package msg

import "time"

// NewTextMessage 创建文字消息。
func NewTextMessage(senderID, senderName, content, conversationID string) *Message {
	return &Message{
		SenderID:       senderID,
		SenderName:     senderName,
		Content:        content,
		MessageType:    MessageTypeText,
		ConversationID: conversationID,
		Timestamp:      time.Now(),
	}
}

// NewImageMessage 创建图片消息。
func NewImageMessage(senderID, senderName, imageURL, conversationID string, width, height int) *Message {
	return &Message{
		SenderID:       senderID,
		SenderName:     senderName,
		MessageType:    MessageTypeImage,
		ImageURL:       imageURL,
		ImageWidth:     width,
		ImageHeight:    height,
		ConversationID: conversationID,
		Timestamp:      time.Now(),
	}
}
