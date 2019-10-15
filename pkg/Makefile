PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
# LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)

# BUILD_LDFLAGS := '$(LDFLAGS)'

all: build

getdeps:
	@echo "Installing golint" && go get -u golang.org/x/lint/golint
	@echo "Installing gocyclo" && go get -u github.com/fzipp/gocyclo
	@echo "Installing deadcode" && go get -u github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get -u github.com/client9/misspell/cmd/misspell
	@echo "Installing ineffassign" && go get -u github.com/gordonklaus/ineffassign

verifiers: vet fmt lint cyclo spelling #deadcode

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
	@${GOPATH}/bin/misspell -error `find .`

# Builds minio, runs the verifiers then runs the tests.
check: test
test: verifiers build
	# we already ran vet separately, so safe to turn it off here
	@echo "Running unit tests with GOFLAGS=${GOFLAGS}"
	for pkg in $(shell go list ./... | grep -Ev "stubs|network" ); do \
		go test -v -vet=off $$pkg; \
	done

testrace: verifiers build
	@echo "Running unit tests with GOFLAGS=${GOFLAGS}"
	# we already ran vet separately, so safe to turn it off here
	@CGO_ENABLED=1 go test -v -vet=off -race ./...

coverage: verifiers build
	@(env bash $(PWD)/buildscripts/go-coverage.sh)

generate:
	@echo "Generating modules client stubs"
	go generate github.com/threefoldtech/zos/pkg

build:
	@CGO_ENABLED=0 go build -v ./...