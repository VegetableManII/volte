package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/wonderivan/logger"
)

/*
并发安全集合用来保存客户端连接
新连接接入之后保存对方的IP到集合中
当连接断开时没有从集合中删除???如何判断已经断开
*/
var clientMap *sync.Map

func TransaportWithClient(ctx context.Context, conn *net.UDPConn, coreIn, coreOut chan *Package) {
	clientMap = new(sync.Map)
	data := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			// 释放资源
			// close(pre) // 关闭生产者通道
			logger.Warn("[%v] 与下级节点通信协程退出", ctx.Value("Entity"))
			return
		default:
			n, remote, err := conn.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] Server读取数据错误 %v", ctx.Value("Entity"), err)
			}
			if remote != nil || n != 0 {
				distribute(data[:n], coreIn)
				// 检查该客户端是否已经开启线程服务
				if _, ok := clientMap.Load(remote); ok {
					continue
				} else {
					clientMap.Store(remote, ctx.Value("Entity"))
					go receiveCoreProcessResult(ctx, conn, remote, coreOut)
				}
			} else {
				logger.Info("[%v] Remote[%v] Len[%v]", ctx.Value("Entity"), remote, n)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}

// 接收逻辑核心处理结果
func receiveCoreProcessResult(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, out chan *Package) {
	// 创建write buffer
	var buffer bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] 发送消息协程退出", ctx.Value("Entity"))
			return
		case pkg := <-out:
			if pkg._type == 0x01 {
				err := binary.Write(&buffer, binary.BigEndian, pkg)
				if err != nil {
					logger.Error("[%v] EpsMsg转化[]byte失败 %v", ctx.Value("Entity"), err)
					continue
				}
				if pkg.Destation { // 上行，remote为服务端，remote参数不用传递
					writeToRemote(ctx, conn, nil, buffer.Bytes())
				} else { // 下行，remote为客户端，remote参数需要传递
					writeToRemote(ctx, conn, remote, buffer.Bytes())
				}
			} else {
				err := binary.Write(&buffer, binary.BigEndian, pkg)
				if err != nil {
					logger.Error("[%v] SipMsg转化[]byte失败 %v", ctx.Value("Entity"), err)
					continue
				}
				if pkg.Destation { // 上行，remote为服务端，remote参数不用传递
					writeToRemote(ctx, conn, nil, buffer.Bytes())
				} else { // 下行，remote为客户端，remote参数需要传递
					writeToRemote(ctx, conn, remote, buffer.Bytes())
				}
			}
			buffer.Reset()
		}
	}
}
func writeToRemote(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, data []byte) {
	var err error
	conn.RemoteAddr()
	if remote == nil {
		_, err = conn.Write(data)
	} else {
		_, err = conn.WriteToUDP(data, remote)
	}
	if err != nil {
		logger.Error("[%v] 消息发送失败 %v %v", ctx.Value("Entity"), err, data)
	}
}

// 采用分发订阅模式分发eps网络信令和sip信令
func distribute(data []byte, c chan *Package) {
	cmsg := new(CommonMsg)
	cmsg.Init(data)
	c <- &Package{cmsg, false}
}

// 功能实体作为客户端与其上层服务端交互
func TransportWithServer(ctx context.Context, lo *net.UDPConn, remote *net.UDPAddr, coreIn, coreOut chan *Package) {
	// 开启协程向服务端写数据
	go receiveCoreProcessResult(ctx, lo, remote, coreOut)
	// 循环读取服务端消息
	data := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 与上级节点通信协程退出...", ctx.Value("Entity"))
			return
		default:
			n, _, err := lo.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] Client读取数据错误 %v", ctx.Value("Entity"), err)
			}
			if n != 0 {
				distribute(data[:n], coreIn)
			} else {
				logger.Info("[%v] Remote[%v] Len[%v]", ctx.Value("Entity"), remote, n)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}

// 主要用于基站实现消息的代理转发, 将ue消息转发至网络侧
func EnodebProxyMessage(ctx context.Context, src, dest *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发协程退出...", ctx.Value("Entity"))
			return
		default:
			// 循环代理转发用户侧到网络侧消息
			n, err := io.Copy(dest, src) // 阻塞式,copy有deadline,如果src传输数据过快copy会一直进行复制
			if err != nil {
				logger.Error("[%v] 基站转发消息失败 %v %v", ctx.Value("Entity"), n, err)
			}
		}
	}
}

// 主要用于PGW实现消息的代理转发, 将EPS域消息转发至IMS域
func PGWProxyMessage(ctx context.Context, src, dest *net.UDPConn) {
	for {
		select {
		case <-ctx.Done():
			logger.Warn("[%v] PGW转发协程退出...", ctx.Value("Entity"))
			return
		default:
			// 循环代理转发EPS侧到IMS侧消息
			n, err := io.Copy(dest, src) // 阻塞式,copy有deadline,如果src传输数据过快copy会一直进行复制
			if err != nil {
				logger.Error("[%v] PGW转发消息失败 %v %v", ctx.Value("Entity"), n, err)
			}
		}
	}
}
