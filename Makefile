.PHONY: all help dep test install hookInstall build dist bootstrap

# reference https://github.com/technosophos/helm-template

HELM_HOME ?= $(shell helm home)
HELM_PLUGIN_DIR ?= ${HELM_HOME}/plugins/kubeschema
HAS_DEP := $(shell command -v dep;)
VERSION := $(shell sed -n -e 's/version:[ "]*\([^"]*\).*/\1/p' plugin.yaml)
DIST := ./_dist
LDFLAGS := "-X main.version=${VERSION} -extldflags '-static'"

all: help

help:				## Show this help
	@scripts/help.sh

dep: 				## Get the dependencies
	@dep status

test:				## Test potential bugs and race conditions
	@scripts/test.sh

install: bootstrap build
	mkdir -p ${HELM_PLUGIN_DIR}
	cp kubeschema ${HELM_PLUGIN_DIR}
	cp plugin.yaml ${HELM_PLUGIN_DIR}

hookInstall: bootstrap build

build:
	go build -o kubeschema -ldflags $(LDFLAGS) ./main.go

dist:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kubeschema -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/kubeschema-linux-$(VERSION).tgz kubeschema README.md LICENSE plugin.yaml
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o kubeschema -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/kubeschema-macos-$(VERSION).tgz kubeschema README.md LICENSE plugin.yaml
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o kubeschema.exe -ldflags $(LDFLAGS) ./main.go
	tar -zcvf $(DIST)/kubeschema-windows-$(VERSION).tgz kubeschema.exe README.md LICENSE plugin.yaml

bootstrap:
ifndef HAS_DEP
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
endif
	dep ensure
