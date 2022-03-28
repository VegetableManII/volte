package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/VegetableManII/volte/controller"
	"github.com/VegetableManII/volte/entity"
	"github.com/VegetableManII/volte/modules"
	. "github.com/VegetableManII/volte/modules"

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
	coreIn := make(chan *modules.Package, 4) // 原生数据输入核心处理器
	coreOutUp := make(chan *Package, 2)      // 核心处理器解析后的数据输出上行结果
	coreOutDown := make(chan *Package, 2)    // 核心处理器解析后的数据输出下行结果
	quit := make(chan os.Signal)
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

func init() {
	localhost = viper.GetString("hss.host")
	icscf := viper.GetString(entity.Domain + ".i-cscf.host")
	scscf := viper.GetString(entity.Domain + ".s-cscf.host")

	db := viper.GetString("mysql.host")
	logger.Info("配置文件读取成功 HSS.host: %v i-cscf.host: %v s-cscf.host: %v", localhost, icscf, scscf)

	self = new(controller.HssEntity)
	self.Init(db)
	self.Points["ICSCF"] = icscf
	self.Points["SCSCF"] = scscf
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{EPCPROTOCAL, MultiMediaAuthenticationRequest}, self.MultimediaAuthorizationRequestF)
}
