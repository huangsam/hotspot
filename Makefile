# --- Configuration Variables ---

# Binary name (Can be overridden: make BINARY_NAME=new_app build)
BINARY_NAME   ?= hotspot
# Output directory for the built binary
BIN_DIR       ?= bin
# Main source file for the command
MAIN_FILE     ?= main.go

# Tools (Can be overridden to use specific versions or wrappers)
GO            ?= go
GOLANGCI_LINT ?= golangci-lint
GORELEASER    ?= goreleaser

# Integration tests/linting included by default (set to 0 to disable)
INTEGRATION   ?= 0

# Default target for 'make'
.DEFAULT_GOAL := build

# --- Phony Targets ---
# .PHONY: explicitly declares targets that do not represent files
.PHONY: all build clean install test bench format lint check snapshot release fuzz fuzz-quick fuzz-long profile help

# --- Targets ---

# Build the binary
# The 'build' target is now just an alias for the specific binary file path.
# This leverages Makefile's dependency rules.
build: $(BIN_DIR)/$(BINARY_NAME)

# Rule to create the binary file.
# The automatic variable $@ holds the name of the target (e.g., bin/hotspot)
$(BIN_DIR)/$(BINARY_NAME): $(MAIN_FILE)
	@echo "🛠 Building $(BINARY_NAME)..."
	# Ensure the output directory exists
	@mkdir -p $(BIN_DIR)
	# Compile the application
	@$(GO) build -o $@ $(MAIN_FILE)
	@echo "✅ Build complete: $@"

# Clean build and release artifacts
clean:
	@echo "🧹 Cleaning $(BIN_DIR) and dist directories..."
	@rm -rf $(BIN_DIR) dist
	@echo "✅ Clean complete"

# Install the built binary to $GOPATH/bin
install: $(BIN_DIR)/$(BINARY_NAME)
	@echo "📦 Installing $(BINARY_NAME) to $$(go env GOPATH)/bin..."
	@mkdir -p $$(go env GOPATH)/bin
	@cp $(BIN_DIR)/$(BINARY_NAME) $$(go env GOPATH)/bin/
	@echo "✅ Installed: $$(go env GOPATH)/bin/$(BINARY_NAME)"

# Reinstall the built binary
reinstall: clean install

# Run tests
# FORCE=1: Bypass test cache
# INTEGRATION=1: Include integration tests
test:
	@echo "🧪 Running tests..."
	@if [ "$(INTEGRATION)" = "1" ]; then \
		echo "Including integration tests..."; \
		if [ "$(FORCE)" = "1" ]; then \
			$(GO) test -count=1 ./...; \
			$(GO) test -tags integration -count=1 ./integration; \
		else \
			$(GO) test ./...; \
			$(GO) test -tags integration ./integration; \
		fi; \
	else \
		if [ "$(FORCE)" = "1" ]; then \
			$(GO) test -count=1 ./...; \
		else \
			$(GO) test ./...; \
		fi; \
	fi

# Convenience aliases for common test scenarios
test-force: export FORCE=1
test-force: test
test-all: export INTEGRATION=1
test-all: test
test-all-force: export FORCE=1
test-all-force: export INTEGRATION=1
test-all-force: test

# Run benchmarks
bench:
	@echo "⏱ Running benchmarks..."
	@$(GO) test -bench=. ./...

# Run fuzz tests
# FUZZTIME: Duration to run fuzz tests (default: 10s)
# FUZZFUNC: Specific fuzz function to run (optional)
FUZZTIME ?= 10s
fuzz:
	@echo "🔬 Running fuzz tests..."
	@if [ -n "$(FUZZFUNC)" ]; then \
		echo "Running specific fuzz function: $(FUZZFUNC)"; \
		for pkg in ./internal ./core; do \
			$(GO) test -fuzz=$(FUZZFUNC) -fuzztime=$(FUZZTIME) $$pkg || exit 1; \
		done; \
		$(GO) test -tags integration -fuzz=$(FUZZFUNC) -fuzztime=$(FUZZTIME) ./integration || exit 1; \
	else \
		for pkg in ./internal ./core; do \
			for fuzzfunc in $$($(GO) test -list=Fuzz $$pkg | grep ^Fuzz); do \
				echo "Running $$fuzzfunc in $$pkg"; \
				$(GO) test -fuzz=$$fuzzfunc -fuzztime=$(FUZZTIME) $$pkg || exit 1; \
			done; \
		done; \
		for fuzzfunc in $$($(GO) test -tags integration -list=Fuzz ./integration | grep ^Fuzz); do \
			echo "Running $$fuzzfunc in ./integration"; \
			$(GO) test -tags integration -fuzz=$$fuzzfunc -fuzztime=$(FUZZTIME) ./integration || exit 1; \
		done; \
	fi
	@echo "✅ Fuzz tests complete"

# Run full profiling workflow: build, profile, and show top functions
# PROFILE_PREFIX: Prefix for profile output files (default: hotspot-profile)
# PROFILE_ARGS: Arguments to pass to hotspot for profiling (default: files --limit 10)
PROFILE_PREFIX ?= hotspot-profile
PROFILE_ARGS ?= files --limit 10
profile: $(BIN_DIR)/$(BINARY_NAME)
	@echo "🔬 Running full profiling workflow..."
	@echo "Running: ./$(BIN_DIR)/$(BINARY_NAME) --profile $(PROFILE_PREFIX) $(PROFILE_ARGS)"
	@./$(BIN_DIR)/$(BINARY_NAME) --profile $(PROFILE_PREFIX) $(PROFILE_ARGS)
	@echo ""
	@echo "🔍 Top CPU functions:"
	@go tool pprof -top $(PROFILE_PREFIX).cpu.prof | head -20
	@echo ""
	@echo "🔍 Top memory allocations:"
	@go tool pprof -top $(PROFILE_PREFIX).mem.prof | head -20

# Convenience aliases for fuzz testing
fuzz-quick: FUZZTIME=5s
fuzz-quick: fuzz
fuzz-long: FUZZTIME=60s
fuzz-long: fuzz

# Format code
format:
	@echo "📐 Formatting code..."
	@$(GOLANGCI_LINT) run --fix
	@$(GOLANGCI_LINT) fmt
	@if [ "$(INTEGRATION)" = "1" ]; then \
		echo "📐 Including integration format..."; \
		$(GOLANGCI_LINT) run --build-tags integration --fix ./integration; \
	fi
	@echo "✅ Format complete"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@$(GOLANGCI_LINT) run
	@if [ "$(INTEGRATION)" = "1" ]; then \
		echo "🔍 Including integration lint..."; \
		$(GOLANGCI_LINT) run --build-tags integration ./integration; \
	fi
	@echo "✅ Lint complete"

# Run all checks (Format, Lint, Test)
check: format lint test
	@echo "✅ All checks passed"

# Run a snapshot release
snapshot: clean
	@echo "🚀 Running snapshot release..."
	@$(GORELEASER) release --snapshot

# Run a real release
release: clean
	@echo "🚀 Running real release..."
	@$(GORELEASER) release

# Show help
help:
	@echo
	@echo "✨ $(BINARY_NAME) Development Makefile Targets ✨"
	@echo
	@echo "  make build (default)     - Builds the binary into $(BIN_DIR)/$(BINARY_NAME)."
	@echo "  make clean               - Removes build artifacts ($(BIN_DIR)) and release files (dist)."
	@echo "  make install             - Installs the built binary to $$(go env GOPATH)/bin."
	@echo "  make reinstall           - Reinstalls the built binary."
	@echo "  make test                - Runs unit tests."
	@echo "  make test-force          - Force runs unit tests (bypasses cache)."
	@echo "  make test-all            - Runs unit + integration tests."
	@echo "  make test-all-force      - Force runs all tests (bypasses cache)."
	@echo "  make bench               - Runs Go benchmarks."
	@echo "  make fuzz                - Runs fuzz tests (default 10s, use FUZZTIME=30s)."
	@echo "  make fuzz-quick          - Runs fuzz tests for 5 seconds."
	@echo "  make fuzz-long           - Runs fuzz tests for 60 seconds."
	@echo "  make profile             - Run full profiling workflow and show top functions."
	@echo "  make format              - Runs code formatting."
	@echo "  make lint                - Runs static analysis and checks."
	@echo "  make check               - Executes format, lint, and test sequentially."
	@echo "  make snapshot            - Runs a snapshot release via $(GORELEASER)."
	@echo "  make release             - Runs a full release via $(GORELEASER)."
	@echo "  make help                - Shows this help message."
	@echo
