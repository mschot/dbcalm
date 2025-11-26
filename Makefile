.PHONY: help dev kill build clean setup-runtime build-main build-db-cmd build-cmd e2e-build e2e-test e2e-test-debian-mariadb e2e-test-debian-mysql e2e-test-rocky-mariadb e2e-test-rocky-mysql e2e-clean package-deb package-rpm package-deb-docker package-rpm-docker package-both-docker package-clean install-deb install-rpm

# Enable Docker BuildKit for better caching and modern features
export DOCKER_BUILDKIT=1

# Default target
.DEFAULT_GOAL := help

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)DBCalm Master Makefile$(NC)"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

kill: ## Kill all running dbcalm processes
	@echo "$(YELLOW)Killing all dbcalm processes...$(NC)"
	@pkill -9 dbcalm 2>/dev/null || true
	@pkill -9 dbcalm-db-cmd 2>/dev/null || true
	@pkill -9 dbcalm-cmd 2>/dev/null || true
	@echo "$(GREEN)✓ All processes killed$(NC)"

build-main: ## Build main dbcalm application
	@echo "$(YELLOW)Building main dbcalm app...$(NC)"
	@$(MAKE) -C app build
	@echo "$(GREEN)✓ Main app built: app/bin/dbcalm$(NC)"

build-db-cmd: ## Build dbcalm-db-cmd service
	@echo "$(YELLOW)Building dbcalm-db-cmd service...$(NC)"
	@$(MAKE) -C cmd build-db-cmd
	@echo "$(GREEN)✓ DB cmd service built: cmd/dbcalm-db-cmd$(NC)"

build-cmd: ## Build dbcalm-cmd service
	@echo "$(YELLOW)Building dbcalm-cmd service...$(NC)"
	@$(MAKE) -C cmd build-cmd
	@echo "$(GREEN)✓ Cmd service built: cmd/dbcalm-cmd$(NC)"

build: build-main build-db-cmd build-cmd ## Build all services

setup-runtime: ## Create /var/run/dbcalm with correct permissions
	@echo "$(YELLOW)Setting up runtime directory...$(NC)"
	@sudo mkdir -p /var/run/dbcalm
	@if id -u mysql >/dev/null 2>&1; then \
		sudo chown mysql:mysql /var/run/dbcalm; \
	else \
		echo "$(YELLOW)Warning: mysql user not found, using current user$(NC)"; \
		sudo chown $(USER):$(USER) /var/run/dbcalm; \
	fi
	@sudo chmod 775 /var/run/dbcalm
	@echo "$(GREEN)✓ Runtime directory ready: /var/run/dbcalm$(NC)"

clean: ## Remove all built binaries
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(MAKE) -C app clean 2>/dev/null || true
	@$(MAKE) -C cmd clean 2>/dev/null || true
	@rm -f cmd/dbcalm-db-cmd cmd/dbcalm-cmd
	@echo "$(GREEN)✓ Clean complete$(NC)"

dev: kill build setup-runtime ## Kill processes, rebuild all, and start services (cmd services in background, main app in foreground)
	@echo ""
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)  Starting DBCalm Development Environment$(NC)"
	@echo "$(GREEN)========================================$(NC)"
	@echo ""
	@echo "$(YELLOW)Starting dbcalm-db-cmd service in background...$(NC)"
	@if id -u mysql >/dev/null 2>&1; then \
		sudo -u mysql ./cmd/dbcalm-db-cmd & \
		echo "$(GREEN)✓ dbcalm-db-cmd running (PID $$!)$(NC)"; \
	else \
		./cmd/dbcalm-db-cmd & \
		echo "$(YELLOW)⚠ dbcalm-db-cmd running as $(USER) (PID $$!)$(NC)"; \
	fi
	@sleep 1
	@echo ""
	@echo "$(YELLOW)Starting dbcalm-cmd service in background...$(NC)"
	@sudo ./cmd/dbcalm-cmd &
	@echo "$(GREEN)✓ dbcalm-cmd running (PID $$!)$(NC)"
	@sleep 1
	@echo ""
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)  Starting main dbcalm server...$(NC)"
	@echo "$(GREEN)========================================$(NC)"
	@echo ""
	@./app/bin/dbcalm --config app/config.dev.yml server

e2e-build: build ## Build all binaries and prepare for E2E tests
	@echo "$(YELLOW)Preparing E2E test artifacts...$(NC)"
	@mkdir -p tests/e2e/artifacts tests/e2e/test-results
	@cp app/bin/dbcalm tests/e2e/artifacts/
	@cp cmd/dbcalm-db-cmd tests/e2e/artifacts/
	@cp cmd/dbcalm-cmd tests/e2e/artifacts/
	@echo "$(GREEN)✓ E2E artifacts ready$(NC)"

e2e-test-debian-mariadb: e2e-build ## Run E2E tests on Debian with MariaDB
	@echo "$(GREEN)Running E2E tests: Debian + MariaDB$(NC)"
	@cd tests/e2e/common && DISTRO_DIR=debian DISTRO=debian DB_TYPE=mariadb docker compose -p dbcalm-e2e-go-deb-mariadb up --build --force-recreate --abort-on-container-exit --exit-code-from test-runner

e2e-test-debian-mysql: e2e-build ## Run E2E tests on Debian with MySQL
	@echo "$(GREEN)Running E2E tests: Debian + MySQL$(NC)"
	@cd tests/e2e/common && DISTRO_DIR=debian DISTRO=debian DB_TYPE=mysql docker compose -p dbcalm-e2e-go-deb-mysql up --build --force-recreate --abort-on-container-exit --exit-code-from test-runner

e2e-test-rocky-mariadb: e2e-build ## Run E2E tests on Rocky Linux with MariaDB
	@echo "$(GREEN)Running E2E tests: Rocky + MariaDB$(NC)"
	@cd tests/e2e/common && DISTRO_DIR=rocky DISTRO=rocky DB_TYPE=mariadb docker compose -p dbcalm-e2e-go-rocky-mariadb up --build --force-recreate --abort-on-container-exit --exit-code-from test-runner

e2e-test-rocky-mysql: e2e-build ## Run E2E tests on Rocky Linux with MySQL
	@echo "$(GREEN)Running E2E tests: Rocky + MySQL$(NC)"
	@cd tests/e2e/common && DISTRO_DIR=rocky DISTRO=rocky DB_TYPE=mysql docker compose -p dbcalm-e2e-go-rocky-mysql up --build --force-recreate --abort-on-container-exit --exit-code-from test-runner

e2e-test: e2e-test-debian-mariadb e2e-test-debian-mysql e2e-test-rocky-mariadb e2e-test-rocky-mysql ## Run all E2E tests (all OS/DB combinations)
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)  All E2E tests completed!$(NC)"
	@echo "$(GREEN)========================================$(NC)"

e2e-clean: ## Clean up E2E test artifacts and Docker resources
	@echo "$(YELLOW)Cleaning E2E test artifacts...$(NC)"
	@rm -rf tests/e2e/artifacts/* tests/e2e/test-results/*
	@cd tests/e2e/common && docker compose -p dbcalm-e2e-go-deb-mariadb down -v 2>/dev/null || true
	@cd tests/e2e/common && docker compose -p dbcalm-e2e-go-deb-mysql down -v 2>/dev/null || true
	@cd tests/e2e/common && docker compose -p dbcalm-e2e-go-rocky-mariadb down -v 2>/dev/null || true
	@cd tests/e2e/common && docker compose -p dbcalm-e2e-go-rocky-mysql down -v 2>/dev/null || true
	@echo "$(GREEN)✓ E2E cleanup complete$(NC)"

package-deb: ## Build .deb package locally (requires nFPM)
	@echo "$(GREEN)Building Debian package locally...$(NC)"
	@./build/build-deb.sh
	@echo "$(GREEN)✓ Debian package built: build/dist/*.deb$(NC)"

package-rpm: ## Build .rpm package locally (requires nFPM)
	@echo "$(GREEN)Building RPM package locally...$(NC)"
	@./build/build-rpm.sh
	@echo "$(GREEN)✓ RPM package built: build/dist/*.rpm$(NC)"

package-deb-docker: ## Build .deb package in Docker (Ubuntu 22.04)
	@echo "$(GREEN)Building Debian package in Docker...$(NC)"
	@mkdir -p build/dist && rm -f build/dist/*.deb
	@docker build -f build/Dockerfile.deb -t dbcalm-build-deb .
	@docker run --rm -v $(PWD)/build/dist:/build/build/dist dbcalm-build-deb
	@echo "$(GREEN)✓ Debian package built: build/dist/*.deb$(NC)"

package-rpm-docker: ## Build .rpm package in Docker (Rocky 9)
	@echo "$(GREEN)Building RPM package in Docker...$(NC)"
	@mkdir -p build/dist && rm -f build/dist/*.rpm
	@docker build -f build/Dockerfile.rpm -t dbcalm-build-rpm .
	@docker run --rm -v $(PWD)/build/dist:/build/build/dist dbcalm-build-rpm
	@echo "$(GREEN)✓ RPM package built: build/dist/*.rpm$(NC)"

package-both-docker: ## Build both .deb and .rpm packages in parallel using Docker
	@echo "$(GREEN)Building both Debian and RPM packages in parallel...$(NC)"
	@mkdir -p build/dist
	@rm -f build/dist/*.deb build/dist/*.rpm
	@echo "$(YELLOW)Starting Debian build...$(NC)"
	@docker build -f build/Dockerfile.deb -t dbcalm-build-deb . > /tmp/deb-build.log 2>&1 & \
	DEB_PID=$$!; \
	echo "$(YELLOW)Starting RPM build...$(NC)"; \
	docker build -f build/Dockerfile.rpm -t dbcalm-build-rpm . > /tmp/rpm-build.log 2>&1 & \
	RPM_PID=$$!; \
	echo "$(YELLOW)Waiting for Docker builds to complete...$(NC)"; \
	wait $$DEB_PID; DEB_EXIT=$$?; \
	wait $$RPM_PID; RPM_EXIT=$$?; \
	if [ $$DEB_EXIT -ne 0 ]; then \
		echo "$(RED)✗ Debian build failed!$(NC)"; \
		cat /tmp/deb-build.log; \
		exit 1; \
	fi; \
	if [ $$RPM_EXIT -ne 0 ]; then \
		echo "$(RED)✗ RPM build failed!$(NC)"; \
		cat /tmp/rpm-build.log; \
		exit 1; \
	fi; \
	echo "$(GREEN)✓ Docker images built successfully$(NC)"; \
	echo "$(YELLOW)Running package builds...$(NC)"; \
	docker run --rm -v $(PWD)/build/dist:/build/build/dist dbcalm-build-deb & \
	DEB_RUN_PID=$$!; \
	docker run --rm -v $(PWD)/build/dist:/build/build/dist dbcalm-build-rpm & \
	RPM_RUN_PID=$$!; \
	wait $$DEB_RUN_PID; DEB_RUN_EXIT=$$?; \
	wait $$RPM_RUN_PID; RPM_RUN_EXIT=$$?; \
	if [ $$DEB_RUN_EXIT -ne 0 ] || [ $$RPM_RUN_EXIT -ne 0 ]; then \
		echo "$(RED)✗ Package build failed!$(NC)"; \
		exit 1; \
	fi
	@echo ""
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)  Both packages built successfully!$(NC)"
	@echo "$(GREEN)========================================$(NC)"
	@ls -lh build/dist/

package-clean: ## Clean up package build artifacts
	@echo "$(YELLOW)Cleaning package artifacts...$(NC)"
	@rm -rf build/dist/ bin/
	@echo "$(GREEN)✓ Package artifacts cleaned$(NC)"

install-deb: ## Install .deb package locally (requires sudo)
	@echo "$(YELLOW)Installing Debian package...$(NC)"
	@sudo dpkg -i build/dist/*.deb || (sudo apt-get install -f -y && sudo dpkg -i build/dist/*.deb)
	@echo "$(GREEN)✓ Package installed successfully!$(NC)"

install-rpm: ## Install .rpm package locally (requires sudo)
	@echo "$(YELLOW)Installing RPM package...$(NC)"
	@sudo dnf install -y build/dist/*.rpm || sudo yum install -y build/dist/*.rpm
	@echo "$(GREEN)✓ Package installed successfully!$(NC)"
