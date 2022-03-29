package entity

import "github.com/spf13/viper"

/*
模拟DNS服务器提供域名解析服务
场景1：注册场景下p-cscf根据request-uri查询i-cscf的地址
场景2：呼叫场景下p-cscf根据route查询s-cscf的地址，跨域时s-cscf根据查询对应域的i-cscf的地址
*/

type Node struct {
	Domain string
	Host   string
}

var Nodes map[string]*Node

func Init() {
	Nodes = make(map[string]*Node)
	// 河北移动
	domain := viper.GetString("hebeiyidong.domain")
	pgw := viper.GetString("hebeiyidong.pgw.host")
	Nodes["pgw."+domain] = &Node{Domain: "pgw." + domain, Host: pgw}
	pcscf := viper.GetString("hebeiyidong.p-cscf.host")
	Nodes["p-cscf."+domain] = &Node{Domain: "p-cscf." + domain, Host: pcscf}
	icscf := viper.GetString("hebeiyidong.i-cscf.host")
	Nodes["i-cscf."+domain] = &Node{Domain: "i-cscf." + domain, Host: icscf}
	scscf := viper.GetString("hebeiyidong.s-cscf.host")
	Nodes["s-cscf."+domain] = &Node{Domain: "s-cscf." + domain, Host: scscf}
	// 重庆电信
	domain = viper.GetString("chongqingdianxin.domain")
	pgw = viper.GetString("chongqingdianxin.pgw.host")
	Nodes["pgw."+domain] = &Node{Domain: "pgw." + domain, Host: pgw}
	pcscf = viper.GetString("chongqingdianxin.p-cscf.host")
	Nodes["p-cscf."+domain] = &Node{Domain: "p-cscf." + domain, Host: pcscf}
	icscf = viper.GetString("chongqingdianxin.i-cscf.host")
	Nodes["i-cscf."+domain] = &Node{Domain: "i-cscf." + domain, Host: icscf}
	scscf = viper.GetString("chongqingdianxin.s-cscf.host")
	Nodes["s-cscf."+domain] = &Node{Domain: "s-cscf." + domain, Host: scscf}
}

func Query(domain string) string {
	if node, ok := Nodes[domain]; !ok {
		return ""
	} else {
		return node.Host
	}
}
