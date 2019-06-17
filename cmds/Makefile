MK_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
OUT = $(shell realpath -m $(MK_DIR)/../bin)
MODULES = $(shell find $(MK_DIR) -maxdepth 1 -type d -not -name .)

all: $(MODULES)

.PHONY: output clean

output:
	mkdir -p $(OUT)

%: %/go.mod output
	cd $(shell dirname $<) && go build -o $(OUT)/$(shell basename `dirname $<`)
