package model

// Price 商品价格信息。
type Price struct {
	CurrentPrice  float64 `json:"current_price"`  // 当前售价（元）
	OriginalPrice float64 `json:"original_price"` // 原价（元）
}

// DeliverySettings 配送设置。
// Choice 可选值: "包邮" | "按距离计费" | "一口价" | "无需邮寄"
type DeliverySettings struct {
	Choice        string  `json:"choice"`          // 配送方式
	PostPrice     float64 `json:"post_price"`      // 运费（元，仅一口价时有效）
	CanSelfPickup bool    `json:"can_self_pickup"` // 是否支持自提
}

// ImageInfo 已上传的图片信息。
type ImageInfo struct {
	URL    string `json:"url"`    // 图片访问 URL
	Width  int    `json:"width"`  // 宽度
	Height int    `json:"height"` // 高度
}

// PublishResult 商品发布结果。
type PublishResult struct {
	ItemID  string `json:"itemId"`  // 发布的商品 ID
	Status  string `json:"status"`  // 发布状态
	Message string `json:"message"` // 状态描述
}
