package sip

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Header中的CSeq序列号(RFC3261-8.1.1.5)
type CSeq struct {
	CSeq   int    // 请求的序列号
	Method string // 请求的方法
}

func parseCSeq(value string) (m CSeq, err error) {
	m = CSeq{}
	err = m.parse(value)
	return
}

// 字符串表达
func (m CSeq) String() string {
	return fmt.Sprintf("%d %s", m.CSeq, m.Method)
}

// 解析CSeq序列号
func (m *CSeq) parse(value string) (err error) {
	// 拆分字符串
	args := strings.Split(value, " ")
	if len(args) != 2 {
		err = errors.New("sip: message header cseq format error")
		return
	}
	// 生成
	if m.CSeq, err = strconv.Atoi(args[0]); err != nil {
		err = errors.New("sip: message header cseq number error")
		return
	}
	m.Method = args[1]
	return
}
