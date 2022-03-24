go build ./entity/hss
go build ./entity/i-cscf
go build ./entity/s-cscf
go build ./entity/p-cscf
go build ./entity/pgw

kill -9 ./hss -f ./config.yml
kill -9 ./s-cscf -f ./config.yml
kill -9 ./i-cscf -f ./config.yml
kill -9 ./p-cscf -f ./config.yml
kill -9 ./pgw -f ./config.yml


nohup ./hss -f ./config.yml &
nohup ./s-cscf -f ./config.yml &
nohup ./i-cscf -f ./config.yml &
nohup ./p-cscf -f ./config.yml &
nohup ./pgw -f ./config.yml &

