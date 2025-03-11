# Coding Standards for Document Management Platform

## Introduction

This document outlines the coding standards and best practices for the Document Management Platform. These standards ensure consistency, maintainability, and quality across the codebase. All developers working on the project are expected to follow these guidelines.

The Document Management Platform is built using Go (Golang) and follows Clean Architecture and Domain-Driven Design principles. These coding standards are designed to align with these architectural choices while leveraging Go's strengths and idioms.

## Code Organization

The Document Management Platform follows Clean Architecture principles with a clear separation of concerns. Each microservice is organized into layers with specific responsibilities and dependencies.

### Project Structure

The project follows this directory structure:

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

### Package Organization

- Each package should have a single, well-defined responsibility
- Package names should be short, concise, and descriptive
- Avoid package names that are too generic (e.g., `util`, `common`)
- Organize packages by domain concept, not by technical function
- Keep package dependencies clean and avoid circular dependencies
- Follow the dependency rule of Clean Architecture: dependencies point inward

Example of good package organization:

```go
// domain/models/document.go - Domain entity
package models

// domain/repositories/document_repository.go - Repository interface
package repositories

// application/usecases/document_usecase.go - Use case implementation
package usecases

// infrastructure/persistence/postgres/document_repository.go - Repository implementation
package postgres

// api/handlers/document_handler.go - API handler
package handlers
```

### File Organization

- Each file should focus on a single responsibility
- Keep files to a reasonable size (generally under 500 lines)
- Group related functionality in the same file
- Place interfaces and their implementations in separate files
- Name files descriptively based on their content
- Use suffixes to indicate file purpose (e.g., `_test.go`, `_mock.go`)

Example of good file organization:

```
domain/models/document.go           # Document entity
domain/models/document_metadata.go  # Document metadata value object
domain/repositories/document_repository.go  # Document repository interface
infrastructure/persistence/postgres/document_repository.go  # PostgreSQL implementation
infrastructure/persistence/postgres/document_repository_test.go  # Tests
```

### Clean Architecture Layers

Follow the Clean Architecture layers and dependency rules:

1. **Domain Layer** (innermost)
   - Contains business entities and logic
   - Has no dependencies on other layers
   - Defines interfaces that outer layers implement

2. **Application Layer**
   - Contains use cases that orchestrate domain entities
   - Depends only on the domain layer
   - Defines interfaces for infrastructure services

3. **Infrastructure Layer**
   - Implements interfaces defined in domain and application layers
   - Contains adapters for external services, databases, etc.
   - Depends on domain and application layers

4. **API Layer** (outermost)
   - Contains delivery mechanisms (HTTP handlers, etc.)
   - Depends on application and domain layers
   - Handles request/response formatting and validation

Dependencies must always point inward. Outer layers can depend on inner layers, but inner layers cannot depend on outer layers.

## Naming Conventions

Consistent naming is crucial for code readability and maintainability. Follow these naming conventions throughout the codebase.

### General Principles

- Use meaningful, descriptive names
- Prioritize clarity over brevity
- Be consistent with naming patterns
- Follow Go conventions (CamelCase, not snake_case)
- Use standard abbreviations consistently
- Avoid Hungarian notation
- Avoid overly long names (generally under 30 characters)

### Package Names

- Use short, lowercase, single-word names
- Avoid underscores or mixed case
- Choose descriptive, unique names
- Avoid generic names like `util` or `common`
- Use plural form for packages containing multiple similar items

Good examples:
```go
package models
package repositories
package handlers
package postgres
```

Bad examples:
```go
package documentModels  // Mixed case
package document_repository  // Underscores
package util  // Too generic
```

### File Names

- Use snake_case for file names
- Name files after their primary content
- Use suffixes to indicate purpose (_test.go, _mock.go)
- Group related functionality with consistent prefixes

Good examples:
```
document.go
document_metadata.go
document_repository.go
document_repository_test.go
```

### Variable Names

- Use camelCase for variable names
- Start with lowercase letter
- Choose descriptive names that indicate purpose
- Use short names for limited scopes, longer names for wider scopes
- Use plural forms for collections/slices/arrays

Good examples:
```go
var documentID string
var userCount int
var documents []Document
var i int  // Only for very short loops
```

Bad examples:
```go
var DocumentID string  // Starts with uppercase
var d string  // Too short and unclear
var number_of_users int  // Uses snake_case
```

### Constant Names

- Use camelCase for package-level constants
- Use ALL_CAPS with underscores for unchanging values

Good examples:
```go
const maxDocumentSize = 100 * 1024 * 1024  // 100MB
const DEFAULT_TIMEOUT = 30 * time.Second
const MAX_RETRY_COUNT = 3
```

### Function and Method Names

- Use camelCase starting with a verb
- Be descriptive about what the function does
- Follow Go conventions (e.g., `len`, `close`)
- Getters don't need `Get` prefix
- Use consistent naming for similar operations

Good examples:
```go
func createDocument(doc *Document) error
func findDocumentByID(id string) (*Document, error)
func (d *Document) validate() error
func (r *documentRepository) Save(doc *models.Document) error
```

### Interface Names

- Use camelCase
- Don't use `I` prefix or `Interface` suffix
- Name after the behavior they represent
- Single-method interfaces often end with `-er` suffix

Good examples:
```go
type DocumentRepository interface {...}
type StorageService interface {...}
type Reader interface {...}
type Writer interface {...}
```

Bad examples:
```go
type IDocumentRepository interface {...}  // 'I' prefix
type DocumentRepositoryInterface interface {...}  // 'Interface' suffix
```

### Struct Names

- Use PascalCase (capitalized camelCase)
- Use nouns or noun phrases
- Be descriptive about what the struct represents
- Implementation of interfaces should indicate the implementation detail

Good examples:
```go
type Document struct {...}
type DocumentMetadata struct {...}
type PostgresDocumentRepository struct {...}
type S3StorageService struct {...}
```

### Error Names

- Use camelCase starting with `err`
- For error variables, use `ErrXxx` format
- For custom error types, use `XxxError` suffix

Good examples:
```go
var ErrDocumentNotFound = errors.New("document not found")
var ErrInvalidDocumentID = errors.New("invalid document ID")
type ValidationError struct {...}
```

### Test Names

- Use the format `TestXxx` for test functions
- Include the function being tested and the scenario
- Use underscores to separate parts for readability
- Be descriptive about the test case and expected outcome

Good examples:
```go
func TestCreateDocument_ValidInput_Success(t *testing.T) {...}
func TestFindDocumentByID_DocumentExists_ReturnsDocument(t *testing.T) {...}
func TestFindDocumentByID_DocumentNotFound_ReturnsError(t *testing.T) {...}
```

## Formatting and Style

Consistent formatting and style improve code readability and maintainability. The Document Management Platform follows standard Go formatting conventions with some additional guidelines.

### Code Formatting

- Use `gofmt` or `goimports` to automatically format code
- Run `make fmt` before committing code
- Configure your editor to format on save if possible
- CI pipeline will verify formatting compliance

The project uses `golangci-lint` with the following formatting linters:
- `gofmt`: Standard Go formatting
- `goimports`: Standard import formatting
- `whitespace`: Whitespace linting
- `stylecheck`: Style checking

### Import Organization

- Group imports into standard library, external packages, and internal packages
- Sort imports alphabetically within each group
- Use blank lines to separate groups
- Use the full import path, not dot imports
- Avoid unused imports

Example:
```go
import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/document-management/backend/domain/models"
	"github.com/document-management/backend/pkg/errors"
)
```

### Line Length

- Aim for a maximum line length of 100 characters
- Break long lines at logical points
- Use line breaks to improve readability
- Indent continuation lines one level

Example:
```go
func LongFunctionNameWithManyParameters(
	ctx context.Context,
	docID string,
	options *DownloadOptions,
	user *models.User,
) (*models.Document, error) {
	// Function body
}
```

### Indentation and Spacing

- Use tabs for indentation (not spaces)
- Use a single blank line to separate logical sections
- Use a single blank line between functions and methods
- Use a single blank line after package declaration and imports
- Use spaces around operators
- No space between function name and opening parenthesis
- No space between parentheses and parameters

Example:
```go
package handlers

import (
	"context"
	"net/http"
)

func HandleRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// First logical section
	user := getUserFromContext(ctx)
	if user == nil {
		return nil, errors.ErrUnauthorized
	}

	// Second logical section
	response, err := processRequest(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}
```

### Comments

- Write comments in complete sentences with proper punctuation
- Use `//` for single-line comments, not `/*...*/`
- Add a space after `//`
- Place comments on their own line before the code they describe
- Keep comments up-to-date when code changes
- Focus on explaining "why", not "what" (the code shows what)
- Use godoc-compatible comments for exported functions and types

Example:
```go
// calculateTotalSize returns the total size of all documents in bytes.  
// It includes both active and archived documents.
func calculateTotalSize(documents []*models.Document) int64 {
	var total int64
	for _, doc := range documents {
		// Skip documents that are marked for deletion
		if doc.Status == models.StatusDeleting {
			continue
		}
		total += doc.Size
	}
	return total
}
```

### Control Structures

- Place opening braces on the same line as the statement
- Use early returns to reduce nesting
- Keep `else` statements on the same line as the closing brace
- Prefer positive conditions for better readability
- Use parentheses to clarify complex conditions

Example:
```go
func processDocument(doc *models.Document) error {
	if doc == nil {
		return errors.ErrInvalidDocument
	}

	if doc.Size > maxDocumentSize {
		return errors.ErrDocumentTooLarge
	}

	if err := validateDocument(doc); err != nil {
		return err
	}

	// Process document
	return nil
}
```

### Function and Method Organization

- Keep functions focused on a single responsibility
- Limit function length (generally under 50 lines)
- Order methods consistently (e.g., constructors first, then core methods, then helpers)
- Group related functions together
- Place exported functions before unexported functions

Example:
```go
// NewDocumentService creates a new document service instance.
func NewDocumentService(repo repositories.DocumentRepository, storage services.StorageService) *DocumentService {
	return &DocumentService{
		repo:    repo,
		storage: storage,
	}
}

// CreateDocument creates a new document.
func (s *DocumentService) CreateDocument(ctx context.Context, doc *models.Document) (string, error) {
	// Implementation
}

// findExistingDocument is a helper function to find an existing document.
func (s *DocumentService) findExistingDocument(ctx context.Context, id string) (*models.Document, error) {
	// Implementation
}
```

## Error Handling

Proper error handling is critical for building robust and maintainable software. The Document Management Platform follows these error handling guidelines.

### Error Types

The project uses several types of errors:

1. **Standard errors**: Simple errors created with `errors.New()` or `fmt.Errorf()`
2. **Custom error types**: Structs that implement the `error` interface
3. **Sentinel errors**: Predefined error values for specific error conditions
4. **Wrapped errors**: Errors that contain additional context

Example:
```go
// Sentinel errors
var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrInvalidDocumentID = errors.New("invalid document ID")
)

// Custom error type
type ValidationError struct {
	Field string
	Value interface{}
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s with value %v: %s", e.Field, e.Value, e.Reason)
}
```

### Error Handling Patterns

- Always check error returns
- Handle errors at the appropriate level
- Don't ignore errors without a good reason
- Use early returns for error conditions
- Wrap errors to add context when crossing package boundaries
- Don't use panics for normal error handling

Example:
```go
func (s *documentService) GetDocument(ctx context.Context, id string) (*models.Document, error) {
	if id == "" {
		return nil, ErrInvalidDocumentID
	}

	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrDocumentNotFound) {
			return nil, err // Return sentinel error directly
		}
		return nil, fmt.Errorf("failed to find document: %w", err) // Wrap other errors
	}

	return doc, nil
}
```

### Error Wrapping

- Use `fmt.Errorf()` with `%w` verb to wrap errors
- Add context that helps diagnose the problem
- Include relevant parameters or state information
- Don't expose sensitive information in error messages
- Use `errors.Is()` and `errors.As()` to check wrapped errors

Example:
```go
func (r *documentRepository) FindByID(ctx context.Context, id string) (*models.Document, error) {
	var doc models.Document
	err := r.db.Where("id = ? AND tenant_id = ?", id, getTenantID(ctx)).First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("database error when finding document %s: %w", id, err)
	}

	return &doc, nil
}
```

### Error Logging

- Log errors at the appropriate level
- Include context information in logs
- Don't log the same error multiple times
- Use appropriate log levels (debug, info, warn, error)
- Log at the point of handling, not at the point of occurrence

Example:
```go
func (h *documentHandler) GetDocument(c *gin.Context) {
	id := c.Param("id")
	doc, err := h.useCase.GetDocument(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		
		h.logger.Error("Failed to get document", 
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, doc)
}
```

### HTTP Error Responses

- Use appropriate HTTP status codes
- Provide consistent error response format
- Don't expose internal error details to clients
- Map domain errors to appropriate HTTP errors
- Include error codes for client-side error handling

Example:
```go
// Error response format
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error mapping function
func mapErrorToHTTP(err error) (int, ErrorResponse) {
	switch {
	case errors.Is(err, ErrDocumentNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    "DOCUMENT_NOT_FOUND",
			Message: "The requested document could not be found",
		}
	case errors.Is(err, ErrInvalidDocumentID):
		return http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_DOCUMENT_ID",
			Message: "The document ID provided is invalid",
		}
	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		}
	}
}
```

## Documentation

Good documentation is essential for maintainability and knowledge sharing. The Document Management Platform follows these documentation guidelines.

### Code Comments

- Use godoc-compatible comments for all exported declarations
- Start comments with the name of the thing being documented
- Write comments in complete sentences with proper punctuation
- Focus on explaining "why" rather than "what"
- Document non-obvious behavior and edge cases
- Keep comments up-to-date when code changes

Example:
```go
// DocumentService provides operations for managing documents.
type DocumentService struct {
	repo    repositories.DocumentRepository
	storage services.StorageService
}

// NewDocumentService creates a new document service with the given repository and storage service.
func NewDocumentService(repo repositories.DocumentRepository, storage services.StorageService) *DocumentService {
	return &DocumentService{
		repo:    repo,
		storage: storage,
	}
}

// CreateDocument creates a new document with the given metadata and content.
// It returns the ID of the created document or an error if the operation fails.
// The document will be stored in a temporary location until virus scanning is complete.
func (s *DocumentService) CreateDocument(ctx context.Context, doc *models.Document, content io.Reader) (string, error) {
	// Implementation
}
```

### Package Documentation

- Add a package comment to each package
- Describe the package's purpose and usage
- Include examples if appropriate
- Place the comment directly before the package clause

Example:
```go
// Package models contains the domain models for the Document Management Platform.
// It defines the core entities and value objects used throughout the system.
package models
```

### API Documentation

- Document all API endpoints using OpenAPI/Swagger
- Include request/response formats, parameters, and error responses
- Provide examples for common use cases
- Keep API documentation in sync with implementation

The API documentation is generated from code annotations and is available at `/swagger/index.html` when running the API service.

### README Files

- Include a README.md in each major directory
- Describe the purpose and contents of the directory
- Provide context for how the components fit together
- Include links to relevant documentation

Example README.md for a package:
```markdown
# Document Service

This package implements the document service, which provides operations for managing documents in the system.

## Components

- `document_service.go`: Main service implementation
- `virus_scanning.go`: Virus scanning integration
- `thumbnail_generator.go`: Thumbnail generation for documents

## Usage

The document service is initialized in `cmd/api/main.go` and injected into the document handler.

## Testing

Run tests with `go test -v ./...`
```

### Architecture Documentation

- Document architectural decisions in Architecture Decision Records (ADRs)
- Keep high-level architecture diagrams up-to-date
- Document system components and their interactions
- Include rationale for significant design decisions

Architecture documentation is maintained in the `docs/architecture/` directory.

## Testing

Testing is a critical part of maintaining code quality. The Document Management Platform follows these testing guidelines. For more detailed testing information, see [testing.md](./testing.md).

### Test Coverage

- Aim for at least 80% code coverage overall
- Critical paths should have 90%+ coverage
- Run tests with coverage reporting: `make coverage`
- Focus on testing business logic and error handling
- Don't chase 100% coverage at the expense of meaningful tests

The CI pipeline enforces minimum coverage thresholds.

### Unit Testing

- Test each function or method in isolation
- Use table-driven tests for multiple test cases
- Mock external dependencies using interfaces
- Use descriptive test names that explain the scenario and expected outcome
- Follow the Arrange-Act-Assert pattern

Example:
```go
func TestDocumentService_CreateDocument_ValidInput(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewDocumentRepository(t)
	mockStorage := mocks.NewStorageService(t)
	service := NewDocumentService(mockRepo, mockStorage)
	
	doc := &models.Document{Name: "test.pdf", ContentType: "application/pdf"}
	content := strings.NewReader("test content")
	
	mockStorage.On("StoreTemporary", mock.Anything, mock.Anything, mock.Anything).Return("temp-location", nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return("doc-123", nil)
	
	// Act
	id, err := service.CreateDocument(context.Background(), doc, content)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "doc-123", id)
	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}
```

### Integration Testing

- Test interactions between components
- Use test containers for external dependencies
- Focus on testing repository implementations, API handlers, etc.
- Clean up test data after tests
- Use a separate test database

Integration tests are located in the `test/integration/` directory.

### End-to-End Testing

- Test complete workflows from API to database
- Verify system behavior from the user's perspective
- Use realistic test data
- Test happy paths and error scenarios
- Include performance and security considerations

End-to-end tests are located in the `test/e2e/` directory.

### Mocking

- Use mockery to generate mocks from interfaces
- Define clear expectations for mock behavior
- Verify that all expected calls were made
- Don't mock the system under test
- Keep mocks simple and focused

Generate mocks with `make generate-mocks`.

### Test Fixtures

- Use test fixtures for complex test data
- Store fixtures in the `test/testdata/` directory
- Use helper functions to load fixtures
- Keep fixtures maintainable and up-to-date
- Document the purpose and structure of fixtures

Example:
```go
func loadDocumentFixture(t *testing.T, name string) *models.Document {
	path := filepath.Join("testdata", "documents", name+".json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	
	var doc models.Document
	err = json.Unmarshal(data, &doc)
	require.NoError(t, err)
	
	return &doc
}
```

## Performance

Performance is a key consideration for the Document Management Platform, which needs to handle 10,000 document uploads daily and provide fast search capabilities.

### General Guidelines

- Write efficient code, but prioritize readability and maintainability
- Optimize only after profiling identifies bottlenecks
- Document performance considerations and trade-offs
- Include performance tests for critical paths
- Consider resource usage (CPU, memory, network, disk)

The system must meet these performance requirements:
- API response time under 2 seconds
- Document processing time under 5 minutes
- Search queries under 2 seconds

### Memory Management

- Be mindful of memory allocations, especially in hot paths
- Use appropriate data structures for the task
- Consider using sync.Pool for frequently allocated objects
- Avoid unnecessary copying of large data
- Use streaming for large file operations

Example:
```go
// Good: Stream file directly to storage
func (s *storageService) StoreDocument(ctx context.Context, key string, content io.Reader) error {
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		Body:   content,
	})
	return err
}

// Bad: Load entire file into memory
func (s *storageService) StoreDocument(ctx context.Context, key string, content io.Reader) error {
	// Don't do this for large files!
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}
```

### Database Operations

- Use indexes appropriately
- Write efficient queries
- Use pagination for large result sets
- Consider query caching for frequent queries
- Use connection pooling
- Monitor query performance

Example:
```go
// Good: Use pagination and efficient querying
func (r *documentRepository) FindByFolder(ctx context.Context, folderID string, page, pageSize int) ([]*models.Document, int64, error) {
	var docs []*models.Document
	var total int64
	
	db := r.db.Where("folder_id = ? AND tenant_id = ?", folderID, getTenantID(ctx))
	
	// Get total count
	if err := db.Model(&models.Document{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Get paginated results
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&docs).Error; err != nil {
		return nil, 0, err
	}
	
	return docs, total, nil
}
```

### Concurrency

- Use goroutines appropriately for concurrent operations
- Be careful with shared state and race conditions
- Use proper synchronization (mutex, channels, etc.)
- Consider using worker pools for CPU-intensive tasks
- Use context for cancellation and timeouts

Example:
```go
// Process documents concurrently with a worker pool
func processDocuments(ctx context.Context, docs []*models.Document) error {
	concurrency := 5
	semaphore := make(chan struct{}, concurrency)
	errCh := make(chan error, len(docs))
	
	for _, doc := range docs {
		semaphore <- struct{}{} // Acquire semaphore
		
		go func(d *models.Document) {
			defer func() { <-semaphore }() // Release semaphore
			
			err := processDocument(ctx, d)
			errCh <- err
		}(doc)
	}
	
	// Wait for all goroutines to finish
	for i := 0; i < concurrency; i++ {
		semaphore <- struct{}{}
	}
	
	// Check for errors
	for i := 0; i < len(docs); i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	
	return nil
}
```

### Caching

- Use caching for frequently accessed data
- Implement appropriate cache invalidation
- Consider cache expiration policies
- Use distributed caching (Redis) for shared data
- Document caching strategies

Example:
```go
// Use Redis for caching document metadata
func (r *documentRepository) FindByID(ctx context.Context, id string) (*models.Document, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("document:%s", id)
	var doc models.Document
	
	// Check cache
	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache hit
		err = json.Unmarshal([]byte(cachedData), &doc)
		if err == nil {
			return &doc, nil
		}
	}
	
	// Cache miss, get from database
	err = r.db.Where("id = ? AND tenant_id = ?", id, getTenantID(ctx)).First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	
	// Store in cache
	cachedData, err = json.Marshal(doc)
	if err == nil {
		r.cache.Set(ctx, cacheKey, string(cachedData), 15*time.Minute)
	}
	
	return &doc, nil
}
```

### Profiling and Benchmarking

- Use Go's built-in profiling tools (pprof)
- Write benchmarks for performance-critical code
- Regularly run performance tests
- Monitor performance metrics in production
- Document performance characteristics

Example benchmark:
```go
func BenchmarkDocumentService_CreateDocument(b *testing.B) {
	// Setup
	service := setupDocumentService()
	doc := &models.Document{Name: "test.pdf", ContentType: "application/pdf"}
	content := strings.NewReader("test content")
	
	// Reset timer before the loop
	b.ResetTimer()
	
	// Run the benchmark
	for i := 0; i < b.N; i++ {
		_, err := service.CreateDocument(context.Background(), doc, content)
		if err != nil {
			b.Fatal(err)
		}
		
		// Reset content reader for next iteration
		content.Reset()
	}
}
```

## Security

Security is a critical concern for the Document Management Platform, which handles sensitive documents and requires strict tenant isolation.

### General Security Guidelines

- Follow the principle of least privilege
- Validate all input data
- Use secure defaults
- Keep dependencies up-to-date
- Follow secure coding practices
- Run security scanning tools regularly

The system must meet these security requirements:
- Complete tenant isolation
- Document encryption at rest and in transit
- Virus scanning for all uploaded documents
- Role-based access control

### Authentication and Authorization

- Use JWT tokens for authentication
- Validate tokens on every request
- Extract tenant context from tokens
- Implement role-based access control
- Check permissions for all operations
- Use secure token handling practices

Example:
```go
// Middleware to extract tenant context from JWT
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := extractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
		
		// Extract tenant ID from token
		tenantID, ok := token.Claims.(jwt.MapClaims)["tenant_id"].(string)
		if !ok || tenantID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant context"})
			c.Abort()
			return
		}
		
		// Set tenant ID in context
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// Helper to check permissions
func (s *documentService) checkPermission(ctx context.Context, docID string, permission string) error {
	userID, ok := ctx.Value("user_id").(string)
	if !ok || userID == "" {
		return ErrUnauthorized
	}
	
	hasPermission, err := s.permissionRepo.CheckPermission(ctx, userID, docID, permission)
	if err != nil {
		return err
	}
	
	if !hasPermission {
		return ErrForbidden
	}
	
	return nil
}
```

### Input Validation

- Validate all input data, including API requests and file uploads
- Use strong typing and explicit validation
- Sanitize data before using it in queries or commands
- Implement proper error handling for validation failures
- Document validation rules

Example:
```go
// Validate document creation request
func validateCreateDocumentRequest(req *dto.CreateDocumentRequest) error {
	if req.Name == "" {
		return &ValidationError{Field: "name", Reason: "cannot be empty"}
	}
	
	if req.ContentType == "" {
		return &ValidationError{Field: "content_type", Reason: "cannot be empty"}
	}
	
	if !isAllowedContentType(req.ContentType) {
		return &ValidationError{Field: "content_type", Reason: "not allowed"}
	}
	
	if req.FolderID != "" {
		if !isValidUUID(req.FolderID) {
			return &ValidationError{Field: "folder_id", Reason: "invalid format"}
		}
	}
	
	return nil
}

// Check if content type is allowed
func isAllowedContentType(contentType string) bool {
	allowedTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		// Add other allowed types
	}
	
	for _, allowed := range allowedTypes {
		if contentType == allowed {
			return true
		}
	}
	
	return false
}
```

### Data Protection

- Encrypt sensitive data at rest
- Use TLS for all communications
- Implement proper key management
- Follow the principle of data minimization
- Implement secure data deletion

Example:
```go
// Store document with encryption
func (s *storageService) StoreDocument(ctx context.Context, key string, content io.Reader) error {
	// Use S3 server-side encryption with KMS
	_, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(s.bucketName),
		Key:                  aws.String(key),
		Body:                 content,
		ServerSideEncryption: aws.String("aws:kms"),
		SSEKMSKeyId:          aws.String(s.kmsKeyID),
	})
	return err
}
```

### Tenant Isolation

- Enforce tenant isolation at all layers
- Include tenant ID in all database queries
- Use tenant-specific storage paths
- Validate tenant context in all operations
- Implement proper error handling for tenant validation

Example:
```go
// Repository with tenant isolation
func (r *documentRepository) FindByID(ctx context.Context, id string) (*models.Document, error) {
	tenantID, err := getTenantID(ctx)
	if err != nil {
		return nil, err
	}
	
	var doc models.Document
	err = r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	
	return &doc, nil
}

// Helper to get tenant ID from context
func getTenantID(ctx context.Context) (string, error) {
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return "", ErrMissingTenantContext
	}
	return tenantID, nil
}
```

### Virus Scanning

- Scan all uploaded documents for viruses
- Quarantine infected documents
- Implement proper error handling for scanning failures
- Document virus scanning process
- Monitor scanning results

Example:
```go
// Scan document for viruses
func (s *virusScanningService) ScanDocument(ctx context.Context, path string) (bool, error) {
	// Download document from temporary storage
	file, err := s.storage.GetTemporaryFile(ctx, path)
	if err != nil {
		return false, fmt.Errorf("failed to get file for scanning: %w", err)
	}
	defer file.Close()
	
	// Scan document
	result, err := s.scanner.ScanReader(file)
	if err != nil {
		return false, fmt.Errorf("virus scanning failed: %w", err)
	}
	
	// Check result
	if result.Infected {
		s.logger.Warn("Virus detected in document",
			zap.String("path", path),
			zap.String("virus_name", result.VirusName))
		return true, nil
	}
	
	return false, nil
}
```

### Security Testing

- Include security tests in the test suite
- Use static analysis tools (gosec)
- Perform regular security audits
- Test for common vulnerabilities
- Document security testing procedures

Example security test:
```go
func TestDocumentService_GetDocument_TenantIsolation(t *testing.T) {
	// Arrange
	service := setupDocumentService()
	
	// Create documents for two different tenants
	doc1ID := createTestDocument(t, service, "tenant1")
	doc2ID := createTestDocument(t, service, "tenant2")
	
	// Act & Assert - Tenant 1 can access their document
	ctx1 := contextWithTenant("tenant1")
	doc1, err := service.GetDocument(ctx1, doc1ID)
	assert.NoError(t, err)
	assert.NotNil(t, doc1)
	
	// Act & Assert - Tenant 1 cannot access tenant 2's document
	_, err = service.GetDocument(ctx1, doc2ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrDocumentNotFound))
}
```

## Code Review

Code reviews are an essential part of maintaining code quality and knowledge sharing. The Document Management Platform follows these code review guidelines.

### Code Review Process

- All code changes must be reviewed before merging
- Create a pull request for each change
- Provide a clear description of the changes
- Link to relevant issues or requirements
- Address all review comments
- Obtain approval before merging

The CI pipeline must pass before a pull request can be merged.

### Review Checklist

When reviewing code, consider the following aspects:

- **Functionality**: Does the code work as intended?
- **Architecture**: Does the code follow Clean Architecture principles?
- **Code Quality**: Is the code well-written, maintainable, and efficient?
- **Testing**: Are there appropriate tests with good coverage?
- **Security**: Are there any security concerns?
- **Performance**: Are there any performance issues?
- **Documentation**: Is the code well-documented?
- **Error Handling**: Is error handling appropriate and consistent?
- **Consistency**: Does the code follow project conventions and standards?

### Providing Feedback

- Be constructive and respectful
- Focus on the code, not the person
- Explain the reasoning behind suggestions
- Provide examples when appropriate
- Distinguish between required changes and suggestions
- Acknowledge good practices and improvements

Example feedback:
```
- The error handling here could be improved by wrapping the error with context about the operation being performed.
- Consider using a more descriptive variable name than `d` to improve readability.
- Great job on the comprehensive test coverage for this function!
```

### Responding to Feedback

- Address all feedback promptly
- Explain your reasoning if you disagree
- Ask for clarification if needed
- Make requested changes or explain why not
- Thank reviewers for their input
- Learn from the feedback for future code

Example response:
```
- Updated the error handling to include more context.
- Renamed the variable to `document` for clarity.
- Thanks for the feedback on the tests!
```

### Automated Code Reviews

The project uses several automated tools to assist with code reviews:

- **golangci-lint**: Static code analysis
- **gosec**: Security-focused static analysis
- **go vet**: Go code correctness checker
- **go test**: Automated tests
- **go test -race**: Race condition detection

These tools run automatically in the CI pipeline for each pull request.

## Conclusion

These coding standards provide a foundation for building a high-quality, maintainable Document Management Platform. By following these guidelines, we ensure consistency across the codebase and make it easier for team members to collaborate effectively.

Remember that these standards are not set in stone. They should evolve as the project grows and as we learn from experience. If you have suggestions for improving these standards, please discuss them with the team.

## References

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design by Eric Evans](https://domainlanguage.com/ddd/)
- [Project README](../README.md)
- [Testing Guidelines](./testing.md)
- [Clean Architecture ADR](../architecture/adr/0001-use-clean-architecture.md)
- [Microservices ADR](../architecture/adr/0002-use-microservices.md)