package controller

import (
	"regexp"
)

var ipaddrRegxp = regexp.MustCompile(`((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})(\.((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})){3}`)

type EnodebEntity struct {
	TAI string // AP接入点标识
}
