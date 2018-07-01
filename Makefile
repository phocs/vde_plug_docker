GO_SRC := ./src
export GOPATH=$(shell pwd)

PLUGIN_NAME := vde_plug_docker

SERVICE_DIR := ./service
SERVICE := $(PLUGIN_NAME).service
SOCKET := $(PLUGIN_NAME).socket

DH_DOCKERFILE := ./docker/Dockerfile
DH_PLUGIN_DIR := ./dh_plugin
DH_PLUGIN_TAG ?= latest
DH_PLUGIN_NAME := $(DH_REPO)vde

SYSTEMD_DIR := /lib/systemd/system
LIB_DOCKER_DIR := /usr/lib/docker

all: build
plugin: build plugin-build

build:
	echo ${PATH}
	ln -sf vendor ${GO_SRC}
	go build -v . || true
	rm -rf ${GO_SRC}

install:
	cp ./${PLUGIN_NAME} ${LIB_DOCKER_DIR}/${PLUGIN_NAME}
	cp $(SERVICE_DIR)/${SOCKET} ${SYSTEMD_DIR}/
	cp $(SERVICE_DIR)/${SERVICE} ${SYSTEMD_DIR}/
	systemctl enable ${PLUGIN_NAME}
	systemctl start ${PLUGIN_NAME}

uninstall:
	systemctl stop ${SOCKET}
	systemctl stop ${SERVICE}
	systemctl disable ${SERVICE}
	rm ${SYSTEMD_DIR}/${SERVICE}
	rm ${SYSTEMD_DIR}/${SOCKET}
	rm ${LIB_DOCKER_DIR}/${PLUGIN_NAME}

plugin-build:
	docker build -f ${DH_DOCKERFILE} -t ${DH_PLUGIN_NAME}:rootfs .
	mkdir -p ${DH_PLUGIN_DIR}/rootfs
	docker create --name tmp ${DH_PLUGIN_NAME}:rootfs
	docker export tmp | tar -x -C ${DH_PLUGIN_DIR}/rootfs
	cp ./docker/config.json ${DH_PLUGIN_DIR}
	docker rm -vf tmp

plugin-install:
	docker plugin create ${DH_PLUGIN_NAME}:${DH_PLUGIN_TAG} ${DH_PLUGIN_DIR}
	docker plugin enable ${DH_PLUGIN_NAME}:${DH_PLUGIN_TAG}

plugin-uninstall:
	docker plugin disable -f ${DH_PLUGIN_NAME}:${DH_PLUGIN_TAG} || true
	docker plugin rm -f ${DH_PLUGIN_NAME}:${DH_PLUGIN_TAG}

clean:
	rm -rf ./vde_plug_docker ${DH_PLUGIN_DIR}/ ${GO_SRC}
	docker rmi -f ${DH_PLUGIN_NAME}:rootfs || true
