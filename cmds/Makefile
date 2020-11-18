OUT = $(shell realpath -m ../bin)
branch = $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision = $(shell git rev-parse HEAD)
dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
version = github.com/threefoldtech/zos/pkg/version
ldflags = '-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty) -extldflags "-static"'

all: $(shell ls -d */)
	strip $(OUT)/*

.PHONY: output clean identityd identityd/

output:
	mkdir -p $(OUT)

identityd:
	cd identityd && GOOS=linux go build -ldflags $(ldflags) -o $(OUT)/identityd

# this one is expanded by the `all` rule.
identityd/:
	cd identityd && GOOS=linux go build -ldflags $(ldflags) -o $(OUT)/identityd

%:	%/*.go output
	cd $(shell dirname $<) && CGO_ENABLED=0 GOOS=linux go build -ldflags $(ldflags) -o $(OUT)/$(shell basename `dirname $<`)
