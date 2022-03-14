package controller

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"
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

func (p *PgwEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
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

func (p *PgwEntity) CreateSessionRequestF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)
	logger.Info("[%v] Receive From MME: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	data := pkg.GetData()
	args := common.StrLineUnmarshal(data)
	// 获取eNodeB-ID
	utranCellID := args["UTRAN-CELL-ID-3GPP"]
	// 分配IP地址
	ippoollock.Lock()
	l := len(ippool)
	ip := ippool[l-1]
	ippool = ippool[:l-1]
	ippoollock.Unlock()
	args["IP"] = ip
	delete(args, "QCI")
	var err error
	utran, ok := UeCache.Get(utranCellID)
	if !ok { // 不存在该无线接入点的缓存
		val := make(map[string]struct{})
		val[ip] = struct{}{}
		err = UeCache.Add(utranCellID, val, cache.NoExpiration)
	} else {
		utran.(map[string]struct{})[ip] = struct{}{}
		UeCache.Set(utranCellID, utran, cache.NoExpiration)
	}
	if err != nil {
		return err
	}
	return nil
}

func (p *PgwEntity) SIPREQUESTF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))

	host := p.Points["CSCF"]
	common.PackUpImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipRequest, pkg.GetData(), host, nil, nil, up) // 上行
	return nil
}

func (p *PgwEntity) SIPRESPONSEF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipresp, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	// 无线接入点
	accPoint := sipresp.Header.AccessNetworkInfo

	var remote *net.UDPAddr

	common.PackUpImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipResponse, []byte(sipresp.String()), "eNodeB", remote, pkg.Conn, down)
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
