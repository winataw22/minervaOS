PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)

all: build

getdeps:
	@echo "Installing golint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.0
	@echo "Installing gocyclo" && go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "Installing deadcode" && go install github.com/remyoudompheng/go-misc/deadcode@latest
	@echo "Installing misspell" && go install github.com/client9/misspell/cmd/misspell@latest
	@echo "Installing ineffassign" && go install github.com/gordonklaus/ineffassign@latest
	@echo "Installing staticcheck" && go install honnef.co/go/tools/cmd/staticcheck@latest

verifiers: vet fmt lint cyclo spelling static #deadcode

vet:
	@echo "Running $@"
	@go vet -atomic -bool -copylocks -nilfunc -printf -rangeloops -unreachable -unsafeptr -unusedresult ./...

fmt:
	@echo "Running $@"
	@gofmt -d .

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golangci-lint run

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
	@${GOPATH}/bin/staticcheck -- ./...

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
