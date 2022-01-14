package common

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

type Msg struct {
	Type  byte // 0x01 eps 0x00 ims
	Data1 *EpsMsg
	Data2 *SipMsg
}

func (m *Msg) GetUniqueMethod() [2]byte {
	uniq := [2]byte{}
	if m.Type == EPSPROTOCAL {
		uniq[0] = m.Data1._type
		uniq[1] = m.Data1._msg
	} else {
		uniq[0] = m.Data2._type
		uniq[1] = m.Data2._msg
	}
	return uniq
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

func (e *EpsMsg) GetIMSI() [4]byte {
	return e._imsi
}

func (e *EpsMsg) GetData() []byte {
	return e._data[:]
}

// todo 定义sip消息结构
type SipMsg struct {
	_type byte // 0x01 表示电路域协议
	_msg  byte
	_size [2]byte    // data字段的长度
	_data [1020]byte // 最大65535字节大小
}

func (s *SipMsg) Init(data []byte) {
	s._type = SIPPROTOCAL
	s._msg = data[1]
	copy(s._size[:], data[2:4])
	copy(s._data[:], data[4:])
}
