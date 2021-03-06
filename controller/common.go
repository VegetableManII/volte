package controller

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/VegetableManII/volte/modules"
	"github.com/VegetableManII/volte/sip"
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

var defExpire time.Duration = 120 * time.Second
var UARegPrefix = "ua:"
var MARegPrefix = "ma:"
var AddrPrefix = "addr:"
var UeInfoPrefix = "uinfo:"

type Cache struct {
	*cache.Cache
}

func initCache() *Cache {
	return &Cache{
		cache.New(cache.NoExpiration, time.Second),
	}
}

// PGW 更新基站网络连接
func (p *Cache) updateAddress(key string, val *net.UDPAddr) error {
	_, ok := p.Get(key)
	if !ok { // 不存在该无线接入点的缓存
		err := p.Add(key, val, cache.NoExpiration)
		if err != nil {
			return err
		}
	}
	p.Set(key, val, cache.NoExpiration)
	return nil
}

// PGW 根据基站标识获取基站网络连接
func (p *Cache) getAddress(key string) *net.UDPAddr {
	ra, ok := p.Get(key)
	if !ok {
		return nil
	}
	return ra.(*net.UDPAddr)
}

// 查看缓存的用户请求是否存在
func (c *Cache) getUserRegistReq(key string) (*sip.Message, bool) {
	m, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	rc := m.(*RegistCombine)
	return rc.Req, true
}

// 缓存用户注册请求，默认2分钟后过期，等待HSS返回
func (c *Cache) setUserRegistReq(key string, msg *sip.Message) {
	rc := new(RegistCombine)
	rc.Req = msg
	rc.XRES = "NONE"
	c.Set(key, rc, defExpire)
}

// // ICSCF 查看缓存的用户请求是否存在
// func (i *Cache) getUserRegistReq(key string) (*sip.Message, bool) {
// 	m, ok := i.Get(key)
// 	if !ok {
// 		return nil, false
// 	}
// 	rc := m.(*RegistCombine)
// 	return rc.Req, true
// }

// // ICSCF 缓存用户注册请求，默认2分钟后过期，等待HSS返回
// func (i *Cache) setUserRegistReq(key string, msg *sip.Message) {
// 	rc := new(RegistCombine)
// 	rc.Req = msg
// 	rc.XRES = "NONE"
// 	i.Set(key, rc, defExpire)
// }

// SCSCF 添加用户注册请求对应鉴权向量
func (i *Cache) setUserRegistXRES(key string, val string) error {
	// 首先查看是否存在请求
	m, expire, ok := i.GetWithExpiration(key)
	if !ok {
		return errors.New("ErrNotFoundRequest")
	}
	rc := m.(*RegistCombine)
	rc.XRES = val
	remain := time.Until(expire)
	i.Set(key, rc, remain)
	return nil
}

// SCSCF 查看用户注册请求对应鉴权向量
func (s *Cache) getUserRegistXRES(key string) string {
	m, ok := s.Get(key)
	if !ok {
		return ""
	}
	rc := m.(*RegistCombine)
	if rc.XRES == "NONE" {
		return ""
	}
	return rc.XRES
}

// SCSCF 删除用户注册请求和鉴权向量
func (s *Cache) delUserRegistReqXRES(key string) {
	s.Delete(key)
}

// SCSCF 添加用户信息到系统
func (s *Cache) updateUserInfo(key string, val *User) {
	s.Set(key, val, cache.NoExpiration)
}

// ICSCF/SCSCF 查询用户信息
func (i *Cache) getUserInfo(key string) *User {
	m, ok := i.Get(key)
	if !ok {
		return nil
	}
	ue := m.(*User)
	return ue
}
