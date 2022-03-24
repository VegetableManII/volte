go build ./entity/hss
go build ./entity/i-cscf
go build ./entity/s-cscf
go build ./entity/p-cscf
go build ./entity/pgw

kill -9 $(ps aux|grep "./s-cscf -f ./config.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./s-cscf -f ./config.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./i-cscf -f ./config.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./p-cscf -f ./config.yml"|awk 'NR==1{ print $2 }')
kill -9 $(ps aux|grep "./pgw -f ./config.yml"|awk 'NR==1{ print $2 }')


nohup ./hss -f ./config.yml &
nohup ./s-cscf -f ./config.yml &
nohup ./i-cscf -f ./config.yml &
nohup ./p-cscf -f ./config.yml &
nohup ./pgw -f ./config.yml &

