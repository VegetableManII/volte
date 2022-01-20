package controller

import (
	"context"
	"log"

	"github.com/VegetableManII/volte/common"
	"github.com/VegetableManII/volte/parser"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type CscfEntity struct {
}

func (this *CscfEntity) Init() {
}

// HSS可以接收eps电路协议也可以接收SIP协议
func (this *CscfEntity) CoreProcessor(ctx context.Context, in, out chan *common.Package) {
	for {
		select {
		case msg := <-in:
			sip, err := parser.ParseMessage(msg.GetData())
			if err != nil {
				logger.Error("[%v] SIP消息解析失败, %v", ctx.Value("Entity"), err)
			}
			logger.Info("[%v] %v", ctx.Value("Entity"), sip.Short())
			log.Println("GetBody", sip.GetBody())
			log.Println("AllHeaders", sip.AllHeaders())
			log.Println("String", sip.String())
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}
