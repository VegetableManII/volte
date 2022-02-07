package controller

import (
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"log"
	"math/rand"
	"time"

	"github.com/VegetableManII/volte/common"
	"github.com/wmnsk/milenage"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"

	"github.com/wonderivan/logger"
)

type HssEntity struct {
	*Mux
	dbclient *gorm.DB
	Points   map[string]string
}

var defaultHash hash.Hash
var defaultAuth string = "offical@hebeiyidong.3gpp.net"

func (this *HssEntity) Init(dbhost string) {
	// 初始化路由
	this.Mux = new(Mux)
	this.router = make(map[[2]byte]BaseSignallingT)
	// 初始化支持的加密算法
	this.Points = make(map[string]string)
	defaultHash = md5.New()
	// 初始化数据库连接
	db, err := gorm.Open("mysql", dbhost)
	if err != nil {
		log.Panicln("HSS初始化数据库连接失败", err)
	}
	this.dbclient = db
}

// HSS可以接收epc电路协议也可以接收SIP协议
func (this *HssEntity) CoreProcessor(ctx context.Context, in, up, down chan *common.Package) {
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
			err = f(ctx, msg, up, down)
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
func (this *HssEntity) AuthenticationInformatRequestF(ctx context.Context, m *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From MME: %v", ctx.Value("Entity"), string(m.GetData()))
	data := m.GetData()
	hashtable := common.StrLineUnmarshal(data)
	imsi := hashtable["IMSI"]
	// TODO ue携带自身支持的加密算法方式
	// 查询数据库
	user, err := GetUserByIMSI(ctx, this.dbclient, imsi)
	if err != nil {
		return err
	}
	// 针对该用户生成随机数nonce
	rand.Seed(time.Now().UnixNano())
	nonce := rand.Int31()
	// 加密得到密文
	seed := fmt.Sprintf("%s %s %d %s %s %s %d", user.IMSI,
		user.Mnc, user.Mcc, user.Apn, user.SipUserName, user.SipDNS, nonce)
	defaultHash.Write([]byte(seed))
	xres := defaultHash.Sum(nil)
	auth := defaultAuth
	kasme := "md5"
	var response = map[string]string{
		"IMSI":         imsi,
		HSS_RESP_AUTH:  auth,
		HSS_RESP_RAND:  fmt.Sprintf("%d", nonce),
		HSS_RESP_KASME: kasme,
		HSS_RESP_XRES:  hex.EncodeToString(xres),
	}
	host := this.Points["MME"]
	common.PackageOut(common.EPCPROTOCAL, common.AuthenticationInformatResponse, response, host, down) // 下行
	return nil
}

// HSS 接收Update Location Request请求，将用户APN信息响应给MME用于和PGW建立承载
func (this *HssEntity) UpdateLocationRequestF(ctx context.Context, p *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From MME: %v", ctx.Value("Entity"), string(p.GetData()))
	data := p.GetData()
	table := common.StrLineUnmarshal(data)
	imsi := table["IMSI"]
	// 查询数据库
	user, err := GetUserByIMSI(ctx, this.dbclient, imsi)
	if err != nil {
		return err
	}
	var response = map[string]string{
		"IMSI": imsi,
		"APN":  user.Apn,
	}
	host := this.Points["MME"]
	common.PackageOut(common.EPCPROTOCAL, common.UpdateLocationACK, response, host, down) // 下行
	return nil
}

func (this *HssEntity) MultimediaAuthorizationRequestF(ctx context.Context, p *common.Package, up, down chan *common.Package) error {
	defer common.Recover(ctx)

	logger.Info("[%v] Receive From S-CSCF: %v", ctx.Value("Entity"), string(p.GetData()))
	table := common.StrLineUnmarshal(p.GetData())
	un := table["username"]
	user, err := GetUserBySipUserName(ctx, this.dbclient, un)
	if err != nil {
		return err
	}
	AUTN, XRES, CK, IK, err := generateAV(user.RootK, user.Opc)
	if err != nil {
		return err
	}
	var response = map[string]string{
		"UserName": un,
		"AUTN":     hex.EncodeToString(AUTN),
		"XRES":     hex.EncodeToString(XRES),
		"CK":       hex.EncodeToString(CK),
		"IK":       hex.EncodeToString(IK),
	}
	host := this.Points["SCSCF"]
	common.PackageOut(common.EPCPROTOCAL, common.UpdateLocationACK, response, host, down) // 下行
	return nil
}

func generateRandN(n int) []byte {
	r := make([]byte, 0, 16)
	for i := 0; i < n; i++ {
		rand.Seed(time.Now().UnixNano())
		i := rand.Intn(128)
		r = append(r, byte(i))
	}
	return r
}

func generateAV(K, Opc string) (AUTN, XRES, CK, IK []byte, err error) {
	// 生成固定SQN
	SQN := []byte{0x01}
	// 生成16字节随机数RAND
	ran := generateRandN(16)
	// 根据Milenage算法生成四元鉴权向量组
	kbs, err := hex.DecodeString(K)
	if err != nil {
		return
	}
	opcbs, err := hex.DecodeString(Opc)
	if err != nil {
		return
	}
	milenage := milenage.NewWithOPc(
		kbs,
		opcbs,
		ran,
		binary.BigEndian.Uint64(SQN),
		0x0000,
	)
	MAC, err := milenage.F1()
	if err != nil {
		return
	}
	XRES, CK, IK, AK, err := milenage.F2345()
	if err != nil {
		return
	}
	AUTN = xor(SQN, AK)
	AUTN = append(AUTN, []byte{0x00, 0x00}...)
	AUTN = append(AUTN, MAC...)
	return
}

func xor(a []byte, b []byte) []byte {
	l3 := 0
	l1 := len(a)
	l2 := len(b)
	if l1 > l2 {
		l3 = l1
		// b补全0
		sub := l1 - l2
		for ; sub > 0; sub-- {
			b = append([]byte{0x00}, b...)
		}
	} else {
		l3 = l2
		// a补全0
		sub := l1 - l2
		for ; sub > 0; sub-- {
			a = append([]byte{0x00}, a...)
		}
	}
	c := make([]byte, 0, l3)
	for i := 0; i < l3; i++ {
		c = append(c, a[i]^b[i])
	}
	return c
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
	RootK       string    `gorm:"column:root_k"`
	Opc         string    `gorm:"column:opc"`
	Mnc         string    `gorm:"column:mnc" json:"mnc"` // 移动网号
	Mcc         int32     `gorm:"column:mcc" json:"mcc"` // 国家码
	Apn         string    `gorm:"column:apn" json:"apn"`
	IP          string    `gorm:"column:ip" json:"ip"`
	SipUserName string    `gorm:"column:sip_username" json:"sip_username"`
	SipDNS      string    `gorm:"column:sip_dns" json:"sip_dns"`
	Ctime       time.Time `gorm:"column:ctime"`
	Utime       time.Time `gorm:"column:utime"`
}

func (User) TableName() string {
	return "users"
}

func GetUserByIMSI(ctx context.Context, db *gorm.DB, imsi string) (*User, error) {
	ret := new(User)
	err := db.Model(User{}).Where("imsi=?", imsi).Find(ret).Error
	if err != nil {
		logger.Error("[%v] HSS获取用户信息失败,IMSI=%v,ERR=%v", ctx.Value("Entity"), imsi, err)
		return nil, err
	}
	return ret, nil
}

func GetUserBySipUserName(ctx context.Context, db *gorm.DB, un string) (*User, error) {
	ret := new(User)
	err := db.Model(User{}).Where("username=?", un).Find(ret).Error
	if err != nil {
		logger.Error("[%v] HSS获取用户信息失败,Sip_User_Name=%v,ERR=%v", ctx.Value("Entity"), un, err)
		return nil, err
	}
	return ret, nil
}

func CreateUser(ctx context.Context, db *gorm.DB, user *User) error {
	err := db.Create(user).Error
	if err != nil {
		logger.Error("[%v] HSS创建用户信息失败,USER=%v,ERR=%v", ctx.Value("Entity"), *user, err)
		return err
	}
	return nil
}
