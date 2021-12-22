/*
eNodeB主要功能：消息转发
根据不同的消息类型转发到EPC网络还是IMS网络
*/
package main

import (
	"fmt"
	"log"
	"math/big"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var host string
var serverConn *net.UDPConn
var clientSubNetIP net.IP

func main() {
	log.Println(serverConn, clientSubNetIP)
	// 开启与ue通信的协程

	// 开启与mme通信的协程

	// 开启与pgw通信的协程
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
	host = viper.GetString("enodb")
	logger.Info("配置文件读取成功", "eNodeB running on %v", host)
	// 创建与UE的UDP连接
	serverConn, clientSubNetIP = initServer(host)
	// TODO 创建于MME的UDP连接
	// TODO 创建于PGW的UDP连接
}

// UDP服务端
func initServer(h string) (*net.UDPConn, net.IP) {
	s1 := strings.Split(h, ":")
	addr, sub, e := net.ParseCIDR(s1[0])
	_ = addr
	if e != nil {
		log.Panicln("host配置解析失败", e)
	}
	port, e := strconv.Atoi(s1[1])
	if e != nil {
		log.Panicln("端口信息获取错误", e)
	}
	// 启动监听本地端口
	conn, e := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if e != nil {
		log.Panicln("UDP服务器启动失败", e)
	}
	logger.Info("UDP服务器启动成功 %v %v", conn.LocalAddr().String(), conn.LocalAddr().Network())
	// 构造子网
	subNetIP := getSunBroadcastIP(sub)
	return conn, subNetIP
}

func getSunBroadcastIP(addr *net.IPNet) net.IP {
	ip, mask, sub := big.NewInt(0), big.NewInt(0), big.NewInt(0)
	ip.SetBytes(addr.IP)
	mask.SetBytes(addr.Mask)
	sub.Add(ip, mask)
	sub.Or(ip, mask.Not(mask))
	sub64 := sub.Int64()
	subip := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", byte(sub64>>24), byte(sub64>>16), byte(sub64>>8), byte(sub64)))
	return subip
}
