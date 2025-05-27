---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 10
description: >
  Quick start guide for setting up and running Distributed LLM
---

## Prerequisites

Before getting started, ensure you have the following installed:

- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **Kubernetes cluster**: One of the following:
  - [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
  - [k3d](https://k3d.io/v5.4.6/#installation)
  - [minikube](https://minikube.sigs.k8s.io/docs/start/)
- **Protocol Buffers compiler**: `sudo apt install protobuf-compiler` or `brew install protobuf`

## Installation

### Option 1: Local Development

```bash
# Clone the repository
git clone https://github.com/Billy-Davies-2/distributed-llm.git
cd distributed-llm

# Install dependencies and build
make deps
make build

# Generate protobuf code
make proto
```

### Option 2: Docker Development Environment

```bash
# Start the full development environment
make dev-start

# This will start:
# - Multiple agent nodes
# - Prometheus monitoring
# - Grafana dashboards
```

### Option 3: Kubernetes Deployment

```bash
# Setup a local k3d cluster
make setup-k3d

# Deploy the application
kubectl apply -f deployments/
```

## Running the System

### Local Agents

Start individual agent nodes locally:

```bash
# Terminal 1: Start first agent
make run-agent

# Terminal 2: Start second agent with different ports
go run ./cmd/agent --node-id=agent-2 --bind-port=8081 --gossip-port=7947 --metrics-port=9091

# Terminal 3: Start TUI client
make run-tui-local
```

### Docker Environment

```bash
# Start Docker environment
make dev-start

# Connect TUI to Docker agents
make run-tui-docker
```

### Kubernetes Environment

```bash
# Deploy to k3d cluster
make k3d-start

# Connect TUI to Kubernetes cluster
make run-tui-k8s
```

## Using the TUI

The Terminal User Interface provides a retro-style interface for managing your distributed LLM cluster:

- **Nodes Tab**: View cluster nodes, their status, and resources
- **Models Tab**: Manage LLM models and their distribution
- **Inference Tab**: Run inference requests across the cluster

### Navigation

- `Tab` / `Shift+Tab`: Switch between tabs
- `↑` / `↓`: Navigate lists
- `Enter`: Select/activate items
- `q`: Quit application

## Verification

### Check Node Health

```bash
# View metrics from any agent
curl http://localhost:9090/metrics

# Check cluster status
kubectl get pods -l app=distributed-llm-agent
```

### Monitor Performance

Access monitoring interfaces:

- **Prometheus**: http://localhost:9091 (Docker) or port-forward for K8s
- **Grafana**: http://localhost:3000 (admin/admin)

### Test gRPC Communication

```bash
# Test node discovery
grpcurl -plaintext localhost:8080 distributed_llm.NodeService/GetNodeInfo

# Test compressed communication
grpcurl -H 'grpc-encoding: gzip' -plaintext localhost:8080 distributed_llm.DiscoveryService/DiscoverNodes
```

## Next Steps

- [Architecture Overview](../architecture/) - Understanding the system design
- [Deployment Guide](../deployment/) - Production deployment strategies
- [Monitoring Setup](../monitoring/) - Comprehensive monitoring configuration
- [Development Guide](../development/) - Contributing to the project
