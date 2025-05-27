# Distributed LLM Cluster

A distributed system for running large language models across multiple nodes with intelligent layer distribution and gossip-based discovery.

## Features

- **Distributed Inference**: Distribute model layers across multiple nodes
- **Gossip Protocol**: Automatic node discovery and cluster formation
- **Resource Management**: Intelligent allocation based on CPU, memory, and GPU resources
- **Metrics & Monitoring**: Prometheus metrics with Grafana dashboards
- **Terminal UI**: Real-time cluster monitoring and management
- **Container Native**: Docker and Kubernetes support

## Quick Start

### Local Development with Docker

The recommended way to develop and test the system:

```bash
# Start development environment
make dev-start

# Run TUI locally connecting to Docker agents
make run-tui-docker
```

This starts 3 agent containers with full monitoring stack. See [Local Development Guide](docs/LOCAL_DEV_TUI.md) for details.

### Kubernetes Deployment

```bash
# Deploy to minikube
make minikube-start

# Run TUI locally connecting to K8s
make run-tui-k8s
```

### Manual Build

```bash
# Build binaries
make build

# Run agent
./bin/agent --node-id=node1

# Run TUI (in another terminal)
./bin/tui --seed-nodes=localhost:8080
```

## Documentation

- **[Local Development](docs/LOCAL_DEV_TUI.md)** - Complete guide for local development
- **[Overview](docs/overview.md)** - System architecture and design
- **[Metrics](docs/METRICS.md)** - Monitoring and observability
- **[Testing](docs/TEST_CONFIG.md)** - Testing strategy and configuration

## Services

| Service | Port | Purpose |
|---------|------|---------|
| Agent gRPC | 8080 | Node communication |
| Agent Gossip | 7946 | Cluster discovery |
| Agent Metrics | 9090 | Prometheus metrics |
| Prometheus | 9093 | Metrics collection |
| Grafana | 3000 | Visualization |

## Development Commands

```bash
# Start development environment
make dev-start              # Start Docker containers
make dev-stop               # Stop containers  
make dev-restart            # Restart environment
make dev-logs               # View all logs
make dev-status             # Show service status
make dev-clean              # Clean up everything

# TUI connection modes
make run-tui-docker         # Connect to Docker agents
make run-tui-k8s            # Connect to Kubernetes agents  
make run-tui-local          # Connect to local agents

# Testing
make test                   # Run unit tests
make test-integration       # Run integration tests
make test-e2e               # Run end-to-end tests
```

Central hub for quick start and project overview. See [docs/overview.md](docs/overview.md) for detailed information.