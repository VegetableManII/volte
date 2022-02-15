package controller

import (
	"context"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func TestCreateUser(t *testing.T) {
	db, err := gorm.Open("mysql", "root:@tcp(127.0.0.1:3306)/volte?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		t.Fatalf("%v", err)
	}
	u := User{
		IMSI:        "123456789",
		Mnc:         "01",
		Mcc:         86,
		Apn:         "hebeiyidong",
		IP:          "2.2.2.2",
		SipUserName: "jiqimao",
		SipDNS:      "3gpp.net",
		Ctime:       time.Now(),
		Utime:       time.Now(),
	}
	err = CreateUser(context.Background(), db, &u)
	if err != nil {
		t.Log(err)
	}
}
