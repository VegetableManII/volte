package controller

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type I_CscfEntity struct {
	SipVia string
	Points map[string]string
	*Mux
	userAuthCache map[string]string
	authMutex     sync.Mutex
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (i *I_CscfEntity) Init(host string) {
	i.Mux = new(Mux)
	i.SipVia = "SIP/2.0/UDP " + host + ";branch="
	i.Points = make(map[string]string)
	i.router = make(map[[2]byte]BaseSignallingT)
	i.userAuthCache = make(map[string]string)
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
	sipreq.Header.Via.Add(i.SipVia + strconv.FormatInt(modules.GenerateSipBranch(), 16))
	sipreq.Header.MaxForwards.Reduce()
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		user := sipreq.Header.From.Username()
		// 查看本地是否存在鉴权缓存
		i.authMutex.Lock()
		RES, ok := i.userAuthCache[user]
		i.authMutex.Unlock()
		if !ok {
			// 首次注册请求，请求S-CSCF拿到用户向量
			pkg.SetFixedConn(i.Points["SCSCF"])
			pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
			modules.Send(pkg, up)
		} else {
			values := parseAuthentication(sipreq.Header.Authorization)

			if RES != "" && RES == values["response"] {
				sresp := sip.NewResponse(sip.StatusOK, &sipreq)
				logger.Info("[%v] 鉴权认证成功: %v", ctx.Value("Entity"), sresp.String())
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sresp.String())
			} else {
				i.authMutex.Lock()
				delete(i.userAuthCache, user)
				i.authMutex.Unlock()
				sresp := sip.NewResponse(sip.StatusUnauthorized, &sipreq)
				logger.Info("[%v] 发起对UE鉴权: %v", ctx.Value("Entity"), sresp.String())
				pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sresp.String())
			}
			pkg.SetFixedConn(i.Points["PCSCF"])
			modules.Send(pkg, down)
		}
	case sip.MethodInvite:

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

func (i *I_CscfEntity) MutimediaAuthorizationAnswerF(ctx context.Context, m *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From S-CSCF: \n%v", ctx.Value("Entity"), string(m.GetData()))
	// 获得用户鉴权信息
	resp := modules.StrLineUnmarshal(m.GetData())
	user := resp["UserName"]
	AUTN := resp["AUTN"]
	XRES := resp["XRES"]
	RAND := resp["RAND"]
	// 保存用户鉴权
	i.authMutex.Lock()
	i.userAuthCache[user] = XRES
	i.authMutex.Unlock()

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
