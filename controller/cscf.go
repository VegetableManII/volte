package controller

import (
	"context"
	"crypto/md5"
	"hash"
	"log"
	"sync"

	"github.com/VegetableManII/volte/base"
	"github.com/VegetableManII/volte/common"
	"github.com/VegetableManII/volte/parser"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type UserAuthentication struct {
	Nonce     string
	RealM     string
	QOP       string
	Algorithm string
}

type CscfEntity struct {
	SipURI    string
	algorithm map[string]hash.Hash
	users     map[string]UserAuthentication
	ueMutex   sync.Mutex
	Points    map[string]string
	route     *Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (this *CscfEntity) Init() {
	this.SipURI = "x-cscf.hebeiyidong.3gpp.net"
	this.algorithm = make(map[string]hash.Hash)
	md5 := md5.New()
	this.algorithm["AKAv1-MD5"] = md5
	this.users = make(map[string]UserAuthentication)
	this.Points = make(map[string]string)
}

// HSS可以接收eps电路协议也可以接收SIP协议
func (this *CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	for {
		select {
		case msg := <-in:
			f, ok := this.route.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			sip, err := parser.ParseMessage(msg.GetData())
			if err != nil {
				logger.Error("[%v] SIP消息解析失败, %v", ctx.Value("Entity"), err)
			}
			logger.Info("[%v] %v", ctx.Value("Entity"), sip.String())
			f(ctx, msg, up, down)
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (this *CscfEntity) SIPREQUESTF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	logger.Info("[%v] Receive From PGW: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipm, err := parser.ParseMessage(m.GetData())
	if err != nil {
		return err
	}
	req := sipm.(*base.Request)
	log.Println(req)
	// 查看本地是否存在鉴权缓存
	host := this.Points["HSS"]
	common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, m.GetData(), host, up) // 上行
	return nil
}
