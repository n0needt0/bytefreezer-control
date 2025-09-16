# ByteFreezer Control Service Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Binary info
BINARY_NAME=bytefreezer-control
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test deps fmt vet lint docker-build docker-run help

all: clean fmt vet test build ## Run all checks and build

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

build-linux: ## Build the binary for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) .

clean: ## Remove build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format the code
	@echo "Formatting code..."
	$(GOFMT) ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

lint: ## Run golint (requires golint installation)
	@echo "Running golint..."
	@command -v golint >/dev/null 2>&1 || { echo >&2 "golint not installed. Run: go install golang.org/x/lint/golint@latest"; exit 1; }
	golint ./...

validate-config: build ## Validate configuration
	@echo "Validating configuration..."
	./$(BINARY_NAME) --validate-config

run: build ## Build and run the service
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

run-dev: ## Run in development mode
	@echo "Running $(BINARY_NAME) in development mode..."
	$(GOCMD) run . --dev

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 \
		-v $(PWD)/config.yaml:/app/config.yaml:ro \
		$(BINARY_NAME):latest

# CI/CD helpers
ci-test: deps fmt vet test ## Run CI tests
	@echo "CI tests completed successfully"

ci-build: ci-test build-linux ## Build for CI/CD
	@echo "CI build completed successfully"

install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	cp $(BINARY_NAME) $(GOPATH)/bin/

uninstall: ## Remove binary from GOPATH/bin
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Security scanning (requires gosec)
security-scan: ## Run security scan with gosec
	@echo "Running security scan..."
	@command -v gosec >/dev/null 2>&1 || { echo >&2 "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; exit 1; }
	gosec ./...

# Performance profiling
profile-cpu: build ## Run with CPU profiling
	@echo "Running with CPU profiling..."
	./$(BINARY_NAME) -cpuprofile=cpu.prof

profile-mem: build ## Run with memory profiling
	@echo "Running with memory profiling..."
	./$(BINARY_NAME) -memprofile=mem.prof

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)