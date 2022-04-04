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
	"errors"
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
	Host   string
	Points map[string]string
	*Mux
	sCache *Cache
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (s *S_CscfEntity) Init(domain, host string) {
	s.Mux = new(Mux)
	s.Host = host
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
				"Host":     s.Host,
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
				s.sCache.updateUserInfo(UeInfoPrefix+name, u)
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
		if first, _ := sipreq.Header.Via.FirstAddrInfo(); strings.Contains(first, "s-cscf") {
			// 查询被叫用户，修改无线接入点信息，直接向下行转发
			callee := sipreq.RequestLine.RequestURI.Username
			logger.Warn("[%v] 被叫%v", ctx.Value("Entity"), callee)
			user := s.sCache.getUserInfo(UeInfoPrefix + callee)
			logger.Warn("[%v] 被叫接入点%v", ctx.Value("Entity"), user.AccessPoint)
			sipreq.Header.Via.AddServerInfo()
			sipreq.Header.AccessNetworkInfo = user.AccessPoint
			pkg.SetFixedConn(s.Points["PCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else {
			//
			sipreq.Header.Via.AddServerInfo()
			domain := sipreq.RequestLine.RequestURI.Domain
			user := sipreq.Header.From.URI.Username
			caller := s.sCache.getUserInfo(UeInfoPrefix + user)
			logger.Warn("caller: %v, callee domain: %v", caller, domain)
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
			} else { // 不同域 查询对应域的ICSCF网络地址,向对应域发起请求
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
	// 解析SIP消息
	sipresp, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		// TODO 错误处理
		return err
	}
	via1, _ := sipresp.Header.Via.FirstAddrInfo()
	// 删除Via头部信息
	sipresp.Header.Via.RemoveFirst()
	sipresp.Header.MaxForwards.Reduce()
	via2, _ := sipresp.Header.Via.FirstAddrInfo()
	// 当前跳为s-cscf，下一跳不是s-cscf，则说明响应来自另一个域,更新无线接入点
	if strings.Contains(via1, "s-cscf") && strings.Contains(via2, "s-cscf") {
		caller := sipresp.Header.To.URI.Username
		logger.Warn("[%v] 主叫%v", ctx.Value("Entity"), caller)
		user := s.sCache.getUserInfo(UeInfoPrefix + caller)
		logger.Warn("[%v] 主叫接入点%v", ctx.Value("Entity"), user.AccessPoint)
		sipresp.Header.AccessNetworkInfo = user.AccessPoint
	}
	// 如果下一跳via包含s-cscf说明是另一个域的响应
	if strings.Contains(via2, "s-cscf") {
		logger.Info("[%v] Receive From Other I-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetFixedConn("OTHER")
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, up)
		return nil
	}
	// INVITE请求，被叫响应应答
	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	pkg.SetFixedConn(s.Points["PCSCF"])
	pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	modules.Send(pkg, down)
	return nil
}

func (s *S_CscfEntity) MutimediaAuthorizationAnswerF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 获得用户鉴权信息
	resp := modules.StrLineUnmarshal(pkg.GetData())
	user := resp["UserName"]
	AUTN := resp["AUTN"]
	XRES := resp["XRES"]
	RAND := resp["RAND"]
	// 首先获取缓存中的请求
	req, ok := s.sCache.getUserRegistReq(ReqPrefix + user)
	if !ok {
		// 鉴权请求已过期
		sipresp := sip.NewResponse(sip.StatusGone, req)
		pkg.SetFixedConn(s.Points["PCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
		return errors.New("ErrRequestExpired")
	}
	// 保存用户鉴权
	err := s.sCache.setUserRegistXRES(ReqPrefix+user, XRES)
	if err != nil {
		sipresp := sip.NewResponse(sip.StatusServerTimeout, req)
		pkg.SetFixedConn(s.Points["PCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
		// 删除注册请求
		s.sCache.delUserRegistReqXRES(ReqPrefix + user)
		return err
	}
	// 组装WWW-Authenticate
	autn, _ := hex.DecodeString(AUTN)
	rand, _ := hex.DecodeString(RAND)
	nonce := append(rand, autn...)
	wwwAuth := fmt.Sprintf(`Digest realm=hebeiyidomg.3gpp.net nonce=%s qop=auth-int algorithm=AKAv1-MD5`, base64.StdEncoding.EncodeToString(nonce))
	// 向终端发起鉴权

	sipresp := sip.NewResponse(sip.StatusUnauthorized, req)
	sipresp.Header.WWWAuthenticate = wwwAuth
	// 用户鉴权信息经过s-cscf发起
	// 转发给s-cscf的时候会携带自身的via header
	sipresp.Header.Via.RemoveFirst()
	sipresp.Header.MaxForwards.Reduce()
	pkg.SetFixedConn(s.Points["PCSCF"])
	pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	modules.Send(pkg, down)
	logger.Info("[%v] MAA响应: %v", ctx.Value("Entity"), sipresp.String())
	return nil
}

func parseAuthentication(authHeader string) map[string]string {
	res := make(map[string]string)
	auth := strings.TrimLeft(authHeader, "Digest ")
	logger.Warn("Authorization := %v", auth)
	items := strings.Split(auth, ",")
	for _, item := range items {
		val := strings.Split(item, "=")
		if len(val) >= 2 {
			res[val[0]] = val[1]
		}
	}
	return res
}
