package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/cv-cat/xianyuapis/pkg/util"
	"nhooyr.io/websocket"
)

// SendText 发送文字消息。
func (ws *XianyuWS) SendText(ctx context.Context, cid, toID, text string) error {
	payload := map[string]any{
		"contentType": 1,
		"text":        map[string]any{"text": text},
	}
	return ws.sendEncodedMessage(ctx, cid, toID, 1, payload)
}

// SendImage 发送图片消息。
func (ws *XianyuWS) SendImage(ctx context.Context, cid, toID, imageURL string, width, height int) error {
	payload := map[string]any{
		"contentType": 2,
		"image": map[string]any{
			"pics": []map[string]any{
				{"type": 0, "url": imageURL, "width": width, "height": height},
			},
		},
	}
	return ws.sendEncodedMessage(ctx, cid, toID, 2, payload)
}

// sendEncodedMessage 编码并发送 LWP 消息。
func (ws *XianyuWS) sendEncodedMessage(ctx context.Context, cid, toID string,
	contentType int, contentPayload map[string]any,
) error {
	contentJSON, _ := json.Marshal(contentPayload)
	contentBase64 := base64.StdEncoding.EncodeToString(contentJSON)

	lwpMsg := map[string]any{
		"lwp":     "/r/MessageSend/sendByReceiverScope",
		"headers": map[string]string{"mid": util.GenerateMid()},
		"body": []any{
			map[string]any{
				"uuid":             util.GenerateUUID(),
				"cid":              cid + "@goofish",
				"conversationType": float64(1),
				"content": map[string]any{
					"contentType": float64(101),
					"custom": map[string]any{
						"type": float64(contentType),
						"data": contentBase64,
					},
				},
				"redPointPolicy":       float64(0),
				"extension":            map[string]any{"extJson": "{}"},
				"ctx":                  map[string]any{"appVersion": "1.0", "platform": "web"},
				"mtags":                map[string]any{},
				"msgReadStatusSetting": float64(1),
			},
			map[string]any{
				"actualReceivers": []string{toID + "@goofish", ws.myID + "@goofish"},
			},
		},
	}

	msgData, _ := json.Marshal(lwpMsg)
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()
	return ws.ws.Write(ctx, websocket.MessageText, msgData)
}
