package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	. "github.com/VegetableManII/volte/common"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	loConn                   *net.UDPConn
	ueBroadcastAddr          *net.UDPAddr
	scanTime                 int
	TAI                      int
	lohost, mmehost, pgwhost string
	ues                      map[uint32]struct{}
	mu                       sync.Mutex
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "eNodeB")
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sysinfo := fmt.Sprintf("TAI=%d", TAI)
	// 开启广播工作消息
	go downLinkMessage(ctx, loConn, ueBroadcastAddr, scanTime, []byte(sysinfo))
	// 接收用户随机接入消息

	// 开启ue和mme/pgw的转发协程
	go enodebProxyMessage(ctx, loConn, mmehost, pgwhost) // 转发用户上行数据
	// 开启ue的信令广播协程
	go broadMessageFromNet(ctx, lohost, loConn, ueBroadcastAddr)
	<-quit
	logger.Warn("[eNodeB] eNodeB 功能实体退出...")
	cancel()
	logger.Warn("[eNodeB] eNodeB 资源释放完成...")

}

// 读取配置文件
func init() {
	hostPort := viper.GetInt("eNodeB.host.port")
	enodebBroadcastPort := viper.GetInt("eNodeB.broadcast.port")
	scanTime = viper.GetInt("eNodeB.scan.time")
	TAI = viper.GetInt("eNodeB.TAI")
	// 启动与ue连接的服务器
	loConn, ueBroadcastAddr = initUeServer(hostPort, enodebBroadcastPort)
	lohost = viper.GetString("EPC.eNodeB.host")
	// 创建于MME的UDP连接
	mmehost = viper.GetString("EPC.mme.host")
	// 创建于PGW的UDP连接
	pgwhost = viper.GetString("EPC.pgw.host")
	logger.Info("配置文件读取成功", "")
}

// 与ue连接的UDP服务端
func initUeServer(port int, bport int) (*net.UDPConn, *net.UDPAddr) {
	ipnet, err := getLocalInternelIPNet()
	if err != nil {
		log.Panicln("获取本地IP地址失败", err)
	}
	host := ipnet.IP.To4().String() + ":" + strconv.Itoa(port)
	la, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Panicln("eNodeB host配置解析失败", err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panicln("eNodeB host监听失败", err)
	}
	bip, err := lastAddr(ipnet)
	if err != nil {
		log.Panicln("获取本地广播地址失败", err)
	}
	broadcast := bip.String() + ":" + strconv.Itoa(bport)
	ra, err := net.ResolveUDPAddr("udp4", broadcast)
	if err != nil {
		log.Panicln("eNodeB 广播地址配置解析失败", err)
	}

	logger.Info("ue UDP广播服务器启动成功 [%v]", la)
	logger.Info("UDP广播子网 [%v]", ra)
	return conn, ra
}

// 主要用于基站实现消息的代理转发, 将ue消息转发至网络侧
func enodebProxyMessage(ctx context.Context, src *net.UDPConn, mme, pgw string) {
	var n int
	var err error
	var raddr *net.UDPAddr

	var RandomAccess uint32 = 0xFFFFFFFF
	f := func(data []byte) (num uint32) {
		var i uint8 = 0
		for offset, v := range data {
			i = uint8(v)
			num += uint32(i << uint8(offset))
		}
		return
	}
	for {
		data := make([]byte, 1024)
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发协程退出...", ctx.Value("Entity"))
			return
		default:
			n, raddr, err = src.ReadFromUDP(data)
			if err != nil && n == 0 {
				logger.Error("[%v] 基站接收消息失败 %v %v", ctx.Value("Entity"), n, err)
			}
			logger.Info("[%v] 基站接收消息%v %v(%v byte)", ctx.Value("Entity"), data[0:4], string(data[4:n]), n)
			// 如果用户随机接入则响应给用户分配的唯一ID
			rand := f(data[0:4])

			if rand == RandomAccess {
				sum := fnv.New32().Sum([]byte(raddr.String()))
				ueid := f(sum)
				mu.Lock()
				ues[uint32(ueid)] = struct{}{}
				mu.Unlock()
				downLinkMessage(ctx, src, raddr, -1, sum)
				continue
			}

			if data[4] == EPCPROTOCAL {
				err = UpLinkTransportEnb(ctx, mme, data[:n])
				if err != nil {
					logger.Error("[%v] 基站转发消息失败[to mme] %v %v", ctx.Value("Entity"), n, err)
				}
				logger.Info("[%v] 基站转发消息[to mme] %v", ctx.Value("Entity"), string(data[4:]))
			} else {
				err = UpLinkTransportEnb(ctx, pgw, data[:n])
				if err != nil {
					logger.Error("[%v] 基站转发消息失败[to pgw] %v %v", ctx.Value("Entity"), n, err)
				}
				logger.Info("[%v] 基站转发消息[to pgw] %v", ctx.Value("Entity"), string(data))
			}
		}
	}
}

// 广播基站工作消息
// scan = 0, 广播网络侧消息
// scan = >0, 间断广播工作消息让UE捕获
// scan = -1, 此时remote为具体的ue，为端到端发送
func downLinkMessage(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, scan int, msg []byte) {
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
			if scan <= 0 {
				return
			}
			time.Sleep(time.Duration(scan) * time.Second)
			logger.Info("[%v] 广播工作消息... [%v]", ctx.Value("Entity"), n)
		}
	}
}

func broadMessageFromNet(ctx context.Context, host string, bconn *net.UDPConn, baddr *net.UDPAddr) {
	lo, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		logger.Fatal("解析地址失败 %v", err)
	}
	logger.Info("服务监听启动成功 %v", lo.String())
	for {
		conn, err := net.ListenUDP("udp4", lo)
		if err != nil {
			log.Panicln("udp server 监听失败", err)
		}
		logger.Info("服务器启动成功[%v]", lo)
		data := make([]byte, 1024)
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发广播网络侧消息协程退出...", ctx.Value("Entity"))
			return
		default:
			n, remote, err := conn.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] 读取网络侧数据错误 %v", ctx.Value("Entity"), err)
			}
			if n != 0 && remote != nil {
				// 将收到的消息广播出去
				downLinkMessage(ctx, bconn, baddr, 0, data[:n])
			} else {
				logger.Info("[%v] Remote[%v] Len[%v]", ctx.Value("Entity"), remote, n)
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func getLocalInternelIPNet() (*net.IPNet, error) {
	if net.FlagUp != 1 {
		return nil, errors.New("ErrNoNet")
	}
	ifs, e := net.Interfaces()
	if e != nil {
		return nil, e
	}
	for i := 0; i < len(ifs); i++ {
		addrs, e := ifs[i].Addrs()
		if e != nil {
			return nil, e
		}
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.To4().IsLoopback() && isLan(ipnet.IP.String()) {
				return ipnet, nil
			}
		}
	}
	return nil, errors.New("ErrNetInterfaceNotFound")
}

var LanIPSeg = [4]string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.1/8",
}

func isLan(s string) bool {
	if ip := net.ParseIP(s); ip != nil {
		for _, network := range LanIPSeg {
			_, subnet, _ := net.ParseCIDR(network)
			if subnet.Contains(ip) {
				return true
			}
		}
	}
	return false
}

func lastAddr(n *net.IPNet) (net.IP, error) {
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("ErrNoIPv6")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}
