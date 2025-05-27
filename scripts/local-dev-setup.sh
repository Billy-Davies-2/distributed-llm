#!/bin/bash

# Local Development Setup for Distributed LLM
# This script sets up kind cluster and builds local images

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running from project root
if [ ! -f "go.mod" ] || [ ! -f "Dockerfile" ]; then
    echo_error "Please run this script from the project root directory"
    exit 1
fi

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install kind if not present
install_kind() {
    if command_exists kind; then
        echo_info "kind is already installed"
        kind version
        return
    fi
    
    echo_info "Installing kind..."
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) echo_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    # Download and install kind
    curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-$ARCH
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/kind
    
    echo_info "kind installed successfully"
    kind version
}

# Install kubectl if not present
install_kubectl() {
    if command_exists kubectl; then
        echo_info "kubectl is already installed"
        kubectl version --client
        return
    fi
    
    echo_info "Installing kubectl..."
    
    # Download kubectl
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/kubectl
    
    echo_info "kubectl installed successfully"
    kubectl version --client
}

# Check Docker
check_docker() {
    if ! command_exists docker; then
        echo_error "Docker is required but not installed. Please install Docker first."
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        echo_error "Docker daemon is not running. Please start Docker."
        exit 1
    fi
    
    echo_info "Docker is running"
}

# Create kind cluster
create_cluster() {
    echo_info "Creating kind cluster..."
    
    # Create kind config
    cat > kind-config.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: distributed-llm
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
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
- role: worker
- role: worker
- role: worker
EOF
    
    # Delete existing cluster if it exists
    if kind get clusters | grep -q "distributed-llm"; then
        echo_warn "Deleting existing cluster..."
        kind delete cluster --name distributed-llm
    fi
    
    # Create cluster
    kind create cluster --config kind-config.yaml
    
    # Wait for cluster to be ready
    echo_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s
    
    echo_info "Cluster created successfully"
    kubectl get nodes
}

# Build and load Docker image
build_and_load_image() {
    echo_info "Building Docker image..."
    
    # Build the image
    docker build -t distributed-llm:latest .
    
    # Load image into kind cluster
    echo_info "Loading image into kind cluster..."
    kind load docker-image distributed-llm:latest --name distributed-llm
    
    echo_info "Image loaded successfully"
}

# Deploy to cluster
deploy_to_cluster() {
    echo_info "Deploying to cluster..."
    
    # Apply deployments
    kubectl apply -f deployments/
    
    # Wait for deployments to be ready
    echo_info "Waiting for deployments to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment --all 2>/dev/null || true
    
    echo_info "Deployment complete"
    kubectl get pods -o wide
}

# Show cluster info
show_cluster_info() {
    echo_info "Cluster Information:"
    echo "===================="
    echo "Cluster: $(kubectl config current-context)"
    echo "Nodes:"
    kubectl get nodes
    echo
    echo "Pods:"
    kubectl get pods -o wide
    echo
    echo "Services:"
    kubectl get services
    echo
    echo_info "To interact with the cluster:"
    echo "  kubectl get pods"
    echo "  kubectl logs <pod-name>"
    echo "  kubectl exec -it <pod-name> -- /bin/sh"
    echo
    echo_info "To run TUI locally:"
    echo "  make run-tui"
    echo
    echo_info "To rebuild and redeploy:"
    echo "  make docker-build && kind load docker-image distributed-llm:latest --name distributed-llm && kubectl rollout restart deployment"
    echo
    echo_info "To delete cluster:"
    echo "  kind delete cluster --name distributed-llm"
}

# Main execution
main() {
    echo_info "Starting local development setup..."
    
    check_docker
    install_kind
    install_kubectl
    create_cluster
    build_and_load_image
    deploy_to_cluster
    show_cluster_info
    
    echo_info "Setup complete! 🎉"
}

# Run main function
main "$@"
