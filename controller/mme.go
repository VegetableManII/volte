package controller

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"volte/common"

	"github.com/wonderivan/logger"
)

type UeAuthXRES struct {
	xres map[[4]byte]string
	mu   sync.Mutex
}

var ueAuthXres *UeAuthXRES

func init() {
	ueAuthXres = new(UeAuthXRES)
	ueAuthXres.xres = make(map[[4]byte]string)
}

/*
 epc 域功能实体 MME 的逻辑代码，判断eNodeB转发过来的数据类型，如果是SIP类型则不做处理丢弃
*/

func ProcessMMECore(ctx context.Context, coreIn, coreOut chan *common.Msg) {
	var err error
	for {
		select {
		case m := <-coreIn:
			if m.Type == common.SIPPROTOCAL { // 不处理SIP协议
				continue
			}
			err = processormme(m, coreOut)
			if err != nil {
				logger.Error("[%v] MME消息处理失败 %v %v", ctx.Value("Entity"), m, err)
			}
		}
	}
}

func processormme(m *common.Msg, out chan *common.Msg) error {
	if m.Data1 == nil {
		return errors.New("ErrData")
	}
	switch m.Data1.GetType() {
	case common.AttachRequest:
		// 附着请求，携带IMSI，和客户端支持的加密方法，拿到IMSI向HSS发起Authentication Informat Request请求
		imsi := m.Data1.GetIMSI()
		// TODO ue携带自身支持的加密算法方式
		// 组装请求内容
		req := new(common.EpcMsg)
		req.Construct(common.EPCPROTOCAL, common.AuthenticationInformatRequest,
			[2]byte{0}, imsi, []byte{})
		wrap := new(common.Msg)
		wrap.Type = common.EPCPROTOCAL
		wrap.Data1 = req
		out <- wrap
	case common.AuthenticationInformatResponse:
		// HSS 响应Authentication Informat Response，拿到用户签名XRES、HSS服务器的Auth信息、随机数nonce和加密方法Kasme
		imsi := m.Data1.GetIMSI()
		// 获取data部分的响应信息
		data := m.Data1.GetData()
		resp := common.StrLineUnmarshal(data)
		xres := resp[HSS_RESP_XRES]
		ueAuthXres.mu.Lock()
		ueAuthXres.xres[imsi] = xres
		ueAuthXres.mu.Unlock()
		// 下行发送给ue三项，HSS服务器的Auth信息、随机数nonce和加密方法Kasme
		// 组装下行数据内容
		down := new(common.EpcMsg)
		delete(resp, HSS_RESP_XRES) // 删除XRES项s
		ext := common.StrLineMarshal(resp)
		len := len([]byte(ext))
		size := [2]byte{}
		binary.BigEndian.PutUint16(size[:], uint16(len))
		down.Construct(common.EPCPROTOCAL, common.AuthenticationRequest, size, imsi, []byte(ext))
		wrap := new(common.Msg)
		wrap.Type = common.EPCPROTOCAL
		wrap.Data1 = down
		out <- wrap
	}
	return errors.New("ErrUnsupportMethod")
}
