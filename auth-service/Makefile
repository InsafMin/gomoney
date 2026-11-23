.PHONY: build test coverage run clean

SERVICE_NAME=auth-service
PKG=./...

build:
	@go build -o bin/auth ./cmd/auth

run: build
	@./bin/auth

test:
	@go test -v -race $(PKG)

coverage:
	@go test -cover $(PKG) -coverprofile=coverage.out
	@go tool cover -func=coverage.out

clean:
	@rm -rf bin coverage.out
