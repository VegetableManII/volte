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
	BEATHEART   byte = 0x0F
)

// epc message的消息类型
const (
	AttachRequest byte = 0x00 // UE发起Attach请求
	// AuthenticationInformatRequest   byte = 0x01
	// AuthenticationInformatResponse  byte = 0x02
	// AuthenticationRequest           byte = 0x03 // 网络侧向UE发起，UE侧需要实现该接口
	// AuthenticationResponse          byte = 0x04 // UE响应网络侧，由UE实现
	// UpdateLocationRequest           byte = 0x05
	// UpdateLocationACK               byte = 0x06
	// CreateSessionRequest            byte = 0x07
	// CreateSessionResponse           byte = 0x08
	// QCI                             byte = 0x09
	AttachAccept                    byte = 0x0A // 网络侧向UE发起，通知附着成功
	UserAuthorizationRequest        byte = 0x0B
	UserAuthorizationAnswer         byte = 0x0C
	MultiMediaAuthenticationRequest byte = 0x0D
	MultiMediaAuthenticationAnswer  byte = 0x0E
)

// sip message的消息类型
const (
	SipRequest  byte = 0x00
	SipResponse byte = 0x01
)

type CommonMsg struct {
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
  0 | p | m | size  |
  1	|     data      |
SIP消息布局
	| 0 | 1 | 2 | 3 |
  0	|     data      |

SIP Header 格式如下
	请求：REGISTER sip:apn.sip.voice.ng4t.com SIP/2.0
	响应：SIP/2.0 401 Unauthorized
	找到第一个\r\n的位置，	左边部分即为SIP Header部分

*/
// 接收消息时通过字节流创建Package
func (p *Package) Init(data []byte) error {
	// 填充消息字节数据
	if data[0] == EPCPROTOCAL {
		l := binary.BigEndian.Uint16(data[2:4])
		p._protocal = data[0]
		p._method = data[1]
		p._size = l
		copy(p._data[:], data[4:l+4])
	} else {
		m, err := GetSipMethod(data)
		if err != nil {
			return err
		}
		p._protocal = SIPPROTOCAL
		p._method = m
		p._size = uint16(len(data))
		copy(p._data[:], data[:])
	}
	return nil
}

func (p *Package) SetFixedConn(dst string) {
	p.FixedConn = FixedConn(dst)
}

func (p *Package) SetDynamicConn(conn *net.UDPConn) {
	p.conn = conn
}

func (p *Package) SetDynamicAddr(addr *net.UDPAddr) {
	p.remoteAddr = addr
}

func (p *Package) DefaultDynamic() {
	p.conn = nil
	p.remoteAddr = nil
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
	return p._protocal == 0x0F &&
		p._method == 0x0F && p._size == 0x0F0F
}

func (p *Package) GetRoute() [2]byte {
	return [2]byte{p._protocal, p._method}
}

// 获取消息的内容截断末尾的'\0'
func (p *Package) GetData() []byte {
	return p._data[:p._size]
}

func (msg *CommonMsg) GetEpcMessage() []byte {
	buf := new(bytes.Buffer)
	buf.Grow(4 + int(msg._size))
	binary.Write(buf, binary.BigEndian, msg._protocal)
	binary.Write(buf, binary.BigEndian, msg._method)
	binary.Write(buf, binary.BigEndian, msg._size)
	binary.Write(buf, binary.BigEndian, msg._data[:msg._size])
	return buf.Bytes()
}

func (msg *CommonMsg) GetSipMessage() []byte {
	return msg._data[:msg._size]
}
