#!/bin/sh

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


kill -9 $(ps aux|grep -v "./hss"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep -v "./s-cscf -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep -v "./i-cscf -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep -v "./p-cscf -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep -v "./pgw -d $domain -f ./config.yml"|grep -v grep|awk 'NR==1{ print $2 }')

nohup ./hss &
nohup ./s-cscf -d $domain -f ./config.yml &
nohup ./i-cscf -d $domain -f ./config.yml &
nohup ./p-cscf -d $domain -f ./config.yml &
nohup ./pgw -d $domain -f ./config.yml &

