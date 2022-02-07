package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type P_CscfEntity struct {
	SipURI string
	SipVia string
	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (p *P_CscfEntity) Init(host string) {
	p.Mux = new(Mux)
	p.SipURI = "p-cscf.hebeiyidong.3gpp.net"
	p.SipVia = "SIP/2.0/UDP " + host + ";branch="
	p.Points = make(map[string]string)
	p.router = make(map[[2]byte]BaseSignallingT)
}

func (p *P_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	for {
		select {
		case msg := <-in:
			f, ok := p.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] P-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			f(ctx, msg, up, down)
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] P-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (p *P_CscfEntity) SIPREQUESTF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From PGW: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(m.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		// 增加Via头部信息
		sipreq.Header.Via.Add(p.SipVia + strconv.FormatInt(common.GenerateSipBranch(), 16))
		sipreq.Header.MaxForwards.Reduce()
		// 检查头部内容是否首次注册
		if strings.Contains(sipreq.Header.Authorization, "response") { // 包含响应内容则为第二次注册请求
			// TODO 第二次注册请求时PCSCF可以直接转发给SCSCF
		} else {
			// 第一次注册请求，P-CSCF处理，填充Authorization头部
			user := sipreq.Header.From.URI.Username
			domain := sipreq.Header.From.URI.Domain
			username := user + "@" + domain
			auth := fmt.Sprintf("Digest username=%s integrity protection:no", username)
			sipreq.Header.Authorization = auth
			// 第一次注册请求SCSCF还未与UE绑定所以转发给ICSCF
			// common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, []byte(sipreq.String()), p.Points["ICSCF"], up)
		}
		// 先统一转发给ICSCF，待后期完善与HSS的交互之后实现第二次注册直接转发给SCSCF
		common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, []byte(sipreq.String()), p.Points["ICSCF"], up)
	case "INVITE":
		return nil
	}
	return nil
}
