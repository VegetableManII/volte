package modules

import (
	"context"
	"errors"
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
func ReceiveClientMessage(ctx context.Context, conn *net.UDPConn, in chan *Package) {
	for {
		select {
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] 与下级节点通信协程退出", ctx.Value("Entity"))
			return
		default:
			data := make([]byte, 10240) // 10KB
			n, ra, err := conn.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] Server读取数据错误 %v", ctx.Value("Entity"), err)
			}
			if n != 0 {
				// 心跳兼容
				if data[0] == 0x0F && data[1] == 0x0F && data[2] == 0x0F && data[3] == 0x0F &&
					data[4] == 0x0F && data[5] == 0x0F && data[6] == 0x0F && data[7] == 0x0F {
					pkg := &Package{
						CommonMsg: CommonMsg{
							_unique:   0x0F0F0F0F,
							_protocal: 0x0F,
							_method:   0x0F,
							_size:     0x0F0F,
						},
						FixedConn: FixedConn(data[8:n]),
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
				if pkg.remoteAddr != nil && pkg.conn != nil {
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
				if pkg.remoteAddr != nil && pkg.conn != nil {
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

// 需要向其他功能实体发送数据是的通用方法，异步接收
func sendUDPMessage(ctx context.Context, host string, data []byte) (err error) {
	defer Recover(ctx)
	var n int
	ra, err := net.Dial("udp4", host)
	if err != nil {
		return err
	}
	defer ra.Close()
	n, err = ra.Write(data)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("ErrSendEmpty")
	}
	return nil
}

// 采用分发订阅模式分发epc网络信令和sip信令
func distribute(ctx context.Context, data []byte, ra *net.UDPAddr, conn *net.UDPConn, c chan *Package) {
	defer Recover(ctx)
	pkg := new(Package)
	err := pkg.Init(data)

	if err == nil {
		if pkg._protocal == EPCPROTOCAL && pkg._method == MultiMediaAuthenticationRequest {
			pkg.SetDynamicAddr(ra)
		}
		pkg.SetDynamicConn(conn) // 默认信息包都携带自身连接conn，用于需要时进行动态连接响应
		c <- pkg                 // 默认 发送核心处理
	} else {
		logger.Error("[%v] 消息分发失败, Error: %v", ctx.Value("Entity"), err)
	}
}
