package controller

import (
	"context"
	"errors"
	"sync"

	"github.com/VegetableManII/volte/common"

	"github.com/wonderivan/logger"
)

type UeAuthXRES struct {
	xres map[string]string
	mu   sync.Mutex
}

type MmeEntity struct {
	*Mux
	ue     *UeAuthXRES
	Points map[string]string
}

func (this *MmeEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	// 初始化ue鉴权信息
	this.ue = new(UeAuthXRES)
	this.ue.xres = make(map[string]string)
	this.Points = make(map[string]string)
}

// epc 域功能实体 MME 的逻辑代码，判断eNodeB转发过来的数据类型，如果是SIP类型则不做处理丢弃
func (this *MmeEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
	var err error
	for {
		select {
		case msg := <-in:
			if msg.GetType() == common.SIPPROTOCAL { // 不处理SIP协议
				continue
			}
			f, ok := this.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] MME不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			err = f(ctx, msg, up, down)
			if err != nil {
				logger.Error("[%v] MME消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] MME逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

// 附着请求，携带IMSI，和客户端支持的加密方法，拿到IMSI向HSS发起Authentication Informat Request请求
func (this *MmeEntity) AttachRequestF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(m.GetData()))
	data := m.GetData()
	args := common.StrLineUnmarshal(data)
	imsi := args["IMSI"]
	// TODO ue携带自身支持的加密算法方式
	// 组装请求内容
	req := map[string]string{
		"IMSI": imsi,
	}

	host := this.Points["HSS"]
	common.PackageOut(common.EPCPROTOCAL, common.AuthenticationInformatRequest, req, host, up) // 上行
	return nil
}

// HSS 响应Authentication Informat Response，拿到用户签名XRES、HSS服务器的Auth信息、随机数nonce和加密方法Kasme
func (this *MmeEntity) AuthenticationInformatResponseF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(m.GetData()))
	// 获取data部分的响应信息
	data := m.GetData()
	args := common.StrLineUnmarshal(data)
	imsi := args["IMSI"]
	xres := args[AV_XRES]
	this.ue.mu.Lock()
	this.ue.xres[imsi] = xres
	this.ue.mu.Unlock()
	// 下行发送给ue三项，HSS服务器的Auth信息、随机数nonce和加密方法Kasme
	// 组装下行数据内容
	delete(args, AV_XRES)

	host := this.Points["eNodeB"]
	common.PackageOut(common.EPCPROTOCAL, common.AuthenticationInformatRequest, args, host, down) // 下行
	return nil
}

// UE终端 响应AuthenticationResponse，比较用户RES是否与XRES一致
func (this *MmeEntity) AuthenticationResponseF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From eNodeB: \n%v", ctx.Value("Entity"), string(m.GetData()))
	data := m.GetData()
	args := common.StrLineUnmarshal(data)
	imsi := args["IMSI"]
	res := args["RES"]
	// 对比RES和XRES
	this.ue.mu.Lock()
	xres := this.ue.xres[imsi]
	this.ue.mu.Unlock()
	if res != xres {
		// 鉴权失败，重新发起鉴权请求
		host := this.Points["eNodeB"]
		common.PackageOut(common.EPCPROTOCAL, common.AuthenticationInformatRequest, nil, host, down)
		return errors.New("ErrAuthenFailed")
	}
	// 向上行HSS发送位置更新请求
	host := this.Points["HSS"]
	common.PackageOut(common.EPCPROTOCAL, common.UpdateLocationRequest, nil, host, up) // 上行
	return nil
}

// 接收HSS的 Update Location ACK 响应得到用户的APN，再请求PGW完成承载的建立（该部分暂不实现，得到ACK后直接向用户发送Attach Accept）
func (this *MmeEntity) UpdateLocationACKF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From HSS: \n%v", ctx.Value("Entity"), string(m.GetData()))
	data := m.GetData()
	args := common.StrLineUnmarshal(data)
	/*
		1.获得APN
		2.请求PGW建立承载
	*/
	pgwaddr := args["APN"]
	ueip := args["IP"]
	// 请求PGW建立IMS信令承载
	common.PackageOut(common.EPCPROTOCAL, common.CreateSessionRequest, args, pgwaddr, up)
	// 响应UE终端附着允许的响应
	host := this.Points["eNodeB"]
	t2 := map[string]string{
		"IP": ueip,
	}
	common.PackageOut(common.EPCPROTOCAL, common.AttachAccept, t2, host, up)
	return nil
}
