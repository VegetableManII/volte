package sip

import (
	"strings"
)

// SIP的头域字段名信息
type HeaderFieldItem struct {
	Name string // 正常字段名
	Abbr string // 简写字段名
}

// SIP的头域字段名
var (
	HeaderFieldVia             = HeaderFieldItem{"Via", "v"}
	HeaderFieldFrom            = HeaderFieldItem{"From", "f"}
	HeaderFieldTo              = HeaderFieldItem{"To", "t"}
	HeaderFieldCallID          = HeaderFieldItem{"Call-ID", "i"}
	HeaderFieldCSeq            = HeaderFieldItem{"CSeq", ""}
	HeaderFieldMaxForwards     = HeaderFieldItem{"Max-Forwards", ""}
	HeaderFieldContentType     = HeaderFieldItem{"Content-Type", "c"}
	HeaderFieldContentLength   = HeaderFieldItem{"Content-Length", "l"}
	HeaderFieldContact         = HeaderFieldItem{"Contact", "m"}
	HeaderFieldExpires         = HeaderFieldItem{"Expires", ""}
	HeaderFieldRoute           = HeaderFieldItem{"Route", ""}
	HeaderFieldRecordRoute     = HeaderFieldItem{"Record-Route", ""}
	HeaderFieldUserAgent       = HeaderFieldItem{"User-Agent", ""}
	HeaderFieldAuthorization   = HeaderFieldItem{"Authorization", ""}
	HeaderFieldWWWAuthenticate = HeaderFieldItem{"WWW-Authenticate", ""}
)

func (f HeaderFieldItem) LowerName() string {
	return strings.ToLower(f.Name)
}
