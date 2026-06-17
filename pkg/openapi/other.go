// Package openapi — 其他模块。
//
// 翻译自 Python SDK goofish_api/api/other.py
package openapi

import "context"

// OtherService 其他模块，查询快递公司等辅助信息。
type OtherService struct {
	client *GoofishClient
}

// GetExpressCompanies 查询支持的快递公司列表。
//
// 对应 API: POST /api/open/express/companies
//
// 返回: 快递公司列表 JSON
//
// 示例:
//
//	resp, err := client.Other.GetExpressCompanies(ctx)
func (s *OtherService) GetExpressCompanies(ctx context.Context) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/express/companies", map[string]any{})
}
