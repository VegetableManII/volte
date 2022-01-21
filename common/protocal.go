package common

import (
	"encoding/binary"
)

const (
	EPSPROTOCAL byte = 0x01
	SIPPROTOCAL byte = 0x00
)

// eps message的消息类型
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
)

// sip message的消息类型
const (
	INVITE    byte = 0x00
	ACK       byte = 0x01
	BYE       byte = 0x02
	CANCEL    byte = 0x03
	OPTIONS   byte = 0x04
	PRACK     byte = 0x05
	SUBSCRIBE byte = 0x06
	NOTIFY    byte = 0x07
	PUBLISH   byte = 0x08
	INFO      byte = 0x09
	MESSAGE   byte = 0x0A
	UPDATE    byte = 0x0B
	REGISTER  byte = 0x0C
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
	l := binary.BigEndian.Uint16(data[2:4])
	e._type = data[0]
	e._method = data[1]
	e._size = l
	copy(e._data[:], data[4:l+4])
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
