# Contact Scrape Makefile

.PHONY: build run clean install dev test help

# Variables
BINARY_NAME=contact-scrape
BUILD_DIR=build
PORT=8080

help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
dev: ## Run the application in development mode
	@echo "Starting development server..."
	@go run .

run: build ## Build and run the application
	@echo "Starting $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

##@ Build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux: ## Build for Linux (useful for deployment)
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux .
	@echo "Linux binary built: $(BUILD_DIR)/$(BINARY_NAME)-linux"

##@ Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

##@ Deployment
install: build ## Install the service (requires sudo)
	@echo "Installing contact-scrape service..."
	@sudo mkdir -p /opt/contact-scrape
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /opt/contact-scrape/
	@sudo cp index.html /opt/contact-scrape/
	@sudo mkdir -p /opt/contact-scrape/data
	@sudo chown -R www-data:www-data /opt/contact-scrape
	@sudo cp contact-scrape.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable contact-scrape
	@echo "Service installed. Start with: sudo systemctl start contact-scrape"

uninstall: ## Uninstall the service (requires sudo)
	@echo "Uninstalling contact-scrape service..."
	@sudo systemctl stop contact-scrape 2>/dev/null || true
	@sudo systemctl disable contact-scrape 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/contact-scrape.service
	@sudo systemctl daemon-reload
	@sudo rm -rf /opt/contact-scrape
	@echo "Service uninstalled."

status: ## Check service status
	@sudo systemctl status contact-scrape

start: ## Start the service
	@sudo systemctl start contact-scrape

stop: ## Stop the service
	@sudo systemctl stop contact-scrape

restart: ## Restart the service
	@sudo systemctl restart contact-scrape

logs: ## View service logs
	@sudo journalctl -u contact-scrape -f

##@ File Management
upload-xlsx: ## Upload contacts.xlsx to server (set SERVER variable)
	@if [ -z "$(SERVER)" ]; then echo "Usage: make upload-xlsx SERVER=user@hostname"; exit 1; fi
	@echo "Uploading contacts.xlsx to $(SERVER)..."
	@scp contacts.xlsx $(SERVER):/opt/contact-scrape/
	@ssh $(SERVER) "sudo chown www-data:www-data /opt/contact-scrape/contacts.xlsx"
	@ssh $(SERVER) "sudo systemctl restart contact-scrape"
	@echo "File uploaded and service restarted."

##@ Monitoring
monitor: ## Monitor the service with file watching
	@echo "Monitoring contacts.xlsx for changes..."
	@while true; do \
		inotifywait -e modify contacts.xlsx 2>/dev/null && \
		echo "File changed, reloading..." && \
		curl -X POST http://localhost:$(PORT)/kontakt/reload; \
		sleep 1; \
	done

##@ Utilities
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f data/contacts.json

setup-dirs: ## Create necessary directories
	@mkdir -p data
	@mkdir -p $(BUILD_DIR)

check-file: ## Check if contacts.xlsx exists and show info
	@if [ -f "contacts.xlsx" ]; then \
		echo "✓ contacts.xlsx found"; \
		ls -lh contacts.xlsx; \
	else \
		echo "✗ contacts.xlsx not found"; \
		echo "Please place your Excel file in the current directory"; \
	fi

##@ Docker (Optional)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t contact-scrape .

docker-run: ## Run in Docker container
	@echo "Running Docker container..."
	@docker run -p $(PORT):$(PORT) -v $(PWD)/contacts.xlsx:/app/contacts.xlsx -v $(PWD)/data:/app/data contact-scrape

##@ Information
info: ## Show application information
	@echo "Contact Scrape Application"
	@echo "========================="
	@echo "Port: $(PORT)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Build dir: $(BUILD_DIR)"
	@echo ""
	@echo "Endpoints:"
	@echo "  http://localhost:$(PORT)/kontakt         - Web interface"
	@echo "  http://localhost:$(PORT)/kontakt/contacts - JSON API"
	@echo "  http://localhost:$(PORT)/kontakt/reload   - Reload data (POST)"