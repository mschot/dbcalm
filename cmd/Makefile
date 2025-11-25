.PHONY: build build-db-cmd build-cmd build-all clean install install-db-cmd install-cmd install-all test test-db-cmd test-cmd test-all fmt vet run-db-cmd run-cmd

DB_CMD_BINARY=dbcalm-db-cmd
CMD_BINARY=dbcalm-cmd
INSTALL_PATH=/usr/local/bin

# Build both services
build-all: build-db-cmd build-cmd

# Build db-cmd service
build-db-cmd:
	@echo "Building $(DB_CMD_BINARY)..."
	@ln -sf go-db-cmd.mod go.mod
	@ln -sf go-db-cmd.sum go.sum
	go build -o $(DB_CMD_BINARY) cmd/db-cmd/main.go

# Build cmd service
build-cmd:
	@echo "Building $(CMD_BINARY)..."
	@ln -sf go-cmd.mod go.mod
	@ln -sf go-cmd.sum go.sum
	go build -o $(CMD_BINARY) cmd/cmd/main.go

# Default build target (builds both)
build: build-all

# Build static binaries
build-static-db-cmd:
	@echo "Building static $(DB_CMD_BINARY)..."
	@ln -sf go-db-cmd.mod go.mod
	@ln -sf go-db-cmd.sum go.sum
	CGO_ENABLED=1 go build -ldflags '-linkmode external -extldflags "-static"' -o $(DB_CMD_BINARY) cmd/db-cmd/main.go

build-static-cmd:
	@echo "Building static $(CMD_BINARY)..."
	@ln -sf go-cmd.mod go.mod
	@ln -sf go-cmd.sum go.sum
	CGO_ENABLED=1 go build -ldflags '-linkmode external -extldflags "-static"' -o $(CMD_BINARY) cmd/cmd/main.go

build-static: build-static-db-cmd build-static-cmd

# Install targets
install-db-cmd: build-db-cmd
	@echo "Installing $(DB_CMD_BINARY) to $(INSTALL_PATH)..."
	sudo cp $(DB_CMD_BINARY) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(DB_CMD_BINARY)

install-cmd: build-cmd
	@echo "Installing $(CMD_BINARY) to $(INSTALL_PATH)..."
	sudo cp $(CMD_BINARY) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(CMD_BINARY)

install-all: install-db-cmd install-cmd

install: install-all

rundir:
	@echo "Creating /var/run/dbcalm directory..."
	sudo mkdir -p /var/run/dbcalm
	sudo chown mysql:dbcalm /var/run/dbcalm
	sudo chmod 775 /var/run/dbcalm

# Clean targets
clean:
	@echo "Cleaning..."
	rm -f $(DB_CMD_BINARY) $(CMD_BINARY)
	rm -f go.mod go.sum

# Test targets
test-db-cmd:
	@echo "Running db-cmd tests..."
	@ln -sf go-db-cmd.mod go.mod
	@ln -sf go-db-cmd.sum go.sum
	go test -v ./db-cmd-internal/... ./cmd/db-cmd/...

test-cmd:
	@echo "Running cmd tests..."
	@ln -sf go-cmd.mod go.mod
	@ln -sf go-cmd.sum go.sum
	go test -v ./cmd-internal/... ./cmd/cmd/...

test-shared:
	@echo "Running shared tests..."
	cd shared && go test -v ./...

test-all: test-shared test-db-cmd test-cmd

test: test-all

# Format targets
fmt-db-cmd:
	@echo "Formatting db-cmd code..."
	@ln -sf go-db-cmd.mod go.mod
	@ln -sf go-db-cmd.sum go.sum
	go fmt ./db-cmd-internal/... ./cmd/db-cmd/...

fmt-cmd:
	@echo "Formatting cmd code..."
	@ln -sf go-cmd.mod go.mod
	@ln -sf go-cmd.sum go.sum
	go fmt ./cmd-internal/... ./cmd/cmd/...

fmt-shared:
	@echo "Formatting shared code..."
	cd shared && go fmt ./...

fmt: fmt-shared fmt-db-cmd fmt-cmd

# Vet targets
vet-db-cmd:
	@echo "Vetting db-cmd code..."
	@ln -sf go-db-cmd.mod go.mod
	@ln -sf go-db-cmd.sum go.sum
	go vet ./db-cmd-internal/... ./cmd/db-cmd/...

vet-cmd:
	@echo "Vetting cmd code..."
	@ln -sf go-cmd.mod go.mod
	@ln -sf go-cmd.sum go.sum
	go vet ./cmd-internal/... ./cmd/cmd/...

vet-shared:
	@echo "Vetting shared code..."
	cd shared && go vet ./...

vet: vet-shared vet-db-cmd vet-cmd

# Run targets
run-db-cmd: build-db-cmd rundir
	@echo "Running $(DB_CMD_BINARY)..."
	sudo -u mysql ./$(DB_CMD_BINARY)

run-cmd: build-cmd rundir
	@echo "Running $(CMD_BINARY)..."
	sudo ./$(CMD_BINARY)

# Help target
help:
	@echo "Available targets:"
	@echo "  build, build-all       - Build both services (default)"
	@echo "  build-db-cmd          - Build database command service"
	@echo "  build-cmd             - Build system command service"
	@echo "  build-static          - Build static binaries for both services"
	@echo "  install, install-all  - Install both services"
	@echo "  install-db-cmd        - Install database command service"
	@echo "  install-cmd           - Install system command service"
	@echo "  clean                 - Remove built binaries"
	@echo "  test, test-all        - Run all tests"
	@echo "  test-db-cmd           - Run database command service tests"
	@echo "  test-cmd              - Run system command service tests"
	@echo "  test-shared           - Run shared package tests"
	@echo "  fmt                   - Format all code"
	@echo "  vet                   - Vet all code"
	@echo "  run-db-cmd            - Build and run database command service"
	@echo "  run-cmd               - Build and run system command service"
	@echo "  help                  - Show this help message"
