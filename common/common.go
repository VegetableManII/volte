package common

import (
	"context"
	"errors"
	"strings"

	"github.com/wonderivan/logger"
)

// 异常恢复
func Recover(ctx context.Context, host string, in, up, down chan *Package,
	recvf func(ctx context.Context, host string, in chan *Package),
	procdownf func(ctx context.Context, down chan *Package),
	procupf func(ctx context.Context, up chan *Package),
	proccore func(ctx context.Context, in, up, down chan *Package)) {
	defer func() {
		err := recover()
		if err != nil {
			logger.Error("程序异常Panic %v", err)
		}
	}()
	go recvf(ctx, host, in)
	go procdownf(ctx, down)
	go procupf(ctx, up)
	go proccore(ctx, in, up, down)
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

// EPC 网络通用发送消息方法
func PackageOut(protocal, method byte, data map[string]string, dest string, out chan *Package) {
	cmsg := new(CommonMsg)
	res := StrLineMarshal(data)
	size := len([]byte(res))
	cmsg.Construct(protocal, method, size, []byte(res))
	out <- &Package{cmsg, dest}
}

// IMS 网络通用发送消息方法
func RawPackageOut(protocal, method byte, data []byte, dest string, out chan *Package) {
	cmsg := new(CommonMsg)
	size := len(data)
	cmsg.Construct(protocal, method, size, data)
	out <- &Package{cmsg, dest}

}

func GetIMSI(data []byte) (string, error) {
	m := StrLineUnmarshal(data)
	imsi, ok := m["imsi"]
	if !ok {
		return "", errors.New("ErrEmptyIMSI")
	}
	return imsi, nil
}
