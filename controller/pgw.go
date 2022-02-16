package controller

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type Bearer struct {
	Conn   net.Conn
	DestID string
	Net    string
	Dialer net.Dialer
	Qci    QCI
}

type QCI struct {
	Value    int
	Priority int
	Delay    time.Duration
}

var (
	IMS_SIGNALLING = QCI{Value: 5, Priority: 1, Delay: time.Millisecond * 100}
	CacheVedio     = QCI{Value: 6, Priority: 6, Delay: time.Millisecond * 300}
	AudioVedio     = QCI{Value: 7, Priority: 7, Delay: time.Millisecond * 100}
	TCP_APP_VIP    = QCI{Value: 8, Priority: 8, Delay: time.Millisecond * 300}
	TCP_APP_NVIP   = QCI{Value: 9, Priority: 9, Delay: time.Millisecond * 300}
)

type PgwEntity struct {
	*Mux
	Users  map[string]*Bearer
	ueMux  sync.Mutex
	Points map[string]string
}

func (this *PgwEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	this.Points = make(map[string]string)
	this.Users = make(map[string]*Bearer)
}

func (this *PgwEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	var err error
	for {
		select {
		case msg := <-in:
			f, ok := this.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] PGW不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			err = f(ctx, msg, up, down)
			if err != nil {
				logger.Error("[%v] PGW消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
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
	// 分配IP地址
	args["IP"] = "10.10.10.1"
	delete(args, "QCI")
	return nil
}

func (p *PgwEntity) SIPREQUESTF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	host := p.Points["CSCF"]
	common.PackUpImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipRequest, pkg.GetData(), host, up) // 上行
	return nil
}

func (p *PgwEntity) SIPRESPONSEF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From P-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	common.PackUpImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipResponse, []byte(sipreq.String()), p.Points["eNodeB"], down)
	return nil
}
