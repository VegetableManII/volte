package controller

import (
	"context"
	"fmt"
	"hash"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/common"
	sip "github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type S_CscfEntity struct {
	SipURI    string
	SipVia    string
	algorithm map[string]hash.Hash
	userAuth  map[string]string
	uaMutex   sync.Mutex
	userReq   map[string]*sip.Message
	urMutex   sync.Mutex
	Points    map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (s *S_CscfEntity) Init(host string) {
	s.Mux = new(Mux)
	s.SipURI = "x-cscf.hebeiyidong.3gpp.net"
	s.SipVia = "SIP/2.0/UDP " + host + ";branch="
	s.Points = make(map[string]string)
	s.router = make(map[[2]byte]BaseSignallingT)
}

func (s *S_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	for {
		select {
		case msg := <-in:
			f, ok := s.router[msg.GetUniqueMethod()]
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

func (s *S_CscfEntity) SIPREQUESTF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From PGW: %v", ctx.Value("Entity"), string(m.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(m.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		return regist(ctx, s.SipVia, s.userAuth, &s.uaMutex, s.userReq, &s.urMutex, &sipreq,
			s.Points["HSS"], s.Points["PGW"], up, down)
	case "INVITE":
		return nil
	}
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

func (s *S_CscfEntity) UserAuthorizationAnswerF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: %v", ctx.Value("Entity"), string(m.GetData()))
	host := s.Points["PGW"]
	// 获得用户鉴权信息
	resp := common.StrLineUnmarshal(m.GetData())
	un := resp["UserName"]
	nonce := resp["Nonce"]
	realm := resp["Realm"]
	code := resp["Code"]

	// 组装WWW-Authenticate
	wwwAuth := fmt.Sprintf(`Digest realm="%v", nonce="%v", qop="auth-int", algorithm=AKAv1-MD5`, realm, nonce)
	// 获取用户请求缓存
	s.urMutex.Lock()
	sipreq, ok := s.userReq[un]
	s.urMutex.Unlock()
	if !ok {
		logger.Error("[%v] Lose User Request Cache %v", ctx.Value("Entity"), un)
	}
	// 保存用户鉴权
	s.uaMutex.Lock()
	s.userAuth[un] = code
	s.uaMutex.Unlock()
	// 响应终端鉴权失败
	sipresp := sip.NewResponse(sip.StatusUnauthorized, sipreq)
	sipresp.Header.WWWAuthenticate = wwwAuth
	common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sipresp.String()), host, down)
	return nil
}

func regist(ctx context.Context, via string, ua map[string]string, uamux *sync.Mutex, ur map[string]*sip.Message, urmux *sync.Mutex, req *sip.Message, hss, pgw string, up, down chan *common.Package) error {
	// P-CSCF处理，填充Authorization头部
	user := req.Header.From.Username()
	if req.Header.From.Username() == "" {
		user = req.Header.From.URI.Username
	}
	auth := fmt.Sprintf("Digest username=%v integrity protection:no", user)
	// 转发给I-CSCF
	/*
		I-CSCF查询所在域的S-CSCF转发给S-CSCF
	*/
	via += fmt.Sprintf("%d", common.GenerateSipBranch())
	req.Header.Via.Add(via)
	req.Header.Authorization = auth
	// 查看本地是否存在鉴权缓存
	uamux.Lock()
	auth, ok := ua[req.Header.From.DisplayName]
	uamux.Unlock()
	if !ok {
		// 缓存本次请求
		urmux.Lock()
		ur[req.Header.From.DisplayName] = req
		urmux.Unlock()
		// 向HSS查询信息
		username := req.Header.From.Username()
		table := map[string]string{
			"UserName": username,
		}
		common.PackageOut(common.EPCPROTOCAL, common.UserAuthorizationRequest, table, hss, up) // 上行
		return nil
	} else {
		respauth := parseAuthentication(req.Header.Authorization)
		if auth != respauth {
			sresp := sip.NewResponse(sip.StatusUnauthorized, req)
			common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sresp.String()), pgw, down)
		} else {
			sresp := sip.NewResponse(sip.StatusOK, req)
			common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sresp.String()), pgw, down)
		}
		return nil
	}

}
