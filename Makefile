.PHONY: help build test clean install lint fmt vet run dev

# Variables
BINARY_NAME=ultrathink
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
# Build tags for FTS5 support
BUILD_TAGS=-tags "fts5"
CGO_ENABLED=CGO_ENABLED=1

# Default target
help:
	@echo "Ultrathink - AI-Powered Memory System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run all tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install       - Install binary to GOPATH/bin"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make run           - Build and run"
	@echo "  make dev           - Run in development mode"
	@echo "  make deps          - Download dependencies"
	@echo "  make build-all     - Build for all platforms"

# Build the binary (requires CGO for SQLite)
build:
	@echo "Building $(BINARY_NAME)..."
	$(CGO_ENABLED) go build $(BUILD_TAGS) $(LDFLAGS) -o $(BINARY_NAME) cmd/ultrathink/main.go
	@echo "✅ Build complete: ./$(BINARY_NAME)"

# Build for all platforms (requires cross-compilation CGO setup)
build-all:
	@echo "Building for all platforms..."
	@mkdir -p dist
	$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 go build $(BUILD_TAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 cmd/ultrathink/main.go
	$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 go build $(BUILD_TAGS) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 cmd/ultrathink/main.go
	@echo "✅ Multi-platform build complete in dist/"
	@echo "Note: Linux/Windows builds require cross-compilation CGO setup"
	@ls -lh dist/

# Run tests (with FTS5 support)
test:
	@echo "Running tests..."
	$(CGO_ENABLED) go test $(BUILD_TAGS) ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(CGO_ENABLED) go test $(BUILD_TAGS) ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Run tests verbosely
test-verbose:
	@echo "Running tests (verbose)..."
	$(CGO_ENABLED) go test $(BUILD_TAGS) ./... -v -race

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.out coverage.html
	go clean
	@echo "✅ Clean complete"

# Install binary
install:
	@echo "Installing $(BINARY_NAME)..."
	$(CGO_ENABLED) go install $(BUILD_TAGS) $(LDFLAGS) cmd/ultrathink/main.go
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run linters
lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint not installed. Run: brew install golangci-lint" && exit 1)
	$(CGO_ENABLED) golangci-lint run $(BUILD_TAGS) ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	$(CGO_ENABLED) go vet $(BUILD_TAGS) ./...
	@echo "✅ Vet complete"

# Run the binary
run: build
	./$(BINARY_NAME)

# Development mode with live reload (requires air)
dev:
	@which air > /dev/null || (echo "❌ air not installed. Run: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies ready"

# Verify environment
verify:
	@echo "Verifying development environment..."
	@go version || (echo "❌ Go not installed" && exit 1)
	@sqlite3 --version || (echo "❌ SQLite not installed" && exit 1)
	@node --version || (echo "⚠️  Node.js not installed (needed for npm wrapper)")
	@echo "✅ Environment verified"
