package config

import (
	"flag"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

type Node struct {
	VirtualAddr string
	ActualAddr  string
}

var Domain string
var Elements map[string]*Node

func init() {
	var confile string
	flag.StringVar(&Domain, "d", "", "网络域")
	flag.StringVar(&confile, "f", "", "配置文件路径")
	flag.Parse()

	_, err := os.Stat(confile)
	if err != nil || len(os.Args) < 3 {
		flag.Usage()
		os.Exit(0)
	}

	path, err := os.Getwd()
	if err != nil {
		log.Fatal("获取运行目录失败")
	}

	args := strings.Split(os.Args[0], "/")
	pgnm := args[len(args)-1]
	logconf = strings.ReplaceAll(logconf, "#entity", pgnm)
	if runtime.GOOS != "windows" {
		logger.SetLogger(logconf)
	} else {
		logconf = strings.Replace(logconf, "/tmp", ".", 1)
	}

	logger.SetLogPathTrim(path)
	viper.SetConfigFile(confile)
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
	Elements = make(map[string]*Node, 5)
	hssv := viper.GetString(Domain + ".hss.vip")
	hssa := viper.GetString(Domain + ".hss.host")
	Elements["HSS"] = &Node{VirtualAddr: hssv, ActualAddr: hssa}
	scscfv := viper.GetString(Domain + ".s-cscf.vip")
	scscfa := viper.GetString(Domain + ".s-cscf.host")
	Elements["SCSCF"] = &Node{VirtualAddr: scscfv, ActualAddr: scscfa}
	icscfv := viper.GetString(Domain + ".i-cscf.vip")
	icscfa := viper.GetString(Domain + ".i-cscf.host")
	Elements["ICSCF"] = &Node{VirtualAddr: icscfv, ActualAddr: icscfa}
	pcscfv := viper.GetString(Domain + ".p-cscf.vip")
	pcscfa := viper.GetString(Domain + ".p-cscf.host")
	Elements["PCSCF"] = &Node{VirtualAddr: pcscfv, ActualAddr: pcscfa}
	pgwv := viper.GetString(Domain + ".pgw.vip")
	pgwa := viper.GetString(Domain + ".pgw.host")
	Elements["PGW"] = &Node{VirtualAddr: pgwv, ActualAddr: pgwa}
	if Domain == "hebeiyidong" {
		icscfv := viper.GetString("chongqingdianxin.i-cscf.vip")
		icscfa := viper.GetString("chongqingdianxin.i-cscf.host")
		Elements["OTHER"] = &Node{VirtualAddr: icscfv, ActualAddr: icscfa}
	} else {
		icscfv := viper.GetString("hebeiyidong.i-cscf.vip")
		icscfa := viper.GetString("hebeiyidong.i-cscf.host")
		Elements["OTHER"] = &Node{VirtualAddr: icscfv, ActualAddr: icscfa}
	}

}

var logconf string = `{"TimeFormat":"2006-01-02 15:04:05","File": {"filename": "/tmp/logs/#entity.app.log","level": "INFO","daily": true,"maxlines": 1000000,"maxsize": 1,"maxdays": -1,"append": true,"permit": "0660"}}`
