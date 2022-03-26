/*
◆ 注册功能：为用户指定某个S-CSCF来执行SIP注册。
◆ 消息流处理功能：从HSS中获取S-CSCF的地址，转发SIP请求；将其他网络传来的SIP请求路由到S-CSCF。
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

	"github.com/wonderivan/logger"
)

type RegistCombine struct {
	Req  *sip.Message
	XRES string
}
type User struct {
	Domain      string
	AccessPoint string // 接入基站
}

type I_CscfEntity struct {
	Points map[string]string
	*Mux
	iCache *Cache
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (i *I_CscfEntity) Init(domain, host string) {
	i.Mux = new(Mux)
	sip.ServerDomain = domain
	sip.ServerIP = strings.Split(host, ":")[0]
	sip.ServerPort, _ = strconv.Atoi(strings.Split(host, ":")[1])
	i.Points = make(map[string]string)
	i.router = make(map[[2]byte]BaseSignallingT)
	i.iCache = initCache()
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

func (i *I_CscfEntity) SIPREQUESTF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	// 解析SIP消息
	sipreq, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		return err
	}
	// 增加Via头部信息
	user := sipreq.Header.From.Username()
	sipreq.Header.MaxForwards.Reduce()
	sipreq.Header.Via.SetReceivedInfo("UDP", fmt.Sprintf("%s:%d", sip.ServerIP, sip.ServerPort))
	sipreq.Header.Via.AddServerInfo()
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		// TODO 根据Request-URI获取对应域，向HSS询问对应域的cscf的IP地址
		if !strings.Contains(sipreq.Header.Authorization, "response") {
			// 首次注册请求，请求S-CSCF拿到用户向量
			i.iCache.setUserRegistReq("req"+user, &sipreq)
			pkg.SetFixedConn(i.Points["SCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		} else { // 第二次发起注册，进行用户身份验证
			downlink := i.Points["PCSCF"]
			pkg.SetFixedConn(downlink)

			values := parseAuthentication(sipreq.Header.Authorization)
			XRES := i.iCache.getUserRegistXRES("req" + user)
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
				i.iCache.delUserRegistReqXRES(user)
				if err := i.iCache.setUserInfo(name, u); err != nil {
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
				i.iCache.delUserRegistReqXRES(user)
				sresp := sip.NewResponse(sip.StatusUnauthorized, &sipreq)
				logger.Info("[%v] 发起对UE鉴权: %v", ctx.Value("Entity"), sresp.String())
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sresp.String())
				modules.Send(pkg, down)
			}
		}
	case sip.MethodInvite:
		domain := sipreq.RequestLine.RequestURI.Domain
		user := sipreq.RequestLine.RequestURI.Username
		callee := i.iCache.getUserInfo(user)

		logger.Warn("callee domain: %v, request domain: %v", callee.Domain, domain)

		if callee == nil {
			// 被叫用户在系统中找不到
			sipresp := sip.NewResponse(sip.StatusRequestTerminated, &sipreq)
			pkg.SetFixedConn(i.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
			modules.Send(pkg, down)
			return nil
		}
		// INVITE 回话建立请求，分为 同域 和 不同域
		// 向对应域的ICSCF发起请求
		if callee.Domain == domain { // 同一域 直接返回被叫地址，无需更改无线接入点
			// 添加自身Via标识
			sipreq.Header.Via.SetReceivedInfo("UDP", fmt.Sprintf("%s:%d", sip.ServerIP, sip.ServerPort))
			sipreq.Header.Via.AddServerInfo()
			pkg.SetFixedConn(i.Points["PCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else { // 不同域 查询对应域的ICSCF网络地址，修改无线接入点信息，向对应域发起请求
		}
	}
	return nil
}

func (i *I_CscfEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	// 解析SIP消息
	sipresp, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		// TODO 错误处理
		return err
	}
	// 删除Via头部信息
	sipresp.Header.Via.RemoveFirst()
	sipresp.Header.MaxForwards.Reduce()
	// INVITE请求，被叫响应应答
	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	pkg.SetFixedConn(i.Points["PCSCF"])
	pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	modules.Send(pkg, down)
	return nil
}

func (i *I_CscfEntity) MutimediaAuthorizationAnswerF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 获得用户鉴权信息
	resp := modules.StrLineUnmarshal(pkg.GetData())
	user := resp["UserName"]
	AUTN := resp["AUTN"]
	XRES := resp["XRES"]
	RAND := resp["RAND"]
	// 首先获取缓存中的请求
	req, ok := i.iCache.getUserRegistReq("req" + user)
	if !ok {
		// 鉴权请求已过期
		sipresp := sip.NewResponse(sip.StatusGone, req)
		pkg.SetFixedConn(i.Points["PCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
		return errors.New("ErrRequestExpired")
	}
	// 保存用户鉴权
	err := i.iCache.setUserRegistXRES("req"+user, XRES)
	if err != nil {
		sipresp := sip.NewResponse(sip.StatusServerTimeout, req)
		pkg.SetFixedConn(i.Points["PCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
		// 删除注册请求
		i.iCache.delUserRegistReqXRES(user)
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
	pkg.SetFixedConn(i.Points["PCSCF"])
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
