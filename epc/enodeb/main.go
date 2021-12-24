/*
eNodeB主要功能：消息转发
根据不同的消息类型转发到EPC网络还是IMS网络
*/
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	. "epc/enodeb/service"
	"log"
	"net"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	loConn, mmeConn, pgwConn *net.UDPConn
	ueBroadcastAddr          *net.UDPAddr
	host, enodebBroadcastNet string
	wg                       *sync.WaitGroup
	scanTime                 int
)

type Msg struct {
	Type  byte // 0x01 epc 0x00 ims
	Data1 *EpcMsg
	Data2 *SipMsg
}

func main() {
	wg = new(sync.WaitGroup)
	msgRecvChan := make(chan *Msg, 2)
	msgSendChan := make(chan *Msg, 2)
	wg.Add(1)
	// 开启与ue通信的协程
	go exchangeWithUe(wg, loConn, ueBroadcastAddr, msgRecvChan, msgSendChan, scanTime)
	// 开启与mme通信的协程

	// 开启与pgw通信的协程
	wg.Wait()
	loConn.Close()
}

// 读取配置文件
func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath("../conf/")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host = viper.GetString("eNodeB.host")
	enodebBroadcastNet = viper.GetString("eNodeB.broadcast.net")
	scanTime = viper.GetInt("eNodeB.scan.time")
	logger.Info("配置文件读取成功", "")
	// 启动与ue连接的服务器
	loConn, ueBroadcastAddr = initUeServer(host, enodebBroadcastNet)
	// 作为客户端与epc网络连接

	// 创建于MME的UDP连接
	//mme := viper.GetString("EPC.mme")
	//mmeConn = connectEPC(mme)
	// TODO 创建于PGW的UDP连接
	//pgw := viper.GetString("EPC.pgw")
	//pgwConn = connectEPC(pgw)
}

// 与ue连接的UDP服务端
func initUeServer(host string, broadcast string) (*net.UDPConn, *net.UDPAddr) {
	la, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Panicln("eNodeB host配置解析失败", err)
	}
	ra, err := net.ResolveUDPAddr("udp4", "255.255.255.255:12345")
	if err != nil {
		log.Panicln("eNodeB 广播地址配置解析失败", err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panicln("eNodeB host监听失败", err)
	}
	// 设置读超时
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		log.Panicln(err)
	}
	logger.Info("ue UDP广播服务器启动成功 [%v]", host)
	logger.Info("UDP广播子网 [%v]", broadcast)
	return conn, ra
}

func connectEPC(dest string) *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp4", dest)
	if err != nil {
		logger.Error("地址解析错误 %v", err)
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Panicln("epc UDP客户端启动失败", err)
	}
	logger.Info("epc UDP客户端启动成功 %v %v", conn.LocalAddr().String(), conn.LocalAddr().Network())
	return conn
}

func exchangeWithUe(wg *sync.WaitGroup, conn *net.UDPConn, raddr *net.UDPAddr, recv, send chan *Msg, scan int) {
	defer wg.Done()
	// 使用context管理多个用户连接的写入
	ctx, cancel := context.WithCancel(context.Background())
	// 广播基站工作消息
	go func(ctx context.Context) {
		_ = ctx
		for {
			n, err := conn.WriteToUDP([]byte("Broadcast to Ue"), raddr)
			if err != nil {
				logger.Error("广播开始工作消息失败......  %v", err)
			}
			logger.Info("Write to %v Len:%v", raddr, n)
			time.Sleep(1 * time.Second)
		}
	}(ctx)
	// 启动写协程
	// go writeToUe(ctx, conn, raddr, send)
	// go writeToUe(ctx, conn, raddr, recv) // debug

	buf := make([]byte, 32)
	for {
		n, r, err := conn.ReadFromUDP(buf)
		if err != nil {
			logger.Error("Ue Server读取UDP数据错误 %v", err)
			break
		}
		if raddr != nil || n != 0 {
			logger.Warn("Read From Ue[%v] Len: %v Data: %v", r, n, buf)
			// 分发消息给订阅通道
			distribute(buf, recv)
		} else {
			logger.Info("remote: %v, len: %v", r, n)
			time.Sleep(1 * time.Second)
		}
		buf = buf[:] // 清空
	}
	// 退出所有的ue端写协程
	cancel()
}

// 采用订阅模式分发epc网络信令和sip信令
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

func writeToUe(c context.Context, conn *net.UDPConn, raddr *net.UDPAddr, ch chan *Msg) {
	// 创建write buffer
	var buffer bytes.Buffer
	var n int
	select {
	case msg := <-ch:
		if msg.Type == 0x01 {
			err := binary.Write(&buffer, binary.BigEndian, msg.Data1)
			if err != nil {
				logger.Error("EpcMsg转化[]byte失败 %v", err)
			}
			n, err = conn.WriteTo([]byte("wdnmdwdnmd"), raddr)
			if err != nil {
				logger.Error("EpcMsg广播消息发送失败 %v %v", err, buffer.Bytes())
			}
		} else {
			err := binary.Write(&buffer, binary.BigEndian, msg.Data2)
			if err != nil {
				logger.Error("SipMsg转化[]byte失败 %v", err)
			}
			n, err = conn.WriteTo(buffer.Bytes(), raddr)
			if err != nil {
				logger.Error("SipMsg广播消息发送失败 %v %v", err, buffer.Bytes())
			}
		}
		logger.Info("Write to Ue[%v] Len: %v Data: %v", raddr, n, buffer.Bytes())
		buffer.Reset()
	}
}
