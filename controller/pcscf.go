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
	SipVia string
	Points map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (p *P_CscfEntity) Init(host string) {
	p.Mux = new(Mux)
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
			err := f(ctx, msg, up, down)
			if err != nil {
				logger.Error("[%v] P-CSCF消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] P-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (p *P_CscfEntity) SIPREQUESTF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From PGW: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}

	// 增加Via头部信息
	sipreq.Header.Via.Add(p.SipVia + strconv.FormatInt(common.GenerateSipBranch(), 16))
	sipreq.Header.MaxForwards.Reduce()
	// 检查头部内容是否首次注册
	if strings.Contains(sipreq.Header.Authorization, "response") { // 包含响应内容则为第二次注册请求
		// TODO 第二次注册请求时PCSCF可以直接转发给ICSCF
		common.ImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipRequest, []byte(sipreq.String()), p.Points["ICSCF"], nil, nil, up)
	} else {
		// 第一次注册请求，P-CSCF处理，填充Authorization头部
		user := sipreq.Header.From.URI.Username
		domain := sipreq.Header.From.URI.Domain
		username := user + "@" + domain
		auth := fmt.Sprintf("Digest username=%s integrity protection:no", username)
		sipreq.Header.Authorization = auth
		// 第一次注册请求SCSCF还未与UE绑定所以转发给ICSCF
		common.ImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipRequest, []byte(sipreq.String()), p.Points["ICSCF"], nil, nil, up)
	}
	return nil
}

func (p *P_CscfEntity) SIPRESPONSEF(ctx context.Context, pkg *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From I-CSCF: \n%v", ctx.Value("Entity"), string(pkg.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(pkg.GetData())))
	if err != nil {
		return err
	}
	// 删除Via头部信息
	sipreq.Header.Via.RemoveFirst()
	sipreq.Header.MaxForwards.Reduce()
	common.ImsMsg(pkg.CommonMsg, common.SIPPROTOCAL, common.SipResponse, []byte(sipreq.String()), p.Points["PGW"], nil, nil, down)
	return nil
}
