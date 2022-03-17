package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/VegetableManII/volte/controller"
	. "github.com/VegetableManII/volte/modules"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	self                            *controller.PgwEntity
	localhost, eNodeBhost, cscfHost string
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "PGW")
	coreIn := make(chan *Package, 4)
	coreOutUp := make(chan *Package, 2)
	coreOutDown := make(chan *Package, 2)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	conn := CreateServer(localhost)
	go ReceiveClientMessage(ctx, conn, coreIn)

	go ProcessDownStreamData(ctx, coreOutDown)
	go ProcessUpStreamData(ctx, coreOutUp)

	go self.CoreProcessor(ctx, coreIn, coreOutUp, coreOutDown)

	<-quit
	logger.Warn("[PGW] pgw 功能实体退出...")
	cancel()
	logger.Warn("[PGW] pgw 子协程退出完成...")
}

func init() {
	localhost = viper.GetString("EPC.pgw.host")
	eNodeBhost = viper.GetString("EPC.eNodeB.host")
	cscfHost = viper.GetString("IMS.p-cscf.host")
	logger.Info("配置文件读取成功", "")
	self = new(controller.PgwEntity)
	self.Init()
	self.Points["CSCF"] = cscfHost
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{EPCPROTOCAL, AttachRequest}, self.AttachRequestF)
	self.Regist([2]byte{SIPPROTOCAL, SipRequest}, self.SIPREQUESTF)
	self.Regist([2]byte{SIPPROTOCAL, SipResponse}, self.SIPRESPONSEF)
}
