.PHONY: build run test coverage clean

SERVICE_NAME=analytics-service
PKG=./...

build:
	@go build -o bin/analytics ./cmd/analytics

run: build
	@./bin/analytics

test:
	@go test -v -race -coverprofile=coverage.out $(PKG)

coverage:
	@go test -coverprofile=coverage.out $(PKG)
	@go tool cover -func=coverage.out

clean:
	@rm -rf bin coverage.out coverage.html
