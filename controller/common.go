package controller

import (
	"context"
	"net"

	"github.com/VegetableManII/volte/modules"
	"github.com/patrickmn/go-cache"
)

// 鉴权向量
const (
	AV_AUTN = "AUTN"
	AV_RAND = "RAND"
	AV_XRES = "XRES"
	AV_IK   = "IK"
	AV_CK   = "CK"
)

// 定义基础路由转发方法
type BaseSignallingT func(context.Context, *modules.Package, chan *modules.Package, chan *modules.Package) error

// 路由转发器
type Mux struct {
	router map[[2]byte]BaseSignallingT
}

// 路由注册
func (m *Mux) Regist(r [2]byte, f BaseSignallingT) {
	if m.router == nil {
		m.router = make(map[[2]byte]BaseSignallingT)
	}
	m.router[r] = f
}

// VoLTE网络中各个功能实体的逻辑处理器实体抽象基类对象
type Base interface {
	CoreProcessor(context.Context, chan *modules.Package, chan *modules.Package, chan *modules.Package)
}

// MME 和 PGW 用于缓存UE和其接入点的关系
var Cache *cache.Cache

func init() {
	Cache = cache.New(cache.NoExpiration, cache.NoExpiration)
}

func updateAddress(ra *net.UDPAddr, enb string) error {
	_, ok := Cache.Get(enb)
	if !ok { // 不存在该无线接入点的缓存
		val := ra
		err := Cache.Add(enb, val, cache.NoExpiration)
		if err != nil {
			return err
		}
	}
	return nil
}

func getAP(p *modules.Package) (*net.UDPConn, *net.UDPAddr) {
	if modules.ConnectionExist(p) {
		return p.Conn, p.RemoteAddr
	}
	return nil, nil
}

func bindUeWithAP(ip, ap string) {
	Cache.Set(ip, ap, cache.NoExpiration)
}
