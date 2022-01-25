# !/bin/sh

nohup ./releases/bin/hss -f ./releases/conf/config.yml &
nohup ./releases/bin/xcscf -f ./releases/conf/config.yml &
nohup ./releases/bin/mme -f ./releases/conf/config.yml &
nohup ./releases/bin/pgw -f ./releases/conf/config.yml &
nohup ./releases/bin/enb -f ./releases/conf/config.yml &