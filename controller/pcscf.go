/*
◆ 转发UE发来的SIP注册请求给I-CSCF，由UE提供的域名决定I-CSCF；
◆ 转发UE发来的SIP消息给S-CSCF，由P-CSCF在UE发起注册流程时确定S-CSCF。
*/

package controller

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/config"
	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type P_CscfEntity struct {
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (p *P_CscfEntity) Init(domain, host string) {
	p.Mux = new(Mux)
	sip.ServerDomain = domain
	sip.ServerIP = strings.Split(host, ":")[0]
	sip.ServerPort, _ = strconv.Atoi(strings.Split(host, ":")[1])
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
	sipreq.Header.MaxForwards.Reduce()
	sipreq.Header.Via.SetReceivedInfo("UDP", fmt.Sprintf("%s:%d", sip.ServerIP, sip.ServerPort))
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		logger.Info("[%v] Receive From PGW: \n%v", ctx.Value("Entity"), string(pkg.GetData()))

		sipreq.Header.Via.AddServerInfo()
		// 检查头部内容是否首次注册
		if strings.Contains(sipreq.Header.Authorization, "response") { // 包含响应内容则为第二次注册请求
			pkg.SetShortConn(config.Elements["ICSCF"].ActualAddr)
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
			pkg.SetShortConn(config.Elements["ICSCF"].ActualAddr)
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		}
	case sip.MethodInvite, sip.MethodPrack, sip.MethodUpdate:
		via, _ := sipreq.Header.Via.FirstAddrInfo()
		if strings.Contains(via, "s-cscf") { // INVITE请求来自SCSCF
			logger.Info("[%v][%v] Receive From SCSCF: \n%v", ctx.Value("Entity"), sip.ServerDomain, string(pkg.GetData()))
			// 向下行转发请求
			sipreq.Header.Via.AddServerInfo()
			pkg.SetShortConn(config.Elements["PGW"].ActualAddr)
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else { // INVITE请求来自PGW
			logger.Info("[%v][%v] Receive From PGW: \n%v", ctx.Value("Entity"), sip.ServerDomain, string(pkg.GetData()))
			sipreq.Header.Via.AddServerInfo()
			// 向上行转发请求
			pkg.SetShortConn(config.Elements["SCSCF"].ActualAddr)
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
			// 向主叫响应trying
			if sipreq.RequestLine.Method == sip.MethodInvite {
				sipresp := sip.NewResponse(sip.StatusTrying, &sipreq)
				pkg0 := new(modules.Package)
				pkg0.SetShortConn(config.Elements["PGW"].ActualAddr)
				pkg0.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
				modules.Send(pkg0, down)
			}
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
	// 判断下一跳是否是s-cscf
	via, _ := sipresp.Header.Via.FirstAddrInfo()
	if strings.Contains(via, "s-cscf") {
		logger.Info("[%v][%v] Receive From PGW: \n%v", ctx.Value("Entity"), sip.ServerDomain, string(pkg.GetData()))
		pkg.SetShortConn(config.Elements["SCSCF"].ActualAddr)
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, up)
	} else { // 来自上行ICSCF的一般响应
		logger.Info("[%v][%v] Receive From SCSCF: \n%v", ctx.Value("Entity"), sip.ServerDomain, string(pkg.GetData()))
		pkg.SetShortConn(config.Elements["PGW"].ActualAddr)
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
	}
	return nil
}
