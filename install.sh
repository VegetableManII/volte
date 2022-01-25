# !/bin/sh

yum update
yum install git
## 安装go
wget https://golang.org/dl/go1.17.2.linux-amd64.tar.gz
tar -C /usr/local -xzvf  ./go1.17.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

go version

## 拉取项目
git clone https://github.com/VegetableManII/volte.git

cd volte && git checkout 3-ims-logic-layer

sh ./run.sh