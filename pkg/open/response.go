package open

import (
	"encoding/json"
	"fmt"
)

// ApiResponse 开放平台 API 统一响应。
//
// 对应 Python 版 ApiResponse，封装响应的 code/message/data/success 字段。
// 开放平台所有接口返回统一结构：
//
//	{
//	  "code": 0,           // 0 表示成功，其他为错误码
//	  "message": "success", // 提示信息
//	  "data": { ... },      // 业务数据
//	  "success": true       // 是否成功
//	}
type ApiResponse struct {
	Code    int             `json:"code"`    // 错误码，0 表示成功
	Message string          `json:"message"` // 提示信息
	Data    json.RawMessage `json:"data"`    // 原始业务数据（按需解析）
	Success bool            `json:"success"` // 是否成功
}

// IsSuccess 判断请求是否成功（code == 0）。
func (r *ApiResponse) IsSuccess() bool {
	return r != nil && r.Code == 0
}

// UnmarshalData 将 data 字段解析到指定结构体。
//
// 使用示例：
//
//	var list struct{ List []AuthorizedName `json:"list"` }
//	if err := resp.UnmarshalData(&list); err != nil { ... }
func (r *ApiResponse) UnmarshalData(v any) error {
	if r == nil {
		return fmt.Errorf("open: response is nil")
	}
	if len(r.Data) == 0 {
		return nil
	}
	return json.Unmarshal(r.Data, v)
}

// String 返回响应的简洁字符串表示（用于日志）。
func (r *ApiResponse) String() string {
	if r == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ApiResponse{code=%d, success=%v, message=%s}", r.Code, r.Success, r.Message)
}
