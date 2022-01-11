/*
eNodeB主要功能：消息转发
根据不同的消息类型转发到EPC网络还是IMS网络
*/
package main

import (
	"context"
	. "epc/common"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
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
	quit := make(chan os.Signal, 6)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGSTOP)
	// 开启广播工作消息
	go broadWorkingMessage(ctx, loConn, ueBroadcastAddr, scanTime, []byte("Broadcast to Ue"))
	// 开启ue和mme的转发协程
	go EnodebProxyMessage(ctx, loConn, mmeConn)
	go broadMessageFromNet(ctx, mmeConn, loConn, ueBroadcastAddr)
	// 开启ue和pgw的转发协程
	// go EnodebProxyMessage(ctx, loConn, pgwConn)
	// go EnodebProxyMessage(ctx, pgwConn, loConn)
	<-quit
	logger.Warn("[eNodeB] eNodeB 功能实体退出...")
	cancel()
	logger.Warn("[eNodeB] eNodeB 资源释放完成...")

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
	mme := viper.GetString("EPC.mme.host")
	mmeConn = ConnectServer(mme)
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
// scan = 0, 广播网络侧消息
func broadWorkingMessage(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, scan int, msg []byte) {
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站工作广播协程退出...", ctx.Value("Entity"))
			return
		default:
			n, err := conn.WriteToUDP(msg, remote)
			if err != nil {
				logger.Error("[%v] 广播开始工作消息失败... %v", ctx.Value("Entity"), err)
			}
			if scan == 0 {
				logger.Info("[%v] 广播网络侧消息... [%v]", ctx.Value("Entity"), n)
				return
			}
			time.Sleep(time.Duration(scan) * time.Second)
			logger.Info("[%v] 广播工作消息... [%v]", ctx.Value("Entity"), n)
		}
	}
}

func broadMessageFromNet(ctx context.Context, from *net.UDPConn, to *net.UDPConn, baddr *net.UDPAddr) {
	data := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发广播网络侧消息协程退出...", ctx.Value("Entity"))
			return
		default:
			n, remote, err := from.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] 读取网络侧数据错误 %v", ctx.Value("Entity"), err)
			}
			logger.Info("[%v] 读取网络侧数据 %v", ctx.Value("Entity"), data[:n])
			if n != 0 && remote != nil {
				// 将收到的消息广播出去
				broadWorkingMessage(ctx, to, baddr, 0, data[:n])
			} else {
				logger.Info("[%v] Remote[%v] Len[%v]", ctx.Value("Entity"), remote, n)
				time.Sleep(2 * time.Second)
			}
		}
	}
}
