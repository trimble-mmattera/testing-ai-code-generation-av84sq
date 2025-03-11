package s3

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"         // v1.44.0+
	"github.com/aws/aws-sdk-go/aws/credentials" // v1.44.0+
	"github.com/aws/aws-sdk-go/aws/request"  // v1.44.0+
	"github.com/aws/aws-sdk-go/aws/session"  // v1.44.0+
	"github.com/aws/aws-sdk-go/awstesting/mock" // v1.44.0+
	"github.com/aws/aws-sdk-go/service/s3"    // v1.44.0+
	"github.com/aws/aws-sdk-go/service/s3/s3manager" // v1.44.0+
	"github.com/stretchr/testify/assert"     // v1.8.0+
	"github.com/stretchr/testify/require"    // v1.8.0+

	"../../../domain/services"
	"../../../pkg/config"
)

// Test constants
const (
	testTenantID    = "tenant-123"
	testDocumentID  = "doc-123"
	testVersionID   = "v1"
	testFolderID    = "folder-123"
	testContent     = "test document content"
	testContentType = "application/pdf"
)

// Test helper function to create a test S3 configuration
func createTestConfig() config.S3Config {
	return config.S3Config{
		Region:           "us-east-1",
		Endpoint:         "http://localhost:4566",
		AccessKey:        "test",
		SecretKey:        "test",
		Bucket:           "test-bucket",
		TempBucket:       "test-temp-bucket",
		QuarantineBucket: "test-quarantine-bucket",
		ForcePathStyle:   true,
	}
}

// Helper function to create mock S3 client, uploader, and downloader
func createMockS3Client() (*s3.S3, *s3manager.Uploader, *s3manager.Downloader) {
	// Create mock session
	sess := mock.Session
	// Create S3 client, uploader, and downloader
	s3Client := s3.New(sess)
	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)
	
	return s3Client, uploader, downloader
}

// TestNewS3Storage tests the creation of a new S3 storage service
func TestNewS3Storage(t *testing.T) {
	// Create test config
	cfg := createTestConfig()
	
	// Create S3 storage service
	storage, err := NewS3Storage(cfg)
	
	// Assert that storage was created successfully
	assert.NoError(t, err)
	assert.NotNil(t, storage)
	
	// Assert that storage implements the StorageService interface
	var _ services.StorageService = storage
}

// TestStoreTemporary tests storing a document in temporary storage
func TestStoreTemporary(t *testing.T) {
	// Create mock S3 client and session
	s3Client, uploader, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:       s3Client,
		uploader: uploader,
		config:   createTestConfig(),
	}
	
	// Create test content
	content := bytes.NewReader([]byte(testContent))
	
	// Call StoreTemporary
	storagePath, err := storage.StoreTemporary(
		context.Background(),
		testTenantID,
		testDocumentID,
		content,
		int64(len(testContent)),
		testContentType,
	)
	
	// Assert results
	assert.NoError(t, err)
	assert.NotEmpty(t, storagePath)
	assert.Contains(t, storagePath, testTenantID)
	assert.Contains(t, storagePath, testDocumentID)
}

// TestStoreTemporary_Error tests error handling when storing a document fails
func TestStoreTemporary_Error(t *testing.T) {
	// Create mock S3 client and session
	s3Client, uploader, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:       s3Client,
		uploader: uploader,
		config:   createTestConfig(),
	}
	
	// Create test content
	content := bytes.NewReader([]byte(testContent))
	
	// Replace the uploader's Upload function to simulate an error
	originalUpload := uploader.Upload
	uploader.Upload = func(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
		return nil, errors.New("simulated upload error")
	}
	defer func() { uploader.Upload = originalUpload }()
	
	// Call StoreTemporary
	storagePath, err := storage.StoreTemporary(
		context.Background(),
		testTenantID,
		testDocumentID,
		content,
		int64(len(testContent)),
		testContentType,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, storagePath)
	assert.Contains(t, err.Error(), "simulated upload error")
}

// TestStorePermanent tests moving a document from temporary to permanent storage
func TestStorePermanent(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Temporary path
	tempPath := "s3://test-temp-bucket/tenant-123/doc-123"
	
	// Call StorePermanent
	permanentPath, err := storage.StorePermanent(
		context.Background(),
		testTenantID,
		testDocumentID,
		testVersionID,
		testFolderID,
		tempPath,
	)
	
	// Assert results
	assert.NoError(t, err)
	assert.NotEmpty(t, permanentPath)
	assert.Contains(t, permanentPath, testTenantID)
	assert.Contains(t, permanentPath, testDocumentID)
	assert.Contains(t, permanentPath, testFolderID)
	assert.Contains(t, permanentPath, testVersionID)
}

// TestStorePermanent_Error tests error handling when moving a document to permanent storage fails
func TestStorePermanent_Error(t *testing.T) {
	// Create mock S3 client that returns an error
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Temporary path
	tempPath := "s3://test-temp-bucket/tenant-123/doc-123"
	
	// Replace the CopyObjectWithContext function to simulate an error
	originalCopyObj := s3Client.CopyObjectWithContext
	s3Client.CopyObjectWithContext = func(
		ctx aws.Context,
		input *s3.CopyObjectInput,
		opts ...request.Option,
	) (*s3.CopyObjectOutput, error) {
		return nil, errors.New("simulated copy error")
	}
	defer func() { s3Client.CopyObjectWithContext = originalCopyObj }()
	
	// Call StorePermanent
	permanentPath, err := storage.StorePermanent(
		context.Background(),
		testTenantID,
		testDocumentID,
		testVersionID,
		testFolderID,
		tempPath,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, permanentPath)
	assert.Contains(t, err.Error(), "simulated copy error")
}

// TestMoveToQuarantine tests moving a document from temporary to quarantine storage
func TestMoveToQuarantine(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Temporary path
	tempPath := "s3://test-temp-bucket/tenant-123/doc-123"
	
	// Call MoveToQuarantine
	quarantinePath, err := storage.MoveToQuarantine(
		context.Background(),
		testTenantID,
		testDocumentID,
		tempPath,
	)
	
	// Assert results
	assert.NoError(t, err)
	assert.NotEmpty(t, quarantinePath)
	assert.Contains(t, quarantinePath, testTenantID)
	assert.Contains(t, quarantinePath, testDocumentID)
	assert.Contains(t, quarantinePath, "quarantine")
}

// TestMoveToQuarantine_Error tests error handling when moving a document to quarantine fails
func TestMoveToQuarantine_Error(t *testing.T) {
	// Create mock S3 client that returns an error
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Temporary path
	tempPath := "s3://test-temp-bucket/tenant-123/doc-123"
	
	// Replace the CopyObjectWithContext function to simulate an error
	originalCopyObj := s3Client.CopyObjectWithContext
	s3Client.CopyObjectWithContext = func(
		ctx aws.Context,
		input *s3.CopyObjectInput,
		opts ...request.Option,
	) (*s3.CopyObjectOutput, error) {
		return nil, errors.New("simulated copy error")
	}
	defer func() { s3Client.CopyObjectWithContext = originalCopyObj }()
	
	// Call MoveToQuarantine
	quarantinePath, err := storage.MoveToQuarantine(
		context.Background(),
		testTenantID,
		testDocumentID,
		tempPath,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, quarantinePath)
	assert.Contains(t, err.Error(), "simulated copy error")
}

// TestGetDocument tests retrieving a document from storage
func TestGetDocument(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Replace the GetObjectWithContext function to return test content
	originalGetObj := s3Client.GetObjectWithContext
	s3Client.GetObjectWithContext = func(
		ctx aws.Context,
		input *s3.GetObjectInput,
		opts ...request.Option,
	) (*s3.GetObjectOutput, error) {
		return &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(testContent))),
		}, nil
	}
	defer func() { s3Client.GetObjectWithContext = originalGetObj }()
	
	// Storage path
	storagePath := "s3://test-bucket/tenant-123/folder-123/doc-123"
	
	// Call GetDocument
	document, err := storage.GetDocument(
		context.Background(),
		storagePath,
	)
	
	// Assert results
	assert.NoError(t, err)
	assert.NotNil(t, document)
	
	// Read the document content
	content, err := ioutil.ReadAll(document)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// TestGetDocument_Error tests error handling when retrieving a document fails
func TestGetDocument_Error(t *testing.T) {
	// Create mock S3 client that returns an error
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Replace the GetObjectWithContext function to simulate an error
	originalGetObj := s3Client.GetObjectWithContext
	s3Client.GetObjectWithContext = func(
		ctx aws.Context,
		input *s3.GetObjectInput,
		opts ...request.Option,
	) (*s3.GetObjectOutput, error) {
		return nil, errors.New("simulated get error")
	}
	defer func() { s3Client.GetObjectWithContext = originalGetObj }()
	
	// Storage path
	storagePath := "s3://test-bucket/tenant-123/folder-123/doc-123"
	
	// Call GetDocument
	document, err := storage.GetDocument(
		context.Background(),
		storagePath,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Nil(t, document)
	assert.Contains(t, err.Error(), "simulated get error")
}

// TestGetPresignedURL tests generating a presigned URL for direct document download
func TestGetPresignedURL(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Storage path - using a valid path format
	storagePath := "s3://test-bucket/tenant-123/folder-123/doc-123"
	
	// Call GetPresignedURL
	presignedURL, err := storage.GetPresignedURL(
		context.Background(),
		storagePath,
		"test.pdf",
		3600,
	)
	
	// Assert results - in mock environment, we expect a URL but it may not be valid
	assert.NoError(t, err)
	assert.NotEmpty(t, presignedURL)
	
	// Verify content disposition header is set
	assert.Contains(t, presignedURL, "response-content-disposition")
	assert.Contains(t, presignedURL, "test.pdf")
}

// TestGetPresignedURL_Error tests error handling when generating a presigned URL fails
func TestGetPresignedURL_Error(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Invalid storage path format to trigger an error
	storagePath := "invalid-path"
	
	// Call GetPresignedURL
	presignedURL, err := storage.GetPresignedURL(
		context.Background(),
		storagePath,
		"test.pdf",
		3600,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, presignedURL)
}

// TestDeleteDocument tests deleting a document from storage
func TestDeleteDocument(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Storage path
	storagePath := "s3://test-bucket/tenant-123/folder-123/doc-123"
	
	// Call DeleteDocument
	err := storage.DeleteDocument(
		context.Background(),
		storagePath,
	)
	
	// Assert results
	assert.NoError(t, err)
}

// TestDeleteDocument_Error tests error handling when deleting a document fails
func TestDeleteDocument_Error(t *testing.T) {
	// Create mock S3 client that returns an error
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Replace the DeleteObjectWithContext function to simulate an error
	originalDeleteObj := s3Client.DeleteObjectWithContext
	s3Client.DeleteObjectWithContext = func(
		ctx aws.Context,
		input *s3.DeleteObjectInput,
		opts ...request.Option,
	) (*s3.DeleteObjectOutput, error) {
		return nil, errors.New("simulated delete error")
	}
	defer func() { s3Client.DeleteObjectWithContext = originalDeleteObj }()
	
	// Storage path
	storagePath := "s3://test-bucket/tenant-123/folder-123/doc-123"
	
	// Call DeleteDocument
	err := storage.DeleteDocument(
		context.Background(),
		storagePath,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated delete error")
}

// TestCreateBatchArchive tests creating a compressed archive of multiple documents
func TestCreateBatchArchive(t *testing.T) {
	// Create mock S3 client and session
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Replace the GetObjectWithContext function to return test content
	originalGetObj := s3Client.GetObjectWithContext
	s3Client.GetObjectWithContext = func(
		ctx aws.Context,
		input *s3.GetObjectInput,
		opts ...request.Option,
	) (*s3.GetObjectOutput, error) {
		return &s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(testContent))),
		}, nil
	}
	defer func() { s3Client.GetObjectWithContext = originalGetObj }()
	
	// Storage paths and filenames
	storagePaths := []string{
		"s3://test-bucket/tenant-123/folder-123/doc-1",
		"s3://test-bucket/tenant-123/folder-123/doc-2",
	}
	filenames := []string{
		"document1.pdf",
		"document2.pdf",
	}
	
	// Call CreateBatchArchive
	archive, err := storage.CreateBatchArchive(
		context.Background(),
		storagePaths,
		filenames,
	)
	
	// Assert results
	assert.NoError(t, err)
	assert.NotNil(t, archive)
	
	// Read the archive content
	archiveContent, err := ioutil.ReadAll(archive)
	assert.NoError(t, err)
	
	// Create a reader for the ZIP file
	zipReader, err := zip.NewReader(bytes.NewReader(archiveContent), int64(len(archiveContent)))
	assert.NoError(t, err)
	
	// Check the archive contents
	assert.Len(t, zipReader.File, 2)
	
	// Verify filenames and contents
	for i, expectedName := range filenames {
		assert.Equal(t, expectedName, zipReader.File[i].Name)
		
		// Open and read the file
		fileReader, err := zipReader.File[i].Open()
		assert.NoError(t, err)
		
		fileContent, err := ioutil.ReadAll(fileReader)
		assert.NoError(t, err)
		fileReader.Close()
		
		assert.Equal(t, testContent, string(fileContent))
	}
}

// TestCreateBatchArchive_Error tests error handling when creating a batch archive fails
func TestCreateBatchArchive_Error(t *testing.T) {
	// Create mock S3 client that returns an error
	s3Client, _, _ := createMockS3Client()
	
	// Create S3 storage service with mock client
	storage := &S3Storage{
		s3:     s3Client,
		config: createTestConfig(),
	}
	
	// Replace the GetObjectWithContext function to simulate an error
	originalGetObj := s3Client.GetObjectWithContext
	s3Client.GetObjectWithContext = func(
		ctx aws.Context,
		input *s3.GetObjectInput,
		opts ...request.Option,
	) (*s3.GetObjectOutput, error) {
		return nil, errors.New("simulated get error")
	}
	defer func() { s3Client.GetObjectWithContext = originalGetObj }()
	
	// Storage paths and filenames
	storagePaths := []string{
		"s3://test-bucket/tenant-123/folder-123/doc-1",
	}
	filenames := []string{
		"document1.pdf",
	}
	
	// Call CreateBatchArchive
	archive, err := storage.CreateBatchArchive(
		context.Background(),
		storagePaths,
		filenames,
	)
	
	// Assert results
	assert.Error(t, err)
	assert.Nil(t, archive)
	assert.Contains(t, err.Error(), "simulated get error")
}

// TestParseBucketAndKey tests parsing a storage path into bucket and key components
func TestParseBucketAndKey(t *testing.T) {
	// Create S3 storage service
	storage := &S3Storage{
		config: createTestConfig(),
	}
	
	// Test cases
	testCases := []struct {
		storagePath  string
		expectBucket string
		expectKey    string
		expectError  bool
	}{
		{
			storagePath:  "s3://test-temp-bucket/tenant-123/doc-123",
			expectBucket: "test-temp-bucket",
			expectKey:    "tenant-123/doc-123",
			expectError:  false,
		},
		{
			storagePath:  "s3://test-bucket/tenant-123/folder-123/doc-123",
			expectBucket: "test-bucket",
			expectKey:    "tenant-123/folder-123/doc-123",
			expectError:  false,
		},
		{
			storagePath:  "s3://test-quarantine-bucket/tenant-123/doc-123",
			expectBucket: "test-quarantine-bucket",
			expectKey:    "tenant-123/doc-123",
			expectError:  false,
		},
		{
			storagePath:  "invalid-path",
			expectBucket: "",
			expectKey:    "",
			expectError:  true,
		},
	}
	
	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.storagePath, func(t *testing.T) {
			bucket, key, err := storage.parseBucketAndKey(tc.storagePath)
			
			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, bucket)
				assert.Empty(t, key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectBucket, bucket)
				assert.Equal(t, tc.expectKey, key)
			}
		})
	}
}