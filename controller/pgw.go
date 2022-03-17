package controller

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"
	"github.com/patrickmn/go-cache"

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
		case msg := <-in:
			// 兼容心跳包
			if msg.CommonMsg == nil && msg.RemoteAddr != nil && msg.Conn == nil {
				updateUtranAddress(ctx, msg.RemoteAddr, msg.Destation)
			} else {
				f, ok := p.router[msg.GetUniqueMethod()]
				if !ok {
					logger.Error("[%v] PGW不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
					continue
				}
				err = f(ctx, msg, up, down)
				if err != nil {
					logger.Error("[%v] PGW消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
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
	utranCellID := args["UTRAN-CELL-ID-3GPP"]
	// 分配IP地址
	ippoollock.Lock()
	l := len(ippool)
	ip := ippool[l-1]
	ippool = ippool[:l-1]
	ippoollock.Unlock()
	args["IP"] = ip
	// 绑定UE IP和基站的关系
	UeCache.Set(ip, utranCellID, cache.NoExpiration)
	// 响应UE
	modules.EpcMsg(pkg.CommonMsg, modules.EPCPROTOCAL, modules.AttachAccept, args, "eNodeB", pkg.RemoteAddr, pkg.Conn, down)
	return nil
}

func (p *PgwEntity) SIPREQUESTF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))

	host := p.Points["CSCF"]
	modules.ImsMsg(pkg.CommonMsg, modules.SIPPROTOCAL, modules.SipRequest, pkg.GetData(), host, nil, nil, up) // 上行
	return nil
}

func (p *PgwEntity) SIPRESPONSEF(ctx context.Context, pkg *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipresp, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	// 无线接入点
	accPoint := sipresp.Header.AccessNetworkInfo

	var remote *net.UDPAddr
	ra, ok := UeCache.Get(accPoint)
	if !ok {
		return errors.New("ErrNotFoundAPAddr")
	}
	remote = ra.(*net.UDPAddr)
	modules.ImsMsg(pkg.CommonMsg, modules.SIPPROTOCAL, modules.SipResponse, []byte(sipresp.String()),
		"eNodeB", remote, pkg.Conn, down)
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
