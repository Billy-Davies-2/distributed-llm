# Local Development with Retro TUI and Kubernetes

This guide explains how to develop and test the Distributed LLM system with a retro terminal interface running locally and agents running in a realistic Kubernetes environment.

## Architecture Overview

```
┌─────────────────┐    ┌────────────────────────────────────┐
│  Local TUI      │    │     Kubernetes Cluster            │
│  (Your Machine) │    │                                    │
│                 │    │  ┌───────────┐ ┌───────────────┐   │
│  ┌───────────┐  │    │  │  Agent 1  │ │  Prometheus  │   │
│  │ Retro TUI │  │ gRPC │  │           │ │              │   │
│  │ + Glitch  │◄─┼────┼─►│  Agent 2  │ │  Grafana     │   │
│  │ Effects   │  │    │  │           │ │              │   │
│  └───────────┘  │    │  │  Agent 3  │ └───────────────┘   │
└─────────────────┘    │  └───────────┘                     │
                       └────────────────────────────────────┘
```

## Quick Start

### Option 1: k3d (Recommended - Lightweight)

```bash
# 1. Setup k3d cluster with agents
make setup-k3d

# 2. Run TUI locally (in another terminal)
make run-tui-k3d

# 3. Check status
make k3d-status

# 4. View logs
make k3d-logs

# 5. Cleanup when done
make k3d-cleanup
```

### Option 2: kind (Full Kubernetes Experience)

```bash
# 1. Setup kind cluster with agents  
make setup-kind

# 2. Run TUI locally
make run-tui-k8s

# 3. Check status
kubectl get pods -o wide

# 4. Cleanup
kind delete cluster --name distributed-llm
```

## TUI Features and Controls

### Retro Terminal Effects

The TUI includes authentic retro terminal glitches and effects:

- **Random glitches**: Text corruption, static lines, scanlines
- **Screen flicker**: Occasional screen-wide interference
- **Character corruption**: Random characters replaced with terminal symbols
- **Color inversion**: Sections with inverted colors
- **Scan lines**: Horizontal interference patterns

### Controls

| Key | Action |
|-----|--------|
| `Tab` | Switch between tabs (Nodes/Models/Inference) |
| `↑↓` | Navigate through items |
| `g` | Toggle glitch effects on/off |
| `G` | Trigger intense glitch burst |
| `1-5` | Set glitch intensity (1=low, 5=high) |
| `q` | Quit |

### Connection Modes

The TUI can connect to agents in different ways:

```bash
# Connect to k3d cluster
./bin/tui --seed-nodes=localhost:8080 --docker

# Connect to Kubernetes cluster
./bin/tui --k8s-namespace=default

# Connect to specific seed nodes
./bin/tui --seed-nodes=192.168.1.100:8080,192.168.1.101:8080

# Debug mode with verbose logging
./bin/tui --seed-nodes=localhost:8080 --log-level=debug
```

## Development Workflow

### 1. Initial Setup

```bash
# Clone and build
git clone <repo>
cd distributed-llm
make build

# Choose your cluster type
make setup-k3d  # or make setup-kind
```

### 2. Development Cycle

```bash
# Make code changes...

# Rebuild images and deploy
make docker-build
./hack/setup-local-k3d.sh  # rebuilds and redeploys

# Test with TUI
make run-tui-k3d

# View logs
make k3d-logs
```

### 3. Testing Different Scenarios

```bash
# Test with multiple agents
kubectl scale daemonset distributed-llm-agent --replicas=5

# Test agent failure
kubectl delete pod -l app=distributed-llm-agent | head -1

# Test network partitions
kubectl patch daemonset distributed-llm-agent -p '{"spec":{"template":{"spec":{"nodeSelector":{"kubernetes.io/arch":"amd64"}}}}}'
```

## Monitoring and Debugging

### Prometheus Metrics

Access metrics at:
- k3d: http://localhost:9090
- kind: http://localhost:30090

Key metrics to watch:
- `llm_nodes_total`: Number of discovered nodes
- `llm_network_messages_total`: Network communication
- `llm_inference_requests_total`: Inference requests
- `llm_node_resources_*`: Resource utilization

### Grafana Dashboards

Access Grafana at:
- k3d: http://localhost:3000
- kind: http://localhost:30030

Login: admin/admin

### Logs

```bash
# All agent logs
kubectl logs -l app=distributed-llm-agent -f

# Specific pod logs
kubectl logs distributed-llm-agent-xxxxx -f

# Previous container logs (if restarted)
kubectl logs distributed-llm-agent-xxxxx -f --previous
```

## Troubleshooting

### TUI Connection Issues

```bash
# Check if agents are accessible
curl http://localhost:8080/health  # k3d
curl http://localhost:30080/health # kind

# Test gRPC connection
grpcurl -plaintext localhost:8080 list

# Check DNS resolution
nslookup distributed-llm-agent.default.svc.cluster.local
```

### Agent Issues

```bash
# Check pod status
kubectl get pods -o wide

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp

# Check resource usage
kubectl top pods
kubectl top nodes
```

### Cluster Issues

```bash
# k3d cluster info
k3d cluster list
k3d node list

# kind cluster info  
kind get clusters
kind get nodes --name distributed-llm
```

## Performance Tuning

### Resource Limits

Adjust resource limits in the deployment:

```yaml
resources:
  requests:
    cpu: 100m      # Minimum CPU
    memory: 128Mi  # Minimum memory
  limits:
    cpu: 500m      # Maximum CPU
    memory: 512Mi  # Maximum memory
```

### Cluster Sizing

For k3d:
```bash
# Create cluster with more agents
k3d cluster create distributed-llm --agents 5 --agents-memory 1g
```

For kind:
```bash
# Edit /tmp/kind-config.yaml to add more worker nodes
```

### TUI Performance

```bash
# Reduce glitch intensity for better performance
./bin/tui --seed-nodes=localhost:8080 # Press '1' for low intensity

# Disable glitches entirely
./bin/tui --seed-nodes=localhost:8080 # Press 'g' to toggle off
```

## Advanced Usage

### Custom Agent Configuration

```bash
# Deploy with custom configuration
kubectl set env daemonset/distributed-llm-agent LOG_LEVEL=debug
kubectl set env daemonset/distributed-llm-agent METRICS_INTERVAL=5s
```

### Network Policies

```bash
# Test network isolation
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: agent-isolation
spec:
  podSelector:
    matchLabels:
      app: distributed-llm-agent
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from: []
  egress:
  - to: []
EOF
```

### Load Testing

```bash
# Use the TUI to generate inference requests
# Or use grpcurl for direct testing
for i in {1..100}; do
  grpcurl -plaintext localhost:8080 \
    -d '{"node_id":"test"}' \
    distributed_llm.NodeService/GetResources &
done
```

## Cleanup

### Full Cleanup

```bash
# k3d
make k3d-cleanup

# kind
kind delete cluster --name distributed-llm
docker stop kind-registry && docker rm kind-registry

# Remove images
docker rmi $(docker images "*distributed-llm*" -q)
```

### Partial Cleanup

```bash
# Just restart agents
kubectl rollout restart daemonset/distributed-llm-agent

# Reset monitoring
kubectl delete namespace monitoring
```

This setup provides a realistic testing environment that closely mirrors production Kubernetes deployments while keeping the TUI running locally for fast development iteration.
