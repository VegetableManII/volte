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
	self      *controller.S_CscfEntity
	localhost string
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "S-CSCF")
	coreIn := make(chan *Package, 4)
	coreOutUp := make(chan *Package, 2)
	coreOutDown := make(chan *Package, 2)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 开启IMS域的逻辑处理协程
	go self.CoreProcessor(ctx, coreIn, coreOutUp, coreOutDown)

	conn := CreateServer(localhost)
	go ReceiveMessage(ctx, conn, coreIn)
	go ProcessDownStreamData(ctx, coreOutDown)
	go ProcessUpStreamData(ctx, coreOutUp)

	<-quit
	logger.Warn("[S-CSCF] s-cscf 功能实体退出...")
	cancel()
	logger.Warn("[S-CSCF] s-cscf 子协程退出完成...")
}

func init() {
	hss := viper.GetString("HSS.host")
	icscf := viper.GetString("IMS.i-cscf.host")
	localhost = viper.GetString("IMS.s-cscf.host")
	dns := viper.GetString("IMS.domain")
	logger.Info("配置文件读取成功", "")
	// 启动 CSCF 的UDP服务器
	self = new(controller.S_CscfEntity)
	self.Init(dns)
	self.Points["HSS"] = hss
	self.Points["ICSCF"] = icscf
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{SIPPROTOCAL, SipRequest}, self.SIPREQUESTF)
}
