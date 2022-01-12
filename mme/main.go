package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"volte/common"
	. "volte/common"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	eNodeBConn, hssConn *net.UDPConn
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "MME")
	preParseC := make(chan *Msg, 2)
	// postParseC := make(chan *Msg, 2)
	quit := make(chan os.Signal, 6)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGSTOP)
	// 开启与eNodeB交互的协程
	// go common.ExchangeWithClient(ctx, mmeConn, preParseC, postParseC)
	go common.ExchangeWithClient(ctx, eNodeBConn, preParseC, preParseC) // debug
	// 开启与HSS交互的协程

	<-quit
	logger.Warn("[MME] mme 功能实体退出...")
	cancel()
	logger.Warn("[MME] mme 子协程退出完成...")
}

func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host := viper.GetString("EPC.mme.host")
	hssHost := viper.GetString("EPC.hss.host")
	logger.Info("配置文件读取成功", "")
	// 启动 MME 的UDP服务器
	eNodeBConn = common.InitServer(host)
	// 创建连接 HSS 的客户端
	hssConn = common.ConnectServer(hssHost)
}
