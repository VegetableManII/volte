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

type Cache struct {
	*cache.Cache
}

func initCache() *Cache {
	return &Cache{
		cache.New(cache.NoExpiration, time.Second),
	}
}

// PGW 更新基站网络地址
func (p *Cache) updateAddress(ra *net.UDPAddr, enb string) error {
	_, ok := p.Get(enb)
	val := ra
	if !ok { // 不存在该无线接入点的缓存
		err := p.Add(enb, ra, cache.NoExpiration)
		if err != nil {
			return err
		}
	}
	p.Set(enb, val, cache.NoExpiration)
	return nil
}

// PGW 获取基站地址
func (p *Cache) getAddress(name string) *net.UDPAddr {
	ra, ok := p.Get(name)
	if !ok {
		return nil
	}
	return ra.(*net.UDPAddr)
}

// PGW 设置基站地址
func (p *Cache) setAddress(ip, ap string) {
	p.Add(ip, ap, cache.NoExpiration)
}

// ICSCF 查看用户请求是否存在
func (i *Cache) getUserRegistReq(key string) (*sip.Message, bool) {
	m, ok := i.Get(key)
	if !ok {
		return nil, false
	}
	rc := m.(*RegistCombine)
	return rc.Req, true
}

// ICSCF 添加用户注册请求，默认2分钟后过期
func (i *Cache) setUserRegistReq(key string, msg *sip.Message) {
	rc := new(RegistCombine)
	rc.Req = msg
	rc.XRES = "NONE"
	i.Set(key, msg, defExpire)
}

// ICSCF 添加用户注册请求对应鉴权向量
func (i *Cache) setUserRegistXRES(key string, val string) error {
	// 首先查看是否存在请求
	m, expire, ok := i.GetWithExpiration(key)
	if !ok {
		return errors.New("ErrNotFoundRequest")
	}
	rc := m.(*RegistCombine)
	rc.XRES = val
	remain := expire.Sub(time.Now())
	i.Set(key, rc, remain)
	return nil
}

// ICSCF 查看用户注册请求对应鉴权向量
func (i *Cache) getUserRegistXRES(key string) string {
	m, ok := i.Get(key)
	if !ok {
		return ""
	}
	rc := m.(*RegistCombine)
	if rc.XRES == "NONE" {
		return ""
	}
	return rc.XRES
}

// ICSCF 删除用户注册请求和鉴权向量
func (i *Cache) delUserRegistReqXRES(key string) {
	i.Delete(key)
}

// ICSCF 添加请求和对应分支的缓存
func (i *Cache) addRequestCache(key string, meth, bh string, req *sip.Message) error {
	rc := new(RequestCache)
	rc.Type = meth
	rc.Branch = bh
	rc.Req = req
	i.Set(key, rc, cache.NoExpiration)
	return nil
}

// ICSCF 根据分支获取对应请求
func (i *Cache) getRequestCache(key string) *RequestCache {
	m, ok := i.Get(key)
	if !ok {
		return nil
	}
	rc := m.(*RequestCache)
	return rc
}

// SCSCF 添加用户信息到系统
func (s *Cache) setUserInfo(key string, val *User) error {
	return s.Add(key, val, cache.NoExpiration)
}

// SCSCF 查询用户信息
func (s *Cache) getUserInfo(key string) *User {
	m, ok := s.Get(key)
	if !ok {
		return nil
	}
	ue := m.(*User)
	return ue
}
