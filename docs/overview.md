# DOS-LLVM: Distributed Large Language Model Cluster

```
██████╗  ██████╗ ███████╗      ██╗     ██╗     ██╗   ██╗███╗   ███╗
██╔══██╗██╔═══██╗██╔════╝      ██║     ██║     ██║   ██║████╗ ████║
██║  ██║██║   ██║███████╗█████╗██║     ██║     ██║   ██║██╔████╔██║
██║  ██║██║   ██║╚════██║╚════╝██║     ██║     ╚██╗ ██╔╝██║╚██╔╝██║
██████╔╝╚██████╔╝███████║      ███████╗███████╗ ╚████╔╝ ██║ ╚═╝ ██║
╚═════╝  ╚═════╝ ╚══════╝      ╚══════╝╚══════╝  ╚═══╝  ╚═╝     ╚═╝
                    
              Distributed Large Language Model Cluster v1.0
```

**🎨 Featuring a Retro MS-DOS Aesthetic with Cool-Retro-Term Green Theme!**

Heavily inspired by [exo](https://github.com/exo-explore/exo)

A distributed P2P network for running Large Language Models across multiple nodes using Kubernetes. This project enables efficient distributed inference by splitting LLM layers across cluster nodes, all wrapped in a beautiful retro terminal interface reminiscent of classic computing.

## ✨ Features

- **Retro MS-DOS Interface**: Beautiful green-on-black terminal aesthetic inspired by cool-retro-term
- **ASCII Art UI**: Classic terminal graphics with box drawing characters and retro styling
- **P2P Network**: Uses HashiCorp Memberlist for node discovery and gossip protocol
- **gRPC Communication**: High-performance communication between nodes using Protocol Buffers
- **Kubernetes Native**: Runs as a DaemonSet for automatic scaling across cluster nodes
- **Interactive TUI**: Bubble Tea-powered terminal interface with vintage computer vibes
- **Real-time Monitoring**: Live CPU, RAM, and GPU status with retro progress bars
- **Layer Distribution**: Intelligent distribution of LLM layers based on node capabilities

## 🎮 Quick Start with Retro Demo

```bash
# Launch the retro demonstration interface
./hack/retro-demo.sh
```

This will present you with a classic DOS-style menu system to explore all features!

## Architecture

### Components

- **Agent**: Runs on each Kubernetes node, handles resource monitoring and LLM inference
- **TUI Client**: Interactive terminal interface for cluster management and inference
- **P2P Network**: Memberlist-based gossip protocol for node discovery
- **gRPC Services**: High-performance communication between nodes

### Communication Flow

1. Agents join the P2P network using memberlist gossip protocol
2. Nodes broadcast their resources (CPU, RAM, GPU) to the network
3. TUI client connects to the network to monitor and manage the cluster
4. Models are uploaded to shared storage (ReadWriteMany PVC)
5. Inference requests are distributed across nodes based on layer assignments

## Prerequisites

- Go 1.24.3+
- Kubernetes cluster (minikube, k3d, or production cluster)
- Docker for building container images
- kubectl configured for your cluster

## Quick Start

### Local Development

1. **Clone and setup**:
   ```bash
   git clone <repository-url>
   cd distributed-llm
   go mod tidy
   ```

2. **Build the project**:
   ```bash
   make build
   ```

3. **Run locally** (for development):
   ```bash
   # Terminal 1 - Start first agent
   ./bin/agent --node-id=node1 --bind-port=8080 --gossip-port=7946
   
   # Terminal 2 - Start second agent (joins first)
   ./bin/agent --node-id=node2 --bind-port=8081 --gossip-port=7947 --seed-nodes=localhost:7946
   
   # Terminal 3 - Start TUI client
   ./bin/tui
   ```

### Running with Minikube

1. **Start minikube**:
   ```bash
   make minikube-start
   ```

2. **Build and deploy**:
   ```bash
   # Build Docker image in minikube
   eval $(minikube docker-env)
   docker build -t distributed-llm:latest .
   
   # Deploy to minikube
   kubectl apply -f deployments/
   ```

3. **Monitor the deployment**:
   ```bash
   kubectl get pods -n distributed-llm
   kubectl logs -f daemonset/distributed-llm-agent -n distributed-llm
   ```

4. **Access the TUI**:
   ```bash
   kubectl port-forward svc/distributed-llm-api 8080:8080 -n distributed-llm
   # In another terminal
   ./bin/tui
   ```

### Running with K3d

1. **Start k3d cluster**:
   ```bash
   make k3d-start
   ```

2. **Load image and deploy**:
   ```bash
   # Build and load image
   docker build -t distributed-llm:latest .
   k3d image import distributed-llm:latest --cluster distributed-llm
   
   # Deploy
   kubectl apply -f deployments/
   ```

3. **Access services**:
   ```bash
   kubectl port-forward svc/distributed-llm-api 8080:8080 -n distributed-llm
   ```

## Configuration

### Environment Variables

- `NODE_ID`: Unique identifier for the node (defaults to hostname)
- `BIND_PORT`: gRPC server port (default: 8080)
- `GOSSIP_PORT`: Memberlist gossip port (default: 7946)
- `SEED_NODES`: Comma-separated list of initial nodes to join

### Kubernetes Configuration

The deployment includes:
- **Namespace**: `distributed-llm`
- **DaemonSet**: Ensures one agent per node
- **ServiceAccount**: RBAC permissions for cluster resources
- **PVC**: Shared storage for models (requires ReadWriteMany)
- **Services**: ClusterIP for internal communication, LoadBalancer for external access

## Development

### Project Structure

```
distributed-llm/
├── cmd/                    # Application entrypoints
│   ├── agent/             # Agent main
│   └── tui/               # TUI main
├── internal/              # Private application code
│   ├── agent/             # Agent-specific logic
│   ├── k8s/               # Kubernetes client code
│   ├── network/           # P2P networking
│   └── tui/               # TUI implementation
├── pkg/                   # Public API
│   ├── config/            # Configuration
│   ├── models/            # Data models
│   └── proto/             # Generated protobuf code
├── proto/                 # Protobuf definitions
├── deployments/           # Kubernetes manifests
└── hack/               # Utility and hack scripts (moved files from scripts/)
```

### Building

```bash
# Generate protobuf code
make proto

# Build binaries
make build

# Run tests
make test

# Format code
make fmt

# Lint code
make lint
```

### Docker

```bash
# Build image
make docker-build

# For minikube development
eval $(minikube docker-env)
docker build -t distributed-llm:latest .
```

## Usage

### TUI Interface

The TUI provides three main tabs:

1. **Nodes**: View all connected nodes, their resources, and status
2. **Models**: Manage uploaded models and view their distribution
3. **Inference**: Submit inference requests and view results

### Key Bindings

- `Tab`: Switch between tabs
- `↑/↓` or `k/j`: Navigate lists
- `q` or `Ctrl+C`: Quit

### API Examples

```bash
# Health check
curl http://localhost:8080/health

# Get node resources
grpcurl -plaintext localhost:8080 distributed_llm.NodeService/GetResources
```

## Troubleshooting

### Common Issues

1. **Pods not starting**: Check if NFS storage class is available for ReadWriteMany PVC
2. **Network connectivity**: Ensure gossip ports (7946) are accessible between nodes
3. **Resource limits**: Adjust memory/CPU limits based on your cluster capacity

### Logs

```bash
# Agent logs
kubectl logs -f daemonset/distributed-llm-agent -n distributed-llm

# All pods in namespace
kubectl logs -f -l app=distributed-llm-agent -n distributed-llm
```

### Debugging

```bash
# Port forward for direct access
kubectl port-forward pod/<agent-pod> 8080:8080 -n distributed-llm

# Exec into pod
kubectl exec -it <agent-pod> -n distributed-llm -- /bin/sh
```

## Roadmap

- [ ] Model upload via TUI
- [ ] Automatic layer distribution optimization
- [ ] llama.cpp integration
- [ ] GPU utilization and allocation
- [ ] Model quantization support
- [ ] Inference performance metrics
- [ ] Multi-model support
- [ ] Web UI alternative to TUI

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.