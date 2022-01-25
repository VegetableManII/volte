package common

import (
	"flag"
	"log"
	"os"

	"github.com/spf13/viper"
	"github.com/wonderivan/logger"
)

func init() {
	var confile string
	flag.StringVar(&confile, "f", "", "配置文件路径")
	flag.Parse()

	path, err := os.Getwd()
	if err != nil {
		log.Fatal("获取运行目录失败")
	}
	logger.SetLogger(logconf)
	logger.SetLogPathTrim(path)
	viper.SetConfigFile(confile)
	if e := viper.ReadInConfig(); e != nil {
		log.Panicln("配置文件读取失败", e)
	}
}

var logconf string = `{
    "TimeFormat":"2006-01-02 15:04:05", // 输出日志开头时间格式
    "Console": {            // 控制台日志配置
        "level": "TRAC",    // 控制台日志输出等级
        "color": true       // 控制台日志颜色开关 
    },
    "File": {                   // 文件日志配置
        "filename": "/tmp/#entity.app.log",  // 初始日志文件名
        "level": "TRAC",        // 日志文件日志输出等级
        "daily": true,          // 跨天后是否创建新日志文件，当append=true时有效
        "maxlines": 1000000,    // 日志文件最大行数，当append=true时有效
        "maxsize": 1,           // 日志文件最大大小，当append=true时有效
        "maxdays": -1,          // 日志文件有效期
        "append": true,         // 是否支持日志追加
        "permit": "0660"        // 新创建的日志文件权限属性
    }
}`
