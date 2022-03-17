package controller

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"

	_ "github.com/go-sql-driver/mysql"

	"github.com/wonderivan/logger"
)

type S_CscfEntity struct {
	SipURI        string
	SipVia        string
	core          chan *modules.Package
	userAuthCache map[string]string
	authMutex     sync.Mutex
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
	s.userAuthCache = make(map[string]string)
}

func (s *S_CscfEntity) CoreProcessor(ctx context.Context, in, up, down chan *modules.Package) {
	s.core = in
	for {
		select {
		case pkg := <-in:
			f, ok := s.router[pkg.GetRoute()]
			if !ok {
				logger.Error("[%v] S-CSCF不支持的消息类型数据 %v", ctx.Value("Entity"), pkg.GetRoute())
				continue
			}
			err := f(ctx, pkg, up, down)
			if err != nil {
				logger.Error("[%v] P-CSCF消息处理失败 %v %v", ctx.Value("Entity"), pkg, err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] S-CSCF逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (s *S_CscfEntity) SIPREQUESTF(ctx context.Context, p *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From ICSCF: \n%v", ctx.Value("Entity"), string(p.GetData()))
	// 解析SIP消息
	sipreq, err := sip.NewMessage(strings.NewReader(string(p.GetData())))
	if err != nil {
		return err
	}
	switch sipreq.RequestLine.Method {
	case "REGISTER":
		return s.regist(ctx, &sipreq, p.CommonMsg, up, down)
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
	return res
}

func (s *S_CscfEntity) MutimediaAuthorizationAnswerF(ctx context.Context, m *modules.Package, up, down chan *modules.Package) error {
	defer modules.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(m.GetData()))
	// TODO  使用CK和IK完成与UE的IPSec隧道的建立
	return nil
}

func (s *S_CscfEntity) regist(ctx context.Context, req *sip.Message, msg *modules.CommonMsg, up, down chan *modules.Package) error {
	user := req.Header.From.Username()
	downlink := s.Points["ICSCF"]
	uplink := s.Points["HSS"]
	// 查看本地是否存在鉴权缓存
	s.authMutex.Lock()
	RES, ok := s.userAuthCache[user]
	s.authMutex.Unlock()
	if !ok {
		logger.Info("[%v] 首次注册: %v", ctx.Value("Entity"), user)
		// 向HSS发起MAR，再收到MAA，同步实现
		// 向HSS查询信息
		table := map[string]string{
			"UserName": user,
		}
		m, err := modules.MARSyncRequest(ctx, msg, modules.EPCPROTOCAL, modules.MultiMediaAuthenticationRequest, table, uplink)
		if err != nil {
			logger.Error("[%v] HSS Response Error %v", ctx.Value("Entity"), err)
			sipresp := sip.NewResponse(sip.StatusNoResponse, req)
			modules.ImsMsg(msg, modules.SIPPROTOCAL, modules.SipResponse, []byte(sipresp.String()), s.Points["ICSCF"], nil, nil, down)
		} else {
			// 获得用户鉴权信息
			resp := modules.StrLineUnmarshal(m.GetData())
			user := resp["UserName"]
			AUTN := resp["AUTN"]
			XRES := resp["XRES"]
			RAND := resp["RAND"]
			// 保存用户鉴权
			s.authMutex.Lock()
			s.userAuthCache[user] = XRES
			s.authMutex.Unlock()

			// 组装WWW-Authenticate
			autn, _ := hex.DecodeString(AUTN)
			rand, _ := hex.DecodeString(RAND)
			nonce := append(rand, autn...)
			wwwAuth := fmt.Sprintf(`Digest realm=hebeiyidomg.3gpp.net nonce=%s qop=auth-int algorithm=AKAv1-MD5`, base64.StdEncoding.EncodeToString(nonce))
			// 向终端发起鉴权
			sipresp := sip.NewResponse(sip.StatusUnauthorized, req)
			sipresp.Header.WWWAuthenticate = wwwAuth
			logger.Info("[%v] MAA响应: %v", ctx.Value("Entity"), sipresp.String())

			modules.ImsMsg(msg, modules.SIPPROTOCAL, modules.SipResponse, []byte(sipresp.String()), downlink, nil, nil, down)
			// 透传MAA响应给自己的路由
			p := &modules.Package{
				Destation:  downlink,
				RemoteAddr: nil,
				Conn:       nil,
			}
			p.CommonMsg = m
			s.core <- p
		}
	} else {
		values := parseAuthentication(req.Header.Authorization)
		if RES != "" && RES == values["response"] {
			sresp := sip.NewResponse(sip.StatusOK, req)
			logger.Info("[%v] 鉴权认证成功: %v", ctx.Value("Entity"), sresp.String())

			modules.ImsMsg(msg, modules.SIPPROTOCAL, modules.SipResponse, []byte(sresp.String()), downlink, nil, nil, down)
		} else {
			s.authMutex.Lock()
			delete(s.userAuthCache, user)
			s.authMutex.Unlock()
			sresp := sip.NewResponse(sip.StatusUnauthorized, req)
			logger.Info("[%v] 发起对UE鉴权: %v", ctx.Value("Entity"), sresp.String())
			modules.ImsMsg(msg, modules.SIPPROTOCAL, modules.SipResponse, []byte(sresp.String()), downlink, nil, nil, down)
		}
	}
	return nil
}
