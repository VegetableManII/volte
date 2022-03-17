package modules

import (
	"bytes"
	"context"
	"encoding/binary"
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
						FixedConn: FixedConn(data[8:]),
					}
					pkg.SetDynamicConn(ra, conn)
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
	// 创建write buffer
	var buffer bytes.Buffer
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
				err = binary.Write(&buffer, binary.BigEndian, pkg.CommonMsg)
				if err != nil {
					logger.Error("[%v] 序列化失败 %v", ctx.Value("Entity"), err)
					continue
				}
				// 同步响应结果 或 使用动态连接
				if pkg.remoteAddr != nil && pkg.conn != nil {
					n, err := pkg.conn.WriteToUDP(buffer.Bytes(), pkg.remoteAddr)
					if err != nil || n == 0 {
						logger.Error("[%v] 同步响应下级节点失败 %v", ctx.Value("Entity"), err)
					}
				} else { // 使用固定连接
					err = sendUDPMessage(ctx, host, buffer.Bytes())
					if err != nil {
						logger.Error("[%v] 请求下级节点失败 %v", ctx.Value("Entity"), err)
					}
				}

			} else {
				if pkg.remoteAddr != nil && pkg.conn != nil {
					n, err := pkg.conn.WriteToUDP(pkg.GetSipBody(), pkg.remoteAddr)
					if err != nil || n == 0 {
						logger.Error("[%v] 同步响应下级节点失败 %v", ctx.Value("Entity"), err)
					}
				} else {
					err = sendUDPMessage(ctx, host, pkg.GetSipBody())
					if err != nil {
						logger.Error("[%v] 请求下级节点失败 %v", ctx.Value("Entity"), err)
					}
				}
			}
		}
		buffer.Reset()
	}
}

func ProcessUpStreamData(ctx context.Context, up chan *Package) {
	var buffer bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 与上级节点通信协程退出...", ctx.Value("Entity"))
			return
		case pkt := <-up:
			host := string(pkt.FixedConn)
			var err error
			if pkt._protocal == EPCPROTOCAL {
				err = binary.Write(&buffer, binary.BigEndian, pkt.CommonMsg)
				if err != nil {
					logger.Error("[%v] 序列化失败 %v", ctx.Value("Entity"), err)
					continue
				}
				err = sendUDPMessage(ctx, host, buffer.Bytes())
				if err != nil {
					logger.Error("[%v] 请求上级节点失败 %v", ctx.Value("Entity"), err)
				}
			} else {
				err = sendUDPMessage(ctx, host, pkt.GetSipBody())
				if err != nil {
					logger.Error("[%v] 请求上级节点失败 %v", ctx.Value("Entity"), err)
				}
			}
		}
		buffer.Reset()
	}
}

func MAASyncResponse(pkg *Package, out chan *Package) {
	out <- &Package{pkg.CommonMsg, "", pkg.DynamicConn}
}

func MARSyncRequest(ctx context.Context, pkg *Package) (*CommonMsg, error) {
	buf := new(bytes.Buffer)
	buf.Grow(65535)
	binary.Write(buf, binary.BigEndian, pkg.CommonMsg)
	resp, err := sendUDPMessageWaitResp(ctx, string(pkg.FixedConn), buf.Bytes())
	if err != nil {
		return nil, err
	}
	pk := new(Package)
	pk.Init(resp)
	return &pk.CommonMsg, nil
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

// 同步接收响应
func sendUDPMessageWaitResp(ctx context.Context, host string, data []byte) (response []byte, err error) {
	defer Recover(ctx)
	var n int
	ra, err := net.Dial("udp4", host)
	if err != nil {
		return nil, err
	}
	n, err = ra.Write(data)
	if n == 0 {
		return nil, errors.New("ErrSendEmpty")
	}
	ra.SetReadDeadline(time.Now().Add(5 * time.Second)) // 等待响应的过期时间为5秒
	buf := make([]byte, 1024)
	n, err = ra.Read(buf)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("ErrReceiveEmpty")
	}
	return buf[:n], nil
}

// 采用分发订阅模式分发epc网络信令和sip信令
func distribute(ctx context.Context, data []byte, ra *net.UDPAddr, conn *net.UDPConn, c chan *Package) {
	defer Recover(ctx)
	pkg := new(Package)
	pkg.Init(data)
	c <- pkg // 默认 发送核心处理
}
