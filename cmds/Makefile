OUT = $(shell realpath -m ../bin)
branch = $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision = $(shell git rev-parse HEAD)
dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
version = github.com/threefoldtech/zos/pkg/version
ldflags = '-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

all: $(shell ls -d */)
	strip $(OUT)/*

.PHONY: output clean

output:
	mkdir -p $(OUT)

%:	%/*.go output
	cd $(shell dirname $<) && go build -ldflags $(ldflags) -o $(OUT)/$(shell basename `dirname $<`)
