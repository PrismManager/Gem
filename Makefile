.PHONY: build test lint clean install uninstall run dev check

# Build settings
BINARY_NAME=gem
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d %H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X 'main.BuildTime=$(BUILD_TIME)'"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME)
	@echo "Build complete: $(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@go clean
	@echo "Clean complete"

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)"

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from /usr/local/bin/$(BINARY_NAME)"

# Run the binary
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Build for multiple platforms
release:
	@echo "Building for multiple platforms..."
	@mkdir -p release
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o release/$(BINARY_NAME)-linux-amd64
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o release/$(BINARY_NAME)-linux-arm64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o release/$(BINARY_NAME)-darwin-amd64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o release/$(BINARY_NAME)-darwin-arm64
	@echo "Release builds complete in release/ directory"

dev: build install check
	@echo ""
	@echo "Finished building and installing $(BINARY_NAME)"

check: 
	@$(BINARY_NAME) 