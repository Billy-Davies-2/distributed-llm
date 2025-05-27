---
title: "Distributed LLM Documentation"
linkTitle: "Home"
type: docs
menu:
  main:
    weight: 20
---

# Distributed LLM

A high-performance, distributed system for running Large Language Models across multiple nodes with advanced monitoring, autoscaling, and real-time coordination.

## 🚀 Features

- **Distributed Architecture**: Run LLMs across multiple nodes for scalability
- **Real-time Monitoring**: Comprehensive Prometheus metrics and Grafana dashboards
- **Auto-scaling**: VPA and KEDA integration for intelligent resource management
- **Modern TUI**: Beautiful terminal interface with retro glitch effects
- **Kubernetes Native**: Built for cloud-native deployments
- **Protocol Buffers**: High-performance gRPC communication with compression
- **P2P Discovery**: Automatic node discovery and cluster formation

## 🏃 Quick Start

### Prerequisites

- Go 1.21+
- Docker
- Kubernetes cluster (kind/k3d/minikube)
- Protocol Buffers compiler

### Build and Run Locally

```bash
# Clone the repository
git clone https://github.com/Billy-Davies-2/distributed-llm.git
cd distributed-llm

# Build the project
make build

# Start local development environment
make dev-start

# Run the TUI client
make run-tui-docker
```

### Deploy to Kubernetes

```bash
# Setup k3d cluster
make setup-k3d

# Deploy monitoring stack
kubectl apply -f deployments/prometheus/
kubectl apply -f deployments/autoscaling/

# Deploy agents
kubectl apply -f deployments/agent/
```

## 📊 Architecture

The system consists of several key components:

- **Agent Nodes**: Compute nodes that host LLM layers
- **TUI Client**: Terminal interface for cluster management
- **Discovery Service**: Node discovery and cluster coordination
- **Metrics System**: Prometheus monitoring with Grafana visualization
- **Autoscaling**: VPA for vertical scaling, KEDA for horizontal scaling

## 📖 Documentation Sections

- [**Getting Started**](./getting-started/): Installation and basic usage
- [**Architecture**](./architecture/): System design and components
- [**Deployment**](./deployment/): Kubernetes deployment guides
- [**Monitoring**](./monitoring/): Metrics and observability
- [**API Reference**](./api/): gRPC API documentation
- [**Development**](./development/): Contributing and development setup

## 🔧 Development

See our [development guide](./development/) for information on:

- Setting up the development environment
- Running tests and benchmarks
- Contributing guidelines
- Code coverage and quality standards

## 📈 Monitoring

The system provides comprehensive monitoring through:

- **Prometheus Metrics**: Node health, resource usage, performance metrics
- **Grafana Dashboards**: Visual monitoring and alerting
- **Auto-scaling**: Automatic resource adjustment based on load

## 🏗️ Built With

- **Go**: Core application language
- **gRPC & Protocol Buffers**: High-performance communication
- **Kubernetes**: Container orchestration
- **Prometheus & Grafana**: Monitoring and visualization
- **Bubble Tea**: Terminal UI framework
- **Docker**: Containerization

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/Billy-Davies-2/distributed-llm/blob/main/LICENSE) file for details.
