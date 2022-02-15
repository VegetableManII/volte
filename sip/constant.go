package sip

const (
	// SIP版本号
	SIPVersion = "SIP/2.0"

	// 协议类型
	SchemeSip = "sip"

	// SIP请求方法-RFC3261
	MethodInvite   = "INVITE"
	MethodAck      = "ACK"
	MethodBye      = "BYE"
	MethodCancel   = "CANCEL"
	MethodOptions  = "OPTIONS"
	MethodRegister = "REGISTER"

	// SIP请求方法-其他协议
	MethodPrack     = "PRACK"     // [RFC3262]
	MethodSubscribe = "SUBSCRIBE" // [RFC6665]
	MethodNotify    = "NOTIFY"    // [RFC6665]
	MethodPublish   = "PUBLISH"   // [RFC3903]
	MethodInfo      = "INFO"      // [RFC6086]
	MethodRefer     = "REFER"     // [RFC3515]
	MethodMessage   = "MESSAGE"   // [RFC3428]
	MethodUpdate    = "UPDATE"    // [RFC3311]
	MethodPing      = "PING"      // [https://tools.ietf.org/html/draft-fwmiller-ping-03]
)
