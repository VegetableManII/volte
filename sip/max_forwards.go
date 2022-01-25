package sip

import (
	"fmt"
	"strconv"
)

// 默认最大跳数
var MaxForwardsCount int = 70

// 数据包的最大转发次数
type MaxForwards struct {
	value int // 内部的实际值
}

func parseMaxForwards(str string) (item MaxForwards, err error) {
	value, err := strconv.Atoi(str)
	if err != nil {
		return
	}
	item = MaxForwards{
		value: value,
	}
	return
}

// 重置Max-Forwards数量
func (mf *MaxForwards) Reset() {
	mf.value = MaxForwardsCount
}

// Max-Forwards数量
func (mf *MaxForwards) Reduce() {
	if mf.value == 0 {
		return
	}
	mf.value -= 1
}

// 字符串表示
func (mf MaxForwards) String() string {
	return fmt.Sprintf("%d", mf.value)
}
