package controller

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"net"
	"regexp"
	"sync"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"
)

const RandomAccess uint32 = 0x0F0F0F0F
const InviteAction uint32 = 0x00000000

var ipaddrRegxp = regexp.MustCompile(`((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})(\.((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})){3}`)

type Ue struct {
	Vip  string
	UeID uint32
}

type EnodebEntity struct {
	TAI    string // AP接入点标识
	user   map[uint32]struct{}
	userMu sync.Mutex
	ip     map[string]uint32
	ipMu   sync.Mutex
}

func (e *EnodebEntity) Init() {
	e.user = make(map[uint32]struct{})
}

func (e *EnodebEntity) UeRandomAccess(data []byte, raddr *net.UDPAddr) (bool, []byte) {
	rand := getUEID(data[0:4])
	if rand == RandomAccess {
		h := fnv.New32()
		_, _ = h.Write([]byte(raddr.String()))
		sum := h.Sum(nil)
		ueid := getUEID(sum)
		e.userMu.Lock()
		e.user[uint32(ueid)] = struct{}{}
		e.userMu.Unlock()
		return true, sum
	}
	return false, nil
}

func (e *EnodebEntity) VIP(data []byte) []byte {
	ueid := getUEID(data[0:4])
	// AttachAccept响应则记录UEIP
	switch data[0] {
	case modules.EPCPROTOCAL:
		if data[1] == modules.AttachAccept {
			msg := modules.StrLineUnmarshal(data[4:])
			ip := msg["IP"]
			e.ipMu.Lock()
			e.ip[ip] = ueid
			e.ipMu.Unlock()
		}
	case modules.SIPPROTOCAL:
		// INVITE 请求
		// 网络侧负责将Contact头解析成用户的IP，在eNodeB上只能解析IP格式无法解析域名格式
		if data[1] == modules.SipRequest {
			msg, err := sip.NewMessage(bytes.NewReader(data[4:]))
			if err != nil {
				host := msg.RequestLine.RequestURI.Domain
				callingIP := ipaddrRegxp.FindString(host)
				if callingIP != "" {
					return nil
				}
				e.ipMu.Lock()
				ueid = e.ip[callingIP]
				e.ipMu.Unlock()
				// 使用被叫ID替换消息包头
				binary.BigEndian.PutUint32(data[0:4], ueid)
			}
		}
	}
	return data
}

func (e *EnodebEntity) Attached(addr *net.UDPAddr) (uint32, bool) {
	h := fnv.New32()
	_, _ = h.Write([]byte(addr.String()))
	sum := h.Sum(nil)
	ueid := getUEID(sum)
	e.userMu.Lock()
	if _, ok := e.user[ueid]; !ok {
		return 0, false
	}
	e.userMu.Unlock()
	return ueid, true

}

func getUEID(data []byte) uint32 {
	return binary.BigEndian.Uint32(data[0:4])
}
