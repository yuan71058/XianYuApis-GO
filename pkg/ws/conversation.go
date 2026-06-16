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
	SenderID   string `json:"send_user_id"`
	SenderName string `json:"send_user_name"`
	Message    any    `json:"message"`
}

// ListAllConversations 获取与指定用户的全部历史聊天记录。
func (ws *XianyuWS) ListAllConversations(ctx context.Context, cid string) ([]*ConversationMessage, error) {
	hdr := http.Header{}
	hdr.Set("Cookie", ws.api.CookieString())
	hdr.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	hdr.Set("Origin", "https://www.goofish.com")

	conn, _, err := websocket.Dial(ctx, wsBaseURL, &websocket.DialOptions{
		HTTPHeader:      hdr,
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		return nil, fmt.Errorf("conversation: dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	if err := initWSConnection(ctx, conn, ws.api, ws.deviceID); err != nil {
		return nil, fmt.Errorf("conversation: init: %w", err)
	}

	time.Sleep(3 * time.Second)

	var messages []*ConversationMessage
	hasMore := true
	nextCursor := float64(0)

	for hasMore {
		sendMid := util.GenerateMid()
		msg := map[string]any{
			"lwp":     "/r/MessageManager/listUserMessages",
			"headers": map[string]string{"mid": sendMid},
			"body":    []any{cid + "@goofish", false, float64(9007199254740991), float64(20), false},
		}

		if nextCursor > 0 {
			msg["body"].([]any)[2] = nextCursor
		}

		resp, err := wsSendAndWaitOnConn(ctx, conn, sendMid, msg, 10*time.Second)
		if err != nil {
			return nil, fmt.Errorf("conversation: request: %w", err)
		}

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

	msg := map[string]any{
		"lwp":     "/r/SingleChatConversation/create",
		"headers": map[string]string{"mid": util.GenerateMid()},
		"body": []any{
			map[string]any{
				"pairFirst":  toID + "@goofish",
				"pairSecond": ws.myID + "@goofish",
				"bizType":    "1",
				"extension":  map[string]any{"itemId": itemID},
				"ctx":        map[string]any{"appVersion": "1.0", "platform": "web"},
			},
		},
	}

	msgData, _ := json.Marshal(msg)
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()
	return ws.ws.Write(ctx, websocket.MessageText, msgData)
}

// initWSConnection 在给定连接上完成初始化。
func initWSConnection(ctx context.Context, conn *websocket.Conn, api *apis.XianyuAPI, deviceID string) error {
	token, err := api.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("initWSConnection: get token: %w", err)
	}

	regMid := util.GenerateMid()
	regMsg := map[string]any{
		"lwp": "/reg",
		"headers": map[string]string{
			"cache-header": "app-key token ua wv",
			"app-key":      "444e9908a51d1cb236a27862abc769c9",
			"token":        token,
			"ua":           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 DingTalk(2.1.5) OS(Windows/10) Browser(Chrome/146.0.0.0) DingWeb/2.1.5 IMPaaS DingWeb/2.1.5",
			"dt":           "j",
			"wv":           "im:3,au:3,sy:6",
			"sync":         "0,0;0;0;",
			"did":          deviceID,
			"mid":          regMid,
		},
	}

	_, err = wsSendAndWaitOnConn(ctx, conn, regMid, regMsg, 5*time.Second)
	if err != nil {
		return fmt.Errorf("initWSConnection: register: %w", err)
	}

	now := time.Now().UnixMilli()
	ackMid := util.GenerateMid()
	syncMsg := map[string]any{
		"lwp":     "/r/SyncStatus/ackDiff",
		"headers": map[string]string{"mid": ackMid},
		"body": []map[string]any{
			{"pipeline": "sync", "tooLong2Tag": "PNM,1", "channel": "sync", "topic": "sync",
				"highPts": 0, "pts": now * 1000, "seq": 0, "timestamp": now},
		},
	}
	if err := conn.Write(ctx, websocket.MessageText, mustMarshal(syncMsg)); err != nil {
		return fmt.Errorf("initWSConnection: send sync: %w", err)
	}

	return nil
}

// wsSendAndWaitOnConn 在指定连接上发送消息并等待匹配 mid 的响应。
func wsSendAndWaitOnConn(ctx context.Context, conn *websocket.Conn, mid string, msg map[string]any, timeout time.Duration) (map[string]any, error) {
	msgData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, msgData); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for response mid=%s", mid)
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}

		var rawMsg map[string]any
		if err := json.Unmarshal(data, &rawMsg); err != nil {
			continue
		}

		// 发送 ACK
		if headers, ok := rawMsg["headers"].(map[string]any); ok {
			ack := map[string]any{
				"code": 200,
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

		// 检查 mid 是否匹配
		if headers, ok := rawMsg["headers"].(map[string]any); ok {
			respMid := fmt.Sprintf("%v", headers["mid"])
			if respMid == mid {
				return rawMsg, nil
			}
		}
	}
}
