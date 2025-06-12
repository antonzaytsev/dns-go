.PHONY: build run test clean docker-build docker-run fmt vet lint

# Build the application
build:
	go build -o dns-server .

# Run the application with default settings
run:
	./dns-server

# Run tests
test:
	go test -v ./...

# Run with development settings
run-dev:
	go run . -listen=127.0.0.1 -port=5053 -log=./logs/dns-requests.log -log-level=debug -cache-size=1000

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f dns-server
	rm -rf logs/*

# Docker build
docker-build:
	docker build -t dns-go .

# Docker run
docker-run:
	docker-compose up -d

# Docker stop
docker-stop:
	docker-compose down

# Install dependencies
deps:
	go mod tidy
	go mod download

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the DNS server binary"
	@echo "  run         - Run the DNS server with default settings"
	@echo "  run-dev     - Run in development mode with debug logging"
	@echo "  test        - Run all tests"
	@echo "  fmt         - Format Go code"
	@echo "  vet         - Run go vet"
	@echo "  clean       - Clean build artifacts and logs"
	@echo "  docker-build- Build Docker image"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  docker-stop - Stop Docker containers"
	@echo "  deps        - Install and tidy dependencies"
	@echo "  help        - Show this help message"

# Default target
all: fmt vet build 