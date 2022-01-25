package sip

import (
	"errors"
	"fmt"
	"strings"
)

// SIP请求消息的起始行(RFC3261-7.1)
type RequestLine struct {
	Method     string // 请求方法
	RequestURI URI    // SIP消息目标地址，无"<>"尖括号
	SIPVersion string // SIP版本号
}

func NewRequestLine(str string) (rl RequestLine, err error) {
	rl = RequestLine{}
	err = rl.parse(str)
	return
}

// 获取请求的用户名
func (rl RequestLine) Username() string {
	return rl.RequestURI.Username
}

// 更新URI地址
func (rl *RequestLine) UpdateRequestURI(domain string) {
	rl.RequestURI.Domain = domain
}

// 字符串表达
func (rl RequestLine) String() string {
	return fmt.Sprintf("%s %s %s", rl.Method, rl.RequestURI.String(), rl.SIPVersion)
}

// 解析起始行
func (rl *RequestLine) parse(line string) (err error) {
	// 使用空格拆分
	args := strings.Split(line, " ")
	if len(args) != 3 {
		err = errors.New("sip: message request line format error")
		return
	}
	// 生成
	rl.Method = args[0]
	if rl.RequestURI, err = NewURI(args[1]); err != nil {
		return
	}
	rl.SIPVersion = args[2]
	return
}
