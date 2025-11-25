# DBCalm Go Application Makefile

# Variables
BINARY_NAME=dbcalm
BUILD_DIR=bin
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

# Main targets
.PHONY: all build run clean install test help

all: clean build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbcalm
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build with optimizations
build-optimized:
	@echo "Building $(BINARY_NAME) with optimizations..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbcalm
	@echo "Optimized build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the server
run: build
	@echo "Starting DBCalm server..."
	./$(BUILD_DIR)/$(BINARY_NAME) server

# Run without building (useful during development)
run-dev:
	@echo "Starting DBCalm server (dev mode)..."
	DBCALM_DEV_MODE=1 $(GO) run ./cmd/dbcalm --config config.dev.yml server

# Run specific command
run-cmd:
	@echo "Running: $(BINARY_NAME) $(CMD)"
	./$(BUILD_DIR)/$(BINARY_NAME) $(CMD)

# User management commands
user-add:
	./$(BUILD_DIR)/$(BINARY_NAME) users add $(USER)

user-list:
	./$(BUILD_DIR)/$(BINARY_NAME) users list

# Client management commands
client-add:
	./$(BUILD_DIR)/$(BINARY_NAME) clients add "$(LABEL)"

client-list:
	./$(BUILD_DIR)/$(BINARY_NAME) clients list

# Install to system
install: build-optimized
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete"

# Uninstall from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	$(GO) clean
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -cover ./...
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/dbcalm
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/dbcalm
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/dbcalm
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/dbcalm
	@echo "Multi-platform build complete"

# Show help
help:
	@echo "DBCalm Go Application - Available targets:"
	@echo ""
	@echo "  make build              - Build the application"
	@echo "  make build-optimized    - Build with size optimizations"
	@echo "  make run                - Build and run the server"
	@echo "  make run-dev            - Run in development mode (no build)"
	@echo "  make run-cmd CMD='...'  - Run a specific command"
	@echo ""
	@echo "User management:"
	@echo "  make user-add USER=username    - Add a new user"
	@echo "  make user-list                 - List all users"
	@echo ""
	@echo "Client management:"
	@echo "  make client-add LABEL='name'   - Add a new client"
	@echo "  make client-list               - List all clients"
	@echo ""
	@echo "Installation:"
	@echo "  make install            - Install to /usr/local/bin"
	@echo "  make uninstall          - Remove from /usr/local/bin"
	@echo ""
	@echo "Development:"
	@echo "  make test               - Run tests"
	@echo "  make test-coverage      - Run tests with coverage"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Lint code (requires golangci-lint)"
	@echo "  make tidy               - Tidy dependencies"
	@echo "  make deps               - Download dependencies"
	@echo ""
	@echo "Other:"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make build-all          - Build for multiple platforms"
	@echo "  make help               - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make run                          - Start the server"
	@echo "  make user-add USER=admin          - Add admin user"
	@echo "  make run-cmd CMD='backup full'    - Create full backup"
