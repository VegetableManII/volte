package sip

import (
	"fmt"
	"strings"
)

// Via信息列表
type ViaList struct {
	value             []Via  // 内部实际的值
	receivedTransport string // 接收时的协议
	receivedAddr      string // 接收时的实际地址
}

// 添加一条记录
func (vl *ViaList) Add(str string) (err error) {
	item, err := parseVia(str)
	if err != nil {
		return
	}
	vl.value = append(vl.value, item)
	return
}

// (接收消息) 设置信息
func (vl *ViaList) SetReceivedInfo(transport string, address string) {
	vl.receivedTransport = transport
	vl.receivedAddr = address
}

// 获取第一个Via中的地址信息用于转发，bool代表是否使用了实际地址
func (vl ViaList) FirstAddrInfo() (string, bool) {
	via := vl.value[0]
	ip, _ := via.Arguments.Get("received")
	port, _ := via.Arguments.Get("rport")
	if len(ip) > 0 && len(port) > 0 {
		return fmt.Sprintf("%s:%s", ip, port), true
	} else {
		return via.Client, false
	}
}

// (转发请求) 设置第一个Via的接收实际信息
func (vl *ViaList) UpdateReceivedInfo() {
	ipPort := strings.Split(vl.receivedAddr, ":")
	// FIXME 更好的方式
	if vl.value[0].Transport != "TCP" && vl.value[0].Transport != "UDP" {
		vl.value[0].Transport = "UDP"
	}
	vl.value[0].Arguments.Set("received", ipPort[0])
	vl.value[0].Arguments.Set("rport", ipPort[1])
	return
}

// (转发请求) 添加当前服务器的信息到Via的开头
func (vl *ViaList) AddServerInfo() {
	via := Via{
		SIPVersion: SIPVersion,
		Transport:  strings.ToUpper(vl.receivedTransport),
		Client:     ServerDomainHost(),
		Arguments: NewArgs(map[string]string{
			"branch": vl.TransactionBranch(),
			// "branch": "z9hG4bK" + util.GenerateNonce(29),
			"rport": "",
		}),
	}
	vl.value = append([]Via{via}, vl.value...)
	return
}

// (转发应答) 移除第一个是自己服务器的Via
func (vl *ViaList) RemoveFirst() {
	via := vl.value[0]
	if via.Client == ServerDomainHost() || via.Client == ServerDomain {
		vl.value = vl.value[1:]
	}
	return
}

// 获取当前事务标识
func (vl ViaList) TransactionBranch() (result string) {
	result, _ = vl.value[0].Arguments.Get("branch")
	return
}
