package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	. "github.com/VegetableManII/volte/common"
	"github.com/VegetableManII/volte/controller"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	self    *controller.HssEntity
	mmeConn *net.UDPConn
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "HSS")
	coreIChan := make(chan *Msg, 2) // 原生数据输入
	coreOChan := make(chan *Msg, 2) // 解析后的数据输出
	quit := make(chan os.Signal, 6)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// 开启与mme交互的协程
	go TransaportWithClient(ctx, mmeConn, coreIChan, coreOChan)
	// go common.ExchangeWithClient(ctx, mmeConn, preParseC, preParseC) // debug

	go self.CoreProcessor(ctx, coreIChan, coreOChan)
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
	host := viper.GetString("EPS.hss.host")
	dbhost := viper.GetString("mysql.host")
	logger.Info("配置文件读取成功", "")
	// 启动 HSS 的UDP服务器
	mmeConn = InitServer(host)
	// 创建连接 HSS 的客户端
	// hssConn = common.ConnectEPS(hssHost)
	self = new(controller.HssEntity)
	self.Init(dbhost)
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{EPSPROTOCAL, AuthenticationInformatRequest}, self.AuthenticationInformatRequestF)
	self.Regist([2]byte{EPSPROTOCAL, UpdateLocationRequest}, self.UpdateLocationRequestF)
}
