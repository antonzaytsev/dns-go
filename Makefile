.PHONY: build build-dev build-prod build-web run test clean docker-build docker-run fmt vet lint deps help all

# Build variables
DNS_APP_NAME := dns-server
WEB_APP_NAME := web-dashboard
API_APP_NAME := api-server
DNS_MAIN_PATH := ./cmd/dns-server
WEB_MAIN_PATH := ./cmd/web-dashboard
API_MAIN_PATH := ./cmd/api-server
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

# Build DNS server (production)
build-dns-prod:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(DNS_APP_NAME) $(DNS_MAIN_PATH)

# Build web dashboard (production)
build-web-prod:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(WEB_APP_NAME) $(WEB_MAIN_PATH)

# Build API server (production)
build-api-prod:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(API_APP_NAME) $(API_MAIN_PATH)

# Build DNS server (development)
build-dns-dev:
	go build $(LDFLAGS_DEV) -o $(DNS_APP_NAME) $(DNS_MAIN_PATH)

# Build web dashboard (development)
build-web-dev:
	go build $(LDFLAGS_DEV) -o $(WEB_APP_NAME) $(WEB_MAIN_PATH)

# Build API server (development)
build-api-dev:
	go build $(LDFLAGS_DEV) -o $(API_APP_NAME) $(API_MAIN_PATH)

# Build all applications (production)
build-prod: build-dns-prod build-web-prod build-api-prod

# Build all applications (development)
build-dev: build-dns-dev build-web-dev build-api-dev

# Default build target (development)
build: build-dev

# Run the DNS server with default settings
run:
	./$(DNS_APP_NAME)

# Run the web dashboard with default settings
run-web:
	./$(WEB_APP_NAME)

# Run the API server with default settings
run-api:
	./$(API_APP_NAME)

# Run tests with coverage
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run tests for specific package
test-pkg:
	go test -v -race ./internal/cache

# Run DNS server with development settings
run-dns-dev:
	go run $(DNS_MAIN_PATH) -listen=127.0.0.1 -port=5053 -log=./logs/dns-requests.log -log-level=debug -cache-size=1000

# Run API server with development settings
run-api-dev:
	go run $(API_MAIN_PATH) -port=8080 -log-file=./logs/dns-requests.log

# Run web dashboard with development settings
run-web-dev:
	go run $(WEB_MAIN_PATH) -port=8080 -log-file=./logs/dns-requests.log

# Build and serve React frontend
frontend-install:
	cd frontend && npm install

# Start React development server
frontend-dev:
	cd frontend && npm start

# Build React frontend for production
frontend-build:
	cd frontend && npm run build

# Run both API and frontend in development mode
run-dev: build-api-dev
	./$(API_APP_NAME) -port=8080 -log-file=./logs/dns-requests.log &
	cd frontend && npm start

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
	rm -f $(DNS_APP_NAME) $(WEB_APP_NAME) $(API_APP_NAME)
	rm -f coverage.out
	rm -rf logs/*
	rm -rf dist/
	rm -rf frontend/build/
	rm -rf frontend/node_modules/
	go clean -cache
	go clean -testcache

# Docker build all services
docker-build:
	docker-compose build

# Docker build specific services
docker-build-dns:
	docker-compose build dns-server

docker-build-api:
	docker-compose build api-server

docker-build-frontend:
	docker-compose build frontend

# Docker run all services
docker-run:
	docker-compose up -d

# Docker run with build
docker-up:
	docker-compose up --build -d

# Docker stop
docker-stop:
	docker-compose down

# Docker logs
docker-logs:
	docker-compose logs -f

# Start development environment
docker-dev:
	./start-dev.sh

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
build-all: dist
	@echo "Building DNS server for multiple platforms..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(DNS_APP_NAME)-linux-amd64 $(DNS_MAIN_PATH)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(DNS_APP_NAME)-linux-arm64 $(DNS_MAIN_PATH)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(DNS_APP_NAME)-darwin-amd64 $(DNS_MAIN_PATH)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(DNS_APP_NAME)-darwin-arm64 $(DNS_MAIN_PATH)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(DNS_APP_NAME)-windows-amd64.exe $(DNS_MAIN_PATH)
	@echo "Building web dashboard for multiple platforms..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(WEB_APP_NAME)-linux-amd64 $(WEB_MAIN_PATH)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(WEB_APP_NAME)-linux-arm64 $(WEB_MAIN_PATH)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(WEB_APP_NAME)-darwin-amd64 $(WEB_MAIN_PATH)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(WEB_APP_NAME)-darwin-arm64 $(WEB_MAIN_PATH)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(WEB_APP_NAME)-windows-amd64.exe $(WEB_MAIN_PATH)

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

# Show build info from DNS server binary
info:
	./$(DNS_APP_NAME) -version

# Show build info from web dashboard binary
info-web:
	./$(WEB_APP_NAME) -version

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
	@echo "  build          - Build both DNS server and web dashboard (development)"
	@echo "  build-dev      - Build both applications with debug info"
	@echo "  build-prod     - Build both applications optimized for production"
	@echo "  build-dns-dev  - Build DNS server with debug info"
	@echo "  build-dns-prod - Build DNS server optimized for production"
	@echo "  build-web-dev  - Build web dashboard with debug info"
	@echo "  build-web-prod - Build web dashboard optimized for production"
	@echo "  build-all      - Cross-compile both applications for multiple platforms"
	@echo "  run            - Run the DNS server with default settings"
	@echo "  run-web        - Run the web dashboard with default settings"
	@echo "  run-dev        - Run both applications in development mode"
	@echo "  run-dns-dev    - Run DNS server in development mode with debug logging"
	@echo "  run-web-dev    - Run web dashboard in development mode"
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
	@echo "  info           - Show version from DNS server binary"
	@echo "  info-web       - Show version from web dashboard binary"
	@echo "  help           - Show this help message"

# Default target
all: fmt vet build test 