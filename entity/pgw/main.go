package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	. "github.com/VegetableManII/volte/common"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	laddr, raddr     *net.UDPAddr
	loConn, cscfConn *net.UDPConn
	cscf             string
)

/*
	读协程读消息->解析前管道->协议解析->解析后管道->写协程写消息
		readGoroutine --->> chan *Msg --->> parser --->> chan *Msg --->> writeGoroutine
*/
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "PGW")
	// coreIC := make(chan *Msg, 2)
	// coreOC := make(chan *Msg, 2)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		cscfConn = connectCSCF(ctx, laddr, raddr)
		if cscfConn == nil {
			logger.Warn("[PGW] 连接至CSCF失败, 正在进行重试...")
			continue
		}
		break
	}

	// 开启EPS域和IMS域的消息转发协程
	go PGWProxyMessage(ctx, loConn, cscfConn)
	go PGWProxyMessage(ctx, cscfConn, loConn)

	<-quit
	logger.Warn("[PGW] pgw 功能实体退出...")
	cancel()
	logger.Warn("[PGW] pgw 子协程退出完成...")
}

func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host := viper.GetString("EPS.pgw.host")
	cscf = viper.GetString("IMS.x-cscf.host")
	logger.Info("配置文件读取成功", "")
	// 启动 PGW 的UDP服务器
	loConn, laddr = initServer(host)
}

func initServer(h string) (*net.UDPConn, *net.UDPAddr) {
	la, err := net.ResolveUDPAddr("udp4", h)
	if err != nil {
		log.Fatal("PGW 启动监听失败")
	}
	conn, err := net.ListenUDP("udp", la)
	if err != nil {
		log.Fatal("PGW 启动监听失败")
	}
	return conn, la
}

func connectCSCF(ctx context.Context, laddr, raddr *net.UDPAddr) *net.UDPConn {
	conn, err := net.DialUDP("udp4", laddr, raddr)
	if err != nil {
		logger.Error("[%v] 连接至CSCF失败 %v", ctx.Value("Entity"), err)
		return nil
	}
	return conn
}
