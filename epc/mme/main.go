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
	"time"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	mmeConn, hssConn *net.UDPConn
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "MME")
	producer := make(chan *Msg, 2)
	consumer := make(chan *Msg, 2)
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGSTOP)
	// 开启与eNodeB交互的协程
	go common.ExchangeWithClient(ctx, mmeConn, producer, consumer)
	// 开启与HSS交互的协程

	<-quit
	logger.Warn("[MME] mme 功能实体退出...")
	cancel()
	err1 := mmeConn.Close()
	err2 := hssConn.Close()
	if err1 != nil || err2 != nil {
		logger.Fatal("[MME] mme socket资源关闭错误 mmeSock:%v hssSock:%v", err1, err2)
	}
	time.Sleep(2 * time.Second)
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
	mmeConn = common.InitServer(host)
	// 创建连接 HSS 的客户端
	hssConn = common.ConnectEPC(hssHost)
}
