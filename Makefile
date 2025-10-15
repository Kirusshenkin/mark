.PHONY: build run test clean deps docker help

# Variables
BINARY_NAME=bot
BINARY_PATH=bin/$(BINARY_NAME)
DOCKER_IMAGE=crypto-trading-bot
DOCKER_TAG=latest
GO_FILES=$(shell find . -name '*.go' -type f)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-w -s -X main.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# Default target
all: clean deps build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) cmd/bot/main.go
	@echo "Build complete: $(BINARY_PATH)"

# Run the application
run:
	@echo "Running application..."
	$(GORUN) cmd/bot/main.go

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f bot
	@rm -f coverage.txt coverage.html
	$(GOCLEAN)

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run --timeout=5m

# Vet code
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Check for security issues
security:
	@echo "Checking for security issues..."
	@which gosec > /dev/null || (echo "gosec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec -quiet ./...

# Build for production (Linux AMD64)
build-prod:
	@echo "Building for production (Linux AMD64)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) cmd/bot/main.go
	@echo "Production build complete: $(BINARY_PATH)"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 cmd/bot/main.go
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 cmd/bot/main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 cmd/bot/main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 cmd/bot/main.go
	@echo "Multi-platform build complete"

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Docker build with no cache
docker-build-nc:
	@echo "Building Docker image (no cache)..."
	docker build --no-cache -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Docker run (detached)
docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d
	@echo "Services started. Use 'make docker-logs' to view logs"

# Docker run (foreground)
docker-run:
	@echo "Starting services with Docker Compose (foreground)..."
	docker-compose up

# Docker stop
docker-down:
	@echo "Stopping services..."
	docker-compose down

# Docker stop and remove volumes
docker-down-v:
	@echo "Stopping services and removing volumes..."
	docker-compose down -v

# Docker restart
docker-restart:
	@echo "Restarting services..."
	docker-compose restart

# View Docker logs
docker-logs:
	docker-compose logs -f

# View bot logs only
docker-logs-bot:
	docker-compose logs -f bot

# View DB logs only
docker-logs-db:
	docker-compose logs -f postgres

# Docker exec into bot container
docker-shell:
	docker-compose exec bot sh

# Docker exec into DB container
docker-db-shell:
	docker-compose exec postgres psql -U postgres -d crypto_trading_bot

# Create database backup
backup:
	@echo "Creating database backup..."
	@mkdir -p backups
	docker-compose exec -T postgres pg_dump -U postgres crypto_trading_bot > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Backup created in backups/"

# Restore database from backup
restore:
	@echo "Enter backup file path:"
	@read -r backup_file; \
	docker-compose exec -T postgres psql -U postgres crypto_trading_bot < $$backup_file
	@echo "Database restored"

# Create .env from example
init:
	@if [ -f .env ]; then \
		echo ".env already exists. Remove it first if you want to recreate."; \
	else \
		cp .env.example .env; \
		echo ".env created from .env.example"; \
		echo "Please edit .env file with your configuration"; \
	fi

# Validate .env file
validate-env:
	@echo "Validating .env file..."
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Run 'make init' first."; \
		exit 1; \
	fi
	@grep -q "TELEGRAM_BOT_TOKEN=your" .env && echo "Warning: TELEGRAM_BOT_TOKEN not configured" || true
	@grep -q "BYBIT_API_KEY=your" .env && echo "Warning: BYBIT_API_KEY not configured" || true
	@echo "Validation complete"

# Check code quality
check: fmt vet lint test
	@echo "All checks passed!"

# Pre-commit hook
pre-commit: fmt vet lint
	@echo "Pre-commit checks passed!"

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	$(GOMOD) download
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Development environment ready!"

# Help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Building:"
	@echo "  make build         - Build the application"
	@echo "  make build-prod    - Build for production (Linux AMD64)"
	@echo "  make build-all     - Build for multiple platforms"
	@echo ""
	@echo "Running:"
	@echo "  make run           - Run the application locally"
	@echo ""
	@echo "Testing:"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make bench         - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run linter"
	@echo "  make security      - Check for security issues"
	@echo "  make check         - Run all quality checks"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps          - Install/update dependencies"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-up     - Start services (detached)"
	@echo "  make docker-run    - Start services (foreground)"
	@echo "  make docker-down   - Stop services"
	@echo "  make docker-down-v - Stop services and remove volumes"
	@echo "  make docker-restart- Restart services"
	@echo "  make docker-logs   - View all logs"
	@echo "  make docker-logs-bot   - View bot logs"
	@echo "  make docker-logs-db    - View database logs"
	@echo "  make docker-shell  - Shell into bot container"
	@echo "  make docker-db-shell   - Shell into database"
	@echo ""
	@echo "Database:"
	@echo "  make backup        - Create database backup"
	@echo "  make restore       - Restore database from backup"
	@echo ""
	@echo "Setup:"
	@echo "  make init          - Create .env from example"
	@echo "  make validate-env  - Validate .env configuration"
	@echo "  make dev-setup     - Setup development environment"
	@echo ""
