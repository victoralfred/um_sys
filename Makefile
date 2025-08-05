.PHONY: help test test-unit test-integration test-e2e lint fmt build run migrate-up migrate-down docker-up docker-down coverage clean

# Default target
help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run unit tests"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-e2e          - Run end-to-end tests"
	@echo "  lint              - Run linters"
	@echo "  fmt               - Format code"
	@echo "  build             - Build the application"
	@echo "  run               - Run the application"
	@echo "  migrate-up        - Run database migrations"
	@echo "  migrate-down      - Rollback database migrations"
	@echo "  docker-up         - Start Docker services"
	@echo "  docker-down       - Stop Docker services"
	@echo "  coverage          - Generate test coverage report"
	@echo "  clean             - Clean build artifacts"

# Testing targets
test:
	@echo "Running all tests..."
	@go test -v -race -coverprofile=coverage.out ./...

test-unit:
	@echo "Running unit tests..."
	@go test -v -race -short ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -race -run Integration ./tests/integration/...

test-e2e:
	@echo "Running e2e tests..."
	@go test -v -race -run E2E ./tests/e2e/...

# Code quality targets
lint:
	@echo "Running linters..."
	@golangci-lint run --fix

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Build targets
build:
	@echo "Building application..."
	@go build -o bin/server cmd/server/main.go
	@go build -o bin/worker cmd/worker/main.go
	@go build -o bin/migration cmd/migration/main.go

# Run targets
run: build
	@echo "Running application..."
	@./bin/server

# Database targets
migrate-up:
	@echo "Running migrations..."
	@migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/user_management?sslmode=disable" up

migrate-down:
	@echo "Rolling back migrations..."
	@migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/user_management?sslmode=disable" down 1

# Docker targets
docker-up:
	@echo "Starting Docker services..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker services..."
	@docker-compose down

# Coverage targets
coverage: test
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean -cache