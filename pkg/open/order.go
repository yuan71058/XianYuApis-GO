package open

import (
	"context"
	"fmt"
)

// OrderService 订单模块，提供订单查询、发货处理、卡密管理等功能。
//
// 对应 Python 版 Order 类，共 4 个方法。
type OrderService struct {
	client *Client
}

// GetOrderListRequest 查询订单列表的请求参数。
type GetOrderListRequest struct {
	OrderStatus  *OrderStatus  `json:"order_status,omitempty"`  // 订单状态（可选）
	RefundStatus *RefundStatus `json:"refund_status,omitempty"` // 退款状态（可选）
	OrderTime    []int64       `json:"order_time,omitempty"`    // 订单时间范围 [start, end]
	PayTime      []int64       `json:"pay_time,omitempty"`      // 支付时间范围
	ConsignTime  []int64       `json:"consign_time,omitempty"`  // 发货时间范围
	ConfirmTime  []int64       `json:"confirm_time,omitempty"`  // 确认收货时间范围
	RefundTime   []int64       `json:"refund_time,omitempty"`   // 退款时间范围
	UpdateTime   []int64       `json:"update_time,omitempty"`   // 更新时间范围
	PageNo       int           `json:"page_no"`                 // 页码
	PageSize     int           `json:"page_size"`               // 每页大小
}

// GetOrderList 查询订单列表。
//
// 对应 API: POST /api/open/order/list
func (s *OrderService) GetOrderList(ctx context.Context, req *GetOrderListRequest) (*ApiResponse, error) {
	if req == nil {
		req = &GetOrderListRequest{}
	}
	if req.PageNo <= 0 {
		req.PageNo = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	return s.client.request(ctx, "/api/open/order/list", req)
}

// GetOrderDetail 查询订单详情。
//
// 对应 API: POST /api/open/order/detail
//
// 参数：
//   - orderNo: 闲鱼订单号
func (s *OrderService) GetOrderDetail(ctx context.Context, orderNo string) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/order/detail", map[string]any{
		"order_no": orderNo,
	})
}

// KamOrderList 查询订单卡密列表。
//
// 对应 API: POST /api/open/order/kam/list
//
// 参数：
//   - orderNo: 闲鱼订单号
func (s *OrderService) KamOrderList(ctx context.Context, orderNo string) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/order/kam/list", map[string]any{
		"order_no": orderNo,
	})
}

// OrderShipRequest 订单物流发货请求参数。
type OrderShipRequest struct {
	OrderNo        string `json:"order_no"`                   // 订单号（必填）
	WaybillNo      string `json:"waybill_no"`                 // 运单号（必填）
	ExpressName    string `json:"express_name"`               // 快递公司名称（必填）
	ExpressCode    string `json:"express_code"`               // 快递公司编码（必填）
	ShipName       string `json:"ship_name,omitempty"`        // 收货人姓名
	ShipMobile     string `json:"ship_mobile,omitempty"`      // 收货人手机号
	ShipDistrictID *int64 `json:"ship_district_id,omitempty"` // 收货人地区 ID（无则必传省市区）
	ShipProvName   string `json:"ship_prov_name,omitempty"`   // 收货人所在省份
	ShipCityName   string `json:"ship_city_name,omitempty"`   // 收货人所在城市
	ShipAreaName   string `json:"ship_area_name,omitempty"`   // 收货人所在区县
}

// OrderShip 订单物流发货。
//
// 对应 API: POST /api/open/order/ship
//
// 必填字段：OrderNo、WaybillNo、ExpressName、ExpressCode
// 若 ShipDistrictID 未传，则 ShipProvName/ShipCityName/ShipAreaName 必填。
func (s *OrderService) OrderShip(ctx context.Context, req *OrderShipRequest) (*ApiResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("open: order ship request is nil")
	}
	if req.OrderNo == "" || req.WaybillNo == "" || req.ExpressName == "" || req.ExpressCode == "" {
		return nil, fmt.Errorf("open: order ship requires order_no, waybill_no, express_name, express_code")
	}
	if req.ShipDistrictID == nil &&
		(req.ShipProvName == "" || req.ShipCityName == "" || req.ShipAreaName == "") {
		return nil, fmt.Errorf("open: order ship requires either ship_district_id or prov/city/area")
	}
	return s.client.request(ctx, "/api/open/order/ship", req)
}
