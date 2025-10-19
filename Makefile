.PHONY: build clean install test fmt vet help

# Binary name
BINARY_NAME=hotspot

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go
	@echo "✓ Build complete: ./$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "✓ Clean complete"

# Install to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BINARY_NAME) $(GOPATH)/bin/
	@echo "✓ Installed: $(GOPATH)/bin/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Vet complete"

# Run all checks
check: fmt vet test
	@echo "✓ All checks passed"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build    - Build the binary"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make install  - Install to \$$GOPATH/bin"
	@echo "  make test     - Run tests"
	@echo "  make fmt      - Format code"
	@echo "  make vet      - Run go vet"
	@echo "  make check    - Run fmt, vet, and test"
	@echo "  make help     - Show this help message"

# Default target
.DEFAULT_GOAL := build
