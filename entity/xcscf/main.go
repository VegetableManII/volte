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
	self    *controller.CscfEntity
	epsConn *net.UDPConn
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "CSCF")
	coreIC := make(chan *Package, 2)
	coreOC := make(chan *Package, 2)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// 开启与eps域交互的协程
	go TransaportWithClient(ctx, epsConn, coreIC, coreOC)
	// 开启IMS域的逻辑处理协程
	go self.CoreProcessor(ctx, coreIC, coreOC)

	<-quit
	logger.Warn("[X-CSCF] x-cscf 功能实体退出...")
	cancel()
	logger.Warn("[X-CSCF] x-cscf 子协程退出完成...")
}

func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host := viper.GetString("IMS.x-cscf.host")
	logger.Info("配置文件读取成功", "")
	// 启动 CSCF 的UDP服务器
	epsConn = InitServer(host)
	self.Init()
}
