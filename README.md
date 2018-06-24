# 高管持股变动提醒

## 简介
定时检测高管持股变动情况，发现新的高管持股变动信息自动发送邮件进行提醒

## 安装
* 根据参考配置文件`stock_conf.example.toml`设定配置并保存到`stock_conf.example.toml`
* 安装数据库`postgresql`
* 执行程序
```bash
go get github.com/glymehrvrd/stock_manager_trade_notify
go build ./ && ./stock_manager_trade_notify
```

## tips
推荐使用139邮件作为收件邮箱，可以实现免费短信提醒