package main

import (
	"epc/common"
	"log"
	"net"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

var (
	mmeConn, hssConn *net.UDPConn
)

func init() {
	viper.SetConfigName("config.yml")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	host := viper.GetString("EPC.mme.host")
	hssHost := viper.GetString("EPC.hss.host")
	logger.Info("配置文件读取成功", "")
	// 启动 MME 的UDP服务器
	mmeConn = common.InitServer(host)
	// 创建连接 HSS 的客户端
	hssConn = common.ConnectEPC(hssHost)
}
