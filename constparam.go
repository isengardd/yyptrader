package main

const (
	BUY_NONE int = iota
	BUY_DOWN     //买跌
	BUY_UP       //买涨
)

const (
	_  int = iota
	AG     //白银
	NE     //镍
)

var ONE_DAY_SECOND int64 = 86400

var TRADE_END_BEGIN int = 4*3600 - 10
var TRADE_END_END int = 7*3600 + 10

// 手续费率
var AG_FEE_8 float32 = 0.6

var AG_PRICE_8 int = 8
