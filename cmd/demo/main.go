// XianYuApis-GO 综合功能测试 Demo（单文件整合版）
//
// 覆盖项目全部功能：
//  1. pkg/apis   — 逆向 API（Cookie 鉴权）：Token、商品、上传、发货等 9 个方法
//  2. pkg/open   — 开放平台 SDK（app_key 鉴权）：User/Good/Order/Other 共 18 个方法
//  3. pkg/search — 搜索爬虫（Cookie 鉴权）：Search、SearchAll
//  4. pkg/ws     — WebSocket 实时通信：扫码/手动登录 + 消息收发
//
// 运行方式: go run ./cmd/demo/
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/model"
	"github.com/cv-cat/xianyuapis/pkg/msg"
	"github.com/cv-cat/xianyuapis/pkg/open"
	"github.com/cv-cat/xianyuapis/pkg/search"
	"github.com/cv-cat/xianyuapis/pkg/util"
	"github.com/cv-cat/xianyuapis/pkg/ws"
	"go.uber.org/zap"
)

// 分隔线
const sep = "=============================================="

func main() {
	printHeader()
	choice := readMenuChoice()
	switch choice {
	case "1":
		runApisDemo()
	case "2":
		runOpenDemo()
	case "3":
		runSearchDemo()
	case "4":
		runWSDemo()
	case "5":
		runAllAPIs()
	default:
		fmt.Println("无效选择")
	}

	// 测试完毕后暂停，等待用户关闭
	fmt.Println()
	fmt.Println(sep)
	fmt.Println("  测试结束，按回车键退出...")
	fmt.Println(sep)
	readLine()
}

// printHeader 打印主菜单和前置条件说明。
func printHeader() {
	fmt.Println(sep)
	fmt.Println("  闲鱼 API 综合功能测试 Demo")
	fmt.Println(sep)
	fmt.Println()
	fmt.Println("【测试模块】")
	fmt.Println("  1. 逆向 API（pkg/apis，Cookie 鉴权）— 9 个方法")
	fmt.Println("     前置: 闲鱼账号已登录，可获取浏览器 Cookie")
	fmt.Println()
	fmt.Println("  2. 开放平台 API（pkg/open，app_key 鉴权）— 18 个方法")
	fmt.Println("     前置: 已在闲管家 https://goofish.pro 申请 app_key/app_secret")
	fmt.Println()
	fmt.Println("  3. 搜索爬虫（pkg/search，Cookie 鉴权）— 2 个方法")
	fmt.Println("     前置: 闲鱼账号已登录，可获取浏览器 Cookie")
	fmt.Println()
	fmt.Println("  4. WebSocket 实时通信（pkg/ws，扫码/手动登录）")
	fmt.Println("     前置: 闲鱼账号可扫码登录，或手动获取 Cookie+Token")
	fmt.Println()
	fmt.Println("  5. 运行全部 API 测试（1→2→3，不含 WebSocket）")
	fmt.Println()
	fmt.Println("【快速开始】")
	fmt.Println("  - 首次测试建议选 1 或 3（仅需 Cookie）")
	fmt.Println("  - 商家对接建议选 2（需开放平台账号）")
	fmt.Println("  - 实时消息建议选 4（需扫码或手动 Token）")
	fmt.Println()
	fmt.Print("请输入选择 (1/2/3/4/5): ")
}

// readMenuChoice 读取用户菜单选择。
func readMenuChoice() string {
	var choice string
	fmt.Scanln(&choice)
	return strings.TrimSpace(choice)
}

// runAllAPIs 按顺序执行 API 模块测试（不含 WebSocket）。
func runAllAPIs() {
	fmt.Println()
	fmt.Println(sep)
	fmt.Println("【全部 API 测试】将顺序执行 3 个模块，共 29 个方法")
	fmt.Println(sep)
	fmt.Println()
	fmt.Println("[说明] 每个模块独立鉴权，可中途 Ctrl+C 退出")
	fmt.Println("[说明] 失败的用例不会阻断后续测试")
	fmt.Println()

	fmt.Println(sep)
	fmt.Println("【模块 1/3】逆向 API（pkg/apis）")
	fmt.Println(sep)
	runApisDemo()

	fmt.Println("\n", sep)
	fmt.Println("【模块 2/3】开放平台 API（pkg/open）")
	fmt.Println(sep)
	runOpenDemo()

	fmt.Println("\n", sep)
	fmt.Println("【模块 3/3】搜索爬虫（pkg/search）")
	fmt.Println(sep)
	runSearchDemo()

	fmt.Println("\n", sep)
	fmt.Println("全部 API 测试完成")
	fmt.Println(sep)
}

// ============================================================
// 模块 1: pkg/apis 逆向 API 测试
// ============================================================

// runApisDemo 测试 pkg/apis 所有方法。
//
// 需要 Cookie 鉴权，覆盖：
//   - GetToken / RefreshToken / RefreshMtopToken（Token 管理）
//   - GetItemInfo（商品详情）
//   - GetPublicChannel（推荐标签）
//   - GetDefaultLocation（默认位置）
//   - UploadMedia（图片上传）
//   - PublishItem（商品发布）
//   - ConfirmShipping（自动发货）
func runApisDemo() {
	fmt.Println("\n--- 逆向 API（pkg/apis）测试 ---")
	fmt.Println()
	fmt.Println("[步骤 1] 准备闲鱼登录 Cookie")
	fmt.Println("  1.1 打开浏览器，访问 https://www.goofish.com 并登录闲鱼账号")
	fmt.Println("  1.2 按 F12 打开开发者工具 → 切换到 Network 标签页")
	fmt.Println("  1.3 刷新页面，在请求列表中点击任意一个请求")
	fmt.Println("  1.4 在 Request Headers 中找到 Cookie 字段，复制完整值")
	fmt.Println("[提示] Cookie 必须包含 unb 字段（用户ID），否则视为未登录")
	fmt.Println("[提示] Cookie 有效期约 24 小时，过期后需重新获取")
	fmt.Println()

	api, err := buildAPIFromCookie()
	if err != nil {
		fmt.Printf("\n[失败] 创建 API 实例失败: %v\n", err)
		printCookieTroubleshooting(err)
		return
	}

	fmt.Println()
	fmt.Println("[步骤 2] 开始执行 9 个 API 测试用例")
	fmt.Println("[说明] 每个用例会显示测试名称、执行结果和响应预览")
	fmt.Println("[说明] 需要输入参数的用例会提示，回车可使用默认值")
	fmt.Println()

	ctx := context.Background()
	passed, failed := 0, 0

	// 辅助函数：执行单个测试用例
	runCase := func(name string, fn func() error) {
		fmt.Printf("\n[测试] %s\n", name)
		if err := fn(); err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
			failed++
		} else {
			fmt.Printf("  ✓ 成功\n")
			passed++
		}
	}

	// 1. GetToken — 获取 WebSocket accessToken
	fmt.Println("[说明] 1/9: 调用 mtop.taobao.idlemessage.pc.login.token 获取 WebSocket 通信 Token")
	fmt.Println("[预期] 返回一串 Base64 编码的 Token 字符串，长度约 100-200 字符")
	var accessToken string
	runCase("GetToken 获取 WebSocket Token", func() error {
		token, err := api.GetToken(ctx)
		if err != nil {
			return err
		}
		accessToken = token
		fmt.Printf("  Token 长度: %d, 预览: %s...\n", len(token), previewStr(token, 20))
		return nil
	})

	// 2. RefreshMtopToken — 刷新 _m_h5_tk Cookie
	fmt.Println("[说明] 2/9: 刷新 _m_h5_tk Cookie，避免 mtop 接口签名失效")
	fmt.Println("[预期] 无返回值，刷新后后续 mtop 请求签名有效")
	runCase("RefreshMtopToken 刷新 mtop Token", func() error {
		api.RefreshMtopToken(ctx)
		fmt.Printf("  mtop Token 已刷新（无返回值）\n")
		return nil
	})

	// 3. GetItemInfo — 获取商品详情
	fmt.Println("[说明] 3/9: 调用 mtop.taobao.idle.publish.item.detail 查询商品详情")
	fmt.Println("[预期] 返回商品标题、价格、图片、卖家信息等")
	fmt.Println()
	fmt.Print("请输入要测试的商品 ID（回车使用默认示例 771765432109）: ")
	itemID := readLine()
	if itemID == "" {
		itemID = "771765432109"
		fmt.Printf("使用默认商品 ID: %s\n", itemID)
	}
	fmt.Println("[提示] 商品 ID 可从闲鱼商品详情页 URL 中获取（如 /item/771765432109）")
	runCase("GetItemInfo 获取商品详情", func() error {
		result, err := api.GetItemInfo(ctx, itemID)
		if err != nil {
			return err
		}
		printResultPreview(result)
		return nil
	})

	// 4. GetDefaultLocation — 获取默认地理位置
	fmt.Println("[说明] 4/9: 调用 mtop.taobao.idle.user.location 获取默认发布位置")
	fmt.Println("[预期] 返回省份、城市、区县等地理位置信息")
	runCase("GetDefaultLocation 获取默认位置", func() error {
		result, err := api.GetDefaultLocation(ctx)
		if err != nil {
			return err
		}
		printResultPreview(result)
		return nil
	})

	// 5. GetPublicChannel — 获取推荐标签和分类
	fmt.Println("[说明] 5/9: 调用 mtop.taobao.idle.publish.channel 获取推荐标签和分类")
	fmt.Println("[预期] 返回可用分类列表、推荐标签等")
	runCase("GetPublicChannel 获取推荐标签", func() error {
		result, err := api.GetPublicChannel(ctx, "测试商品标题", nil)
		if err != nil {
			return err
		}
		printResultPreview(result)
		return nil
	})

	// 6. UploadMedia — 上传图片
	fmt.Println("[说明] 6/9: 调用 mtop.taobao.idle.publish.image.upload 上传商品图片")
	fmt.Println("[预期] 返回图片 URL、宽度、高度")
	fmt.Println()
	fmt.Print("请输入要上传的图片路径（回车跳过）: ")
	imgPath := readLine()
	if imgPath != "" {
		fmt.Println("[提示] 支持 jpg/png 格式，建议小于 5MB")
		runCase("UploadMedia 上传图片", func() error {
			result, err := api.UploadMedia(ctx, imgPath)
			if err != nil {
				return err
			}
			fmt.Printf("  URL: %s, 尺寸: %dx%d\n", result.URL, result.Width, result.Height)
			return nil
		})
	} else {
		fmt.Println("[跳过] UploadMedia（未提供图片路径）")
	}

	// 7. PublishItem — 发布商品（需要图片，可选）
	fmt.Println("[说明] 7/9: 调用 mtop.taobao.idle.publish.item.edit 发布商品到闲鱼")
	fmt.Println("[预期] 返回新商品的 item_id")
	fmt.Println("[警告] 此操作会真实发布商品到闲鱼，请谨慎测试")
	fmt.Println()
	fmt.Print("是否测试 PublishItem 发布商品？(y/N): ")
	if strings.EqualFold(readLine(), "y") {
		fmt.Print("请输入图片路径（多个用逗号分隔）: ")
		imgPaths := strings.Split(readLine(), ",")
		fmt.Print("请输入商品描述: ")
		desc := readLine()
		if desc == "" {
			desc = "测试商品"
		}
		runCase("PublishItem 发布商品", func() error {
			result, err := api.PublishItem(ctx, imgPaths, desc, nil, modelDeliverySettings())
			if err != nil {
				return err
			}
			printResultPreview(result)
			return nil
		})
	} else {
		fmt.Println("[跳过] PublishItem（用户取消）")
	}

	// 8. ConfirmShipping — 自动确认发货
	fmt.Println("[说明] 8/9: 调用 mtop.taobao.idle.trade.ship.confirm 确认订单发货")
	fmt.Println("[预期] 返回成功状态")
	fmt.Println()
	fmt.Print("请输入要确认发货的订单 ID（回车跳过）: ")
	orderID := readLine()
	if orderID != "" {
		runCase("ConfirmShipping 自动确认发货", func() error {
			result, err := api.ConfirmShipping(ctx, orderID)
			if err != nil {
				return err
			}
			printResultPreview(result)
			return nil
		})
	} else {
		fmt.Println("[跳过] ConfirmShipping（未提供订单 ID）")
	}

	// 9. RefreshToken — 刷新登录态
	fmt.Println("[说明] 9/9: 调用 mtop.taobao.idle.user.token.refresh 刷新登录态")
	fmt.Println("[预期] Cookie 中的登录态字段被更新")
	runCase("RefreshToken 刷新登录态", func() error {
		if err := api.RefreshToken(ctx); err != nil {
			return err
		}
		fmt.Printf("  登录态已刷新\n")
		return nil
	})

	_ = accessToken // 保留 token 供后续 WebSocket 使用
	printTestSummary("pkg/apis", passed, failed)
}

// ============================================================
// 模块 2: pkg/open 开放平台 API 测试
// ============================================================

// runOpenDemo 测试 pkg/open 所有 18 个方法。
//
// 需要 app_key + app_secret 鉴权，覆盖：
//   - User: GetAuthorizeList
//   - Good: 12 个方法（类目/属性/列表/详情/SKU/创建/批量创建/上架/下架/编辑/库存/删除）
//   - Order: 4 个方法（列表/详情/卡密/发货）
//   - Other: GetExpressCompanies
func runOpenDemo() {
	fmt.Println("\n--- 开放平台 API（pkg/open）测试 ---")
	fmt.Println()
	fmt.Println("[步骤 1] 准备开放平台凭证")
	fmt.Println("  1.1 访问闲管家 https://goofish.pro/register 注册账号")
	fmt.Println("  1.2 登录后绑定闲鱼账号，进入「开放平台」开通管家应用")
	fmt.Println("  1.3 在应用详情页获取 app_key 和 app_secret")
	fmt.Println("  1.4 （可选）商务对接模式需提供 seller_id")
	fmt.Println("[提示] 闲管家是闲鱼官方商家工具平台，app_key 需订购 ERP 版本")
	fmt.Println("[提示] 申请详情参考操作手册: https://m.goofish.pro/app/operation.html")
	fmt.Println("[提示] API 调用域名: https://open.goofish.pro（仅用于接口调用，非网站）")
	fmt.Println()

	var appKey, appSecret, sellerID string
	fmt.Print("请输入 app_key: ")
	appKey = readLine()
	fmt.Print("请输入 app_secret: ")
	appSecret = readLine()
	fmt.Print("请输入 seller_id（商务对接模式，回车跳过）: ")
	sellerID = readLine()

	if appKey == "" || appSecret == "" {
		fmt.Println("\n[失败] app_key 和 app_secret 不能为空")
		fmt.Println("[排查] 请访问闲管家 https://goofish.pro/register 注册并申请")
		return
	}

	// 创建客户端
	opts := []open.Option{open.WithDebug(false)}
	if sellerID != "" {
		opts = append(opts, open.WithSellerID(sellerID))
		fmt.Printf("[提示] 已启用商务对接模式，seller_id=%s\n", sellerID)
	}
	client := open.NewClient(appKey, appSecret, opts...)

	fmt.Println()
	fmt.Println("[步骤 2] 开始执行 18 个 API 测试用例")
	fmt.Println("[说明] 测试顺序: User(1) → Good(12) → Order(4) → Other(1)")
	fmt.Println("[说明] 涉及商品操作的用例需要 product_id，建议先创建商品再测试后续操作")
	fmt.Println("[说明] 危险操作（创建/删除）会二次确认")
	fmt.Println()

	ctx := context.Background()
	passed, failed := 0, 0

	runCase := func(name string, fn func() error) {
		fmt.Printf("\n[测试] %s\n", name)
		if err := fn(); err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
			failed++
		} else {
			fmt.Printf("  ✓ 成功\n")
			passed++
		}
	}

	// ===== User 模块（1 个方法）=====
	fmt.Println("--- User 模块（1/4 模块，共 1 个方法）---")
	fmt.Println("[说明] 1/18: 查询当前 app_key 授权的店铺列表")
	fmt.Println("[预期] 返回授权店铺列表，包含店铺ID、名称、状态等")
	runCase("User.GetAuthorizeList 查询授权店铺", func() error {
		resp, err := client.User.GetAuthorizeList(ctx)
		if err != nil {
			return err
		}
		printOpenResp(resp)
		return nil
	})

	// ===== Good 模块（12 个方法）=====
	fmt.Println("\n--- Good 模块（2/4 模块，共 12 个方法）---")
	fmt.Println("[说明] 商品管理模块，覆盖类目、属性、列表、详情、SKU、CRUD 等操作")
	fmt.Println()

	// 1. GetProductCategoryList
	spBizType := open.SpBizTypeMobile
	fmt.Println("[说明] 2/18: 查询商品类目列表（手机数码类）")
	fmt.Println("[预期] 返回类目树结构，包含类目ID、名称、层级等")
	runCase("Good.GetProductCategoryList 查询商品类目", func() error {
		resp, err := client.Good.GetProductCategoryList(ctx,
			open.ItemBizTypeCommon, &spBizType, nil)
		if err != nil {
			return err
		}
		printOpenResp(resp)
		return nil
	})

	// 2. GetProductPvList
	fmt.Println("[说明] 3/18: 查询商品属性（基于类目 channel_cat_id）")
	fmt.Println("[预期] 返回属性列表，如品牌、型号、成色等")
	fmt.Println()
	fmt.Print("请输入 channel_cat_id（回车使用默认 e11455b218c06e7ae10cfa39bf43dc0f）: ")
	channelCatID := readLine()
	if channelCatID == "" {
		channelCatID = "e11455b218c06e7ae10cfa39bf43dc0f"
		fmt.Printf("使用默认 channel_cat_id: %s\n", channelCatID)
	}
	fmt.Println("[提示] channel_cat_id 可从 GetProductCategoryList 响应中获取")
	runCase("Good.GetProductPvList 查询商品属性", func() error {
		resp, err := client.Good.GetProductPvList(ctx,
			open.ItemBizTypeCommon, open.SpBizTypeMobile, channelCatID, nil)
		if err != nil {
			return err
		}
		printOpenResp(resp)
		return nil
	})

	// 3. GetProductList
	fmt.Println("[说明] 4/18: 查询商品列表（在售状态）")
	fmt.Println("[预期] 返回商品列表，包含 product_id、标题、价格、库存等")
	runCase("Good.GetProductList 查询商品列表", func() error {
		saleStatus := open.SaleStatusOnSale
		resp, err := client.Good.GetProductList(ctx, &open.GetProductListRequest{
			SaleStatus: &saleStatus,
			PageNo:     1,
			PageSize:   10,
		})
		if err != nil {
			return err
		}
		printOpenResp(resp)
		return nil
	})

	// 4. GetProductDetail
	fmt.Println("[说明] 5/18: 查询商品详情（需要 product_id）")
	fmt.Println("[预期] 返回商品完整信息，包含标题、价格、图片、SKU 等")
	fmt.Println()
	fmt.Print("请输入要查询的 product_id（回车跳过）: ")
	productIDStr := readLine()
	var productID int64
	if productIDStr != "" {
		fmt.Sscanf(productIDStr, "%d", &productID)
		fmt.Println("[提示] product_id 可从 GetProductList 响应中获取")
		runCase("Good.GetProductDetail 查询商品详情", func() error {
			resp, err := client.Good.GetProductDetail(ctx, productID)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.GetProductDetail（未提供 product_id）")
		fmt.Println("[提示] 后续 SKU/上架/下架/编辑/库存/删除操作均依赖 product_id")
	}

	// 5. GetProductSkuList
	fmt.Println("[说明] 6/18: 查询商品规格 SKU 列表（需要 product_id）")
	fmt.Println("[预期] 返回 SKU 列表，包含规格ID、价格、库存等")
	if productID != 0 {
		runCase("Good.GetProductSkuList 查询商品规格", func() error {
			resp, err := client.Good.GetProductSkuList(ctx, []int64{productID})
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.GetProductSkuList（未提供 product_id）")
	}

	// 6. CreateProduct
	fmt.Println("[说明] 7/18: 创建商品（危险操作，会真实创建商品）")
	fmt.Println("[预期] 返回新商品的 product_id")
	fmt.Println("[警告] 此操作会真实创建商品，请谨慎测试")
	fmt.Println()
	fmt.Print("是否测试 Good.CreateProduct 创建商品？(y/N): ")
	if strings.EqualFold(readLine(), "y") {
		runCase("Good.CreateProduct 创建商品", func() error {
			productData := map[string]any{
				"item_biz_type":  open.ItemBizTypeCommon,
				"sp_biz_type":    open.SpBizTypeMobile,
				"channel_cat_id": channelCatID,
				"price":          550000,
				"original_price": 700000,
				"stock":          10,
				"publish_shop": []map[string]any{
					{
						"images":  []string{"https://img.alicdn.com/bao/uploaded/O1CN01test.jpg"},
						"title":   "测试商品标题",
						"content": "测试商品描述",
					},
				},
			}
			resp, err := client.Good.CreateProduct(ctx, productData)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.CreateProduct（用户取消）")
	}

	// 7. ProductBatchCreate
	fmt.Println("[说明] 8/18: 批量创建商品（危险操作，会真实创建商品）")
	fmt.Println("[预期] 返回批量创建结果，包含成功/失败记录")
	fmt.Println()
	fmt.Print("是否测试 Good.ProductBatchCreate 批量创建？(y/N): ")
	if strings.EqualFold(readLine(), "y") {
		runCase("Good.ProductBatchCreate 批量创建商品", func() error {
			productList := []any{
				map[string]any{
					"item_key":      fmt.Sprintf("test_%d", time.Now().UnixMilli()),
					"item_biz_type": open.ItemBizTypeCommon,
					"sp_biz_type":   open.SpBizTypeMobile,
					"price":         100000,
					"stock":         5,
				},
			}
			resp, err := client.Good.ProductBatchCreate(ctx, productList)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.ProductBatchCreate（用户取消）")
	}

	// 8. ProductPublish
	fmt.Println("[说明] 9/18: 上架商品到指定会员店铺（需要 product_id 和会员名）")
	fmt.Println("[预期] 返回上架结果")
	if productID != 0 {
		fmt.Println()
		fmt.Print("请输入闲鱼会员名（逗号分隔，回车跳过）: ")
		userNamesStr := readLine()
		if userNamesStr != "" {
			userNames := strings.Split(userNamesStr, ",")
			fmt.Println("[提示] 会员名可从 GetAuthorizeList 响应中获取")
			runCase("Good.ProductPublish 上架商品", func() error {
				resp, err := client.Good.ProductPublish(ctx, &open.ProductPublishRequest{
					ProductID: productID,
					UserName:  userNames,
				})
				if err != nil {
					return err
				}
				printOpenResp(resp)
				return nil
			})
		} else {
			fmt.Println("[跳过] Good.ProductPublish（未提供会员名）")
		}
	} else {
		fmt.Println("[跳过] Good.ProductPublish（未提供 product_id）")
	}

	// 9. ProductDownShelf
	fmt.Println("[说明] 10/18: 下架商品（需要 product_id）")
	fmt.Println("[预期] 返回下架结果")
	if productID != 0 {
		runCase("Good.ProductDownShelf 下架商品", func() error {
			resp, err := client.Good.ProductDownShelf(ctx, productID)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.ProductDownShelf（未提供 product_id）")
	}

	// 10. ProductEdit
	fmt.Println("[说明] 11/18: 编辑商品信息（需要 product_id）")
	fmt.Println("[预期] 返回编辑结果")
	if productID != 0 {
		runCase("Good.ProductEdit 编辑商品", func() error {
			editData := map[string]any{
				"product_id": productID,
				"stock":      20,
			}
			resp, err := client.Good.ProductEdit(ctx, editData)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.ProductEdit（未提供 product_id）")
	}

	// 11. ProductEditStock
	fmt.Println("[说明] 12/18: 修改商品库存和价格（需要 product_id）")
	fmt.Println("[预期] 返回修改结果")
	if productID != 0 {
		newPrice := int64(600000)
		newStock := 15
		fmt.Println("[提示] 测试将价格改为 600000 分（6000 元），库存改为 15")
		runCase("Good.ProductEditStock 修改库存价格", func() error {
			resp, err := client.Good.ProductEditStock(ctx, &open.ProductEditStockRequest{
				ProductID: productID,
				Price:     &newPrice,
				Stock:     &newStock,
			})
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Good.ProductEditStock（未提供 product_id）")
	}

	// 12. ProductDelete
	fmt.Println("[说明] 13/18: 删除商品（危险操作，不可恢复）")
	fmt.Println("[预期] 返回删除结果")
	fmt.Println("[警告] 此操作不可恢复，请谨慎测试")
	if productID != 0 {
		fmt.Println()
		fmt.Print("是否测试 Good.ProductDelete 删除商品？(y/N): ")
		if strings.EqualFold(readLine(), "y") {
			runCase("Good.ProductDelete 删除商品", func() error {
				resp, err := client.Good.ProductDelete(ctx, productID)
				if err != nil {
					return err
				}
				printOpenResp(resp)
				return nil
			})
		} else {
			fmt.Println("[跳过] Good.ProductDelete（用户取消）")
		}
	} else {
		fmt.Println("[跳过] Good.ProductDelete（未提供 product_id）")
	}

	// ===== Order 模块（4 个方法）=====
	fmt.Println("\n--- Order 模块（3/4 模块，共 4 个方法）---")
	fmt.Println("[说明] 订单管理模块，覆盖列表、详情、卡密、发货操作")
	fmt.Println()

	// 13. GetOrderList
	fmt.Println("[说明] 14/18: 查询订单列表（待发货状态）")
	fmt.Println("[预期] 返回订单列表，包含订单号、买家、金额、商品信息等")
	runCase("Order.GetOrderList 查询订单列表", func() error {
		orderStatus := open.OrderStatusPendingShipment
		resp, err := client.Order.GetOrderList(ctx, &open.GetOrderListRequest{
			OrderStatus: &orderStatus,
			PageNo:      1,
			PageSize:    10,
		})
		if err != nil {
			return err
		}
		printOpenResp(resp)
		return nil
	})

	// 14. GetOrderDetail
	fmt.Println("[说明] 15/18: 查询订单详情（需要订单号）")
	fmt.Println("[预期] 返回订单完整信息，包含收货地址、物流、商品等")
	fmt.Println()
	fmt.Print("请输入订单号（回车跳过）: ")
	orderNo := readLine()
	if orderNo != "" {
		fmt.Println("[提示] 订单号可从 GetOrderList 响应中获取")
		runCase("Order.GetOrderDetail 查询订单详情", func() error {
			resp, err := client.Order.GetOrderDetail(ctx, orderNo)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})

		// 15. KamOrderList
		fmt.Println("[说明] 16/18: 查询订单卡密列表（虚拟商品）")
		fmt.Println("[预期] 返回卡密列表，包含卡号、密码、状态等")
		runCase("Order.KamOrderList 查询订单卡密", func() error {
			resp, err := client.Order.KamOrderList(ctx, orderNo)
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Order.GetOrderDetail（未提供订单号）")
		fmt.Println("[跳过] Order.KamOrderList（未提供订单号）")
	}

	// 16. OrderShip
	fmt.Println("[说明] 17/18: 物流发货（危险操作，会真实发货）")
	fmt.Println("[预期] 返回发货结果")
	fmt.Println("[警告] 此操作会真实发货，请谨慎测试")
	fmt.Println()
	fmt.Print("是否测试 Order.OrderShip 物流发货？(y/N): ")
	if strings.EqualFold(readLine(), "y") {
		fmt.Println("[提示] 请准备以下信息：订单号、运单号、快递公司、收货地址")
		shipOrderNo := readInput("  订单号: ")
		waybillNo := readInput("  运单号: ")
		expressName := readInput("  快递公司名称（如: 顺丰）: ")
		expressCode := readInput("  快递公司编码（如: SF）: ")
		shipProv := readInput("  省份（如: 广东省）: ")
		shipCity := readInput("  城市（如: 深圳市）: ")
		shipArea := readInput("  区县（如: 南山区）: ")
		fmt.Println("[提示] 快递公司编码可从 Other.GetExpressCompanies 响应中获取")
		runCase("Order.OrderShip 物流发货", func() error {
			resp, err := client.Order.OrderShip(ctx, &open.OrderShipRequest{
				OrderNo:      shipOrderNo,
				WaybillNo:    waybillNo,
				ExpressName:  expressName,
				ExpressCode:  expressCode,
				ShipProvName: shipProv,
				ShipCityName: shipCity,
				ShipAreaName: shipArea,
			})
			if err != nil {
				return err
			}
			printOpenResp(resp)
			return nil
		})
	} else {
		fmt.Println("[跳过] Order.OrderShip（用户取消）")
	}

	// ===== Other 模块（1 个方法）=====
	fmt.Println("\n--- Other 模块（4/4 模块，共 1 个方法）---")
	fmt.Println("[说明] 18/18: 查询快递公司列表")
	fmt.Println("[预期] 返回快递公司列表，包含名称、编码、logo 等")
	runCase("Other.GetExpressCompanies 查询快递公司", func() error {
		resp, err := client.Other.GetExpressCompanies(ctx)
		if err != nil {
			return err
		}
		printOpenResp(resp)
		return nil
	})

	printTestSummary("pkg/open", passed, failed)
}

// ============================================================
// 模块 3: pkg/search 搜索爬虫测试
// ============================================================

// runSearchDemo 测试 pkg/search 所有方法。
//
// 需要 Cookie 鉴权，覆盖：
//   - Search（单页搜索）
//   - SearchAll（多页聚合）
func runSearchDemo() {
	fmt.Println("\n--- 搜索爬虫（pkg/search）测试 ---")
	fmt.Println()
	fmt.Println("[步骤 1] 准备闲鱼登录 Cookie")
	fmt.Println("  1.1 打开浏览器，访问 https://www.goofish.com 并登录闲鱼账号")
	fmt.Println("  1.2 按 F12 打开开发者工具 → Network 标签页")
	fmt.Println("  1.3 刷新页面，点击任意请求，复制 Request Headers 中的 Cookie 值")
	fmt.Println("[提示] Cookie 必须包含 unb 字段（用户ID），否则视为未登录")
	fmt.Println()

	api, err := buildAPIFromCookie()
	if err != nil {
		fmt.Printf("\n[失败] 创建 API 实例失败: %v\n", err)
		printCookieTroubleshooting(err)
		return
	}

	fmt.Println()
	fmt.Println("[步骤 2] 设置搜索参数")
	fmt.Println("[说明] 搜索接口为 mtop.taobao.idlemtopsearch.pc.search")
	fmt.Println("[说明] 单页返回最多 60 个商品，多页可聚合更多结果")
	fmt.Println()
	fmt.Print("请输入搜索关键词（回车使用默认 '机械键盘'）: ")
	keyword := readLine()
	if keyword == "" {
		keyword = "机械键盘"
		fmt.Printf("使用默认关键词: %s\n", keyword)
	}
	fmt.Println("[提示] 关键词越具体，搜索结果越精准")

	crawler := search.New(api)
	ctx := context.Background()
	passed, failed := 0, 0

	runCase := func(name string, fn func() error) {
		fmt.Printf("\n[测试] %s\n", name)
		if err := fn(); err != nil {
			fmt.Printf("  ✗ 失败: %v\n", err)
			failed++
		} else {
			fmt.Printf("  ✓ 成功\n")
			passed++
		}
	}

	// 1. Search — 单页搜索
	fmt.Println("[说明] 1/2: 单页搜索（第 1 页，每页 30 条）")
	fmt.Println("[预期] 返回商品列表，包含标题、价格、卖家、地区、详情链接等")
	runCase("Search 单页搜索", func() error {
		items, err := crawler.Search(ctx, &search.Request{
			Keyword:     keyword,
			PageNumber:  1,
			RowsPerPage: 30,
		})
		if err != nil {
			return err
		}
		printSearchItems(items, 5) // 只展示前 5 个
		return nil
	})

	// 2. SearchAll — 多页聚合
	fmt.Println("[说明] 2/2: 多页聚合搜索（自动翻页，聚合多页结果）")
	fmt.Println("[预期] 返回聚合后的商品列表，数量为 单页数量 × 页数")
	fmt.Println()
	fmt.Print("请输入最大页数（回车使用默认 2）: ")
	maxPagesStr := readLine()
	maxPages := 2
	if maxPagesStr != "" {
		fmt.Sscanf(maxPagesStr, "%d", &maxPages)
	}
	if maxPages <= 0 {
		maxPages = 2
	}
	fmt.Printf("[提示] 将聚合 %d 页，预计返回约 %d 个商品\n", maxPages, maxPages*30)
	fmt.Println("[提示] 页数过多可能触发风控，建议不超过 5 页")
	runCase("SearchAll 多页聚合搜索", func() error {
		items, err := crawler.SearchAll(ctx, &search.Request{
			Keyword:     keyword,
			RowsPerPage: 30,
		}, maxPages)
		if err != nil {
			return err
		}
		fmt.Printf("  共聚合 %d 页，得到 %d 个商品\n", maxPages, len(items))
		printSearchItems(items, 5)
		return nil
	})

	printTestSummary("pkg/search", passed, failed)
}

// ============================================================
// 模块 4: pkg/ws WebSocket 实时通信测试
// ============================================================

// runWSDemo 测试 pkg/ws WebSocket 实时通信。
//
// 覆盖：
//   - 扫码登录 / 手动 Cookie+Token 登录
//   - WebSocket 连接、消息收发、心跳保活
func runWSDemo() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("\n--- WebSocket 实时通信（pkg/ws）测试 ---")
	fmt.Println()
	fmt.Println("[步骤 1] 选择登录方式")
	fmt.Println("  1. 扫码登录（自动）— 适合首次使用，会显示二维码")
	fmt.Println("  2. 手动输入 Cookie + Token（推荐）— 避免风控，适合长期使用")
	fmt.Println()
	fmt.Println("[说明] 扫码登录会调用 mtop.taobao.idle.qrcode.login 接口轮询登录状态")
	fmt.Println("[说明] 手动登录需要先在浏览器控制台执行脚本获取 Token")
	fmt.Println()
	fmt.Print("请输入选择 (1/2，回车默认 2): ")

	var choice string
	fmt.Scanln(&choice)
	if choice == "" {
		choice = "2"
	}

	var api *apis.XianyuAPI
	var accessToken string
	var err error

	switch choice {
	case "2":
		api, accessToken, err = manualCookieAndTokenLogin()
	default:
		fmt.Println("\n[步骤 1] 启动扫码登录流程...")
		fmt.Println("[说明] 程序将显示二维码，请使用闲鱼 APP 扫码确认")
		fmt.Println("[提示] 二维码有效期 2 分钟，超时需重新生成")
		api, err = qrcodeLogin()
	}

	if err != nil {
		logger.Fatal("登录失败", zap.Error(err))
	}

	logger.Info("登录成功",
		zap.String("deviceID", api.DeviceID()),
		zap.String("cookiePreview", previewCookie(api)),
	)

	fmt.Println()
	fmt.Println("[步骤 2] 创建 WebSocket 客户端")
	// 创建 WebSocket 客户端
	wsClient, err := ws.NewWithAPI(api)
	if err != nil {
		logger.Fatal("创建 WebSocket 客户端失败", zap.Error(err))
	}

	// 设置消息处理回调
	fmt.Println("[步骤 3] 设置消息处理回调")
	fmt.Println("[说明] 收到文字消息会自动回复 '你好 XXX，你说了: XXX'")
	fmt.Println("[说明] 收到图片消息会自动回复 '收到图片: URL'")
	wsClient.SetMessageHandler(func(m *msg.Message) {
		switch {
		case m.IsText():
			fmt.Printf("\n[文字消息] %s(%s): %s\n", m.SenderName, m.SenderID, m.Content)
			reply := fmt.Sprintf("你好 %s，你说了: %s", m.SenderName, m.Content)
			if err := wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, reply); err != nil {
				logger.Error("发送文字失败", zap.Error(err))
			}
		case m.IsImage():
			fmt.Printf("\n[图片消息] %s(%s): %s (%dx%d)\n",
				m.SenderName, m.SenderID, m.ImageURL, m.ImageWidth, m.ImageHeight)
			reply := fmt.Sprintf("收到图片: %s", m.ImageURL)
			if err := wsClient.SendText(context.Background(), m.ConversationID, m.SenderID, reply); err != nil {
				logger.Error("发送文字失败", zap.Error(err))
			}
		default:
			fmt.Printf("\n[%v消息] %s(%s)\n", m.MessageType, m.SenderName, m.SenderID)
		}
	})

	fmt.Println("[步骤 4] 启动 Token 自动刷新（每 50 分钟刷新一次）")
	wsClient.StartTokenRefresher()

	// 建立 WebSocket 连接
	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("[步骤 5] 建立 WebSocket 连接")
	fmt.Println("[说明] 连接地址: wss://wss-goofish.dingtalk.com/connection")
	fmt.Println("[说明] 协议: LWP 3.0，支持心跳保活")
	fmt.Println()
	fmt.Println("正在连接 WebSocket...")
	if err := wsClient.ConnectWithToken(sigCtx, accessToken); err != nil {
		logger.Fatal("WebSocket 连接失败", zap.Error(err))
	}
	logger.Info("WebSocket 连接成功")

	fmt.Println()
	fmt.Println("[步骤 6] 开始监听消息")
	fmt.Println("[说明] 程序将阻塞监听，直到收到 Ctrl+C 或 SIGTERM 信号")
	fmt.Println("[说明] 可在闲鱼 APP 中向测试账号发送消息进行验证")
	fmt.Println("[提示] 按 Ctrl+C 退出程序")
	fmt.Println()
	fmt.Println(sep)
	fmt.Println("  等待消息中... (Ctrl+C 退出)")
	fmt.Println(sep)

	if err := wsClient.Start(); err != nil {
		if err != context.Canceled {
			logger.Fatal("WebSocket 错误", zap.Error(err))
		}
	}

	wsClient.Stop()
	logger.Info("程序退出")
}

// qrcodeLogin 扫码登录
func qrcodeLogin() (*apis.XianyuAPI, error) {
	return apis.QrcodeLogin(apis.QrcodeLoginConfig{
		PollInterval: 3 * time.Second,
		Timeout:      120 * time.Second,
		ShowQR:       true,
	})
}

// manualCookieAndTokenLogin 手动输入 Cookie 和 Token 登录
//
// 步骤:
//  1. 浏览器打开 https://www.goofish.com 并登录
//  2. F12 → Console → 粘贴以下代码并回车:
//     fetch('https://h5api.m.goofish.com/h5/mtop.taobao.idlemessage.pc.login.token/1.0/?jsv=2.7.2&appKey=34839810&t='+Date.now()+'&sign=&v=1.0&type=originaljson&dataType=json&timeout=20000&api=mtop.taobao.idlemessage.pc.login.token&sessionOption=AutoLoginOnly', {method:'POST', headers:{'content-type':'application/x-www-form-urlencoded'}, body:'data=%7B%22appKey%22%3A%22444e9908a51d1cb236a27862abc769c9%22%2C%22deviceId%22%3A%22test-device-id%22%7D', credentials:'include'}).then(r=>r.json()).then(d=>console.log('TOKEN:', d.data?.accessToken))
//  3. 复制输出的 TOKEN 值
//  4. 从 F12 → Network → 请求头中复制完整 Cookie
func manualCookieAndTokenLogin() (*apis.XianyuAPI, string, error) {
	fmt.Println()
	fmt.Println(sep)
	fmt.Println("  手动 Cookie + Token 登录指南")
	fmt.Println(sep)
	fmt.Println()
	fmt.Println("[步骤 1] 获取你的闲鱼用户ID (unb)")
	fmt.Println("  1.1 浏览器打开 https://www.goofish.com 并登录闲鱼")
	fmt.Println("  1.2 F12 → Application → Cookies → 找到 unb 字段")
	fmt.Println("  1.3 复制 unb 的值（纯数字）")
	fmt.Println()
	fmt.Print("请输入你的闲鱼用户ID (unb): ")
	var unb string
	fmt.Scanln(&unb)
	if unb == "" {
		return nil, "", fmt.Errorf("unb 不能为空")
	}

	deviceID := util.GenerateDeviceID(unb)
	fmt.Printf("\n[步骤 2] 已生成 DeviceID: %s\n", deviceID)
	fmt.Println("[说明] DeviceID 基于 unb 生成，用于设备标识")
	fmt.Println()

	fmt.Println("[步骤 3] 在浏览器控制台获取 Token")
	fmt.Println("  3.1 回到 https://www.goofish.com 页面")
	fmt.Println("  3.2 F12 → Console 标签页")
	fmt.Println("  3.3 先粘贴第1行加载 MD5 库，等待输出 'MD5库加载完成':")
	fmt.Println()
	fmt.Println("  第1行（加载MD5库）:")
	fmt.Println("  " + "`" + "var s=document.createElement('script');s.src='https://cdn.bootcdn.net/ajax/libs/blueimp-md5/2.19.0/js/md5.min.js';document.head.appendChild(s);setTimeout(()=>console.log('MD5库加载完成'),1000)" + "`")
	fmt.Println()
	fmt.Println("  3.4 等待 'MD5库加载完成' 后，粘贴第2行获取 Token:")
	fmt.Println()
	fmt.Println("  第2行（获取Token）:")
	fmt.Printf("  %s\n", fmt.Sprintf("`(async()=>{let t=Date.now(),tk=document.cookie.match(/_m_h5_tk=([^;]+)/)[1].split('_')[0],d=JSON.stringify({appKey:'444e9908a51d1cb236a27862abc769c9',deviceId:'%s'}),sign=md5(tk+'&'+t+'&34839810&'+d);let r=await fetch('https://h5api.m.goofish.com/h5/mtop.taobao.idlemessage.pc.login.token/1.0/?jsv=2.7.2&appKey=34839810&t='+t+'&sign='+sign+'&v=1.0&type=originaljson&dataType=json&timeout=20000&api=mtop.taobao.idlemessage.pc.login.token&sessionOption=AutoLoginOnly',{method:'POST',headers:{'content-type':'application/x-www-form-urlencoded','origin':'https://www.goofish.com','referer':'https://www.goofish.com/'},body:'data='+encodeURIComponent(d),credentials:'include'});let j=await r.json();console.log('TOKEN:',j.data?.accessToken);console.log('FULL:',JSON.stringify(j))})()`", deviceID))
	fmt.Println()
	fmt.Println("  3.5 复制控制台输出的 'TOKEN:' 后面的值")
	fmt.Println()

	fmt.Println("[步骤 4] 获取完整 Cookie")
	fmt.Println("  4.1 F12 → Network 标签页")
	fmt.Println("  4.2 刷新页面，点击第一个请求")
	fmt.Println("  4.3 在 Request Headers 中找到 Cookie 字段")
	fmt.Println("  4.4 复制完整的 Cookie 值（以 unb= 开头的一长串）")
	fmt.Println()
	fmt.Println(sep)
	fmt.Println()

	fmt.Print("请粘贴 Token: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("读取 Token 失败")
	}
	token := strings.TrimSpace(scanner.Text())

	if token == "" {
		return nil, "", fmt.Errorf("Token 不能为空")
	}
	fmt.Println("[确认] Token 已输入，长度: ", len(token))

	fmt.Print("\n请粘贴 Cookie 字符串: ")
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("读取 Cookie 失败")
	}
	cookieStr := strings.TrimSpace(scanner.Text())

	if cookieStr == "" {
		return nil, "", fmt.Errorf("Cookie 不能为空")
	}

	cookies := util.ParseCookies(cookieStr)
	if _, ok := cookies["unb"]; !ok {
		return nil, "", fmt.Errorf("Cookie 中缺少 unb 字段，请确保已登录")
	}

	fmt.Printf("[确认] 解析到 %d 个 Cookie 字段，unb=%s\n", len(cookies), cookies["unb"])

	api, err := apis.New(cookies, deviceID)
	if err != nil {
		return nil, "", fmt.Errorf("创建 API 实例失败: %w", err)
	}

	fmt.Println("[确认] API 实例创建成功，可以开始 WebSocket 通信")
	return api, token, nil
}

// previewCookie 生成 Cookie 预览字符串
func previewCookie(api *apis.XianyuAPI) string {
	cookieStr := api.CookieString()
	if cookieStr == "" {
		return "(empty)"
	}
	parts := strings.Split(cookieStr, "; ")
	var preview []string
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			v := kv[1]
			if len(v) > 10 {
				v = v[:10] + "..."
			}
			preview = append(preview, kv[0]+"="+v)
		}
	}
	return strings.Join(preview, "; ")
}

// ============================================================
// 通用辅助函数
// ============================================================

// buildAPIFromCookie 从用户输入的 Cookie 创建 XianyuAPI 实例。
//
// 创建后自动设置风控验证码处理回调：
//   - 触发风控时提取验证链接，打印给用户
//   - 用户在浏览器中完成验证后，重新粘贴 Cookie
//   - 程序更新 Cookie 并自动重试
func buildAPIFromCookie() (*apis.XianyuAPI, error) {
	fmt.Print("请粘贴 Cookie 字符串: ")
	cookieStr := readLine()
	if cookieStr == "" {
		return nil, fmt.Errorf("Cookie 不能为空")
	}

	cookies := util.ParseCookies(cookieStr)
	if _, ok := cookies["unb"]; !ok {
		return nil, fmt.Errorf("Cookie 中缺少 unb 字段，请确保已登录")
	}

	fmt.Printf("[确认] 解析到 %d 个 Cookie 字段，unb=%s\n", len(cookies), cookies["unb"])
	api, err := apis.New(cookies, "")
	if err != nil {
		return nil, err
	}

	// 设置风控验证码处理回调
	api.SetCaptchaHandler(captchaHandler)

	return api, nil
}

// captchaHandler 风控验证码处理回调。
//
// 当 mtop 接口返回风控验证码时调用：
//  1. 打印验证链接，提示用户在浏览器中打开
//  2. 等待用户完成验证后重新粘贴 Cookie
//  3. 返回新 Cookie 供 API 实例更新
func captchaHandler(verifyURL string) (string, error) {
	fmt.Println()
	fmt.Println(sep)
	fmt.Println("  [风控验证码处理]")
	fmt.Println(sep)
	fmt.Println()
	fmt.Println("[步骤 1] 在浏览器中打开以下验证链接:")
	fmt.Println()
	fmt.Printf("  %s\n", verifyURL)
	fmt.Println()
	fmt.Println("[步骤 2] 在浏览器中完成验证（滑动验证码/短信验证等）")
	fmt.Println("[说明] 验证完成后，页面会显示成功或自动跳转")
	fmt.Println()
	fmt.Println("[步骤 3] 完成验证后，重新复制 Cookie:")
	fmt.Println("  3.1 F12 → Network 标签页")
	fmt.Println("  3.2 刷新 https://www.goofish.com 页面")
	fmt.Println("  3.3 点击第一个请求，复制 Request Headers 中的 Cookie 值")
	fmt.Println()
	fmt.Println("[提示] 验证后 Cookie 会更新，必须重新复制")
	fmt.Println("[提示] 输入 q 取消验证并退出")
	fmt.Println()

	fmt.Print("请粘贴验证后的新 Cookie（或输入 q 取消）: ")
	newCookie := readLine()
	if newCookie == "" {
		return "", fmt.Errorf("用户未输入 Cookie")
	}
	if strings.EqualFold(newCookie, "q") {
		return "", fmt.Errorf("用户取消验证")
	}

	// 校验新 Cookie 包含 unb
	newCookies := util.ParseCookies(newCookie)
	if _, ok := newCookies["unb"]; !ok {
		return "", fmt.Errorf("新 Cookie 中缺少 unb 字段，请确保已登录")
	}

	fmt.Printf("[确认] 新 Cookie 已接收，解析到 %d 个字段\n", len(newCookies))
	return newCookie, nil
}

// readLine 读取一行输入（支持空格）。
func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}

// readInput 打印提示并读取一行输入。
func readInput(prompt string) string {
	fmt.Print(prompt)
	return readLine()
}

// previewStr 截取字符串前 n 个字符并加省略号。
func previewStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// printResultPreview 打印 map 响应的预览（前 500 字符）。
func printResultPreview(result map[string]any) {
	if result == nil {
		fmt.Printf("  响应为 nil\n")
		return
	}
	// 检查 ret 字段
	if ret, ok := result["ret"].([]any); ok && len(ret) > 0 {
		fmt.Printf("  ret: %v\n", ret)
	}
	// 打印 data 字段预览
	if data, ok := result["data"]; ok {
		dataJSON, _ := json.Marshal(data)
		preview := string(dataJSON)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("  data 预览: %s\n", preview)
	}
}

// printOpenResp 打印开放平台 API 响应。
func printOpenResp(resp *open.ApiResponse) {
	if resp == nil {
		fmt.Printf("  响应为 nil\n")
		return
	}
	fmt.Printf("  %s\n", resp.String())
	if resp.IsSuccess() {
		// 打印 data 预览
		if len(resp.Data) > 0 {
			preview := string(resp.Data)
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			fmt.Printf("  data 预览: %s\n", preview)
		}
	}
}

// printSearchItems 打印搜索结果列表。
func printSearchItems(items []*search.Item, limit int) {
	if len(items) == 0 {
		fmt.Printf("  未找到商品\n")
		return
	}
	show := len(items)
	if show > limit {
		show = limit
	}
	fmt.Printf("  共 %d 个商品，展示前 %d 个:\n", len(items), show)
	for i := 0; i < show; i++ {
		item := items[i]
		fmt.Printf("  %d. %s\n", i+1, item.Title)
		fmt.Printf("     卖家: %s | 地区: %s | 价格: %s\n",
			item.UserName, item.Area, item.SoldPrice)
		fmt.Printf("     详情: %s\n", item.DetailURL)
	}
}

// printTestSummary 打印测试汇总。
func printTestSummary(module string, passed, failed int) {
	fmt.Println()
	fmt.Println(sep)
	fmt.Printf("  【%s】测试汇总\n", module)
	fmt.Println(sep)
	fmt.Printf("  通过: %d, 失败: %d, 总计: %d\n", passed, failed, passed+failed)
	if failed > 0 {
		fmt.Printf("  ⚠ 有 %d 个测试失败\n", failed)
		fmt.Println()
		fmt.Println("[排查建议]")
		fmt.Println("  1. 检查 Cookie/Token 是否过期，重新获取后重试")
		fmt.Println("  2. 检查网络连接是否正常")
		fmt.Println("  3. 检查请求参数是否正确（如商品ID、订单号是否存在）")
		fmt.Println("  4. 若返回签名错误，尝试 RefreshMtopToken 后重试")
		fmt.Println("  5. 若返回限流错误，等待 1-2 分钟后重试")
	} else {
		fmt.Printf("  ✓ 全部通过\n")
	}
	fmt.Println(sep)
}

// printCookieTroubleshooting 打印 Cookie 相关错误的排查建议。
func printCookieTroubleshooting(err error) {
	fmt.Println()
	fmt.Println("[排查建议]")
	fmt.Println("  1. 确认闲鱼账号已登录: 访问 https://www.goofish.com 检查登录状态")
	fmt.Println("  2. 确认 Cookie 完整: F12 → Network → 复制完整的 Cookie 值")
	fmt.Println("  3. 确认 Cookie 包含 unb 字段: 这是用户ID，缺失表示未登录")
	fmt.Println("  4. Cookie 可能已过期: 重新登录闲鱼并获取新的 Cookie")
	fmt.Printf("  5. 错误详情: %v\n", err)
}

// modelDeliverySettings 返回默认配送设置（包邮）。
func modelDeliverySettings() model.DeliverySettings {
	return model.DeliverySettings{
		Choice:        "包邮",
		CanSelfPickup: false,
	}
}
