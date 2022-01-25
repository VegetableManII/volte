package sip

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// 头信息参数，在Header中，或者在某个字段中的key[=value]形式的参数
type Args struct {
	keys   []string
	values []string
}

func NewArgs(kvMap map[string]string) (item Args) {
	item = Args{}
	for k, v := range kvMap {
		item.keys = append(item.keys, k)
		item.values = append(item.values, v)
	}
	return
}

func parseArgs(str string) (item Args) {
	str = strings.TrimSpace(str)
	item = Args{}
	item.parse(str, ";")
	return
}

func ParseArgsComma(str string) (item Args) {
	str = strings.TrimSpace(str)
	item = Args{}
	item.parse(str, ",")
	return
}

// 获取键对应的值
func (h Args) Get(key string) (value string, err error) {
	for i, k := range h.keys {
		if k == key {
			value = h.values[i]
			return
		}
	}
	err = errors.New("sip: args no value for key")
	return
}

// 设置键对应的值
func (h *Args) Set(key string, value string) {
	for i, k := range h.keys {
		if k == key {
			h.values[i] = value
			return
		}
	}
	h.keys = append(h.keys, key)
	h.values = append(h.values, value)
}

// 使用分号开头，用key[=value]方式，通过分号拼接成字符串
func (h Args) String() string {
	return h.customString(func(key string, value string) string {
		if len(value) == 0 {
			return fmt.Sprintf(";%s", key)
		} else {
			return fmt.Sprintf(";%s=%s", key, value)
		}
	})
}

// 通过逗号和空格，拼接成字符串
func (h Args) CommaString() (result string) {
	result = h.customString(func(key string, value string) string {
		return fmt.Sprintf(",%s=%s", key, strconv.Quote(value))
	})
	result = result[1:]
	return
}

// 从一个完整的头信息中解析key/value对
func (h *Args) parse(str string, seperator string) {
	argItems := strings.Split(str, seperator)
	if len(argItems) < 2 {
		return
	}
	for _, argItem := range argItems[1:] {
		tmp := strings.TrimSpace(argItem)
		if len(tmp) == 0 {
			continue
		}
		key, value := h.parseItem(tmp)
		h.keys = append(h.keys, key)
		h.values = append(h.values, value)
	}
}

// 解析一个key[=value]字符串
func (h Args) parseItem(item string) (key string, value string) {
	if i := strings.Index(item, "="); i < 0 {
		key = item
	} else {
		key = item[:i]
		value = item[i+1:]
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
	}
	return
}

// 自定义输出
func (h Args) customString(f func(string, string) string) (result string) {
	if len(h.keys) == 0 {
		return
	}
	for i, key := range h.keys {
		value := h.values[i]
		result += f(key, value)
	}
	return
}
