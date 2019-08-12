OUT = ../../bin
branch = $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision = $(shell git rev-parse HEAD)
dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
version = github.com/threefoldtech/zosv2/modules/version
ldflags = '-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

# _base is a hack to only build modules that we know are building atm
# _base: contd flistd storaged upgraded networkd identityd provisiond
_base: $(shell ls -d */)

all: _base
	strip $(OUT)/*

.PHONY: output clean

tidy:
	$(foreach var,$(shell ls -d */),pushd $(var);go mod tidy;popd;)
	git add */go.sum */go.mod

output:
	mkdir -p $(OUT)

%: %/go.mod output
	cd $(shell dirname $<) && go build -ldflags $(ldflags) -o $(OUT)/$(shell basename `dirname $<`)
