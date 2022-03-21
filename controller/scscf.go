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

type User struct {
	Address     string
	Name        string
	AccessPoint string // 无线接入点基站
}

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
	sipreq, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		user := sipreq.Header.From.Username()
		uplink := s.Points["HSS"]
		// 向HSS发起MAR，再收到MAA，同步实现
		// 向HSS查询信息
		table := map[string]string{
			"UserName": user,
		}
		pkg.SetFixedConn(uplink)
		pkg.Construct(modules.EPCPROTOCAL, modules.MultiMediaAuthenticationRequest, modules.StrLineMarshal(table))
		modules.Send(pkg, up)
	case "INVITE":
		// 呼叫
		// 同域 和 不同域
		dns := sipreq.RequestLine.RequestURI.Domain
		user := sipreq.RequestLine.RequestURI.Username
		callee := s.sCache.getUserInfo(user)
		if callee == nil {
			// 被叫在系统中找不到
			sipresp := sip.NewResponse(sip.StatusRequestTerminated, &sipreq)
			pkg.SetFixedConn(s.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
			modules.Send(pkg, down)
			return nil
		}
		// 打上自己的Via
		// 向对应域的ICSCF发起请求
		if callee.Address == dns {
			// 同一域 直接返回被叫地址，无需更改无线接入点
			sipreq.Header.Via.Add(s.SipVia + strconv.FormatInt(modules.GenerateSipBranch(), 16))
			pkg.SetFixedConn(s.Points["ICSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, down)
		} else {
			// 询问HSS 对应域的ICSCF地址
			// 修改无线接入点信息，向对应域发起请求
		}
		return nil
	case "PRACK":
		downlink := s.Points["ICSCF"]
		pkg.SetFixedConn(downlink)
		// 用户完成注册后，登记用户信息到系统中
		u := new(User)
		u.Name = sipreq.Header.From.Username()
		u.Address = sipreq.Header.From.URI.Domain
		u.AccessPoint = sipreq.Header.AccessNetworkInfo
		if err := s.sCache.setUserInfo(u.Name, u); err != nil {
			// 注册失败
			sipresp := sip.NewResponse(sip.StatusServerInternalError, &sipreq)
			pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
			modules.Send(pkg, down)
			return err
		}
		// 注册成功
		sipresp := sip.NewResponse(sip.StatusOK, &sipreq)
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, down)
	}
	return nil
}
