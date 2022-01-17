package controller

import (
	"context"
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
	ue *UeAuthXRES
}

func (this *MmeEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	// 初始化ue鉴权信息
	this.ue = new(UeAuthXRES)
	this.ue.xres = make(map[string]string)
}

// eps 域功能实体 MME 的逻辑代码，判断eNodeB转发过来的数据类型，如果是SIP类型则不做处理丢弃
func (this *MmeEntity) CoreProcessor(ctx context.Context, in, out chan *common.Msg) {
	var err error
	for {
		select {
		case msg := <-in:
			if msg.Type == common.SIPPROTOCAL { // 不处理SIP协议
				continue
			}
			f, ok := this.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] MME不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
			}
			err = f(ctx, msg, out)
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
func (this *MmeEntity) AttachRequestF(ctx context.Context, m *common.Msg, out chan *common.Msg) error {
	logger.Info("[%v] Receive From eNodeB: %v", ctx.Value("Entity"), string(m.Data1.GetData()))

	imsi, err := common.GetIMSI(m.Data1.GetData())
	if err != nil {
		return err
	}
	// TODO ue携带自身支持的加密算法方式
	// 组装请求内容
	req := map[string]string{
		"imsi": imsi,
	}
	common.WrapOutEPS(common.EPSPROTOCAL, common.AuthenticationInformatRequest, req, true, out) // 上行
	return nil
}

// HSS 响应Authentication Informat Response，拿到用户签名XRES、HSS服务器的Auth信息、随机数nonce和加密方法Kasme
func (this *MmeEntity) AuthenticationInformatResponseF(ctx context.Context, m *common.Msg, out chan *common.Msg) error {
	logger.Info("[%v] Receive From HSS: %v", ctx.Value("Entity"), string(m.Data1.GetData()))
	imsi, err := common.GetIMSI(m.Data1.GetData())
	if err != nil {
		return err
	}
	// 获取data部分的响应信息
	data := m.Data1.GetData()
	resp := common.StrLineUnmarshal(data)
	xres := resp[HSS_RESP_XRES]
	this.ue.mu.Lock()
	this.ue.xres[imsi] = xres
	this.ue.mu.Unlock()
	// 下行发送给ue三项，HSS服务器的Auth信息、随机数nonce和加密方法Kasme
	// 组装下行数据内容
	delete(resp, HSS_RESP_XRES)                                                                   // 删除XRES项s
	common.WrapOutEPS(common.EPSPROTOCAL, common.AuthenticationInformatRequest, resp, false, out) // 下行
	return nil
}
