package apis

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// GetToken 获取 WebSocket 连接所需的 accessToken。
//
// 该接口对应闲鱼 mtop.taobao.idlemessage.pc.login.token API。
// 如果返回令牌过期，会自动重试最多 3 次（循环重试，非递归，避免栈溢出）。
//
// 返回值:
//   - string: accessToken，用于 WebSocket /reg 注册
//   - error: 获取失败时的错误
func (api *XianyuAPI) GetToken(ctx context.Context) (string, error) {
	dataVal := fmt.Sprintf(`{"appKey":"444e9908a51d1cb236a27862abc769c9","deviceId":"%s"}`, api.deviceID)

	extra := map[string]string{
		"spm_cnt": "a21ybx.im.0.0",
		"spm_pre": "a21ybx.item.want.1.14ad3da6ALVq3n",
		"log_id":  "14ad3da6ALVq3n",
	}

	var lastErr error
	for retry := 0; retry < 3; retry++ {
		result, err := api.doMtopRequest(ctx,
			"mtop.taobao.idlemessage.pc.login.token", "1.0", dataVal, toValues(extra))
		if err != nil {
			lastErr = err
			continue
		}

		// 检查是否令牌过期
		if ret, ok := result["ret"].([]any); ok && len(ret) > 0 {
			if retStr, ok := ret[0].(string); ok && retStr == "令牌过期" {
				api.logger.Warn("token expired, retrying", zap.Int("retry", retry+1))
				time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
				continue
			}
		}

		// 提取 accessToken
		if data, ok := result["data"].(map[string]any); ok {
			if token, ok := data["accessToken"].(string); ok {
				return token, nil
			}
		}

		return "", fmt.Errorf("apis: token not found in response: %v", result)
	}

	return "", fmt.Errorf("apis: get token failed after 3 retries: %w", lastErr)
}

// RefreshToken 刷新登录态。
//
// 该接口对应 mtop.taobao.idlemessage.pc.loginuser.get API，
// 用于维持长期运行的 WebSocket 连接不掉线。建议每 10 分钟调用一次。
//
// 返回值:
//   - error: 刷新失败时的错误
func (api *XianyuAPI) RefreshToken(ctx context.Context) error {
	extra := map[string]string{
		"spm_cnt": "a21ybx.im.0.0",
		"spm_pre": "a21ybx.item.want.1.12523da6waCtUp",
		"log_id":  "12523da6waCtUp",
	}
	_, err := api.doMtopRequest(ctx,
		"mtop.taobao.idlemessage.pc.loginuser.get", "1.0", "{}", toValues(extra))
	if err != nil {
		return fmt.Errorf("apis: refresh token: %w", err)
	}
	return nil
}

// toValues 将 map[string]string 转换为 url.Values。
func toValues(m map[string]string) map[string][]string {
	if m == nil {
		return nil
	}
	v := make(map[string][]string, len(m))
	for k, val := range m {
		v[k] = []string{val}
	}
	return v
}
