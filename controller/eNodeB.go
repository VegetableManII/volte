package controller

import (
	"bytes"
	"encoding/binary"
	"hash/fnv"
	"net"
	"sync"

	"github.com/VegetableManII/volte/common"
)

var RandomAccess uint32 = 0xFFFFFFFF

type EnodebEntity struct {
	TAI     int
	user    map[uint32]struct{}
	userMu  sync.Mutex
	request map[uint32]uint32
	reqMu   sync.Mutex
}

func (e *EnodebEntity) Init() {
	e.user = make(map[uint32]struct{})
	e.request = make(map[uint32]uint32)
}

func (e *EnodebEntity) UeRandAccess(data []byte, raddr *net.UDPAddr) (bool, []byte) {
	rand := parseRandAccess(data[0:4])
	if rand == RandomAccess {
		sum := fnv.New32().Sum([]byte(raddr.String()))
		ueid := parseRandAccess(sum)
		e.userMu.Lock()
		e.user[uint32(ueid)] = struct{}{}
		e.userMu.Unlock()
		return true, sum
	}
	return false, nil
}

func (e *EnodebEntity) GenerateUpLinkData(data []byte, n int, mme, pgw string) ([]byte, string, error) {
	request_id := common.GenerateRequestID()
	ueid := parseRandAccess(data[0:4])
	e.reqMu.Lock()
	e.request[request_id] = ueid
	e.reqMu.Unlock()

	buf := new(bytes.Buffer)
	dst := ""
	msg := new(common.CommonMsg)
	if data[4] == common.EPCPROTOCAL { // EPC 消息
		msg.ConstructWithReqID(request_id, data[4], data[5], len(data[8:]), []byte(data[8:]))
		dst = mme
	} else { // IMS 消息
		dst = pgw
	}
	err := binary.Write(buf, binary.BigEndian, msg)
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), dst, nil
}

func parseRandAccess(data []byte) (num uint32) {
	var i uint8 = 0
	for offset, v := range data {
		i = uint8(v)
		num += uint32(i << uint8(offset))
	}
	return
}

func (e *EnodebEntity) ParseDownLinkData(data []byte) {
	reqid := parseRandAccess(data[4:8])
	e.reqMu.Lock()
	ueid := e.request[reqid]
	e.reqMu.Unlock()
	binary.BigEndian.PutUint32(data[0:4], ueid)
}
