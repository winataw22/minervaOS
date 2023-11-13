OUT=$(shell realpath -m bin)
GOPATH=$(shell go env GOPATH)
branch=$(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision=$(shell git rev-parse HEAD)
dirty=$(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
ldflags='-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

all: getdeps test

getdeps:
	@echo "Installing staticcheck" && go get -u honnef.co/go/tools/cmd/staticcheck && go install honnef.co/go/tools/cmd/staticcheck
	@echo "Installing gocyclo" && go get -u github.com/fzipp/gocyclo/cmd/gocyclo && go install github.com/fzipp/gocyclo/cmd/gocyclo
	@echo "Installing deadcode" && go get -u github.com/remyoudompheng/go-misc/deadcode && go install github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get -u github.com/client9/misspell/cmd/misspell && go install github.com/client9/misspell/cmd/misspell
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
	go mod tidy


verifiers: fmt lint cyclo deadcode spelling staticcheck

fmt:
	@echo "Running $@"
	@gofmt -d .

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golangci-lint run

cyclo:
	@echo "Running $@"
	@${GOPATH}/bin/gocyclo -over 100 .

deadcode:
	@echo "Running $@"
	@${GOPATH}/bin/deadcode -test $(shell go list ./...) || true

spelling:
	@echo "Running $@"
	@${GOPATH}/bin/misspell -i monitord -error `find .`

staticcheck:
	@echo "Running $@"
	@${GOPATH}/bin/staticcheck -- ./...

test: verifiers
	go test -v -vet=off ./...

benchmarks: 
	go test -v -vet=off ./... -bench=. -count 1 -benchtime=10s -benchmem -run=^#

coverage: clean 
	mkdir coverage
	go test -v -vet=off ./... -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

testrace: verifiers
	go test -v -race -vet=off ./...

run: 
	go run main.go

build:
	go build -o bin/zos-update-worker main.go 
	
clean:
	rm ./coverage -rf
	rm ./bin -rf
