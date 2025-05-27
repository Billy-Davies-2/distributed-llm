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

## run-tui-docker: Run TUI locally connecting to Docker agents
run-tui-docker: 
	$(GOCMD) run ./cmd/tui --docker --log-level=debug

## run-tui-k8s: Run TUI locally connecting to Kubernetes agents
run-tui-k8s:
	$(GOCMD) run ./cmd/tui --k8s-namespace=default --log-level=debug

## run-tui-k3d: Run TUI locally connecting to k3d cluster
run-tui-k3d:
	$(GOCMD) run ./cmd/tui --seed-nodes=localhost:8080 --docker --log-level=debug

## run-tui-local: Run TUI locally connecting to local agents
run-tui-local:
	$(GOCMD) run ./cmd/tui --seed-nodes=localhost:8080,localhost:8081,localhost:8082 --log-level=debug

## setup-k3d: Setup local k3d cluster for development
setup-k3d:
	@echo "Setting up k3d development cluster..."
	./hack/setup-local-k3d.sh

## setup-kind: Setup local kind cluster for development
setup-kind:
	@echo "Setting up kind development cluster..."
	./hack/setup-local-k8s.sh

## k3d-logs: View agent logs in k3d cluster
k3d-logs:
	kubectl logs -l app=distributed-llm-agent -f

## k3d-status: Check status of k3d cluster
k3d-status:
	@echo "Cluster status:"
	@k3d cluster list
	@echo "\nPods:"
	@kubectl get pods -o wide
	@echo "\nServices:"
	@kubectl get svc

## k3d-cleanup: Clean up k3d cluster
k3d-cleanup:
	k3d cluster delete distributed-llm || true
	k3d registry delete k3d-registry.localhost || true

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

## metrics: Start local Prometheus server for development
metrics:
	@echo "Starting Prometheus server..."
	@mkdir -p tmp/prometheus
	@docker run -d --name prometheus-dev \
		-p 9091:9090 \
		-v $(PWD)/deployments/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
		-v $(PWD)/tmp/prometheus:/prometheus \
		prom/prometheus:v2.45.0 \
		--config.file=/etc/prometheus/prometheus.yml \
		--storage.tsdb.path=/prometheus \
		--web.console.libraries=/etc/prometheus/console_libraries \
		--web.console.templates=/etc/prometheus/consoles \
		--web.enable-lifecycle || echo "Prometheus already running"
	@echo "Prometheus available at http://localhost:9091"

## metrics-stop: Stop local Prometheus server
metrics-stop:
	@echo "Stopping Prometheus server..."
	@docker stop prometheus-dev || true
	@docker rm prometheus-dev || true

## grafana: Start local Grafana server for development
grafana:
	@echo "Starting Grafana server..."
	@docker run -d --name grafana-dev \
		-p 3000:3000 \
		-e "GF_SECURITY_ADMIN_PASSWORD=admin" \
		-v grafana-storage:/var/lib/grafana \
		grafana/grafana:10.0.0 || echo "Grafana already running"
	@echo "Grafana available at http://localhost:3000 (admin/admin)"

## grafana-stop: Stop local Grafana server
grafana-stop:
	@echo "Stopping Grafana server..."
	@docker stop grafana-dev || true
	@docker rm grafana-dev || true

## monitoring: Start full monitoring stack (Prometheus + Grafana)
monitoring: metrics grafana
	@echo "Monitoring stack started"
	@echo "Prometheus: http://localhost:9091"
	@echo "Grafana: http://localhost:3000 (admin/admin)"

## monitoring-stop: Stop full monitoring stack
monitoring-stop: metrics-stop grafana-stop
	@echo "Monitoring stack stopped"

## metrics-check: Check metrics endpoint
metrics-check:
	@echo "Checking metrics endpoint..."
	@curl -s http://localhost:9090/metrics | head -20 || echo "Metrics endpoint not available"

## dev-start: Start local development environment with Docker Compose
dev-start:
	@echo "Starting local development environment..."
	docker-compose up -d --build
	@echo "Waiting for agents to be ready..."
	@sleep 10
	@echo "Development environment ready! Use 'make run-tui-docker' to start TUI"
	@echo "Prometheus: http://localhost:9093"
	@echo "Grafana: http://localhost:3000 (admin/admin)"

## dev-stop: Stop local development environment
dev-stop:
	@echo "Stopping local development environment..."
	docker-compose down

## dev-restart: Restart local development environment
dev-restart: dev-stop dev-start

## dev-logs: Show logs from all development services
dev-logs:
	docker-compose logs -f

## dev-status: Show status of development services
dev-status:
	docker-compose ps

## dev-clean: Clean up development environment
dev-clean:
	@echo "Cleaning up development environment..."
	docker-compose down -v --remove-orphans
	docker system prune -f

## dev-gpu: Start development environment with GPU support
dev-gpu:
	@echo "Starting GPU-enabled development environment..."
	docker-compose --profile gpu up -d --build
	@echo "GPU development environment ready!"

## docs: Generate Hugo documentation site
docs:
	@echo "Generating Hugo documentation..."
	@if ! command -v hugo >/dev/null 2>&1; then \
		echo "Installing Hugo..."; \
		if command -v snap >/dev/null 2>&1; then \
			sudo snap install hugo; \
		else \
			echo "Please install Hugo: https://gohugo.io/installation/"; \
			exit 1; \
		fi; \
	fi
	@cd docs-site && hugo --minify --destination ../docs-output
	@echo "Documentation generated in docs-output/"
	@echo "To serve locally: make docs-serve"

## docs-serve: Serve documentation locally
docs-serve:
	@echo "Starting Hugo development server..."
	@if ! command -v hugo >/dev/null 2>&1; then \
		echo "Hugo not found. Run 'make docs' first."; \
		exit 1; \
	fi
	@cd docs-site && hugo server --bind 0.0.0.0 --port 1313 --disableFastRender
	@echo "Documentation server started at http://localhost:1313"

## docs-build: Build documentation for GitHub Pages
docs-build:
	@echo "Building documentation for GitHub Pages..."
	@cd docs-site && hugo --minify --baseURL "https://billy-davies-2.github.io/distributed-llm/" --destination ../docs
	@echo "Documentation built for GitHub Pages in docs/"

## docs-clean: Clean documentation build artifacts
docs-clean:
	@echo "Cleaning documentation artifacts..."
	@rm -rf docs-output/ docs-site/public/ docs/

## docs-dev: Start documentation development environment
docs-dev: docs-clean
	@echo "Setting up documentation development environment..."
	@cd docs-site && \
		if [ ! -d "themes/docsy" ]; then \
			echo "Installing Docsy theme..."; \
			git submodule add https://github.com/google/docsy.git themes/docsy || true; \
			git submodule update --init --recursive; \
		fi
	@make docs-serve
