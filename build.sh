#!/bin/sh

# 启动一套IMS网络需要指定该网络的名称，如hebeiyidong
if [ $# -lt 1 ]; then
echo "usage: build.sh domain"
exit
fi

domain=$1
echo "start $domain"

go build ./entity/hss
go build ./entity/i-cscf
go build ./entity/s-cscf
go build ./entity/p-cscf
go build ./entity/pgw


kill -9 $(ps aux|grep "./hss"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./s-cscf -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./i-cscf -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./p-cscf -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./pgw -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')

nohup ./hss &
nohup ./s-cscf -d $domain -f ./config.yml &
nohup ./i-cscf -d $domain -f ./config.yml &
nohup ./p-cscf -d $domain -f ./config.yml &
nohup ./pgw -d $domain -f ./config.yml &

