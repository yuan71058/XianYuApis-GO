// XianYuApis-GO 开放平台 API 与搜索爬虫 Demo
//
// 演示 pkg/open（闲鱼开放平台 SDK）和 pkg/search（商品搜索爬虫）的用法。
//
// 运行方式: go run ./cmd/open_demo/
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cv-cat/xianyuapis/pkg/apis"
	"github.com/cv-cat/xianyuapis/pkg/open"
	"github.com/cv-cat/xianyuapis/pkg/search"
	"github.com/cv-cat/xianyuapis/pkg/util"
)

func main() {
	fmt.Println("====================================")
	fmt.Println("  闲鱼开放平台 API & 搜索爬虫 Demo")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("选择功能:")
	fmt.Println("  1. 开放平台 API（app_key 鉴权，商家管理）")
	fmt.Println("  2. 商品搜索爬虫（cookie 鉴权，关键词搜索）")
	fmt.Println()
	fmt.Print("请输入选择 (1/2): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		runOpenAPIDemo()
	case "2":
		runSearchDemo()
	default:
		fmt.Println("无效选择")
		os.Exit(1)
	}
}

// runOpenAPIDemo 演示开放平台 API 用法。
//
// 开放平台 API 使用 app_key + app_secret 鉴权，适用于商家商品/订单管理。
// 需要先在闲鱼开放平台申请应用获取 app_key 和 app_secret。
func runOpenAPIDemo() {
	fmt.Println("\n--- 开放平台 API Demo ---")
	fmt.Println("请先在闲鱼开放平台 (https://open.goofish.pro) 申请应用")

	var appKey, appSecret string
	fmt.Print("请输入 app_key: ")
	fmt.Scanln(&appKey)
	fmt.Print("请输入 app_secret: ")
	fmt.Scanln(&appSecret)

	if appKey == "" || appSecret == "" {
		fmt.Println("app_key 和 app_secret 不能为空")
		return
	}

	// 创建客户端
	client := open.NewClient(appKey, appSecret, open.WithDebug(false))
	ctx := context.Background()

	// 1. 查询授权店铺
	fmt.Println("\n[1] 查询授权店铺列表...")
	resp, err := client.User.GetAuthorizeList(ctx)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	fmt.Printf("响应: %s\n", resp)
	if resp.IsSuccess() {
		fmt.Println("请求成功")
		resp.UnmarshalData(&struct{}{}) // 按需解析
	} else {
		fmt.Printf("请求失败: code=%d, message=%s\n", resp.Code, resp.Message)
	}

	// 2. 查询商品类目
	fmt.Println("\n[2] 查询商品类目（普通商品/手机）...")
	spBizType := open.SpBizTypeMobile
	resp, err = client.Good.GetProductCategoryList(ctx,
		open.ItemBizTypeCommon, &spBizType, nil)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	fmt.Printf("响应: %s\n", resp)

	// 3. 查询商品列表
	fmt.Println("\n[3] 查询商品列表（销售中）...")
	saleStatus := open.SaleStatusOnSale
	resp, err = client.Good.GetProductList(ctx, &open.GetProductListRequest{
		SaleStatus: &saleStatus,
		PageNo:     1,
		PageSize:   20,
	})
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	fmt.Printf("响应: %s\n", resp)

	// 4. 查询快递公司
	fmt.Println("\n[4] 查询快递公司列表...")
	resp, err = client.Other.GetExpressCompanies(ctx)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	fmt.Printf("响应: %s\n", resp)
}

// runSearchDemo 演示商品搜索爬虫用法。
//
// 搜索爬虫使用浏览器 Cookie 鉴权，基于 mtop.taobao.idlemtopsearch.pc.search API。
// 需要先从浏览器获取闲鱼登录 Cookie。
func runSearchDemo() {
	fmt.Println("\n--- 商品搜索爬虫 Demo ---")
	fmt.Println("需要闲鱼登录 Cookie（从浏览器 F12 → Network → 请求头复制）")

	var cookieStr, keyword string
	fmt.Print("请粘贴 Cookie 字符串: ")
	fmt.Scanln(&cookieStr)
	if cookieStr == "" {
		fmt.Println("Cookie 不能为空")
		return
	}

	fmt.Print("请输入搜索关键词: ")
	fmt.Scanln(&keyword)
	if keyword == "" {
		keyword = "机械键盘"
		fmt.Printf("使用默认关键词: %s\n", keyword)
	}

	// 解析 Cookie 并创建 API 实例
	cookies := util.ParseCookies(cookieStr)
	if _, ok := cookies["unb"]; !ok {
		fmt.Println("Cookie 中缺少 unb 字段，请确保已登录")
		return
	}

	api, err := apis.New(cookies, "")
	if err != nil {
		fmt.Printf("创建 API 实例失败: %v\n", err)
		return
	}

	// 创建搜索爬虫
	crawler := search.New(api)
	ctx := context.Background()

	// 搜索第 1 页
	fmt.Printf("\n搜索 \"%s\" 第 1 页...\n", keyword)
	items, err := crawler.Search(ctx, &search.Request{
		Keyword:     keyword,
		PageNumber:  1,
		RowsPerPage: 30,
	})
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
		return
	}

	if len(items) == 0 {
		fmt.Println("未找到商品")
		return
	}

	fmt.Printf("\n找到 %d 个商品:\n", len(items))
	for i, item := range items {
		fmt.Printf("  %d. %s\n", i+1, item.Title)
		fmt.Printf("     卖家: %s | 地区: %s | 价格: %s\n",
			item.UserName, item.Area, item.SoldPrice)
		fmt.Printf("     详情: %s\n", item.DetailURL)
	}
}
