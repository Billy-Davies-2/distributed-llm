#!/bin/bash
set -euo pipefail

# Local Development Setup for Distributed LLM with k3d
# This script sets up a local k3d cluster for realistic testing (lighter alternative to kind)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="distributed-llm"
REGISTRY_NAME="k3d-registry.localhost"
REGISTRY_PORT="5000"

echo "🚀 Setting up local k3d development environment..."

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
    
    if ! command_exists k3d; then
        echo "📦 Installing k3d..."
        curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    fi
    
    if ! command_exists kubectl; then
        echo "📦 Installing kubectl..."
        curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/kubectl
    fi
    
    echo "✅ Prerequisites satisfied"
}

# Create k3d cluster with registry
create_cluster() {
    echo "🏗️  Creating k3d cluster with local registry..."
    
    # Delete existing cluster if it exists
    if k3d cluster list | grep -q "^${CLUSTER_NAME}"; then
        echo "🗑️  Deleting existing cluster..."
        k3d cluster delete "${CLUSTER_NAME}"
    fi
    
    # Create registry
    k3d registry create ${REGISTRY_NAME} --port ${REGISTRY_PORT} || true
    
    # Create cluster with registry and port mappings
    k3d cluster create ${CLUSTER_NAME} \
        --registry-use k3d-${REGISTRY_NAME}:${REGISTRY_PORT} \
        --port "8080:30080@loadbalancer" \
        --port "9090:30090@loadbalancer" \
        --port "3000:30030@loadbalancer" \
        --agents 3 \
        --agents-memory 512m \
        --servers-memory 512m \
        --wait
    
    echo "✅ k3d cluster created successfully"
}

# Build and load images
build_images() {
    echo "🔨 Building and loading Docker images..."
    
    cd "$PROJECT_ROOT"
    
    # Build main agent image
    echo "📦 Building agent image..."
    docker build -t k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/distributed-llm-agent:latest .
    docker push k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/distributed-llm-agent:latest
    
    # Build GPU image if GPU support is available
    if command_exists nvidia-smi; then
        echo "📦 Building GPU agent image..."
        docker build -f Dockerfile.gpu -t k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/distributed-llm-agent-gpu:latest .
        docker push k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/distributed-llm-agent-gpu:latest
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
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
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
      evaluation_interval: 15s
    rule_files:
      # - "first_rules.yml"
      # - "second_rules.yml"
    scrape_configs:
    - job_name: 'prometheus'
      static_configs:
      - targets: ['localhost:9090']
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
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names: ['default']
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_label_app]
        action: keep
        regex: distributed-llm-agent
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics
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
        - name: GF_INSTALL_PLUGINS
          value: "grafana-piechart-panel"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        volumeMounts:
        - name: grafana-storage
          mountPath: /var/lib/grafana
      volumes:
      - name: grafana-storage
        emptyDir: {}
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
        image: k3d-${REGISTRY_NAME}:${REGISTRY_PORT}/distributed-llm-agent:latest
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
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        command:
        - "/bin/agent"
        args:
        - "--node-id=\$(NODE_NAME)-\$(POD_NAME)"
        - "--bind-port=8080"
        - "--gossip-port=7946"
        - "--metrics-port=9090"
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 256Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
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
    
    # Wait for at least one agent pod to be ready
    kubectl wait --for=condition=ready --timeout=300s pod -l app=distributed-llm-agent --field-selector=status.phase=Running | head -1
    
    echo "✅ All deployments ready"
}

# Display connection info
show_connection_info() {
    echo ""
    echo "🎉 Local k3d development environment ready!"
    echo ""
    echo "📊 Services (accessible via k3d load balancer):"
    echo "  • Prometheus: http://localhost:9090"
    echo "  • Grafana: http://localhost:3000 (admin/admin)"
    echo "  • Agent gRPC: localhost:8080"
    echo ""
    echo "🔍 Useful commands:"
    echo "  • View cluster: k3d cluster list"
    echo "  • View pods: kubectl get pods -o wide"
    echo "  • View services: kubectl get svc"
    echo "  • View agent logs: kubectl logs -l app=distributed-llm-agent -f"
    echo "  • Scale agents: kubectl patch daemonset distributed-llm-agent -p '{\"spec\":{\"template\":{\"spec\":{\"nodeSelector\":{\"node-role.kubernetes.io/agent\":\"true\"}}}}}'"
    echo ""
    echo "🎮 Start TUI with:"
    echo "  make run-tui-k3d"
    echo "  # or"
    echo "  ./bin/tui --seed-nodes=localhost:8080 --docker"
    echo ""
    echo "🧹 Cleanup with:"
    echo "  k3d cluster delete ${CLUSTER_NAME}"
    echo "  k3d registry delete ${REGISTRY_NAME}"
}

# Handle errors
handle_error() {
    echo "❌ Error occurred. Cleaning up..."
    k3d cluster delete "${CLUSTER_NAME}" 2>/dev/null || true
    exit 1
}

# Main execution
main() {
    trap handle_error ERR
    
    check_prerequisites
    create_cluster
    build_images
    deploy_monitoring
    deploy_agents
    wait_for_deployments
    show_connection_info
}

# Run main function
main "$@"
