/*
◆ 注册功能：为用户指定某个S-CSCF来执行SIP注册。
◆ 消息流处理功能：从HSS中获取S-CSCF的地址，转发SIP请求；将其他网络传来的SIP请求路由到S-CSCF。
*/

package controller

import (
	"bytes"
	"context"
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
	// user := sipreq.Header.From.Username()
	// sipreq.Header.MaxForwards.Reduce()
	// sipreq.Header.Via.SetReceivedInfo("UDP", fmt.Sprintf("%s:%d", sip.ServerIP, sip.ServerPort))
	// sipreq.Header.Via.AddServerInfo()
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		// TODO
	case sip.MethodInvite, sip.MethodPrack, sip.MethodUpdate:

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
	via, _ := sipresp.Header.Via.FirstAddrInfo()
	logger.Warn("ICSCF@@@@@@@@@first: %v, server: %v", via, sip.ServerDomainHost())
	// 删除Via头部信息
	sipresp.Header.Via.RemoveFirst()
	sipresp.Header.MaxForwards.Reduce()
	// 如果下一个via包含s-cscf说明是另一个域的响应
	if strings.Contains(via, "s-cscf") {
		logger.Info("[%v] Receive From Other I-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetFixedConn("127.0.0.1:54322")
		pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
		modules.Send(pkg, up)
		return nil
	}
	// INVITE请求，被叫响应应答
	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	pkg.SetFixedConn(i.Points["PCSCF"])
	pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	modules.Send(pkg, down)
	return nil
}
