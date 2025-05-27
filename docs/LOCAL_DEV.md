# Local Development Guide

This guide will help you set up a local Kubernetes cluster using kind to test the distributed LLM project.

## Prerequisites

- Docker installed and running
- Go 1.24.3+ installed
- Internet connection (for downloading kind and kubectl)

## Quick Start

### Option 1: Automated Setup (Recommended)

Run the automated setup script:

```bash
./scripts/local-dev-setup.sh
```

This script will:
1. Install kind and kubectl if not present
2. Create a 4-node kind cluster
3. Build the Docker image locally
4. Load the image into the cluster
5. Deploy the application

### Option 2: Manual Setup

1. **Install kind:**
   ```bash
   # For Linux
   curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
   chmod +x ./kind
   sudo mv ./kind /usr/local/bin/kind
   ```

2. **Create kind cluster:**
   ```bash
   make kind-start
   ```

3. **Build and deploy:**
   ```bash
   make docker-build
   kind load docker-image distributed-llm:latest --name distributed-llm
   kubectl apply -f deployments/
   ```

## Development Workflow

### Building and Testing

1. **Build locally:**
   ```bash
   make build
   ```

2. **Run TUI locally:**
   ```bash
   make run-tui
   ```

3. **Run agent locally:**
   ```bash
   make run-agent
   ```

### Kubernetes Development

1. **Rebuild and redeploy:**
   ```bash
   make kind-reload
   ```

2. **View logs:**
   ```bash
   make kind-logs
   ```

3. **Check cluster status:**
   ```bash
   kubectl get pods -o wide
   kubectl get nodes
   ```

### Debugging

1. **Get pod logs:**
   ```bash
   kubectl logs -l app=distributed-llm-agent --tail=100
   ```

2. **Exec into pod:**
   ```bash
   kubectl exec -it <pod-name> -- /bin/sh
   ```

3. **Port forward to access locally:**
   ```bash
   kubectl port-forward service/distributed-llm-api 8080:8080
   ```

### Testing the Retro TUI

1. **Run TUI locally (connects to local agents):**
   ```bash
   ./bin/tui
   ```

2. **Test different tabs:**
   - Press `TAB` to switch between tabs
   - Use `↑/↓` to navigate nodes
   - Press `q` to quit

## Cluster Architecture

The kind cluster consists of:
- 1 control-plane node
- 3 worker nodes
- DaemonSet runs one agent per node
- Headless service for peer discovery
- LoadBalancer service for external access

## Troubleshooting

### Common Issues

1. **Image not found:**
   ```bash
   # Rebuild and reload image
   make kind-reload
   ```

2. **Pods not starting:**
   ```bash
   # Check events
   kubectl describe pods
   kubectl get events --sort-by=.metadata.creationTimestamp
   ```

3. **Network issues:**
   ```bash
   # Check services
   kubectl get svc
   kubectl describe svc distributed-llm-agent
   ```

### Clean Up

1. **Delete cluster:**
   ```bash
   make kind-stop
   ```

2. **Clean build artifacts:**
   ```bash
   make clean
   ```

## Configuration

The application can be configured via:
- Command-line flags (see `--help`)
- Environment variables
- ConfigMap in `deployments/service.yaml`

## Next Steps

1. Upload LLM models to test distributed inference
2. Scale the cluster by adding more worker nodes
3. Monitor resource usage and layer distribution
4. Test fault tolerance by stopping nodes

## Resources

- [kind Documentation](https://kind.sigs.k8s.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [gRPC Documentation](https://grpc.io/docs/)
