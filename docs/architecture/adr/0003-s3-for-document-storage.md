# ADR 0003: S3 for Document Storage

## Status

Accepted

## Context

The Document Management Platform requires a secure, scalable, and durable storage solution for documents uploaded by users. The system needs to handle 10,000 document uploads daily (averaging 3MB per document), provide encryption at rest and in transit, maintain strict tenant isolation, and support different storage categories (temporary, permanent, quarantine) for document processing workflows. The technical specifications explicitly require using AWS S3 for document storage.

## Decision

We will use AWS S3 (Simple Storage Service) as the primary storage solution for all document content in the Document Management Platform. Document metadata will be stored separately in PostgreSQL as defined in the data storage strategy, while the actual document binary content will reside in S3 buckets. The Storage Service microservice will provide an abstraction layer over S3, implementing the StorageService interface defined in the domain layer.

## Storage Architecture

The S3 storage architecture will consist of the following components:

### Bucket Structure

We will use three separate S3 buckets to manage documents in different states:

- **Temporary Bucket**: `tenant-temp` - Holds documents during initial upload and processing (virus scanning)
- **Permanent Bucket**: `tenant-docs` - Stores validated documents after successful processing
- **Quarantine Bucket**: `tenant-quarantine` - Isolates documents identified as potentially malicious

### Object Key Structure

Object keys will follow these patterns to ensure proper organization and tenant isolation:

- Temporary: `temp/{tenantId}/{documentId}`
- Permanent: `{tenantId}/{folderId}/{documentId}/{versionId}`
- Quarantine: `quarantine/{tenantId}/{documentId}`

### Lifecycle Policies

The following lifecycle policies will be applied:

- Temporary Bucket: Delete objects after 24 hours (cleanup for abandoned uploads)
- Permanent Bucket: Retain indefinitely (or according to tenant-specific retention policies)
- Quarantine Bucket: Delete objects after 90 days (retention for security analysis)

### Access Patterns

The system will support the following access patterns:

- Direct download via the API (streaming content)
- Presigned URL generation for client-side downloads
- Batch downloads via ZIP archive creation
- Multipart uploads for large documents

## Encryption Strategy

All document content will be encrypted to ensure data security:

### Encryption at Rest

We will use AWS S3 Server-Side Encryption with AWS KMS (SSE-KMS) to encrypt all objects stored in S3. This provides:

- AES-256 encryption for all stored documents
- Centralized key management through AWS KMS
- Audit trail for encryption key usage
- Tenant-specific KMS keys for enhanced isolation

### Encryption in Transit

All communication with S3 will use HTTPS (TLS 1.2+) to ensure encryption in transit. This includes:

- API calls to S3 from the Storage Service
- Presigned URL access by clients
- Internal service-to-service communication

### Key Management

AWS KMS will be used for encryption key management with the following approach:

- Customer Master Keys (CMKs) managed in KMS
- Automatic key rotation on a yearly basis
- IAM policies restricting key usage to authorized services
- CloudTrail logging of all key usage for audit purposes

## Tenant Isolation

Strict tenant isolation will be maintained through multiple mechanisms:

### Path-Based Isolation

All object keys will include the tenant ID as a prefix, ensuring logical separation of tenant data within shared buckets.

### IAM Policies

IAM policies will restrict access to specific path prefixes based on tenant context in authenticated requests.

### Application-Level Enforcement

The Storage Service will enforce tenant context validation on all operations, preventing cross-tenant access.

### Encryption Isolation

For enhanced isolation, tenant-specific KMS keys can be used for encryption, ensuring that even with access to the raw S3 data, cross-tenant access would be prevented.

## Consequences

### Positive

- Highly durable storage (99.999999999% durability) for document content
- Virtually unlimited scalability to handle growing document volumes
- Built-in encryption capabilities for security compliance
- Cost-effective storage with pay-as-you-go pricing model
- Integrated with AWS ecosystem for monitoring, logging, and security
- Support for lifecycle policies to automate document retention
- High availability (99.99%) meeting system uptime requirements
- Presigned URLs for efficient direct downloads without API proxying
- Multipart upload support for large documents
- Versioning capabilities for future document versioning features

### Negative

- AWS vendor lock-in for storage functionality
- Network egress costs for document downloads
- Limited query capabilities requiring separate metadata storage
- Eventual consistency model may require careful handling
- Requires careful IAM policy management for security
- Potential complexity in managing cross-region replication for disaster recovery

## Implementation

The implementation will follow these guidelines:

### Storage Service Interface

Following Clean Architecture principles, we will define a `StorageService` interface in the domain layer with the following operations:

```go
type StorageService interface {
    StoreTemporary(ctx context.Context, tenantID, documentID string, content io.Reader, size int64, contentType string) (string, error)
    StorePermanent(ctx context.Context, tenantID, documentID, versionID, folderID, tempPath string) (string, error)
    MoveToQuarantine(ctx context.Context, tenantID, documentID, tempPath string) (string, error)
    GetDocument(ctx context.Context, storagePath string) (io.ReadCloser, error)
    GetPresignedURL(ctx context.Context, storagePath, fileName string, expirationSeconds int) (string, error)
    DeleteDocument(ctx context.Context, storagePath string) error
    CreateBatchArchive(ctx context.Context, storagePaths, filenames []string) (io.ReadCloser, error)
}
```

### S3 Implementation

The infrastructure layer will provide an S3 implementation of this interface using the AWS SDK for Go:

```go
type s3Storage struct {
    client     *s3.S3
    uploader   *s3manager.Uploader
    downloader *s3manager.Downloader
    config     config.S3Config
}

func NewS3Storage(config config.S3Config) services.StorageService {
    // Initialize AWS session and S3 client
    // Return s3Storage implementation
}
```

### Error Handling

The S3 implementation will handle various error conditions including:

- Network connectivity issues
- Authentication/authorization failures
- Resource not found errors
- Bucket access problems
- Quota or limit exceeded errors

Errors will be translated to domain-specific errors when appropriate.

### Performance Optimization

Performance will be optimized through:

- Connection pooling for S3 clients
- Multipart uploads for large documents
- Appropriate buffer sizes for streaming operations
- Presigned URLs for direct client downloads
- Concurrent operations for batch processing

### Monitoring and Logging

The implementation will include:

- Detailed logging of all storage operations
- Metrics for upload/download volumes and latencies
- Error rate tracking
- Storage usage monitoring
- Integration with AWS CloudWatch for alerts

## Alternatives Considered

### Local File System Storage

Storing documents on the local file system of application servers. Rejected due to lack of durability, scalability limitations, and complexity in managing distributed access.

### Database BLOB Storage

Storing document content as BLOBs in PostgreSQL. Rejected due to performance impact on the database, inefficient use of database resources, and scaling limitations for large document volumes.

### Alternative Cloud Storage (Google Cloud Storage, Azure Blob Storage)

Using storage services from other cloud providers. Rejected due to the explicit requirement for AWS S3 in the technical specifications.

### Object Storage with MinIO

Using MinIO as an S3-compatible storage solution. Could be considered for development environments but rejected for production due to the explicit requirement for AWS S3.

### Content Delivery Network (CDN) Storage

Using a CDN as the primary storage. Rejected as CDNs are optimized for content delivery rather than primary storage, though CloudFront may be integrated with S3 for efficient delivery.

## References

- AWS S3 Documentation (https://docs.aws.amazon.com/s3/)
- AWS KMS Documentation (https://docs.aws.amazon.com/kms/)
- Technical Specifications Section 2.4.1: Technical Constraints
- Technical Specifications Section 5.3.2: Data Storage Solution Rationale
- Technical Specifications Section 6.3: Storage Service Design
- ADR-0001: Use Clean Architecture
- ADR-0002: Use Microservices