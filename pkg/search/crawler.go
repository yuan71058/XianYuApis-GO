// Package search 封装闲鱼商品搜索爬虫。
//
// 对应 Python 版 XIE7654/goofish_api/spider/xianyu_sign.py，
// 基于 mtop.taobao.idlemtopsearch.pc.search API 实现关键词搜索。
//
// 与 pkg/open（开放平台官方 API）不同，本包使用浏览器 Cookie 鉴权，
// 属于逆向 API，适用于个人搜索爬取场景。
//
// 依赖 pkg/apis.XianyuAPI 的 DoMtopRequest 公开方法发送签名请求。
//
// 基本用法：
//
//	api, _ := apis.New(cookies, "")
//	crawler := search.New(api)
//	items, err := crawler.Search(ctx, &search.Request{
//	    Keyword:     "bebebus安全座椅",
//	    PageNumber:  1,
//	    RowsPerPage: 30,
//	})
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cv-cat/xianyuapis/pkg/apis"
)

// 详情页 URL 模板。
const DetailURLTemplate = "https://www.goofish.com/item?id=%s"

// Request 搜索请求参数。
type Request struct {
	Keyword     string `json:"keyword"`             // 搜索关键词（必需）
	PageNumber  int    `json:"pageNumber"`          // 页码，从 1 开始
	RowsPerPage int    `json:"rowsPerPage"`         // 每页数量，默认 30
	FromFilter  bool   `json:"fromFilter"`          // 是否来自筛选
	SortValue   string `json:"sortValue,omitempty"` // 排序值
	SortField   string `json:"sortField,omitempty"` // 排序字段
}

// Item 搜索结果中的单个商品。
//
// 对应 Python 版 parse_item_data 解析后的结构。
type Item struct {
	UserName  string `json:"user_name"`  // 卖家用户名
	Area      string `json:"area"`       // 地区
	SoldPrice string `json:"sold_price"` // 售价
	Title     string `json:"title"`      // 商品标题
	DetailURL string `json:"detail_url"` // 详情页 URL
	ItemID    string `json:"item_id"`    // 商品 ID
	CreatedAt string `json:"created_at"` // 采集时间
}

// Crawler 闲鱼商品搜索爬虫。
//
// 对应 Python 版 XianYuCrawler，基于 apis.XianyuAPI 发送签名请求。
type Crawler struct {
	api *apis.XianyuAPI
}

// New 创建搜索爬虫实例。
//
// 参数：
//   - api: 已登录的 apis.XianyuAPI 实例（需包含有效 Cookie）
func New(api *apis.XianyuAPI) *Crawler {
	return &Crawler{api: api}
}

// Search 执行关键词搜索，返回单页结果。
//
// 对应 Python 版 fetch_page_data + parse_item_data。
//
// 参数：
//   - ctx: 请求上下文
//   - req:  搜索请求参数
//
// 返回值：
//   - []Item: 解析后的商品列表
//   - error: 请求或解析错误
func (c *Crawler) Search(ctx context.Context, req *Request) ([]*Item, error) {
	if req == nil {
		return nil, fmt.Errorf("search: request is nil")
	}
	if req.Keyword == "" {
		return nil, fmt.Errorf("search: keyword is required")
	}
	if req.PageNumber <= 0 {
		req.PageNumber = 1
	}
	if req.RowsPerPage <= 0 {
		req.RowsPerPage = 30
	}

	// 构建请求数据（与 Python 版 data_dict 结构一致）
	dataDict := map[string]any{
		"pageNumber":        req.PageNumber,
		"keyword":           req.Keyword,
		"fromFilter":        req.FromFilter,
		"rowsPerPage":       req.RowsPerPage,
		"sortValue":         req.SortValue,
		"sortField":         req.SortField,
		"customDistance":    "",
		"gps":               "",
		"propValueStr":      map[string]string{"searchFilter": "publishDays:1;"},
		"customGps":         "",
		"searchReqFromPage": "pcSearch",
		"extraFilterValue":  "{}",
		"userPositionJson":  "{}",
	}

	dataJSON, err := json.Marshal(dataDict)
	if err != nil {
		return nil, fmt.Errorf("search: marshal data: %w", err)
	}

	// 通过 apis.XianyuAPI.DoMtopRequest 发送签名请求
	// API: mtop.taobao.idlemtopsearch.pc.search / 1.0
	extra := url.Values{
		"spm_cnt":     {"a21ybx.search.0.0"},
		"spm_pre":     {"a21ybx.home.searchInput.0"},
		"accountSite": {"xianyu"},
	}

	result, err := c.api.DoMtopRequest(ctx,
		"mtop.taobao.idlemtopsearch.pc.search", "1.0", string(dataJSON), extra)
	if err != nil {
		return nil, fmt.Errorf("search: do mtop request: %w", err)
	}

	// 检查响应状态
	if ret, ok := result["ret"].([]any); ok && len(ret) > 0 {
		if retStr, ok := ret[0].(string); ok && !containsSuccess(retStr) {
			return nil, fmt.Errorf("search: request failed: %s", retStr)
		}
	}

	return parseSearchResult(result), nil
}

// SearchAll 搜索所有页（最多 maxPages 页），聚合结果。
//
// 对应 Python 版 run() 方法的主循环。
//
// 参数：
//   - ctx:       请求上下文
//   - req:       搜索请求（PageNumber 会被忽略，从 1 开始递增）
//   - maxPages:  最大页数
func (c *Crawler) SearchAll(ctx context.Context, req *Request, maxPages int) ([]*Item, error) {
	if maxPages <= 0 {
		maxPages = 1
	}

	var allItems []*Item
	for page := 1; page <= maxPages; page++ {
		req.PageNumber = page
		items, err := c.Search(ctx, req)
		if err != nil {
			return allItems, fmt.Errorf("search: page %d failed: %w", page, err)
		}
		if len(items) == 0 {
			break // 无更多数据
		}
		allItems = append(allItems, items...)
	}
	return allItems, nil
}

// parseSearchResult 从 mtop 响应中解析商品列表。
//
// 对应 Python 版 parse_item_data，从 data.resultList[].data.item.main.exContent 提取字段。
func parseSearchResult(result map[string]any) []*Item {
	data, _ := result["data"].(map[string]any)
	if data == nil {
		return nil
	}
	resultList, _ := data["resultList"].([]any)
	if len(resultList) == 0 {
		return nil
	}

	var items []*Item
	for _, raw := range resultList {
		item := parseItem(raw)
		if item != nil {
			items = append(items, item)
		}
	}
	return items
}

// parseItem 解析单个商品数据。
//
// 路径: item.data.item.main.exContent
// 字段: userNickName, area, detailParams.title, detailParams.soldPrice, itemId
func parseItem(raw any) *Item {
	itemMap, _ := raw.(map[string]any)
	if itemMap == nil {
		return nil
	}

	data, _ := itemMap["data"].(map[string]any)
	if data == nil {
		return nil
	}
	item, _ := data["item"].(map[string]any)
	if item == nil {
		return nil
	}
	main, _ := item["main"].(map[string]any)
	if main == nil {
		return nil
	}
	exContent, _ := main["exContent"].(map[string]any)
	if exContent == nil {
		return nil
	}

	detailParams, _ := exContent["detailParams"].(map[string]any)

	return &Item{
		UserName:  getString(exContent, "userNickName"),
		Area:      getString(exContent, "area"),
		SoldPrice: getString(detailParams, "soldPrice"),
		Title:     getString(detailParams, "title"),
		ItemID:    getString(exContent, "itemId"),
		DetailURL: fmt.Sprintf(DetailURLTemplate, getString(exContent, "itemId")),
	}
}

// getString 从 map 中安全读取字符串字段。
func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return fmt.Sprintf("%v", s)
	default:
		return fmt.Sprintf("%v", s)
	}
}

// containsSuccess 检查响应 ret 字段是否包含 SUCCESS。
func containsSuccess(s string) bool {
	return len(s) >= 7 && s[:7] == "SUCCESS"
}
