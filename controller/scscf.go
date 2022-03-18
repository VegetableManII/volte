package controller

import (
	"context"
	"strings"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type S_CscfEntity struct {
	SipVia string
	core   chan *modules.Package

	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (s *S_CscfEntity) Init(host string) {
	s.Mux = new(Mux)
	s.SipVia = "SIP/2.0/UDP " + host + ";branch="
	s.Points = make(map[string]string)
	s.router = make(map[[2]byte]BaseSignallingT)

}

func (s *S_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *modules.Package) {
	s.core = in
	for {
		select {
		case pkg := <-in:
			f, ok := s.router[pkg.GetRoute()]
			if !ok {
				logger.Error("[%v] S-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), pkg.GetRoute())
				continue
			}
			err := f(ctx, pkg, up, down)
			if err != nil {
				logger.Error("[%v] P-CSCF消息处理失败 %x %v %v", ctx.Value("Entity"), pkg.GetRoute(), string(pkg.GetData()), err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] S-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (s *S_CscfEntity) SIPREQUESTF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From ICSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		user := sipreq.Header.From.Username()
		downlink := s.Points["ICSCF"]
		uplink := s.Points["HSS"]
		// 向HSS发起MAR，再收到MAA，同步实现
		// 向HSS查询信息
		table := map[string]string{
			"UserName": user,
		}
		pkg.SetFixedConn(uplink)
		pkg.Construct(modules.EPCPROTOCAL, modules.MultiMediaAuthenticationRequest, modules.StrLineMarshal(table))
		resp, err := modules.MARSyncRequest(ctx, pkg)
		if err != nil {
			logger.Error("[%v] HSS Response Error %v", ctx.Value("Entity"), err)
			sipresp := sip.NewResponse(sip.StatusNoResponse, &sipreq)
			pkg.SetFixedConn(s.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
			modules.Send(pkg, down)
		} else {
			pkg.SetFixedConn(downlink)
			pkg.Construct(modules.EPCPROTOCAL, modules.MultiMediaAuthenticationAnswer, resp)
			modules.Send(pkg, down)
		}
	case "INVITE":
		return nil
	}
	return nil
}
