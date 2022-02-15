package sip

// 后续消息流处理的服务器列表
type RecordRoute struct {
	value []User // 实际的值
}

func parseRecordRoute(str string, oldValue RecordRoute) (newValue RecordRoute, err error) {
	item, err := parseUser(str)
	if err != nil {
		return
	}
	newValue.value = append(oldValue.value, item)
	return
}

// 将本机信息加入RecordRoute节点列表中
func (rr *RecordRoute) AddServerInfo() {
	item := User{
		DisplayName: "",
		URI: URI{
			Scheme:   SchemeSip,
			Username: "",
			Domain:   ServerDomainHost(),
			Arguments: NewArgs(map[string]string{
				"lr": "",
			}),
		},
		Arguments: Args{},
	}
	rr.value = append([]User{item}, rr.value...)
}
