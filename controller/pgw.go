package controller

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"

	"github.com/wonderivan/logger"
)

type PgwEntity struct {
	*Mux
	Points    map[string]string
	UtranConn *UtranConn
}

func (this *PgwEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	this.Points = make(map[string]string)
	this.UtranConn = new(UtranConn)
}

func (this *PgwEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	var err error
	for {
		select {
		case msg := <-in:
			// 兼容心跳包
			if msg.CommonMsg == nil && msg.RemoteAddr != nil && msg.Conn == nil {
				this.updateUtranAddress(ctx, msg.RemoteAddr, msg.Destation)
			} else {
				f, ok := this.router[msg.GetUniqueMethod()]
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

func (p *PgwEntity) updateUtranAddress(ctx context.Context, ra *net.UDPAddr, enb string) error {
	p.UtranConn.Lock()
	p.UtranConn.RemoteAddr[enb] = ra
	p.UtranConn.Unlock()
	return nil
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
	common.PackUpImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipRequest, pkg.GetData(), host, nil, nil, up) // 上行
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

	var remote *net.UDPAddr
	p.UtranConn.Lock()
	if p.UtranConn.RemoteAddr == nil {
		return errors.New("ErrUtranConn")
	} else {
		remote = p.UtranConn.RemoteAddr
	}
	p.UtranConn.Unlock()

	logger.Info("%v", remote)

	common.PackUpImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipResponse, []byte(sipreq.String()), p.Points["eNodeB"], remote, pkg.Conn, down)
	return nil
}
