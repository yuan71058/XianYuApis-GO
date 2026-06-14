package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cv-cat/xianyuapis/pkg/msg"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

// Start 启动消息接收循环。
//
// 该函数阻塞运行，持续读取 WebSocket 消息并处理。
// 返回条件: 连接关闭、context 被取消、致命错误
func (ws *XianyuWS) Start() error {
	for {
		select {
		case <-ws.ctx.Done():
			return ws.ctx.Err()
		default:
		}

		_, data, err := ws.ws.Read(ws.ctx)
		if err != nil {
			return fmt.Errorf("ws: read: %w", err)
		}

		// 解析 LWP 消息
		var rawMsg map[string]any
		if err := json.Unmarshal(data, &rawMsg); err != nil {
			ws.logger.Warn("failed to parse message", zap.Error(err))
			continue
		}

		// 发送 ACK
		if err := ws.sendACK(rawMsg); err != nil {
			ws.logger.Warn("send ack failed", zap.Error(err))
		}

		// 处理消息
		ws.handleMessage(rawMsg)
	}
}

// sendACK 发送 LWP 协议 ACK 确认。
func (ws *XianyuWS) sendACK(rawMsg map[string]any) error {
	headers, _ := rawMsg["headers"].(map[string]any)
	if headers == nil {
		return nil
	}

	ack := map[string]any{
		"code": float64(200),
		"headers": map[string]any{
			"mid": headers["mid"],
			"sid": headers["sid"],
		},
	}
	for _, key := range []string{"app-key", "ua", "dt"} {
		if v, ok := headers[key]; ok {
			ack["headers"].(map[string]any)[key] = v
		}
	}

	ackData, _ := json.Marshal(ack)
	return ws.ws.Write(ws.ctx, websocket.MessageText, ackData)
}

// handleMessage 处理收到的消息。
func (ws *XianyuWS) handleMessage(rawMsg map[string]any) {
	body, _ := rawMsg["body"].(map[string]any)
	if body == nil {
		return
	}

	// 尝试解析 syncPushPackage（无需解密）
	if syncPush, ok := body["syncPushPackage"].(map[string]any); ok {
		ws.handleSyncPush(syncPush)
		return
	}

	// 获取原始加密数据
	encryptedData := extractEncryptedData(rawMsg)
	if encryptedData == "" {
		return
	}

	// 解密
	decrypted, err := util.Decrypt(encryptedData)
	if err != nil {
		ws.logger.Warn("decrypt message failed", zap.Error(err))
		return
	}

	// 解析 JSON
	var data map[string]any
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		ws.logger.Warn("unmarshal decrypted message failed", zap.Error(err))
		return
	}

	// 提取消息字段
	msgData := extractMessageFields(data)
	if msgData == nil {
		return
	}

	message := &msg.Message{
		SenderName:     toString(msgData["reminderTitle"]),
		SenderID:       toString(msgData["senderUserId"]),
		Content:        toString(msgData["reminderContent"]),
		ConversationID: extractConversationID(data),
	}

	// 调用回调
	ws.mu.RLock()
	handler := ws.msgHandler
	ws.mu.RUnlock()

	if handler != nil {
		handler(message)
	}
}

// handleSyncPush 处理无需解密的 syncPushPackage 消息。
func (ws *XianyuWS) handleSyncPush(syncPush map[string]any) {
	dataArr, _ := syncPush["data"].([]any)
	for _, item := range dataArr {
		dataMap, _ := item.(map[string]any)
		dataStr, _ := dataMap["data"].(string)
		if dataStr == "" {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			continue
		}
		msgData := extractMessageFields(data)
		if msgData == nil {
			continue
		}
		message := &msg.Message{
			SenderName:     toString(msgData["reminderTitle"]),
			SenderID:       toString(msgData["senderUserId"]),
			Content:        toString(msgData["reminderContent"]),
			ConversationID: extractConversationID(data),
		}
		ws.mu.RLock()
		handler := ws.msgHandler
		ws.mu.RUnlock()
		if handler != nil {
			handler(message)
		}
	}
}

// extractEncryptedData 从原始 LWP 消息中提取加密数据字符串。
func extractEncryptedData(rawMsg map[string]any) string {
	// 路径 1: body 是 map，直接取 data 字段
	body, _ := rawMsg["body"].(map[string]any)
	if body != nil {
		if data, ok := body["data"].(string); ok {
			return data
		}
		if syncPush, ok := body["syncPushPackage"].(map[string]any); ok {
			if dataArr, ok := syncPush["data"].([]any); ok && len(dataArr) > 0 {
				if dataMap, ok := dataArr[0].(map[string]any); ok {
					if s, ok := dataMap["data"].(string); ok {
						return s
					}
				}
			}
		}
	}

	// 路径 2: body 是数组
	if items, ok := rawMsg["body"].([]any); ok && len(items) > 0 {
		if s, ok := items[0].(string); ok {
			return s
		}
	}

	return ""
}

// extractMessageFields 从解密后的 JSON 数据中提取消息字段。
func extractMessageFields(data map[string]any) map[string]any {
	outer, _ := data["1"].(map[string]any)
	if outer == nil {
		return nil
	}
	result, _ := outer["10"].(map[string]any)
	return result
}

// extractConversationID 从解密后的 JSON 数据中提取会话 ID。
func extractConversationID(data map[string]any) string {
	outer, _ := data["1"].(map[string]any)
	if outer == nil {
		return ""
	}
	cidRaw, _ := outer["2"].(string)
	if idx := strings.Index(cidRaw, "@"); idx > 0 {
		return cidRaw[:idx]
	}
	return cidRaw
}

// toString 安全地将任意类型转为 string。
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%g", s)
	default:
		return fmt.Sprintf("%v", s)
	}
}
