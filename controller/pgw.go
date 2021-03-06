/*
	PGW的主要功能：
	1、区分上下行数据，并做出转发动作
*/
package controller

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/config"
	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type Pool struct {
	CurIP  uint32
	Mask   uint32
	LastIP uint32
	sync.Mutex
}

type PgwEntity struct {
	*Mux
	pool   *Pool
	pCache *Cache
}

func initpool(cidr string) *Pool {
	_, net, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ip := binary.BigEndian.Uint32(net.IP)
	mask := binary.BigEndian.Uint32(net.Mask)
	last := ip | ^mask
	return &Pool{
		CurIP:  ip,
		Mask:   mask,
		LastIP: last,
	}
}

func (p *PgwEntity) Init(dhcp string) {
	// 初始化路由
	p.Mux = new(Mux)
	p.router = make(map[[2]byte]BaseSignallingT)
	p.pool = initpool(dhcp)
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
				p.pCache.updateAddress(AddrPrefix+pkg.GetShortConn(), pkg.GetLongConnAddr())
				logger.Info("心跳%v", AddrPrefix+pkg.GetShortConn())
			} else {
				f, ok := p.router[pkg.GetRoute()]
				if !ok {
					logger.Error("[%v] PGW不支持的消息类型数据 %x %v", ctx.Value("Entity"), pkg.GetRoute(), pkg.GetLongConnAddr().String())
					continue
				}
				err = f(ctx, pkg, up, down)
				if err != nil {
					logger.Error("[%v] PGW消息处理失败 %x %v %v %v", pkg.GetRoute(), pkg.GetLongConnAddr().String(), string(pkg.GetData()), err)
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
	ip, err := p.getIP()
	if err != nil {
		return err
	}
	args["IP"] = ip.String()
	enb := args["UTRAN-CELL-ID-3GPP"]
	p.pCache.updateAddress(AddrPrefix+enb, pkg.GetLongConnAddr())
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
	utran := sipreq.Header.AccessNetworkInfo
	logger.Info("接入点 %v", utran)
	raddr := p.pCache.getAddress(AddrPrefix + utran)
	// 判断来自上游节点还是下游节点
	logger.Info("%v %v %v", AddrPrefix+utran, pkg.GetLongConnAddr().String(), raddr.String())
	if pkg.GetLongConnAddr().String() != raddr.String() {
		// 来自上游节点，向下游转发
		logger.Info("[%v] Receive From PCSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetLongAddr(raddr)
		modules.Send(pkg, down) // 下行
	} else {
		// 来自下游节点，向上游转发
		logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetShortConn(config.Elements["PCSCF"].ActualAddr)
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
	utran := sipresp.Header.AccessNetworkInfo
	logger.Info("接入点 %v", utran)
	raddr := p.pCache.getAddress(AddrPrefix + utran)
	// 判断来自上游节点还是下游节点
	if pkg.GetLongConnAddr().String() != raddr.String() {
		// 来自上游，向下游转发
		logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		// 请求寻找无线接入点
		utran := strings.Trim(sipresp.Header.AccessNetworkInfo, " ")
		raddr := p.pCache.getAddress(AddrPrefix + utran)
		pkg.SetLongAddr(raddr)
		modules.Send(pkg, down)
	} else {
		logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
		pkg.SetShortConn(config.Elements["PCSCF"].ActualAddr)
		modules.Send(pkg, up)
	}
	return nil
}

func (p *PgwEntity) getIP() (net.IP, error) {
	p.pool.Lock()
	cur := p.pool.CurIP
	last := p.pool.LastIP
	p.pool.Unlock()
	cur = cur + 1
	if cur == last {
		return nil, errors.New("ErrNotEnoughIP")
	}
	ip := make([]byte, 4)
	binary.BigEndian.PutUint32(ip, cur)
	return ip, nil
}
