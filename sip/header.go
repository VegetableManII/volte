package sip

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// SIP头域(RFC3261-7.3)
// 行范式：header = "header-name" HCOLON header-value *(COMMA header-value)
// 头域至少包含TO、FROM、CSeq、Call-ID、Max-Forwards、Via字段
type Header struct {
	Via             ViaList     // (RFC3261-8.1.1.7) 请求路径
	From            User        // (RFC3261-8.1.1.3) 请求的原始发起者
	To              User        // (RFC3261-8.1.1.2) 请求的原始到达者
	CallID          string      // (RFC3261-8.1.1.4) 唯一标志
	CSeq            CSeq        // (RFC3261-8.1.1.5) 命令序列号
	MaxForwards     MaxForwards // (RFC3261-8.1.1.6) 最大转发数量限制
	ContentLength   int         // 正文长度
	ContentType     string      // (可选) 正文格式描述
	Contact         *User       // (可选) (RFC3261-8.1.1.8) 直接访问方式
	Expires         Expires     // (可选) 消息或内容过期时间
	Route           Route       // (可选) 请求的路由表
	RecordRoute     RecordRoute // (可选) 后续消息流处理的服务器列表
	UserAgent       string      // (可选) UAC的信息
	Authorization   string      // (可选) 用户认证信息
	WWWAuthenticate string      // (可选) 支持的认证方式和适用realm的参数的拒绝原因
	UnsupportLines  []string    // 暂不支持的行
}

// 设置To的标签为From标签
func (h *Header) UpdateToTagWithFromTag() {
	if v, e := h.From.Arguments.Get("tag"); e == nil {
		h.To.Arguments.Set("tag", v)
	}
}

// 字符串表达
func (h Header) String() (result string) {
	// FIXME 采用逗号方式
	for _, via := range h.Via.value {
		result += h.lineString(HeaderFieldVia.Name, via.String())
	}
	for _, route := range h.Route.value {
		result += h.lineString(HeaderFieldRoute.Name, route.String())
	}
	for _, recordRoute := range h.RecordRoute.value {
		result += h.lineString(HeaderFieldRecordRoute.Name, recordRoute.String())
	}
	result += h.lineString(HeaderFieldMaxForwards.Name, h.MaxForwards.String())
	result += h.lineString(HeaderFieldFrom.Name, h.From.String())
	result += h.lineString(HeaderFieldTo.Name, h.To.String())
	result += h.lineString(HeaderFieldCallID.Name, h.CallID)
	result += h.lineString(HeaderFieldCSeq.Name, h.CSeq.String())
	result += h.lineString(HeaderFieldContentLength.Name, fmt.Sprintf("%d", h.ContentLength))
	if len(h.ContentType) > 0 {
		result += h.lineString(HeaderFieldContentType.Name, h.ContentType)
	}
	if h.Contact != nil {
		result += h.lineString(HeaderFieldContact.Name, h.Contact.String())
	}
	result += h.lineString(HeaderFieldExpires.Name, h.Expires.String())
	if len(h.UserAgent) > 0 {
		result += h.lineString(HeaderFieldUserAgent.Name, h.UserAgent)
	}
	if len(h.Authorization) > 0 {
		result += h.lineString(HeaderFieldAuthorization.Name, h.Authorization)
	}
	if len(h.WWWAuthenticate) > 0 {
		result += h.lineString(HeaderFieldWWWAuthenticate.Name, h.WWWAuthenticate)
	}
	for _, line := range h.UnsupportLines {
		result += h.emptyLineString(line)
	}
	return
}

// 解析消息头的单个行
func (h *Header) parse(line string) (err error) {
	// 解析冒号的位置
	keyPosition := strings.Index(line, ":")
	if keyPosition == -1 {
		err = errors.New("sip: message header line no colon")
		return
	}
	// 解析key和value并设置到结果中
	key := strings.ToLower(strings.TrimSpace(line[:keyPosition]))
	value := strings.TrimSpace(line[keyPosition+1:])
	switch key {
	case HeaderFieldVia.LowerName(), HeaderFieldVia.Abbr:
		err = h.Via.Add(value)
	case HeaderFieldFrom.LowerName(), HeaderFieldFrom.Abbr:
		h.From, err = parseUser(value)
	case HeaderFieldTo.LowerName(), HeaderFieldTo.Abbr:
		h.To, err = parseUser(value)
	case HeaderFieldCallID.LowerName(), HeaderFieldCallID.Abbr:
		h.CallID = value
	case HeaderFieldCSeq.LowerName(), HeaderFieldCSeq.Abbr:
		h.CSeq, err = parseCSeq(value)
	case HeaderFieldMaxForwards.LowerName(), HeaderFieldMaxForwards.Abbr:
		h.MaxForwards, err = parseMaxForwards(value)
	case HeaderFieldContentLength.LowerName(), HeaderFieldContentLength.Abbr:
		h.ContentLength, err = strconv.Atoi(value)
	case HeaderFieldContentType.LowerName(), HeaderFieldContentType.Abbr:
		h.ContentType = value
	case HeaderFieldContact.LowerName(), HeaderFieldContact.Abbr:
		if contact, e := parseUser(value); e != nil {
			err = e
		} else {
			h.Contact = &contact
		}
	case HeaderFieldExpires.LowerName(), HeaderFieldExpires.Abbr:
		h.Expires, err = parseExpires(value)
	case HeaderFieldRoute.LowerName(), HeaderFieldRoute.Abbr:
		h.Route, err = parseRoute(value, h.Route)
	case HeaderFieldRecordRoute.LowerName(), HeaderFieldRecordRoute.Abbr:
		h.RecordRoute, err = parseRecordRoute(value, h.RecordRoute)
	case HeaderFieldUserAgent.LowerName(), HeaderFieldUserAgent.Abbr:
		h.UserAgent = value
	case HeaderFieldAuthorization.LowerName(), HeaderFieldAuthorization.Abbr:
		h.Authorization = value
	case HeaderFieldWWWAuthenticate.LowerName(), HeaderFieldWWWAuthenticate.Abbr:
		h.WWWAuthenticate = value
	default:
		h.UnsupportLines = append(h.UnsupportLines, line)
	}
	return
}

// 单行输出，兼容空行
func (h Header) lineString(key string, value string) string {
	if len(value) == 0 {
		return ""
	}
	return key + ": " + value + CRLF
}

func (h Header) emptyLineString(line string) string {
	return line + CRLF
}
