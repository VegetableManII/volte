package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/VegetableManII/volte/config"
	"github.com/VegetableManII/volte/controller"
	. "github.com/VegetableManII/volte/modules"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	self      *controller.I_CscfEntity
	localhost string
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "I-CSCF")
	coreIn := make(chan *Package, 4)
	coreOutUp := make(chan *Package, 2)
	coreOutDown := make(chan *Package, 2)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	conn := CreateServer(localhost)
	go ReceiveMessage(ctx, conn, coreIn)
	go ProcessDownStreamData(ctx, coreOutDown)
	go ProcessUpStreamData(ctx, coreOutUp)
	// 开启IMS域的逻辑处理协程
	go self.CoreProcessor(ctx, coreIn, coreOutUp, coreOutDown)

	<-quit
	logger.Warn("[I-CSCF] i-cscf 功能实体退出...")
	cancel()
	logger.Warn("[I-CSCF] i-cscf 子协程退出完成...")
}

func init() {
	scscf := viper.GetString(config.Domain + ".s-cscf.host")
	localhost = viper.GetString(config.Domain + ".i-cscf.host")
	dns := viper.GetString(config.Domain + ".domain")
	logger.Info("配置文件读取成功", "")
	// 启动 ISCF 的UDP服务器
	self = new(controller.I_CscfEntity)
	self.Init("i-cscf."+dns, localhost)
	self.Points["SCSCF"] = scscf
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{SIPPROTOCAL, SipRequest}, self.SIPREQUESTF)
	self.Regist([2]byte{SIPPROTOCAL, SipResponse}, self.SIPRESPONSEF)
	self.Regist([2]byte{EPCPROTOCAL, MultiMediaAuthenticationAnswer}, self.MutimediaAuthorizationAnswerF)
}
