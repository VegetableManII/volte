package controller

import (
	"context"
	"sync"

	"github.com/VegetableManII/volte/modules"

	"github.com/wonderivan/logger"
)

type PgwEntity struct {
	*Mux
	Points map[string]string
}

func (this *PgwEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	this.Points = make(map[string]string)
}

func (p *PgwEntity) CoreProcessor(ctx context.Context, in, up, down chan *modules.Package) {
	var err error
	for {
		select {
		case pkg := <-in:
			// 兼容心跳包
			if pkg.IsBeatHeart() {
				updateAddress(pkg.GetDynamicAddr(), pkg.GetFixedConn())
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

// 附着请求，携带IMSI，和客户端支持的加密方法，拿到IMSI向HSS发起Authentication Informat Request请求
func (p *PgwEntity) AttachRequestF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)
	logger.Info("[%v] Receive From MME(ENB): \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	data := pkg.GetData()
	args := modules.StrLineUnmarshal(data)
	// 获取eNodeB-ID
	AP := args["UTRAN-CELL-ID-3GPP"]
	// 分配IP地址
	ippoollock.Lock()
	l := len(ippool)
	ip := ippool[l-1]
	ippool = ippool[:l-1]
	ippoollock.Unlock()
	args["IP"] = ip
	// 绑定UE IP和基站的关系
	bindUeWithAP(ip, AP)
	// 响应UE
	addr, con := getAP(pkg)
	pkg.SetFixedConn("eNodeB")
	pkg.SetDynamicConn(addr, con)
	pkg.Construct(modules.EPCPROTOCAL, modules.AttachAccept, modules.StrLineMarshal(args))
	modules.Send(pkg, down)
	return nil
}

func (p *PgwEntity) SIPREQUESTF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))

	pkg.SetFixedConn(p.Points["CSCF"])
	pkg.Construct(modules.SIPPROTOCAL, modules.SipRequest, "")
	modules.Send(pkg, up) // 上行
	return nil
}

func (p *PgwEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// INVITE 请求寻找无线接入点
	// accPoint := sipresp.Header.AccessNetworkInfo

	// var remote *net.UDPAddr
	// ra, ok := Cache.Get(accPoint)
	// if !ok {
	// 	return errors.New("ErrNotFoundAPAddr")
	// }

	addr, con := getAP(pkg)
	pkg.SetFixedConn("eNodeB")
	pkg.SetDynamicConn(addr, con)
	modules.Send(pkg, down)
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
