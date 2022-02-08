package sip

import (
	"errors"
	"regexp"
	"strings"
)

// URI信息(RFC3261-19.1)
// 格式：sip:user:password@host:port;uri-parameters?headers
type URI struct {
	Scheme    string // 协议类型
	Username  string // 用户名
	Password  string // 密码
	Domain    string // 域或IP地址
	Arguments Args   // 参数列表
}

// TODO 增加URI列表的解析

func NewURI(str string) (item URI, err error) {
	str = strings.TrimSpace(str)
	item = URI{}
	err = item.parse(str)
	return
}

// 字符串输出
func (u URI) String() (result string) {
	result += u.Scheme + ":"
	if len(u.Username) > 0 {
		tmp := u.Username
		if len(u.Password) > 0 {
			tmp += ":" + u.Password
		}
		result += tmp + "@"
	}
	result += u.Domain + u.Arguments.String()
	return
}

// 判断是否相等
func (u URI) IsEqual(obj URI) bool {
	return u.Scheme == obj.Scheme && u.Username == obj.Username && u.Domain == obj.Domain
}

// 解析URI
func (u *URI) parse(str string) (err error) {
	result := uriRegExpWithUser.FindStringSubmatch(str)
	if len(result) != 5 {
		nuResult := uriRegExpNoUser.FindStringSubmatch(str)
		if len(nuResult) != 4 {
			err = errors.New("sip: message uri format error")
			return
		}
		result = []string{nuResult[0], nuResult[1], "", nuResult[2], nuResult[3]}
	}
	u.Scheme = result[1]
	if len(result[2]) > 0 {
		parts := strings.Split(result[2], ":")
		u.Username = parts[0]
		if len(parts) > 1 {
			u.Password = parts[1]
		}
	}
	u.Domain = result[3]
	u.Arguments = parseArgs(result[4])
	return
}

// xxx:xxx@xxxx
//       xxxx 例如：sip:user:password@host:port;uri-parameters?headers
// 第一部分匹配协议，有至少一个字母组成，sip
// 第二部分匹配用户名和密码，除了@字符之外的至少一个任意字符
// 第三部分匹配域名，排除空格、制表符\t、换行符\r\n、换页符\v\f和;字符之外的至少一个任意字符，即遇到空格、制表符和;就停止
// 第四部分匹配餐护士，匹配排除\r\n之外的任意个字符，参数可能在多个空格或换行之后
var uriRegExpWithUser = regexp.MustCompile("^([A-Za-z]+):([^@]+)@([^\\s;]+)(.*)$")
var uriRegExpNoUser = regexp.MustCompile("^([A-Za-z]+):([^\\s;]+)(.*)$")
