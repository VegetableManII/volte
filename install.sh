# !/bin/sh

yum update
yum install git
## 安装go
wget https://golang.org/dl/go1.17.2.linux-amd64.tar.gz
tar -C /usr/local -xzvf  ./go1.17.2.linux-amd64.tar.gz
mkdir -p /root/go/src
mkdir -p /root/go/bin
mkdir -p /root/go/mod
export PATH=$PATH:/usr/local/go/bin
export GOROOT=/usr/local/go
export GOPATH=/root/go

go version

## 拉取项目
git clone https://github.com/VegetableManII/volte.git

cd volte && git checkout 3-ims-logic-layer

sh ./run.sh