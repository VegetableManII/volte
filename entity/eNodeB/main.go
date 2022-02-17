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

	"github.com/VegetableManII/volte/common"
	"github.com/VegetableManII/volte/controller"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

type CoreNetConnection struct {
	PgwAddr string
	MmeAddr string
	MmeConn net.Conn
	PgwConn net.Conn
}

var (
	entity          *controller.EnodebEntity
	broadcastConn   *net.UDPConn
	ueBroadcastAddr *net.UDPAddr
	scanTime        int
	coreConnection  *CoreNetConnection
	beatheart       int
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "Entity", "eNodeB")
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// 开启广播工作消息
	go downLinkMessageTransport(ctx, broadcastConn, ueBroadcastAddr, scanTime, []byte("RandomAccess"))

	// 开启ue与核心网的通信协程
	go broadMessageFromNet(ctx, coreConnection, broadcastConn, ueBroadcastAddr)
	<-quit
	logger.Warn("[eNodeB] eNodeB 功能实体退出...")
	cancel()
	logger.Warn("[eNodeB] eNodeB 资源释放完成...")

}

// 读取配置文件
func init() {
	entity = new(controller.EnodebEntity)
	entity.Init()

	broadcastServerPort := viper.GetInt("eNodeB.broadcast.server.port")
	enodebBroadcastPort := viper.GetInt("eNodeB.broadcast.port")
	scanTime = viper.GetInt("eNodeB.scan.time")
	entity.TAI = viper.GetInt("eNodeB.TAI")
	beatheart = viper.GetInt("eNodeB.beatheart.time")
	// 启动与ue连接的服务器
	broadcastConn, ueBroadcastAddr = initUeServer(broadcastServerPort, enodebBroadcastPort)
	coreConnection = new(CoreNetConnection)
	// 创建于MME的UDP连接
	coreConnection.MmeAddr = viper.GetString("EPC.mme.host")
	// 创建于PGW的UDP连接
	coreConnection.PgwAddr = viper.GetString("EPC.pgw.host")
	logger.Info("配置文件读取成功", "")
}

// 与ue连接的UDP服务端
func initUeServer(port int, bport int) (*net.UDPConn, *net.UDPAddr) {
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
func downLinkMessageTransport(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, scan int, msg []byte) {
	defer common.Recover(ctx)

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

func broadMessageFromNet(ctx context.Context, coreConn *CoreNetConnection, bConn *net.UDPConn, bAddr *net.UDPAddr) {
	var err error
	coreConn.MmeConn, err = net.Dial("udp4", coreConn.MmeAddr)
	if err != nil {
		logger.Info("[%v] 连接核心网MME失败 %v", ctx.Value("Entity"), err)
		return
	}
	coreConn.PgwConn, err = net.Dial("udp4", coreConn.PgwAddr)
	if err != nil {
		logger.Info("[%v] 连接核心网PGW失败 %v", ctx.Value("Entity"), err)
		return
	}
	// 向mme和pgw发送心跳包，让对端知道自己的公网IP和端口
	//go heartbeat(ctx, coreConn.MmeConn, beatheart)
	//go heartbeat(ctx, coreConn.PgwConn, beatheart)
	go proxy(ctx, coreConn.MmeConn, bConn, bAddr)
	go proxy(ctx, coreConn.PgwConn, bConn, bAddr)
	go proxyMessageFromUEtoCoreNet(ctx, bConn, coreConn)
}

func heartbeat(ctx context.Context, conn net.Conn, period int) {
	for {
		_, err := conn.Write([]byte{0x13, 0x14})
		if err != nil {
			logger.Error("心跳探测发送失败 %v", err)
			return
		}
		time.Sleep(time.Second * time.Duration(period))
	}
}

func proxy(ctx context.Context, conn net.Conn, bconn *net.UDPConn, baddr *net.UDPAddr) {
	defer common.Recover(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发广播网络侧消息协程退出...", ctx.Value("Entity"))
			return
		default:
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				// logger.Error("[%v] 读取网络侧数据错误 %v", ctx.Value("Entity"), err)
				continue
			}
			if n != 0 {
				entity.ParseDownLinkData(data)
				// 将收到的消息广播出去
				downLinkMessageTransport(ctx, bconn, baddr, 0, data[:n])
			}
		}
	}
}

// 将ue消息转发至网络侧
func proxyMessageFromUEtoCoreNet(ctx context.Context, src *net.UDPConn, cConn *CoreNetConnection) {
	defer common.Recover(ctx)

	var n int
	var err error
	var raddr *net.UDPAddr
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发协程退出...", ctx.Value("Entity"))
			return
		default:
			data := make([]byte, 1024)
			n, raddr, err = src.ReadFromUDP(data)
			if err != nil && n == 0 {
				logger.Error("[%v] 基站接收消息失败 %v %v", ctx.Value("Entity"), n, err)
			}
			logger.Info("[%v] 基站接收消息%v %v(%v byte)", ctx.Value("Entity"), data[0:4], string(data[4:n]), n)
			// 如果用户随机接入则响应给用户分配的唯一ID
			if ok, id := entity.UeRandomAccess(data, raddr); ok {
				downLinkMessageTransport(ctx, src, raddr, -1, id)
				continue
			}
			message, dest, err := entity.GenerateUpLinkData(data, n, "MME", "PGW")
			if err != nil {
				if err.Error() == "ErrNeedAccessInfo" {
					downLinkMessageTransport(ctx, src, raddr, -1, []byte("RandomAccess"))
				}
				logger.Info("[%v] 基站转发消息[to %v] %v", ctx.Value("Entity"), dest, string(data[8:]))
				continue
			}
			if dest == "MME" {
				err = upLinkTransport(ctx, cConn.MmeConn, message[:n])
				if err != nil {
					logger.Error("[%v] 基站转发消息失败[to mme] %v %v", ctx.Value("Entity"), n, err)
				}
				logger.Info("[%v] 基站转发消息[to mme] %v %v", ctx.Value("Entity"), data[0:8], string(data[8:]))
			} else if dest == "PGW" {
				err = upLinkTransport(ctx, cConn.PgwConn, message[:n])
				if err != nil {
					logger.Error("[%v] 基站转发消息失败[to pgw] %v %v", ctx.Value("Entity"), n, err)
				}
				logger.Info("[%v] 基站转发消息[to pgw] %v %v", ctx.Value("Entity"), data[0:4], string(data[4:]))
			} else {
				logger.Info("[%v] 基站转发消息[to %v] %v", ctx.Value("Entity"), dest, string(data[4:]))
			}
		}
	}
}

func upLinkTransport(ctx context.Context, conn net.Conn, msg []byte) error {
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
