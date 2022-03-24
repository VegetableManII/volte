/*
◆ 注册功能：接收注册请求后，通过HSS使注册请求生效。
◆ 消息流处理：控制已注册的会话终端，可作为Proxy-Server。接收请求后，进行内部处理或转发，也可作为UA，中断或发起SIP事务。
◆ 与业务平台进行交互，提供多媒体业务。
*/
package controller

import (
	"bytes"
	"context"
	"strconv"

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
	sCache *Cache
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (s *S_CscfEntity) Init(domain string) {
	s.Mux = new(Mux)
	s.SipVia = "SIP/2.0/UDP s-cscf@" + domain + ";branch="
	s.Points = make(map[string]string)
	s.router = make(map[[2]byte]BaseSignallingT)
	s.sCache = initCache()
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
	sipreq, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		user := sipreq.Header.From.Username()
		// 向HSS发起MAR，查询信息
		table := map[string]string{
			"UserName": user,
		}
		pkg.SetFixedConn(s.Points["HSS"])
		pkg.Construct(modules.EPCPROTOCAL, modules.MultiMediaAuthenticationRequest, modules.StrLineMarshal(table))
		modules.Send(pkg, up)
	case "INVITE":
		// INVITE 回话建立请求，分为 同域 和 不同域
		domain := sipreq.RequestLine.RequestURI.Domain
		user := sipreq.RequestLine.RequestURI.Username
		callee := s.sCache.getUserInfo(user)
		if callee == nil {
			// 被叫用户在系统中找不到
			sipresp := sip.NewResponse(sip.StatusRequestTerminated, &sipreq)
			pkg.SetFixedConn(s.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
			modules.Send(pkg, down)
			return nil
		}
		// 向对应域的ICSCF发起请求
		if callee.Domain == domain { // 同一域 直接返回被叫地址，无需更改无线接入点
			// 添加自身Via标识
			sipreq.Header.Via.Add(s.SipVia + strconv.FormatInt(modules.GenerateSipBranch(), 16))
			pkg.SetFixedConn(s.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else { // 不同域 查询对应域的ICSCF网络地址，修改无线接入点信息，向对应域发起请求
		}
		// 向主叫发起域响应trying
		pkg0 := new(modules.Package)
		sipresp := sip.NewResponse(sip.StatusTrying, &sipreq)
		pkg0.SetFixedConn(s.Points["ICSCF"])
		pkg0.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg0, down)
	}
	return nil
}

func (s *S_CscfEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)
	logger.Info("[%v] Receive From ICSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))

	sipresp, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		// TODO
		return err
	}
	// S-CSCF接收到SIP响应消息则一定是被叫对INVITE请求的响应
	sipresp.Header.Via.RemoveFirst()
	pkg.SetFixedConn(s.Points["ICSCF"])
	pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	modules.Send(pkg, down)
	return nil
}
