package main

import (
	"database/sql"
	//"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	///"net/url"
	"strconv"
	"strings"
	//"jasonstruct"
	"compress/gzip"
	"math"
	"os"
	"time"

	"github.com/bitly/go-simplejson"
	_ "github.com/go-sql-driver/mysql"
)

var SQL_CONNECT *sql.DB = nil
var sql_user string = ""
var sql_pwd string = ""
var sql_port int = 0

func yypGetDB() *sql.DB {

	if sql_user == "" || sql_pwd == "" || sql_port == 0 {
		return nil
	}

	//http://www.01happy.com/golang-mysql-demo/
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(localhost:%d)/zc_auto?charset=utf8", sql_user, sql_pwd, sql_port))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return db
}

func yypInitParam() {
	fi, err := os.Open("yyp.json")
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)
	//fmt.Println(string(fd))
	config_data, _ := simplejson.NewJson([]byte(fd))
	sql_user, _ = config_data.Get("sql_user").String()
	sql_pwd, _ = config_data.Get("sql_pwd").String()
	sql_port, _ = config_data.Get("sql_port").Int()
}

func yypGetOrderId() int {
	//fmt.Println("yypGetOrderId")
	/////////////////// request data persecond
	req, err := http.NewRequest("GET", "http://wp.100bei.com/nhpme/auth/order/queryCurrentOrder.do", nil)
	if err != nil {
		return 0
	}

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}

	defer resp.Body.Close()

	if data, err := ioutil.ReadAll(resp.Body); err == nil {
		//fmt.Printf("%s\n", data)
		js_data, _ := simplejson.NewJson([]byte(data))
		requestId, js_err := js_data.Get("data").GetIndex(0).Int()
		if js_err != nil {
			//fmt.Println(js_err)
			return 0
		}
		return requestId
	}
	return 0
}

func yypIsTradeTime() bool {
	// 每周一8:00 到 周六 4:00 每日 4:00到6:00休市
	nowtime := time.Now()
	var nHour int
	var nMin int
	var nSec int
	nHour, nMin, nSec = nowtime.Clock()
	var nTodaySec = nHour*3600 + nMin*60 + nSec
	if nowtime.Weekday() >= time.Tuesday && nowtime.Weekday() <= time.Friday {
		if nTodaySec <= TRADE_END_BEGIN || nTodaySec >= TRADE_END_END {
			return true
		}
	} else if nowtime.Weekday() == time.Monday {
		if nTodaySec >= TRADE_START_MONDAY {
			return true
		}
	} else if nowtime.Weekday() == time.Saturday {
		if nTodaySec <= TRADE_END_BEGIN {
			return true
		}
	}

	return false
}

func yypAddHeaderClient(header *http.Header) {
	header.Add("Accept", "application/json, text/javascript")
	header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 5.1.1; YQ601 Build/LMY47V; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/53.0.2785.49 Mobile MQQBrowser/6.2 TBS/043024 Safari/537.36 MicroMessenger/6.5.3.980 NetType/WIFI Language/zh_CN")
	header.Add("X-Requested-With", "XMLHttpRequest")
	header.Add("Content-Type", "application/x-www-form-urlencoded")
	header.Add("Accept-Encoding", "gzip, deflate")
	header.Add("Accept-Language", "zh-CN,en-US;q=0.8")
	//req.Header.Add("Cookie", "last_cu_price=30468.00; last_ag_price=4128.00;") //PHPSESSID=8723fvfca7vcl9oe5h6de8c901; wxid=oNw45t1VGZLu178NBp1XhssDIKoM; yh_User=afabN1Ph6XxU%2BYgiMnWPOVexRvnkYJ6944qj313GWx7mkoBEKUuOk5k%2B9tk%7Cbff962373ec2a19087abb4745a30a274")

	//	var tAgPrice http.Cookie = http.Cookie{}
	//	tAgPrice.Name = "last_ag_price"
	//	tAgPrice.Value = "4128.00"
	//	req.AddCookie(&tAgPrice)
	//	var tCuPrice http.Cookie = http.Cookie{}
	//	tCuPrice.Name = "last_cu_price"
	//	tCuPrice.Value = "30468.00"
	//	req.AddCookie(&tCuPrice)
}

func yypAddHeaderCookie(header *http.Header) {
	if SQL_CONNECT == nil {
		return
	}

	rows, err := SQL_CONNECT.Query("select id,name,val from zc_param")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer rows.Close()

	strParam := ""
	for rows.Next() {
		var id int
		var name string
		var val string
		err := rows.Scan(&id, &name, &val)
		if err != nil {
			fmt.Println(err)
			continue
		}

		strParam = fmt.Sprintf("%s=%s", name, val)
		header.Add("Cookie", strParam)
	}
}

func yypCreateOrder(iDirection int, buyPrice int, buyCount int) bool {
	rid := yypGetRequestId()
	if rid == "" {
		return false
	}

	requri := fmt.Sprintf("http://wp.100bei.com/nhpme/auth/order/createOrder.do?productId=46&type=%d&count=%d&useCoupon=0&couponCount=0&couponId=7&requestId=%s&toplimit=0&bottomlimit=0.3&moreCouponRule=0&contract=HGAG&price=%d&couponName=8%%E5%%85%%83&fee=0.6",
		iDirection, buyCount, rid, buyPrice)
	//fmt.Println(requri)
	req, err := http.NewRequest("GET", requri, nil)

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false
	}

	defer resp.Body.Close()

	return true
}

func yypCloseOrder(orderId int) bool {
	rid := yypGetRequestId()
	if rid == "" {
		return false
	}

	requri := fmt.Sprintf("http://wp.100bei.com/nhpme/auth/order/closeOrder.do?orderId=%d&orderType=1&contract=HGAG&requestId=%s",
		orderId, rid)
	//fmt.Println(requri)
	req, err := http.NewRequest("GET", requri, nil)

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false
	}

	defer resp.Body.Close()

	return true
}

func yypRequestBalance() float32 {
	req, err := http.NewRequest("GET", "http://wp.100bei.com/nhpme/auth/customer/customerInfo.do", nil)
	if err != nil {
		fmt.Println(err)
		return 0.0
	}

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return 0.0
	}

	defer resp.Body.Close()

	if data, err := ioutil.ReadAll(resp.Body); err == nil {
		//fmt.Printf("%s\n", data)

		js_data, _ := simplejson.NewJson([]byte(data))
		//fmt.Println(js_data)
		my_balance, js_err := js_data.Get("data").Get("balance").Float64()
		if js_err == nil {
			//fmt.Println(str_real_Price)
			return (float32)(my_balance)
		}
		return 0.0
	}

	return 0.0
}

func yypGetOrderDetail() (*OrderData, error) {
	req, err := http.NewRequest("GET", "http://wp.100bei.com/nhpme/auth/order/currentOrder.do?queryDb=1", nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer resp.Body.Close()

	if data, err := ioutil.ReadAll(resp.Body); err == nil {
		//fmt.Printf("%s\n", data)

		js_data, _ := simplejson.NewJson([]byte(data))
		//fmt.Println(js_data)
		var js_err error
		var buyDir int
		var buyPrice float64
		var orderId int
		var price int
		var count int
		var contract string
		var buyTime string
		buyDir, js_err = js_data.Get("data").GetIndex(0).Get("buyDirection").Int()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		buyPrice, js_err = js_data.Get("data").GetIndex(0).Get("buyPrice").Float64()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		orderId, js_err = js_data.Get("data").GetIndex(0).Get("orderId").Int()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		price, js_err = js_data.Get("data").GetIndex(0).Get("price").Int()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		count, js_err = js_data.Get("data").GetIndex(0).Get("count").Int()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		contract, js_err = js_data.Get("data").GetIndex(0).Get("contract").String()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		buyTime, js_err = js_data.Get("data").GetIndex(0).Get("addTime").String()
		if js_err != nil {
			fmt.Println(js_err)
			return nil, js_err
		}
		myOrder := &OrderData{
			OrderId:      orderId,
			BuyPrice:     (float32)(buyPrice),
			BuyDirection: buyDir,
			Price:        price,
			Count:        count,
			Contract:     contract,
			BuyTime:      buyTime,
			MaxPrice:     0,
			MinPrice:     0,
		}

		return myOrder, nil
	}

	return nil, err
}

func yypGetRequestId() string {
	req, err := http.NewRequest("GET", "http://wp.100bei.com/nhpme/auth/order/sendOrder.do", nil)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	defer resp.Body.Close()

	if data, err := ioutil.ReadAll(resp.Body); err == nil {
		//fmt.Printf("%s\n", data)
		js_data, _ := simplejson.NewJson([]byte(data))
		requestId, js_err := js_data.Get("data").Get("requestId").String()
		if js_err != nil {
			fmt.Println(js_err)
			return ""
		}
		return requestId
	}

	return ""
}

func queryRealTimePrice() float32 {
	//%%2CJO_9753%%2CJO_42757%%2CJO_38493%%2CJO_111%%2CJO_9833%%2CJO_64084%%2CJO_357%%2CJO_429%%2CJO_61810%%2CJO_61811%%2CJO_61813%%2CJO_61815%%2CJO_61817%%2CJO_9754%%2CJO_76%%2CJO_74%%2CJO_38496%%2CJO_38497%%2CJO_38498%%2CJO_60376%%2CJO_9834%%2CJO_9835%%2CJO_42758%%2CJO_42761
	reqUri := fmt.Sprintf("http://api.jijinhao.com/realtime/quotejs.htm?codes=JO_63737&currentPage=1&pageSize=6&_=%d", int32(time.Now().Unix()))
	req, err := http.NewRequest("GET", reqUri, nil)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	req.Header.Add("Accept", "application/javascript, */*;q=0.8")
	req.Header.Add("Referer", "http://www.cngold.org/img_date/ygy.html")
	req.Header.Add("Accept-Language", "zh-Hans-CN,zh-Hans;q=0.5")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64; Trident/7.0; Touch; rv:11.0) like Gecko")
	req.Header.Add("Accept-Encoding", "gzip, deflate")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	defer resp.Body.Close()

	//resp.Body = &gzipReader{body: resp.Body}
	if resp.StatusCode == 200 {
		var data string
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, _ := gzip.NewReader(resp.Body)
			for {
				buf := make([]byte, 1024)
				n, err := reader.Read(buf)

				if err != nil && err != io.EOF {
					panic(err)
				}

				if n == 0 {
					break
				}
				data += string(buf)
			}
		default:
			data_body, _ := ioutil.ReadAll(resp.Body)
			data = string(data_body)

		}

		// "var quot_str ="

		//fmt.Println(data)
		if data != "" {
			idx := strings.Index(data, "[")
			lastIdx := strings.LastIndex(data, "]")
			if idx != -1 {
				data = data[idx+1 : lastIdx] //去掉数组标志 []
				//fmt.Println(data)
				js_data, _ := simplejson.NewJson([]byte(data))
				//fmt.Println(js_data)
				str_real_Price, js_err := js_data.Get("data").GetIndex(0).Get("quote").Get("q63").String()
				if js_err == nil {
					//fmt.Println(str_real_Price)
					real_price, _ := strconv.ParseFloat(str_real_Price, 32)
					return (float32)(real_price + 3)
				}
				return 0
			}

		}
	}

	return 0
}

func yypInsertDataToDB(agPrice float32) {
	if SQL_CONNECT == nil {
		fmt.Println("mysql unconnected!")
		return
	}

	if agPrice != 0 {
		rows, err := SQL_CONNECT.Query("INSERT INTO price_ag_new VALUES(NOW(1),?)", agPrice)
		if err != nil {
			fmt.Println(err)
		}
		defer rows.Close()
	}
}

func yypStrategy(curPrice float32, ppOrder **OrderData) {
	//yypCreateOrder(BUY_DOWN, 8, 1)
	//yypCloseOrder("28883497")
	// 如果当前有订单
	// else 当前没有订单
	// todo 这里要考虑短期剧烈导致的强制平仓的情况

	nTmpOrder := yypGetOrderId()
	// 容错处理，防止出现订单状态不一致的情况
	if nTmpOrder == 0 && *ppOrder != nil {
		*ppOrder = nil
	} else if *ppOrder == nil && nTmpOrder != 0 {
		*ppOrder, _ = yypGetOrderDetail()
		if *ppOrder == nil {
			fmt.Println("yypStrategy getorderdatail failed")
			return
		}
		the_time, err := time.ParseInLocation("2006-01-02 15:04:05", (*ppOrder).BuyTime, time.Local)
		if err == nil {
			var nBuyTime uint = (uint)(the_time.Unix())
			(*ppOrder).MaxPrice = yypGetPriceFromDate(nBuyTime, "MAX")
			(*ppOrder).MinPrice = yypGetPriceFromDate(nBuyTime, "MIN")
		} else {
			fmt.Println(err)
		}
	}

	var myOrder *OrderData = *ppOrder
	if myOrder != nil {
		if myOrder.MaxPrice < curPrice {
			myOrder.MaxPrice = curPrice
		}
		if myOrder.MinPrice > curPrice {
			myOrder.MinPrice = curPrice
		}

		var bClosing bool = false
		fMaxProfit := yypGetHistoryMaxProfit(myOrder)
		fMaxProfitRate := (float32)(math.Abs((float64)(fMaxProfit))) / (float32)(myOrder.Price)
		fCurProfit := yypGetCurProfit(curPrice, myOrder)
		fCurProfitRate := (float32)(math.Abs((float64)(fCurProfit))) / (float32)(myOrder.Price)
		fDiffRate := (float32)(math.Abs((float64)(fMaxProfit-fCurProfit))) / (float32)(myOrder.Price)
		//亏损20 % 平仓
		if fCurProfit < 0 && fCurProfitRate >= 0.20 {
			bClosing = true
		}

		if !bClosing &&
			fMaxProfit > 0 &&
			fMaxProfitRate >= 0.10 &&
			fMaxProfit > fCurProfit &&
			fDiffRate > 0.15 {
			bClosing = true
		}

		if bClosing {
			yypCloseOrder(myOrder.OrderId)
			fmt.Println(fmt.Sprintf("%s, Price: %f, Close", time.Now().Format("2006-01-02 15:04:05"), curPrice))
			nTmpOrder := yypGetOrderId()
			if nTmpOrder != 0 {
				fmt.Println("CloseOrderFail!")
			} else {
				// 清空订单信息
				*ppOrder = nil
			}
		}
	} else {
		// 超过2日最高价，或者跌破2日最低价，开仓
		var nTargetTime uint = yypGetTargetTimeStamp()
		var fMax float32 = yypGetPriceFromDate(nTargetTime, "MAX")
		var fMin float32 = yypGetPriceFromDate(nTargetTime, "MIN")
		if (fMax - fMin) <= 30 {
			return
		}

		var buy_type int = BUY_NONE
		if curPrice >= fMax {
			buy_type = BUY_UP
		} else if curPrice <= fMin {
			buy_type = BUY_DOWN
		}

		if buy_type == BUY_NONE {
			return
		}

		if yypCanCreateOrder(AG_PRICE_8, 1) {
			yypCreateOrder(buy_type, AG_PRICE_8, 1)
			fmt.Println(fmt.Sprintf("%s, Price: %f, Buy", time.Now().Format("2006-01-02 15:04:05"), curPrice))
			*ppOrder, _ = yypGetOrderDetail()
			if *ppOrder != nil && (*ppOrder).OrderId != 0 {
				(*ppOrder).MaxPrice = (*ppOrder).BuyPrice
				(*ppOrder).MinPrice = (*ppOrder).BuyPrice
			}
		}
	}
}

func yypCanCreateOrder(price int, count int) bool {
	// 最后确认一次是否没有该产品的订单
	checkOrder, _ := yypGetOrderDetail()
	if checkOrder != nil && checkOrder.OrderId != 0 {
		return false
	}

	// 总资金足够
	myBalance := yypRequestBalance()
	minMoney := ((float32)(price) + AG_FEE_8) * (float32)(count)
	if myBalance < minMoney {
		return false
	}

	return true
}

func yypGetTargetTimeStamp() uint {
	// 周1和周2向前取4天，其他时间取2天
	nowtime := time.Now()
	if nowtime.Weekday() == time.Monday || nowtime.Weekday() == time.Tuesday {
		return (uint)(nowtime.Unix() - ONE_DAY_SECOND*4)
	}
	return (uint)(nowtime.Unix() - ONE_DAY_SECOND*2)
}

func yypGetHistoryMaxProfit(myOrder *OrderData) float32 {
	var priceWave float32 = yypGetUnitWavePrice(AG, myOrder.Price)

	if myOrder.BuyDirection == BUY_UP {
		return (myOrder.MaxPrice - myOrder.BuyPrice) * priceWave
	} else if myOrder.BuyDirection == BUY_DOWN {
		return (myOrder.BuyPrice - myOrder.MinPrice) * priceWave
	} else {
		return 0
	}
	return 0
}

func yypGetCurProfit(curPrice float32, myOrder *OrderData) float32 {
	var priceWave float32 = yypGetUnitWavePrice(AG, myOrder.Price)
	if myOrder.BuyDirection == BUY_UP {
		return (curPrice - myOrder.BuyPrice) * priceWave
	} else if myOrder.BuyDirection == BUY_DOWN {
		return (myOrder.BuyPrice - curPrice) * priceWave
	} else {
		return 0
	}
	return 0
}

func yypGetUnitWavePrice(product int, price int) float32 {
	if product == AG {
		if price == AG_PRICE_8 {
			return 0.1
		}
	}
	return 0
}

func yypGetPriceFromDate(nTimeStamp uint, strFunc string) float32 {
	if SQL_CONNECT == nil {
		return 0
	}

	if strFunc != "MAX" && strFunc != "MIN" {
		return 0
	}

	rows, err := SQL_CONNECT.Query(fmt.Sprintf("select %s(price) from price_ag_new WHERE log_time>FROM_UNIXTIME(%d)", strFunc, nTimeStamp))
	if err != nil {
		fmt.Println(err)
		return 0
	}

	defer rows.Close()

	for rows.Next() {
		var price float32
		err := rows.Scan(&price)
		if err != nil {
			fmt.Println(err)
			continue
		}

		return price
	}
	return 0
}

func main() {
	yypInitParam()

	SQL_CONNECT = yypGetDB()
	defer SQL_CONNECT.Close()

	var num int = 0
	myOrder, _ := yypGetOrderDetail()
	//	myOrder := &OrderData{
	//		OrderId:      "1111",
	//		BuyPrice:     3900,
	//		BuyDirection: 1,
	//		Price:        8,
	//		Count:        1,
	//		Contract:     "XAG1",
	//		BuyTime:      "2017-03-03 12:00:00",
	//		MaxPrice:     10,
	//		MinPrice:     10,
	//	}
	if myOrder != nil && myOrder.OrderId != 0 && myOrder.BuyTime != "" {
		the_time, err := time.ParseInLocation("2006-01-02 15:04:05", myOrder.BuyTime, time.Local)
		if err == nil {
			var nBuyTime uint = (uint)(the_time.Unix())
			myOrder.MaxPrice = yypGetPriceFromDate(nBuyTime, "MAX")
			myOrder.MinPrice = yypGetPriceFromDate(nBuyTime, "MIN")
		} else {
			fmt.Println(err)
		}
	}

	t1 := time.NewTimer(time.Millisecond * 500)

	for {
		select {
		case <-t1.C:
			if yypIsTradeTime() {
				curPrice := queryRealTimePrice()
				//fmt.Println(curPrice)

				yypStrategy(curPrice, &myOrder)

				yypInsertDataToDB(curPrice)
			}

			num = num + 1
			num = num % 5
			if num == 0 {
				// 5秒请求一次，用于保持session
				yypGetOrderId()
			}

			t1.Reset(time.Millisecond * 1000)
		}
	}
}
