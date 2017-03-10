package main

// JASON数据结构
//type RequestDatas struct {
//	RequestId    string  // 下订单的请求ID
//	Balance      float32 // 总资金
//	OrderId      string  // 订单ID
//	BuyPrice     float32 // 订单购买价
//	BuyDirection int     // 开仓方向
//	Count        int     // 购买份数
//	Price        int     // 订单单位额度
//	Contract     string  // 订单产品名称 "XAG1"
//	AddTime      string  // 下单时间
//}

//type ReceiveData struct {
//	Data RequestDatas
//}

//type QueryOrderDo struct {
//	Data string
//}

// 订单结构体
type OrderData struct {
	OrderId      int
	BuyPrice     float32 //订单购买价
	BuyDirection int
	Price        int //订单单位额度
	Count        int
	Contract     string
	BuyTime      string //下单时间
	MaxPrice     float32
	MinPrice     float32
}
