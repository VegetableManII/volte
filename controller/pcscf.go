package controller

import (
	"context"
	"strings"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type P_CscfEntity struct {
	SipURI string
	SipVia string
	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (p *P_CscfEntity) Init(host string) {
	p.Mux = new(Mux)
	p.SipURI = "x-cscf.hebeiyidong.3gpp.net"
	p.SipVia = "SIP/2.0/UDP " + host + ";branch="
	p.Points = make(map[string]string)
	p.router = make(map[[2]byte]BaseSignallingT)
}

func (p *P_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	for {
		select {
		case msg := <-in:
			f, ok := p.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			f(ctx, msg, up, down)
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (p *P_CscfEntity) SIPREQUESTF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From PGW: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(m.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		parseAuthentication("")
	case "INVITE":
		return nil
	}
	return nil
}
