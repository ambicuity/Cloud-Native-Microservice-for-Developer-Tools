#!/bin/bash

# demo.sh - Demonstration script for the Build Service

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_highlight() {
    echo -e "${YELLOW}$1${NC}"
}

print_header "Cloud-Native Build Service Demo"

print_status "This demo shows the key features of the build service:"
echo "1. 🏗️  Go-based microservice with REST API"
echo "2. 🗄️  PostgreSQL database integration"
echo "3. 📊 Prometheus metrics and health checks"
echo "4. 🐳 Docker containerization"
echo "5. ☸️  Kubernetes deployment manifests"
echo "6. 🧪 Comprehensive test coverage"
echo ""

print_header "Build Information"
print_status "Go version:"
go version

print_status "Binary information:"
if [ -f "build-service" ]; then
    ls -lh build-service
    file build-service
else
    print_status "Building binary..."
    make build
    ls -lh build-service
    file build-service
fi

print_header "Test Results"
print_status "Running comprehensive tests..."
make test

print_header "Coverage Report"
print_status "Generating coverage report..."
make coverage

if [ -f "coverage.html" ]; then
    print_status "Coverage report generated: coverage.html"
    # Extract coverage percentage
    COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
    print_highlight "Test Coverage: $COVERAGE"
fi

print_header "Project Structure"
print_status "Project files:"
tree -I '.git|build-service*|coverage.*' . 2>/dev/null || find . -name "*.go" -o -name "*.yml" -o -name "*.yaml" -o -name "Makefile" -o -name "Dockerfile" -o -name "*.sh" -o -name "*.md" | grep -v ".git" | sort

print_header "Docker Support"
if command -v docker &> /dev/null; then
    print_status "Docker is available"
    print_status "You can build the Docker image with:"
    print_highlight "  make docker"
    print_status "You can run the full stack with:"
    print_highlight "  make docker-run"
else
    print_status "Docker not available in this environment"
fi

print_header "Kubernetes Support"
if command -v kubectl &> /dev/null; then
    print_status "kubectl is available"
    print_status "You can deploy to Kubernetes with:"
    print_highlight "  make k8s-deploy"
else
    print_status "kubectl not available in this environment"
fi

print_header "API Endpoints"
print_status "The service provides the following endpoints:"
echo "📊 Health Check:     GET  /api/v1/health"
echo "🏗️  Create Build:     POST /api/v1/builds"
echo "📋 List Builds:      GET  /api/v1/builds"
echo "🔍 Get Build:        GET  /api/v1/builds/{id}"
echo "📈 Metrics:          GET  /metrics"

print_header "Performance Characteristics"
print_status "Running benchmarks..."
make benchmark

print_header "Demo Complete"
print_status "✅ The Cloud-Native Build Service demonstrates:"
echo "   • Scalable microservice architecture"
echo "   • Database integration with PostgreSQL"
echo "   • Cloud-native deployment with Docker & Kubernetes"
echo "   • Comprehensive testing and monitoring"
echo "   • Enterprise-grade development tooling"
echo ""
print_status "🚀 Ready for production deployment!"