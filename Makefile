# Root Makefile for PROG+HTML project
.PHONY: all dev run build clean help

# Variables
MAIN_PORT ?= 80
KONTAKT_PORT ?= 8081
BUILD_DIR = build

##@ Development
alldev: ## Start both main and kontakt services
	@echo "Starting all services..."
	@make -j2 main-dev kontakt-dev

main-dev: ## Start main service
	@echo "Starting main service on port $(MAIN_PORT)..."
	@PORT=$(MAIN_PORT) go run main.go

kontakt-dev: ## Start kontakt service
	@echo "Starting kontakt service on port $(KONTAKT_PORT)..."
	@cd kontakt && PORT=$(KONTAKT_PORT) make dev

##@ Build
build: ## Build all services
	@echo "Building all services..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/main main.go
	@cd kontakt && make build

##@ Clean
clean: ## Clean all build artifacts
	@rm -rf $(BUILD_DIR)
	@cd kontakt && make clean
	@echo "All build artifacts removed"

##@ Help
help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)