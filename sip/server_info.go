package sip

import (
	"fmt"
)

var (
	ServerIP     string // 区域IP
	ServerDomain string // 区域域名，用于Via
	ServerPort   int    // 区域端口号
)

// 区域名称，IP:Port格式
func ServerIpHost() string {
	return fmt.Sprintf("%s:%d", ServerIP, ServerPort)
}

// 区域名称，Domain:Port格式
func ServerDomainHost() string {
	return fmt.Sprintf("%s:%d", ServerDomain, ServerPort)
}
