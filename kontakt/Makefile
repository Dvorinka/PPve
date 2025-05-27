# Makefile for kontakt service
.PHONY: dev run build clean

PORT ?= 8081
BINARY_NAME := kontakt-service
BUILD_DIR := build

##@ Development
dev: ## Run in development mode
	@echo "Starting kontakt service..."
	@go run contact-scrape.go

run: build ## Run built binary
	@$(BUILD_DIR)/$(BINARY_NAME)

build: ## Build the application
	@echo "Building kontakt service..."
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) contact-scrape.go

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)
	@echo "Build artifacts removed"