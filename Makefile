.PHONY: dev install

dev:
	@echo "Starting development server..."
	go run main.go

install:
	@echo "Installing dependencies..."
	go mod tidy
	go get -u gopkg.in/gomail.v2
	@echo "Build complete. Run with: go run main.go"
