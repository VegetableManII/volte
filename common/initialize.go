package common

import (
	"log"
	"os"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

func init() {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal("获取运行目录失败")
	}
	logger.SetLogPathTrim(path)
	viper.SetConfigType("yml")
	viper.AddConfigPath(".") // 设置配置文件与可执行文件在同一目录可供编译后的程序使用
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
}
