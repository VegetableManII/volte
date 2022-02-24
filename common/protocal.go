package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"strings"
)

const (
	EPCPROTOCAL byte = 0x01
	SIPPROTOCAL byte = 0x00
	BEATHEART   byte = 0xFF
)

// epc message的消息类型
const (
	AttachRequest                   byte = 0x00 // UE发起Attach请求
	AuthenticationInformatRequest   byte = 0x01
	AuthenticationInformatResponse  byte = 0x02
	AuthenticationRequest           byte = 0x03 // 网络侧向UE发起，UE侧需要实现该接口
	AuthenticationResponse          byte = 0x04 // UE响应网络侧，由UE实现
	UpdateLocationRequest           byte = 0x05
	UpdateLocationACK               byte = 0x06
	CreateSessionRequest            byte = 0x07
	CreateSessionResponse           byte = 0x08
	QCI                             byte = 0x09
	AttachAccept                    byte = 0x0A // 网络侧向UE发起，通知附着成功
	UserAuthorizationRequest        byte = 0x0B
	UserAuthorizationAnswer         byte = 0x0C
	MultiMediaAuthenticationRequest byte = 0x0D
	MultiMediaAuthenticationAnswer  byte = 0x0E
	HeartBeatUpdate                 byte = 0xFF
)

// sip message的消息类型
const (
	SipRequest  byte = 0x00
	SipResponse byte = 0x01
)

type Package struct {
	*CommonMsg
	Destation  string       // 异步响应
	RemoteAddr *net.UDPAddr // 客户端地址
	Conn       *net.UDPConn // 客户端连接，同步响应
}
type CommonMsg struct {
	_unique   uint32 // 全局唯一ID，供基站区分不同用户请求使用
	_protocal uint8  // 0x01 表示电路域协议
	_method   uint8
	_size     uint16      // data字段的长度
	_data     [65535]byte // 最大65535字节大小
}

func (p *Package) GetUniqueMethod() [2]byte {
	uniq := [2]byte{p._protocal, p._method}
	return uniq
}

func (m *CommonMsg) Init(data []byte) {
	if data[4] == EPCPROTOCAL {
		m._unique = binary.BigEndian.Uint32(data[0:4])
		l := binary.BigEndian.Uint16(data[6:8])
		m._protocal = data[4]
		m._method = data[5]
		m._size = l
		copy(m._data[:], data[8:l+8])
	} else {
		// SIP Header 格式如下
		/*
			请求：REGISTER sip:apn.sip.voice.ng4t.com SIP/2.0
			响应：SIP/2.0 401 Unauthorized
			找到第一个0x0d 0x0a的位置，	左边部分即为SIP Header部分
		*/
		startline := strings.Split(string(data[4:]), "\r\n")
		if len(startline) >= 1 {
			ss := strings.Split(startline[0], " ")
			if len(ss) == 3 {
				if len(ss[2]) == 3 { // 请求
					if strings.ToUpper(ss[2][:3]) == "SIP" {
						m._method = SipRequest
					}
				} else if len(ss) == 3 { // 响应
					if strings.ToUpper(ss[0][:3]) == "SIP" {
						m._method = SipResponse
					}
				}
				m._unique = binary.BigEndian.Uint32(data[0:4])
				m._protocal = SIPPROTOCAL
				m._size = uint16(len(data[4:]))
				copy(m._data[:], data[4:])
			}
		}
	}

}

func (msg *CommonMsg) Construct(_type, _method byte, size int, data []byte) {
	tmp := make([]byte, 1024)
	copy(tmp, data)
	msg._data = [65535]byte{}
	msg._protocal = _type
	msg._method = _method
	msg._size = uint16(size)
	copy(msg._data[:], tmp)
}

func (msg *CommonMsg) GetData() []byte {
	return msg._data[:msg._size]
}

func (msg *CommonMsg) GetIMSBody() []byte {
	uqi := [4]byte{}
	binary.BigEndian.PutUint32(uqi[:], msg._unique)
	return append(uqi[:], msg._data[:msg._size]...)
}

func (msg *CommonMsg) GetType() byte {
	return msg._protocal
}
func (msg *CommonMsg) GetReqID() uint32 {
	return msg._unique
}

// 打包 EPC 网络通用发送消息方法
func PackUpEpcMsg(msg *CommonMsg, _p, _m byte, data map[string]string, dest string, ra *net.UDPAddr, conn *net.UDPConn, out chan *Package) {
	res := StrLineMarshal(data)
	size := len([]byte(res))
	msg.Construct(_p, _m, size, []byte(res))
	if ra != nil && conn != nil {
		out <- &Package{msg, dest, ra, conn}
		return
	}
	out <- &Package{msg, dest, nil, nil}
}

func MAASyncResponse(msg *CommonMsg, _p, _m byte, data map[string]string, ra *net.UDPAddr, conn *net.UDPConn, out chan *Package) {
	res := StrLineMarshal(data)
	size := len([]byte(res))
	msg.Construct(_p, _m, size, []byte(res))
	out <- &Package{msg, "", ra, conn}
}

func MARSyncRequest(ctx context.Context, msg *CommonMsg, _p, _m byte, data map[string]string, dest string) (*CommonMsg, error) {
	res := StrLineMarshal(data)
	size := len([]byte(res))
	msg.Construct(_p, _m, size, []byte(res))
	buf := new(bytes.Buffer)
	buf.Grow(1024)
	binary.Write(buf, binary.BigEndian, msg)
	err, resp := sendUDPMessageWaitResp(ctx, dest, buf.Bytes())
	if err != nil {
		return nil, err
	}
	m := new(CommonMsg)
	m.Init(resp)
	return m, nil
}

// IMS 网络通用发送消息方法
func PackUpImsMsg(msg *CommonMsg, _p, _m byte, data []byte, dest string, ra *net.UDPAddr, conn *net.UDPConn, out chan *Package) {
	size := len(data)
	msg.Construct(_p, _m, size, data)
	if ra != nil && conn != nil {
		out <- &Package{msg, dest, ra, conn}
		return
	}
	out <- &Package{msg, dest, nil, nil}
}
