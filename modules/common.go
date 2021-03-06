package modules

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"time"

	"github.com/wonderivan/logger"
)

// 异常恢复
func Recover(ctx context.Context) {
	err := recover()
	if err != nil {
		logger.Error("[%v] 程序异常Panic %v", ctx.Value("Entity"), err)
		data := make([]byte, 2048)
		n := runtime.Stack(data[:], false)
		logger.Error("程序堆栈信息 %s", string(data[:n]))
	}
}

// 按行分割，获取键值对内容，需要明确给出字节流边界，防止把 ‘\0’ 字符识别为字符串中的字符
func StrLineUnmarshal(d []byte) map[string]string {
	m := make(map[string]string, 1)

	s := string(d)
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		kv := strings.Split(line, "=")
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

// 将kv关系转换为按行分割的字符串内容
func StrLineMarshal(m map[string]string) string {
	line := ""
	if m == nil {
		return line
	}
	lines := make([]string, 0, len(m))
	for k, v := range m {
		lines = append(lines, k+"="+v)
	}
	return strings.Join(lines, "\r\n")
}

// 生成sip消息的唯一branch
func GenerateSipBranch() int64 {
	return time.Now().UnixNano()
}

// 判断Package包中是否存在连接
func ConnectionExist(p *Package) bool {
	return p.longc.remoteAddr != nil && p.longc.conn != nil
}

// 判断sip消息的是请求还是响应
func GetSipMethod(data []byte) (byte, error) {
	startline := strings.Split(string(data), CRLF)
	if len(startline) >= 1 {
		ss := strings.Split(startline[0], " ")
		if len(ss) >= 3 && strings.ToUpper(ss[0][:3]) == "SIP" {
			return SipResponse, nil
		} else {
			return SipRequest, nil
		}
	}
	return 0, errors.New("ErrInvalidStartLine")
}
