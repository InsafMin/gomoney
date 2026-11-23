.PHONY: build test coverage run clean

SERVICE_NAME=transaction-service
PKG=./...

build:
	@go build -o bin/transaction ./cmd/transaction

run: build
	@./bin/transaction

test:
	@go test -v -race $(PKG)

coverage:
	@go test -cover $(PKG) -coverprofile=coverage.out
	@go tool cover -func=coverage.out

clean:
	@rm -rf bin coverage.out
