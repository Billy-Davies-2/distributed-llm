#!/bin/bash

# Install necessary dependencies for the Distributed LLM project

# Update package list
sudo apt-get update

# Install Go if not already installed
if ! command -v go &> /dev/null
then
    echo "Go not found. Installing Go..."
    wget https://golang.org/dl/go1.20.5.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.20.5.linux-amd64.tar.gz
    echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
    source ~/.bashrc
fi

# Install kubectl for Kubernetes management
if ! command -v kubectl &> /dev/null
then
    echo "kubectl not found. Installing kubectl..."
    curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x ./kubectl
    sudo mv ./kubectl /usr/local/bin/kubectl
fi

# Install other dependencies as needed
# Add any additional installation commands here

echo "Installation complete. You can now build and run the Distributed LLM project."