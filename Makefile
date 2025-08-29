# Makefile for Cloud Native Build Service

# Variables
APP_NAME = build-service
VERSION ?= latest
DOCKER_IMAGE = $(APP_NAME):$(VERSION)
NAMESPACE = build-service

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# Build parameters
BINARY_NAME = build-service
BINARY_UNIX = $(BINARY_NAME)-linux

.PHONY: all build clean test coverage deps docker k8s-deploy k8s-delete help

all: deps test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v .

# Build for Linux (useful for Docker)
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -o $(BINARY_UNIX) .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover ./... -coverprofile=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. ./...

# Build Docker image
docker: build-linux
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

# Push Docker image
docker-push: docker
	@echo "Pushing Docker image $(DOCKER_IMAGE)..."
	docker push $(DOCKER_IMAGE)

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up --build

# Stop Docker Compose services
docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

# Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	./deploy.sh

# Delete from Kubernetes
k8s-delete:
	@echo "Deleting from Kubernetes..."
	kubectl delete -f k8s/build-service.yaml --ignore-not-found=true
	kubectl delete -f k8s/postgres.yaml --ignore-not-found=true

# Port forward to access service locally
k8s-port-forward:
	@echo "Port forwarding to service..."
	kubectl port-forward -n $(NAMESPACE) svc/build-service 8080:80

# View logs
k8s-logs:
	@echo "Viewing logs..."
	kubectl logs -n $(NAMESPACE) -l app=build-service -f

# Scale service
k8s-scale:
	@echo "Scaling service to $(REPLICAS) replicas..."
	kubectl scale deployment build-service --replicas=$(REPLICAS) -n $(NAMESPACE)

# Development mode (run locally with file watching)
dev:
	@echo "Starting development mode..."
	@echo "Make sure PostgreSQL is running locally"
	$(GOCMD) run .

# Lint code (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Check for security vulnerabilities
security-check:
	@echo "Running security check..."
	$(GOCMD) list -json -m all | nancy sleuth

# Full CI pipeline
ci: deps lint fmt test coverage build

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-linux   - Build for Linux"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage"
	@echo "  benchmark     - Run benchmarks"
	@echo "  docker        - Build Docker image"
	@echo "  docker-push   - Push Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  docker-stop   - Stop Docker Compose"
	@echo "  k8s-deploy    - Deploy to Kubernetes"
	@echo "  k8s-delete    - Delete from Kubernetes"
	@echo "  k8s-port-forward - Port forward to service"
	@echo "  k8s-logs      - View service logs"
	@echo "  k8s-scale     - Scale service (REPLICAS=n)"
	@echo "  dev           - Run in development mode"
	@echo "  lint          - Lint code"
	@echo "  fmt           - Format code"
	@echo "  ci            - Full CI pipeline"
	@echo "  help          - Show this help"