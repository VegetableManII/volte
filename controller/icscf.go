package controller

import (
	"context"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type I_CscfEntity struct {
	SipURI string
	SipVia string
	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (i *I_CscfEntity) Init(host string) {
	i.Mux = new(Mux)
	i.SipURI = "i-cscf.hebeiyidong.3gpp.net"
	i.SipVia = "SIP/2.0/UDP " + host + ";branch="
	i.Points = make(map[string]string)
	i.router = make(map[[2]byte]BaseSignallingT)
}

func (i *I_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	for {
		select {
		case msg := <-in:
			f, ok := i.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] I-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			f(ctx, msg, up, down)
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] I-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (i *I_CscfEntity) SIPREQUESTF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From PGW: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(m.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		// TODO 如果SIP消息中没有S-CSCF的路由则询问HSS
		// TODO	I-CSCF询问HSS得到S-CSCF列表然后选择转发给S-CSCF
		// 增加Via头部信息
		sipreq.Header.Via.Add(i.SipVia + strconv.FormatInt(common.GenerateSipBranch(), 16))
		sipreq.Header.MaxForwards.Reduce()
		common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, []byte(sipreq.String()), i.Points["SCSCF"], up)
	case "INVITE":
		return nil
	}
	return nil
}

func (i *I_CscfEntity) SIPRESPONSEF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From S-CSCF: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(m.GetData())))
	if err != nil {
		return err
	}
	// 删除Via头部信息
	sipreq.Header.Via.RemoveFirst()
	sipreq.Header.MaxForwards.Reduce()
	common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sipreq.String()), i.Points["PCSCF"], down)
	return nil
}
