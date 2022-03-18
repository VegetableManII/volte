package modules

import (
	"bytes"
	"encoding/binary"
	"net"
)

const CRLF = "\r\n"

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

type CommonMsg struct {
	_unique   uint32      // 全局唯一ID，供基站区分不同用户请求使用
	_protocal uint8       // 0x01 表示电路域协议
	_method   uint8       // 对应协议的不同请求响应方法
	_size     uint16      // data字段的长度
	_data     [65535]byte // 最大65535字节大小
}
type FixedConn string // 固定连接
type DynamicConn struct {
	remoteAddr *net.UDPAddr // 客户端地址
	conn       *net.UDPConn // 客户端动态连接
}
type Package struct {
	CommonMsg
	FixedConn
	DynamicConn
}

/*
EPC消息内容布局 byte
	| 0 | 1 | 2 | 3 |
	|    uniq_id    |
	| p | m | size  |
	|     data      |
SIP消息布局
	| 0 | 1 | 2 | 3 |
	|    uniq_id    |
	|     data      |

SIP Header 格式如下
	请求：REGISTER sip:apn.sip.voice.ng4t.com SIP/2.0
	响应：SIP/2.0 401 Unauthorized
	找到第一个\r\n的位置，	左边部分即为SIP Header部分

*/
// 接收消息时通过字节流创建Package
func (p *Package) Init(data []byte) error {
	// 填充消息字节数据
	if data[4] == EPCPROTOCAL {
		p._unique = binary.BigEndian.Uint32(data[0:4])
		l := binary.BigEndian.Uint16(data[6:8])
		p._protocal = data[4]
		p._method = data[5]
		p._size = l
		copy(p._data[:], data[8:l+8])
	} else {
		m, err := GetSipMethod(data[4:])
		if err != nil {
			return err
		}
		p._unique = binary.BigEndian.Uint32(data[0:4])
		p._protocal = SIPPROTOCAL
		p._method = m
		p._size = uint16(len(data[4:]))
		copy(p._data[:], data[4:])
	}
	return nil
}

func (p *Package) SetFixedConn(dst string) {
	p.FixedConn = FixedConn(dst)
}

func (p *Package) SetDynamicConn(addr *net.UDPAddr, conn *net.UDPConn) {
	p.remoteAddr = addr
	p.conn = conn
}

func (p *Package) GetFixedConn() string {
	return string(p.FixedConn)
}

func (p *Package) GetDynamicAddr() *net.UDPAddr {
	return p.DynamicConn.remoteAddr
}

func (p *Package) GetDynamicConn() *net.UDPConn {
	return p.DynamicConn.conn
}

// 发送消息时结构化创建Package
func (p *Package) Construct(_type, _method byte, body string) {
	// 消息构建
	p._protocal = _type
	p._method = _method
	size := len([]byte(body))
	if size == 0 { // 消息转发，内容不需要改变
		return
	}
	p._data = [65535]byte{}
	p._size = uint16(size)
	copy(p._data[:], []byte(body))
}

func (p *Package) IsBeatHeart() bool {
	return p._unique == 0x0F0F0F0F && p._protocal == 0x0F &&
		p._method == 0x0F && p._size == 0x0F0F
}

func (p *Package) GetRoute() [2]byte {
	route := [2]byte{p._protocal, p._method}
	return route
}

// 获取消息的内容截断末尾的'\0'
func (msg *CommonMsg) GetData() []byte {
	return msg._data[:msg._size]
}

func (msg *CommonMsg) GetEpcMessage() []byte {
	buf := new(bytes.Buffer)
	buf.Grow(8 + int(msg._size))
	binary.Write(buf, binary.BigEndian, msg._unique)
	binary.Write(buf, binary.BigEndian, msg._protocal)
	binary.Write(buf, binary.BigEndian, msg._method)
	binary.Write(buf, binary.BigEndian, msg._size)
	binary.Write(buf, binary.BigEndian, msg._data[:msg._size])
	return buf.Bytes()
}

func (msg *CommonMsg) GetSipMessage() []byte {
	uqi := [4]byte{}
	binary.BigEndian.PutUint32(uqi[:], msg._unique)
	return append(uqi[:], msg._data[:msg._size]...)
}
