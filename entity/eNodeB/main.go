package main

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/VegetableManII/volte/modules"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

type CoreNetConnection struct {
	PgwAddr   string
	PgwConn   net.Conn
	beatheart int // 心跳时间间隔
}

var (
	bConn       *net.UDPConn
	bAddr       *net.UDPAddr
	sTime       int
	NetSideConn *CoreNetConnection
	TAI         string
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "eNodeB")
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// 开启广播工作消息，不区分ue
	go working(ctx, bConn, bAddr, sTime, []byte(TAI))

	// 建立ue与核心网的通信隧道
	go tunneling(ctx, NetSideConn, bConn, bAddr)
	<-quit
	logger.Warn("[eNodeB] eNodeB 功能实体退出...")
	cancel()
	logger.Warn("[eNodeB] eNodeB 资源释放完成...")

}

// 读取配置文件
func init() {
	sport := viper.GetInt("eNodeB.server.port")
	bcPort := viper.GetInt("eNodeB.broadcast.port")
	sTime = viper.GetInt("eNodeB.scan.time")
	TAI = viper.GetString("eNodeB.TAI")

	// 启动与ue连接的服务器
	bConn, bAddr = initAPServer(sport, bcPort)
	NetSideConn = new(CoreNetConnection)
	// 创建于PGW的UDP连接
	NetSideConn.PgwAddr = viper.GetString("EPC.pgw.host")
	NetSideConn.beatheart = viper.GetInt("eNodeB.beatheart.time")
	logger.Info("配置文件读取成功", "")
}

// 与ue连接的UDP服务端
func initAPServer(port int, bport int) (*net.UDPConn, *net.UDPAddr) {
	localIP, err := getLocalLanIP()
	if err != nil {
		log.Panicln("获取本地IP地址失败", err)
	}
	host := localIP.IP.To4().String() + ":" + strconv.Itoa(port)
	la, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Panicln("eNodeB host配置解析失败", err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panicln("eNodeB host监听失败", err)
	}
	bip, err := lastAddr(localIP)
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

// 广播基站工作消息
// scan = 0, 广播网络侧消息
// scan = >0, 间断广播工作消息让UE捕获
// scan = -1, 此时remote为具体的ue，为端到端发送
func working(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, scan int, msg []byte) {
	defer modules.Recover(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站工作广播协程退出...", ctx.Value("Entity"))
			return
		default:
			_, err := conn.WriteToUDP(msg, remote)
			if err != nil {
				logger.Error("[%v] 广播开始工作消息失败... %v", ctx.Value("Entity"), err)
			}
			if scan <= 0 {
				return
			}
			time.Sleep(time.Duration(scan) * time.Second)
		}
	}
}

func tunneling(ctx context.Context, coreConn *CoreNetConnection, bConn *net.UDPConn, bAddr *net.UDPAddr) {
	var err error
	coreConn.PgwConn, err = net.Dial("udp4", coreConn.PgwAddr)
	if err != nil {
		logger.Info("[%v] 连接核心网PGW失败 %v", ctx.Value("Entity"), err)
		return
	}
	// 向mme和pgw发送心跳包，让对端知道自己的公网IP和端口
	go heartbeat(ctx, coreConn.PgwConn, coreConn.beatheart)
	go forwardMsgFromNetToUe(ctx, coreConn.PgwConn, bConn, bAddr)
	go forwardMsgFromUeToNet(ctx, bConn, coreConn)
}

func heartbeat(ctx context.Context, conn net.Conn, period int) {
	for {
		signal := []byte{0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F, 0x0F}
		msg := append(signal, []byte(TAI)...)
		_, err := conn.Write(msg)
		if err != nil {
			logger.Error("心跳探测发送失败 %v", err)
			return
		}
		time.Sleep(time.Second * time.Duration(period))
	}
}

func forwardMsgFromNetToUe(ctx context.Context, conn net.Conn, bconn *net.UDPConn, baddr *net.UDPAddr) {
	defer modules.Recover(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发广播网络侧消息协程退出...", ctx.Value("Entity"))
			return
		default:
			data := make([]byte, 10240) // 最多读取10KB数据包
			n, err := conn.Read(data)
			if err != nil {
				logger.Error("[%v] 读取网络侧数据错误 %v", ctx.Value("Entity"), err)
				continue
			}
			if n != 0 {
				logger.Error("[%v] 基站接收来自网络侧消息 %v(%v bytes)", ctx.Value("Entity"), data[:n], n)
				// 将收到的消息广播出去
				working(ctx, bconn, baddr, 0, data[:n])
			}
		}
	}
}

// 将ue消息转发至网络侧
func forwardMsgFromUeToNet(ctx context.Context, src *net.UDPConn, cConn *CoreNetConnection) {
	defer modules.Recover(ctx)

	var n int
	var err error
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发协程退出...", ctx.Value("Entity"))
			return
		default:
			data := make([]byte, 10240) // 10KB
			n, _, err = src.ReadFromUDP(data)
			if err != nil && n == 0 {
				logger.Error("[%v] 基站接收消息失败 %x %v", ctx.Value("Entity"), n, err)
			}
			logger.Error("[%v] 基站接收来自Ue消息 %v(%v bytes)", ctx.Value("Entity"), data[:n], n)
			err = send(ctx, cConn.PgwConn, data[:n])
			if err != nil {
				logger.Error("[%v] 基站转发消息失败[to pgw] %v %v", ctx.Value("Entity"), n, err)
			}
		}
	}
}

func send(ctx context.Context, conn net.Conn, msg []byte) error {
	_, err := conn.Write(msg)
	if err != nil {
		return err
	}
	return nil
}

func getLocalLanIP() (*net.IPNet, error) {
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
		return net.IP{}, errors.New("ErrNoIPv4")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}
