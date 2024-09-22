OUT=$(shell realpath -m bin)
GOPATH=$(shell go env GOPATH)
branch=$(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision=$(shell git rev-parse HEAD)
dirty=$(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
ldflags='-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

all: getdeps test

getdeps:
	@echo "Installing golangci-lint" && go get github.com/golangci/golangci-lint/cmd/golangci-lint && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go mod tidy

lint:
	@echo "Running $@"
	golangci-lint run -c ../../.golangci.yml

test: lint
	go test -v -vet=off ./...

benchmarks: 
	go test -v -vet=off ./... -bench=. -count 1 -benchtime=10s -benchmem -run=^#

coverage: clean 
	mkdir coverage
	go test -v -vet=off ./... -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

testrace: lint
	go test -v -race -vet=off ./...

run: 
	go run main.go

build:
	go build -o bin/zos-update-worker main.go 
	
clean:
	rm ./coverage -rf
	rm ./bin -rf
