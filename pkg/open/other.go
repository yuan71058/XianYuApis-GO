package open

import "context"

// OtherService 其他模块，提供快递公司等辅助查询。
//
// 对应 Python 版 Other 类。
type OtherService struct {
	client *Client
}

// GetExpressCompanies 获取支持的快递公司列表。
//
// 对应 API: POST /api/open/express/companies
//
// 返回值：
//   - *ApiResponse: 响应数据，data 字段为快递公司列表
//   - error: 请求错误
//
// 示例：
//
//	resp, err := client.Other.GetExpressCompanies(ctx)
func (s *OtherService) GetExpressCompanies(ctx context.Context) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/express/companies", map[string]any{})
}
