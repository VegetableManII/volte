package modules

import (
	"context"
	"encoding/binary"
	"log"
	"net"
	"time"

	"github.com/wonderivan/logger"
)

func CreateServer(host string) *net.UDPConn {
	lo, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		logger.Fatal("解析地址失败 %v", err)
	}
	logger.Info("服务监听启动成功 %v", lo.String())
	conn, err := net.ListenUDP("udp4", lo)
	if err != nil {
		log.Panicln("udp server 监听失败", err)
	}
	logger.Info("服务器启动成功[%v]", lo)
	return conn
}

// 通用网络中的功能实体与接收客户端数据的通用方法
func ReceiveMessage(ctx context.Context, conn *net.UDPConn, in chan *Package) {
	for {
		defer Recover(ctx)
		select {
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] 接收消息协程退出", ctx.Value("Entity"))
			return
		default:
			data := make([]byte, 10240) // 10KB
			n, ra, err := conn.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] Server读取数据错误 %v", ctx.Value("Entity"), err)
			}
			if n != 0 {
				// 心跳兼容
				if data[0] == 0x0F && data[1] == 0x0F && data[2] == 0x0F && data[3] == 0x0F {
					pkg := &Package{
						msg: CommonMsg{
							_protocal: 0x0F,
							_method:   0x0F,
							_size:     0x0F0F,
						},
						shortc: ShortConn(data[4:n]),
					}
					pkg.SetLongAddr(ra)
					in <- pkg
					continue
				}
				distribute(ctx, data[:n], ra, conn, in)
			} else {
				logger.Info("[%v] Read Len[%v]", ctx.Value("Entity"), n)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}

// 接收逻辑核心处理结果
func ProcessDownStreamData(ctx context.Context, down chan *Package) {
	for {
		defer Recover(ctx)
		select {
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] 发送下行消息协程退出", ctx.Value("Entity"))
			return
		case pkg := <-down:
			host := string(pkg.shortc)
			var err error
			if pkg.msg._protocal == EPCPROTOCAL {
				// 使用下游固定地址 或 使用下游连接
				if host == "" {
					n, err := pkg.longc.conn.WriteToUDP(pkg.msg.GetEpcMessage(), pkg.longc.remoteAddr)
					if err != nil || n == 0 {
						logger.Error("[%v] 向下行连接发送数据失败 err: %v, down: %v", ctx.Value("Entity"), err, pkg.longc.remoteAddr)
					}
				} else { // 使用固定连接
					err = sendUDPMessage(ctx, host, pkg.msg.GetEpcMessage())
					if err != nil {
						logger.Error("[%v] 向下行固定网络地址发送数据失败 err: %v, dowm: %v", ctx.Value("Entity"), err, host)
					}
				}

			} else {
				if host == "" {
					n, err := pkg.longc.conn.WriteToUDP(pkg.msg.GetSipMessage(), pkg.longc.remoteAddr)
					if err != nil || n == 0 {
						logger.Error("[%v] 向下行连接发送数据失败 err: %v, down: %v", ctx.Value("Entity"), err, pkg.longc.remoteAddr)
					}
				} else {
					err = sendUDPMessage(ctx, host, pkg.msg.GetSipMessage())
					if err != nil {
						logger.Error("[%v] 向下行固定网络地址发送数据失败 err: %v, dowm: %v", ctx.Value("Entity"), err, host)
					}
				}
			}
		}
	}
}

func ProcessUpStreamData(ctx context.Context, up chan *Package) {
	for {
		defer Recover(ctx)
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 与上行节点通信协程退出...", ctx.Value("Entity"))
			return
		case pkt := <-up:
			host := string(pkt.shortc)
			var err error
			if pkt.msg._protocal == EPCPROTOCAL {
				err = sendUDPMessage(ctx, host, pkt.msg.GetEpcMessage())
				if err != nil {
					logger.Error("[%v] 向上行节点发送数据失败 err: %v, up: %v", ctx.Value("Entity"), err, host)
				}
			} else {
				err = sendUDPMessage(ctx, host, pkt.msg.GetSipMessage())
				if err != nil {
					logger.Error("[%v] 向上行节点发送数据失败 err: %v, up: %v", ctx.Value("Entity"), err, host)
				}
			}
		}
	}
}

func Send(pkg *Package, out chan *Package) {
	out <- pkg
}

// 需要向其他功能实体发送数据是的通用方法，异步接收
func sendUDPMessage(ctx context.Context, host string, data []byte) (err error) {
	defer Recover(ctx)
	ra, err := net.Dial("udp4", host)
	if err != nil {
		return err
	}
	defer ra.Close()
	err = binary.Write(ra, binary.BigEndian, data)
	if err != nil {
		return err
	}
	return nil
}

// 采用分发订阅模式分发epc网络信令和sip信令
func distribute(ctx context.Context, data []byte, ra *net.UDPAddr, conn *net.UDPConn, c chan *Package) {
	defer Recover(ctx)
	pkg := new(Package)
	err := pkg.Init(data)
	if err != nil {
		logger.Error("[%v] 消息分发失败, Error: %v", ctx.Value("Entity"), err)
	} else {
		pkg.SetLongAddr(ra)   // 默认携带请求对端地址，用于判断是上行还是下行
		pkg.SetLongConn(conn) // 默认信息包都携带自身连接conn，用于需要时进行动态连接响应
		c <- pkg
	}
}
