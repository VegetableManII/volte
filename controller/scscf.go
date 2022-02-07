package controller

import (
	"context"
	"encoding/base64"
	"encoding/hex"
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
	SipURI        string
	SipVia        string
	algorithm     map[string]hash.Hash
	userAuthCache map[string]string
	authMutex     sync.Mutex
	userReqCache  map[string]*sip.Message
	reqMutex      sync.Mutex
	Points        map[string]string
	*Mux
}

// 暂时先试用固定的uri，后期实现dns使用域名加IP的映射方式
func (s *S_CscfEntity) Init(host string) {
	s.Mux = new(Mux)
	s.SipURI = "s-cscf.hebeiyidong.3gpp.net"
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
				logger.Error("[%v] S-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			f(ctx, msg, up, down)
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] S-CSCF逻辑核心退出", ctx.Value("Entity"))
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
		return regist(ctx, s.SipVia, s.userAuthCache, &s.authMutex, s.userReqCache, &s.reqMutex, &sipreq,
			s.Points["HSS"], s.Points["ICSCF"], up, down)
	case "INVITE":
		return nil
	}
	return nil
}

func parseAuthentication(authHeader string) map[string]string {
	res := make(map[string]string)
	items := strings.Split(authHeader, " ")
	for _, item := range items {
		val := strings.Split(item, "=")
		if len(val) >= 2 {
			res[val[0]] = val[1]
		}
	}
	_ = res
	return nil
}

func (s *S_CscfEntity) MutimediaAuthorizationAnswerF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: %v", ctx.Value("Entity"), string(m.GetData()))
	host := s.Points["ICSCF"]
	// 获得用户鉴权信息
	resp := common.StrLineUnmarshal(m.GetData())
	user := resp["UserName"]
	AUTN := resp["AUTN"]
	XRES := resp["XRES"]
	CK := resp["CK"]
	IK := resp["IK"]
	// TODO  使用CK和IK完成与UE的IPSec隧道的建立
	_, _ = CK, IK
	// 组装WWW-Authenticate
	abs, _ := hex.DecodeString(AUTN)
	wwwAuth := fmt.Sprintf(`Digest realm=hebeiyidomg.3gpp.net nonce=%s qop=auth-int algorithm=AKAv1-MD5`, base64.StdEncoding.EncodeToString(abs))
	// 获取用户请求缓存
	s.reqMutex.Lock()
	sipreq, ok := s.userReqCache[user]
	s.reqMutex.Unlock()
	if !ok {
		logger.Error("[%v] Lose User Request Cache %v", ctx.Value("Entity"), user)
	}
	// 保存用户鉴权
	s.authMutex.Lock()
	s.userAuthCache[user] = XRES
	s.authMutex.Unlock()
	// 响应终端鉴权失败
	sipresp := sip.NewResponse(sip.StatusUnauthorized, sipreq)
	sipresp.Header.WWWAuthenticate = wwwAuth
	common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sipresp.String()), host, down)
	return nil
}

func regist(ctx context.Context, via string, authCache map[string]string, auMux *sync.Mutex, reqCache map[string]*sip.Message, reqMux *sync.Mutex, req *sip.Message, hss, downlink string, up, down chan *common.Package) error {
	// 查看本地是否存在鉴权缓存
	auth := req.Header.Authorization
	values := parseAuthentication(auth)
	auMux.Lock()
	RES, ok := authCache[values["username"]]
	auMux.Unlock()
	if !ok {
		// 向HSS发起MAR，再收到MAA，异步实现
		// 缓存本次请求
		reqMux.Lock()
		reqCache[auth] = req
		// 向HSS查询信息
		table := map[string]string{
			"UserName": values["username"],
		}
		common.PackageOut(common.EPCPROTOCAL, common.MultiMediaAuthenticationRequest, table, hss, up) // 上行
		return nil
	} else {
		values := parseAuthentication(req.Header.Authorization)
		if RES != "" && RES == values["response"] {
			sresp := sip.NewResponse(sip.StatusOK, req)
			common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sresp.String()), downlink, down)
		} else {
			sresp := sip.NewResponse(sip.StatusUnauthorized, req)
			common.RawPackageOut(common.SIPPROTOCAL, common.SipResponse, []byte(sresp.String()), downlink, down)
		}
		return nil
	}
}
