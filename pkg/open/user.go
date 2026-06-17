package open

import "context"

// UserService 用户模块，管理店铺授权。
//
// 对应 Python 版 User 类。
type UserService struct {
	client *Client
}

// GetAuthorizeList 查询已授权的闲鱼店铺列表。
//
// 对应 API: POST /api/open/user/authorize/list
//
// 返回值：
//   - *ApiResponse: 响应数据，data 字段为店铺列表
//   - error: 请求错误
//
// 示例：
//
//	resp, err := client.User.GetAuthorizeList(ctx)
//	if resp.IsSuccess() {
//	    var list struct{ List []map[string]any `json:"list"` }
//	    resp.UnmarshalData(&list)
//	}
func (s *UserService) GetAuthorizeList(ctx context.Context) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/user/authorize/list", map[string]any{})
}
