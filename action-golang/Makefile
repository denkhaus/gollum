# Makefile for building and pushing the Claude Code Go Action Image
# Usage: make build, make push, make test, make all

# Configuration
IMAGE_NAME ?= denkhaus/golang-claude-action
IMAGE_TAG ?= latest
FULL_IMAGE ?= $(IMAGE_NAME):$(IMAGE_TAG)
REGISTRY ?= docker.io

# Version arguments for Docker build
GOLANGCI_LINT_VERSION ?= v2.6.1
GH_VERSION ?= v2.60.0

# Color output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: all build push test login clean shell help

# Default target
all: build

# Build the Docker image
build:
	@echo "$(BLUE)Building Docker image: $(FULL_IMAGE)$(NC)"
	docker build \
		--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
		--build-arg GH_VERSION=$(GH_VERSION) \
		-t $(FULL_IMAGE) \
		-f Dockerfile .
	@echo "$(GREEN)Build complete: $(FULL_IMAGE)$(NC)"

# Push the Docker image to registry
push: build
	@echo "$(BLUE)Pushing image to registry: $(REGISTRY)/$(FULL_IMAGE)$(NC)"
	docker push $(REGISTRY)/$(FULL_IMAGE)
	@echo "$(GREEN)Push complete: $(REGISTRY)/$(FULL_IMAGE)$(NC)"

# Login to Docker registry (optional - use if you need authentication)
login:
	@echo "$(YELLOW)Logging in to $(REGISTRY)...$(NC)"
	@echo "Hint: Use 'docker login' manually or set up CI credentials"
	docker login $(REGISTRY)

# Test the Docker image locally
test:
	@echo "$(BLUE)Testing Docker image: $(FULL_IMAGE)$(NC)"
	@echo "$(YELLOW)Running go version...$(NC)"
	docker run --rm $(FULL_IMAGE) go version
	@echo "$(YELLOW)Running gh version...$(NC)"
	docker run --rm $(FULL_IMAGE) gh --version
	@echo "$(YELLOW)Running golangci-lint version...$(NC)"
	docker run --rm $(FULL_IMAGE) golangci-lint --version
	@echo "$(YELLOW)Running mockgen version...$(NC)"
	docker run --rm $(FULL_IMAGE) mockgen -version
	@echo "$(YELLOW)Checking timezone...$(NC)"
	@docker run --rm $(FULL_IMAGE) date +%Z
	@echo "$(GREEN)All tests passed!$(NC)"

# Interactive shell in the container
shell:
	@echo "$(BLUE)Starting interactive shell in $(FULL_IMAGE)$(NC)"
	docker run --rm -it $(FULL_IMAGE) /bin/sh

# Clean up dangling images
clean:
	@echo "$(BLUE)Cleaning up dangling Docker images...$(NC)"
	docker image prune -f
	@echo "$(GREEN)Cleanup complete$(NC)"

# Build and push with version tag
release: GOLANGCI_LINT_VERSION=v2.6.1
release: GH_VERSION=v2.60.0
release: IMAGE_TAG=$(shell date +%Y%m%d)
release: build push
	@echo "$(GREEN)Release $(IMAGE_TAG) pushed successfully$(NC)"

# Help target
help:
	@echo "$(BLUE)Claude Code Go Action - Makefile Help$(NC)"
	@echo ""
	@echo "Available targets:"
	@echo "  $(GREEN)all$(NC)       - Build the image (default)"
	@echo "  $(GREEN)build$(NC)     - Build the Docker image"
	@echo "  $(GREEN)push$(NC)      - Build and push to registry"
	@echo "  $(GREEN)test$(NC)      - Test the image locally"
	@echo "  $(GREEN)shell$(NC)     - Start interactive shell in container"
	@echo "  $(GREEN)login$(NC)     - Login to Docker registry"
	@echo "  $(GREEN)clean$(NC)     - Clean up dangling images"
	@echo "  $(GREEN)release$(NC)   - Build and push with date-based tag"
	@echo "  $(GREEN)help$(NC)      - Show this help message"
	@echo ""
	@echo "Configuration variables:"
	@echo "  IMAGE_NAME    - Docker image name (default: denkhaus/gollum-claude-action)"
	@echo "  IMAGE_TAG     - Docker image tag (default: latest)"
	@echo "  REGISTRY      - Docker registry (default: docker.io)"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make IMAGE_TAG=v1.0.0 push"
	@echo "  make REGISTRY=ghcr.io IMAGE_NAME=yourusername/gollum-claude-action push"
