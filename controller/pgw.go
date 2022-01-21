package controller

import (
	"context"
	"sync"

	"github.com/VegetableManII/volte/common"

	"github.com/wonderivan/logger"
)

type PgwEntity struct {
	*Mux
	Points map[string]string
	sync.Mutex
}

func (this *PgwEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	this.Points = make(map[string]string)
}

func (this *PgwEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	var err error
	for {
		select {
		case msg := <-in:
			f, ok := this.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] PGW不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
			}
			err = f(ctx, msg, up, down)
			if err != nil {
				logger.Error("[%v] PGW消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] PGW逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (this *PgwEntity) SIPREGISTERF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	logger.Info("[%v] Receive From eNodeB: %v", ctx.Value("Entity"), string(m.GetData()))
	this.Lock()
	host := this.Points["CSCF"]
	this.Unlock()
	common.RawPackageOut(common.SIPPROTOCAL, common.REGISTER, m.GetData(), host, up) // 上行
	return nil
}
