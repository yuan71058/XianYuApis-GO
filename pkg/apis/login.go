package apis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// GetToken 获取 WebSocket 连接所需的 accessToken。
//
// 该接口对应闲鱼 mtop.taobao.idlemessage.pc.login.token API。
// 如果返回令牌过期，会自动重试最多 3 次。
// 如果触发风控验证码（RGV587_ERROR::SM），会等待冷却后重试。
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

	for retry := 0; retry < 3; retry++ {
		result, err := api.doMtopRequest(ctx,
			"mtop.taobao.idlemessage.pc.login.token", "1.0", dataVal, toValues(extra))
		if err != nil {
			api.logger.Warn("get token request failed, retrying", zap.Int("retry", retry+1), zap.Error(err))
			time.Sleep(time.Duration(retry+1) * time.Second)
			continue
		}

		// 检查是否触发风控验证码
		if isCaptchaResponse(result) {
			api.logger.Warn("触发风控验证码，等待冷却后重试",
				zap.Int("retry", retry+1),
			)
			// 风控验证码需要重新登录获取新 Cookie
			// 等待 30 秒冷却后重试
			cooldown := 30 * time.Second
			fmt.Printf("\n  [风控] 触发验证码拦截，等待 %v 后重试 (%d/3)...\n", cooldown, retry+1)
			fmt.Println("  [提示] 请重新扫码登录以获取新的 Cookie 和 Token")
			time.Sleep(cooldown)

			// 刷新 _m_h5_tk
			api.RefreshMtopToken(ctx)
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

	return "", fmt.Errorf("获取 WS Token 失败（3 次重试后）：可能触发风控验证，请重新扫码登录获取新的 Cookie 和 Token")
}

// isCaptchaResponse 检查 mtop 响应是否为风控验证码拦截。
func isCaptchaResponse(result map[string]any) bool {
	ret, _ := result["ret"].([]any)
	for _, r := range ret {
		if s, ok := r.(string); ok {
			sUpper := strings.ToUpper(s)
			if strings.Contains(sUpper, "FAIL_SYS_USER_VALIDATE") ||
				strings.Contains(sUpper, "RGV587_ERROR") {
				return true
			}
		}
	}
	return false
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
