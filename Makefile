# Contact Scrape Makefile

.PHONY: build run clean install dev test help

# Variables
BINARY_NAME=contact-scrape
KONTAKT_BINARY=kontakt-service
BUILD_DIR=build
PORT=80
KONTAKT_PORT=8081

help: ## Show this help message
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
dev: ## Run both applications in development mode
	@echo "Starting main application and kontakt service..."
	@if lsof -i :$(PORT) > /dev/null; then \
		echo "Error: Port $(PORT) is already in use"; \
		echo "Please stop the existing service or change PORT in Makefile"; \
		exit 1; \
	fi
	@go run main.go &
	@sleep 2
	@cd kontakt && PORT=$(KONTAKT_PORT) go run contact-scrape.go

run: build ## Build and run both applications
	@echo "Starting $(BINARY_NAME) and $(KONTAKT_BINARY)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) &
	@cd kontakt && ./$(BUILD_DIR)/$(KONTAKT_BINARY)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

##@ Build
build: ## Build both applications
	@echo "Building $(BINARY_NAME) and $(KONTAKT_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@cd kontakt && go build -o $(BUILD_DIR)/$(KONTAKT_BINARY) .
	@echo "Binaries built: $(BUILD_DIR)/$(BINARY_NAME) and kontakt/$(BUILD_DIR)/$(KONTAKT_BINARY)"

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

##@ Service Management
install-service: build ## Install as systemd service (Linux)
	@echo "Installing kontakt service..."
	@sudo cp kontakt/contact-scrape.service /etc/systemd/system/
	@sudo systemctl daemon-reload
	@sudo systemctl enable kontakt-scrape
	@echo "Service installed. Start with: sudo systemctl start kontakt-scrape"

start-service: ## Start the kontakt service
	@sudo systemctl start kontakt-scrape

stop-service: ## Stop the kontakt service
	@sudo systemctl stop kontakt-scrape

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
		echo " contacts.xlsx found"; \
		ls -lh contacts.xlsx; \
	else \
		echo " contacts.xlsx not found"; \
		echo "Please place your Excel file in the current directory"; \
	fi

##@ Docker
docker-build: ## Build Docker images
	@echo "Building Docker images..."
	@docker build -t main-app .
	@cd kontakt && docker build -t kontakt-service .

docker-run: ## Run in Docker containers
	@echo "Running Docker containers..."
	@docker run -p $(PORT):$(PORT) main-app &
	@cd kontakt && docker run -p $(KONTAKT_PORT):$(KONTAKT_PORT) kontakt-service

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