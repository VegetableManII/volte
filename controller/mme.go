package controller

import (
	"context"
	"encoding/binary"
	"sync"
	"volte/common"

	"github.com/wonderivan/logger"
)

type UeAuthXRES struct {
	xres map[[4]byte]string
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
	this.ue.xres = make(map[[4]byte]string)
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
			f := this.router[msg.GetUniqueMethod()]
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
	imsi := m.Data1.GetIMSI()
	// TODO ue携带自身支持的加密算法方式
	// 组装请求内容
	common.WrapOutEPS(common.EPSPROTOCAL, common.AuthenticationInformatRequest, imsi, nil, out)
	return nil
}

// HSS 响应Authentication Informat Response，拿到用户签名XRES、HSS服务器的Auth信息、随机数nonce和加密方法Kasme
func (this *MmeEntity) AuthenticationInformatResponseF(ctx context.Context, m *common.Msg, out chan *common.Msg) error {
	imsi := m.Data1.GetIMSI()
	// 获取data部分的响应信息
	data := m.Data1.GetData()
	resp := common.StrLineUnmarshal(data)
	xres := resp[HSS_RESP_XRES]
	this.ue.mu.Lock()
	this.ue.xres[imsi] = xres
	this.ue.mu.Unlock()
	// 下行发送给ue三项，HSS服务器的Auth信息、随机数nonce和加密方法Kasme
	// 组装下行数据内容
	down := new(common.EpsMsg)
	delete(resp, HSS_RESP_XRES) // 删除XRES项s
	ext := common.StrLineMarshal(resp)
	len := len([]byte(ext))
	size := [2]byte{}
	binary.BigEndian.PutUint16(size[:], uint16(len))
	down.Construct(common.EPSPROTOCAL, common.AuthenticationRequest, size, imsi, []byte(ext))
	wrap := new(common.Msg)
	wrap.Type = common.EPSPROTOCAL
	wrap.Data1 = down
	out <- wrap
	return nil
}
