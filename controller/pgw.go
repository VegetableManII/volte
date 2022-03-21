/*
	PGW的主要功能：
	1、接收UE附着请求，并为UE分配该域的IP地址
	2、向上行链路PCSCF转发SIP消息
	3、根据SIP消息中的P-Access-Network-Info，向下行链路转发SIP消息
*/
package controller

import (
	"bytes"
	"context"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type PgwEntity struct {
	*Mux
	Points   map[string]string
	DHCPConf string
	pCache   *Cache
}

func (p *PgwEntity) Init(dhcp string) {
	// 初始化路由
	p.Mux = new(Mux)
	p.router = make(map[[2]byte]BaseSignallingT)
	p.Points = make(map[string]string)
	p.DHCPConf = dhcp
	p.pCache = initCache()
	// 初始化IP地址池子

}

func (p *PgwEntity) CoreProcessor(ctx context.Context, in, up, down chan *modules.Package) {
	var err error
	for {
		select {
		case pkg := <-in:
			// 兼容心跳包
			if pkg.IsBeatHeart() {
				p.pCache.updateAddress(pkg.GetDynamicAddr(), pkg.GetFixedConn())
			} else {
				f, ok := p.router[pkg.GetRoute()]
				if !ok {
					logger.Error("[%v] PGW不支持的消息类型数据 %v", ctx.Value("Entity"), pkg)
					continue
				}
				err = f(ctx, pkg, up, down)
				if err != nil {
					logger.Error("[%v] PGW消息处理失败 %v %v", ctx.Value("Entity"), pkg, err)
				}
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] PGW逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

// 附着请求
func (p *PgwEntity) AttachRequestF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)
	logger.Info("[%v] Receive From MME(ENB): \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	data := pkg.GetData()
	args := modules.StrLineUnmarshal(data)
	// 分配IP地址
	ippoollock.Lock()
	l := len(ippool)
	ip := ippool[l-1]
	ippool = ippool[:l-1]
	ippoollock.Unlock()
	args["IP"] = ip
	// Attach过程仅仅是基站和PGW的交互过程消息体可以直接保存基站的网络连接
	// 接收Attach消息时，消息体携带基站的网络连接，所以无需通过基站标识从缓存中查找
	pkg.Construct(modules.EPCPROTOCAL, modules.AttachAccept, modules.StrLineMarshal(args))
	modules.Send(pkg, down)
	return nil
}

func (p *PgwEntity) SIPREQUESTF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	sipreq, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		return err
	}
	via, _ := sipreq.Header.Via.FirstAddrInfo()
	if strings.Contains(via, "p-cscf") { // 来自上行发起的INVITE请求
		logger.Info("[%v] Receive From PCSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		// 根据INVITE请求中的P-Access-Network-Info从缓存中获取对应基站的网络连接
		utran := sipreq.Header.AccessNetworkInfo
		raddr := p.pCache.getAddress(utran)
		if raddr != nil {
			// 向基站转发INVITE请求
			pkg.SetDynamicAddr(raddr)
			modules.Send(pkg, down) // 下行
		}
	} else { // 来自下行发起的REGISTER请求
		logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetFixedConn(p.Points["CSCF"])
		modules.Send(pkg, up) // 上行
	}
	return nil
}

func (p *PgwEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	sipresp, err := sip.NewMessage(bytes.NewReader(pkg.GetData()))
	if err != nil {
		// TODO 失败处理
		return err
	}
	if sipresp.ResponseLine.StatusCode == sip.StatusSessionProgress.Code {
		logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetFixedConn(p.Points["PCSCF"])
		modules.Send(pkg, up)
	} else {
		logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		// 请求寻找无线接入点
		utran := strings.Trim(sipresp.Header.AccessNetworkInfo, " ")
		raddr := p.pCache.getAddress(utran)

		pkg.SetFixedConn("eNodeB")
		pkg.SetDynamicAddr(raddr)
		modules.Send(pkg, down)
	}
	return nil
}

var ippool = []string{
	"10.10.10.1",
	"10.10.10.2",
	"10.10.10.3",
	"10.10.10.4",
	"10.10.10.5",
	"10.10.10.6",
	"10.10.10.7",
	"10.10.10.8",
	"10.10.10.9",
	"10.10.10.10",
	"10.10.10.11",
}
var ippoollock sync.Mutex
