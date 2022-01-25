package common

import (
	"encoding/binary"
	"strings"
)

const (
	EPCPROTOCAL byte = 0x01
	SIPPROTOCAL byte = 0x00
)

// epc message的消息类型
const (
	AttachRequest                  byte = 0x00
	AuthenticationInformatRequest  byte = 0x01
	AuthenticationInformatResponse byte = 0x02
	AuthenticationRequest          byte = 0x03
	AuthenticationResponse         byte = 0x04
	UpdateLocationRequest          byte = 0x05
	UpdateLocationACK              byte = 0x06
	CreateSessionRequest           byte = 0x07
	CreateSessionResponse          byte = 0x08
	QCI                            byte = 0x09
	AttachAccept                   byte = 0x0A
	UserAuthorizationRequest       byte = 0x0B
	UserAuthorizationAnswer        byte = 0x0C
)

// sip message的消息类型
const (
	SipRequest  byte = 0x00
	SipResponse byte = 0x01
)

type Package struct {
	*CommonMsg
	Destation string // true上行、false下行
}
type CommonMsg struct {
	_type   uint8 // 0x01 表示电路域协议
	_method uint8
	_size   uint16     // data字段的长度
	_data   [1020]byte // 最大65535字节大小
}

func (p *Package) GetUniqueMethod() [2]byte {
	uniq := [2]byte{p._type, p._method}
	return uniq
}

func (e *CommonMsg) Init(data []byte) {
	if data[0] == EPCPROTOCAL {
		l := binary.BigEndian.Uint16(data[2:4])
		e._type = data[0]
		e._method = data[1]
		e._size = l
		copy(e._data[:], data[4:l+4])
	} else {
		// SIP Header 格式如下
		/*
			请求：REGISTER sip:apn.sip.voice.ng4t.com SIP/2.0
			响应：SIP/2.0 401 Unauthorized
			找到第一个0x0d 0x0a的位置，	左边部分即为SIP Header部分
		*/
		startline := strings.Split(string(data), "\r\n")
		if len(startline) >= 1 {
			ss := strings.Split(startline[0], " ")
			if len(ss) == 3 {
				if len(ss[2]) == 3 { // 请求
					if strings.ToUpper(ss[2][:3]) == "SIP" {
						e._method = SipRequest
					}
				} else if len(ss) == 3 { // 响应
					if strings.ToUpper(ss[0][:3]) == "SIP" {
						e._method = SipResponse
					}
				}
				e._type = SIPPROTOCAL
				e._size = uint16(len(data))
				copy(e._data[:], data)
			}
		}

	}

}

func (e *CommonMsg) Construct(t, m byte, s int, d []byte) {
	e._type = t
	e._method = m
	e._size = uint16(s)
	copy(e._data[:], d[0:e._size])
}

func (e *CommonMsg) GetData() []byte {
	return e._data[:e._size]
}

func (e *CommonMsg) GetType() byte {
	return e._type
}
