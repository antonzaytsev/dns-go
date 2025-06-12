.PHONY: build build-dev build-prod run test clean docker-build docker-run fmt vet lint deps help all

# Build variables
APP_NAME := dns-server
MAIN_PATH := ./cmd/dns-server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d " " -f 3)

# Linker flags to inject build metadata
LDFLAGS := -ldflags "\
    -X 'dns-go/pkg/version.Version=$(VERSION)' \
    -X 'dns-go/pkg/version.GitCommit=$(GIT_COMMIT)' \
    -X 'dns-go/pkg/version.BuildDate=$(BUILD_DATE)' \
    -w -s"

# Development build flags (with debug info)
LDFLAGS_DEV := -ldflags "\
    -X 'dns-go/pkg/version.Version=$(VERSION)' \
    -X 'dns-go/pkg/version.GitCommit=$(GIT_COMMIT)' \
    -X 'dns-go/pkg/version.BuildDate=$(BUILD_DATE)'"

# Build the application (production)
build-prod:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(APP_NAME) $(MAIN_PATH)

# Build the application (development)
build-dev:
	go build $(LDFLAGS_DEV) -o $(APP_NAME) $(MAIN_PATH)

# Default build target (development)
build: build-dev

# Run the application with default settings
run:
	./$(APP_NAME)

# Run tests with coverage
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run tests for specific package
test-pkg:
	go test -v -race ./internal/cache

# Run with development settings
run-dev:
	go run $(MAIN_PATH) -listen=127.0.0.1 -port=5053 -log=./logs/dns-requests.log -log-level=debug -cache-size=1000

# Benchmark tests
bench:
	go test -bench=. -benchmem ./...

# Format code
fmt:
	go fmt ./...
	goimports -w -local dns-go .

# Vet code
vet:
	go vet ./...

# Lint code (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Security scan (requires gosec)
security:
	@which gosec > /dev/null || (echo "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec ./...

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	rm -f coverage.out
	rm -rf logs/*
	go clean -cache
	go clean -testcache

# Docker build
docker-build:
	docker build -t dns-go .

# Docker run
docker-run:
	docker-compose up -d

# Docker stop
docker-stop:
	docker-compose down

# Docker build with multi-stage optimization
docker-build-prod:
	docker build --target production -t dns-go:latest .

# Install dependencies
deps:
	go mod tidy
	go mod download
	go mod verify

# Update dependencies
deps-update:
	go get -u ./...
	go mod tidy

# Generate code (if needed)
generate:
	go generate ./...

# Cross-compile for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(APP_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(APP_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(APP_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(APP_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(APP_NAME)-windows-amd64.exe $(MAIN_PATH)

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

# Show build info from binary
info:
	./$(APP_NAME) -version

# Create release directory
dist:
	mkdir -p dist

# Development setup
dev-setup: deps
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the DNS server binary (development)"
	@echo "  build-dev      - Build with debug info"
	@echo "  build-prod     - Build optimized production binary"
	@echo "  build-all      - Cross-compile for multiple platforms"
	@echo "  run            - Run the DNS server with default settings"
	@echo "  run-dev        - Run in development mode with debug logging"
	@echo "  test           - Run all tests with coverage"
	@echo "  test-pkg       - Run tests for specific package"
	@echo "  bench          - Run benchmark tests"
	@echo "  fmt            - Format Go code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  security       - Run security scan with gosec"
	@echo "  clean          - Clean build artifacts and caches"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run with Docker Compose"
	@echo "  docker-stop    - Stop Docker containers"
	@echo "  deps           - Install and tidy dependencies"
	@echo "  deps-update    - Update all dependencies"
	@echo "  dev-setup      - Install development tools"
	@echo "  version        - Show build version information"
	@echo "  info           - Show version from built binary"
	@echo "  help           - Show this help message"

# Default target
all: fmt vet build test 