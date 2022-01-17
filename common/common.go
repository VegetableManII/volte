package common

import (
	"encoding/binary"
	"strings"

	"github.com/wonderivan/logger"
)

// 按行分割，获取键值对内容
func StrLineUnmarshal(d []byte) map[string]string {
	m := make(map[string]string, 1)

	s := string(d)
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		kv := strings.Split(line, "=")
		m[kv[0]] = kv[1]
	}
	return m
}

// 将kv关系转换为按行分割的字符串内容
func StrLineMarshal(m map[string]string) string {
	line := ""
	if m == nil {
		return line
	}
	for k, v := range m {
		line += k + "=" + v + "\r\n"
	}
	return line
}

// EPS 网络通用发送消息方法
func WrapOutEPS(protocal, method byte, imsi [4]byte, data map[string]string, out chan *Msg) {
	down := new(EpsMsg)
	res := StrLineMarshal(data)
	if res != "" {
		size := [2]byte{}
		l := len([]byte(res))
		binary.BigEndian.PutUint16(size[:], uint16(l+4))
		down.Construct(protocal, method, size, imsi, []byte(res))
	} else {
		size := [2]byte{0, 0}
		binary.BigEndian.PutUint16(size[:], uint16(4))
		logger.Debug("send data %v", imsi)
		down.Construct(protocal, method, size, imsi, []byte{0})
	}
	wrap := new(Msg)
	wrap.Type = EPSPROTOCAL
	wrap.Data1 = down
	logger.Debug("send data %v %v", wrap.Type, *wrap.Data1)
	out <- wrap
}
