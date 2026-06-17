package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/msg"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

// recvLoop 后台消息接收循环。
//
// 与 Python 版 _recv_loop() 完全对齐:
//  1. 持续读取 WebSocket 消息
//  2. 发送 ACK
//  3. 检查 mid 是否匹配 pending 请求
//  4. 非请求响应则作为推送消息处理
func (ws *XianyuWS) recvLoop() {
	msgCount := 0
	for {
		// 使用带超时的 context，避免 Read 永久阻塞
		readCtx, readCancel := context.WithTimeout(ws.ctx, 60*time.Second)
		_, data, err := ws.ws.Read(readCtx)
		readCancel()

		if err != nil {
			// context 取消说明要退出
			if ws.ctx.Err() != nil {
				return
			}
			// 超时重试
			if readCtx.Err() == context.DeadlineExceeded && ws.ctx.Err() == nil {
				continue
			}
			ws.logger.Warn("ws: read error",
				zap.Int("msgsRead", msgCount),
				zap.Error(err),
			)
			select {
			case ws.recvErrCh <- fmt.Errorf("ws: read: %w", err):
			default:
			}
			// 标记连接断开
			ws.mu.Lock()
			ws.connected = false
			ws.mu.Unlock()
			return
		}
		msgCount++

		var rawMsg map[string]any
		if err := json.Unmarshal(data, &rawMsg); err != nil {
			ws.logger.Warn("ws: parse message failed",
				zap.Int("bytes", len(data)),
				zap.Error(err),
			)
			continue
		}

		// 记录收到的每条消息
		lwp, _ := rawMsg["lwp"].(string)
		ws.logger.Debug("ws: recv",
			zap.Int("seq", msgCount),
			zap.String("lwp", lwp),
			zap.Int("bytes", len(data)),
		)

		// 发送 ACK（与 Python 版 _handle_message 中 ACK 逻辑对齐）
		ws.sendACK(rawMsg)

		// 检查是否匹配 pending 请求（与 Python 版 if mid in self._pending 对齐）
		headers, _ := rawMsg["headers"].(map[string]any)
		if headers != nil {
			mid := fmt.Sprintf("%v", headers["mid"])
			if mid != "" {
				ws.pendingMu.Lock()
				ch, ok := ws.pending[mid]
				ws.pendingMu.Unlock()
				if ok {
					ws.logger.Info("ws: pending matched", zap.String("mid", mid))
					select {
					case ch <- rawMsg:
					default:
					}
					continue
				}
			}
		}

		// 非请求响应 —— 服务器主动推送的消息
		ws.handleMessage(rawMsg)
	}
}

// Start 等待消息循环结束。
func (ws *XianyuWS) Start() error {
	select {
	case err := <-ws.recvErrCh:
		return err
	case <-ws.ctx.Done():
		return ws.ctx.Err()
	}
}

// sendACK 发送 LWP 协议 ACK 确认。
func (ws *XianyuWS) sendACK(rawMsg map[string]any) {
	headers, ok := rawMsg["headers"].(map[string]any)
	if !ok || headers == nil {
		return
	}

	mid, _ := headers["mid"].(string)
	if mid == "" {
		if midVal, ok := headers["mid"].(float64); ok {
			mid = fmt.Sprintf("%.0f", midVal)
		} else {
			mid = util.GenerateMid()
		}
	}

	sid, _ := headers["sid"].(string)
	if sid == "" {
		if sidVal, ok := headers["sid"].(float64); ok {
			sid = fmt.Sprintf("%.0f", sidVal)
		}
	}

	ack := map[string]any{
		"code": 200,
		"headers": map[string]string{
			"mid": mid,
			"sid": sid,
		},
	}
	ackHeaders := ack["headers"].(map[string]string)
	for _, key := range []string{"app-key", "ua", "dt"} {
		if v, ok := headers[key]; ok {
			switch sv := v.(type) {
			case string:
				ackHeaders[key] = sv
			case float64:
				ackHeaders[key] = fmt.Sprintf("%.0f", sv)
			default:
				ackHeaders[key] = fmt.Sprintf("%v", sv)
			}
		}
	}

	ackData, _ := json.Marshal(ack)
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()
	if err := ws.ws.Write(ws.ctx, websocket.MessageText, ackData); err != nil {
		ws.logger.Warn("ws: send ack failed", zap.Error(err))
	}
}

// handleMessage 处理服务器推送的消息。
//
// 与 Python 版 _handle_push_message 和 PushMessageParser 对齐:
//  1. 从 body.syncPushPackage.data 中提取 data 字段
//  2. 解密: 先尝试 base64→UTF-8→JSON，失败则 base64→msgpack→JSON
//  3. 解析: 提取 msg["1"]["10"]["reminderContent"] 等字段
//  4. 解码内容: 从 msg["1"]["6"]["3"]["1"] 中 base64 解码出文本/图片
func (ws *XianyuWS) handleMessage(rawMsg map[string]any) {
	// 调试日志：记录收到的原始推送消息
	lwp, _ := rawMsg["lwp"].(string)
	ws.logger.Debug("ws: handleMessage called", zap.String("lwp", lwp))

	body, _ := rawMsg["body"].(map[string]any)
	if body == nil {
		ws.logger.Debug("ws: handleMessage skip - no body", zap.String("lwp", lwp))
		return
	}

	syncPkg, _ := body["syncPushPackage"].(map[string]any)
	if syncPkg == nil {
		ws.logger.Debug("ws: handleMessage skip - no syncPushPackage", zap.String("lwp", lwp))
		return
	}
	dataList, _ := syncPkg["data"].([]any)
	if len(dataList) == 0 {
		ws.logger.Debug("ws: handleMessage skip - empty dataList", zap.String("lwp", lwp))
		return
	}

	ws.logger.Info("ws: handleMessage processing", zap.Int("dataCount", len(dataList)))

	for i, item := range dataList {
		syncData, _ := item.(map[string]any)
		if syncData == nil {
			ws.logger.Debug("ws: handleMessage skip item - not map", zap.Int("index", i))
			continue
		}
		dataStr, _ := syncData["data"].(string)
		if dataStr == "" {
			ws.logger.Debug("ws: handleMessage skip item - empty data", zap.Int("index", i))
			continue
		}

		// 解密数据
		decrypted, err := decryptPushData(dataStr)
		if err != nil {
			// chatType 系统提示和 msgpack 解密失败都是正常情况，记录后跳过
			ws.logger.Debug("ws: handleMessage decrypt failed", zap.Int("index", i), zap.Error(err))
			continue
		}

		// 解析消息
		message := ws.parsePushMessage(decrypted)
		if message == nil {
			// /s/para 在线状态、非聊天 /s/sync 等消息无法解析为聊天消息，记录后跳过
			ws.logger.Debug("ws: handleMessage parsePushMessage returned nil", zap.Int("index", i))
			continue
		}

		// 调用回调
		ws.mu.RLock()
		handler := ws.msgHandler
		ws.mu.RUnlock()
		if handler != nil {
			ws.logger.Info("ws: calling msgHandler",
				zap.String("convID", message.ConversationID),
				zap.String("senderName", message.SenderName),
				zap.String("content", truncate(message.Content, 50)),
			)
			handler(message)
		} else {
			ws.logger.Warn("ws: msgHandler is nil, message dropped")
		}
	}
}

// decryptPushData 解密推送消息数据。
//
// 与 Python 版 PushMessageParser.decrypt_push_data 对齐:
// 先尝试 base64→UTF-8→JSON，失败则 base64→msgpack→JSON。
// chatType 消息是系统提示（如"欢迎语"），Python 版返回 None 跳过。
func decryptPushData(data string) (map[string]any, error) {
	// 方式1: base64 → UTF-8 → JSON
	if decoded, err := util.Base64DecodeUTF8(data); err == nil {
		var result map[string]any
		if err := json.Unmarshal([]byte(decoded), &result); err == nil {
			// chatType 消息是系统提示，跳过（与 Python 版一致）
			if _, ok := result["chatType"]; ok {
				return nil, fmt.Errorf("system chatType message, skip")
			}
			return result, nil
		}
	}

	// 方式2: base64 → msgpack → JSON（通过 util.Decrypt）
	decrypted, err := util.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(decrypted), &result); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted: %w", err)
	}

	return result, nil
}

// parsePushMessage 解析解密后的推送消息。
//
// 与 Python 版 PushMessageParser.parse 对齐:
//  1. 标准聊天消息: msg["1"]["10"]["reminderContent"]
//  2. 卡片消息: msg["1"]["6"]["3"]
//  3. 卡片更新: msg["4"]["reminderContent"]
func (ws *XianyuWS) parsePushMessage(msg map[string]any) *msg.Message {
	msg1, _ := msg["1"].(map[string]any)

	// 标准聊天消息: msg["1"]["10"]["reminderContent"]
	if msg1 != nil {
		msg10, _ := msg1["10"].(map[string]any)
		if msg10 != nil {
			if _, ok := msg10["reminderContent"]; ok {
				return ws.parseStandardChat(msg1, msg10)
			}
		}

		// 卡片消息: msg["1"]["6"]["3"]
		msg6, _ := msg1["6"].(map[string]any)
		if msg6 != nil {
			if _, ok := msg6["3"]; ok {
				return ws.parseCardChat(msg1, msg6)
			}
		}
	}

	// 卡片更新消息: msg["1"] 为字符串, msg["4"]["reminderContent"]
	if _, ok := msg["1"].(string); ok {
		msg4, _ := msg["4"].(map[string]any)
		if msg4 != nil {
			if _, ok := msg4["reminderContent"]; ok {
				return ws.parseCardUpdate(msg, msg4)
			}
		}
	}

	return nil
}

// parseStandardChat 解析标准聊天消息。
func (ws *XianyuWS) parseStandardChat(msg1, msg10 map[string]any) *msg.Message {
	senderIDRaw := toString(msg10["senderUserId"])
	senderID := strings.Split(senderIDRaw, "@")[0]
	senderName := toString(msg10["senderNick"])
	if senderName == "" {
		senderName = toString(msg10["reminderTitle"])
	}
	cidRaw := toString(msg1["2"])
	cid := strings.Split(cidRaw, "@")[0]

	// 解码内容（返回文本、图片URL列表、消息类型、首张图片宽高）
	textContent, images, msgTypeStr, imgW, imgH := decodeContent(msg1)

	// 如果没有解码出内容，使用 reminderContent
	if textContent == "" && len(images) == 0 {
		textContent = toString(msg10["reminderContent"])
		msgTypeStr = "text"
	}

	// 转换消息类型
	msgType := msg.MessageTypeText
	if msgTypeStr == "image" {
		msgType = msg.MessageTypeImage
	}

	return &msg.Message{
		SenderName:     senderName,
		SenderID:       senderID,
		Content:        textContent,
		ConversationID: cid,
		MessageType:    msgType,
		ImageURL:       strings.Join(images, ","),
		ImageWidth:     imgW,
		ImageHeight:    imgH,
	}
}

// parseCardChat 解析卡片消息。
func (ws *XianyuWS) parseCardChat(msg1, msg6 map[string]any) *msg.Message {
	msg6_3, _ := msg6["3"].(map[string]any)
	textContent := "[卡片消息]"
	if msg6_3 != nil {
		if t := toString(msg6_3["2"]); t != "" {
			textContent = t
		}
	}
	cidRaw := toString(msg1["2"])
	cid := strings.Split(cidRaw, "@")[0]

	return &msg.Message{
		SenderName:     "系统",
		SenderID:       "",
		Content:        textContent,
		ConversationID: cid,
		MessageType:    msg.MessageTypeText,
	}
}

// parseCardUpdate 解析卡片更新消息。
func (ws *XianyuWS) parseCardUpdate(rawMsg, msg4 map[string]any) *msg.Message {
	senderIDRaw := toString(msg4["senderUserId"])
	senderID := strings.Split(senderIDRaw, "@")[0]
	senderName := toString(msg4["reminderTitle"])
	if senderName == "" {
		senderName = "系统"
	}
	cidRaw := toString(rawMsg["2"])
	cid := strings.Split(cidRaw, "@")[0]

	return &msg.Message{
		SenderName:     senderName,
		SenderID:       senderID,
		Content:        toString(msg4["reminderContent"]),
		ConversationID: cid,
		MessageType:    msg.MessageTypeText,
	}
}

// decodeContent 从推送消息中解码内容。
//
// 与 Python 版 PushMessageParser._decode_content 对齐:
// 闲鱼消息内容有两种格式:
//   - 格式1: msg6_3["1"] 为 base64 字符串，解码后得到 {"contentType":1,"text":{...}}（文字消息常用）
//   - 格式2: msg6_3["5"] 为 JSON 字符串，直接包含 {"contentType":2,"image":{...}}（图片消息常用）
//
// 返回: (文本内容, 图片URL列表, 消息类型, 首图宽度, 首图高度)
func decodeContent(msg1 map[string]any) (string, []string, string, int, int) {
	msg6, _ := msg1["6"].(map[string]any)
	if msg6 == nil {
		return "", nil, "text", 0, 0
	}
	msg6_3, _ := msg6["3"].(map[string]any)
	if msg6_3 == nil {
		return "", nil, "text", 0, 0
	}

	// 优先尝试格式2: msg6_3["5"] 直接 JSON（图片/位置/链接等消息常用此格式）
	if field5, ok := msg6_3["5"].(string); ok && field5 != "" {
		var data map[string]any
		if err := json.Unmarshal([]byte(field5), &data); err == nil {
			contentType := 0
			if ct, ok := data["contentType"].(float64); ok {
				contentType = int(ct)
			}

			switch contentType {
			case 1:
				// 文本消息
				if textObj, ok := data["text"].(map[string]any); ok {
					return toString(textObj["text"]), nil, "text", 0, 0
				}
			case 2:
				// 图片消息
				if imageObj, ok := data["image"].(map[string]any); ok {
					pics, _ := imageObj["pics"].([]any)
					var urls []string
					var w, h int
					for i, p := range pics {
						if picMap, ok := p.(map[string]any); ok {
							if url := toString(picMap["url"]); url != "" {
								urls = append(urls, url)
							}
							if i == 0 {
								if pw, ok := picMap["width"].(float64); ok {
									w = int(pw)
								}
								if ph, ok := picMap["height"].(float64); ok {
									h = int(ph)
								}
							}
						}
					}
					if len(urls) > 0 {
						return "", urls, "image", w, h
					}
				}
				if picURL := toString(data["picUrl"]); picURL != "" {
					return "", []string{picURL}, "image", 0, 0
				}
			case 3:
				// 语音消息
				return "[语音消息]", nil, "text", 0, 0
			case 7:
				// 商品卡片消息
				if itemCard, ok := data["itemCard"].(map[string]any); ok {
					if item, ok := itemCard["item"].(map[string]any); ok {
						title := toString(item["title"])
						price := toString(item["price"])
						pic := toString(item["mainPic"])
						itemID := toString(item["itemId"])
						tip := toString(itemCard["itemTip"])
						text := fmt.Sprintf("[%s] %s %s", tip, title, price)
						if itemID != "" {
							text += fmt.Sprintf(" (id:%s)", itemID)
						}
						var imgs []string
						if pic != "" {
							imgs = []string{pic}
						}
						return text, imgs, "text", 0, 0
					}
				}
			case 17:
				// 转账消息
				if transfer, ok := data["transfer"].(map[string]any); ok {
					// 从 transfer 中提取转账信息
					amount := toString(transfer["amount"])
					memo := toString(transfer["memo"])
					tradeNO := toString(transfer["tradeNo"])
					text := "[转账]"
					if amount != "" {
						text = fmt.Sprintf("[转账] ¥%s", amount)
					}
					if memo != "" {
						text += " " + memo
					}
					if tradeNO != "" {
						text += fmt.Sprintf(" (交易号:%s)", tradeNO)
					}
					// 如果 transfer 中没有 amount，尝试从 action.page.url 提取 tradeNO
					if tradeNO == "" {
						if action, ok := transfer["action"].(map[string]any); ok {
							if page, ok := action["page"].(map[string]any); ok {
								url := toString(page["url"])
								if strings.Contains(url, "tradeNO=") {
									parts := strings.Split(url, "tradeNO=")
									if len(parts) > 1 {
										tradeNO = strings.SplitN(parts[1], "&", 2)[0]
										text += fmt.Sprintf(" (交易号:%s)", tradeNO)
									}
								}
							}
						}
					}
					return text, nil, "text", 0, 0
				}
			case 30:
				// 位置消息
				if locCard, ok := data["locationCard"].(map[string]any); ok {
					// 从 URL 中提取经纬度
					url := ""
					if action, ok := locCard["action"].(map[string]any); ok {
						if page, ok := action["page"].(map[string]any); ok {
							url = toString(page["url"])
						}
					}
					lat, lng := extractLatLng(url)
					text := "[位置]"
					if lat != "" && lng != "" {
						text = fmt.Sprintf("[位置] 经度:%s 纬度:%s", lng, lat)
					}
					return text, nil, "text", 0, 0
				}
			default:
				// 未知 contentType，打印调试信息
				dataJSON, _ := json.Marshal(data)
				preview := string(dataJSON)
				if len(preview) > 300 {
					preview = preview[:300] + "..."
				}
				fmt.Printf("[decodeContent] 未知 contentType=%d, data: %s\n", contentType, preview)
			}
		}
	}

	// 格式1: msg6_3["1"] 为 base64 字符串（文字消息常用此格式）
	customData, _ := msg6_3["1"].(string)
	if customData == "" {
		return "", nil, "text", 0, 0
	}

	decoded, err := util.Base64DecodeUTF8(customData)
	if err != nil {
		return "", nil, "text", 0, 0
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return "", nil, "text", 0, 0
	}

	contentType := 0
	if ct, ok := data["contentType"].(float64); ok {
		contentType = int(ct)
	}

	// 文本消息
	if contentType == 1 {
		if textObj, ok := data["text"].(map[string]any); ok {
			return toString(textObj["text"]), nil, "text", 0, 0
		}
		if textStr, ok := data["text"].(string); ok {
			return textStr, nil, "text", 0, 0
		}
	}

	// 图片消息: image.pics 数组格式
	if contentType == 2 {
		if imageObj, ok := data["image"].(map[string]any); ok {
			pics, _ := imageObj["pics"].([]any)
			var urls []string
			var w, h int
			for i, p := range pics {
				if picMap, ok := p.(map[string]any); ok {
					if url := toString(picMap["url"]); url != "" {
						urls = append(urls, url)
					}
					if i == 0 {
						if pw, ok := picMap["width"].(float64); ok {
							w = int(pw)
						}
						if ph, ok := picMap["height"].(float64); ok {
							h = int(ph)
						}
					}
				}
			}
			if len(urls) > 0 {
				return "", urls, "image", w, h
			}
		}
		// picUrl 回退
		if picURL := toString(data["picUrl"]); picURL != "" {
			return "", []string{picURL}, "image", 0, 0
		}
	}

	// 语音消息
	if contentType == 3 {
		return "[语音消息]", nil, "text", 0, 0
	}

	// 回退1: 尝试 text 字段
	if textObj, ok := data["text"].(map[string]any); ok {
		return toString(textObj["text"]), nil, "text", 0, 0
	}

	// 回退2: 尝试 picUrl 字段（与 Python 版对齐）
	if picURL := toString(data["picUrl"]); picURL != "" {
		return "", []string{picURL}, "image", 0, 0
	}

	return "", nil, "text", 0, 0
}

// extractLatLng 从 URL 查询参数中提取经纬度。
// 闲鱼位置消息的 URL 格式: ...&longitude=xxx&latitude=xxx...
func extractLatLng(urlStr string) (lat, lng string) {
	if urlStr == "" {
		return "", ""
	}
	// 查找 latitude 和 longitude 参数
	for _, part := range strings.Split(urlStr, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "latitude":
			lat = kv[1]
		case "longitude":
			lng = kv[1]
		}
	}
	return lat, lng
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

// truncate 截断字符串。
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
