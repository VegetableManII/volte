# !/bin/sh

yum update
yum install git
yum localinstall https://dev.mysql.com/get/mysql57-community-release-el7-9.noarch.rpm
## 安装go
wget https://golang.org/dl/go1.17.2.linux-amd64.tar.gz
tar -C /usr/local -xzvf  ./go1.17.2.linux-amd64.tar.gz
mkdir -p /root/go/src & mkdir -p /root/go/bin & mkdir -p /root/go/mod
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile & echo 'export GOROOT=/usr/local/go' >> /etc/profile & echo 'export GOPATH=/root/go' >> /etc/profile
source /etc/profile
go version

## 拉取项目
git clone https://github.com/VegetableManII/volte.git

cd volte

go build ./entity/hss &
go build ./entity/i-cscf &
go build ./entity/mme &
go build ./entity/p-cscf &
go build ./entity/pgw &
go build ./entity/s-cscf

# 关闭防火墙
systemctl stop firewalld
systemctl disable firewalld

nohup ./hss -f ./config.yml &
nohup ./mme -f ./config.yml &
nohup ./s-cscf -f ./config.yml &
nohup ./i-cscf -f ./config.yml &
nohup ./p-cscf -f ./config.yml &
nohup ./pgw -f ./config.yml &

CGO_ENABLED=0 
GOOS=windows/darwin
GOARCH=amd64 
go build main.go