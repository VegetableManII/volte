package common

type Msg struct {
	Type  byte // 0x01 epc 0x00 ims
	Data1 *EpcMsg
	Data2 *SipMsg
}

// epc网络电路协议消息结构
type EpcMsg struct {
	_type byte // 0x01 表示电路域协议
	_msg  byte
	// 0x00 Attach Request
	// 0x01 Authentication Informat Request
	// 0x02 Authentication Informat Response
	// 0x03 Authentication Request
	// 0x04 Authentication Response
	// 0x05 Update Location Request
	// 0x06 Update Location ACK
	// 0x07 Create Session Request
	// 0x08 Create Session Response
	// 0x09 QCI *
	// 0x0A Attach Accept
	_size [2]byte // data字段的长度
	_imsi [4]byte // IMSI
	_data [24]byte
}

func (e *EpcMsg) Init(data []byte) {
	e._type = 0x01
	e._msg = data[1]
	copy(e._size[:], data[2:4])
	copy(e._imsi[:], data[4:8])
	copy(e._data[:24], data[8:])
}

// todo 定义sip消息结构
type SipMsg struct {
}

func (s *SipMsg) Init() {

}
