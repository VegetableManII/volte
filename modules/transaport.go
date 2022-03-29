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
						CommonMsg: CommonMsg{
							_protocal: 0x0F,
							_method:   0x0F,
							_size:     0x0F0F,
						},
						FixedConn: FixedConn(data[4:n]),
					}
					pkg.SetDynamicAddr(ra)
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
			logger.Warn("[%v] 发送消息协程退出", ctx.Value("Entity"))
			return
		case pkg := <-down:
			host := string(pkg.FixedConn)
			var err error
			if pkg._protocal == EPCPROTOCAL {
				// 同步响应结果 或 使用动态连接
				if host == "" {
					n, err := pkg.conn.WriteToUDP(pkg.GetEpcMessage(), pkg.remoteAddr)
					if err != nil || n == 0 {
						logger.Error("[%v] 同步响应下级节点失败 %v", ctx.Value("Entity"), err)
					}
				} else { // 使用固定连接
					err = sendUDPMessage(ctx, host, pkg.GetEpcMessage())
					if err != nil {
						logger.Error("[%v] 请求下级节点失败 %v", ctx.Value("Entity"), err)
					}
				}

			} else {
				if host == "" {
					n, err := pkg.conn.WriteToUDP(pkg.GetSipMessage(), pkg.remoteAddr)
					if err != nil || n == 0 {
						logger.Error("[%v] 同步响应下级节点失败 %v", ctx.Value("Entity"), err)
					}
				} else {
					err = sendUDPMessage(ctx, host, pkg.GetSipMessage())
					if err != nil {
						logger.Error("[%v] 请求下级节点失败 %v", ctx.Value("Entity"), err)
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
			logger.Warn("[%v] 与上级节点通信协程退出...", ctx.Value("Entity"))
			return
		case pkt := <-up:
			host := string(pkt.FixedConn)
			var err error
			if pkt._protocal == EPCPROTOCAL {
				err = sendUDPMessage(ctx, host, pkt.GetEpcMessage())
				if err != nil {
					logger.Error("[%v] 请求上级节点失败 %v", ctx.Value("Entity"), err)
				}
			} else {
				err = sendUDPMessage(ctx, host, pkt.GetSipMessage())
				if err != nil {
					logger.Error("[%v] 请求上级节点失败 %v", ctx.Value("Entity"), err)
				}
			}
		}
	}
}

func Send(pkg *Package, out chan *Package) {
	out <- pkg
}

// func SyncRequest(ctx context.Context, pkg *Package) (*Package, error) {
// 	logger.Error("[%v] 同步请求", ctx.Value("Entity"))
// 	host := pkg.GetFixedConn()
// 	ra, err := net.Dial("udp4", host)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ra.SetReadDeadline(time.Now().Add(time.Second * 5))
// 	defer ra.Close()
// 	err = binary.Write(ra, binary.BigEndian, pkg.CommonMsg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	data := make([]byte, 65535)
// 	n, err := ra.Read(data)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if n == 0 {
// 		logger.Error("[%v] 同步请求,响应数据结果为空", ctx.Value("Entity"))
// 		return nil, errors.New("ErrEmptyResponse")
// 	}
// 	pkg0 := new(Package)
// 	err = pkg0.Init(data)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// 同步响应忽略数据来源
// 	// pkg0.SetDynamicAddr()
// 	// pkg0.SetDynamicConn()
// 	return pkg0, nil
// }

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
		pkg.SetDynamicAddr(ra)   // 默认携带请求对端地址，用于判断是上行还是下行
		pkg.SetDynamicConn(conn) // 默认信息包都携带自身连接conn，用于需要时进行动态连接响应
		c <- pkg
	}
}
