package common

import (
	"log"
	"net"

	"github.com/wonderivan/logger"
)

func ConnectEPC(dest string) *net.UDPConn {
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

func InitServer(host string) *net.UDPConn {
	lo, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		log.Panicln("udp server host配置解析失败", err)
	}
	conn, err := net.ListenUDP("udp4", lo)
	if err != nil {
		log.Panicln("udp server 监听失败", err)
	}
	logger.Info("服务器启动成功[%v]", lo)
	return conn
}
