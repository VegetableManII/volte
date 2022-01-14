package common

const (
	EPSPROTOCAL byte = 0x01
	SIPPROTOCAL byte = 0x02
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

type Msg struct {
	Type  byte // 0x01 eps 0x00 ims
	Data1 *EpsMsg
	Data2 *SipMsg
}

// eps网络电路协议消息结构
type EpsMsg struct {
	_type byte // 0x01 表示电路域协议
	_msg  byte
	_size [2]byte    // data字段的长度
	_imsi [4]byte    // IMSI
	_data [1016]byte // 最大65535字节大小
}

func (e *EpsMsg) Init(data []byte) {
	e._type = EPSPROTOCAL
	e._msg = data[1]
	copy(e._size[:], data[2:4])
	copy(e._imsi[:], data[4:8])
	copy(e._data[:24], data[8:])
}

func (e *EpsMsg) Construct(t, m byte, s [2]byte, i [4]byte, d []byte) {
	e._type = t
	e._msg = m
	e._size = s
	copy(e._data[:], d)
}

func (e *EpsMsg) GetType() byte {
	return e._type
}

func (e *EpsMsg) GetIMSI() [4]byte {
	return e._imsi
}

func (e *EpsMsg) GetData() []byte {
	return e._data[:]
}

// todo 定义sip消息结构
type SipMsg struct {
}

func (s *SipMsg) Init() {

}
