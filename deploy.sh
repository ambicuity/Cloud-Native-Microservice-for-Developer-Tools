#!/bin/bash

# deploy.sh - Deployment script for Kubernetes

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_warning "kubectl not found. Please install kubectl to deploy to Kubernetes."
    exit 1
fi

print_status "Deploying to Kubernetes..."

# Apply namespace and PostgreSQL first
print_status "Deploying PostgreSQL..."
kubectl apply -f k8s/postgres.yaml

# Wait for PostgreSQL to be ready
print_status "Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n build-service --timeout=300s

# Deploy the build service
print_status "Deploying Build Service..."
kubectl apply -f k8s/build-service.yaml

# Wait for deployment to be ready
print_status "Waiting for Build Service deployment..."
kubectl wait --for=condition=available deployment/build-service -n build-service --timeout=300s

print_status "Deployment completed successfully!"

# Display service information
print_status "Service information:"
kubectl get pods -n build-service
kubectl get services -n build-service

print_status "To access the service, run:"
print_status "kubectl port-forward -n build-service svc/build-service 8080:80"