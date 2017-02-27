package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	///"net/url"
	"strconv"
	"strings"
	//"jasonstruct"
	"compress/gzip"
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
	fmt.Println(string(fd))
	config_data, _ := simplejson.NewJson([]byte(fd))
	sql_user, _ = config_data.Get("sql_user").String()
	sql_pwd, _ = config_data.Get("sql_pwd").String()
	sql_port, _ = config_data.Get("sql_port").Int()
}

func yypGetOrderId() string {
	//fmt.Println("yypGetOrderId")
	/////////////////// request data persecond
	req, err := http.NewRequest("GET", "http://wp.100bei.com/nhpme/auth/order/queryCurrentOrder.do", nil)
	if err != nil {
		return ""
	}

	yypAddHeaderClient(&req.Header)
	yypAddHeaderCookie(&req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}

	defer resp.Body.Close()

	if data, err := ioutil.ReadAll(resp.Body); err == nil {
		//fmt.Printf("%s\n", data)
		var s QueryOrderDo
		json.Unmarshal([]byte(data), &s)
		//fmt.Println(s)

		return s.Data
	}
	return ""
}

func yypIsTradeTime() bool {
	// 每周一8:00 到 周六 4:00 每日 4:00到7:00休市
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
		if nTodaySec >= TRADE_END_END+3600 {
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

	requri := fmt.Sprintf("http://wp.100bei.com/nhpme/auth/order/createOrder.do?productId=46&type=%d&count=%d&useCoupon=0&couponCount=0&couponId=7&requestId=%s&toplimit=0&bottomlimit=0.5&moreCouponRule=0&contract=XAG1&price=%d&couponName=8%%E5%%85%%83&fee=0.6",
		iDirection, buyCount, rid, buyPrice)
	fmt.Println(requri)
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

func yypCloseOrder(orderId string) bool {
	rid := yypGetRequestId()
	if rid == "" {
		return false
	}

	requri := fmt.Sprintf("http://wp.100bei.com/nhpme/auth/order/closeOrder.do?orderId=%s&orderType=1&contract=XAG1&requestId=%s",
		orderId, rid)
	fmt.Println(requri)
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
		var s ReceiveData
		json.Unmarshal([]byte(data), &s)
		fmt.Println(s)

		return s.Data.Balance
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

		var s ReceiveData
		json.Unmarshal([]byte(data), &s)
		//fmt.Println(s)
		myOrder := &OrderData{
			OrderId:      s.Data.OrderId,
			BuyPrice:     s.Data.BuyPrice,
			BuyDirection: s.Data.BuyDirection,
			Price:        s.Data.Price,
			Count:        s.Data.Count,
			Contract:     s.Data.Contract,
			BuyTime:      s.Data.AddTime,
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
		var s ReceiveData
		json.Unmarshal([]byte(data), &s)
		fmt.Println(s)

		return s.Data.RequestId
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

func yypStrategy(curPrice float32, myOrder *OrderData) {
	//yypCreateOrder(BUY_DOWN, 8, 1)
	//yypCloseOrder("28883497")
	// 如果当前有订单
	// else 当前没有订单
	if myOrder != nil {
		if myOrder.MaxPrice < curPrice {
			myOrder.MaxPrice = curPrice
		}
		if myOrder.MinPrice > curPrice {
			myOrder.MinPrice = curPrice
		}
	} else {

	}
}

func yypGetPriceFromDate(strDate string, bMax bool) float32 {
	if SQL_CONNECT == nil {
		return 0
	}

	var funcSQL string = "MIN"
	if bMax == true {
		funcSQL = "MAX"
	}

	rows, err := SQL_CONNECT.Query(fmt.Sprintf("select %s(price) from price_ag_new WHERE log_time>\"%s\"", funcSQL, strDate))
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

	//	var num int = 0
	//	myOrder, _ := yypGetOrderDetail()
	//	if myOrder != nil && myOrder.OrderId != "" && myOrder.BuyTime != "" {
	//		myOrder.MaxPrice = yypGetPriceFromDate(myOrder.BuyTime, true)
	//		myOrder.MinPrice = yypGetPriceFromDate(myOrder.BuyTime, false)
	//	}

	t1 := time.NewTimer(time.Millisecond * 500)

	for {
		select {
		case <-t1.C:
			if yypIsTradeTime() {
				curPrice := queryRealTimePrice()
				fmt.Println(curPrice)

				//yypStrategy(curPrice, myOrder)

				yypInsertDataToDB(curPrice)
			}
			//			} else {
			//				// 由于执行的是每日清仓原则，所以休市期清空所有的订单
			//				myOrder = nil
			//			}
			//			num = num + 1
			//			num = num % 5
			//			if num == 0 {
			//				// 5秒请求一次，用于保持session
			//				yypGetOrderId()
			//			}

			t1.Reset(time.Millisecond * 1000)
		}
	}
}
