package controller

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/fnv"
	"net"
	"sync"

	"github.com/VegetableManII/volte/common"
	"github.com/wonderivan/logger"
)

var RandomAccess uint32 = 0x0F0F0F0F

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

func (e *EnodebEntity) UeRandomAccess(data []byte, raddr *net.UDPAddr) (bool, []byte) {
	rand := parseRandAccess(data[0:4])
	if rand == RandomAccess {
		logger.Info("ue 随机接入 %x %x", rand, RandomAccess)
		sum := fnv.New32().Sum([]byte(raddr.String()))

		ueid := parseRandAccess(sum[len(sum)-4:])

		logger.Info("ueid(hex):%x ueid:%v", sum[len(sum)-4:], ueid)

		e.userMu.Lock()
		e.user[uint32(ueid)] = struct{}{}
		e.userMu.Unlock()
		return true, sum[len(sum)-4:]
	}
	return false, nil
}

func (e *EnodebEntity) GenerateUpLinkData(data []byte, n int, mme, pgw string) ([]byte, string, error) {
	request_id := common.GenerateRequestID()
	ueid := parseRandAccess(data[0:4])
	e.userMu.Lock()
	if _, ok := e.user[ueid]; !ok {
		return nil, "", errors.New("ErrNeedAccessInfo")
	}
	e.userMu.Unlock()
	e.reqMu.Lock()
	e.request[request_id] = ueid
	e.reqMu.Unlock()

	buf := new(bytes.Buffer)
	dst := ""
	msg := new(common.CommonMsg)
	if data[4] == common.EPCPROTOCAL { // EPC 消息
		msg.ConstructWithReqID(request_id, data[4], data[5], len(data[8:]), []byte(data[8:]))
		dst = mme
		err := binary.Write(buf, binary.BigEndian, msg)
		if err != nil {
			return nil, "", err
		}
		return buf.Bytes(), dst, nil
	} else { // IMS 消息
		dst = pgw
		return data[4:], dst, nil
	}

}

func parseRandAccess(data []byte) uint32 {
	return binary.BigEndian.Uint32(data[0:4])
}

func (e *EnodebEntity) ParseDownLinkData(data []byte) {
	reqid := parseRandAccess(data[4:8])
	e.reqMu.Lock()
	ueid := e.request[reqid]
	e.reqMu.Unlock()
	binary.BigEndian.PutUint32(data[0:4], ueid)
}
