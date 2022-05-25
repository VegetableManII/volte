/*
◆ 注册功能：为用户指定某个S-CSCF来执行SIP注册。
◆ 消息流处理功能：从HSS中获取S-CSCF的地址，转发SIP请求；将其他网络传来的SIP请求路由到S-CSCF。
*/

package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/config"
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
	*Mux
	iCache *Cache
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (i *I_CscfEntity) Init(domain, host string) {
	i.Mux = new(Mux)
	sip.ServerDomain = domain
	sip.ServerIP = strings.Split(host, ":")[0]
	sip.ServerPort, _ = strconv.Atoi(strings.Split(host, ":")[1])
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
				logger.Error("[%v] I-CSCF消息处理失败 %x %v %v", ctx.Value("Entity"), pkg.GetRoute(), string(pkg.GetData()), err)
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
	sipreq.Header.MaxForwards.Reduce()
	sipreq.Header.Via.SetReceivedInfo("UDP", fmt.Sprintf("%s:%d", sip.ServerIP, sip.ServerPort))
	sipreq.Header.Via.AddServerInfo()
	switch sipreq.RequestLine.Method {
	case sip.MethodRegister:
		logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		//根据Request-URI获取对应域，向HSS询问对应域的cscf的IP地址
		user := sipreq.Header.From.Username()
		// 先缓存请求
		i.iCache.setUserRegistReq(UARegPrefix+user, &sipreq)
		// 向HSS发起UAR，查询信息
		table := map[string]string{
			"UserName": user,
		}
		pkg.SetShortConn(config.Elements["HSS"].ActualAddr)
		pkg.Construct(modules.EPCPROTOCAL, modules.UserAuthorizationRequest, modules.StrLineMarshal(table))
		modules.Send(pkg, up)
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
	// 删除第一个Via头部信息
	sipresp.Header.Via.RemoveFirst()
	sipresp.Header.MaxForwards.Reduce()
	via, _ := sipresp.Header.Via.FirstAddrInfo()
	// 判断下一跳是否是s-cscf
	if strings.Contains(via, "s-cscf") {
		// 跨域
		return nil
	}
	logger.Info("[%v][%v] Receive From SCSCF: \n%v", ctx.Value("Entity"), sip.ServerDomain, string(pkg.GetData()))
	pkg.SetShortConn(config.Elements["PCSCF"].ActualAddr)
	pkg.Construct(modules.SIPPROTOCAL, modules.SipResponse, sipresp.String())
	modules.Send(pkg, down)

	return nil
}

func (i *I_CscfEntity) UserAuthorizationAnswerF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	resp := modules.StrLineUnmarshal(pkg.GetData())
	scscf := resp["S-CSCF"]
	user := resp["UserName"]
	sipreq, ok := i.iCache.getUserRegistReq(UARegPrefix + user)
	if !ok {
		logger.Info("[%v] %s's REGISTER Message Not Found or Expired.", ctx.Value("Entity"), user)
		return errors.New("RequestNotFound")
	}
	// 转发给S-CSCF
	pkg.SetShortConn(scscf)
	pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, sipreq.String())
	modules.Send(pkg, up)
	return nil
}
