#!/bin/bash

# Deploy monitoring stack for distributed LLM system
# This script deploys Prometheus, Grafana, and the distributed LLM agents

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prereqs() {
    print_status "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed"
        exit 1
    fi
    
    # Check if we can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check Docker for building images
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Create namespace
create_namespace() {
    print_status "Creating namespace..."
    kubectl create namespace distributed-llm --dry-run=client -o yaml | kubectl apply -f -
    print_success "Namespace created/updated"
}

# Build and push Docker image
build_image() {
    print_status "Building Docker image..."
    docker build -t distributed-llm/agent:latest .
    
    # If using minikube, load image into minikube
    if kubectl config current-context | grep -q minikube; then
        print_status "Loading image into minikube..."
        minikube image load distributed-llm/agent:latest
    elif kubectl config current-context | grep -q kind; then
        print_status "Loading image into kind..."
        kind load docker-image distributed-llm/agent:latest
    fi
    
    print_success "Docker image built and loaded"
}

# Deploy Prometheus
deploy_prometheus() {
    print_status "Deploying Prometheus..."
    kubectl apply -f deployments/prometheus/prometheus-k8s.yaml
    
    # Wait for Prometheus to be ready
    print_status "Waiting for Prometheus to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/prometheus -n distributed-llm
    
    print_success "Prometheus deployed and ready"
}

# Deploy Grafana
deploy_grafana() {
    print_status "Deploying Grafana..."
    
    # Create Grafana deployment if it doesn't exist
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: distributed-llm
  labels:
    app: grafana
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
        image: grafana/grafana:10.0.0
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          value: "admin"
        - name: GF_USERS_ALLOW_SIGN_UP
          value: "false"
        volumeMounts:
        - name: grafana-storage
          mountPath: /var/lib/grafana
        - name: grafana-config
          mountPath: /etc/grafana/provisioning/datasources
        - name: grafana-dashboards-config
          mountPath: /etc/grafana/provisioning/dashboards
        - name: grafana-dashboards
          mountPath: /var/lib/grafana/dashboards
      volumes:
      - name: grafana-storage
        emptyDir: {}
      - name: grafana-config
        configMap:
          name: grafana-datasources
      - name: grafana-dashboards-config
        configMap:
          name: grafana-dashboards-config
      - name: grafana-dashboards
        configMap:
          name: grafana-dashboards
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: distributed-llm
  labels:
    app: grafana
spec:
  type: NodePort
  ports:
  - port: 3000
    targetPort: 3000
    nodePort: 30300
  selector:
    app: grafana
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: distributed-llm
data:
  datasource.yml: |
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus:9090
      isDefault: true
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards-config
  namespace: distributed-llm
data:
  dashboards.yml: |
    apiVersion: 1
    providers:
    - name: 'default'
      orgId: 1
      folder: ''
      type: file
      disableDeletion: false
      updateIntervalSeconds: 10
      options:
        path: /var/lib/grafana/dashboards
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards
  namespace: distributed-llm
data:
  distributed-llm.json: |
$(cat deployments/grafana/dashboard.json | sed 's/^/    /')
EOF
    
    # Wait for Grafana to be ready
    print_status "Waiting for Grafana to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/grafana -n distributed-llm
    
    print_success "Grafana deployed and ready"
}

# Deploy agents
deploy_agents() {
    print_status "Deploying distributed LLM agents..."
    kubectl apply -f deployments/agent/agent-deployment.yaml
    
    # Wait for agents to be ready
    print_status "Waiting for agents to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/distributed-llm-agent -n distributed-llm
    
    print_success "Agents deployed and ready"
}

# Setup port forwarding
setup_port_forwarding() {
    print_status "Setting up port forwarding..."
    
    # Kill any existing port forwards
    pkill -f "kubectl port-forward" || true
    
    # Forward Prometheus
    kubectl port-forward -n distributed-llm svc/prometheus 9090:9090 &
    PROMETHEUS_PID=$!
    
    # Forward Grafana
    kubectl port-forward -n distributed-llm svc/grafana 3000:3000 &
    GRAFANA_PID=$!
    
    # Give services time to start
    sleep 5
    
    print_success "Port forwarding setup complete"
    echo "  Prometheus: http://localhost:9090"
    echo "  Grafana: http://localhost:3000 (admin/admin)"
}

# Show status
show_status() {
    print_status "Deployment status:"
    echo
    kubectl get all -n distributed-llm
    echo
    print_success "Deployment complete!"
    echo
    print_status "Access URLs:"
    echo "  Prometheus: http://localhost:9090"
    echo "  Grafana: http://localhost:3000 (admin/admin)"
    echo
    print_status "To check metrics:"
    echo "  kubectl get pods -n distributed-llm"
    echo "  kubectl logs -l app=distributed-llm-agent -n distributed-llm"
    echo "  curl http://localhost:9090/api/v1/query?query=up"
}

# Cleanup function
cleanup() {
    print_status "Cleaning up..."
    kubectl delete namespace distributed-llm --ignore-not-found=true
    pkill -f "kubectl port-forward" || true
    print_success "Cleanup complete"
}

# Main deployment function
deploy() {
    print_status "Starting deployment of distributed LLM monitoring stack..."
    
    check_prereqs
    create_namespace
    build_image
    deploy_prometheus
    deploy_grafana
    deploy_agents
    setup_port_forwarding
    show_status
}

# Parse command line arguments
case "${1:-deploy}" in
    deploy)
        deploy
        ;;
    cleanup)
        cleanup
        ;;
    status)
        show_status
        ;;
    *)
        echo "Usage: $0 [deploy|cleanup|status]"
        echo "  deploy  - Deploy the full monitoring stack (default)"
        echo "  cleanup - Remove all deployed resources"
        echo "  status  - Show current deployment status"
        exit 1
        ;;
esac
