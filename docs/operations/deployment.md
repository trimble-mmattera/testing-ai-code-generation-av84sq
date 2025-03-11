# Deployment Guide

## 1. Introduction

This document provides comprehensive guidelines for deploying the Document Management Platform. It covers all aspects of deployment including infrastructure provisioning, container building, Kubernetes deployment, and CI/CD pipelines across development, staging, and production environments.

### 1.1 Deployment Philosophy

The Document Management Platform follows a deployment philosophy centered around:

- **Infrastructure as Code**: All infrastructure is defined and provisioned using Terraform to ensure consistency and reproducibility.
- **Containerization**: All services are containerized using Docker for consistent environments across development, testing, and production.
- **Orchestration**: Kubernetes is used for container orchestration, enabling scalability, resilience, and efficient resource utilization.
- **CI/CD Automation**: GitHub Actions pipelines automate building, testing, and deploying the platform components.
- **Progressive Deployment**: Different deployment strategies are used for different environments to balance speed and risk.
- **Immutable Infrastructure**: Once deployed, infrastructure components are not modified but replaced with new versions.

### 1.2 Deployment Environments

The platform is deployed across three distinct environments:

- **Development (dev)**: Used for ongoing development and testing. It supports rapid iteration with direct deployments and minimal overhead. Resources are sized for development workloads only.
  
- **Staging (staging)**: Mirrors the production environment but with smaller resource allocations. Used for pre-production validation, integration testing, and user acceptance testing. Implements blue-green deployment for testing production deployment procedures.
  
- **Production (prod)**: The live environment serving actual customers. Uses canary deployment for risk mitigation, has the highest resource allocations, and requires approval for any changes.

### 1.3 Deployment Components

The deployment process involves several key components:

- **Infrastructure Components**: AWS resources provisioned via Terraform (VPC, EKS, RDS, S3, etc.)
- **Container Images**: Docker images for microservices built via GitHub Actions
- **Kubernetes Manifests**: YAML files defining deployments, services, config maps, etc.
- **CI/CD Pipelines**: GitHub Actions workflows for automated building and deployment
- **Configuration Files**: Environment-specific configuration for each component
- **Monitoring & Alerting**: Prometheus, Grafana, AlertManager for deployment monitoring

## 2. Infrastructure Provisioning

### 2.1 Terraform Configuration

The infrastructure is defined using Terraform with a modular structure:

```
/terraform
├── main.tf                # Main configuration file
├── variables.tf           # Input variables
├── outputs.tf             # Output values
├── provider.tf            # Provider configuration
├── environments/          # Environment-specific configurations
│   ├── dev/               # Development environment
│   ├── staging/           # Staging environment
│   └── prod/              # Production environment
└── modules/               # Reusable modules
    ├── vpc/               # VPC configuration
    ├── eks/               # EKS cluster configuration
    ├── rds/               # RDS database configuration
    ├── s3/                # S3 buckets configuration
    ├── kms/               # KMS keys configuration
    └── sqs/               # SQS queues configuration
```

Each module encapsulates a specific component of the infrastructure with its own inputs, outputs, and resources. This modular approach promotes reusability and maintainability.

### 2.2 Environment-Specific Configuration

Environment-specific configurations are stored in the `environments` directory:

```
/environments/
├── dev/
│   └── terraform.tfvars
├── staging/
│   └── terraform.tfvars
└── prod/
    └── terraform.tfvars
```

Each environment has its own `terraform.tfvars` file containing environment-specific variables:

```hcl
# Example of terraform.tfvars for production
environment                = "prod"
region                     = "us-west-2"
vpc_cidr                   = "10.0.0.0/16"
availability_zones         = ["us-west-2a", "us-west-2b", "us-west-2c"]
eks_cluster_version        = "1.25"
node_group_instance_types  = ["m5.2xlarge"]
node_group_desired_size    = 3
node_group_min_size        = 3
node_group_max_size        = 10
db_instance_class          = "db.m5.2xlarge"
db_allocated_storage       = 100
db_multi_az                = true
enable_deletion_protection = true
```

### 2.3 State Management

Terraform state is stored remotely in S3 with locking via DynamoDB:

```hcl
terraform {
  backend "s3" {
    bucket         = "document-mgmt-terraform-state"
    key            = "env/terraform.tfstate"
    region         = "us-west-2"
    encrypt        = true
    dynamodb_table = "document-mgmt-terraform-state-lock"
  }
}
```

This ensures:
- State is securely stored and encrypted
- Multiple users can't modify the infrastructure concurrently
- State history is maintained
- Remote operations can be performed by CI/CD pipelines

### 2.4 Infrastructure Deployment Process

The following process is used to deploy infrastructure changes:

1. **Initialize Terraform**:
   ```bash
   terraform init -backend-config=environments/${ENV}/backend.hcl
   ```

2. **Select Workspace** (environment):
   ```bash
   terraform workspace select ${ENV} || terraform workspace new ${ENV}
   ```

3. **Plan Changes**:
   ```bash
   terraform plan -var-file=environments/${ENV}/terraform.tfvars -out=tfplan
   ```

4. **Review Plan**:
   - Verify the planned changes before applying
   - For production, require approval from at least one infrastructure engineer

5. **Apply Changes**:
   ```bash
   terraform apply tfplan
   ```

6. **Verify Deployment**:
   - Check AWS Console or use AWS CLI to verify resources were created correctly
   - Run validation tests against the new infrastructure

## 3. Container Build Process

### 3.1 Dockerfile Structure

All services use multi-stage Docker builds to minimize image size and improve security:

```dockerfile
# Base build image
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/service

# Runtime image
FROM alpine:3.17

# Add non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copy binary from builder stage
COPY --from=builder /app/main /app/main

# Set executable permissions
RUN chmod +x /app/main

# Expose port
EXPOSE 8080

# Run the service
ENTRYPOINT ["/app/main"]
```

Key features:
- Multi-stage build to separate build and runtime environments
- Minimal base image (alpine) for security and size
- Non-root user for improved security
- Dependencies cached in a separate layer for faster builds

### 3.2 Build Pipeline

Container images are built using GitHub Actions with the following workflow:

```yaml
name: Build Container Images

on:
  push:
    branches: [main, develop]
    paths:
      - 'src/**'
      - '.github/workflows/build.yml'
  pull_request:
    branches: [main, develop]
    paths:
      - 'src/**'
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v1
      
      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v4
        with:
          images: ${{ steps.login-ecr.outputs.registry }}/document-mgmt
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=sha,format=short
            type=semver,pattern={{version}}
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'Dockerfile'
          format: 'table'
          exit-code: '1'
          severity: 'CRITICAL,HIGH'
      
      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

This workflow:
1. Checks out the repository code
2. Sets up Docker Buildx for efficient builds
3. Authenticates with AWS ECR
4. Extracts metadata for proper tagging
5. Scans the Dockerfile for vulnerabilities
6. Builds and pushes the container image to ECR

### 3.3 Image Tagging Strategy

The platform uses a comprehensive tagging strategy:

- **Semantic Versioning**: For released versions (e.g., `v1.2.3`)
- **Git Branch**: For branch-specific builds (e.g., `develop`, `feature-x`)
- **Git Commit SHA**: For precise traceability (e.g., `sha-a1b2c3d`)
- **Environment**: For environment-specific images (e.g., `v1.2.3-prod`)

Production deployments always use specific version tags rather than `latest` to ensure reproducibility.

### 3.4 Security Scanning

All container images are scanned for vulnerabilities before deployment:

1. **Static Analysis**: Using Trivy during the build process
2. **Dependency Scanning**: Using OWASP Dependency Check
3. **Secret Scanning**: Using git-secrets to prevent credential leakage
4. **Base Image Updates**: Regularly updating base images to include security patches

Images with critical vulnerabilities are blocked from being deployed, and images with high vulnerabilities require review and approval.

## 4. Kubernetes Deployment

### 4.1 Kubernetes Manifest Structure

Kubernetes manifests are organized by environment and service:

```
/kubernetes
├── base/                  # Base configurations
│   ├── document-service/  # Document service base config
│   ├── search-service/    # Search service base config
│   └── ...
├── overlays/              # Environment-specific overlays
│   ├── dev/               # Development overlays
│   ├── staging/           # Staging overlays
│   └── prod/              # Production overlays
└── common/                # Common resources
    ├── namespaces.yaml
    ├── network-policies.yaml
    └── resource-quotas.yaml
```

This structure follows the Kustomize pattern with base configurations and environment-specific overlays.

An example of a service deployment manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: document-api
  labels:
    app: document-management
    component: api
    part-of: document-platform
  annotations:
    kubernetes.io/description: "API service for the Document Management Platform"
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/metrics"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: document-management
      component: api
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: document-management
        component: api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: api
        image: ${REPOSITORY}/document-api:${TAG}
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: 8080
          protocol: TCP
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        env:
        - name: ENV
          value: "${ENVIRONMENT}"
        - name: LOG_LEVEL
          value: "info"
        volumeMounts:
        - name: config-volume
          mountPath: /app/config
      volumes:
      - name: config-volume
        configMap:
          name: document-api-config
```

### 4.2 Environment Configuration

Environment-specific configuration is managed using Kubernetes ConfigMaps and Secrets:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: document-api-config
  namespace: document-mgmt-prod
data:
  config.yml: |
    server:
      host: "0.0.0.0"
      port: 8080
    database:
      host: "${DB_HOST}"
      port: 5432
    s3:
      region: "us-west-2"
      bucket: "document-mgmt-prod-documents"
      tempBucket: "document-mgmt-prod-temp"
      quarantineBucket: "document-mgmt-prod-quarantine"
```

Sensitive configuration is stored in Kubernetes Secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: document-api-secrets
  namespace: document-mgmt-prod
type: Opaque
data:
  db_password: ${BASE64_DB_PASSWORD}
  jwt_secret: ${BASE64_JWT_SECRET}
  aws_access_key: ${BASE64_AWS_ACCESS_KEY}
  aws_secret_key: ${BASE64_AWS_SECRET_KEY}
```

### 4.3 Resource Requirements

Resource requirements are defined for each environment:

| Service | Environment | CPU Request | Memory Request | CPU Limit | Memory Limit |
| ------- | ----------- | ----------- | -------------- | --------- | ------------ |
| Document API | Development | 0.5 | 1Gi | 1 | 2Gi |
| Document API | Staging | 1 | 2Gi | 2 | 4Gi |
| Document API | Production | 2 | 4Gi | 4 | 8Gi |
| Storage Service | Development | 0.5 | 1Gi | 1 | 2Gi |
| Storage Service | Staging | 1 | 2Gi | 2 | 4Gi |
| Storage Service | Production | 2 | 4Gi | 4 | 8Gi |
| Search Service | Development | 1 | 2Gi | 2 | 4Gi |
| Search Service | Staging | 2 | 4Gi | 4 | 8Gi |
| Search Service | Production | 4 | 8Gi | 8 | 16Gi |
| Virus Scanning | Development | 0.5 | 1Gi | 1 | 2Gi |
| Virus Scanning | Staging | 1 | 2Gi | 2 | 4Gi |
| Virus Scanning | Production | 2 | 4Gi | 4 | 8Gi |

### 4.4 Health Checks and Probes

All services implement health check endpoints for Kubernetes probes:

- **Liveness Probe**: Checks if the service is running
- **Readiness Probe**: Checks if the service is ready to accept traffic

Example probe configuration:

```yaml
livenessProbe:
  httpGet:
    path: /health/liveness
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
readinessProbe:
  httpGet:
    path: /health/readiness
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

The health check endpoints implement the following checks:
- Liveness: Basic application functionality
- Readiness: Dependency availability (database, S3, etc.)

## 5. Deployment Strategies

### 5.1 Direct Deployment (Development)

The development environment uses a direct deployment strategy for rapid iteration:

1. **Build and Tag Container**: Build and push container with development tag
2. **Update Deployment**: Update the Kubernetes deployment with the new image
3. **Rolling Update**: Kubernetes performs a rolling update of the pods

Implementation using kubectl:

```bash
# Update deployment with new image
kubectl set image deployment/document-api api=${REPOSITORY}/document-api:${TAG} -n document-mgmt-dev

# Watch rollout status
kubectl rollout status deployment/document-api -n document-mgmt-dev
```

Advantages:
- Simple and fast
- Minimal complexity
- Quick feedback loop

Disadvantages:
- No zero-downtime guarantee
- No easy rollback mechanism
- May cause disruption during deployment

### 5.2 Blue-Green Deployment (Staging)

The staging environment uses a blue-green deployment strategy:

1. **Deploy Green Version**: Deploy the new version of the application (green)
2. **Test Green Version**: Validate the green version functionality
3. **Switch Traffic**: Update the service to route traffic to the green version
4. **Verify Green Version**: Monitor the green version for any issues
5. **Remove Blue Version**: Once verified, remove the old version (blue)

Implementation using kubectl:

```bash
# Create blue deployment with new version
kubectl apply -f blue-deployment.yaml -n document-mgmt-staging

# Wait for blue deployment to be ready
kubectl rollout status deployment/document-api-blue -n document-mgmt-staging

# Verify blue deployment health
kubectl exec -it $(kubectl get pods -n document-mgmt-staging -l app=document-management,component=api,version=blue -o jsonpath='{.items[0].metadata.name}') -n document-mgmt-staging -- curl -s http://localhost:8080/health

# Switch traffic to blue deployment
kubectl apply -f service-blue.yaml -n document-mgmt-staging

# Verify application functionality
./run-smoke-tests.sh https://api-staging.document-mgmt.example.com

# Remove green deployment after successful switch
kubectl delete deployment/document-api-green -n document-mgmt-staging
```

Advantages:
- Zero downtime deployment
- Easy and immediate rollback
- Complete testing of new version before traffic switch

Disadvantages:
- Requires double the resources during deployment
- More complex implementation
- Requires manual verification step

### 5.3 Canary Deployment (Production)

The production environment uses a canary deployment strategy:

1. **Deploy Canary**: Deploy a small portion of the new version (canary)
2. **Route Limited Traffic**: Direct a small percentage of traffic to the canary
3. **Monitor Canary**: Watch for errors, performance issues in the canary
4. **Gradually Increase Traffic**: If stable, increase traffic to the canary
5. **Complete Deployment**: Once verified, migrate all traffic to the new version

Implementation using kubectl and service weights:

```bash
# Create canary deployment with new version
kubectl apply -f canary-deployment.yaml -n document-mgmt-prod

# Apply canary service with 10% traffic
kubectl apply -f canary-service-10.yaml -n document-mgmt-prod

# Wait for canary deployment to be ready
kubectl rollout status deployment/document-api-canary -n document-mgmt-prod

# Monitor canary metrics for 15 minutes
echo "Monitoring canary deployment for 15 minutes..."
sleep 900

# Check error rate and response time
canary_error_rate=$(curl -s "http://prometheus.document-mgmt.svc:9090/api/v1/query?query=sum(rate(http_requests_total{job=\"document-api\",status=~\"5..\",version=\"canary\"}[5m]))/sum(rate(http_requests_total{job=\"document-api\",version=\"canary\"}[5m]))" | jq -r '.data.result[0].value[1]')
canary_response_time=$(curl -s "http://prometheus.document-mgmt.svc:9090/api/v1/query?query=histogram_quantile(0.95,sum(rate(http_request_duration_seconds_bucket{job=\"document-api\",version=\"canary\"}[5m]))by(le))" | jq -r '.data.result[0].value[1]')

# If metrics look good, increase traffic to 25%
if (( $(echo "$canary_error_rate < 0.01" | bc -l) )) && (( $(echo "$canary_response_time < 2.0" | bc -l) )); then
  kubectl apply -f canary-service-25.yaml -n document-mgmt-prod
else
  echo "Canary metrics exceeded thresholds. Rolling back."
  kubectl delete deployment/document-api-canary -n document-mgmt-prod
  exit 1
fi

# Continue with progressive traffic shifts (50%, 75%, 100%) with monitoring between each step
```

Advantages:
- Minimal risk with gradual rollout
- Early detection of issues with limited impact
- Real production traffic validation
- Fine-grained control over rollout

Disadvantages:
- Complex implementation
- Longer deployment time
- Requires sophisticated monitoring

### 5.4 Rollback Procedures

Rollback procedures differ by environment and deployment strategy:

**Development Rollback**:
```bash
# Rollback deployment to previous version
kubectl rollout undo deployment/document-api -n document-mgmt-dev

# Check rollback status
kubectl rollout status deployment/document-api -n document-mgmt-dev

# Verify service health
kubectl get pods -n document-mgmt-dev
curl https://api-dev.document-mgmt.example.com/health
```

**Staging Rollback (Blue-Green)**:
```bash
# Switch service back to green deployment
kubectl apply -f service-green.yaml -n document-mgmt-staging

# Verify green deployment is functioning
curl https://api-staging.document-mgmt.example.com/health

# Remove blue deployment
kubectl delete deployment/document-api-blue -n document-mgmt-staging
```

**Production Rollback (Canary)**:
```bash
# Stop routing traffic to canary
kubectl apply -f stable-service-100.yaml -n document-mgmt-prod

# Verify stable version is receiving all traffic
kubectl get svc -n document-mgmt-prod

# Remove canary deployment
kubectl delete deployment/document-api-canary -n document-mgmt-prod

# Monitor service health
curl https://api.document-mgmt.example.com/health
```

## 6. CI/CD Pipelines

### 6.1 Build Pipeline

The build pipeline is responsible for building and pushing container images:

```yaml
name: Build Pipeline

on:
  push:
    branches: [main, develop]
    paths:
      - 'src/**'
      - '.github/workflows/build.yml'
  pull_request:
    branches: [main, develop]
    paths:
      - 'src/**'
  workflow_dispatch:

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.21
      
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: v1.54
          args: --timeout=5m
  
  test:
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.21
      
      - name: Run tests
        run: go test -race -coverprofile=coverage.txt ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.txt
  
  security-scan:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Run gosec
        uses: securego/gosec@master
        with:
          args: ./...
      
      - name: Run dependency check
        uses: actions/dependency-review-action@v2
  
  build:
    runs-on: ubuntu-latest
    needs: [test, security-scan]
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v1
      
      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          push: ${{ github.event_name != 'pull_request' }}
          tags: ${{ steps.login-ecr.outputs.registry }}/document-api:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
      
      - name: Scan image
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: ${{ steps.login-ecr.outputs.registry }}/document-api:${{ github.sha }}
          format: 'sarif'
          output: 'trivy-results.sarif'
      
      - name: Upload scan results
        uses: github/codeql-action/upload-sarif@v2
        with:
          sarif_file: 'trivy-results.sarif'
```

### 6.2 Development Deployment Pipeline

The development deployment pipeline automatically deploys to the development environment:

```yaml
name: Deploy to Development

on:
  push:
    branches: [develop]
  workflow_dispatch:
    inputs:
      deploy_infrastructure:
        description: 'Deploy infrastructure changes'
        required: false
        default: 'false'
        type: boolean
      deploy_services:
        description: 'Deploy application services'
        required: false
        default: 'true'
        type: boolean
      run_integration_tests:
        description: 'Run integration tests'
        required: false
        default: 'true'
        type: boolean

jobs:
  infrastructure:
    runs-on: ubuntu-latest
    if: github.event.inputs.deploy_infrastructure == 'true' || github.event_name == 'push'
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2
        with:
          terraform_version: 1.5.0
      
      - name: Terraform Init
        run: terraform init
        working-directory: src/backend/deploy/terraform
        
      - name: Terraform Apply
        run: terraform apply -auto-approve -var-file=environments/dev/terraform.tfvars
        working-directory: src/backend/deploy/terraform
  
  deploy:
    runs-on: ubuntu-latest
    needs: infrastructure
    if: github.event.inputs.deploy_services == 'true' || github.event_name == 'push'
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Update kubeconfig
        run: aws eks update-kubeconfig --name document-mgmt-dev --region us-west-2
      
      - name: Deploy to Kubernetes
        run: |
          kubectl apply -f kubernetes/common/namespaces.yaml
          kubectl apply -f kubernetes/overlays/dev/
          kubectl rollout status deployment/document-api -n document-mgmt-dev
          kubectl rollout status deployment/storage-service -n document-mgmt-dev
          kubectl rollout status deployment/search-service -n document-mgmt-dev
          kubectl rollout status deployment/virus-scanning-service -n document-mgmt-dev
  
  integration-tests:
    runs-on: ubuntu-latest
    needs: deploy
    if: github.event.inputs.run_integration_tests == 'true' || github.event_name == 'push'
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.21
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Run integration tests
        run: go test -tags=integration ./tests/integration/...
        env:
          API_BASE_URL: https://api-dev.document-mgmt.example.com
          TEST_JWT_TOKEN: ${{ secrets.DEV_JWT_TOKEN }}
```

### 6.3 Staging Deployment Pipeline

The staging deployment pipeline implements blue-green deployment:

```yaml
name: Deploy to Staging

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Update kubeconfig
        run: aws eks update-kubeconfig --name document-mgmt-staging --region us-west-2
      
      - name: Deploy Blue Environment
        run: |
          # Determine if blue or green is currently active
          if kubectl get svc document-api -n document-mgmt-staging -o jsonpath='{.spec.selector.version}' | grep -q "blue"; then
            export NEW_COLOR="green"
            export OLD_COLOR="blue"
          else
            export NEW_COLOR="blue"
            export OLD_COLOR="green"
          fi
          
          echo "Deploying new version to ${NEW_COLOR} environment"
          
          # Deploy new version
          sed -e "s/VERSION/${NEW_COLOR}/g" kubernetes/overlays/staging/document-api-template.yaml > kubernetes/overlays/staging/document-api-${NEW_COLOR}.yaml
          kubectl apply -f kubernetes/overlays/staging/document-api-${NEW_COLOR}.yaml
          
          # Wait for deployment to be ready
          kubectl rollout status deployment/document-api-${NEW_COLOR} -n document-mgmt-staging
      
      - name: Test Blue Environment
        run: |
          # Get the service IP
          export SVC_IP=$(kubectl get svc document-api-${NEW_COLOR} -n document-mgmt-staging -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
          
          # Run smoke tests against new deployment
          ./scripts/run-smoke-tests.sh http://${SVC_IP}
      
      - name: Switch Traffic
        run: |
          # Switch the main service to point to the new deployment
          kubectl patch svc document-api -n document-mgmt-staging -p '{"spec":{"selector":{"version":"'${NEW_COLOR}'"}}}'
          
          # Verify the switch
          sleep 10
          kubectl get svc document-api -n document-mgmt-staging -o jsonpath='{.spec.selector.version}'
      
      - name: Run E2E Tests
        run: |
          # Run E2E tests against the main service
          go test -tags=e2e ./tests/e2e/...
        env:
          API_BASE_URL: https://api-staging.document-mgmt.example.com
          TEST_JWT_TOKEN: ${{ secrets.STAGING_JWT_TOKEN }}
      
      - name: Cleanup Old Deployment
        run: |
          # If tests pass, remove the old deployment
          kubectl delete deployment/document-api-${OLD_COLOR} -n document-mgmt-staging
```

### 6.4 Production Deployment Pipeline

The production deployment pipeline implements canary deployment with approval:

```yaml
name: Deploy to Production

on:
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to deploy (semver or git sha)'
        required: true
        default: 'latest'

jobs:
  plan:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Update kubeconfig
        run: aws eks update-kubeconfig --name document-mgmt-prod --region us-west-2
      
      - name: Generate deployment plan
        run: |
          export VERSION=${{ github.event.inputs.version }}
          echo "Planning deployment of version ${VERSION} to production"
          
          # Generate deployment manifests
          sed -e "s/VERSION/${VERSION}/g" kubernetes/overlays/prod/document-api-canary-template.yaml > kubernetes/overlays/prod/document-api-canary.yaml
          
          # Show what will be deployed
          kubectl diff -f kubernetes/overlays/prod/document-api-canary.yaml || true
    
    outputs:
      version: ${{ github.event.inputs.version }}
  
  approval:
    runs-on: ubuntu-latest
    needs: plan
    steps:
      - name: Approval required
        uses: trstringer/manual-approval@v1
        with:
          secret: ${{ secrets.GITHUB_TOKEN }}
          approvers: tech-lead,infrastructure-admin
          minimum-approvals: 1
          issue-title: "Approval needed for Production Deployment"
          issue-body: "Please approve the deployment of version ${{ needs.plan.outputs.version }} to production."
          exclude-workflow-initiator-as-approver: false
  
  deploy:
    runs-on: ubuntu-latest
    needs: [plan, approval]
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-west-2
      
      - name: Update kubeconfig
        run: aws eks update-kubeconfig --name document-mgmt-prod --region us-west-2
      
      - name: Deploy Canary
        run: |
          export VERSION=${{ needs.plan.outputs.version }}
          
          # Deploy canary with new version
          sed -e "s/VERSION/${VERSION}/g" kubernetes/overlays/prod/document-api-canary-template.yaml > kubernetes/overlays/prod/document-api-canary.yaml
          kubectl apply -f kubernetes/overlays/prod/document-api-canary.yaml
          
          # Wait for canary deployment to be ready
          kubectl rollout status deployment/document-api-canary -n document-mgmt-prod
      
      - name: Route 10% Traffic to Canary
        run: |
          kubectl apply -f kubernetes/overlays/prod/service-canary-10.yaml
          echo "Routing 10% of traffic to canary deployment"
          
          # Wait for traffic to stabilize
          sleep 60
      
      - name: Monitor Canary
        run: |
          # Monitor canary metrics for 15 minutes
          echo "Monitoring canary deployment for 15 minutes..."
          sleep 900
          
          # Get canary metrics from Prometheus
          canary_error_rate=$(curl -s "http://prometheus.document-mgmt.svc:9090/api/v1/query?query=sum(rate(http_requests_total{job=\"document-api\",status=~\"5..\",version=\"canary\"}[5m]))/sum(rate(http_requests_total{job=\"document-api\",version=\"canary\"}[5m]))" | jq -r '.data.result[0].value[1]')
          canary_response_time=$(curl -s "http://prometheus.document-mgmt.svc:9090/api/v1/query?query=histogram_quantile(0.95,sum(rate(http_request_duration_seconds_bucket{job=\"document-api\",version=\"canary\"}[5m]))by(le))" | jq -r '.data.result[0].value[1]')
          
          # Check if metrics are within acceptable ranges
          if (( $(echo "$canary_error_rate < 0.01" | bc -l) )) && (( $(echo "$canary_response_time < 2.0" | bc -l) )); then
            echo "Canary metrics are within acceptable ranges. Proceeding with deployment."
          else
            echo "Canary metrics exceeded thresholds. Rolling back."
            kubectl apply -f kubernetes/overlays/prod/service-stable-100.yaml
            kubectl delete deployment/document-api-canary -n document-mgmt-prod
            exit 1
          fi
      
      # Additional steps for progressive traffic shifting (25%, 50%, 75%, 100%)
      # Each step includes monitoring and validation
```

## 7. Environment Promotion

### 7.1 Promotion Criteria

To ensure quality and stability, changes must meet specific criteria before promotion:

**Development to Staging**:
- All unit tests pass
- Integration tests pass
- Code review completed
- Security scan shows no critical or high issues
- Service health checks pass

**Staging to Production**:
- All E2E tests pass
- Performance tests meet SLA requirements
- Security review completed
- User acceptance testing completed
- Change approval received

These criteria ensure that only high-quality, verified changes are promoted to higher environments.

### 7.2 Promotion Workflow

The promotion workflow follows these steps:

1. **Development**:
   - Changes are developed and tested in feature branches
   - Pull requests are created for integration into the develop branch
   - CI pipeline builds and deploys to development environment
   - Integration tests validate functionality

2. **Staging**:
   - Stable changes from develop are merged into the main branch
   - CI pipeline builds and deploys to staging using blue-green deployment
   - E2E tests validate functionality
   - Performance tests verify SLA compliance

3. **Production**:
   - Release is created from main branch with version tag
   - Manual workflow is triggered specifying the version
   - Approval is required before proceeding
   - Canary deployment with progressive traffic shifting
   - Monitoring verifies stability before full rollout

This promotion workflow ensures controlled progression of changes through environments with appropriate validation at each stage.

### 7.3 Approval Process

The approval process is formalized to ensure proper oversight:

**Development Deployments**:
- Automatic for changes to the develop branch
- No approval required

**Staging Deployments**:
- Automatic for changes to the main branch
- Require tech lead approval for manual deployments

**Production Deployments**:
- Require approval from at least one designated approver
- Approvers include tech leads and infrastructure administrators
- Change must include:
  - Version to be deployed
  - Release notes/changelog
  - Testing evidence
  - Rollback plan

Approvals are tracked in GitHub using the manual approval workflow, providing an audit trail of deployment decisions.

### 7.4 Promotion Verification

After promotion to each environment, verification ensures successful deployment:

**Development Verification**:
- Health check endpoints return success
- Integration tests pass
- Services can be accessed and used
- No errors in logs

**Staging Verification**:
- All services deployed successfully
- Blue-green switch completed without errors
- E2E tests pass
- Performance metrics within expected ranges
- No errors in monitoring

**Production Verification**:
- Canary deployment metrics show no issues
- Progressive traffic shifting completed successfully
- Smoke tests pass
- SLA metrics are maintained
- No errors in monitoring or alerting

## 8. Deployment Verification

### 8.1 Health Checks

Health checks verify service availability and readiness after deployment:

```bash
# Check node status
kubectl get nodes

# Check pod status
kubectl get pods -n document-mgmt

# Check deployment status
kubectl get deployments -n document-mgmt

# Check service status
kubectl get services -n document-mgmt

# Check logs for API service
kubectl logs -n document-mgmt deployment/document-api --tail=100

# Check logs for Worker service
kubectl logs -n document-mgmt deployment/document-worker --tail=100

# Check health endpoint
API_ENDPOINT=$(kubectl get svc -n document-mgmt document-api -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
curl -s http://$API_ENDPOINT/health | jq
```

Every service exposes health endpoints that should be checked after deployment:
- `/health/liveness`: Basic application health
- `/health/readiness`: Dependency availability

### 8.2 Functional Testing

Functional tests validate that the application behaves as expected:

- **Smoke Tests**: Basic functionality tests to verify critical paths
- **Integration Tests**: Tests for service interaction
- **E2E Tests**: Full user journey tests

These tests should be run automatically as part of the deployment pipeline but can also be run manually if needed:

```bash
# Run smoke tests
./scripts/run-smoke-tests.sh https://api-prod.document-mgmt.example.com

# Run integration tests
go test -tags=integration ./tests/integration/...

# Run E2E tests
go test -tags=e2e ./tests/e2e/...
```

### 8.3 Performance Validation

Performance validation ensures the deployment meets SLA requirements:

```bash
# Run performance tests
k6 run performance-tests/api-load-test.js -e ENV=prod

# Check latency metrics in Prometheus
curl -s "http://prometheus.document-mgmt.svc:9090/api/v1/query?query=histogram_quantile(0.95,sum(rate(http_request_duration_seconds_bucket{job=\"document-api\"}[5m]))by(le))"

# Check error rate metrics in Prometheus
curl -s "http://prometheus.document-mgmt.svc:9090/api/v1/query?query=sum(rate(http_requests_total{job=\"document-api\",status=~\"5..\"}[5m]))/sum(rate(http_requests_total{job=\"document-api\"}[5m]))"

# Check resource utilization
kubectl top pods -n document-mgmt
```

Performance validation should verify that:
- API response times are under 2 seconds (95th percentile)
- Error rates are below 1%
- Resource utilization is within expected ranges
- Document processing times are under 5 minutes

### 8.4 Security Validation

Security validation ensures that deployed services meet security requirements:

```bash
# Check NetworkPolicy enforcement
kubectl get networkpolicies -n document-mgmt

# Verify TLS configuration
nmap --script ssl-enum-ciphers -p 443 api.document-mgmt.example.com

# Check for exposed secrets
kubectl get pods -n document-mgmt -o json | jq '.items[].spec.containers[].env[]? | select(.valueFrom.secretKeyRef)'

# Verify RBAC configuration
kubectl auth can-i --list --namespace document-mgmt

# Run security scan
kube-scan scan -n document-mgmt
```

Security validation should verify that:
- HTTPS is enforced for all endpoints
- Network policies restrict unauthorized access
- Secrets are properly managed
- RBAC is correctly configured
- No security vulnerabilities in the deployment

## 9. Deployment Monitoring

### 9.1 Deployment Metrics

Key metrics to monitor during and after deployments:

**Service Health Metrics**:
- Pod ready status
- Container restarts
- Liveness/readiness probe success

**Performance Metrics**:
- API response time
- Error rate
- Request throughput
- Resource utilization (CPU, memory)

**Business Metrics**:
- Document uploads
- Search operations
- Download operations
- Processing success rate

Monitor these metrics using Prometheus and Grafana dashboards specifically designed for deployment monitoring.

### 9.2 Alerting During Deployment

Special alerting considerations during deployments:

- **Silence Non-Critical Alerts**: Configure temporary silences for alerts that might trigger during deployment
- **Deployment-Specific Alerts**: Enable alerts specifically for monitoring deployment progress
- **Staged Alerting**: Different alert thresholds during deployment vs. normal operation

Configure AlertManager to handle deployment-specific alerting:

```yaml
# Excerpt from AlertManager configuration for deployments
route:
  receiver: 'default'
  group_by: ['alertname', 'job']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
  - match:
      deployment_status: 'in_progress'
    receiver: 'deployment-team'
    group_wait: 10s
    group_interval: 30s
    repeat_interval: 1m
```

### 9.3 Deployment Dashboards

Specialized dashboards for monitoring deployments:

- **Deployment Progress Dashboard**: Shows status of deployment rollout
- **Service Comparison Dashboard**: Compares metrics between old and new versions
- **Error Tracking Dashboard**: Focuses on error rates and types during deployment
- **SLA Compliance Dashboard**: Monitors SLA metrics during and after deployment

These dashboards should be accessible to all stakeholders involved in the deployment to provide visibility into the process.

### 9.4 Post-Deployment Analysis

After deployment, conduct analysis to evaluate success and identify improvements:

- **Deployment Duration**: How long did the deployment take?
- **Error Rate Changes**: Did error rates increase or decrease?
- **Performance Impact**: Did response times change?
- **Resource Utilization**: Did resource usage change?
- **User Impact**: Were any users affected?

Document findings and identify action items for process improvement.

## 10. Troubleshooting

### 10.1 Infrastructure Deployment Issues

Common issues with infrastructure deployment and their solutions:

| Issue | Symptoms | Resolution |
| ----- | -------- | ---------- |
| Terraform state lock | "Error acquiring the state lock" | `terraform force-unlock <ID>` |
| Resource conflicts | "Error creating resource, already exists" | Check for resources created outside Terraform |
| Permission issues | "AccessDenied" errors | Verify IAM permissions |
| Dependency failures | "DependencyViolation" errors | Check resource dependencies |

Steps for general troubleshooting:
1. Check Terraform logs
2. Review AWS CloudTrail events
3. Validate IAM permissions
4. Verify resource quotas

### 10.2 Container Build Issues

Common issues with container builds and their solutions:

| Issue | Symptoms | Resolution |
| ----- | -------- | ---------- |
| Build failures | CI pipeline fails at build stage | Check build logs, fix code issues |
| Image push failures | "Access denied" when pushing to ECR | Verify ECR permissions |
| Security scan failures | Trivy reports critical vulnerabilities | Update dependencies, apply patches |
| Caching issues | Slow builds, unexpected behavior | Clear caches, rebuild from scratch |

Steps for debugging Docker builds:
1. Run the build locally with `docker build`
2. Check Docker daemon logs
3. Verify Dockerfile syntax and dependencies
4. Test with minimal base images

### 10.3 Kubernetes Deployment Issues

Common issues with Kubernetes deployments and their solutions:

| Issue | Symptoms | Resolution |
| ----- | -------- | ---------- |
| Pod failures | Pods in CrashLoopBackOff | Check container logs, fix application issues |
| ImagePullBackOff | Pods can't pull images | Verify image exists, check pull secrets |
| Resource constraints | Pods in Pending state | Check node resources, adjust requests/limits |
| Service discovery | Services not reachable | Check selectors, endpoints, network policies |

Commands for Kubernetes troubleshooting:
```bash
# Check pod status
kubectl get pods -n document-mgmt

# Describe pod to see events
kubectl describe pod <pod-name> -n document-mgmt

# Check pod logs
kubectl logs <pod-name> -n document-mgmt

# Check events
kubectl get events -n document-mgmt --sort-by='.lastTimestamp'

# Check service endpoints
kubectl get endpoints -n document-mgmt

# Debug networking
kubectl exec -it <pod-name> -n document-mgmt -- wget -O- <service-name>
```

### 10.4 CI/CD Pipeline Issues

Common issues with CI/CD pipelines and their solutions:

| Issue | Symptoms | Resolution |
| ----- | -------- | ---------- |
| Authentication failures | "Permission denied" errors | Update secrets, verify credentials |
| Timeout issues | Jobs exceed maximum runtime | Optimize workflows, use caching |
| Resource exhaustion | Jobs fail with resource errors | Adjust resource allocations |
| Configuration errors | Invalid workflow configuration | Validate YAML syntax, check actions versions |

Steps for debugging CI/CD issues:
1. Review workflow logs in GitHub Actions
2. Test workflow locally using `act`
3. Validate configuration using workflow validators
4. Break down complex workflows into smaller steps

## 11. Maintenance Procedures

### 11.1 Kubernetes Version Upgrades

Process for upgrading Kubernetes clusters:

1. **Prepare for Upgrade**:
   - Review release notes for the target version
   - Verify application compatibility
   - Update kubectl client

2. **Update EKS Control Plane**:
   ```bash
   # Update control plane
   aws eks update-cluster-version \
     --name document-mgmt-${ENV} \
     --kubernetes-version 1.25
   
   # Check update status
   aws eks describe-update \
     --name document-mgmt-${ENV} \
     --update-id <update-id>
   ```

3. **Update Node Groups**:
   ```bash
   # Update node groups after control plane upgrade
   aws eks update-nodegroup-version \
     --cluster-name document-mgmt-${ENV} \
     --nodegroup-name document-mgmt-${ENV}-nodes
   ```

4. **Validate Upgrade**:
   - Check node versions
   - Verify control plane health
   - Test application functionality

### 11.2 Database Maintenance

Procedures for database maintenance:

1. **Parameter Group Updates**:
   ```bash
   # Create new parameter group or update existing
   aws rds modify-db-parameter-group \
     --db-parameter-group-name document-mgmt-${ENV} \
     --parameters "ParameterName=max_connections,ParameterValue=200,ApplyMethod=pending-reboot"
   
   # Apply parameter group
   aws rds modify-db-instance \
     --db-instance-identifier document-mgmt-${ENV} \
     --db-parameter-group-name document-mgmt-${ENV} \
     --apply-immediately
   ```

2. **Minor Version Updates**:
   ```bash
   # Update RDS instance version
   aws rds modify-db-instance \
     --db-instance-identifier document-mgmt-${ENV} \
     --engine-version <new-version> \
     --apply-immediately
   ```

3. **Maintenance Window**:
   ```bash
   # Set maintenance window
   aws rds modify-db-instance \
     --db-instance-identifier document-mgmt-${ENV} \
     --preferred-maintenance-window "sun:02:00-sun:04:00"
   ```

### 11.3 Security Patching

Process for applying security patches:

1. **Container Image Updates**:
   - Update base images in Dockerfiles
   - Rebuild and deploy updated images

2. **Dependency Updates**:
   ```bash
   # Update Go dependencies
   go get -u ./...
   go mod tidy
   
   # Regenerate go.sum
   go mod verify
   ```

3. **OS Patching for Nodes**:
   ```bash
   # Rotate nodes to apply latest AMI updates
   aws eks update-nodegroup-version \
     --cluster-name document-mgmt-${ENV} \
     --nodegroup-name document-mgmt-${ENV}-nodes \
     --force-update
   ```

### 11.4 Maintenance Windows

Scheduled maintenance windows for different environments:

| Environment | Scheduled Window | Notification Period | Exceptions |
| ----------- | ---------------- | ------------------ | ---------- |
| Development | Anytime | None | None |
| Staging | Weekdays 8pm-10pm | 24 hours | Critical fixes |
| Production | Sundays 2am-4am | 1 week | Critical security patches |

Maintenance activities should be planned during these windows to minimize disruption.

## 12. References

### 12.1 Related Documentation

- [Monitoring Setup](./monitoring.md): Detailed documentation on monitoring configuration
- [Disaster Recovery](./disaster-recovery.md): Procedures for disaster recovery scenarios
- [Security Documentation](../security/authentication.md): Security-related documentation
- [Development Guidelines](../development/coding-standards.md): Standards for development

### 12.2 External Resources

- **AWS Documentation**:
  - [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
  - [S3 Documentation](https://docs.aws.amazon.com/s3/)
  - [RDS Documentation](https://docs.aws.amazon.com/rds/)

- **Kubernetes Documentation**:
  - [Kubernetes Documentation](https://kubernetes.io/docs/)
  - [Kubectl Cheatsheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)

- **Terraform Documentation**:
  - [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
  - [Terraform Best Practices](https://www.terraform-best-practices.com/)

- **GitHub Actions Documentation**:
  - [GitHub Actions Documentation](https://docs.github.com/en/actions)

### 12.3 Command Reference

**Terraform Commands**:
```bash
# Initialize Terraform
terraform init

# Plan changes
terraform plan -var-file=environments/prod/terraform.tfvars -out=tfplan

# Apply changes
terraform apply tfplan

# Destroy resources
terraform destroy -var-file=environments/prod/terraform.tfvars
```

**Kubernetes Commands**:
```bash
# Apply manifests
kubectl apply -f manifest.yaml

# Get resources
kubectl get pods/deployments/services -n namespace

# Describe resource
kubectl describe pod/deployment/service name -n namespace

# Get logs
kubectl logs pod_name -n namespace

# Execute command in pod
kubectl exec -it pod_name -n namespace -- command
```

**AWS CLI Commands**:
```bash
# Update kubeconfig
aws eks update-kubeconfig --name cluster_name --region region

# Describe EKS cluster
aws eks describe-cluster --name cluster_name

# List ECR repositories
aws ecr describe-repositories

# Get ECR login
aws ecr get-login-password | docker login --username AWS --password-stdin account_id.dkr.ecr.region.amazonaws.com
```

**GitHub Actions CLI**:
```bash
# Run workflow locally
act -W .github/workflows/workflow.yml

# List workflows
gh workflow list

# Run workflow
gh workflow run workflow.yml

# View workflow runs
gh run list
```