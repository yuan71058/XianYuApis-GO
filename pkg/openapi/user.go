// Package openapi — 用户模块。
//
// 翻译自 Python SDK goofish_api/api/user.py
package openapi

import "context"

// UserService 用户模块，管理店铺授权。
type UserService struct {
	client *GoofishClient
}

// GetAuthorizeList 查询已授权的闲鱼店铺列表。
//
// 对应 API: POST /api/open/user/authorize/list
//
// 返回: 授权店铺列表 JSON
//
// 示例:
//
//	resp, err := client.User.GetAuthorizeList(ctx)
func (s *UserService) GetAuthorizeList(ctx context.Context) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/user/authorize/list", map[string]any{})
}
