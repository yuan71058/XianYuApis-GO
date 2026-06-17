package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/model"
)

// GetItemInfo 获取闲鱼商品详情。
//
// 参数:
//   - ctx:    请求上下文
//   - itemID: 闲鱼商品 ID
//
// 返回值:
//   - map[string]any: 商品详情 JSON（与原版结构一致）
//   - error: 获取失败时的错误
func (api *XianyuAPI) GetItemInfo(ctx context.Context, itemID string) (map[string]any, error) {
	dataVal := fmt.Sprintf(`{"itemId":"%s"}`, itemID)

	extra := map[string]string{
		"spm_cnt": "a21ybx.im.0.0",
		"spm_pre": "a21ybx.item.want.1.12523da6waCtUp",
		"log_id":  "12523da6waCtUp",
	}

	return api.doMtopRequest(ctx,
		"mtop.taobao.idle.pc.detail", "1.0", dataVal, toValues(extra))
}

// GetPublicChannel 获取商品推荐标签和分类建议。
//
// 用于商品发布时自动推荐属性标签和分类。
//
// 参数:
//   - ctx:    请求上下文
//   - title:  商品标题/描述
//   - images: 已上传的图片信息列表
//
// 返回值:
//   - map[string]any: 推荐结果 JSON，包含 cardList 和 categoryPredictResult
//   - error: 获取失败时的错误
func (api *XianyuAPI) GetPublicChannel(ctx context.Context, title string, images []model.ImageInfo) (map[string]any, error) {
	data := map[string]any{
		"title":        title,
		"lockCpv":      false,
		"multiSKU":     false,
		"publishScene": "mainPublish",
		"scene":        "newPublishChoice",
		"description":  title,
		"imageInfos":   buildImageInfos(images),
		"uniqueCode":   "1775905618164677",
	}

	dataVal, _ := json.Marshal(data)

	extra := map[string]string{
		"v":       "2.0",
		"spm_cnt": "a21ybx.publish.0.0",
		"spm_pre": "a21ybx.item.sidebar.1.67321598K9Vgx8",
		"log_id":  "67321598K9Vgx8",
	}

	return api.doMtopRequest(ctx,
		"mtop.taobao.idle.kgraph.property.recommend", "2.0", string(dataVal), toValues(extra))
}

// GetDefaultLocation 获取默认地理位置信息。
//
// 返回当前账号注册或最近使用的地理位置，用于商品发布时填写发货地址。
//
// 返回值:
//   - map[string]any: 位置信息 JSON，包含 area、city、prov、gps 等字段
//   - error: 获取失败时的错误
func (api *XianyuAPI) GetDefaultLocation(ctx context.Context) (map[string]any, error) {
	dataVal := `{"longitude":118.78248347393424,"latitude":31.91629189813543}`

	extra := map[string]string{
		"spm_cnt": "a21ybx.publish.0.0",
		"spm_pre": "a21ybx.item.sidebar.1.38262218ame5nr",
		"log_id":  "38262218ame5nr",
	}

	result, err := api.doMtopRequest(ctx,
		"mtop.taobao.idle.local.poi.get", "1.0", dataVal, toValues(extra))
	if err != nil {
		return nil, fmt.Errorf("apis: get default location: %w", err)
	}

	return result, nil
}

// PublishItem 发布闲鱼商品。
//
// 完整发布流程:
//  1. 上传所有图片到闲鱼服务器
//  2. 获取推荐标签和分类建议
//  3. 获取默认地理位置
//  4. 构建并发布商品请求
//
// 参数:
//   - ctx:    请求上下文
//   - images: 本地图片文件路径列表
//   - desc:   商品标题和描述
//   - price:  价格信息（nil 时使用系统默认定价）
//   - ds:     配送设置
//
// 返回值:
//   - map[string]any: 发布结果 JSON
//   - error: 发布失败时的错误
func (api *XianyuAPI) PublishItem(ctx context.Context, images []string, desc string,
	price *model.Price, ds model.DeliverySettings,
) (map[string]any, error) {

	// Step 1: 上传所有图片
	var imageInfos []model.ImageInfo
	for _, imgPath := range images {
		result, err := api.UploadMedia(ctx, imgPath)
		if err != nil {
			return nil, fmt.Errorf("apis: upload image %s: %w", imgPath, err)
		}
		imageInfos = append(imageInfos, model.ImageInfo{
			URL:    result.URL,
			Width:  result.Width,
			Height: result.Height,
		})
	}

	// Step 2: 获取推荐标签和分类
	channelResp, err := api.GetPublicChannel(ctx, desc, imageInfos)
	if err != nil {
		return nil, fmt.Errorf("apis: get public channel: %w", err)
	}

	// Step 3: 获取默认位置
	locResp, err := api.GetDefaultLocation(ctx)
	if err != nil {
		return nil, fmt.Errorf("apis: get default location: %w", err)
	}

	// Step 4: 构建并发布
	dataVal := buildPublishPayload(desc, imageInfos, price, ds, channelResp, locResp)
	extra := map[string]string{
		"spm_cnt": "a21ybx.publish.0.0",
		"spm_pre": "a21ybx.home.sidebar.1.46413da6EPl7v5",
		"log_id":  "46413da6EPl7v5",
	}

	return api.doMtopRequest(ctx,
		"mtop.idle.pc.idleitem.publish", "1.0", dataVal, toValues(extra))
}

// buildImageInfos 将 model.ImageInfo 转换为 API 请求格式。
func buildImageInfos(images []model.ImageInfo) []map[string]any {
	result := make([]map[string]any, len(images))
	for i, img := range images {
		result[i] = map[string]any{
			"extraInfo":   map[string]any{"isH": "false", "isT": "false", "raw": "false"},
			"isQrCode":    false,
			"url":         img.URL,
			"heightSize":  img.Height,
			"widthSize":   img.Width,
			"major":       true,
			"type":        0,
			"status":      "done",
		}
	}
	return result
}

// buildPublishPayload 构建商品发布请求体。
// 对齐 Python 版 goofish_apis.py public() 方法的 payload 格式
func buildPublishPayload(desc string, images []model.ImageInfo,
	price *model.Price, ds model.DeliverySettings,
	channelResp, locResp map[string]any,
) string {
	data := map[string]any{
		"freebies":    false,
		"itemTypeStr": "b",
		"quantity":    "1",    // Python 版本为字符串 "1"
		"simpleItem":  "true", // Python 版本为字符串 "true"
		"imageInfoDOList": buildImageDOList(images),
		"itemTextDTO": map[string]any{
			"desc":              desc,
			"title":             desc,
			"titleDescSeparate": false,
		},
		"itemLabelExtList": buildItemLabelExtList(channelResp),
		"itemPriceDTO":     buildPriceDTO(price),
		"userRightsProtocols": []any{
			map[string]any{"enable": false, "serviceCode": "SKILL_PLAY_NO_MIND"},
		},
		"itemPostFeeDTO": buildPostFeeDTO(ds),
		"itemAddrDTO":    buildAddrDTO(locResp),
		"itemCatDTO":     buildCatDTO(channelResp),
		"defaultPrice":   price == nil,
		"uniqueCode":     fmt.Sprintf("%d", time.Now().UnixMilli()), // 动态生成，避免重复
		"sourceId":       "pcMainPublish",
		"bizcode":        "pcMainPublish",
		"publishScene":   "pcMainPublish",
	}

	val, _ := json.Marshal(data)
	return string(val)
}

// buildImageDOList 构建图片 DO 列表。
func buildImageDOList(images []model.ImageInfo) []map[string]any {
	list := make([]map[string]any, len(images))
	for i, img := range images {
		list[i] = map[string]any{
			"extraInfo":   map[string]any{"isH": "false", "isT": "false", "raw": "false"},
			"isQrCode":    false,
			"url":         img.URL,
			"heightSize":  img.Height,
			"widthSize":   img.Width,
			"major":       true,
			"type":        0,
			"status":      "done",
		}
	}
	return list
}

// buildPriceDTO 构建价格 DTO。
func buildPriceDTO(price *model.Price) map[string]any {
	dto := map[string]any{}
	if price != nil {
		if price.CurrentPrice > 0 {
			dto["priceInCent"] = fmt.Sprintf("%d", int(price.CurrentPrice*100))
		}
		if price.OriginalPrice > 0 {
			dto["origPriceInCent"] = fmt.Sprintf("%d", int(price.OriginalPrice*100))
		}
	}
	return dto
}

// buildPostFeeDTO 构建运费 DTO。
func buildPostFeeDTO(ds model.DeliverySettings) map[string]any {
	dto := map[string]any{
		"canFreeShipping": false,
		"supportFreight":  false,
		"onlyTakeSelf":    false,
	}
	switch ds.Choice {
	case "包邮":
		dto["canFreeShipping"] = true
		dto["supportFreight"] = true
	case "按距离计费":
		dto["supportFreight"] = true
		dto["templateId"] = "-100"
	case "一口价":
		dto["supportFreight"] = true
		dto["templateId"] = "0"
		if ds.PostPrice > 0 {
			dto["postPriceInCent"] = fmt.Sprintf("%d", int(ds.PostPrice*100))
		}
	case "无需邮寄":
		dto["templateId"] = "0"
	}
	if ds.CanSelfPickup {
		dto["onlyTakeSelf"] = true
	}
	return dto
}

// buildAddrDTO 从位置响应中构建地址 DTO。
func buildAddrDTO(locResp map[string]any) map[string]any {
	data, _ := locResp["data"].(map[string]any)
	if data == nil {
		return map[string]any{}
	}
	addrs, _ := data["commonAddresses"].([]any)
	if len(addrs) == 0 {
		return map[string]any{}
	}
	addr, _ := addrs[0].(map[string]any)
	return map[string]any{
		"area":       toString(addr["area"]),
		"city":       toString(addr["city"]),
		"divisionId": toString(addr["divisionId"]),
		"gps":        fmt.Sprintf("%s,%s", toString(addr["longitude"]), toString(addr["latitude"])),
		"poiId":      toString(addr["poiId"]),
		"poiName":    toString(addr["poi"]),
		"prov":       toString(addr["prov"]),
	}
}

// buildCatDTO 从频道响应中提取分类信息。
func buildCatDTO(channelResp map[string]any) map[string]any {
	data, _ := channelResp["data"].(map[string]any)
	if data == nil {
		return map[string]any{}
	}
	predict, _ := data["categoryPredictResult"].(map[string]any)
	if predict == nil {
		return map[string]any{}
	}
	return map[string]any{
		"catId":        toString(predict["catId"]),
		"catName":      toString(predict["catName"]),
		"channelCatId": toString(predict["channelCatId"]),
		"tbCatId":      toString(predict["tbCatId"]),
	}
}

// buildItemLabelExtList 从推荐标签结果中提取用户已选中的标签。
// 对齐 Python 版 goofish_apis.py public() 方法中的标签构建逻辑。
// 修复：Python 版中 None 值字段仍会序列化为 null，但闲鱼可能不接受 null，
// 改为省略这些字段（与浏览器实际抓包一致）
func buildItemLabelExtList(channelResp map[string]any) []map[string]any {
	labels := []map[string]any{}
	data, _ := channelResp["data"].(map[string]any)
	if data == nil {
		return labels
	}
	cardList, _ := data["cardList"].([]any)

	for _, card := range cardList {
		cardData, _ := card.(map[string]any)
		cardDataInner, _ := cardData["cardData"].(map[string]any)
		if cardDataInner == nil {
			continue
		}
		propName, _ := cardDataInner["propertyName"].(string)
		propID, _ := cardDataInner["propertyId"].(string)

		valuesList, _ := cardDataInner["valuesList"].([]any)
		if valuesList == nil {
			valuesList = []any{}
		}

		for _, cv := range valuesList {
			cvMap, _ := cv.(map[string]any)
			if clicked, ok := cvMap["isClicked"].(bool); ok && clicked {
				label := map[string]any{
					"channelCateName": cvMap["catName"],
					"channelCatId":    cvMap["channelCatId"],
					"tbCatId":         cvMap["tbCatId"],
					"labelType":       "common",
					"propertyName":    propName,
					"isUserClick":     "1",
					"from":            "newPublishChoice",
					"propertyId":      propID,
					"labelFrom":       "newPublish",
					"text":            cvMap["catName"],
					"properties":      fmt.Sprintf("%s##%s:%s##%s", propID, propName, cvMap["channelCatId"], cvMap["catName"]),
				}
				labels = append(labels, label)
				break
			}
		}
	}
	return labels
}

// ConfirmShipping 自动确认发货。
//
// 与 Python 版 ConfirmShippingService.auto_confirm 对齐:
// 调用 mtop.taobao.idle.logistic.consign.dummy API 确认订单发货。
// 适用于虚拟商品自动发货场景，确认后买家款项将到账。
//
// 参数:
//   - ctx:     请求上下文
//   - orderID: 订单 ID
//
// 返回值:
//   - map[string]any: 确认结果 JSON，成功时 ret 包含 "SUCCESS::调用成功"
//   - error: 请求失败时的错误
func (api *XianyuAPI) ConfirmShipping(ctx context.Context, orderID string) (map[string]any, error) {
	// 构建请求数据（与 Python 版 data_val 格式一致）
	dataVal := fmt.Sprintf(`{"orderId":"%s","tradeText":"","picList":[],"newUnconsign":true}`, orderID)

	extra := map[string]string{
		"spm_cnt": "a21ybx.im.0.0",
	}

	result, err := api.doMtopRequest(ctx,
		"mtop.taobao.idle.logistic.consign.dummy", "1.0", dataVal, toValues(extra))
	if err != nil {
		return nil, fmt.Errorf("apis: confirm shipping: %w", err)
	}

	return result, nil
}

// toString 将任意值转为字符串。
// 修复：float64 使用 strconv.FormatFloat 避免大数字产生科学计数法
// 例如 catId=50014803，%g 会输出 "5.00148e+07"，而 'f' 格式输出 "50014803"
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", s)
	}
}
