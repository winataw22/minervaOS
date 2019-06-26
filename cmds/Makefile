MK_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
OUT = $(shell realpath -m $(MK_DIR)/../bin)
MODULES = $(shell find $(MK_DIR) -maxdepth 1 -type d -not -name .)

all: $(MODULES)

# _base is a hack to only build modules that we know are building atm
_base: contd flistd storaged upgraded

.PHONY: output clean

output:
	mkdir -p $(OUT)

%: %/go.mod output
	cd $(shell dirname $<) && go build -o $(OUT)/$(shell basename `dirname $<`)
