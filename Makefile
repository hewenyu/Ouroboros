# DevOps Agent Makefile

# Variables
BINARY_NAME=devops-agent
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w"
CGO_ENABLED=1

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/server

# Build static binary (for minimal containers)
.PHONY: build-static
build-static:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
		-ldflags '-linkmode external -extldflags "-static" -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -s -w' \
		-o $(BINARY_NAME) ./cmd/server

# Run the application
.PHONY: run
run:
	MCP_ENABLED=false go run ./cmd/server

# Run with MCP stdio
.PHONY: run-mcp
run-mcp:
	go run ./cmd/server

# Run tests
.PHONY: test
test:
	go test -v -race -cover ./...

# Run tests with coverage
.PHONY: coverage
coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	go vet ./...

# Download dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f *.db *.db-wal *.db-shm

# Install development tools
.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Docker build
.PHONY: docker
docker:
	docker build -t $(BINARY_NAME):$(VERSION) .

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Build the binary (default)"
	@echo "  build        - Build the binary"
	@echo "  build-static - Build static binary for minimal containers"
	@echo "  run          - Run the application (MCP disabled)"
	@echo "  run-mcp      - Run the application with MCP stdio"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  tools        - Install development tools"
	@echo "  docker       - Build Docker image"
	@echo "  help         - Show this help message"
