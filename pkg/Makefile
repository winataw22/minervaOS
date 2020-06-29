PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)

all: build

getdeps:
	@echo "Installing golint" && go get -u golang.org/x/lint/golint
	@echo "Installing gocyclo" && go get -u github.com/fzipp/gocyclo
	@echo "Installing deadcode" && go get -u github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get -u github.com/client9/misspell/cmd/misspell
	@echo "Installing ineffassign" && go get -u github.com/gordonklaus/ineffassign

verifiers: vet fmt lint cyclo spelling static #deadcode

vet:
	@echo "Running $@"
	@go vet -atomic -bool -copylocks -nilfunc -printf -rangeloops -unreachable -unsafeptr -unusedresult ./...

fmt:
	@echo "Running $@"
	@gofmt -d .

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golint -set_exit_status $(shell go list ./... | grep -v stubs)

ineffassign:
	@echo "Running $@"
	@${GOPATH}/bin/ineffassign .

cyclo:
	@echo "Running $@"
	@${GOPATH}/bin/gocyclo -over 100 .

deadcode:
	@echo "Running $@"
	@${GOPATH}/bin/deadcode -test $(shell go list ./...) || true

spelling:
	@${GOPATH}/bin/misspell -i monitord -error `find .`

static:
	go run honnef.co/go/tools/cmd/staticcheck -- ./...

check: test
test: verifiers build
	go test -vet=off -v $(shell go list ./... | grep -Ev "stubs|network" )

testrace: verifiers build
	go test -vet=off -v $(shell go list ./... | grep -Ev "stubs|network" )

generate:
	@echo "Generating modules client stubs"
	go generate github.com/threefoldtech/zos/pkg

build:
	@CGO_ENABLED=0 go build -v ./...
