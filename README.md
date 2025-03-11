# Document Management Platform

A secure, scalable system that enables customers to upload, search, and download documents through API integration. This platform addresses the critical need for businesses to tightly integrate document management with their existing business processes.

## Overview

The Document Management Platform is a microservices-based system built on Golang that follows Clean Architecture and Domain-Driven Design principles. It provides comprehensive document management capabilities through well-defined APIs, enabling seamless integration with existing business applications.

## Key Features

The platform offers the following core features:

### Document Management
- Document upload with virus scanning
- Document download (single and batch)
- Document search by content and metadata
- Document listing with pagination and filtering

### Organization
- Folder creation and management
- Hierarchical folder structure
- Document organization within folders

### Security
- Complete tenant isolation
- Role-based access control
- Document encryption at rest and in transit
- Virus scanning for all uploaded documents

### Integration
- Comprehensive REST API
- Webhook notifications for document events
- JWT-based authentication

## Architecture

The Document Management Platform follows Clean Architecture and Domain-Driven Design principles to ensure maintainability and scalability.

### Microservices

The platform consists of the following microservices:

- **API Service**: Handles HTTP requests and provides REST endpoints
- **Worker Service**: Processes asynchronous tasks like virus scanning and document indexing

### Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin v1.9.0+
- **ORM**: GORM v1.25.0+
- **Database**: PostgreSQL 14.0+
- **Search**: Elasticsearch 8.0+
- **Storage**: AWS S3
- **Messaging**: AWS SQS/SNS
- **Caching**: Redis 6.2+
- **Virus Scanning**: ClamAV
- **Authentication**: JWT (golang-jwt/jwt/v4)
- **Logging**: Zap
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry

### Project Structure

The project follows Clean Architecture with the following layers:

```
./
├── api/               # API delivery layer (handlers, middleware, DTOs)
├── application/       # Application use cases
├── cmd/               # Application entry points
│   ├── api/           # API service
│   └── worker/        # Worker service
├── config/            # Configuration files
├── deploy/            # Deployment configurations
│   ├── kubernetes/    # Kubernetes manifests
│   └── terraform/     # Terraform modules
├── domain/            # Domain models and interfaces
│   ├── models/        # Domain entities
│   ├── repositories/  # Repository interfaces
│   └── services/      # Domain service interfaces
├── infrastructure/    # External implementations
│   ├── auth/          # Authentication implementation
│   ├── cache/         # Caching implementation
│   ├── messaging/     # Messaging implementation
│   ├── persistence/   # Database implementation
│   ├── search/        # Search implementation
│   ├── storage/       # Storage implementation
│   ├── thumbnails/    # Thumbnail generation
│   └── virus_scanning/# Virus scanning implementation
├── pkg/               # Shared packages
│   ├── config/        # Configuration utilities
│   ├── errors/        # Error handling
│   ├── logger/        # Logging utilities
│   ├── metrics/       # Metrics collection
│   ├── tracing/       # Distributed tracing
│   ├── utils/         # Utility functions
│   └── validator/     # Input validation
├── scripts/           # Helper scripts
└── test/              # Test utilities and fixtures
    ├── e2e/           # End-to-end tests
    ├── integration/   # Integration tests
    ├── mockery/       # Mock generation
    └── testdata/      # Test data
```

## Getting Started

Follow these instructions to set up the project locally for development and testing purposes.

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Make
- Git
- AWS CLI (for S3 integration)
- kubectl (for Kubernetes deployment)

### Installation

1. Clone the repository
   ```bash
   git clone https://github.com/your-organization/document-management-platform.git
   cd document-management-platform
   ```

2. Set up the development environment
   ```bash
   make setup
   ```

   This command will:
   - Install required Go tools
   - Start required services using Docker Compose
   - Create necessary directories
   - Initialize the local environment

3. Run the application
   ```bash
   # Run the API service
   make run-api

   # Run the worker service
   make run-worker
   ```

   Alternatively, you can use Docker Compose to run all services:
   ```bash
   make docker-compose-up
   ```

### Configuration

The application uses YAML configuration files located in the `config/` directory:

- `default.yml`: Default configuration values
- `development.yml`: Development environment configuration
- `test.yml`: Test environment configuration
- `production.yml`: Production environment configuration

You can override configuration values using environment variables.

## Development

This section provides information for developers working on the Document Management Platform.

### Build

```bash
# Build the application
make build

# Build Docker images
make docker-build
```

### Testing

```bash
# Run all tests
make test

# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e

# Run tests with coverage reporting
make coverage
```

### Code Quality

```bash
# Run linting checks
make lint

# Format code
make fmt

# Generate mocks for testing
make generate-mocks
```

### Database Migrations

```bash
# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create a new migration
make migrate-create name=migration_name
```

### Docker Compose

```bash
# Start all services
make docker-compose-up

# Stop all services
make docker-compose-down

# View logs
make docker-compose-logs
```

## API Documentation

The API documentation is available at `/swagger/index.html` when running the API service. It provides detailed information about all available endpoints, request/response formats, and authentication requirements.

### Authentication

All API requests require a valid JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

The token must include the following claims:
- `sub`: User identifier
- `tenant_id`: Tenant identifier
- `roles`: User roles array

### Main Endpoints

- **Documents**: `/api/v1/documents`
- **Folders**: `/api/v1/folders`
- **Search**: `/api/v1/search`
- **Webhooks**: `/api/v1/webhooks`
- **Health**: `/health`

## Deployment

The Document Management Platform is designed to be deployed to Kubernetes.

### Kubernetes

Kubernetes manifests are available in the `deploy/kubernetes/` directory. You can deploy the application using kubectl:

```bash
kubectl apply -f deploy/kubernetes/
```

### Terraform

Terraform modules for provisioning the required infrastructure are available in the `deploy/terraform/` directory. You can provision the infrastructure using Terraform:

```bash
cd deploy/terraform
terraform init
terraform apply
```

### CI/CD

The project includes GitHub Actions workflows for continuous integration and deployment in the `.github/workflows/` directory.

## Performance and Scalability

The Document Management Platform is designed to handle 10,000+ document uploads daily (averaging 3MB per document) with API response times under 2 seconds. The system scales horizontally to accommodate increased load.

### Monitoring

The platform includes comprehensive monitoring using Prometheus and Grafana. Dashboards are available for system health, performance metrics, and business KPIs.

### Logging

Structured logging is implemented using Zap and can be aggregated using Elasticsearch and Kibana for analysis.

## Security

The Document Management Platform implements multiple security layers:

### Authentication and Authorization

- JWT-based authentication
- Role-based access control
- Complete tenant isolation

### Data Protection

- Encryption at rest (AWS S3 SSE-KMS)
- Encryption in transit (TLS 1.2+)
- Virus scanning for all uploaded documents

### Compliance

The platform is designed to meet SOC2 and ISO27001 compliance requirements.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.