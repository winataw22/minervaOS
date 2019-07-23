OUT = ../../bin

# _base is a hack to only build modules that we know are building atm
_base: contd flistd storaged upgraded networkd identityd provisiond

all: _base
	strip $(OUT)/*

.PHONY: output clean

output:
	mkdir -p $(OUT)

%: %/go.mod output
	cd $(shell dirname $<) && go build -o $(OUT)/$(shell basename `dirname $<`)
