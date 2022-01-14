package common

import (
	"strings"
)

type CtxString string

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
	for k, v := range m {
		line += k + "=" + v + "\r\n"
	}
	return line
}
