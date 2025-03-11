# Development Environment Setup

This document provides comprehensive instructions for setting up the development environment for the Document Management Platform. Following these steps will enable you to build, run, and test the application locally.

## Prerequisites

Before setting up the development environment, ensure you have the following tools installed on your system:

### Required Software

- **Go 1.21+**: The primary programming language for the project. [Download Go](https://golang.org/dl/)
- **Docker**: Used for containerization. [Download Docker](https://www.docker.com/get-started)
- **Docker Compose**: Used for managing multi-container applications. [Installation Guide](https://docs.docker.com/compose/install/)
- **Make**: Used for running common development tasks. Usually pre-installed on Linux/macOS, for Windows use [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)
- **Git**: Version control system. [Download Git](https://git-scm.com/downloads)

You can verify the installations with the following commands:

```bash
go version         # Should show Go version 1.21 or higher
docker --version   # Should show Docker version
docker-compose --version # Should show Docker Compose version
make --version     # Should show Make version
git --version      # Should show Git version
```

### Optional Tools

The following tools are not required but recommended for a better development experience:

- **Visual Studio Code**: Recommended IDE with excellent Go support. [Download VS Code](https://code.visualstudio.com/)
- **GoLand**: Purpose-built IDE for Go development. [Download GoLand](https://www.jetbrains.com/go/)
- **Postman**: API testing tool. [Download Postman](https://www.postman.com/downloads/)

Recommended VS Code extensions for Go development:
- Go (by Go Team at Google)
- Go Test Explorer
- Docker
- YAML
- GitLens

## Installation

Follow these steps to set up the development environment:

### Clone the Repository

```bash
# Clone the repository
git clone https://github.com/document-management/backend.git
cd backend
```

### Install Go Tools

The project requires several Go tools for development. You can install them manually or use the setup script:

```bash
# Install required Go tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.0
go install github.com/vektra/mockery/v2@v2.20.0
```

Alternatively, you can use the `make setup` command which will install these tools for you.

### Set Up Development Environment

The project includes a setup script that initializes the development environment with all required services and configurations:

```bash
# Set up the development environment
make setup
```

This command will:
- Install required Go tools
- Start required services using Docker Compose
- Create necessary directories
- Initialize the local environment with required AWS resources in LocalStack
- Set up the database schema and initial data

If you prefer to set up the environment step by step, you can run the following commands:

```bash
# Start required services
make docker-compose-up

# Wait for services to be ready
# Apply database migrations
make migrate-up

# Initialize LocalStack resources
./scripts/setup-dev.sh
```

## Configuration

The Document Management Platform uses YAML configuration files for different environments.

### Configuration Files

The configuration files are located in the `config/` directory:

- `default.yml`: Default configuration values
- `development.yml`: Development environment configuration
- `test.yml`: Test environment configuration
- `production.yml`: Production environment configuration

For local development, you'll primarily use the `development.yml` file. You can override configuration values using environment variables.

### Environment Variables

The following environment variables can be used to customize the configuration:

- `ENV`: Environment name (development, test, production)
- `CONFIG_FILE`: Path to the configuration file
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

Example:
```bash
ENV=development CONFIG_FILE=./config/development.yml LOG_LEVEL=debug make run
```

### LocalStack Configuration

The development environment uses LocalStack to emulate AWS services locally. The following services are configured:

- **S3**: For document storage
- **SQS**: For message queuing
- **SNS**: For event notifications

The setup script creates the necessary resources in LocalStack:

- S3 buckets for document storage (document, temporary, quarantine)
- SQS queues for document processing
- SNS topics for event notifications

You can access the LocalStack web interface at http://localhost:4566/_localstack/dashboard/

### Database Configuration

The development environment uses PostgreSQL for metadata storage. The default configuration is:

- **Host**: localhost
- **Port**: 5432
- **Database**: document_mgmt
- **Username**: postgres
- **Password**: postgres

You can connect to the database using any PostgreSQL client with these credentials.

## Development Workflow

This section describes the typical development workflow for the Document Management Platform.

### Running the Application

You can run the application in different ways:

**Using Make:**

```bash
# Run the API service
make run SERVICE=api

# Run the worker service
make run SERVICE=worker
```

**Using Docker Compose:**

```bash
# Start all services including API and worker
make docker-compose-up

# View logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

The API service will be available at http://localhost:8080

### Building the Application

To build the application:

```bash
# Build the application
make build

# Build Docker images
make docker-build
```

### Testing

The project includes different types of tests:

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

See [testing.md](./testing.md) for more details on testing.

### Code Quality

Maintain code quality with the following commands:

```bash
# Run linting checks
make lint

# Format code
make fmt

# Generate mocks for testing
make generate-mocks
```

See [coding-standards.md](./coding-standards.md) for more details on coding standards.

### Database Migrations

Manage database migrations with the following commands:

```bash
# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create a new migration
make migrate-create name=migration_name
```

Migration files are stored in `infrastructure/persistence/postgres/migrations/`.

### API Documentation

The API documentation is available at `/swagger/index.html` when running the API service. It provides detailed information about all available endpoints, request/response formats, and authentication requirements.

You can access it at http://localhost:8080/swagger/index.html when running the API service locally.

## Project Structure

The Document Management Platform follows Clean Architecture with the following structure:

### Directory Layout

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

### Key Components

The Document Management Platform consists of the following key components:

- **API Service**: Handles HTTP requests and provides REST endpoints
- **Worker Service**: Processes asynchronous tasks like virus scanning and document indexing
- **PostgreSQL**: Stores document metadata and folder structure
- **Elasticsearch**: Provides full-text search capabilities
- **Redis**: Caches frequently accessed data
- **ClamAV**: Scans uploaded documents for viruses
- **LocalStack**: Emulates AWS services (S3, SQS, SNS) locally

## Troubleshooting

This section provides solutions for common issues you might encounter during development.

### Docker Compose Issues

**Issue**: Services fail to start with Docker Compose

**Solution**:
- Check if the required ports are already in use
- Ensure Docker has enough resources allocated
- Try stopping and removing all containers: `docker-compose down -v`
- Check Docker Compose logs: `docker-compose logs`

**Issue**: LocalStack fails to initialize

**Solution**:
- Ensure Docker has enough memory allocated (at least 4GB)
- Check LocalStack logs: `docker-compose logs localstack`
- Try restarting LocalStack: `docker-compose restart localstack`

### Database Issues

**Issue**: Database migrations fail

**Solution**:
- Ensure PostgreSQL is running: `docker-compose ps postgres`
- Check database logs: `docker-compose logs postgres`
- Verify database connection settings in `config/development.yml`
- Try resetting the database: `docker-compose down -v postgres && docker-compose up -d postgres`

**Issue**: Database connection errors

**Solution**:
- Ensure PostgreSQL is running and healthy
- Check if the database exists: `docker exec -it postgres psql -U postgres -c '\l'`
- Verify connection settings in the configuration file

### Go Build Issues

**Issue**: Go build fails with dependency errors

**Solution**:
- Update Go modules: `go mod tidy`
- Clear Go module cache: `go clean -modcache`
- Ensure you're using Go 1.21 or higher: `go version`

**Issue**: Tests fail

**Solution**:
- Ensure all required services are running
- Check test logs for specific errors
- Run tests with verbose output: `go test -v ./...`
- Verify test environment configuration in `config/test.yml`

### LocalStack Issues

**Issue**: AWS resources not available in LocalStack

**Solution**:
- Run the setup script again: `./scripts/setup-dev.sh`
- Check LocalStack logs: `docker-compose logs localstack`
- Verify LocalStack is healthy: `curl http://localhost:4566/_localstack/health`
- Manually create required resources using AWS CLI with LocalStack endpoint:
  ```bash
  aws --endpoint-url=http://localhost:4566 s3 mb s3://document-bucket
  aws --endpoint-url=http://localhost:4566 sqs create-queue --queue-name document-queue
  aws --endpoint-url=http://localhost:4566 sns create-topic --name document-events
  ```

### Common Error Messages

**Error**: `context deadline exceeded`

**Cause**: Service timeout, usually due to a service being unavailable
**Solution**: Check if all required services are running and healthy

**Error**: `permission denied`

**Cause**: File permission issues
**Solution**: Check file permissions, especially for scripts: `chmod +x scripts/*.sh`

**Error**: `no such file or directory`

**Cause**: Missing file or directory
**Solution**: Verify file paths and ensure all required files are present

## Additional Resources

For more information, refer to the following resources:

### Project Documentation

- [README.md](../README.md): Project overview and general information
- [coding-standards.md](./coding-standards.md): Coding standards and best practices
- [testing.md](./testing.md): Testing guidelines and practices
- [API Documentation](http://localhost:8080/swagger/index.html): API documentation (available when running the API service)

### External Documentation

- [Go Documentation](https://golang.org/doc/): Official Go documentation
- [Docker Documentation](https://docs.docker.com/): Official Docker documentation
- [PostgreSQL Documentation](https://www.postgresql.org/docs/): PostgreSQL documentation
- [Elasticsearch Documentation](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html): Elasticsearch documentation
- [Redis Documentation](https://redis.io/documentation): Redis documentation
- [ClamAV Documentation](https://docs.clamav.net/): ClamAV documentation
- [LocalStack Documentation](https://docs.localstack.cloud/): LocalStack documentation

### Getting Help

If you encounter issues not covered in this guide:

- Check the project's issue tracker for similar problems
- Consult with team members
- Refer to the project's internal documentation
- For external dependencies, consult their official documentation