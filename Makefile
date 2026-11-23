.PHONY: help up down build logs clean test

help:
	@echo "Available commands:"
	@echo "  make up        - Start all services"
	@echo "  make down      - Stop all services"
	@echo "  make build     - Build all services"
	@echo "  make logs      - Show logs"
	@echo "  make clean     - Remove all containers and volumes"
	@echo "  make test      - Run tests"

up:
	docker-compose up -d

down:
	docker-compose down

build:
	docker-compose build

logs:
	docker-compose logs -f

clean:
	docker-compose down -v

test:
	@echo "Running tests..."
	@cd auth-service && go test -v -coverprofile=coverage.out ./...
	@cd transaction-service && go test -v -coverprofile=coverage.out ./...
	@cd analytics-service && go test -v -coverprofile=coverage.out ./...
	@echo "✅ Tests completed"
