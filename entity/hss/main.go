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
	quit := make(chan os.Signal, 6)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go ReceiveClientMessage(ctx, localhost, coreIn)
	go ProcessDownStreamData(ctx, coreOutDown)
	go ProcessUpStreamData(ctx, coreOutUp)

	go self.CoreProcessor(ctx, coreIn, coreOutUp, coreOutDown)

	<-quit
	logger.Warn("[HSS] hss 功能实体退出...")
	cancel()
	logger.Warn("[HSS] hss 子协程退出完成...")
}

func init() {
	localhost = viper.GetString("HSS.host")
	mme := viper.GetString("EPC.mme.host")
	icscf := viper.GetString("IMS.i-cscf.host")
	scscf := viper.GetString("IMS.s-cscf.host")

	dbhost := viper.GetString("mysql.host")
	logger.Info("配置文件读取成功 HSS.host: %v EPC.mme.host: %v IMS.i-cscf.host: %v IMS.s-cscf.host: %v", localhost, mme, icscf, scscf)

	self = new(controller.HssEntity)
	self.Init(dbhost)
	self.Points["MME"] = mme
	self.Points["ICSCF"] = icscf
	self.Points["SCSCF"] = scscf
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{EPCPROTOCAL, AuthenticationInformatRequest}, self.AuthenticationInformatRequestF)
	self.Regist([2]byte{EPCPROTOCAL, UpdateLocationRequest}, self.UpdateLocationRequestF)
	self.Regist([2]byte{EPCPROTOCAL, MultiMediaAuthenticationRequest}, self.MultimediaAuthorizationRequestF)
}
