package apis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/util"
	"go.uber.org/zap"
)

// GetToken 获取 WebSocket 连接所需的 accessToken。
//
// 该接口对应闲鱼 mtop.taobao.idlemessage.pc.login.token API。
// 如果返回令牌过期，会自动重试最多 3 次。
// 如果触发风控验证码（RGV587_ERROR::SM），会尝试调用验证码处理回调：
//   - 若设置了 CaptchaHandler，提取验证链接交给用户验证，验证后更新 Cookie 重试
//   - 若未设置 CaptchaHandler，等待 30 秒冷却后重试
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
			api.logger.Warn("get token request failed, retrying", zap.Int("retry", retry+1), zap.Error(err))
			time.Sleep(time.Duration(retry+1) * time.Second)
			continue
		}

		// 检查是否触发风控验证码
		if isCaptchaResponse(result) {
			verifyURL := extractVerifyURL(result)
			api.logger.Warn("触发风控验证码",
				zap.Int("retry", retry+1),
				zap.String("verifyURL", verifyURL),
			)

			// 如果设置了验证码处理回调，调用它
			if api.captchaHandler != nil && verifyURL != "" {
				fmt.Printf("\n  [风控] 触发验证码拦截，已提取验证链接 (%d/3)\n", retry+1)
				newCookieStr, err := api.captchaHandler(verifyURL)
				if err != nil {
					lastErr = fmt.Errorf("验证码处理失败: %w", err)
					fmt.Printf("  [风控] 验证码处理失败: %v\n", err)
					continue
				}
				if newCookieStr != "" {
					// 更新 Cookie
					api.UpdateCookies(util.ParseCookies(newCookieStr))
					// 刷新 _m_h5_tk
					api.RefreshMtopToken(ctx)
					fmt.Printf("  [风控] Cookie 已更新，重试中...\n")
					continue
				}
			}

			// 默认行为：等待冷却后重试
			cooldown := 30 * time.Second
			fmt.Printf("\n  [风控] 触发验证码拦截，等待 %v 后重试 (%d/3)...\n", cooldown, retry+1)
			if verifyURL != "" {
				fmt.Printf("  [风控] 验证链接（可手动在浏览器中打开）: %s\n", verifyURL)
			}
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

	return "", fmt.Errorf("apis: get token failed after 3 retries (可能触发风控，请稍后重新运行): %w", lastErr)
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

// extractVerifyURL 从风控响应中提取验证链接。
//
// 淘宝/闲鱼 mtop 风控响应格式:
//
//	{
//	  "ret": ["FAIL_SYS_USER_VALIDATE", "RGV587_ERROR::SM::..."],
//	  "data": {"url": "https://h5api.m.taobao.com/h5/mtop.../..."}
//	}
//
// 验证链接通常在 data.url 字段中。
func extractVerifyURL(result map[string]any) string {
	// 尝试 data.url
	if data, ok := result["data"].(map[string]any); ok {
		if url, ok := data["url"].(string); ok && url != "" {
			return url
		}
		// 尝试 data.data.url（嵌套结构）
		if innerData, ok := data["data"].(map[string]any); ok {
			if url, ok := innerData["url"].(string); ok && url != "" {
				return url
			}
		}
	}
	// 尝试 url（顶层）
	if url, ok := result["url"].(string); ok && url != "" {
		return url
	}
	return ""
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
