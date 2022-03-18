package controller

import (
	"context"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type I_CscfEntity struct {
	SipURI string
	SipVia string
	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (i *I_CscfEntity) Init(host string) {
	i.Mux = new(Mux)
	i.SipURI = "i-cscf.hebeiyidong.3gpp.net"
	i.SipVia = "SIP/2.0/UDP " + host + ";branch="
	i.Points = make(map[string]string)
	i.router = make(map[[2]byte]BaseSignallingT)
}

func (i *I_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *modules.Package) {
	for {
		select {
		case pkg := <-in:
			f, ok := i.router[pkg.GetRoute()]
			if !ok {
				logger.Error("[%v] I-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), pkg)
				continue
			}
			err := f(ctx, pkg, up, down)
			if err != nil {
				logger.Error("[%v] P-CSCF消息处理失败 %x %v %v", ctx.Value("Entity"), pkg.GetRoute(), string(pkg.GetData()), err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] I-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (i *I_CscfEntity) SIPREQUESTF(ctx context.Context, p *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(p.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(p.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		// TODO 如果SIP消息中没有S-CSCF的路由则询问HSS
		// TODO	I-CSCF询问HSS得到S-CSCF列表然后选择转发给S-CSCF
		// 增加Via头部信息
		sipreq.Header.Via.Add(i.SipVia + strconv.FormatInt(modules.GenerateSipBranch(), 16))
		sipreq.Header.MaxForwards.Reduce()
		p.SetFixedConn(i.Points["SCSCF"])
		p.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
		modules.Send(p, up)
	case "INVITE":
		return nil
	}
	return nil
}

func (i *I_CscfEntity) SIPRESPONSEF(ctx context.Context, p *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From S-CSCF: \n%v", ctx.Value("Entity"), string(p.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(p.GetData())))
	if err != nil {
		return err
	}
	// 增加说明支持的SIP请求方法

	// 删除Via头部信息
	sipreq.Header.Via.RemoveFirst()
	sipreq.Header.MaxForwards.Reduce()
	p.SetFixedConn(i.Points["PCSCF"])
	p.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipreq.String())
	modules.Send(p, down)
	return nil
}
