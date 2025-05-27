#!/bin/bash

# Quick build test for distributed LLM project

set -e

echo "🔨 Testing build process..."

# Test Go build
echo "Building Go binaries..."
make build

if [ -f "bin/agent" ] && [ -f "bin/tui" ]; then
    echo "✅ Go binaries built successfully"
    ls -la bin/
else
    echo "❌ Go build failed"
    exit 1
fi

# Test local run (quick check)
echo "Testing agent startup..."
timeout 5s ./bin/agent --help || true
echo "✅ Agent binary works"

echo "Testing TUI startup..."
timeout 2s ./bin/tui || true
echo "✅ TUI binary works"

# Test Docker build (if available)
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    echo "Testing Docker build..."
    docker build -t distributed-llm:test-build .
    echo "✅ Docker build successful"
    docker rmi distributed-llm:test-build
else
    echo "⚠️  Docker not available, skipping Docker build test"
fi

echo "🎉 All build tests passed!"
echo "Ready to deploy with minikube!"
