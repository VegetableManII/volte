package controller

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/VegetableManII/volte/common"

	"github.com/wonderivan/logger"
)

type UeAuthXRES struct {
	xres map[string]string
	sync.Mutex
}

type MmeEntity struct {
	*Mux
	ue        *UeAuthXRES
	Points    map[string]string
	UtranConn *UtranConn
}

func (m *MmeEntity) Init() {
	// 初始化路由
	m.Mux = new(Mux)
	m.router = make(map[[2]byte]BaseSignallingT)
	// 初始化ue鉴权信息
	m.ue = new(UeAuthXRES)
	m.ue.xres = make(map[string]string)
	m.Points = make(map[string]string)
}

// epc 域功能实体 MME 的逻辑代码，判断eNodeB转发过来的数据类型，如果是SIP类型则不做处理丢弃
func (m *MmeEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	var err error
	for {
		select {
		case msg := <-in:
			if msg.CommonMsg == nil && msg.RemoteAddr != nil && msg.Conn != nil {
				m.updateUtranAddress(ctx, msg.RemoteAddr)
			} else {
				if msg.GetType() == common.SIPPROTOCAL { // 不处理SIP协议
					continue
				}
				f, ok := m.router[msg.GetUniqueMethod()]
				if !ok {
					logger.Error("[%v] MME不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
					continue
				}
				err = f(ctx, msg, up, down)
				if err != nil {
					logger.Error("[%v] MME消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
				}
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] MME逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

func (m *MmeEntity) updateUtranAddress(ctx context.Context, ra *net.UDPAddr) error {
	m.UtranConn.Lock()
	m.UtranConn.RemoteAddr = ra
	m.UtranConn.Unlock()
	return nil
}

// 附着请求，携带IMSI，和客户端支持的加密方法，拿到IMSI向HSS发起Authentication Informat Request请求
func (m *MmeEntity) AttachRequestF(ctx context.Context, p *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(p.GetData()))

	m.UtranConn.Lock()
	m.UtranConn.RemoteAddr = p.RemoteAddr
	m.UtranConn.Unlock()

	data := p.GetData()
	args := common.StrLineUnmarshal(data)
	imsi := args["IMSI"]
	// TODO ue携带自身支持的加密算法方式
	// 组装请求内容
	req := map[string]string{
		"IMSI": imsi,
	}

	host := m.Points["HSS"]
	common.PackUpEpcMsg(p.CommonMsg, common.EPCPROTOCAL, common.AuthenticationInformatRequest, req, host, nil, nil, up) // 上行
	return nil
}

// HSS 响应Authentication Informat Response，拿到用户签名XRES、HSS服务器的Auth信息、随机数nonce和加密方法Kasme
func (m *MmeEntity) AuthenticationInformatResponseF(ctx context.Context, p *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(p.GetData()))
	// 获取data部分的响应信息
	data := p.GetData()
	args := common.StrLineUnmarshal(data)
	imsi := args["IMSI"]
	xres := args[AV_XRES]
	m.ue.Lock()
	m.ue.xres[imsi] = xres
	m.ue.Unlock()
	// 下行发送给ue三项，HSS服务器的Auth信息、随机数nonce和加密方法Kasme
	// 组装下行数据内容
	delete(args, AV_XRES)

	host := m.Points["eNodeB"]
	var remote *net.UDPAddr
	m.UtranConn.Lock()
	if m.UtranConn.RemoteAddr == nil {
		return errors.New("ErrUtranConn")
	} else {
		remote = m.UtranConn.RemoteAddr
	}
	common.PackUpEpcMsg(p.CommonMsg, common.EPCPROTOCAL, common.AuthenticationInformatRequest, args, host, remote, p.Conn, down) // 下行
	return nil
}

// UE终端 响应AuthenticationResponse，比较用户RES是否与XRES一致
func (m *MmeEntity) AuthenticationResponseF(ctx context.Context, p *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(p.GetData()))
	data := p.GetData()
	args := common.StrLineUnmarshal(data)
	imsi := args["IMSI"]
	res := args["RES"]
	// 对比RES和XRES
	m.ue.Lock()
	xres := m.ue.xres[imsi]
	m.ue.Unlock()
	if res != xres {
		// 鉴权失败，重新发起鉴权请求
		host := m.Points["eNodeB"]
		var remote *net.UDPAddr
		m.UtranConn.Lock()
		if m.UtranConn.RemoteAddr == nil {
			return errors.New("ErrUtranConn")
		} else {
			remote = m.UtranConn.RemoteAddr
		}
		common.PackUpEpcMsg(p.CommonMsg, common.EPCPROTOCAL, common.AuthenticationInformatRequest, nil, host, remote, p.Conn, down)
		return errors.New("ErrAuthenFailed")
	}
	// 向上行HSS发送位置更新请求
	host := m.Points["HSS"]
	common.PackUpEpcMsg(p.CommonMsg, common.EPCPROTOCAL, common.UpdateLocationRequest, nil, host, nil, nil, up) // 上行
	return nil
}

// 接收HSS的 Update Location ACK 响应得到用户的APN，再请求PGW完成承载的建立（该部分暂不实现，得到ACK后直接向用户发送Attach Accept）
func (m *MmeEntity) UpdateLocationACKF(ctx context.Context, p *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(p.GetData()))
	data := p.GetData()
	args := common.StrLineUnmarshal(data)
	/*
		1.获得APN
		2.请求PGW建立承载
	*/
	pgwaddr := args["APN"]
	delete(args, "APN")
	args["QCI"] = "5"
	// 请求PGW建立IMS信令承载
	common.PackUpEpcMsg(p.CommonMsg, common.EPCPROTOCAL, common.CreateSessionRequest, args, pgwaddr, nil, nil, up)
	// 响应UE终端附着允许的响应
	host := m.Points["eNodeB"]
	var remote *net.UDPAddr
	m.UtranConn.Lock()
	if m.UtranConn.RemoteAddr == nil {
		return errors.New("ErrUtranConn")
	} else {
		remote = m.UtranConn.RemoteAddr
	}
	common.PackUpEpcMsg(p.CommonMsg, common.EPCPROTOCAL, common.AttachAccept, args, host, remote, p.Conn, up)
	return nil
}
