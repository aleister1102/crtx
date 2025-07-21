# Variables
BINARY_NAME=crtx
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build binary
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-$(VERSION)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-$(VERSION)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-$(VERSION)-windows-arm64.exe .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-$(VERSION)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-$(VERSION)-darwin-arm64 .

# Run tests
.PHONY: test
test:
	go test -race -coverprofile=coverage.out ./...

# Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	go test -v -race -coverprofile=coverage.out ./...

# Generate test coverage report
.PHONY: coverage
coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Run go vet
.PHONY: vet
vet:
	go vet ./...

# Run go fmt
.PHONY: fmt
fmt:
	go fmt ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf build/
	rm -f coverage.out
	rm -f coverage.html

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod verify

# Tidy dependencies
.PHONY: tidy
tidy:
	go mod tidy

# Install binary to GOPATH/bin
.PHONY: install
install:
	go install $(LDFLAGS) .

# Create a new release tag
.PHONY: tag
tag:
	@read -p "Enter tag version (e.g., v1.0.0): " tag; \
	git tag -a $$tag -m "Release $$tag"; \
	echo "Created tag $$tag. Push with: git push origin $$tag"

# Test with verbose output
.PHONY: test-verbose-run
test-verbose-run: build
	@echo "Testing verbose mode with example.com:"
	@echo "example.com" | ./$(BINARY_NAME) -v -c 3

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  test        - Run tests"
	@echo "  test-verbose- Run tests with verbose output"
	@echo "  test-verbose-run - Test the tool with verbose output"
	@echo "  coverage    - Generate test coverage report"
	@echo "  vet         - Run go vet"
	@echo "  fmt         - Run go fmt"
	@echo "  clean       - Clean build artifacts"
	@echo "  deps        - Install dependencies"
	@echo "  tidy        - Tidy dependencies"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  tag         - Create a new release tag"
	@echo "  help        - Show this help"
