// Package open 封装闲鱼开放平台 API（Goofish Open Platform SDK）。
//
// 该包对应 Python 版 XIE7654/goofish_api 库，使用 app_key + app_secret 鉴权，
// 与 pkg/apis（逆向 API，cookie 鉴权）互补：
//
//   - pkg/apis：基于浏览器 Cookie 的逆向 API，适用于个人 IM/爬虫场景
//   - pkg/open：基于开放平台 app_key 的官方 API，适用于商家商品/订单管理场景
//
// 完整功能模块：
//   - User：店铺授权管理（GetAuthorizeList）
//   - Good：商品 CRUD、类目管理、属性查询（12 个方法）
//   - Order：订单查询、发货处理、卡密管理（4 个方法）
//   - Other：快递公司查询
//
// 基本用法：
//
//	client := open.NewClient("your_app_key", "your_app_secret")
//	resp, err := client.User.GetAuthorizeList(ctx)
//	resp, err = client.Good.GetProductList(ctx, &open.GetProductListRequest{
//	    ProductStatus: open.ProductStatusStatus21,
//	    SaleStatus:    open.SaleStatusOnSale,
//	    PageNo:        1,
//	    PageSize:      50,
//	})
package open
