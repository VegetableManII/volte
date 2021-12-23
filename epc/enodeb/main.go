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
	"fmt"
	"log"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	ueSerConn, mmeConn, pgwConn *net.UDPConn
	wg                          *sync.WaitGroup
	scanTime                    int
)

type Msg struct {
	Type  byte // 0x01 epc 0x00 ims
	Data1 *EpcMsg
	Data2 *SipMsg
}

func main() {
	wg = new(sync.WaitGroup)
	msgChan := make(chan *Msg, 2)
	wg.Add(1)
	// 开启与ue通信的协程
	go exchangeWithUe(wg, ueSerConn, msgChan, scanTime)
	// 开启与mme通信的协程

	// 开启与pgw通信的协程
	wg.Wait()
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
	host := viper.GetString("eNodeB.host")
	enodebBroadcastNet := viper.GetString("eNodeB.broadcast.net")
	ueListenPort := viper.GetInt("eNodeB.listen.port")
	scanTime = viper.GetInt("eNodeB.scan.time")
	// ueListenPort = viper.GetInt("eNodeB.ue.listen.port")
	logger.Info("配置文件读取成功", "")
	// 启动与ue连接的服务器
	ueSerConn = initUeServer(host, ueListenPort, enodebBroadcastNet)
	// 作为客户端与epc网络连接

	// 创建于MME的UDP连接
	mme := viper.GetString("EPC.mme")
	mmeConn = connectEPC(mme)
	// TODO 创建于PGW的UDP连接
	pgw := viper.GetString("EPC.pgw")
	pgwConn = connectEPC(pgw)
}

// 与ue连接的UDP服务端
func initUeServer(host string, port int, broadcast string) *net.UDPConn {
	la, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Panicln("eNodeB host配置解析失败")
	}
	_, sub, e := net.ParseCIDR(broadcast)
	if e != nil {
		log.Panicln("广播子网配置解析失败", e)
	}
	// 构造子网
	subNetIP := getSubBroadcastIP(sub)
	// 和广播子网建立连接
	conn, err := net.DialUDP("udp4", la, &net.UDPAddr{IP: subNetIP, Port: port})
	if err != nil {
		log.Panicln("与广播子网连接建立失败", err)
	}
	logger.Info("ue UDP广播服务器启动成功 [%v:%v]", la.IP, la.Port)
	logger.Info("UDP广播子网 [%v:%v]", sub.IP, port)
	return conn
}

func getSubBroadcastIP(addr *net.IPNet) net.IP {
	ip, mask, sub := big.NewInt(0), big.NewInt(0), big.NewInt(0)
	ip.SetBytes(addr.IP)
	mask.SetBytes(addr.Mask)
	sub.Add(ip, mask)
	sub.Or(ip, mask.Not(mask))
	sub64 := sub.Int64()
	subip := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", byte(sub64>>24), byte(sub64>>16), byte(sub64>>8), byte(sub64)))
	return subip
}

func connectEPC(dest string) *net.UDPConn {
	tmp := strings.Split(dest, ":")
	ip := net.ParseIP(tmp[0])
	port, err := strconv.Atoi(tmp[1])
	if err != nil {
		log.Panicln("epc UDP服务端端口配置解析失败", err)
	}
	addr := net.UDPAddr{
		IP:   ip,
		Port: port,
	}
	conn, err := net.DialUDP("udp4", nil, &addr)
	if err != nil {
		log.Panicln("epc UDP客户端启动失败", err)
	}
	logger.Info("epc UDP客户端启动成功 %v %v", conn.LocalAddr().String(), conn.LocalAddr().Network())
	return conn
}

func exchangeWithUe(wg *sync.WaitGroup, conn *net.UDPConn, c chan *Msg, scan int) {
	defer wg.Done()
	// 广播基站开始工作消息
	_, err := conn.Write([]byte("Broadcast to Ue"))
	if err != nil {
		logger.Error("广播开始工作消息失败......")
		return
	}
	// 使用context管理多个用户连接的写入
	ctx, cancel := context.WithCancel(context.Background())
	buf := make([]byte, 1024)
	buffer := bytes.NewBuffer(buf)
	for {
		buffer.Reset()
		n, raddr, err := conn.ReadFromUDP(buffer.Bytes())
		if err != nil {
			logger.Error("Ue Server读取UDP数据错误 %v", err)
			break
		}
		if raddr == nil {
			time.Sleep(time.Duration(scan) * time.Second)
			continue
		}
		logger.Info("Read From Ue[%v:%v] Len: %v Data: %v", raddr.IP, raddr.Port, n, buffer.Bytes())
		// 分发消息给订阅通道
		distribute(buffer.Bytes(), c)
		// 启动写协程

		go writeToUe(ctx, conn, c)
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

func writeToUe(c context.Context, conn *net.UDPConn, ch chan *Msg) {
	// 创建write buffer
	buf := make([]byte, 1024)
	buffer := bytes.NewBuffer(buf)
	select {
	case msg := <-ch:
		if msg.Type == 0x01 {
			err := binary.Write(buffer, binary.LittleEndian, msg.Data1)
			if err != nil {
				logger.Error("EpcMsg转化[]byte失败 %v", err)
			}
			_, err = conn.Write(buffer.Bytes())
			if err != nil {
				logger.Error("EpcMsg广播消息发送失败 %v %v", err, buffer.Bytes())
			}
		} else {
			err := binary.Write(buffer, binary.BigEndian, msg.Data2)
			if err != nil {
				logger.Error("SipMsg转化[]byte失败 %v", err)
			}
			_, err = conn.Write(buffer.Bytes())
			if err != nil {
				logger.Error("SipMsg广播消息发送失败 %v %v", err, buffer.Bytes())
			}
		}
	}
}
