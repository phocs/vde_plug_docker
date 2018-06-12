PLUGIN_TAG ?= latest
PLUGIN_NAME := $(REPO)vde
export GOPATH=$(shell pwd)

all: build
plugin: build plugin-build

build:
	ln -sf vendor src
	go build -v .
	rm -rf ./src

install:
	cp ./vde_plug_docker /usr/local/bin/
	cp ./vde_plug_docker.service /lib/systemd/system/
	systemctl enable vde_plug_docker
	systemctl start vde_plug_docker

uninstall:
	systemctl stop vde_plug_docker
	systemctl disable vde_plug_docker
	rm /lib/systemd/system/vde_plug_docker.service
	rm /usr/local/bin/vde_plug_docker

plugin-build:
	docker build -t ${PLUGIN_NAME}:rootfs .
	mkdir -p ./plugin/rootfs
	docker create --name tmp ${PLUGIN_NAME}:rootfs
	docker export tmp | tar -x -C ./plugin/rootfs
	cp ./config.json ./plugin/
	docker rm -vf tmp

plugin-install:
	docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin
	docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}

plugin-uninstall:
	docker plugin disable -f ${PLUGIN_NAME}:${PLUGIN_TAG} || true
	docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG}

clean:
	docker rmi -f ${PLUGIN_NAME}:rootfs || true
	rm -rf ./vde_plug_docker ./plugin
