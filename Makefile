.PHONY: help dev kill build clean setup-runtime build-main build-db-cmd build-cmd

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
