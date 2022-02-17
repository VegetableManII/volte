package main

import (
	"context"
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
	self.Points["eNodeB"] = eNodeBhost
	self.Points["CSCF"] = cscfHost
	RegistRouter()
}

func RegistRouter() {
	self.Regist([2]byte{SIPPROTOCAL, SipRequest}, self.SIPREQUESTF)
	self.Regist([2]byte{SIPPROTOCAL, SipResponse}, self.SIPRESPONSEF)
}

func heartbeat(ctx context.Context, pgw *controller.PgwEntity, conn *net.UDPConn) {
	for {
		data := make([]byte, 2)
		_, ra, err := conn.ReadFromUDP(data)
		if err != nil {
			logger.Error("心跳探测接收失败 %v", err)
			return
		}
		conn.WriteToUDP([]byte{0x05, 0x20}, ra)
		pgw.UtranConn.Lock()
		pgw.UtranConn.RemoteAddr = ra
		pgw.UtranConn.Unlock()
	}
}
