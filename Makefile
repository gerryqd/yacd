# yacd - Yet Another CompileDB
# Makefile for building, testing, and managing the project

# Detect OS for cross-platform compatibility
ifeq ($(OS),Windows_NT)
	detected_OS := Windows
	BINARY_EXT := .exe
	RM := del /Q
	MKDIR := mkdir
	SEP := \\
	NULL_DEVICE := nul
else
	detected_OS := $(shell uname -s)
	BINARY_EXT :=
	RM := rm -f
	MKDIR := mkdir -p
	SEP := /
	NULL_DEVICE := /dev/null
endif

# Go related variables
GO := go
BINARY_NAME := yacd$(BINARY_EXT)
BUILD_DIR := build
COVERAGE_FILE := coverage.out
VERSION := $(shell git describe --tags --always --dirty 2>$(NULL_DEVICE) || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>$(NULL_DEVICE) || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X github.com/gerryqd/yacd/cmd.GitCommit=$(COMMIT)"

.PHONY: all build clean test test-coverage test-verbose lint fmt vet help run install

# Default target
all: fmt vet test build

# Build binary
build:
	@echo "Building $(BINARY_NAME) for $(detected_OS)..."
ifeq ($(detected_OS),Windows)
	@if not exist $(BUILD_DIR) $(MKDIR) $(BUILD_DIR)
else
	@$(MKDIR) $(BUILD_DIR) 2>$(NULL_DEVICE) || true
endif
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)$(SEP)$(BINARY_NAME) .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
ifeq ($(detected_OS),Windows)
	@if exist $(BUILD_DIR) rmdir /S /Q $(BUILD_DIR)
	@if exist $(COVERAGE_FILE) del /Q $(COVERAGE_FILE)
	@if exist coverage.html del /Q coverage.html
else
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE)
	@rm -f coverage.html
endif

# Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests and generate coverage report
test-coverage:
	@echo "Running tests and generating coverage report..."
	$(GO) test -v -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run verbose tests
test-verbose:
	@echo "Running verbose tests..."
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Static analysis
vet:
	@echo "Running static analysis..."
	$(GO) vet ./...

# Run golint (requires installation: go install golang.org/x/lint/golint@latest)
lint:
	@echo "Running lint checks..."
	@which golint > /dev/null || (echo "Please install golint first: go install golang.org/x/lint/golint@latest" && exit 1)
	golint ./...

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) .

# Run program with sample usage
run:
	@echo "Running program with sample usage..."
	@$(MAKE) build
	@echo "Example: Generate a sample make log and process it"
ifeq ($(detected_OS),Windows)
	@echo "make -Bnkw clean all > build.log 2>&1"
	@echo ".$(SEP)$(BUILD_DIR)$(SEP)$(BINARY_NAME) -i build.log -o $(BUILD_DIR)$(SEP)compile_commands.json -v"
else
	@echo "make -Bnkw clean all > build.log 2>&1"
	@echo "./$(BUILD_DIR)/$(BINARY_NAME) -i build.log -o $(BUILD_DIR)/compile_commands.json -v"
endif
	@echo "\nOr use direct make integration:"
ifeq ($(detected_OS),Windows)
	@echo ".$(SEP)$(BUILD_DIR)$(SEP)$(BINARY_NAME) -n \"make clean all\" -o $(BUILD_DIR)$(SEP)compile_commands.json -v"
else
	@echo "./$(BUILD_DIR)/$(BINARY_NAME) -n 'make clean all' -o $(BUILD_DIR)/compile_commands.json -v"
endif

# Run benchmark tests
benchmark:
	@echo "Running benchmark tests..."
	$(GO) test -bench=. -benchmem ./...

# Generate module dependency graph (requires installation: go install golang.org/x/tools/cmd/godepgraph@latest)
deps:
	@echo "Generating dependency graph..."
	@which godepgraph > /dev/null || (echo "Please install godepgraph first: go install golang.org/x/tools/cmd/godepgraph@latest" && exit 1)
	godepgraph -s github.com/gerryqd/yacd | dot -Tpng -o $(BUILD_DIR)/deps.png

# Tidy Go modules
mod-tidy:
	@echo "Tidying Go modules..."
	$(GO) mod tidy

# Verify Go modules
mod-verify:
	@echo "Verifying Go modules..."
	$(GO) mod verify

# Help information
help:
	@echo "yacd - Yet Another CompileDB"
	@echo ""
	@echo "Available commands:"
	@echo "  build          Build binary file"
	@echo "  clean          Clean build artifacts"
	@echo "  test           Run all tests"
	@echo "  test-coverage  Run tests and generate coverage report"
	@echo "  test-verbose   Run verbose tests"
	@echo "  fmt            Format code"
	@echo "  vet            Run static analysis"
	@echo "  lint           Run lint checks"
	@echo "  install        Install to GOPATH/bin"
	@echo "  run            Show sample usage examples"
	@echo "  benchmark      Run benchmark tests"
	@echo "  deps           Generate dependency graph"
	@echo "  mod-tidy       Tidy Go modules"
	@echo "  mod-verify     Verify Go modules"
	@echo "  help           Show this help information"
	@echo ""
	@echo "Usage examples:"
	@echo "  make build     # Build project"
	@echo "  make test      # Run tests"
	@echo "  make run       # Run program"