package sip

import (
	"encoding/json"
)

// 行分隔符(CRLF)
const CRLF = "\r\n"

// 是否是有效的行(CRLF)
func IsValidLine(line string) bool {
	l := len(line)
	return l >= 2 && line[l-2:] == CRLF
}

// 移除行的CRLF标志
func RemoveLineCRLF(line string) string {
	return line[:len(line)-2]
}

// TODO 克隆一个对象
func Clone(src interface{}, target interface{}) {
	str, _ := json.Marshal(src)
	_ = json.Unmarshal(str, target)
}
