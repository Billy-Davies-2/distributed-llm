#!/bin/bash

# Local Development Setup for Distributed LLM using Minikube
# This script sets up minikube cluster and builds local images

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

# Install minikube if not present
install_minikube() {
    if command_exists minikube; then
        echo_info "minikube is already installed"
        minikube version
        return
    fi
    
    echo_info "Installing minikube..."
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) echo_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    # Download and install minikube
    curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-$ARCH
    chmod +x minikube-linux-$ARCH
    sudo mv minikube-linux-$ARCH /usr/local/bin/minikube
    
    echo_info "minikube installed successfully"
    minikube version
}

# Install kubectl if not present
install_kubectl() {
    if command_exists kubectl; then
        echo_info "kubectl is already installed"
        kubectl version --client --output=yaml | head -10
        return
    fi
    
    echo_info "Installing kubectl..."
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) echo_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    # Download kubectl
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$ARCH/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/kubectl
    
    echo_info "kubectl installed successfully"
    kubectl version --client --output=yaml | head -10
}

# Create minikube cluster
create_cluster() {
    echo_info "Creating minikube cluster..."
    
    # Stop existing cluster if it exists
    if minikube status >/dev/null 2>&1; then
        echo_warn "Stopping existing minikube cluster..."
        minikube stop
        minikube delete
    fi
    
    # Create cluster with specific configuration
    echo_info "Starting minikube with Docker driver..."
    minikube start \
        --driver=docker \
        --cpus=4 \
        --memory=8192 \
        --disk-size=20gb \
        --nodes=3 \
        --kubernetes-version=stable \
        --extra-config=kubelet.housekeeping-interval=10s
    
    # Enable addons
    echo_info "Enabling minikube addons..."
    minikube addons enable storage-provisioner
    minikube addons enable default-storageclass
    minikube addons enable metrics-server
    
    # Wait for cluster to be ready
    echo_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s
    
    echo_info "Cluster created successfully"
    kubectl get nodes -o wide
}

# Build and load Docker image
build_and_load_image() {
    echo_info "Building Docker image..."
    
    # Set up Docker environment to use minikube's Docker daemon
    eval $(minikube docker-env)
    
    # Build the image inside minikube's Docker daemon
    docker build -t distributed-llm:latest .
    
    echo_info "Image built successfully in minikube's Docker daemon"
    docker images | grep distributed-llm
}

# Deploy to cluster
deploy_to_cluster() {
    echo_info "Deploying to cluster..."
    
    # Apply RBAC first
    kubectl apply -f deployments/rbac.yaml
    
    # Apply other deployments
    kubectl apply -f deployments/service.yaml
    kubectl apply -f deployments/daemonset.yaml
    
    # Wait for deployments to be ready
    echo_info "Waiting for pods to be ready..."
    kubectl wait --for=condition=Ready pods --all --timeout=300s 2>/dev/null || true
    
    echo_info "Deployment complete"
    kubectl get pods -o wide
}

# Show cluster info
show_cluster_info() {
    echo_info "Cluster Information:"
    echo "===================="
    echo "Cluster: $(kubectl config current-context)"
    echo
    echo "Nodes:"
    kubectl get nodes -o wide
    echo
    echo "Pods:"
    kubectl get pods -o wide
    echo
    echo "Services:"
    kubectl get services
    echo
    echo "Minikube IP: $(minikube ip)"
    echo
    echo_info "Useful commands:"
    echo "  kubectl get pods -w                     # Watch pods"
    echo "  kubectl logs -l app=distributed-llm-agent -f  # Follow logs"
    echo "  kubectl exec -it <pod-name> -- /bin/sh  # Shell into pod"
    echo "  minikube service distributed-llm-api    # Access service"
    echo "  minikube dashboard                       # Open Kubernetes dashboard"
    echo
    echo_info "To run TUI locally:"
    echo "  make run-tui"
    echo
    echo_info "To rebuild and redeploy:"
    echo "  eval \$(minikube docker-env)"
    echo "  make docker-build"
    echo "  kubectl rollout restart daemonset/distributed-llm-agent"
    echo
    echo_info "To access minikube dashboard:"
    echo "  minikube dashboard"
    echo
    echo_info "To stop cluster:"
    echo "  minikube stop"
    echo
    echo_info "To delete cluster:"
    echo "  minikube delete"
}

# Main execution
main() {
    echo_info "Starting local development setup with minikube..."
    
    install_minikube
    install_kubectl
    create_cluster
    build_and_load_image
    deploy_to_cluster
    show_cluster_info
    
    echo_info "Setup complete! 🎉"
    echo_info "Your distributed LLM cluster is now running on minikube!"
}

# Run main function
main "$@"
