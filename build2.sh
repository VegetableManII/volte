go build ./entity/hss
go build ./entity/i-cscf
go build ./entity/s-cscf
go build ./entity/p-cscf
go build ./entity/pgw

kill -9 $(ps aux|grep "./hss -f ./config2.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./s-cscf -f ./config2.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./i-cscf -f ./config2.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./p-cscf -f ./config2.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./pgw -f ./config2.yml"|awk 'NR==1{ print $2 }')

nohup ./hss -f ./config2.yml &
nohup ./s-cscf -f ./config2.yml &
nohup ./i-cscf -f ./config2.yml &
nohup ./p-cscf -f ./config2.yml &
nohup ./pgw -f ./config2.yml &

