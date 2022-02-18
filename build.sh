go build ./entity/hss
go build ./entity/i-cscf
go build ./entity/mme
go build ./entity/p-cscf
go build ./entity/pgw
go build ./entity/s-cscf


nohup ./hss -f ./config.yml &
nohup ./mme -f ./config.yml &
nohup ./s-cscf -f ./config.yml &
nohup ./i-cscf -f ./config.yml &
nohup ./p-cscf -f ./config.yml &
nohup ./pgw -f ./config.yml &

