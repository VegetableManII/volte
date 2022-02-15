package sip

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Via请求行的内容(RFC3261-8.1.1.7)
// Example：SIP/2.0/UDP 192.168.0.100:43188;branch=z9hG4bK111643fe9a9f389667c5e7d8873;rport
type Via struct {
	SIPVersion string // SIP版本号
	Transport  string // 协议
	Client     string // 客户端地址
	Arguments  Args   // 参数列表
}

func parseVia(str string) (item Via, err error) {
	str = strings.TrimSpace(str)
	item = Via{}
	err = item.parse(str)
	return
}

// 字符串输出
func (v Via) String() string {
	return fmt.Sprintf("%s/%s %s%s", v.SIPVersion, v.Transport, v.Client, v.Arguments.String())
}

// 解析Via信息
func (v *Via) parse(str string) (err error) {
	result := viaRegExp.FindStringSubmatch(str)
	if len(result) != 5 {
		err = errors.New("sip: message header via format error")
		return
	}
	v.SIPVersion = strings.TrimSpace(result[1])
	v.Transport = strings.TrimSpace(result[2])
	v.Client = strings.TrimSpace(result[3])
	v.Arguments = parseArgs(result[4])
	return
}

var viaRegExp = regexp.MustCompile("^(SIP\\/[^\\/]+)\\/([^ ]+) ([^;]+)(.+)$")
