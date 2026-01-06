.PHONY: list run convert linter build deps test
NOW := $(shell date +"%Y%m%d%H%M%S")

list:
	@LC_ALL=C $(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/(^|\n)# Files(\n|$$)/,/(^|\n)# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'

run:
	go run -ldflags "-X main.version=$(NOW)" ./src/cmd

deps:
	go get -u -t ./...
	go mod tidy

build:
	mkdir -p artifacts/build
	go build -ldflags "-X main.version=$(NOW)" -o artifacts/build/netprobe ./src/cmd 

linter:
	gofmt -w ./src/

test:
	mkdir -p artifacts/coverage
	go test -cover -coverprofile=artifacts/coverage/coverage.out -coverpkg=./... -v ./src/...
	go tool cover -html=artifacts/coverage/coverage.out -o artifacts/coverage/coverage.html
