# Imposter Game - Makefile
# Run 'make help' to see available commands

.PHONY: help build run test test-coverage clean lint dev deps

# Default target
help:
	@echo ""
	@echo "  Imposter Game - Available Commands"
	@echo "  ═══════════════════════════════════"
	@echo ""
	@echo "  make build         Build the server binary"
	@echo "  make run           Run the server (go run)"
	@echo "  make dev           Run with hot reload (requires 'air')"
	@echo "  make test          Run all tests"
	@echo "  make test-coverage Run tests with coverage report"
	@echo "  make lint          Run golangci-lint"
	@echo "  make clean         Remove build artifacts"
	@echo "  make deps          Download dependencies"
	@echo ""
	@echo "  make build-linux   Cross-compile for Linux (Lightsail deploy)"
	@echo "  make deploy        Deploy to production server"
	@echo ""

# ============================================
# BUILD
# ============================================
build:
	@echo "Building server..."
	@mkdir -p bin
	go build -o bin/server ./cmd/server

build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/server-linux ./cmd/server

# ============================================
# RUN
# ============================================
run:
	@echo "Starting server on http://localhost:$(or $(PORT),8080)"
	go run ./cmd/server

dev:
	@command -v air > /dev/null 2>&1 || { echo "Install 'air' first: go install github.com/air-verse/air@latest"; exit 1; }
	air

# ============================================
# TEST
# ============================================
test:
	go test -v ./...

test-coverage:
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report: coverage/coverage.html"

test-race:
	go test -race -v ./...

# ============================================
# QUALITY
# ============================================
lint:
	@command -v golangci-lint > /dev/null 2>&1 || { echo "Install golangci-lint: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run

fmt:
	go fmt ./...
	gofmt -s -w .

vet:
	go vet ./...

# ============================================
# DEPENDENCIES
# ============================================
deps:
	go mod download
	go mod tidy

# ============================================
# CLEAN
# ============================================
clean:
	rm -rf bin/
	rm -rf coverage/
	go clean

# ============================================
# DEPLOY
# ============================================
DEPLOY_HOST ?= your-lightsail-ip
DEPLOY_PATH ?= /opt/imposter

deploy: build-linux
	@echo "Deploying to $(DEPLOY_HOST)..."
	rsync -avz --progress \
		bin/server-linux \
		web/ \
		$(DEPLOY_HOST):$(DEPLOY_PATH)/
	ssh $(DEPLOY_HOST) "sudo mv $(DEPLOY_PATH)/server-linux $(DEPLOY_PATH)/bin/server && sudo systemctl restart imposter"
	@echo "✓ Deployed successfully!"

