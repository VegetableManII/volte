package sip

// 请求的路由表
type Route struct {
	value []User // 实际的值
}

func parseRoute(str string, oldValue Route) (newValue Route, err error) {
	item, err := parseUser(str)
	if err != nil {
		return
	}
	newValue.value = append(oldValue.value, item)
	return
}

// 检查第一个域是否是自己
func (r Route) FirstIsCurrentDomain() bool {
	if len(r.value) == 0 {
		return false
	}
	return r.value[0].URI.Domain == ServerDomainHost()
}

// 获取第一条记录
func (r Route) FirstItem() (item User, isExist bool) {
	if len(r.value) == 0 {
		isExist = false
		return
	}
	item = r.value[0]
	return
}

// 删除第一条记录
func (r *Route) RemoveFirst() {
	if len(r.value) == 0 {
		return
	}
	r.value = r.value[1:]
}
