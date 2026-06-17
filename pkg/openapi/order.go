// Package openapi — 订单模块。
//
// 翻译自 Python SDK goofish_api/api/order.py
// 包含订单查询、详情、卡密查询、发货 4 个方法。
package openapi

import "context"

// OrderService 订单模块，管理订单查询和发货。
type OrderService struct {
	client *GoofishClient
}

// GetOrderListRequest 查询订单列表请求参数。
type GetOrderListRequest struct {
	OrderStatus  *OrderStatus  `json:"order_status,omitempty"`  // 订单状态（可选）
	RefundStatus *RefundStatus `json:"refund_status,omitempty"` // 退款状态（可选）
	OrderTime    []int64       `json:"order_time,omitempty"`    // 下单时间范围 [开始, 结束]
	PayTime      []int64       `json:"pay_time,omitempty"`      // 付款时间范围
	ConsigTime   []int64       `json:"consign_time,omitempty"`  // 发货时间范围
	ConfirmTime  []int64       `json:"confirm_time,omitempty"`  // 确认收货时间范围
	RefundTime   []int64       `json:"refund_time,omitempty"`   // 退款时间范围
	UpdateTime   []int64       `json:"update_time,omitempty"`   // 更新时间范围
	PageNo       int           `json:"page_no"`                 // 页码
	PageSize     int           `json:"page_size"`               // 每页数量
}

// GetOrderList 查询订单列表。
//
// 对应 API: POST /api/open/order/list
//
// 参数:
//   - req: 查询条件，PageNo 和 PageSize 为必填
//
// 示例:
//
//	resp, err := client.Order.GetOrderList(ctx, &openapi.GetOrderListRequest{
//	    PageNo: 1, PageSize: 50,
//	    OrderStatus: openapi.OrderStatus(OrderStatusPendingShipment),
//	})
func (s *OrderService) GetOrderList(ctx context.Context, req *GetOrderListRequest) (map[string]any, error) {
	if req.PageNo == 0 {
		req.PageNo = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 50
	}
	return s.client.request(ctx, "/api/open/order/list", req)
}

// GetOrderDetail 查询订单详情。
//
// 对应 API: POST /api/open/order/detail
//
// 参数:
//   - orderNo: 订单号
func (s *OrderService) GetOrderDetail(ctx context.Context, orderNo string) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/order/detail", map[string]any{
		"order_no": orderNo,
	})
}

// KamOrderList 查询订单卡密信息。
//
// 对应 API: POST /api/open/order/kam/list
//
// 参数:
//   - orderNo: 订单号
func (s *OrderService) KamOrderList(ctx context.Context, orderNo string) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/order/kam/list", map[string]any{
		"order_no": orderNo,
	})
}

// OrderShipRequest 订单发货请求参数。
type OrderShipRequest struct {
	OrderNo        string `json:"order_no"`         // 订单号（必需）
	WaybillNo      string `json:"waybill_no"`       // 运单号（必需）
	ExpressName    string `json:"express_name"`     // 快递公司名称（必需）
	ExpressCode    string `json:"express_code"`     // 快递公司编码（必需）
	ShipName       string `json:"ship_name,omitempty"`       // 收货人姓名
	ShipMobile     string `json:"ship_mobile,omitempty"`     // 收货人手机
	ShipDistrictID *int64 `json:"ship_district_id,omitempty"` // 收货地区ID
	ShipProvName   string `json:"ship_prov_name,omitempty"`   // 省份
	ShipCityName   string `json:"ship_city_name,omitempty"`   // 城市
	ShipAreaName   string `json:"ship_area_name,omitempty"`   // 区县
}

// OrderShip 订单物流发货。
//
// 对应 API: POST /api/open/order/ship
//
// 参数:
//   - req: 发货信息
//
// 示例:
//
//	resp, err := client.Order.OrderShip(ctx, &openapi.OrderShipRequest{
//	    OrderNo:     "1339920336328048683",
//	    WaybillNo:   "25051016899982",
//	    ExpressName: "其他",
//	    ExpressCode: "qita",
//	    ShipName:    "张三",
//	    ShipMobile:  "13800138000",
//	})
func (s *OrderService) OrderShip(ctx context.Context, req *OrderShipRequest) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/order/ship", req)
}
