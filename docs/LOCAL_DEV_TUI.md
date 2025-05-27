# Local Development Guide

This guide explains how to run the Distributed LLM system with the TUI running locally and agents running in Docker containers.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────────────────────────────────┐
│   Local TUI     │    │             Docker Environment               │
│                 │    │                                             │
│  ┌───────────┐  │    │  ┌─────────┐  ┌─────────┐  ┌─────────┐     │
│  │ Discovery │  │ ── │→ │ Agent-1 │  │ Agent-2 │  │ Agent-3 │     │
│  │  Client   │  │    │  │ :8080   │  │ :8081   │  │ :8082   │     │
│  └───────────┘  │    │  └─────────┘  └─────────┘  └─────────┘     │
│                 │    │       │           │           │             │
│  ┌───────────┐  │    │       ▼           ▼           ▼             │
│  │    TUI    │  │    │  ┌─────────────────────────────────────┐   │
│  │Interface  │  │    │  │        Prometheus :9093            │   │
│  └───────────┘  │    │  └─────────────────────────────────────┘   │
└─────────────────┘    │                     │                       │
                       │                     ▼                       │
                       │  ┌─────────────────────────────────────┐   │
                       │  │         Grafana :3000              │   │
                       │  └─────────────────────────────────────┘   │
                       └─────────────────────────────────────────────┘
```

## Quick Start

### 1. Prerequisites

Ensure you have the following installed:
- Docker and Docker Compose
- Go 1.21+
- Make

### 2. Start Development Environment

Use the automated setup script:

```bash
./hack/local-dev-tui.sh
```

Or manually:

```bash
# Start Docker containers
make dev-start

# Run TUI locally (in another terminal)
make run-tui-docker
```

### 3. Access Services

- **TUI**: Runs locally in your terminal
- **Agent APIs**: http://localhost:8080, 8081, 8082
- **Prometheus**: http://localhost:9093
- **Grafana**: http://localhost:3000 (admin/admin)

## Development Workflows

### Standard Development

```bash
# Start environment
make dev-start

# Run TUI in Docker discovery mode
make run-tui-docker

# View logs
make dev-logs

# Stop environment
make dev-stop
```

### Kubernetes Development

```bash
# Deploy to minikube
make minikube-start

# Run TUI connecting to K8s
make run-tui-k8s
```

### Manual Agent Connection

```bash
# Run specific agents locally
go run ./cmd/agent --node-id=local-1 --bind-port=8080
go run ./cmd/agent --node-id=local-2 --bind-port=8081 --seed-nodes=localhost:7946

# Connect TUI to specific agents
make run-tui-local
```

## Configuration Options

### TUI Connection Modes

The TUI supports multiple discovery modes:

#### Docker Mode (Recommended for Local Development)
```bash
go run ./cmd/tui --docker --log-level=debug
```

Automatically discovers agents in Docker containers using common service names.

#### Kubernetes Mode
```bash
go run ./cmd/tui --k8s-namespace=default --log-level=debug
```

Discovers agents through Kubernetes service discovery and DNS.

#### Manual Seed Nodes
```bash
go run ./cmd/tui --seed-nodes=localhost:8080,localhost:8081 --log-level=debug
```

Connects to specific agent addresses.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |
| `AGENT_COUNT` | Number of agents to start | 3 |

## Networking

### Docker Compose Network

The development environment creates a bridge network (`llm-network`) with subnet `172.20.0.0/16`:

- **agent-1**: `172.20.0.2:8080`
- **agent-2**: `172.20.0.3:8080`
- **agent-3**: `172.20.0.4:8080`
- **prometheus**: `172.20.0.5:9090`
- **grafana**: `172.20.0.6:3000`

### Port Mapping

| Service | Internal Port | External Port | Purpose |
|---------|---------------|---------------|---------|
| agent-1 | 8080 | 8080 | gRPC API |
| agent-1 | 7946 | 7946 | Gossip protocol |
| agent-1 | 9090 | 9090 | Prometheus metrics |
| agent-2 | 8080 | 8081 | gRPC API |
| agent-2 | 7946 | 7947 | Gossip protocol |
| agent-2 | 9090 | 9091 | Prometheus metrics |
| agent-3 | 8080 | 8082 | gRPC API |
| agent-3 | 7946 | 7948 | Gossip protocol |
| agent-3 | 9090 | 9092 | Prometheus metrics |
| prometheus | 9090 | 9093 | Web UI |
| grafana | 3000 | 3000 | Web UI |

## Service Discovery

### Docker Service Discovery

The TUI discovery client attempts to connect to:

1. **Named services**: `distributed-llm-agent`, `agent`, `llm-agent`
2. **Localhost ports**: `localhost:8080-8083`
3. **Health checks**: Verifies agent responsiveness before adding

### Kubernetes Service Discovery

For K8s environments, the TUI uses:

1. **Service DNS**: `agent.namespace.svc.cluster.local`
2. **SRV records**: DNS SRV lookup for service discovery
3. **Pod annotations**: Prometheus-style annotations for port discovery

## Monitoring and Observability

### Metrics Collection

Each agent exposes metrics on port 9090:
- Node resource metrics (CPU, memory, GPU)
- Network performance metrics
- Inference request metrics
- Health status

### Prometheus Configuration

The Prometheus configuration supports multiple deployment modes:
- **Docker**: Scrapes container services
- **Local**: Scrapes localhost ports
- **Kubernetes**: Service discovery via pod annotations

### Grafana Dashboards

Pre-configured dashboards show:
- Cluster overview with node status
- Resource utilization trends
- Network topology and performance
- Inference request patterns

## Troubleshooting

### Common Issues

#### TUI Can't Connect to Agents

```bash
# Check agent status
make dev-status

# View agent logs
docker logs distributed-llm-agent-1

# Test direct connection
curl http://localhost:8080/health
```

#### Agents Not Starting

```bash
# Check Docker logs
make dev-logs

# Restart environment
make dev-restart

# Clean and rebuild
make dev-clean
make dev-start
```

#### Metrics Not Available

```bash
# Check Prometheus targets
open http://localhost:9093/targets

# Verify agent metrics endpoint
curl http://localhost:9090/metrics
```

### Debug Mode

Enable debug logging for detailed information:

```bash
# TUI debug mode
go run ./cmd/tui --docker --log-level=debug

# Agent debug mode
LOG_LEVEL=debug make dev-start
```

### Network Issues

If you encounter network connectivity issues:

1. **Check Docker network**: `docker network ls`
2. **Inspect network**: `docker network inspect distributed-llm_llm-network`
3. **Test connectivity**: `docker exec distributed-llm-agent-1 ping agent-2`

## Advanced Configuration

### GPU Support

Enable GPU support for agents:

```bash
# Start with GPU profile
make dev-gpu

# This adds agent-gpu with NVIDIA runtime
```

### Custom Agent Configuration

Modify `docker-compose.yml` to adjust:
- Memory limits
- CPU allocation
- Environment variables
- Volume mounts

### Production-like Testing

For testing production deployment patterns:

```bash
# Use Kubernetes
make minikube-start
make run-tui-k8s

# Or use multiple Docker networks
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up
```

## Development Tips

1. **Keep TUI running**: The TUI will automatically reconnect to agents
2. **Monitor logs**: Use `make dev-logs` in a separate terminal
3. **Restart agents**: Individual agents can be restarted without affecting others
4. **Port conflicts**: Change port mappings in `docker-compose.yml` if needed
5. **Resource testing**: Modify agent resource reporting in discovery testing
