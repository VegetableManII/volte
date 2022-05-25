# !/bin/sh

yum update
yum install git
yum localinstall https://dev.mysql.com/get/mysql57-community-release-el7-9.noarch.rpm

## 安装mysql
yum localinstall https://dev.mysql.com/get/mysql57-community-release-el7-9.noarch.rpm
rpm --import https://repo.mysql.com/RPM-GPG-KEY-mysql-2022
yum install mysql-community-server

show variables like 'validate_%';
set global validate_password_policy=LOW;
set global validate_password_length=0;
validate_password_mixed_case_count=0;
validate_password_number_count=0;
validate_password_special_char_count=0;

set password for root@localhost='';

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

# 关闭防火墙
systemctl stop firewalld
systemctl disable firewalld

# 跨平台编译
CGO_ENABLED=0 
GOOS=windows/darwin
GOARCH=amd64 
go build main.go