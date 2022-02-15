package common

import (
	"context"
	"errors"
	"math/rand"
	"runtime"
	"strings"
	"sync"
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

func GetIMSI(data []byte) (string, error) {
	m := StrLineUnmarshal(data)
	imsi, ok := m["imsi"]
	if !ok || imsi == "" {
		return "", errors.New("ErrEmptyIMSI")
	}
	return imsi, nil
}

func GenerateSipBranch() int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63()
}

type UniqueRequest struct {
	ReqID uint32
	sync.Mutex
}

var uniqReq UniqueRequest

func init() {
	uniqReq.ReqID = 0xFFFFFFFF
}

func GenerateRequestID() uint32 {
	uniqReq.Lock()
	tmp := uniqReq.ReqID
	uniqReq.ReqID = uniqReq.ReqID - 1
	uniqReq.Unlock()
	return tmp
}
