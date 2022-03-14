package controller

import (
	"context"
	"net"
	"sync"

	"github.com/VegetableManII/volte/common"
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
type BaseSignallingT func(context.Context, *common.Package, chan *common.Package, chan *common.Package) error

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
	CoreProcessor(context.Context, chan *common.Package, chan *common.Package, chan *common.Package)
}

// MME 和 PGW 用于缓存UE和其接入点的关系
var UeCache *cache.Cache

func init() {
	UeCache = cache.New(cache.NoExpiration, cache.NoExpiration)
}

func updateUtranAddress(ctx context.Context, ra *net.UDPAddr, enb string) error {
	_, ok := UeCache.Get(enb)
	if !ok { // 不存在该无线接入点的缓存
		val := new(sync.Map)
		err := UeCache.Add(enb, val, cache.NoExpiration)
		if err != nil {
			return err
		}
	}
	return nil
}

func AddCache(ctx context.Context, ra *net.UDPAddr, key string) error {

}
