#!/bin/bash

# build.sh - Build script for the Cloud Native Build Service

set -e

echo "🚀 Building Cloud Native Build Service..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    print_error "Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or higher."
    exit 1
fi

print_status "Go version: $GO_VERSION ✓"

# Download dependencies
print_status "Downloading dependencies..."
go mod download

# Run tests
print_status "Running tests..."
go test -v ./...

# Run tests with coverage
print_status "Running tests with coverage..."
go test -cover ./... -coverprofile=coverage.out

# Display coverage
if [ -f coverage.out ]; then
    COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
    print_status "Test coverage: $COVERAGE"
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    print_status "Coverage report generated: coverage.html"
fi

# Build the application
print_status "Building application..."
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build-service .

# Check if build was successful
if [ -f build-service ]; then
    print_status "Build successful! Binary created: build-service"
    
    # Display binary info
    ls -lh build-service
    
    # Check binary
    file build-service
else
    print_error "Build failed!"
    exit 1
fi

print_status "✅ Build completed successfully!"

# Optional: Build Docker image
if command -v docker &> /dev/null; then
    read -p "Do you want to build Docker image? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Building Docker image..."
        docker build -t build-service:latest .
        print_status "Docker image built successfully!"
        
        # Display image info
        docker images build-service:latest
    fi
fi