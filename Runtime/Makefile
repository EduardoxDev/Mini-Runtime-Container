# GoContainer — Minimal Container Runtime
# ==========================================
# Build, test, and manage the container runtime

BINARY_NAME := gocontainer
BUILD_DIR := bin
MAIN_PKG := ./cmd/gocontainer

# Version info (injected via ldflags)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -ldflags "-s -w \
	-X github.com/user/gocontainer/internal/cli.Version=$(VERSION) \
	-X github.com/user/gocontainer/internal/cli.GitCommit=$(GIT_COMMIT) \
	-X github.com/user/gocontainer/internal/cli.BuildDate=$(BUILD_DATE)"

# Go settings
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0

# Docker settings
DOCKER_IMAGE := gocontainer
DOCKER_TAG := latest

# Alpine minirootfs
ALPINE_VERSION := 3.19
ALPINE_ARCH := x86_64
ROOTFS_DIR := rootfs
ROOTFS_URL := https://dl-cdn.alpinelinux.org/alpine/v$(ALPINE_VERSION)/releases/$(ALPINE_ARCH)/alpine-minirootfs-$(ALPINE_VERSION).0-$(ALPINE_ARCH).tar.gz

# ==========================================
# Main Targets
# ==========================================

.PHONY: all
all: lint test build ## Build, test, and lint everything

.PHONY: build
build: ## Compile the binary for linux/amd64
	@echo "==> Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PKG)
	@echo "==> Binary: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: install
install: build ## Install binary to $GOPATH/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# ==========================================
# Testing
# ==========================================

.PHONY: test
test: ## Run unit tests
	@echo "==> Running unit tests..."
	go test -v -race -count=1 ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	@echo "==> Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "==> Coverage report: coverage.html"

.PHONY: bench
bench: ## Run benchmarks
	@echo "==> Running benchmarks..."
	go test -bench=. -benchmem -benchtime=3s ./internal/...

.PHONY: integration
integration: build rootfs ## Run integration tests (requires root/privileged)
	@echo "==> Running integration tests..."
	go test -v -tags=integration -count=1 ./...

# ==========================================
# Code Quality
# ==========================================

.PHONY: lint
lint: ## Run linter
	@echo "==> Running linter..."
	@which golangci-lint > /dev/null 2>&1 || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"
	golangci-lint run ./... || true

.PHONY: fmt
fmt: ## Format code
	@echo "==> Formatting code..."
	gofmt -s -w .

.PHONY: vet
vet: ## Run go vet
	@echo "==> Running go vet..."
	go vet ./...

# ==========================================
# Root Filesystem
# ==========================================

.PHONY: rootfs
rootfs: ## Download Alpine minirootfs
	@echo "==> Downloading Alpine minirootfs..."
	@mkdir -p $(ROOTFS_DIR)
	@if [ ! -f $(ROOTFS_DIR)/.extracted ]; then \
		curl -fSL $(ROOTFS_URL) | tar xz -C $(ROOTFS_DIR) && \
		touch $(ROOTFS_DIR)/.extracted; \
	else \
		echo "==> Rootfs already extracted"; \
	fi

# ==========================================
# Docker
# ==========================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "==> Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-test
docker-test: ## Run tests inside Docker
	@echo "==> Running tests in Docker..."
	docker build --target test -t $(DOCKER_IMAGE)-test .
	docker run --rm $(DOCKER_IMAGE)-test

.PHONY: docker-run
docker-run: docker-build ## Run a container using Docker (privileged)
	@echo "==> Running gocontainer in Docker..."
	docker run --rm -it --privileged $(DOCKER_IMAGE):$(DOCKER_TAG) run /bin/sh

.PHONY: docker-bench
docker-bench: ## Run benchmarks inside Docker
	@echo "==> Running benchmarks in Docker..."
	docker build --target test -t $(DOCKER_IMAGE)-test .
	docker run --rm $(DOCKER_IMAGE)-test go test -bench=. -benchmem ./internal/...

# ==========================================
# Cleanup
# ==========================================

.PHONY: clean
clean: ## Remove build artifacts
	@echo "==> Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "==> Clean!"

.PHONY: clean-all
clean-all: clean ## Remove all generated files including rootfs
	@rm -rf $(ROOTFS_DIR)

# ==========================================
# Help
# ==========================================

.PHONY: help
help: ## Show this help message
	@echo "GoContainer — Minimal Container Runtime"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
