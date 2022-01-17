package controller

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"math/rand"
	"time"

	"github.com/VegetableManII/volte/common"

	"github.com/wonderivan/logger"
)

type HssEntity struct {
	*Mux
	csupport map[string]hash.Hash
	auth     string
}

var defaultHash hash.Hash
var defaultAuth string = "offical@hebeiyidong.3gpp.net"

func (this *HssEntity) Init() {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	// 初始化支持的加密算法
	this.csupport = make(map[string]hash.Hash)
	defaultHash = md5.New()
}

// HSS可以接收eps电路协议也可以接收SIP协议
func (this *HssEntity) CoreProcessor(ctx context.Context, in, out chan *common.Msg) {
	var err error
	var f BaseSignallingT
	var ok bool
	for {
		select {
		case msg := <-in:
			f, ok = this.router[msg.GetUniqueMethod()]
			if !ok {
				logger.Error("[%v] HSS不支持的消息类型数据 %v", ctx.Value("Entity"), msg)
				continue
			}
			err = f(ctx, msg, out)
			if err != nil {
				logger.Error("[%v] HSS消息处理失败 %v %v", ctx.Value("Entity"), msg, err)
			}
		case <-ctx.Done():
			// 释放资源
			logger.Warn("[%v] HSS逻辑核心退出", ctx.Value("Entity"))
			return
		}
	}
}

// HSS 接收Authentication Informat Request请求，然后查询数据库获得用户信息，生成nonce，选择加密算法，
func (this *HssEntity) AuthenticationInformatRequestF(ctx context.Context, m *common.Msg, out chan *common.Msg) error {
	logger.Info("[%v] Receive From MME: %v", ctx.Value("Entity"), string(m.Data1.GetData()))

	imsi, err := common.GetIMSI(m.Data1.GetData())
	if err != nil {
		return err
	}
	// TODO ue携带自身支持的加密算法方式
	// TODO 查询数据库
	// 针对该用户生成随机数nonce
	rand.Seed(time.Now().UnixNano())
	nonce := rand.Int31()
	// 加密得到密文
	data := fmt.Sprintf("%s %s %s %s %s %s %d", imsi,
		"1", "550", "hebeiyidong", "jiqimao", "3gpp.net", nonce)
	defaultHash.Write([]byte(data))
	xres := defaultHash.Sum(nil)
	auth := defaultAuth
	kasme := "md5"
	var response = map[string]string{
		"imsi":         imsi,
		HSS_RESP_AUTH:  auth,
		HSS_RESP_RAND:  fmt.Sprintf("%d", nonce),
		HSS_RESP_KASME: kasme,
		HSS_RESP_XRES:  hex.EncodeToString(xres),
	}
	common.WrapOutEPS(common.EPSPROTOCAL, common.AuthenticationInformatResponse, response, false, out) // 下行
	return nil
}

/*
 *	HSS数据层实现
 *
 *
 *
 */
type User struct {
	ID          int64     `gorm:"column:id"`
	IMSI        string    `gorm:"column:imsi" json:"imsi"`
	Mnc         int32     `gorm:"column:mnc" json:"mnc"` // 移动网号
	Mcc         int32     `gorm:"column:mcc" json:"mcc"` // 国家码
	Apn         string    `gorm:"column:apn" json:"apn"`
	IP          string    `gorm:"column:ip" json:"ip"`
	SipUserName string    `gorm:"column:sip_username" json:"sip_username"`
	SipDNS      string    `gorm:"column:sip_dns" json:"sip_dns"`
	Nonce       int32     `json:"nonce"`
	Ctime       time.Time `gorm:"column:ctime"`
	Utime       time.Time `gorm:"column:utime"`
}

func (User) TableName() string {
	return "users"
}
