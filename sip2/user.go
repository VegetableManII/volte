package sip

import (
	"fmt"
	"regexp"
	"strings"
)

// SIP用户信息
// Example："1011" <sip:1011@192.168.0.2>;tag=213788011b
type User struct {
	DisplayName string // 显示姓名
	URI         URI    // RequestURI地址
	Arguments   Args   // 参数
}

func parseUser(str string) (item User, err error) {
	str = strings.TrimSpace(str)
	item = User{}
	err = item.parse(str)
	return
}

// 字符串输出
func (u User) String() (result string) {
	if len(u.DisplayName) > 0 {
		result += fmt.Sprintf("\"%s\"", u.DisplayName)
	}
	result += fmt.Sprintf("<%s>%s", u.URI.String(), u.Arguments.String())
	return
}

// 获取用户名
func (u User) Username() string {
	return u.URI.Username
}

// 解析SIP用户
func (u *User) parse(str string) (err error) {
	var uri URI
	result := nameRegexp.FindStringSubmatch(str)
	if len(result) == 0 {
		if uri, err = NewURI(str); err != nil {
			return
		}
		u.URI = uri
		u.Arguments = parseArgs("") // FIXME 这里可能也有参数
		return
	}
	u.DisplayName = strings.Trim(strings.TrimSpace(result[1]), "\"")
	if uri, err = NewURI(result[2]); err != nil {
		return
	}
	u.URI = uri
	u.Arguments = parseArgs(result[3])
	return
}

var nameRegexp = regexp.MustCompile("^([^<]*)<([^>]+)>(.*)$")
