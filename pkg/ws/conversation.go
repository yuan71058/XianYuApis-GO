package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"nhooyr.io/websocket"
)

// ConversationMessage 历史消息数据结构。
type ConversationMessage struct {
	SenderID   string `json:"send_user_id"`   // 发送者用户 ID
	SenderName string `json:"send_user_name"` // 发送者名称
	Message    any    `json:"message"`        // 消息内容（JSON 解析后）
}

// ListAllConversations 获取与指定用户的全部历史聊天记录。
func (ws *XianyuWS) ListAllConversations(ctx context.Context, cid string) ([]*ConversationMessage, error) {
	hdr := http.Header{}
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	hdr.Set("Origin", "https://www.goofish.com")
	hdr.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	hdr.Set("Accept-Language", "zh-CN,zh;q=0.9")

	conn, _, err := websocket.Dial(ctx, wsBaseURL, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		return nil, fmt.Errorf("conversation: dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 初始化新连接
	if err := initWSConnection(ctx, conn, ws.api, ws.deviceID); err != nil {
		return nil, fmt.Errorf("conversation: init: %w", err)
	}

	sendMid := util.GenerateMid()
	msg := buildLWPMessage("/r/MessageManager/listUserMessages",
		map[string]string{"mid": sendMid},
		[]any{cid + "@goofish", false, float64(9007199254740991), float64(20), false},
	)

	var messages []*ConversationMessage
	hasMore := true
	nextCursor := float64(0)

	for hasMore {
		if nextCursor > 0 {
			msg["body"].([]any)[2] = nextCursor
		}

		msgData, _ := json.Marshal(msg)
		if err := conn.Write(ctx, websocket.MessageText, msgData); err != nil {
			return nil, fmt.Errorf("conversation: send request: %w", err)
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, fmt.Errorf("conversation: read response: %w", err)
		}

		var resp map[string]any
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("conversation: parse response: %w", err)
		}

		// 发送 ACK
		if headers, ok := resp["headers"].(map[string]any); ok {
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
			conn.Write(ctx, websocket.MessageText, ackData)
		}

		// 解析消息
		body, _ := resp["body"].(map[string]any)
		if body == nil {
			break
		}
		if hm, ok := body["hasMore"].(float64); ok {
			hasMore = hm == 1
		} else {
			hasMore = false
		}
		nextCursor, _ = body["nextCursor"].(float64)

		userMsgModels, _ := body["userMessageModels"].([]any)
		for _, umm := range userMsgModels {
			um, _ := umm.(map[string]any)
			msgBody, _ := um["message"].(map[string]any)
			extension, _ := msgBody["extension"].(map[string]any)

			contentData, _ := um["content"].(map[string]any)
			var msgJSON any
			if contentData != nil {
				custom, _ := contentData["custom"].(map[string]any)
				if custom != nil {
					dataBase64, _ := custom["data"].(string)
					if dataBase64 != "" {
						decoded, err := base64.StdEncoding.DecodeString(dataBase64)
						if err == nil {
							json.Unmarshal(decoded, &msgJSON)
						}
					}
				}
			}

			messages = append(messages, &ConversationMessage{
				SenderID:   toString(extension["senderUserId"]),
				SenderName: toString(extension["reminderTitle"]),
				Message:    msgJSON,
			})
		}
	}

	return messages, nil
}

// CreateChat 创建一个新的会话。
func (ws *XianyuWS) CreateChat(ctx context.Context, toID, itemID string) error {
	if itemID == "" {
		itemID = "891198795482"
	}

	msg := buildLWPMessage("/r/SingleChatConversation/create", nil, []any{
		map[string]any{
			"pairFirst":  toID + "@goofish",
			"pairSecond": ws.myID + "@goofish",
			"bizType":    "1",
			"extension":  map[string]any{"itemId": itemID},
			"ctx":        map[string]any{"appVersion": "1.0", "platform": "web"},
		},
	})

	msgData, _ := json.Marshal(msg)
	return ws.ws.Write(ctx, websocket.MessageText, msgData)
}

// initWSConnection 在给定 WebSocket 连接上完成初始化流程。
func initWSConnection(ctx context.Context, conn *websocket.Conn, api *apis.XianyuAPI, deviceID string) error {
	token, err := api.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("initWSConnection: get token: %w", err)
	}

	regMsg := buildLWPMessage("/reg", nil, map[string]any{
		"cache-header": "app-key token ua wv",
		"app-key":      "444e9908a51d1cb236a27862abc769c9",
		"token":        token,
		"ua":           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36 DingTalk(2.1.5) OS(Windows/10) Browser(Chrome/133.0.0.0) DingWeb/2.1.5 IMPaaS DingWeb/2.1.5",
		"dt":           "j",
		"wv":           "im:3,au:3,sy:6",
		"sync":         "0,0;0;0;",
		"did":          deviceID,
		"mid":          util.GenerateMid(),
	})
	if err := conn.Write(ctx, websocket.MessageText, mustMarshal(regMsg)); err != nil {
		return fmt.Errorf("initWSConnection: send reg: %w", err)
	}

	now := time.Now().UnixMilli()
	syncMsg := buildLWPMessage("/r/SyncStatus/ackDiff", nil, []map[string]any{
		{"pipeline": "sync", "tooLong2Tag": "PNM,1", "channel": "sync", "topic": "sync",
			"highPts": 0, "pts": now * 1000, "seq": 0, "timestamp": now},
	})
	if err := conn.Write(ctx, websocket.MessageText, mustMarshal(syncMsg)); err != nil {
		return fmt.Errorf("initWSConnection: send sync: %w", err)
	}

	return nil
}
