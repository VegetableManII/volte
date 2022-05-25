package sip

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// SIP通用消息(RFC3261-7)
type Message struct {
	IsRequest    bool         // (二选一) 是否是请求
	IsResponse   bool         // (二选一) 是否是应答
	RequestLine  RequestLine  // 请求行
	ResponseLine ResponseLine // 应答行
	Header       Header       // 消息包头
	Body         string       // 消息正文(RFC3261-7.4)
}

func NewMessage(rd io.Reader) (msg Message, err error) {
	msg = Message{}
	err = msg.read(rd)
	return
}

func NewResponse(code StatusCodeItem, req *Message) *Message {
	return &Message{
		IsRequest:    false,
		IsResponse:   true,
		ResponseLine: NewResponseLineWithStatusCode(code),
		Header: Header{
			Via:               req.Header.Via,
			From:              req.Header.From,
			To:                req.Header.To,
			CSeq:              req.Header.CSeq,
			CallID:            req.Header.CallID,
			MaxForwards:       req.Header.MaxForwards,
			AccessNetworkInfo: req.Header.AccessNetworkInfo,
			ContentLength:     req.Header.ContentLength,
			ContentType:       req.Header.ContentType,
			Contact:           req.Header.Contact,
			Expires:           req.Header.Expires,
			Route:             req.Header.Route,
			RecordRoute:       req.Header.RecordRoute,
			UserAgent:         req.Header.UserAgent,
		},
		Body: "",
	}
}

// 字符串表示
func (m *Message) String() (result string) {
	// 输出首行
	if m.IsRequest {
		result += m.RequestLine.String()
	} else {
		result += m.ResponseLine.String()
	}
	result += CRLF
	// 更新内容长度
	bodyString := ""
	if m.Body != "" {
		bodyString = m.Body
	}
	m.Header.ContentLength = len(bodyString)
	// 写入Header
	result += m.Header.String()
	// 写入分隔的空行
	result += CRLF
	// 写入Body
	result += bodyString
	return
}

// 获取连接协议
func (m Message) Transport() string {
	return m.Header.Via.receivedTransport
}

// 获取连接协议
func (m Message) RealAddress() string {
	return m.Header.Via.receivedAddr
}

// 从IO流中读取SIP通用消息
func (m *Message) read(rd io.Reader) (err error) {
	buf := bufio.NewReaderSize(rd, 65536)
	// 读取起始行
	firstLine, err := m.readLine(buf)
	if err != nil {
		return
	}
	m.IsResponse = len(firstLine) >= 3 && firstLine[0:3] == "SIP"
	m.IsRequest = !m.IsResponse
	if m.IsRequest {
		m.RequestLine, err = NewRequestLine(firstLine)
	} else {
		m.ResponseLine, err = NewResponseLine(firstLine)
	}
	// 读取并解析消息头
	for {
		// 读取一行
		var line string
		line, err = m.readLine(buf)
		// 空行分隔Header和Body
		if len(line) == 0 {
			break
		}
		// 让Header解析行内容
		if err = m.Header.parse(line); err != nil {
			return
		}
	}
	// 生成消息体
	contentLength := m.Header.ContentLength
	if contentLength == 0 {
		m.Body = ""
		return
	}
	bodyBuffer := make([]byte, contentLength)
	n, err := buf.Read(bodyBuffer)
	if err != nil {
		err = errors.New("sip: message body read faild")
		return
	}
	if n != contentLength {
		err = errors.New("sip: message body read length error")
		return
	}
	m.Body = string(bodyBuffer)
	return
}

// 读取一行
func (m Message) readLine(buf *bufio.Reader) (line string, err error) {
	// 从缓冲区读取数据
	line, err = buf.ReadString('\n')
	if err != nil {
		err = errors.New("sip: message line read failed")
		return
	}
	if !IsValidLine(line) {
		err = errors.New("sip: message line no crlf" + line)
		return
	}
	line = RemoveLineCRLF(line)
	line = strings.TrimSpace(line)
	return
}
