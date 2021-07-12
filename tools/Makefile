branch = $(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision = $(shell git rev-parse HEAD)
dirty = $(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
version = github.com/threefoldtech/zos/pkg/version
ldflags = '-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

_base: $(shell ls -d */)

all: _base explorer
	strip $(OUT)/*

.PHONY: output clean

getdeps:
	@echo "Installing golint" && go install golang.org/x/lint/golint@latest
	@echo "Installing gocyclo" && go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "Installing deadcode" && go install github.com/remyoudompheng/go-misc/deadcode@latest
	@echo "Installing misspell" && go install github.com/client9/misspell/cmd/misspell@latest
	@echo "Installing ineffassign" && go install github.com/gordonklaus/ineffassign@latest
	@echo "Installing staticcheck" && go install honnef.co/go/tools/cmd/staticcheck@latest

verifiers: vet fmt lint cyclo spelling static

vet:
	@echo "Running $@"
	@go vet -atomic -bool -copylocks -nilfunc -printf -rangeloops -unreachable -unsafeptr -unusedresult ./...

fmt:
	@echo "Running $@"
	@gofmt -d $(shell ls **/*.go | grep -v statik)

lint:
	@echo "Running $@"
	golint -set_exit_status $(shell go list ./... | grep -v stubs | grep -v generated| grep -v migrations | grep -v statik)

ineffassign:
	@echo "Running $@"
	ineffassign .

cyclo:
	@echo "Running $@"
	gocyclo -over 100 .


spelling:
	misspell -i monitord -error $(shell ls **/*.go | grep -v statik)

static:
	staticcheck -- ./...

# Builds minio, runs the verifiers then runs the tests.
check: test
test: verifiers
	# we already ran vet separately, so safe to turn it off here
	@echo "Running unit tests"
	for pkg in $(shell go list ./... ); do \
		go test -v -vet=off $$pkg; \
	done

testrace: verifiers
	@echo "Running unit tests with -race flag"
	# we already ran vet separately, so safe to turn it off here
	@CGO_ENABLED=1 go test -v -vet=off -race ./...


%: %/*.go output
	cd $(shell dirname $<) && go build -ldflags $(ldflags)

explorer/: explorer/*.go output
	cd $(shell dirname $<)/frontend && yarn install
	cd $(shell dirname $<)/frontend && NODE_ENV=production yarn build
	cd $(shell dirname $<) && go generate
	cd $(shell dirname $<) && go build -ldflags $(ldflags)
