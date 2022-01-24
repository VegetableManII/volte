package controller

import (
	"context"
	"crypto/md5"
	"errors"
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
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (this *CscfEntity) Init() {
	this.Mux = new(Mux)

	this.SipURI = "x-cscf.hebeiyidong.3gpp.net"
	this.algorithm = make(map[string]hash.Hash)
	md5 := md5.New()
	this.algorithm["AKAv1-MD5"] = md5
	this.users = make(map[string]UserAuthentication)
	this.Points = make(map[string]string)
	this.router = make(map[[2]byte]BaseSignallingT)
}

// HSS可以接收epc电路协议也可以接收SIP协议
func (this *CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	for {
		select {
		case msg := <-in:
			f, ok := this.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
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
	vals := req.Headers("Call-Id")
	if len(vals) == 0 {
		return errors.New("ErrMissingParamCall-Id")
	}
	callid := vals[0].String()
	// 查看本地是否存在鉴权缓存
	this.ueMutex.Lock()
	auth, ok := this.users[callid]
	this.ueMutex.Unlock()
	if !ok {
		// 向HSS查询信息
		host := this.Points["HSS"]
		_ = auth
		common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, m.GetData(), host, up) // 上行
	}
	vals = req.Headers("Authentication")
	clientAuth := vals[0].(*base.GenericHeader)
	log.Println(clientAuth.Contents)

	return nil
}

func (this *CscfEntity) SIPRESPONSEF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	logger.Info("[%v] Receive From HSS: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipm, err := parser.ParseMessage(m.GetData())
	if err != nil {
		return err
	}
	req := sipm.(*base.Response)
	log.Println(req)
	values := req.Headers("Call-Id")
	log.Println(values[0].String())
	// 查看本地是否存在鉴权缓存
	host := this.Points["HSS"]
	common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, m.GetData(), host, up) // 上行
	return nil
}
