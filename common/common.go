package common

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
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
		logger.Error("Stack Informmation %s", string(data[:n]))
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

// EPC 网络通用发送消息方法
func PackageOut(protocal, method byte, data map[string]string, dest string, out chan *Package) {
	cmsg := new(CommonMsg)
	res := StrLineMarshal(data)
	size := len([]byte(res))
	cmsg.Construct(protocal, method, size, []byte(res))
	out <- &Package{cmsg, dest, nil, nil}
}

func MAASyncResponse(protocal, method byte, data map[string]string, ra *net.UDPAddr, conn *net.UDPConn, out chan *Package) {
	cmsg := new(CommonMsg)
	res := StrLineMarshal(data)
	size := len([]byte(res))
	cmsg.Construct(protocal, method, size, []byte(res))
	out <- &Package{cmsg, "", ra, conn}
}

func MARSyncRequest(ctx context.Context, protocal, method byte, data map[string]string, dest string) (*CommonMsg, error) {
	cmsg := new(CommonMsg)
	res := StrLineMarshal(data)
	size := len([]byte(res))
	cmsg.Construct(protocal, method, size, []byte(res))
	buf := new(bytes.Buffer)
	buf.Grow(1024)
	binary.Write(buf, binary.BigEndian, cmsg)
	err, resp := sendUDPMessageWaitResp(ctx, dest, buf.Bytes())
	if err != nil {
		return nil, err
	}
	m := new(CommonMsg)
	m.Init(resp)
	return m, nil
}

// IMS 网络通用发送消息方法
func RawPackageOut(protocal, method byte, data []byte, dest string, out chan *Package) {
	cmsg := new(CommonMsg)
	size := len(data)
	cmsg.Construct(protocal, method, size, data)
	out <- &Package{cmsg, dest, nil, nil}

}

func GetIMSI(data []byte) (string, error) {
	m := StrLineUnmarshal(data)
	imsi, ok := m["imsi"]
	if !ok {
		return "", errors.New("ErrEmptyIMSI")
	}
	return imsi, nil
}

func GenerateSipBranch() int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63()
}
