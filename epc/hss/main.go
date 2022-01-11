package main

import (
	"context"
	"epc/common"
	. "epc/common"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	mmeConn *net.UDPConn
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "HSS")
	preParseC := make(chan *Msg, 2)
	// postParseC := make(chan *Msg, 2)
	quit := make(chan os.Signal, 6)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGSTOP)
	// 开启与mme交互的协程
	// go common.ExchangeWithClient(ctx, mmeConn, preParseC, postParseC)
	go common.ExchangeWithClient(ctx, mmeConn, preParseC, preParseC) // debug

	<-quit
	logger.Warn("[HSS] hss 功能实体退出...")
	cancel()
	logger.Warn("[HSS] hss 子协程退出完成...")
}

func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host := viper.GetString("EPC.hss.host")
	logger.Info("配置文件读取成功", "")
	// 启动 HSS 的UDP服务器
	mmeConn = common.InitServer(host)
	// 创建连接 HSS 的客户端
	// hssConn = common.ConnectEPC(hssHost)
}
