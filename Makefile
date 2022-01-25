path=$(shell pwd)
product=releases

default:all

all:pack

pack:compile
	mkdir -p $(product)/entity/conf
	mkdir -p $(product)/entity/bin
	cp entity/enodeb/enodeb $(product)/entity/bin
	cp entity/hss/hss $(product)/entity/bin
	cp entity/mme/mme $(product)/entity/bin
	cp entity/pgw/pgw $(product)/entity/bin
	cp entity/xcscf/xcscf $(product)/entity/bin
	cp config.yml $(product)/entity/conf

compile:show_env
	cd entity/enodeb; $(GOROOT)/bin/go build
	cd entity/hss; $(GOROOT)/bin/go build
	cd entity/mme; $(GOROOT)/bin/go build
	cd entity/pgw; $(GOROOT)/bin/go build
	cd entity/xcscf; $(GOROOT)/bin/go build

show_env:
	echo $(GOPATH)
	echo $(GOROOT)
