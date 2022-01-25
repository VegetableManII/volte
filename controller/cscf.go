package controller

import (
	"context"
	"crypto/md5"
	"hash"
	"log"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip2"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type CscfEntity struct {
	SipURI    string
	algorithm map[string]hash.Hash
	users     map[string]string
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
	this.users = make(map[string]string)
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
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From PGW: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sreq, err := sip.NewMessage(strings.NewReader(string(m.GetData())))
	if err != nil {
		return err
	}
	// 查看本地是否存在鉴权缓存
	this.ueMutex.Lock()
	auth, ok := this.users[sreq.Header.CallID]
	this.ueMutex.Unlock()
	if !ok {
		// 向HSS查询信息
		host := this.Points["HSS"]
		username := sreq.Header.From.Username()
		table := map[string]string{
			"username": username,
		}
		common.PackageOut(common.EPCPROTOCAL, common.UserAuthorizationRequest, table, host, up) // 上行
	}
	respauth := parseAuthentication(sreq.Header.Authorization)
	log.Println(sreq.Header.CallID, auth)

	if auth != respauth {
		sresp := sip.NewResponse(sip.StatusUnauthorized, &sreq)
		common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sresp.String()), this.Points["PGW"], down)
	} else {
		sresp := sip.NewResponse(sip.StatusOK, &sreq)
		common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sresp.String()), this.Points["PGW"], down)
	}
	return nil
}

func (this *CscfEntity) SIPRESPONSEF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	// 查看本地是否存在鉴权缓存
	host := this.Points["PGW"]
	common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, m.GetData(), host, down)
	return nil
}

func parseAuthentication(authHeader string) string {
	items := strings.Split(authHeader, " ")
	for _, item := range items {
		val := strings.Split(strings.Trim(item, ","), "=")
		if len(val) >= 2 {
			if val[0] == "response" {
				return val[1]
			}
		}
	}
	return ""
}

func (this *CscfEntity) UserAuthorizationAnswerF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: %v", ctx.Value("Entity"), string(m.GetData()))
	// 获得用户鉴权信息
	// 查看本地是否存在鉴权缓存
	host := this.Points["PGW"]
	common.RawPackageOut(common.SIPPROTOCAL, common.SipRequest, m.GetData(), host, down)
	return nil
}
