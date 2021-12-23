package service

// enodb 代码实现
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
	_data []byte
}

func (e *EpcMsg) Init(_type byte, msg byte, size [2]byte, imsi [4]byte, data []byte) {
	e._type = _type
	e._msg = msg
	e._size = size
	e._imsi = imsi
	e._data = data
}

// todo 定义sip消息结构
type SipMsg struct {
}
