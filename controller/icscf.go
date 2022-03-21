package controller

import (
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
type RequestCache struct {
	Type   string
	Branch string
	Req    *sip.Message
}

type I_CscfEntity struct {
	SipVia string
	Points map[string]string
	*Mux
	iCache *Cache
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (i *I_CscfEntity) Init(domain string) {
	i.Mux = new(Mux)
	i.SipVia = "SIP/2.0/UDP i-cscf@" + domain + ";branch="
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

	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	// 增加Via头部信息
	branch := strconv.FormatInt(modules.GenerateSipBranch(), 16)
	sipreq.Header.MaxForwards.Reduce()
	user := sipreq.Header.From.Username()
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		// 查看本地是否存在鉴权缓存
		_, ok := i.iCache.getUserRegistReq(user)
		if !ok {
			sipreq.Header.Via.Add(i.SipVia + branch)
			// 首次注册请求，请求S-CSCF拿到用户向量
			pkg.SetFixedConn(i.Points["SCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		} else {
			values := parseAuthentication(sipreq.Header.Authorization)
			RES := i.iCache.getUserRegistXRES(user)
			if RES != "" && RES == values["response"] {
				// 鉴权成功，通知SCSCF保存该用户信息，先缓存
				err := i.iCache.addRequestCache(branch, sip.MethodRegister, branch, &sipreq)
				if err != nil {
					sipresp := sip.NewResponse(sip.StatusGone, &sipreq)
					pkg.SetFixedConn(i.Points["PCSCF"])
					pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
					modules.Send(pkg, down)
					// 删除注册请求
					i.iCache.delUserRegistReqXRES(user)
					return err
				}
				// 使用注册请求内容新建消息报
				sipreq.RequestLine.Method = "PRACK"
				sipreq.Header.Via.Add(i.SipVia + branch)
				pkg.SetFixedConn(i.Points["SCSCF"])
				pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
				modules.Send(pkg, up)
			} else {
				i.iCache.delUserRegistReqXRES(user)
				sresp := sip.NewResponse(sip.StatusUnauthorized, &sipreq)
				logger.Info("[%v] 发起对UE鉴权: %v", ctx.Value("Entity"), sresp.String())
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sresp.String())
			}
			pkg.SetFixedConn(i.Points["PCSCF"])
			modules.Send(pkg, down)
		}
	case sip.MethodInvite:
		// 检查第一个Via是否是SCSCF的标识
		via, _ := sipreq.Header.Via.FirstAddrInfo()
		if strings.Contains(via, "s-cscf") {
			// 被叫 INVITE 请求
			// 添加自身标识
			sipreq.Header.Via.Add(i.SipVia + branch)
			// 向下级请求
			pkg.SetFixedConn(i.Points["PCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
			return nil
		}
		// 建立通话请求首先缓存该请求
		err := i.iCache.addRequestCache(branch, sip.MethodInvite, branch, &sipreq)
		if err != nil {
			sipresp := sip.NewResponse(sip.StatusBusyHere, &sipreq)
			pkg.SetFixedConn(i.Points["PCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
			modules.Send(pkg, down)
			return err
		}
		// 然后转发给S-CSCF
		sipreq.Header.Via.Add(i.SipVia + branch)
		pkg.SetFixedConn(i.Points["SCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
		modules.Send(pkg, up)
	}
	return nil
}

func (i *I_CscfEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From S-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipresp, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	branch := sipresp.Header.Via.TransactionBranch()
	req := i.iCache.getRequestCache(branch)
	if req == nil { // 根据branch无法找对对应的请求则丢弃
		logger.Warn("[%v] 未找对该分支对应请求 branch=%v", ctx.Value("Entity"), branch)
		sipresp := sip.NewResponse(sip.StatusCallTransactionDoesNotExist, req.Req)
		pkg.SetFixedConn(i.Points["PCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
		return nil
	}
	switch req.Type {
	case sip.MethodRegister:
		// 删除Via头部信息
		sipresp.Header.Via.RemoveFirst()
		sipresp.Header.MaxForwards.Reduce()
		pkg.SetFixedConn(i.Points["PCSCF"])
		if sipresp.ResponseLine.StatusCode == sip.StatusOK.Code {
			// 注册完成
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, "")
			modules.Send(pkg, down)
			return nil
		}
		// 注册失败
		user := sipresp.Header.To.Username()
		// 删除注册请求
		i.iCache.delUserRegistReqXRES(user)
		// 响应UE 注册失败
		sipresp.ResponseLine.StatusCode = sip.StatusBusyHere.Code
		sipresp.ResponseLine.ReasonPhrase = sip.StatusBusyHere.Reason
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	case sip.MethodInvite:

	}
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
	req, ok := i.iCache.getUserRegistReq(user)
	if !ok {
		// 鉴权请求已过期
		sipresp := sip.NewResponse(sip.StatusGone, req)
		pkg.SetFixedConn(i.Points["PCSCF"])
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
		return errors.New("ErrRequestExpired")
	}
	// 保存用户鉴权
	err := i.iCache.setUserRegistXRES(user, XRES)
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
	items := strings.Split(authHeader, " ")
	for _, item := range items {
		val := strings.Split(item, "=")
		if len(val) >= 2 {
			res[val[0]] = val[1]
		}
	}
	return res
}
