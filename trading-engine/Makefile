# Trading Engine Makefile

.PHONY: help test test-unit test-integration test-e2e test-all coverage benchmark lint vet security build run clean deps generate

# Default target
help:
	@echo "Available targets:"
	@echo "  deps          Install dependencies"
	@echo "  generate      Generate mocks and protobuf files"
	@echo "  test          Run all tests"
	@echo "  test-unit     Run unit tests only"
	@echo "  test-integration Run integration tests only"
	@echo "  test-e2e      Run end-to-end tests"
	@echo "  coverage      Generate test coverage report"
	@echo "  benchmark     Run performance benchmarks"
	@echo "  lint          Run linting"
	@echo "  vet           Run go vet"
	@echo "  security      Run security checks"
	@echo "  build         Build all binaries"
	@echo "  run           Run the trading engine"
	@echo "  clean         Clean build artifacts"

# Dependencies
deps:
	go mod download
	go mod tidy
	go install github.com/golang/mock/mockgen@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Code generation
generate:
	go generate ./...
	@echo "Code generation complete"

# Testing
test: test-unit test-integration

test-unit:
	go test -v -race -timeout=5m ./internal/... ./pkg/...

test-integration:
	go test -v -race -timeout=10m -tags=integration ./tests/integration/...

test-e2e:
	go test -v -race -timeout=30m -tags=e2e ./tests/e2e/...

test-all: test-unit test-integration test-e2e

coverage:
	go test -v -race -coverprofile=coverage.tmp ./internal/... ./pkg/...
	go tool cover -html=coverage.tmp -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark:
	go test -v -bench=. -benchmem ./internal/... ./pkg/...

# Code quality
lint:
	golangci-lint run ./...

vet:
	go vet ./...

security:
	gosec ./...

# Build
build:
	go build -o bin/trader ./cmd/trader/
	go build -o bin/backtest ./cmd/backtest/
	go build -o bin/cli ./cmd/cli/

run:
	go run ./cmd/trader/

# Cleanup
clean:
	rm -rf bin/
	rm -rf build/
	rm -rf coverage.tmp coverage.html
	go clean -cache

# Development workflow
dev-setup: deps generate
	@echo "Development environment setup complete"

# CI targets
ci: lint vet security test-all
	@echo "CI checks complete"