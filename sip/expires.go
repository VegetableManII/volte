package sip

import (
	"fmt"
	"strconv"
)

// 消息或内容过期时间
type Expires struct {
	value *int // 实际的值
}

func parseExpires(str string) (item Expires, err error) {
	expires, err := strconv.Atoi(str)
	if err != nil {
		return
	}
	item = Expires{
		value: &expires,
	}
	return
}

// 是否是注销请求
func (e Expires) IsRequestLogOut() bool {
	return e.value != nil && *(e.value) == 0
}

// 字符串表达
func (e Expires) String() string {
	if e.value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *(e.value))
}
