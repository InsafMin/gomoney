.PHONY: build test run clean

SERVICE_NAME=api-gateway
PKG=./...

build:
	@go build -o bin/gateway ./cmd/gateway

run: build
	@./bin/gateway

test:
	@go test -v -race $(PKG)

clean:
	@rm -rf bin
