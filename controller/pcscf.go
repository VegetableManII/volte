package controller

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type P_CscfEntity struct {
	SipVia string
	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (p *P_CscfEntity) Init(domain string) {
	p.Mux = new(Mux)
	p.SipVia = "SIP/2.0/UDP p-cscf@" + domain + ";branch="
	p.Points = make(map[string]string)
	p.router = make(map[[2]byte]BaseSignallingT)
}

func (p *P_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *modules.Package) {
	for {
		select {
		case pkg := <-in:
			f, ok := p.router[pkg.GetRoute()]
			if !ok {
				logger.Error("[%v] P-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), pkg)
				continue
			}
			err := f(ctx, pkg, up, down)
			if err != nil {
				logger.Error("[%v] P-CSCF消息处理失败 %x %v %v", ctx.Value("Entity"), pkg.GetRoute(), string(pkg.GetData()), err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] P-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (p *P_CscfEntity) SIPREQUESTF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	// 解析SIP消息
	sipreq, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		// TODO 错误处理
		return err
	}
	branch := strconv.FormatInt(modules.GenerateSipBranch(), 16)
	sipreq.Header.MaxForwards.Reduce()
	// 增加Via头部信息
	sipreq.Header.Via.Add(p.SipVia + branch)

	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		logger.Info("[%v] Receive From PGW: \n%v", ctx.Value("Entity"), string(pkg.GetData()))

		// 检查头部内容是否首次注册
		if strings.Contains(sipreq.Header.Authorization, "response") { // 包含响应内容则为第二次注册请求
			pkg.SetFixedConn(p.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		} else {
			// 第一次注册请求，P-CSCF处理，填充Authorization头部
			user := sipreq.Header.From.URI.Username
			domain := sipreq.Header.From.URI.Domain
			username := user + "@" + domain
			auth := fmt.Sprintf("Digest username=%s integrity protection:no", username)
			sipreq.Header.Authorization = auth
			// 第一次注册请求SCSCF还未与UE绑定所以转发给ICSCF
			pkg.SetFixedConn(p.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		}
	case sip.MethodInvite:
		via, _ := sipreq.Header.Via.FirstAddrInfo()
		if strings.Contains(via, "i-cscf") { // INVITE请求来自ICSCF
			logger.Info("[%v] Receive From ICSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
			// 向下行转发请求
			pkg.SetFixedConn(p.Points["PCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else { // INVITE请求来自PGW
			logger.Info("[%v] Receive From PGW: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
			// 向上行转发请求
			pkg.SetFixedConn(p.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		}
	}
	return nil
}

func (p *P_CscfEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	// 解析SIP消息
	sipresp, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		// TODO 错误处理
		return err
	}
	// 删除第一个Via头部信息
	sipresp.Header.Via.RemoveFirst()
	sipresp.Header.MaxForwards.Reduce()
	if sipresp.ResponseLine.StatusCode == sip.StatusSessionProgress.Code { // 来自下行PGW的回话建立响应
		logger.Info("[%v] Receive From PGW: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetFixedConn(p.Points["ICSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, up)
	} else { // 来自上行ICSCF的一般响应
		logger.Info("[%v] Receive From ICSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetFixedConn(p.Points["PGW"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
	}
	return nil
}
