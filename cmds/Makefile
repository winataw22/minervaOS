OUT = $(shell realpath -m ../bin)
branch = $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision = $(shell git rev-parse HEAD)
dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
version = github.com/threefoldtech/zos/pkg/version
ldflags = '-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty) -extldflags "-static"'

all: identityd internet zos
	strip $(OUT)/*

.PHONY: output clean identityd internet zos

output:
	mkdir -p $(OUT)

identityd:
	cd identityd && GOOS=linux go build -ldflags $(ldflags) -o $(OUT)/identityd

internet: output
	cd internet && CGO_ENABLED=0 GOOS=linux go build -ldflags $(ldflags) -o $(OUT)/internet

zos: output
	cd zos && CGO_ENABLED=0 GOOS=linux go build -ldflags $(ldflags) -o $(OUT)/zos
