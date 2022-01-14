package controller

import (
	"context"
	"volte/common"

	"github.com/wonderivan/logger"
)

type HssEntity struct {
	*Mux
}

func (this *HssEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
}

// HSS可以接收eps电路协议也可以接收SIP协议
func (this *HssEntity) CoreProcessor(ctx context.Context, in, out chan *common.Msg) {
	var err error
	var f BaseSignallingT
	for {
		select {
		case msg := <-in:
			if msg.Type == common.EPSPROTOCAL {
				f = this.router[msg.GetUniqueMethod()]
			} else {
				f = this.router[msg.GetUniqueMethod()]
			}
			err = f(ctx, msg, out)
			if err != nil {
				logger.Error("[%v] MME消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
			}
		}
	}
}
