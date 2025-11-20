.PHONY: build run test clean docker-build docker-push help

# Build variables
BINARY_NAME=whoami
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-w -s -X 'main.Version=$(VERSION)' -X 'main.GitCommit=$(GIT_COMMIT)' -X 'main.BuildTime=$(BUILD_TIME)'"

# Docker variables
DOCKER_REGISTRY?=docker.io
DOCKER_IMAGE?=$(DOCKER_REGISTRY)/whoami
DOCKER_TAG?=$(VERSION)

help: ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) .

run: build ## Run the application locally
	./$(BINARY_NAME)

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

coverage: test ## Show test coverage
	go tool cover -html=coverage.out

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out

docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.

docker-push: docker-build ## Push Docker image to registry
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

docker-run: docker-build ## Run Docker container locally
	docker run --rm -p 8080:8080 \
		-e POD_IP=127.0.0.1 \
		-e HOST_IP=127.0.0.1 \
		-e POD_NAMESPACE=default \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...
	go vet ./...

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

.DEFAULT_GOAL := help
