/*
eNodeB主要功能：消息转发
根据不同的消息类型转发到EPC网络还是IMS网络
*/
package main

import (
	"context"
	"epc/common"
	. "epc/common"
	"log"
	"net"
	"time"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	loConn, mmeConn, pgwConn *net.UDPConn
	ueBroadcastAddr          *net.UDPAddr
	scanTime                 int
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "eNodeB")
	producer := make(chan *Msg, 2)
	consumer := make(chan *Msg, 2)
	// 开启广播工作消息
	go broadWorkingMessage(ctx, loConn, ueBroadcastAddr)
	// 开启与ue通信的协程
	go common.ExchangeWithClient(ctx, loConn, producer, consumer)
	// 开启与mme通信的协程

	// 开启与pgw通信的协程
	cancel()
	loConn.Close()
}

// 读取配置文件
func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host := viper.GetString("eNodeB.host")
	enodebBroadcastNet := viper.GetString("eNodeB.broadcast.net")
	scanTime = viper.GetInt("eNodeB.scan.time")
	logger.Info("配置文件读取成功", "")
	// 启动与ue连接的服务器
	loConn, ueBroadcastAddr = initUeServer(host, enodebBroadcastNet)
	// 作为客户端与epc网络连接

	// 创建于MME的UDP连接
	//mme := viper.GetString("EPC.mme")
	//mmeConn = connectEPC(mme)
	// TODO 创建于PGW的UDP连接
	//pgw := viper.GetString("EPC.pgw")
	//pgwConn = connectEPC(pgw)
}

// 与ue连接的UDP服务端
func initUeServer(host string, broadcast string) (*net.UDPConn, *net.UDPAddr) {
	la, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Panicln("eNodeB host配置解析失败", err)
	}
	ra, err := net.ResolveUDPAddr("udp4", broadcast)
	if err != nil {
		log.Panicln("eNodeB 广播地址配置解析失败", err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panicln("eNodeB host监听失败", err)
	}
	if err != nil {
		log.Panicln(err)
	}
	logger.Info("ue UDP广播服务器启动成功 [%v]", host)
	logger.Info("UDP广播子网 [%v]", broadcast)
	return conn, ra
}

// 广播基站工作消息
func broadWorkingMessage(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr) {
	for {
		n, err := conn.WriteToUDP([]byte("Broadcast to Ue"), remote)
		if err != nil {
			logger.Error("[%v] 广播开始工作消息失败...... %v", ctx.Value("Entity"), err)
		}
		time.Sleep(1 * time.Second)
		logger.Info("[%v] 广播工作消息...... [%v]", ctx.Value("Entity"), n)
	}
}
