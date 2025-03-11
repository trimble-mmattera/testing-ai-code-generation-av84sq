# Testing Strategy

This document outlines the comprehensive testing strategy for the Document Management Platform, covering unit testing, integration testing, end-to-end testing, performance testing, and security testing approaches.

## Testing Strategy Overview

The Document Management Platform implements a comprehensive testing strategy to ensure code quality, functionality, security, and performance. This document outlines the testing approach, tools, and processes used throughout the development lifecycle.

### Testing Principles

- **Shift Left**: Testing begins early in the development process
- **Automation First**: All tests should be automated where possible
- **Comprehensive Coverage**: Multiple testing types to ensure quality
- **Security by Design**: Security testing integrated into the development process
- **Performance Awareness**: Performance testing to meet SLAs

### Testing Types

The platform employs the following types of tests:

1. **Unit Tests**: Verify individual components in isolation
2. **Integration Tests**: Verify interactions between components
3. **End-to-End Tests**: Verify complete workflows
4. **Performance Tests**: Verify system performance under load
5. **Security Tests**: Verify system security and vulnerability protection

### Quality Metrics

The following quality metrics are tracked:

- **Code Coverage**: Minimum 80% overall, 90% for critical paths
- **Test Success Rate**: 100% pass rate required for all tests
- **Performance SLAs**: API response time < 2 seconds, document processing < 5 minutes
- **Security Compliance**: No critical vulnerabilities, compliance with SOC2 and ISO27001

## Unit Testing

Unit tests verify the correctness of individual components in isolation by mocking dependencies.

### Framework and Tools

- **Go Testing Package**: Standard library testing framework
- **Testify**: Assertions, mocks, and test suite functionality
- **Mockery**: Automatic mock generation for interfaces

### Directory Structure

Unit tests are located alongside the code they test with the naming convention `*_test.go`. For example:

```
/domain
  /models
    document.go
    document_test.go
/application
  /usecases
    document_usecase.go
    document_usecase_test.go
```

### Mocking Strategy

The testing approach leverages Go's interface-based design to facilitate effective mocking:

1. Define interfaces for all dependencies
2. Generate mocks using mockery
3. Inject mocks during tests
4. Set expectations on mock method calls
5. Verify expectations after test execution

### Test Naming Conventions

Tests follow the naming convention `TestFunctionName_Scenario_ExpectedBehavior` or `TestFunctionName_When_Then`. For example:

- `TestUploadDocument_ValidFile_ReturnsDocumentID`
- `TestUploadDocument_WhenFileTooLarge_ThenReturnsError`

### Running Unit Tests

Unit tests can be run using the following command:

```bash
./scripts/run-tests.sh -t unit
```

Or with coverage reporting:

```bash
./scripts/run-tests.sh -t unit -c
```

### Example Unit Test

```go
func TestUploadDocument_ValidDocument_ReturnsDocumentID(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewDocumentRepository(t)
    mockStorage := mocks.NewStorageService(t)
    mockSecurity := mocks.NewSecurityService(t)
    
    documentService := service.NewDocumentService(mockRepo, mockStorage, mockSecurity)
    
    validDocument := domain.Document{
        Name: "test.pdf",
        ContentType: "application/pdf",
        Size: 1024,
        TenantID: "tenant-123",
    }
    
    mockStorage.On("StoreTemporary", mock.Anything, mock.Anything).Return("temp-location", nil)
    mockSecurity.On("QueueForScanning", mock.Anything).Return(nil)
    mockRepo.On("Create", mock.Anything).Return("doc-123", nil)
    
    // Act
    docID, err := documentService.UploadDocument(context.Background(), validDocument, bytes.NewReader([]byte("test content")))
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "doc-123", docID)
    mockRepo.AssertExpectations(t)
    mockStorage.AssertExpectations(t)
    mockSecurity.AssertExpectations(t)
}
```

## Integration Testing

Integration tests verify the correct interaction between components and external dependencies.

### Framework and Tools

- **Go Testing Package**: Standard library testing framework
- **Testify**: Assertions and test suite functionality
- **Testcontainers**: Containerized dependencies for testing
- **Docker Compose**: Local environment setup

### Directory Structure

Integration tests are located in the `/test/integration` directory, organized by component:

```
/test
  /integration
    document_test.go
    folder_test.go
    search_test.go
    auth_test.go
```

### Test Environment

Integration tests use containerized dependencies:

- **PostgreSQL**: For metadata storage
- **MinIO**: S3-compatible storage for documents
- **Elasticsearch**: For search functionality
- **Redis**: For caching

These dependencies are automatically started using Docker Compose when running integration tests.

### Running Integration Tests

Integration tests can be run using the following command:

```bash
./scripts/run-tests.sh -t integration
```

Or with coverage reporting:

```bash
./scripts/run-tests.sh -t integration -c
```

### Example Integration Test

```go
func TestDocumentRepositorySuite_TestCreateDocument(t *testing.T) {
    suite.Run(t, new(DocumentRepositorySuite))
}

type DocumentRepositorySuite struct {
    suite.Suite
    repo repositories.DocumentRepository
    ctx  context.Context
}

func (s *DocumentRepositorySuite) SetupSuite() {
    // Initialize database connection
    dbConfig := config.DatabaseConfig{
        Host:     "localhost",
        Port:     5432,
        User:     "testuser",
        Password: "testpassword",
        Database: "testdb",
        SSLMode:  "disable",
    }
    
    err := postgres.Init(dbConfig)
    s.Require().NoError(err)
    
    db := postgres.GetDB()
    err = postgres.Migrate(db)
    s.Require().NoError(err)
    
    s.repo = postgres.NewDocumentRepository(db)
    s.ctx = context.Background()
}

func (s *DocumentRepositorySuite) TearDownSuite() {
    postgres.Close()
}

func (s *DocumentRepositorySuite) TestCreateDocument() {
    // Create a test document
    doc := models.NewDocument("test.pdf", "application/pdf", 1024, "tenant-test-1", "folder-test-1", "user-test-1")
    
    // Save to repository
    docID, err := s.repo.Create(s.ctx, doc)
    s.Require().NoError(err)
    s.Require().NotEmpty(docID)
    
    // Retrieve and verify
    retrievedDoc, err := s.repo.GetByID(s.ctx, docID, "tenant-test-1")
    s.Require().NoError(err)
    s.Equal(doc.Name, retrievedDoc.Name)
    s.Equal(doc.ContentType, retrievedDoc.ContentType)
    s.Equal(doc.Size, retrievedDoc.Size)
    s.Equal(doc.TenantID, retrievedDoc.TenantID)
}
```

## End-to-End Testing

End-to-end tests validate complete user workflows across the entire system.

### Framework and Tools

- **Go Testing Package**: Standard library testing framework
- **Testify**: Assertions and test suite functionality
- **Docker Compose**: Full environment setup

### Directory Structure

End-to-end tests are located in the `/test/e2e` directory, organized by workflow:

```
/test
  /e2e
    document_flow_test.go
    folder_flow_test.go
    search_flow_test.go
```

### Test Environment

End-to-end tests use a complete environment with all dependencies:

- **PostgreSQL**: For metadata storage
- **MinIO**: S3-compatible storage for documents
- **Elasticsearch**: For search functionality
- **Redis**: For caching
- **ClamAV**: For virus scanning

These dependencies are automatically started using Docker Compose when running E2E tests.

### Test Scenarios

End-to-end tests cover the following key workflows:

1. **Document Upload Flow**: Upload document, scan, index, verify availability
2. **Document Search Flow**: Upload documents, search by content and metadata
3. **Folder Management Flow**: Create folders, organize documents, list contents
4. **Batch Operations Flow**: Upload multiple documents, batch download
5. **Tenant Isolation Flow**: Verify tenant isolation across operations

### Running End-to-End Tests

End-to-end tests can be run using the following command:

```bash
./scripts/run-tests.sh -t e2e
```

### Example End-to-End Test

```go
func TestDocumentFlow(t *testing.T) {
    suite.Run(t, new(DocumentFlowTestSuite))
}

type DocumentFlowTestSuite struct {
    suite.Suite
    documentRepo        repositories.DocumentRepository
    documentService     services.DocumentService
    storageService      services.StorageService
    virusScanningService services.VirusScanningService
    searchService       services.SearchService
    documentUseCase     documentusecase.DocumentUseCase
    testTenantID        string
    testUserID          string
    testFolderID        string
    logger              *zap.Logger
    testDocumentIDs     map[string]string
}

func (s *DocumentFlowTestSuite) SetupSuite() {
    // Initialize test environment
    s.testTenantID = uuid.New().String()
    s.testUserID = uuid.New().String()
    s.testFolderID = uuid.New().String()
    s.logger = logger.NewLogger()
    s.testDocumentIDs = make(map[string]string)
}

func (s *DocumentFlowTestSuite) TestDocumentUploadAndProcessingFlow() {
    // Create test document content
    content := []byte("This is a test document for E2E testing")
    
    // Upload document
    docID, err := s.documentUseCase.UploadDocument(
        context.Background(),
        "test.pdf",
        "application/pdf",
        int64(len(content)),
        s.testFolderID,
        s.testTenantID,
        s.testUserID,
        bytes.NewReader(content),
    )
    s.Require().NoError(err)
    s.Require().NotEmpty(docID)
    
    // Process document with clean scan result
    err = s.documentService.ProcessDocumentScanResult(
        context.Background(),
        docID,
        s.testTenantID,
        services.ScanResultClean,
        "",
    )
    s.Require().NoError(err)
    
    // Wait for processing to complete
    err = s.waitForDocumentProcessing(docID, 30*time.Second)
    s.Require().NoError(err)
    
    // Verify document is available
    doc, err := s.documentUseCase.GetDocument(
        context.Background(),
        docID,
        s.testTenantID,
        s.testUserID,
    )
    s.Require().NoError(err)
    s.Equal(models.DocumentStatusAvailable, doc.Status)
    
    // Verify document can be downloaded
    downloadContent, filename, err := s.documentUseCase.DownloadDocument(
        context.Background(),
        docID,
        s.testTenantID,
        s.testUserID,
    )
    s.Require().NoError(err)
    s.Equal("test.pdf", filename)
    
    // Read content and verify it matches original
    downloadedBytes, err := io.ReadAll(downloadContent)
    s.Require().NoError(err)
    s.Equal(content, downloadedBytes)
    
    // Verify document is searchable
    searchResults, err := s.documentUseCase.SearchDocumentsByContent(
        context.Background(),
        "test document",
        s.testTenantID,
        s.testUserID,
        utils.NewPagination(1, 10),
    )
    s.Require().NoError(err)
    s.Require().GreaterOrEqual(searchResults.TotalCount, int64(1))
    
    // Store document ID for cleanup
    s.testDocumentIDs[docID] = docID
}
```

## Performance Testing

Performance tests validate that the system meets performance requirements under various load conditions.

### Framework and Tools

- **k6**: Load and performance testing tool
- **Grafana**: Visualization of performance metrics
- **Prometheus**: Metrics collection

### Directory Structure

Performance tests are located in the `/test/performance` directory:

```
/test
  /performance
    /k6
      document_upload_test.js
      document_download_test.js
      document_search_test.js
      batch_operations_test.js
```

### Test Scenarios

Performance tests cover the following scenarios:

1. **Load Testing**: System behavior under expected load
2. **Stress Testing**: System behavior under extreme load
3. **Endurance Testing**: System behavior over extended periods
4. **Spike Testing**: System behavior under sudden load increases

### Performance Metrics

The following metrics are tracked during performance testing:

- **Response Time**: 95th percentile < 2 seconds
- **Throughput**: Requests per second
- **Error Rate**: < 1% under load
- **Resource Utilization**: CPU, memory, disk, network
- **Document Processing Time**: 99% < 5 minutes

### Running Performance Tests

Performance tests can be run using k6:

```bash
k6 run test/performance/k6/document_upload_test.js
```

Or with specific load parameters:

```bash
k6 run --vus 50 --duration 5m test/performance/k6/document_upload_test.js
```

### Example Performance Test

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const documentUploadDuration = new Trend('document_upload_duration');
const documentUploadFailRate = new Rate('document_upload_fail_rate');
const documentUploadCount = new Counter('document_upload_count');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 10 },  // Ramp up to 10 users
    { duration: '3m', target: 10 },  // Stay at 10 users
    { duration: '1m', target: 50 },  // Ramp up to 50 users
    { duration: '3m', target: 50 },  // Stay at 50 users
    { duration: '1m', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    'document_upload_duration': ['p(95)<2000'], // 95% of uploads must complete within 2s
    'document_upload_fail_rate': ['rate<0.01'],  // Less than 1% of uploads can fail
    'http_req_duration': ['p(95)<2000'],        // 95% of requests must complete within 2s
  },
};

// Helper function to get authentication token
function getAuthToken() {
  // In a real test, this would authenticate with the API
  // For this example, we'll use a placeholder token
  return 'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...';
}

// Helper function to generate test document
function generateTestDocument(size) {
  return Array(size).fill('A').join('');
}

// Main test function
export default function() {
  const token = getAuthToken();
  const tenantId = 'performance-test-tenant';
  const userId = 'performance-test-user';
  const folderId = 'performance-test-folder';
  
  // Generate a test document (approximately 10KB)
  const documentContent = generateTestDocument(10240);
  const documentName = `test-${__VU}-${__ITER}.pdf`;
  
  // Prepare multipart request
  const data = {
    file: http.file(documentContent, documentName, 'application/pdf'),
    name: documentName,
    folder_id: folderId,
  };
  
  // Upload document
  const startTime = new Date().getTime();
  const response = http.post(
    'https://api.example.com/api/v1/documents',
    data,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }
  );
  const endTime = new Date().getTime();
  
  // Record metrics
  documentUploadDuration.add(endTime - startTime);
  documentUploadCount.add(1);
  
  // Check response
  const success = check(response, {
    'Upload successful': (r) => r.status === 202,
    'Document ID returned': (r) => r.json('document_id') !== undefined,
  });
  
  if (!success) {
    documentUploadFailRate.add(1);
    console.log(`Upload failed: ${response.status} ${response.body}`);
  } else {
    documentUploadFailRate.add(0);
  }
  
  // Wait between requests
  sleep(1);
}
```

## Security Testing

Security tests validate that the system is secure and complies with security requirements.

### Framework and Tools

- **Trivy**: Container and filesystem vulnerability scanning
- **gosec**: Go security scanner
- **OWASP Dependency Check**: Third-party dependency vulnerability scanning
- **Gitleaks**: Secret detection
- **CodeQL**: Static application security testing

### Security Test Types

The following security test types are implemented:

1. **SAST (Static Application Security Testing)**: Code analysis for security vulnerabilities
2. **DAST (Dynamic Application Security Testing)**: Runtime security testing
3. **Dependency Scanning**: Third-party dependency vulnerability scanning
4. **Container Scanning**: Container image vulnerability scanning
5. **Secret Detection**: Detection of hardcoded secrets
6. **Compliance Scanning**: Verification of compliance with security standards

### Security Test Scenarios

Security tests cover the following scenarios:

1. **Tenant Isolation**: Verification that tenants cannot access each other's data
2. **Authentication Bypass**: Attempts to bypass JWT validation
3. **Authorization Bypass**: Attempts to access unauthorized resources
4. **Malicious File Upload**: Upload of virus-infected files
5. **Data Encryption**: Verification of encryption at rest and in transit

### Running Security Tests

Security tests can be run using the security-scan workflow:

```bash
# Run all security scans
./scripts/security-scan.sh

# Run specific security scan
./scripts/security-scan.sh -t trivy
```

Or through GitHub Actions by manually triggering the security-scan workflow.

### Security Testing in CI/CD

Security testing is integrated into the CI/CD pipeline:

1. **Pre-commit**: Secret detection using git-secrets
2. **Pull Request**: SAST, dependency scanning, container scanning
3. **Merge to Main**: All security scans
4. **Scheduled**: Weekly comprehensive security scans

### Example Security Test

```go
func TestTenantIsolation(t *testing.T) {
    suite.Run(t, new(SecurityTestSuite))
}

type SecurityTestSuite struct {
    suite.Suite
    documentUseCase documentusecase.DocumentUseCase
    tenant1ID       string
    tenant2ID       string
    userID          string
}

func (s *SecurityTestSuite) SetupSuite() {
    // Initialize test environment
    s.tenant1ID = "security-test-tenant-1"
    s.tenant2ID = "security-test-tenant-2"
    s.userID = "security-test-user"
    
    // Set up document use case with dependencies
    // ...
}

func (s *SecurityTestSuite) TestTenantIsolation_DocumentAccess() {
    // Create document for tenant 1
    content := []byte("Confidential document for tenant 1")
    docID, err := s.documentUseCase.UploadDocument(
        context.Background(),
        "confidential.pdf",
        "application/pdf",
        int64(len(content)),
        "folder-id",
        s.tenant1ID,
        s.userID,
        bytes.NewReader(content),
    )
    s.Require().NoError(err)
    
    // Process document
    // ...
    
    // Attempt to access document using tenant 2 credentials
    _, err = s.documentUseCase.GetDocument(
        context.Background(),
        docID,
        s.tenant2ID,
        s.userID,
    )
    
    // Verify access is denied
    s.Require().Error(err)
    s.True(errors.IsAuthorizationError(err), "Expected authorization error for cross-tenant access")
    
    // Attempt to download document using tenant 2 credentials
    _, _, err = s.documentUseCase.DownloadDocument(
        context.Background(),
        docID,
        s.tenant2ID,
        s.userID,
    )
    
    // Verify access is denied
    s.Require().Error(err)
    s.True(errors.IsAuthorizationError(err), "Expected authorization error for cross-tenant access")
}
```

## Test Automation

Test automation ensures that tests are run consistently and reliably as part of the development process.

### CI/CD Integration

Tests are integrated into the CI/CD pipeline using GitHub Actions:

1. **Pre-commit**: Linting, formatting, unit tests
2. **Pull Request**: Unit tests, integration tests, security scans
3. **Merge to Main**: All tests including E2E tests
4. **Deployment**: Smoke tests, security scans

### Test Execution Workflow

The test execution workflow is defined in `.github/workflows/test.yml` and includes the following jobs:

1. **lint**: Run linting checks
2. **unit-tests**: Run unit tests with coverage reporting
3. **integration-tests**: Run integration tests with coverage reporting
4. **e2e-tests**: Run end-to-end tests
5. **coverage-report**: Generate combined coverage report

### Test Scripts

Test automation is facilitated by scripts in the `/scripts` directory:

- **run-tests.sh**: Run different types of tests
- **generate-mock.sh**: Generate mocks for interfaces
- **docker-build.sh**: Build Docker images for testing
- **setup-dev.sh**: Set up development environment

### Test Reporting

Test results are reported in multiple formats:

1. **JUnit XML**: For CI/CD integration
2. **Coverage HTML**: For human-readable coverage reports
3. **Coverage XML**: For integration with code quality tools
4. **Test Logs**: For debugging test failures

### Failed Test Handling

Failed tests are handled as follows:

1. **CI/CD Pipeline**: Failed tests block the pipeline
2. **Test Reports**: Detailed reports are generated for failed tests
3. **Flaky Tests**: Flaky tests are identified and quarantined
4. **Retry Mechanism**: Transient failures can be retried automatically

## Test Environment Management

Test environments are managed to ensure consistent and reliable testing.

### Local Development Environment

Local development environment is set up using Docker Compose:

```bash
# Start development environment
docker-compose -f docker-compose.dev.yml up -d

# Run tests in development environment
./scripts/run-tests.sh
```

### CI/CD Test Environment

CI/CD test environment is set up using GitHub Actions services:

```yaml
services:
  postgres:
    image: postgres:14
    env:
      POSTGRES_USER: testuser
      POSTGRES_PASSWORD: testpassword
      POSTGRES_DB: testdb
    ports:
      - 5432:5432
  minio:
    image: minio/minio
    env:
      MINIO_ACCESS_KEY: minioadmin
      MINIO_SECRET_KEY: minioadmin
    ports:
      - 9000:9000
  elasticsearch:
    image: elasticsearch:8.0.0
    env:
      discovery.type: single-node
      ES_JAVA_OPTS: -Xms512m -Xmx512m
      xpack.security.enabled: false
    ports:
      - 9200:9200
```

### Test Data Management

Test data is managed as follows:

1. **Test Fixtures**: Static test data in `/test/testdata`
2. **Generated Test Data**: Dynamically generated test data
3. **Test Database**: Fresh database for each test run
4. **Test Cleanup**: Automatic cleanup after tests

### Environment Configuration

Test environment configuration is managed using:

1. **Environment Variables**: For runtime configuration
2. **Configuration Files**: For static configuration
3. **Test-Specific Configuration**: In `/config/test.yml`

## Best Practices

The following best practices are followed for testing the Document Management Platform.

### Test-Driven Development

1. **Write Tests First**: Define expected behavior before implementation
2. **Red-Green-Refactor**: Start with failing tests, make them pass, then refactor
3. **Small Increments**: Make small, testable changes

### Test Organization

1. **Arrange-Act-Assert**: Structure tests with clear setup, action, and verification
2. **One Assertion Per Test**: Focus each test on a single behavior
3. **Descriptive Test Names**: Make test names describe the behavior being tested

### Test Maintenance

1. **Keep Tests DRY**: Use helper functions and test fixtures
2. **Avoid Test Interdependence**: Tests should be independent and isolated
3. **Maintain Test Code Quality**: Apply the same quality standards to test code as production code

### Test Coverage

1. **Focus on Critical Paths**: Ensure high coverage of critical functionality
2. **Cover Edge Cases**: Test boundary conditions and error scenarios
3. **Balance Coverage and Value**: Aim for meaningful coverage, not just high percentages

## Troubleshooting

Common issues and solutions for testing the Document Management Platform.

### Common Test Failures

1. **Database Connection Issues**: Ensure PostgreSQL is running and accessible
2. **S3 Connection Issues**: Ensure MinIO is running and accessible
3. **Elasticsearch Connection Issues**: Ensure Elasticsearch is running and accessible
4. **Timeout Issues**: Increase test timeout for slow operations

### Debugging Tests

1. **Verbose Logging**: Run tests with `-v` flag for detailed output
2. **Single Test Execution**: Run a specific test with `-run TestName`
3. **Test Environment Inspection**: Examine test environment state during failures

### CI/CD Issues

1. **Workflow Logs**: Check GitHub Actions workflow logs for details
2. **Test Artifacts**: Download test artifacts for local inspection
3. **Environment Differences**: Check for differences between local and CI environments

## References

### Internal Documentation

- [Testing Strategy](../architecture/testing-strategy.md)
- [CI/CD Pipeline](../operations/ci-cd-pipeline.md)
- [Security Testing](../security/security-testing.md)

### External Resources

- [Go Testing Package Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [k6 Documentation](https://k6.io/docs/)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)