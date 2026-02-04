# FeedPulse Go - Makefile

.PHONY: all build test test-verbose test-race test-cover clean fmt vet lint help install

# Binary name
BINARY_NAME=feedpulse

# Build directory
BUILD_DIR=bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: fmt vet test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/feedpulse/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GOTEST) -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -race ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with coverage summary
test-cover-summary:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f *.db
	@echo "Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Install binary
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install cmd/feedpulse/main.go

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) fetch --config config.yaml

# Development: watch and rebuild on changes (requires entr)
dev:
	@which entr > /dev/null || (echo "entr not installed. Install it first." && exit 1)
	@echo "Watching for changes..."
	find . -name '*.go' | entr -r make build

# Show test count
test-count:
	@echo "Counting tests..."
	@go test -v ./... 2>&1 | grep -c "^=== RUN" || echo "0"

# Show test LOC
test-loc:
	@echo "Counting test lines of code..."
	@find . -name "*_test.go" -exec wc -l {} + | tail -1

# Full CI pipeline
ci: fmt vet test-race test-cover-summary lint build
	@echo "CI pipeline complete"

# Help target
help:
	@echo "Available targets:"
	@echo "  all               - Format, vet, test, and build"
	@echo "  build             - Build the binary"
	@echo "  test              - Run all tests"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-cover        - Run tests with coverage (HTML report)"
	@echo "  test-cover-summary - Run tests with coverage (summary)"
	@echo "  bench             - Run benchmarks"
	@echo "  fmt               - Format code with gofmt"
	@echo "  vet               - Run go vet"
	@echo "  lint              - Run golangci-lint"
	@echo "  clean             - Remove build artifacts"
	@echo "  deps              - Download dependencies"
	@echo "  tidy              - Tidy dependencies"
	@echo "  install           - Install binary"
	@echo "  run               - Build and run"
	@echo "  dev               - Watch and rebuild (requires entr)"
	@echo "  test-count        - Count number of tests"
	@echo "  test-loc          - Count test lines of code"
	@echo "  ci                - Run full CI pipeline"
	@echo "  help              - Show this help message"
