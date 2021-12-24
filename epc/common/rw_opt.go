package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/wonderivan/logger"
)

var clientMap *sync.Map

func ExchangeWithClient(ctx context.Context, conn *net.UDPConn, producerC, consumerC chan *Msg) {
	clientMap = new(sync.Map)
	data := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			// 释放资源
			close(producerC) // 关闭生产者通道
			logger.Warn("[%v] 信令交互协程退出", ctx.Value("Entity"))
		default:
		}
		n, remote, err := conn.ReadFromUDP(data)
		if err != nil {
			logger.Error("[%v] Server读取数据错误 %v", ctx.Value("Entity"), err)
		}
		if remote != nil || n != 0 {
			clientMap.Store(remote, ctx.Value("Entity"))
			logger.Warn("[%v] Read[%v] Data: %v", ctx.Value("Entity"), n, data[:n])
			distribute(data, producerC)
			go writeToClient(ctx, conn, remote, consumerC)
		} else {
			logger.Info("[%v] Remote[%v] Len[%v]", ctx.Value("Entity"), remote, n)
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func writeToClient(ctx context.Context, conn *net.UDPConn, remote *net.UDPAddr, consumerC chan *Msg) {
	// 检查该客户端是否已经开启线程服务
	if _, ok := clientMap.Load(remote); ok {
		logger.Info("[%v] Client[%v]服务协程已开启", ctx.Value("Entity"), remote)
		return
	}
	// 创建write buffer
	var buffer bytes.Buffer
	var n int
	for {
		select {
		case <-ctx.Done():
			// 释放资源
			close(consumerC) // 关闭消费者通道
			logger.Warn("[%v] 发送信令至客户端协程退出", ctx.Value("Entity"))
		case msg := <-consumerC:
			if msg.Type == 0x01 {
				err := binary.Write(&buffer, binary.BigEndian, msg.Data1)
				if err != nil {
					logger.Error("[%v] EpcMsg转化[]byte失败 %v", ctx.Value("Entity"), err)
					continue
				}
				n, err = conn.WriteToUDP(buffer.Bytes(), remote)
				if err != nil {
					logger.Error("[%v] EpcMsg广播消息发送失败 %v %v", ctx.Value("Entity"), err, buffer.Bytes())
				}
			} else {
				err := binary.Write(&buffer, binary.BigEndian, msg.Data2)
				if err != nil {
					logger.Error("[%v] SipMsg转化[]byte失败 %v", ctx.Value("Entity"), err)
					continue
				}
				n, err = conn.WriteToUDP(buffer.Bytes(), remote)
				if err != nil {
					logger.Error("[%v] SipMsg广播消息发送失败 %v %v", ctx.Value("Entity"), err, buffer.Bytes())
				}
			}
			logger.Info("[%v] Write to Client[%v] Data[%v]:%v", ctx.Value("Entity"), remote, n, buffer.Bytes())
			buffer.Reset()
		}
	}
}

// 采用分发订阅模式分发epc网络信令和sip信令
func distribute(data []byte, c chan *Msg) {
	if data[0] == 0x01 { // epc电路域协议
		msg := new(EpcMsg)
		size := [2]byte{}
		copy(size[:], data[2:4])
		length := [4]byte{}
		copy(length[:], data[4:8])
		msg.Init(data[0], data[1], size, length, data[8:])
		c <- &Msg{
			Type:  0x01,
			Data1: msg,
		}
	} else { // ims协议
		// todo
	}
}
