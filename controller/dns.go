/*
模拟DNS服务器提供域名解析服务
场景1：注册场景下p-cscf根据request-uri查询i-cscf的地址
场景2：呼叫场景下p-cscf根据route查询s-cscf的地址，跨域时s-cscf根据查询对应域的i-cscf的地址

*/
package controller
