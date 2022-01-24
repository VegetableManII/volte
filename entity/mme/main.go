package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	. "github.com/VegetableManII/volte/common"
	"github.com/VegetableManII/volte/controller"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	self      *controller.MmeEntity
	localHost string
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "MME")
	coreIn := make(chan *Package, 4)
	coreOutUp := make(chan *Package, 2)
	coreOutDown := make(chan *Package, 2)
	quit := make(chan os.Signal, 6)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go ReceiveClientMessage(ctx, localHost, coreIn)
	go ProcessDownStreamData(ctx, coreOutDown)
	go ProcessUpStreamData(ctx, coreOutUp)
	// 开启逻辑处理协程
	go self.CoreProcessor(ctx, coreIn, coreOutUp, coreOutDown)

	<-quit
	logger.Warn("[MME] mme 功能实体退出...")
	cancel()
	logger.Warn("[MME] mme 子协程退出完成...")
}

func init() {
	localHost = viper.GetString("EPC.mme.host")
	hssHost := viper.GetString("HSS.host")
	eNodeBhost := viper.GetString("EPC.eNodeB.host")
	logger.Info("配置文件读取成功", "")
	// 创建自身逻辑实体
	self = new(controller.MmeEntity)
	self.Init()
	self.Points["HSS"] = hssHost
	self.Points["eNodeB"] = eNodeBhost
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{EPCPROTOCAL, AttachRequest}, self.AttachRequestF)
	self.Regist([2]byte{EPCPROTOCAL, AuthenticationInformatResponse}, self.AuthenticationInformatResponseF)
	self.Regist([2]byte{EPCPROTOCAL, AuthenticationResponse}, self.AuthenticationResponseF)
}
