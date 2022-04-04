/*
◆ 注册功能：接收注册请求后，通过HSS使注册请求生效。
◆ 消息流处理：控制已注册的会话终端，可作为Proxy-Server。接收请求后，进行内部处理或转发，也可作为UA，中断或发起SIP事务。
◆ 与业务平台进行交互，提供多媒体业务。
*/
package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type S_CscfEntity struct {
	core   chan *modules.Package
	Points map[string]string
	*Mux
	sCache *Cache
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (s *S_CscfEntity) Init(domain, host string) {
	s.Mux = new(Mux)
	sip.ServerDomain = domain
	sip.ServerIP = strings.Split(host, ":")[0]
	sip.ServerPort, _ = strconv.Atoi(strings.Split(host, ":")[1])
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
				logger.Error("[%v] S-CSCF消息处理失败 %x %v %v", ctx.Value("Entity"), pkg.GetRoute(), string(pkg.GetData()), err)
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
	// 增加Via头部信息
	user := sipreq.Header.From.Username()
	sipreq.Header.MaxForwards.Reduce()
	sipreq.Header.Via.SetReceivedInfo("UDP", fmt.Sprintf("%s:%d", sip.ServerIP, sip.ServerPort))
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		// TODO 根据Request-URI获取对应域，向HSS询问对应域的cscf的IP地址
		if !strings.Contains(sipreq.Header.Authorization, "response") {
			// 首次注册请求，请求S-CSCF拿到用户向量
			s.sCache.setUserRegistReq(ReqPrefix+user, &sipreq)
			user := sipreq.Header.From.Username()
			// 向HSS发起MAR，查询信息
			table := map[string]string{
				"UserName": user,
			}
			pkg.SetFixedConn(s.Points["HSS"])
			pkg.Construct(modules.EPCPROTOCAL, modules.MultiMediaAuthenticationRequest, modules.StrLineMarshal(table))
			modules.Send(pkg, up)
		} else { // 第二次发起注册，进行用户身份验证
			downlink := s.Points["PCSCF"]
			pkg.SetFixedConn(downlink)

			values := parseAuthentication(sipreq.Header.Authorization)
			XRES := s.sCache.getUserRegistXRES(ReqPrefix + user)
			res, err := base64.RawStdEncoding.DecodeString(values["response"])
			if err != nil {
				logger.Error("[%v] 解码response失败: %v", ctx.Value("Entity"), err)
				return err
			}
			RES := hex.EncodeToString(res)
			logger.Warn("[%v] XRES: %v, RES: %v(byte: %x)", ctx.Value("Entity"), XRES, RES, res)
			if XRES != "" && RES == XRES { // 验证通过
				// 用户完成注册后，登记用户信息到系统中
				u := new(User)
				name := sipreq.Header.From.Username()
				u.Domain = sipreq.Header.From.URI.Domain
				u.AccessPoint = sipreq.Header.AccessNetworkInfo
				s.sCache.delUserRegistReqXRES(ReqPrefix + user)
				if err := s.sCache.setUserInfo(UeInfoPrefix+name, u); err != nil {
					// 录入系统失败，注册失败
					sipresp := sip.NewResponse(sip.StatusServerInternalError, &sipreq)
					pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
					modules.Send(pkg, down)
					return err
				}
				// 注册成功
				sipresp := sip.NewResponse(sip.StatusOK, &sipreq)
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
				modules.Send(pkg, down)
			} else { // 验证不通过
				s.sCache.delUserRegistReqXRES(ReqPrefix + user)
				sresp := sip.NewResponse(sip.StatusUnauthorized, &sipreq)
				logger.Info("[%v] 发起对UE鉴权: %v", ctx.Value("Entity"), sresp.String())
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sresp.String())
				modules.Send(pkg, down)
			}
		}
	case sip.MethodInvite, sip.MethodPrack, sip.MethodUpdate:
		// 来自另一个域的请求
		if first, _ := sipreq.Header.Via.FirstAddrInfo(); strings.Contains(first, "i-cscf") {
			// 直接向下行转发
			sipreq.Header.Via.AddServerInfo()
			pkg.SetFixedConn(s.Points["PCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else {
			//
			sipreq.Header.Via.AddServerInfo()
			domain := sipreq.RequestLine.RequestURI.Domain
			user := sipreq.RequestLine.RequestURI.Username
			caller := s.sCache.getUserInfo(UeInfoPrefix + user)
			if caller == nil {
				// 主叫用户在系统中找不到
				sipresp := sip.NewResponse(sip.StatusRequestTerminated, &sipreq)
				pkg.SetFixedConn(s.Points["PCSCF"])
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
				modules.Send(pkg, down)
				return nil
			}
			logger.Warn("caller domain: %v, request domain: %v", caller.Domain, domain)
			// INVITE 回话建立请求，分为 同域 和 不同域
			// 向对应域的ICSCF发起请求
			if caller.Domain == domain { // 同一域 直接返回被叫地址，无需更改无线接入点
				pkg.SetFixedConn(s.Points["PCSCF"])
				pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
				modules.Send(pkg, down)
			} else { // 不同域 查询对应域的ICSCF网络地址，修改无线接入点信息，向对应域发起请求
				pkg.SetFixedConn(s.Points["OTHER"])
				pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
				modules.Send(pkg, up)
			}
		}

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
