package common

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
)

// 按行分割，获取键值对内容
func StrLineUnmarshal(d []byte) map[string]string {
	m := make(map[string]string, 1)

	s := string(d)
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		kv := strings.Split(line, "=")
		if len(kv) != 2 {
			m[kv[0]] = ""
		}
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
func WrapOutEPS(protocal, method byte, data map[string]string, dest bool, out chan *Msg) {
	down := new(EpsMsg)
	res := StrLineMarshal(data)
	size := [2]byte{}
	l := len([]byte(res))
	binary.BigEndian.PutUint16(size[:], uint16(l+4))
	down.Construct(protocal, method, size, []byte(res))

	wrap := new(Msg)
	wrap.Type = EPSPROTOCAL
	wrap.Destation = dest
	wrap.Data1 = down
	out <- wrap
}

func GetIMSI(data []byte) (int, error) {
	m := StrLineUnmarshal(data)
	value := m["imsi"]
	imsi, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("ErrIMSIDataFormat")
	}
	return imsi, nil
}
