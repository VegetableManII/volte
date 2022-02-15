package sip

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SIP应答消息的起始行(RFC3261-7.2)
type ResponseLine struct {
	SIPVersion   string // SIP版本号
	StatusCode   int    // 状态码
	ReasonPhrase string // 状态码的说明
}

func NewResponseLine(str string) (rl ResponseLine, err error) {
	rl = ResponseLine{}
	err = rl.parse(str)
	return
}

func NewResponseLineWithStatusCode(code StatusCodeItem) ResponseLine {
	return ResponseLine{
		SIPVersion:   SIPVersion,
		StatusCode:   code.Code,
		ReasonPhrase: code.Reason,
	}
}

// 字符串表达
func (rl ResponseLine) String() string {
	return fmt.Sprintf("%s %d %s", rl.SIPVersion, rl.StatusCode, rl.ReasonPhrase)
}

// 解析状态行
func (rl *ResponseLine) parse(line string) (err error) {
	result := responseLineRegExp.FindStringSubmatch(line)
	if len(result) != 4 {
		err = errors.New("sip: message response line format error")
		return
	}
	rl.SIPVersion = strings.TrimSpace(result[1])
	if rl.StatusCode, err = strconv.Atoi(strings.TrimSpace(result[2])); err != nil {
		err = errors.New("sip: message response line status code error")
		return
	}
	rl.ReasonPhrase = strings.TrimSpace(result[3])
	return
}

var responseLineRegExp = regexp.MustCompile("^([^\\s]+)\\s([0-9]+)\\s(.*)$")
