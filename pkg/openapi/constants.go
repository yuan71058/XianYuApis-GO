// Package openapi — 闲鱼开放平台 API 枚举常量。
//
// 翻译自 Python SDK goofish_api/utils/constants.py
// 所有枚举值为闲鱼开放平台固定标识，不可修改。
package openapi

// ItemBizType 商品类型枚举。
//
// 对应 Python: ItemBizType
type ItemBizType int

const (
	ItemBizTypeCommon        ItemBizType = 2  // 普通商品
	ItemBizTypeInspected     ItemBizType = 0  // 已验货
	ItemBizTypeInspectionBao ItemBizType = 10 // 验货宝
	ItemBizTypeBrandAuth     ItemBizType = 16 // 品牌授权
	ItemBizTypeXianYuSelected ItemBizType = 19 // 闲鱼严选
	ItemBizTypeXianYuFlash   ItemBizType = 24 // 闲鱼特卖
	ItemBizTypeBrandPick     ItemBizType = 26 // 品牌捡漏
)

// SpBizType 行业类型枚举。
//
// 对应 Python: SpBizType
type SpBizType int

const (
	SpBizTypeMobile       SpBizType = 1  // 手机
	SpBizTypeTrend        SpBizType = 2  // 潮品
	SpBizTypeHomeAppliance SpBizType = 3 // 家电
	SpBizTypeInstrument   SpBizType = 8  // 乐器
	SpBizTypeDigital      SpBizType = 9  // 3C数码
	SpBizTypeLuxury       SpBizType = 16 // 奢品
	SpBizTypeMaternal     SpBizType = 17 // 母婴
	SpBizTypeBeauty       SpBizType = 18 // 美妆个护
	SpBizTypeJewelry      SpBizType = 19 // 文玩/珠宝
	SpBizTypeGaming       SpBizType = 20 // 游戏电玩
	SpBizTypeHome         SpBizType = 21 // 家居
	SpBizTypeVirtualGame  SpBizType = 22 // 虚拟游戏
	SpBizTypeAccountRental SpBizType = 23 // 租号
	SpBizTypeBook         SpBizType = 24 // 图书
	SpBizTypeVoucher      SpBizType = 25 // 卡券
	SpBizTypeFood         SpBizType = 27 // 食品
	SpBizTypeTrendyToy    SpBizType = 28 // 潮玩
	SpBizTypeSecondHandCar SpBizType = 29 // 二手车
	SpBizTypePetPlant     SpBizType = 30 // 宠植
	SpBizTypeGift         SpBizType = 31 // 工艺礼品
	SpBizTypeCarService   SpBizType = 33 // 汽车服务
	SpBizTypeOther        SpBizType = 99 // 其他
)

// FlashSaleType 闲鱼特卖类型枚举。
//
// 对应 Python: FlashSaleType
type FlashSaleType int

const (
	FlashSaleTypeLiQi        FlashSaleType = 1    // 临期
	FlashSaleTypeGuPin       FlashSaleType = 2    // 孤品
	FlashSaleTypeDuanMa      FlashSaleType = 3    // 断码
	FlashSaleTypeWeiXia      FlashSaleType = 4    // 微瑕
	FlashSaleTypeWeiHuo      FlashSaleType = 5    // 尾货
	FlashSaleTypeGuanFan     FlashSaleType = 6    // 官翻
	FlashSaleTypeQuanXin     FlashSaleType = 7    // 全新
	FlashSaleTypeFuDai       FlashSaleType = 8    // 福袋
	FlashSaleTypeOther       FlashSaleType = 99   // 其他
	FlashSaleTypeBrandWeiXia FlashSaleType = 2601 // 微瑕
	FlashSaleTypeBrandLiQi   FlashSaleType = 2602 // 临期
	FlashSaleTypeBrandQingCang FlashSaleType = 2603 // 清仓
	FlashSaleTypeBrandGuanFan FlashSaleType = 2604 // 官翻
)

// ProductStatus 管家商品状态枚举。
//
// 对应 Python: ProductStatus
type ProductStatus int

const (
	ProductStatusUnknown    ProductStatus = 0  // 默认值
	ProductStatus21         ProductStatus = 21 // 状态21
	ProductStatus22         ProductStatus = 22 // 状态22
	ProductStatus23         ProductStatus = 23 // 状态23
	ProductStatus31         ProductStatus = 31 // 状态31
	ProductStatus33         ProductStatus = 33 // 状态33
	ProductStatus36         ProductStatus = 36 // 状态36
	ProductStatusNegative1  ProductStatus = -1 // 状态-1
)

// SaleStatus 销售状态枚举。
//
// 对应 Python: SaleStatus
type SaleStatus int

const (
	SaleStatusUnknown           SaleStatus = 0 // 默认值
	SaleStatusPendingPublication SaleStatus = 1 // 待发布
	SaleStatusOnSale            SaleStatus = 2 // 销售中
	SaleStatusOffSale           SaleStatus = 3 // 已下架
)

// OrderStatus 订单状态枚举。
//
// 对应 Python: OrderStatus
type OrderStatus int

const (
	OrderStatusPendingPayment    OrderStatus = 11 // 待付款
	OrderStatusPendingShipment   OrderStatus = 12 // 待发货
	OrderStatusShipped           OrderStatus = 21 // 已发货
	OrderStatusTransactionSuccess OrderStatus = 22 // 交易成功
	OrderStatusRefunded          OrderStatus = 23 // 已退款
	OrderStatusTransactionClosed OrderStatus = 24 // 交易关闭
)

// RefundStatus 退款状态枚举。
//
// 对应 Python: RefundStatus
type RefundStatus int

const (
	RefundStatusNotApplied                   RefundStatus = 0 // 未申请退款
	RefundStatusPendingSellerApproval        RefundStatus = 1 // 待商家处理
	RefundStatusPendingBuyerReturn           RefundStatus = 2 // 待买家退货
	RefundStatusPendingSellerReceive         RefundStatus = 3 // 待商家收货
	RefundStatusRefundClosed                 RefundStatus = 4 // 退款关闭
	RefundStatusRefundSuccess                RefundStatus = 5 // 退款成功
	RefundStatusRefundRejected               RefundStatus = 6 // 已拒绝退款
	RefundStatusPendingReturnAddressConfirm  RefundStatus = 8 // 待确认退货地址
)
