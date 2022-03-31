package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/VegetableManII/volte/controller"
	. "github.com/VegetableManII/volte/modules"

	"github.com/wonderivan/logger"
)

var (
	self      *controller.HssEntity
	localhost string
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "HSS")
	coreIn := make(chan *Package, 4)      // 原生数据输入核心处理器
	coreOutUp := make(chan *Package, 2)   // 核心处理器解析后的数据输出上行结果
	coreOutDown := make(chan *Package, 2) // 核心处理器解析后的数据输出下行结果
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	conn := CreateServer(localhost)
	go ReceiveMessage(ctx, conn, coreIn)
	go ProcessDownStreamData(ctx, coreOutDown)
	go ProcessUpStreamData(ctx, coreOutUp)

	go self.CoreProcessor(ctx, coreIn, coreOutUp, coreOutDown)

	<-quit
	logger.Warn("[HSS] hss 功能实体退出...")
	cancel()
	logger.Warn("[HSS] hss 子协程退出完成...")
}

/*
# 归属地查询服务器
hss:
  host: 127.0.0.1:7777
# 数据库配置信息
mysql:
  host: "root:@tcp(127.0.0.1:3306)/volte?charset=utf8&parseTime=True&loc=Local"
*/

func init() {
	localhost = "127.0.0.1:7777"
	self = new(controller.HssEntity)
	self.Init("root:@tcp(127.0.0.1:3306)/volte?charset=utf8&parseTime=True&loc=Local")
	self.Points["HBICSCF"] = "127.0.0.1:54322"
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{EPCPROTOCAL, MultiMediaAuthenticationRequest}, self.MultimediaAuthorizationRequestF)
}
