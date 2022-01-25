package common

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

// 主要用于基站实现消息的代理转发, 将ue消息转发至网络侧
func EnodebProxyMessage(ctx context.Context, src *net.UDPConn, mme, pgw string) {
	var n int
	var err error
	for {
		data := make([]byte, 1024)
		select {
		case <-ctx.Done():
			logger.Warn("[%v] 基站转发协程退出...", ctx.Value("Entity"))
			return
		default:
			n, _, err = src.ReadFromUDP(data)
			if err != nil && n == 0 {
				logger.Error("[%v] 基站接收消息失败 %v %v", ctx.Value("Entity"), n, err)
			}
			logger.Info("[%v] 基站接收消息%v(byte)", ctx.Value("Entity"), n)
			if data[0] == EPCPROTOCAL {
				err = sendUDPMessage(ctx, mme, data[:n])
				if err != nil {
					logger.Error("[%v] 基站转发消息失败[to mme] %v %v", ctx.Value("Entity"), n, err)
				}
				logger.Info("[%v] 基站转发消息[to mme] %v", ctx.Value("Entity"), string(data[4:]))
			} else {
				err = sendUDPMessage(ctx, pgw, data[:n])
				if err != nil {
					logger.Error("[%v] 基站转发消息失败[to pgw] %v %v", ctx.Value("Entity"), n, err)
				}
				logger.Info("[%v] 基站转发消息[to pgw] %v", ctx.Value("Entity"), string(data))
			}
		}
	}
}

/*
并发安全集合用来保存客户端连接
新连接接入之后保存对方的IP到集合中
当连接断开时没有从集合中删除???如何判断已经断开
*/

// 通用网络中的功能实体与接收客户端数据的通用方法
func ReceiveClientMessage(ctx context.Context, host string, in chan *Package) {
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
	for {
		data := make([]byte, 1024)
		select {
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] 与下级节点通信协程退出", ctx.Value("Entity"))
			return
		default:
			n, _, err := conn.ReadFromUDP(data)
			if err != nil {
				logger.Error("[%v] Server读取数据错误 %v", ctx.Value("Entity"), err)
			}
			logger.Info("[%v] Server读取到%v(byte)数据", ctx.Value("Entity"), n)
			if n != 0 {
				distribute(ctx, data[:n], in)
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
			host := pkg.Destation
			var err error
			if pkg._type == SIPPROTOCAL {
				err = binary.Write(&buffer, binary.BigEndian, pkg.CommonMsg)
				if err != nil {
					logger.Error("[%v] 序列化失败 %v", ctx.Value("Entity"), err)
					continue
				}
				err = sendUDPMessage(ctx, host, buffer.Bytes())
				if err != nil {
					logger.Error("[%v] 请求下级节点失败 %v", ctx.Value("Entity"), err)
				}
			} else {
				err = sendUDPMessage(ctx, host, pkg.GetData())
				if err != nil {
					logger.Error("[%v] 请求下级节点失败 %v", ctx.Value("Entity"), err)
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
			host := pkt.Destation
			var err error
			if pkt._type == EPCPROTOCAL {
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
				err = sendUDPMessage(ctx, host, pkt.GetData())
				if err != nil {
					logger.Error("[%v] 请求上级节点失败 %v", ctx.Value("Entity"), err)
				}
			}

		}
		buffer.Reset()
	}
}

// 需要向其他功能实体发送数据是的通用方法
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
func distribute(ctx context.Context, data []byte, c chan *Package) {
	defer Recover(ctx)
	cmsg := new(CommonMsg)
	cmsg.Init(data)
	c <- &Package{cmsg, "CLIENT"} // 默认 发送给下行节点
}
