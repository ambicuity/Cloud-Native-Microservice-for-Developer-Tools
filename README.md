# Cloud-Native Build Service

A highly scalable and reliable cloud-native microservice that provides a centralized build service for developer tools. This service is designed to handle large-scale build operations with PostgreSQL backend integration for data persistence.

## Features

- **REST API** for build management (create, get, list builds)
- **PostgreSQL Integration** for reliable data storage
- **Prometheus Metrics** for observability and monitoring
- **Health Checks** for service reliability
- **Docker Containerization** for cloud deployment
- **Kubernetes Support** with auto-scaling capabilities
- **Graceful Shutdown** for zero-downtime deployments
- **Comprehensive Testing** with unit tests and benchmarks

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   Build Service   │    │   PostgreSQL    │
│    (Ingress)    │───▶│   (Go Service)    │───▶│    Database     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Prometheus     │
                       │   (Metrics)      │
                       └──────────────────┘
```

## API Endpoints

### Health Check
- `GET /api/v1/health` - Service health status

### Build Management  
- `POST /api/v1/builds` - Create a new build
- `GET /api/v1/builds` - List all builds
- `GET /api/v1/builds/{id}` - Get specific build details

### Monitoring
- `GET /metrics` - Prometheus metrics endpoint

## Quick Start

### Using Docker Compose (Recommended for Development)

1. Clone the repository:
```bash
git clone https://github.com/ambicuity/Cloud-Native-Microservice-for-Developer-Tools.git
cd Cloud-Native-Microservice-for-Developer-Tools
```

2. Start the services:
```bash
docker-compose up --build
```

3. The service will be available at `http://localhost:8080`

### Manual Setup

1. **Prerequisites:**
   - Go 1.21 or higher
   - PostgreSQL 15 or higher

2. **Database Setup:**
```bash
# Create database
createdb buildservice

# Set environment variable
export DATABASE_URL="postgres://username:password@localhost:5432/buildservice?sslmode=disable"
```

3. **Run the service:**
```bash
go mod download
go run .
```

## Usage Examples

### Create a Build
```bash
curl -X POST http://localhost:8080/api/v1/builds \
  -H "Content-Type: application/json" \
  -d '{
    "project_name": "my-project",
    "git_url": "https://github.com/user/repo.git",
    "branch": "main"
  }'
```

### Get Build Status
```bash
curl http://localhost:8080/api/v1/builds/1
```

### List All Builds
```bash
curl http://localhost:8080/api/v1/builds
```

### Check Service Health
```bash
curl http://localhost:8080/api/v1/health
```

## Kubernetes Deployment

### Deploy to Kubernetes

1. **Apply the configurations:**
```bash
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/build-service.yaml
```

2. **Verify deployment:**
```bash
kubectl get pods -n build-service
kubectl get services -n build-service
```

3. **Access the service:**
```bash
kubectl port-forward -n build-service svc/build-service 8080:80
```

### Scaling

The service includes Horizontal Pod Autoscaler (HPA) configuration:
- **Minimum replicas:** 3
- **Maximum replicas:** 10  
- **Scaling triggers:** CPU > 70%, Memory > 80%

## Monitoring & Observability

### Prometheus Metrics

The service exposes the following metrics:

- `builds_total` - Total number of builds processed (labeled by status)
- `build_duration_seconds` - Build duration histogram (labeled by project)
- `active_builds` - Current number of active builds
- `health_status` - Service health status (1=healthy, 0=unhealthy)

### Health Checks

- **Liveness Probe:** `/api/v1/health` (checks service responsiveness)
- **Readiness Probe:** `/api/v1/health` (checks database connectivity)

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:password@localhost:5432/buildservice?sslmode=disable` |
| `PORT` | Service port | `8080` |

### Database Schema

The service automatically creates the following table:

```sql
CREATE TABLE builds (
    id SERIAL PRIMARY KEY,
    project_name VARCHAR(255) NOT NULL,
    git_url VARCHAR(500) NOT NULL,
    branch VARCHAR(100) NOT NULL DEFAULT 'main',
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Build Statuses

- `queued` - Build request received and queued
- `running` - Build is currently in progress  
- `success` - Build completed successfully
- `failed` - Build failed with errors

## Performance Characteristics

### Scalability Features

- **Horizontal Scaling:** Supports multiple replica instances
- **Connection Pooling:** Optimized database connection management
- **Async Processing:** Non-blocking build execution
- **Resource Limits:** Configured CPU and memory limits

### Reliability Features

- **Graceful Shutdown:** 30-second shutdown timeout
- **Health Checks:** Automatic unhealthy instance replacement
- **Database Retries:** Built-in connection retry logic
- **Error Handling:** Comprehensive error responses

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.