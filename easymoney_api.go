package main

import (
	"net/http"
	"fmt"
	"time"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"encoding/json"
	"strings"
	"errors"
	"strconv"
)

const ManagerTradeDataUrl = "http://datainterface.eastmoney.com/EM_DataCenter/JS.aspx?type=GG&sty=GGC&p=1&ps=50&code=%s"

var client *http.Client

func init() {
	client = &http.Client{
		Timeout: 5 * time.Second,
	}
}

type ManagerTradeInfo struct {
	Name              string    `db:"name"`
	Code              string    `db:"code"`
	TradeDate         time.Time `db:"trade_date"`
	Trader            string    `db:"trader"`
	TradeCount        int       `db:"trade_count"`
	TransactionPrice  string    `db:"transaction_price"`
	TransactionReason string    `db:"transaction_reason"`
	TransactionAmount int       `db:"transaction_amount"`
}

func GetManagerTradeInfo(stockCode string) ([]ManagerTradeInfo, error) {
	var ret []ManagerTradeInfo

	req, _ := http.NewRequest("GET", fmt.Sprintf(ManagerTradeDataUrl, stockCode), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.117 Safari/537.36")

	rsp, err := client.Do(req)
	if err != nil {
		e := "get manager trade info failed"
		logrus.WithFields(logrus.Fields{"req": req, "err": err}).Error(e)
		return ret, errors.New(e)
	}
	logrus.WithFields(logrus.Fields{"req": req, "rsp": rsp}).Debug("get manager trade success")

	b, _ := ioutil.ReadAll(rsp.Body)
	if len(b) < 2 {
		e := "response length too small"
		logrus.WithField("body", string(b)).Error(e)
		return ret, errors.New(e)
	}

	var infos []string
	err = json.Unmarshal(b[1:len(b)-1], &infos)
	if err != nil {
		e := "unmarshal response failed"
		logrus.WithFields(logrus.Fields{"body": string(b), "err": err}).Error()
		return ret, errors.New(e)
	}

	for _, info := range infos {
		fields := strings.Split(info, ",")
		if len(fields) != 15 {
			e := "malformed info"
			logrus.WithField("info", info).Error(e)
			return ret, errors.New(e)
		}
		var tradeInfo ManagerTradeInfo
		tradeInfo.Name = fields[9]
		tradeInfo.Code = fields[2]
		tradeInfo.TradeDate, _ = time.ParseInLocation("2006-01-02", fields[5], time.Local)
		tradeInfo.Trader = fields[3]
		tradeInfo.TradeCount, _ = strconv.Atoi(fields[6])
		tradeInfo.TransactionPrice = fields[8]
		tradeInfo.TransactionReason = fields[12]
		tradeInfo.TransactionAmount, _ = strconv.Atoi(fields[13])
		ret = append(ret, tradeInfo)
	}
	return ret, nil
}
