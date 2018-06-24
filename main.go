package main

import (
	_ "github.com/lib/pq"
	"database/sql"
	"github.com/sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/go-gomail/gomail"
	"fmt"
	"html/template"
	"bytes"
	"strconv"
	"os"
	"github.com/robfig/cron"
	"time"
	"os/signal"
	"github.com/BurntSushi/toml"
)

var db *sqlx.DB
var gConfig *Config

func getNewerTradeInfo(code string, infos []ManagerTradeInfo) []ManagerTradeInfo {
	lastInfo := ManagerTradeInfo{}
	err := db.Get(&lastInfo, "select name, code, trade_date, trader, trade_count, transaction_price, transaction_reason, transaction_amount from manager_trade where code=$1 order by id desc limit 1", code)
	if err != nil {
		if err == sql.ErrNoRows {
			return infos
		}
		logrus.Errorf("query row failed, err[%s]", err.Error())
		return nil
	}
	if len(infos) == 0 || lastInfo.TradeDate.After(infos[0].TradeDate) {
		return nil
	}

	// TODO: use bisection
	for i, info := range infos {
		if lastInfo == info {
			return infos[0:i]
		}
	}
	return infos
}

func comma(v int) string {
	var s string
	var negative bool
	if v < 0 {
		negative = true
		v = -1 * v
	}
	for v > 1000 {
		remainder := v % 1000
		v = v / 1000
		s = "," + fmt.Sprintf("%03d", remainder) + s
	}
	s = strconv.Itoa(v) + s
	if negative {
		s = "-" + s
	}
	return s
}

func marshalMailBody(infos []ManagerTradeInfo) string {
	t := template.Must(template.New("mail.template").Funcs(template.FuncMap{
		"comma": comma,
	}).ParseFiles("mail.template"))

	var buf bytes.Buffer
	err := t.Execute(&buf, infos)
	if err != nil {
		logrus.Fatal(err)
	}
	return buf.String()
}

func sendMail(infos []ManagerTradeInfo) {
	if len(infos) == 0 {
		return
	}

	name := infos[0].Name

	m := gomail.NewMessage()
	m.SetAddressHeader("From", gConfig.Mail.From, "")
	m.SetHeader("To", // 收件人
		m.FormatAddress(gConfig.Mail.To, ""),
	)
	m.SetHeader("Subject", fmt.Sprintf("%s，高管持股变动提醒", name))
	m.SetBody("text/html", marshalMailBody(infos))

	d := gomail.NewDialer(gConfig.Mail.SMTP.Host, gConfig.Mail.SMTP.Port, gConfig.Mail.SMTP.User, gConfig.Mail.SMTP.Password)
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

type StockInfo struct {
	Name string `db:"name"`
	Code string `db:"code"`
}

type Config struct {
	CronSchema    string `toml:"cron_schema"`
	StockInterval int `toml:"stock_interval"`
	DB struct {
		User     string
		Password string
		Host     string
		Port     int
		Database string
	}
	Mail struct {
		From string
		To   string
		SMTP struct {
			Host     string
			Port     int
			User     string
			Password string
		}
	}
}

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(os.Stdout)

	var err error
	_, err = toml.DecodeFile("stock_conf.toml", &gConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("parsed config, config: %+v", gConfig)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", gConfig.DB.User, gConfig.DB.Password, gConfig.DB.Host, gConfig.DB.Port, gConfig.DB.Database)
	db, err = sqlx.Open("postgres", dsn)
	if err != nil {
		logrus.Fatalln("open postgres db failed, dsn[%s]", dsn)
	}
	db.Exec("set search_path to stock")

	c := cron.New()
	err = c.AddFunc(gConfig.CronSchema, func() {
		logrus.Infof("[%s] start check stock manager trading info", time.Now().Format("2006-01-02 15:04:05"))

		var stocks []StockInfo

		db.Select(&stocks, "select code from stared_stocks")

		for _, stock := range stocks {
			code := stock.Code
			logrus.Infof("checking stock code: %s", code)

			infos, err := GetManagerTradeInfo(code)
			if err != nil {
				logrus.Errorf("get manager info failed, code[%s] err[%s]", code, err)
				continue
			}

			infos = getNewerTradeInfo(code, infos)
			for i := len(infos) - 1; i >= 0; i-- {
				_, err := db.NamedExec(`insert into manager_trade (name, code, trade_date, trader,trade_count,transaction_price, transaction_reason,transaction_amount) values
 			(:name, :code, :trade_date, :trader, :trade_count, :transaction_price, :transaction_reason, :transaction_amount)`, &infos[i])
				if err != nil {
					logrus.Errorf("insert newer trade info into pgsql failed, err[%s]", err.Error())
				}
			}

			logrus.Infof("stock[%s], new info length[%d]", code, len(infos))

			sendMail(infos)

			time.Sleep(time.Second * time.Duration(gConfig.StockInterval))
		}
	})
	if err != nil {
		logrus.Fatal(err)
	}

	c.Start()

	logrus.Infof("start check stock manager trading info")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Block until a signal is received.
	s := <-sigChan
	fmt.Println("got signal:", s)
	c.Stop()
}
