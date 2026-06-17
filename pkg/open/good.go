package open

import (
	"context"
	"fmt"
)

// GoodService 商品模块，提供商品 CRUD、类目管理、属性查询等功能。
//
// 对应 Python 版 Good 类，共 12 个方法。
type GoodService struct {
	client *Client
}

// --- 商品类目与属性 ---

// GetProductCategoryList 查询商品类目。
//
// 对应 API: POST /api/open/product/category/list
//
// 参数：
//   - itemBizType:    商品类型（必需）
//   - spBizType:      行业类型（可选）
//   - flashSaleType:  闲鱼特卖类型（可选）
func (s *GoodService) GetProductCategoryList(
	ctx context.Context,
	itemBizType ItemBizType,
	spBizType *SpBizType,
	flashSaleType *FlashSaleType,
) (*ApiResponse, error) {
	data := map[string]any{
		"item_biz_type":   itemBizType,
		"sp_biz_type":     spBizType,
		"flash_sale_type": flashSaleType,
	}
	return s.client.request(ctx, "/api/open/product/category/list", data)
}

// GetProductPvList 查询商品属性。
//
// 对应 API: POST /api/open/product/pv/list
//
// 参数：
//   - itemBizType:    商品类型（必需）
//   - spBizType:      行业类型（必需）
//   - channelCatID:   渠道类目 ID（必需）
//   - subPropertyID:  属性值 ID（可选）
func (s *GoodService) GetProductPvList(
	ctx context.Context,
	itemBizType ItemBizType,
	spBizType SpBizType,
	channelCatID string,
	subPropertyID *string,
) (*ApiResponse, error) {
	data := map[string]any{
		"item_biz_type":   itemBizType,
		"sp_biz_type":     spBizType,
		"channel_cat_id":  channelCatID,
		"sub_property_id": subPropertyID,
	}
	return s.client.request(ctx, "/api/open/product/pv/list", data)
}

// --- 商品查询 ---

// GetProductListRequest 查询商品列表的请求参数。
type GetProductListRequest struct {
	OnlineTime    []int64        `json:"online_time,omitempty"`    // 上架时间范围 [start, end]
	OfflineTime   []int64        `json:"offline_time,omitempty"`   // 下架时间范围
	SoldTime      []int64        `json:"sold_time,omitempty"`      // 售罄时间范围
	UpdateTime    []int64        `json:"update_time,omitempty"`    // 更新时间范围
	CreateTime    []int64        `json:"create_time,omitempty"`    // 创建时间范围
	ProductStatus *ProductStatus `json:"product_status,omitempty"` // 商品状态
	SaleStatus    *SaleStatus    `json:"sale_status,omitempty"`    // 销售状态
	PageNo        int            `json:"page_no"`                  // 页码 >= 1 <= 100
	PageSize      int            `json:"page_size"`                // 每页数量 >= 1 <= 100
}

// GetProductList 查询商品列表。
//
// 对应 API: POST /api/open/product/list
func (s *GoodService) GetProductList(ctx context.Context, req *GetProductListRequest) (*ApiResponse, error) {
	if req == nil {
		req = &GetProductListRequest{}
	}
	if req.PageNo <= 0 {
		req.PageNo = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	return s.client.request(ctx, "/api/open/product/list", req)
}

// GetProductDetail 查询商品详情。
//
// 对应 API: POST /api/open/product/detail
//
// 参数：
//   - productID: 管家商品 ID
func (s *GoodService) GetProductDetail(ctx context.Context, productID int64) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/detail", map[string]any{
		"product_id": productID,
	})
}

// GetProductSkuList 查询商品规格。
//
// 对应 API: POST /api/open/product/sku/list
//
// 参数：
//   - productIDs: 管家商品 ID 列表（最多 100 个）
func (s *GoodService) GetProductSkuList(ctx context.Context, productIDs []int64) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/sku/list", map[string]any{
		"product_id": productIDs,
	})
}

// --- 商品创建与发布 ---

// CreateProduct 创建商品（单个）。
//
// 对应 API: POST /api/open/product/create
//
// 参数：
//   - productData: 商品数据（结构参考开放平台文档，包含 item_biz_type、sp_biz_type、
//     channel_cat_id、channel_pv、price、original_price、stock、publish_shop 等）
//
// productData 可以是 map[string]any 或自定义结构体，会原样 JSON 序列化。
func (s *GoodService) CreateProduct(ctx context.Context, productData any) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/create", productData)
}

// ProductBatchCreate 批量创建商品。
//
// 对应 API: POST /api/open/product/batchCreate
//
// 注意：
//   - 字段参数要求与单个创建商品一致
//   - 每批次最多创建 50 个商品
//   - 同批次时 item_key 字段值要唯一
func (s *GoodService) ProductBatchCreate(ctx context.Context, productList []any) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/batchCreate", productList)
}

// ProductPublishRequest 上架商品请求参数。
type ProductPublishRequest struct {
	ProductID          int64    `json:"product_id"`                     // 商品 ID
	UserName           []string `json:"user_name"`                      // 闲鱼会员名列表
	SpecifyPublishTime string   `json:"specify_publish_time,omitempty"` // 指定上架时间 yyyy-MM-dd HH:mm:ss
	NotifyURL          string   `json:"notify_url,omitempty"`           // 上架结果回调地址
}

// ProductPublish 上架商品。
//
// 对应 API: POST /api/open/product/publish
//
// 特别提醒：本接口采用异步方式更新商品信息到闲鱼 App，更新结果通过回调通知。
func (s *GoodService) ProductPublish(ctx context.Context, req *ProductPublishRequest) (*ApiResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("open: product publish request is nil")
	}
	return s.client.request(ctx, "/api/open/product/publish", req)
}

// ProductDownShelf 下架商品。
//
// 对应 API: POST /api/open/product/downShelf
func (s *GoodService) ProductDownShelf(ctx context.Context, productID int64) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/downShelf", map[string]any{
		"product_id": productID,
	})
}

// ProductEdit 编辑商品。
//
// 对应 API: POST /api/open/product/edit
//
// 参数：
//   - productData: 商品数据（必须包含 product_id，其他字段与创建商品类似）
func (s *GoodService) ProductEdit(ctx context.Context, productData any) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/edit", productData)
}

// --- 库存与删除 ---

// ProductEditStockRequest 编辑商品库存请求参数。
type ProductEditStockRequest struct {
	ProductID     int64            `json:"product_id"`               // 商品 ID
	Price         *int64           `json:"price,omitempty"`          // 商品价格（分）
	OriginalPrice *int64           `json:"original_price,omitempty"` // 商品原价（分）
	Stock         *int             `json:"stock,omitempty"`          // 单规格库存
	SkuItems      []map[string]any `json:"sku_items,omitempty"`      // 多规格库存
}

// ProductEditStock 编辑商品库存和价格。
//
// 对应 API: POST /api/open/product/edit/stock
func (s *GoodService) ProductEditStock(ctx context.Context, req *ProductEditStockRequest) (*ApiResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("open: product edit stock request is nil")
	}
	return s.client.request(ctx, "/api/open/product/edit/stock", req)
}

// ProductDelete 删除商品。
//
// 对应 API: POST /api/open/product/delete
func (s *GoodService) ProductDelete(ctx context.Context, productID int64) (*ApiResponse, error) {
	return s.client.request(ctx, "/api/open/product/delete", map[string]any{
		"product_id": productID,
	})
}
