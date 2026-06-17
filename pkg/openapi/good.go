// Package openapi — 商品模块。
//
// 翻译自 Python SDK goofish_api/api/good.py
// 包含商品 CRUD、类目管理、属性查询、库存修改等 12 个方法。
package openapi

import "context"

// GoodService 商品模块，管理商品的全生命周期。
type GoodService struct {
	client *GoofishClient
}

// GetProductCategoryListRequest 查询商品类目请求参数。
type GetProductCategoryListRequest struct {
	ItemBizType    ItemBizType    `json:"item_biz_type"`    // 商品类型（必需）
	SpBizType      *SpBizType     `json:"sp_biz_type,omitempty"`      // 行业类型（可选）
	FlashSaleType  *FlashSaleType `json:"flash_sale_type,omitempty"`  // 闲鱼特卖类型（可选）
}

// GetProductCategoryList 查询商品类目。
//
// 对应 API: POST /api/open/product/category/list
//
// 参数:
//   - itemBizType: 商品类型（必需）
//   - spBizType: 行业类型（可选，传 nil 表示不指定）
//   - flashSaleType: 闲鱼特卖类型（可选）
//
// 示例:
//
//	resp, err := client.Good.GetProductCategoryList(ctx, openapi.ItemBizTypeCommon, nil, nil)
func (s *GoodService) GetProductCategoryList(ctx context.Context, itemBizType ItemBizType, spBizType *SpBizType, flashSaleType *FlashSaleType) (map[string]any, error) {
	data := GetProductCategoryListRequest{
		ItemBizType:   itemBizType,
		SpBizType:     spBizType,
		FlashSaleType: flashSaleType,
	}
	return s.client.request(ctx, "/api/open/product/category/list", data)
}

// GetProductPvListRequest 查询商品属性请求参数。
type GetProductPvListRequest struct {
	ItemBizType   ItemBizType `json:"item_biz_type"`   // 商品类型（必需）
	SpBizType     SpBizType   `json:"sp_biz_type"`     // 行业类型（必需）
	ChannelCatID  string      `json:"channel_cat_id"`  // 渠道类目ID（必需）
	SubPropertyID *string     `json:"sub_property_id,omitempty"` // 属性值ID（可选）
}

// GetProductPvList 查询商品属性。
//
// 对应 API: POST /api/open/product/pv/list
//
// 参数:
//   - itemBizType: 商品类型
//   - spBizType: 行业类型
//   - channelCatID: 渠道类目ID
//   - subPropertyID: 属性值ID（可选，传 nil 表示不指定）
func (s *GoodService) GetProductPvList(ctx context.Context, itemBizType ItemBizType, spBizType SpBizType, channelCatID string, subPropertyID *string) (map[string]any, error) {
	data := GetProductPvListRequest{
		ItemBizType:   itemBizType,
		SpBizType:     spBizType,
		ChannelCatID:  channelCatID,
		SubPropertyID: subPropertyID,
	}
	return s.client.request(ctx, "/api/open/product/pv/list", data)
}

// GetProductListRequest 查询商品列表请求参数。
type GetProductListRequest struct {
	OnlineTime    []int64         `json:"online_time,omitempty"`    // 上架时间范围 [开始时间戳, 结束时间戳]
	OfflineTime   []int64         `json:"offline_time,omitempty"`   // 下架时间范围
	SoldTime      []int64         `json:"sold_time,omitempty"`      // 售罄时间范围
	UpdateTime    []int64         `json:"update_time,omitempty"`    // 更新时间范围
	CreateTime    []int64         `json:"create_time,omitempty"`    // 创建时间范围
	ProductStatus *ProductStatus  `json:"product_status,omitempty"` // 商品状态
	SaleStatus    *SaleStatus     `json:"sale_status,omitempty"`    // 销售状态
	PageNo        int             `json:"page_no"`                  // 页码 (1-100)
	PageSize      int             `json:"page_size"`                // 每页数量 (1-100)
}

// GetProductList 查询商品列表。
//
// 对应 API: POST /api/open/product/list
//
// 参数:
//   - req: 查询条件，PageNo 和 PageSize 为必填（建议默认 1 和 50）
//
// 示例:
//
//	resp, err := client.Good.GetProductList(ctx, &openapi.GetProductListRequest{
//	    PageNo: 1, PageSize: 50,
//	    SaleStatus: openapi.SaleStatus(SaleStatusOnSale),
//	})
func (s *GoodService) GetProductList(ctx context.Context, req *GetProductListRequest) (map[string]any, error) {
	if req.PageNo == 0 {
		req.PageNo = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 50
	}
	return s.client.request(ctx, "/api/open/product/list", req)
}

// GetProductDetail 查询商品详情。
//
// 对应 API: POST /api/open/product/detail
//
// 参数:
//   - productID: 管家商品ID
func (s *GoodService) GetProductDetail(ctx context.Context, productID int64) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/detail", map[string]any{
		"product_id": productID,
	})
}

// GetProductSkuList 查询商品规格。
//
// 对应 API: POST /api/open/product/sku/list
//
// 参数:
//   - productIDs: 管家商品ID列表（最多100个）
func (s *GoodService) GetProductSkuList(ctx context.Context, productIDs []int64) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/sku/list", map[string]any{
		"product_id": productIDs,
	})
}

// CreateProduct 创建商品（单个）。
//
// 对应 API: POST /api/open/product/create
//
// 参数:
//   - productData: 商品数据（完整 JSON 结构，包含价格、库存、图片、类目等）
//
// productData 示例:
//
//	{
//	    "item_biz_type": 2,
//	    "sp_biz_type": 1,
//	    "channel_cat_id": "e11455b218c06e7ae10cfa39bf43dc0f",
//	    "price": 550000,
//	    "original_price": 700000,
//	    "stock": 10,
//	    "publish_shop": [{"images": ["https://..."], "title": "商品标题", "content": "描述"}]
//	}
func (s *GoodService) CreateProduct(ctx context.Context, productData map[string]any) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/create", productData)
}

// ProductBatchCreate 批量创建商品。
//
// 对应 API: POST /api/open/product/batchCreate
//
// 参数:
//   - productList: 商品数据列表（每批次最多50个，同批次 item_key 需唯一）
func (s *GoodService) ProductBatchCreate(ctx context.Context, productList []map[string]any) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/batchCreate", productList)
}

// ProductPublish 上架商品。
//
// 对应 API: POST /api/open/product/publish
//
// 注意: 本接口采用异步方式更新商品信息到闲鱼 App，更新结果通过回调通知。
//
// 参数:
//   - productID: 管家商品ID
//   - userNames: 闲鱼会员名列表
//   - specifyPublishTime: 指定上架时间（格式 yyyy-MM-dd HH:mm:ss，空字符串表示立即上架）
//   - notifyURL: 商品上架结果回调地址（空字符串表示不回调）
func (s *GoodService) ProductPublish(ctx context.Context, productID int64, userNames []string, specifyPublishTime, notifyURL string) (map[string]any, error) {
	data := map[string]any{
		"product_id": productID,
		"user_name":  userNames,
	}
	if specifyPublishTime != "" {
		data["specify_publish_time"] = specifyPublishTime
	}
	if notifyURL != "" {
		data["notify_url"] = notifyURL
	}
	return s.client.request(ctx, "/api/open/product/publish", data)
}

// ProductDownShelf 下架商品。
//
// 对应 API: POST /api/open/product/downShelf
//
// 参数:
//   - productID: 管家商品ID
func (s *GoodService) ProductDownShelf(ctx context.Context, productID int64) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/downShelf", map[string]any{
		"product_id": productID,
	})
}

// ProductEdit 编辑商品信息。
//
// 对应 API: POST /api/open/product/edit
//
// 参数:
//   - productData: 商品编辑数据（必须包含 product_id）
func (s *GoodService) ProductEdit(ctx context.Context, productData map[string]any) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/edit", productData)
}

// ProductEditStock 修改商品库存和价格。
//
// 对应 API: POST /api/open/product/editStock
//
// 参数:
//   - productID: 管家商品ID
//   - price: 价格（单位: 分，如 550000 表示 5500.00 元）
//   - stock: 库存数量
func (s *GoodService) ProductEditStock(ctx context.Context, productID int64, price int64, stock int) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/editStock", map[string]any{
		"product_id": productID,
		"price":      price,
		"stock":      stock,
	})
}

// ProductDelete 删除商品。
//
// 对应 API: POST /api/open/product/delete
//
// 参数:
//   - productID: 管家商品ID
func (s *GoodService) ProductDelete(ctx context.Context, productID int64) (map[string]any, error) {
	return s.client.request(ctx, "/api/open/product/delete", map[string]any{
		"product_id": productID,
	})
}
