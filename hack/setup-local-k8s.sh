#!/bin/bash
set -euo pipefail

# Local Development Setup for Distributed LLM with Kubernetes
# This script sets up a local Kind cluster for realistic testing

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="distributed-llm"
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="5001"

echo "🚀 Setting up local Kubernetes development environment..."

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    echo "📋 Checking prerequisites..."
    
    if ! command_exists docker; then
        echo "❌ Docker is required but not installed"
        exit 1
    fi
    
    if ! command_exists kind; then
        echo "📦 Installing kind..."
        # Install kind
        [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind
    fi
    
    if ! command_exists kubectl; then
        echo "📦 Installing kubectl..."
        curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/kubectl
    fi
    
    echo "✅ Prerequisites satisfied"
}

# Create kind cluster with registry
create_cluster() {
    echo "🏗️  Creating Kind cluster with local registry..."
    
    # Create registry if it doesn't exist
    if ! docker ps | grep -q $REGISTRY_NAME; then
        echo "📦 Creating local Docker registry..."
        docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2
    fi
    
    # Create kind cluster config
    cat << EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 30080
    hostPort: 30080
    protocol: TCP
  - containerPort: 30443
    hostPort: 30443
    protocol: TCP
- role: worker
  labels:
    node-type: "agent"
- role: worker
  labels:
    node-type: "agent"
- role: worker
  labels:
    node-type: "agent"
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
    endpoint = ["http://${REGISTRY_NAME}:5000"]
EOF

    # Delete existing cluster if it exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        echo "🗑️  Deleting existing cluster..."
        kind delete cluster --name "${CLUSTER_NAME}"
    fi
    
    # Create cluster
    echo "🎯 Creating Kind cluster..."
    kind create cluster --config=/tmp/kind-config.yaml
    
    # Connect registry to cluster network
    if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}")" = 'null' ]; then
        docker network connect "kind" "${REGISTRY_NAME}"
    fi
    
    # Document registry
    kubectl apply -f - << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

    echo "✅ Kind cluster created successfully"
}

# Build and load images
build_images() {
    echo "🔨 Building and loading Docker images..."
    
    cd "$PROJECT_ROOT"
    
    # Build main agent image
    echo "📦 Building agent image..."
    docker build -t localhost:${REGISTRY_PORT}/distributed-llm-agent:latest .
    docker push localhost:${REGISTRY_PORT}/distributed-llm-agent:latest
    
    # Build GPU image if GPU support is available
    if command_exists nvidia-smi; then
        echo "📦 Building GPU agent image..."
        docker build -f Dockerfile.gpu -t localhost:${REGISTRY_PORT}/distributed-llm-agent-gpu:latest .
        docker push localhost:${REGISTRY_PORT}/distributed-llm-agent-gpu:latest
    fi
    
    echo "✅ Images built and pushed to local registry"
}

# Deploy monitoring stack
deploy_monitoring() {
    echo "📊 Deploying monitoring stack..."
    
    # Create monitoring namespace
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    # Deploy Prometheus
    kubectl apply -f - << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus'
        - '--web.console.libraries=/etc/prometheus/console_libraries'
        - '--web.console.templates=/etc/prometheus/consoles'
        - '--storage.tsdb.retention.time=200h'
        - '--web.enable-lifecycle'
      volumes:
      - name: config
        configMap:
          name: prometheus-config
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: monitoring
spec:
  selector:
    app: prometheus
  ports:
  - port: 9090
    targetPort: 9090
    nodePort: 30090
  type: NodePort
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
    - job_name: 'kubernetes-pods'
      kubernetes_sd_configs:
      - role: pod
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: \$1:\$2
        target_label: __address__
    - job_name: 'distributed-llm-agents'
      static_configs:
      - targets: ['distributed-llm-agent.default.svc.cluster.local:9090']
      kubernetes_sd_configs:
      - role: service
        namespaces:
          names: ['default']
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_label_app]
        action: keep
        regex: distributed-llm-agent
EOF

    # Deploy Grafana
    kubectl apply -f - << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:latest
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          value: "admin"
        - name: GF_USERS_ALLOW_SIGN_UP
          value: "false"
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: monitoring
spec:
  selector:
    app: grafana
  ports:
  - port: 3000
    targetPort: 3000
    nodePort: 30030
  type: NodePort
EOF

    echo "✅ Monitoring stack deployed"
}

# Deploy distributed LLM agents
deploy_agents() {
    echo "🤖 Deploying distributed LLM agents..."
    
    # Create agent deployment
    kubectl apply -f - << EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: distributed-llm-agent
  labels:
    app: distributed-llm-agent
spec:
  selector:
    matchLabels:
      app: distributed-llm-agent
  template:
    metadata:
      labels:
        app: distributed-llm-agent
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: agent
        image: localhost:${REGISTRY_PORT}/distributed-llm-agent:latest
        ports:
        - containerPort: 8080
          name: grpc
        - containerPort: 7946
          name: gossip
        - containerPort: 9090
          name: metrics
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        command:
        - "/bin/agent"
        args:
        - "--node-id=\$(NODE_NAME)"
        - "--bind-port=8080"
        - "--gossip-port=7946"
        - "--metrics-port=9090"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 5
      nodeSelector:
        node-type: "agent"
---
apiVersion: v1
kind: Service
metadata:
  name: distributed-llm-agent
  labels:
    app: distributed-llm-agent
spec:
  selector:
    app: distributed-llm-agent
  ports:
  - name: grpc
    port: 8080
    targetPort: 8080
  - name: gossip
    port: 7946
    targetPort: 7946
  - name: metrics
    port: 9090
    targetPort: 9090
  type: ClusterIP
---
apiVersion: v1
kind: Service
metadata:
  name: distributed-llm-agent-nodeport
  labels:
    app: distributed-llm-agent
spec:
  selector:
    app: distributed-llm-agent
  ports:
  - name: grpc
    port: 8080
    targetPort: 8080
    nodePort: 30080
  - name: metrics
    port: 9090
    targetPort: 9090
    nodePort: 30090
  type: NodePort
EOF

    echo "✅ Agents deployed"
}

# Wait for deployments
wait_for_deployments() {
    echo "⏳ Waiting for deployments to be ready..."
    
    kubectl wait --for=condition=available --timeout=300s deployment/prometheus -n monitoring
    kubectl wait --for=condition=available --timeout=300s deployment/grafana -n monitoring
    kubectl wait --for=condition=ready --timeout=300s pod -l app=distributed-llm-agent
    
    echo "✅ All deployments ready"
}

# Display connection info
show_connection_info() {
    echo ""
    echo "🎉 Local development environment ready!"
    echo ""
    echo "📊 Services:"
    echo "  • Prometheus: http://localhost:30090"
    echo "  • Grafana: http://localhost:30030 (admin/admin)"
    echo "  • Agent gRPC: localhost:30080"
    echo ""
    echo "🔍 Useful commands:"
    echo "  • View pods: kubectl get pods -o wide"
    echo "  • View services: kubectl get svc"
    echo "  • View agent logs: kubectl logs -l app=distributed-llm-agent"
    echo "  • Port forward agent: kubectl port-forward svc/distributed-llm-agent 8080:8080"
    echo ""
    echo "🎮 Start TUI with:"
    echo "  make run-tui --docker"
    echo "  # or"
    echo "  ./bin/tui --seed-nodes=localhost:30080"
    echo ""
    echo "🧹 Cleanup with:"
    echo "  kind delete cluster --name ${CLUSTER_NAME}"
    echo "  docker stop ${REGISTRY_NAME} && docker rm ${REGISTRY_NAME}"
}

# Main execution
main() {
    check_prerequisites
    create_cluster
    build_images
    deploy_monitoring
    deploy_agents
    wait_for_deployments
    show_connection_info
}

# Handle cleanup on script exit
cleanup() {
    echo "🧹 Cleaning up temporary files..."
    rm -f /tmp/kind-config.yaml
}

trap cleanup EXIT

# Run main function
main "$@"
