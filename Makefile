# Variables
APP_NAME := distributed-llm
DOCKER_TAG := latest
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Go build variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Binary names and paths
AGENT_BINARY := bin/agent
TUI_BINARY := bin/tui
PROTO_DIR := pkg/proto

.PHONY: help build clean test run deps fmt lint docker k8s proto
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build all binaries
build: proto $(AGENT_BINARY) $(TUI_BINARY)

$(AGENT_BINARY): proto
	@echo "Building agent binary..."
	$(GOBUILD) -o $(AGENT_BINARY) ./cmd/agent

$(TUI_BINARY): proto
	@echo "Building TUI binary..."
	$(GOBUILD) -o $(TUI_BINARY) ./cmd/tui

## proto: Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p $(PROTO_DIR)
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/node.proto

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf $(PROTO_DIR)/
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

## deps: Install and tidy dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found, skipping import formatting"; \
	fi

## lint: Lint code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, please install it"; \
	fi

## test: Run all tests
test:
	$(GOTEST) ./...

## test-all: Run comprehensive test suite
test-all:
	@echo "Running comprehensive test suite..."
	./scripts/run-tests.sh

## test-unit: Run unit tests with coverage
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) ./pkg/... ./internal/...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v ./test/integration/...

## test-e2e: Run end-to-end tests
test-e2e:
	@echo "Running end-to-end tests..."
	$(GOTEST) -v ./test/e2e/...

## test-fuzz: Run fuzz tests
test-fuzz:
	@echo "Running fuzz tests..."
	$(GOTEST) -fuzz=. -fuzztime=30s ./test/fuzz/...

## test-bench: Run benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem ./pkg/... ./internal/...

## test-coverage: Generate coverage report
test-coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./pkg/... ./internal/...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	go tool cover -func=$(COVERAGE_FILE)

## test-quick: Run quick test suite
test-quick:
	@echo "Running quick test suite..."
	./scripts/run-tests.sh --quick

## test-ci: Run CI test suite
test-ci:
	@echo "Running CI test suite..."
	./scripts/run-tests.sh --no-e2e --fuzz-time=10s

## run-agent: Run agent locally
run-agent:
	$(GOCMD) run ./cmd/agent

## run-tui: Run TUI locally
run-tui:
	$(GOCMD) run ./cmd/tui

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(DOCKER_TAG) .

## minikube-setup: Setup minikube for development
minikube-setup:
	@echo "Setting up minikube cluster..."
	./scripts/minikube-dev-setup.sh

## minikube-start: Start minikube cluster and deploy
minikube-start:
	@echo "Starting minikube cluster..."
	minikube start --driver=docker --cpus=4 --memory=8192 --nodes=3
	minikube addons enable storage-provisioner
	eval $$(minikube docker-env) && $(MAKE) docker-build
	kubectl apply -f deployments/

## minikube-stop: Stop minikube cluster
minikube-stop:
	@echo "Stopping minikube cluster..."
	minikube stop

## minikube-clean: Delete minikube cluster
minikube-clean:
	@echo "Deleting minikube cluster..."
	minikube delete

## minikube-reload: Reload application in minikube
minikube-reload:
	@echo "Reloading image into minikube..."
	eval $$(minikube docker-env) && $(MAKE) docker-build
	kubectl rollout restart daemonset/$(APP_NAME)-agent

## minikube-logs: Show application logs in minikube
minikube-logs:
	kubectl logs -l app=$(APP_NAME)-agent --tail=100 -f

## minikube-dashboard: Open minikube dashboard
minikube-dashboard:
	minikube dashboard

## kind-setup: Setup kind cluster for development
kind-setup:
	@echo "Setting up kind cluster..."
	./scripts/local-dev-setup.sh

## kind-start: Start kind cluster and deploy
kind-start:
	@echo "Creating kind cluster..."
	kind create cluster --name $(APP_NAME)
	$(MAKE) docker-build
	kind load docker-image $(APP_NAME):$(DOCKER_TAG) --name $(APP_NAME)
	kubectl apply -f deployments/

## kind-stop: Delete kind cluster
kind-stop:
	@echo "Deleting kind cluster..."
	kind delete cluster --name $(APP_NAME)

## kind-reload: Reload application in kind cluster
kind-reload:
	@echo "Reloading image into kind cluster..."
	$(MAKE) docker-build
	kind load docker-image $(APP_NAME):$(DOCKER_TAG) --name $(APP_NAME)
	kubectl rollout restart deployment

## kind-logs: Show application logs in kind
kind-logs:
	kubectl logs -l app=$(APP_NAME) --tail=100 -f

## k3d-start: Start k3d cluster and deploy
k3d-start:
	@echo "Starting k3d cluster..."
	k3d cluster create $(APP_NAME) --agents 3 --port "8080:80@loadbalancer"
	kubectl apply -f deployments/

## k3d-stop: Stop k3d cluster
k3d-stop:
	@echo "Stopping k3d cluster..."
	k3d cluster stop $(APP_NAME)

## k3d-clean: Delete k3d cluster
k3d-clean:
	@echo "Deleting k3d cluster..."
	k3d cluster delete $(APP_NAME)
