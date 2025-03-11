package thumbnails

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"../../../domain/services"
	"../../../pkg/config"
)

// MockStorageService mocks the storage service for testing thumbnail functionality
type MockStorageService struct {
	mock.Mock
}

// StoreDocument mocks storing a document
func (m *MockStorageService) StoreDocument(ctx context.Context, tenantID, documentID, versionID string, content io.Reader) (string, error) {
	args := m.Called(ctx, tenantID, documentID, versionID, content)
	return args.String(0), args.Error(1)
}

// GetDocument mocks retrieving a document
func (m *MockStorageService) GetDocument(ctx context.Context, tenantID, documentID, versionID string) (io.ReadCloser, error) {
	args := m.Called(ctx, tenantID, documentID, versionID)
	if rf, ok := args.Get(0).(io.ReadCloser); ok {
		return rf, args.Error(1)
	}
	return nil, args.Error(1)
}

// DeleteDocument mocks deleting a document
func (m *MockStorageService) DeleteDocument(ctx context.Context, tenantID, documentID, versionID string) error {
	args := m.Called(ctx, tenantID, documentID, versionID)
	return args.Error(0)
}

// GetPresignedURL mocks getting a presigned URL for a document
func (m *MockStorageService) GetPresignedURL(ctx context.Context, tenantID, documentID, versionID string, expirationSeconds int) (string, error) {
	args := m.Called(ctx, tenantID, documentID, versionID, expirationSeconds)
	return args.String(0), args.Error(1)
}

func TestNewThumbnailGenerator(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Call NewThumbnailGenerator
	thumbnailService := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Assert that the returned service is not nil
	assert.NotNil(t, thumbnailService)
	
	// Assert that the returned service implements the ThumbnailService interface
	var _ services.ThumbnailService = thumbnailService
}

func TestThumbnailGenerator_GenerateThumbnail(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test cases for different document types
	testCases := []struct {
		name            string
		documentID      string
		versionID       string
		tenantID        string
		storagePath     string
		documentContent []byte
		contentType     string
		expectError     bool
	}{
		{
			name:            "PDF Document",
			documentID:      "doc-123",
			versionID:       "ver-123",
			tenantID:        "tenant-123",
			storagePath:     "tenants/tenant-123/documents/doc-123/ver-123",
			documentContent: []byte("%PDF-1.5\n...sample PDF content..."),
			contentType:     "application/pdf",
			expectError:     false,
		},
		{
			name:            "JPEG Image",
			documentID:      "doc-124",
			versionID:       "ver-124",
			tenantID:        "tenant-123",
			storagePath:     "tenants/tenant-123/documents/doc-124/ver-124",
			documentContent: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}, // JPEG header
			contentType:     "image/jpeg",
			expectError:     false,
		},
		{
			name:            "PNG Image",
			documentID:      "doc-125",
			versionID:       "ver-125",
			tenantID:        "tenant-123",
			storagePath:     "tenants/tenant-123/documents/doc-125/ver-125",
			documentContent: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
			contentType:     "image/png",
			expectError:     false,
		},
		{
			name:            "Word Document",
			documentID:      "doc-126",
			versionID:       "ver-126",
			tenantID:        "tenant-123",
			storagePath:     "tenants/tenant-123/documents/doc-126/ver-126",
			documentContent: []byte("word document content"),
			contentType:     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			expectError:     false,
		},
		{
			name:            "Text Document",
			documentID:      "doc-127",
			versionID:       "ver-127",
			tenantID:        "tenant-123",
			storagePath:     "tenants/tenant-123/documents/doc-127/ver-127",
			documentContent: []byte("text document content"),
			contentType:     "text/plain",
			expectError:     false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations
			// First, the storage service should be called to get the document
			mockStorage.On("GetDocument", mock.Anything, tc.tenantID, tc.documentID, tc.versionID).
				Return(io.NopCloser(bytes.NewReader(tc.documentContent)), nil).Once()
			
			// Then, the storage service should be called to store the thumbnail
			expectedThumbnailPath := "thumbnails/" + tc.tenantID + "/" + tc.documentID + "/" + tc.versionID
			mockStorage.On("StoreDocument", mock.Anything, tc.tenantID, tc.documentID, tc.versionID, mock.Anything).
				Return(expectedThumbnailPath, nil).Once()
			
			// Call GenerateThumbnail
			thumbnailPath, err := thumbnailGenerator.GenerateThumbnail(
				context.Background(),
				tc.documentID,
				tc.versionID,
				tc.tenantID,
				tc.storagePath,
			)
			
			// Check results
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedThumbnailPath, thumbnailPath)
			}
			
			// Verify all mocks were called as expected
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestThumbnailGenerator_GetThumbnail(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test parameters
	documentID := "doc-123"
	versionID := "ver-123"
	tenantID := "tenant-123"
	
	// Create a sample thumbnail content
	thumbnailContent := bytes.NewReader([]byte("test thumbnail content"))
	
	// Setup mock expectations
	mockStorage.On("GetDocument", mock.Anything, tenantID, documentID, versionID).
		Return(io.NopCloser(thumbnailContent), nil).Once()
	
	// Call GetThumbnail
	result, err := thumbnailGenerator.GetThumbnail(
		context.Background(),
		documentID,
		versionID,
		tenantID,
	)
	
	// Check results
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	// Read the content to verify it matches
	content, err := io.ReadAll(result)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test thumbnail content"), content)
	
	// Verify all mocks were called as expected
	mockStorage.AssertExpectations(t)
}

func TestThumbnailGenerator_GetThumbnailURL(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test parameters
	documentID := "doc-123"
	versionID := "ver-123"
	tenantID := "tenant-123"
	expirationSeconds := 3600
	expectedURL := "https://test-bucket.s3.amazonaws.com/thumbnails/tenant-123/doc-123/ver-123?X-Amz-Algorithm=AWS4-HMAC-SHA256&..."
	
	// Setup mock expectations
	mockStorage.On("GetPresignedURL", mock.Anything, tenantID, documentID, versionID, expirationSeconds).
		Return(expectedURL, nil).Once()
	
	// Call GetThumbnailURL
	url, err := thumbnailGenerator.GetThumbnailURL(
		context.Background(),
		documentID,
		versionID,
		tenantID,
		expirationSeconds,
	)
	
	// Check results
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)
	
	// Verify all mocks were called as expected
	mockStorage.AssertExpectations(t)
}

func TestThumbnailGenerator_DeleteThumbnail(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test parameters
	documentID := "doc-123"
	versionID := "ver-123"
	tenantID := "tenant-123"
	
	// Setup mock expectations
	mockStorage.On("DeleteDocument", mock.Anything, tenantID, documentID, versionID).
		Return(nil).Once()
	
	// Call DeleteThumbnail
	err := thumbnailGenerator.DeleteThumbnail(
		context.Background(),
		documentID,
		versionID,
		tenantID,
	)
	
	// Check results
	assert.NoError(t, err)
	
	// Verify all mocks were called as expected
	mockStorage.AssertExpectations(t)
}

func TestThumbnailGenerator_generateThumbnailPath(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test cases
	testCases := []struct {
		tenantID   string
		documentID string
		versionID  string
		expected   string
	}{
		{
			tenantID:   "tenant-123",
			documentID: "doc-123",
			versionID:  "ver-123",
			expected:   "thumbnails/tenant-123/doc-123/ver-123",
		},
		{
			tenantID:   "tenant-456",
			documentID: "doc-456",
			versionID:  "ver-456",
			expected:   "thumbnails/tenant-456/doc-456/ver-456",
		},
	}
	
	for _, tc := range testCases {
		// Use reflection or test the path format indirectly through the public API
		// For this test, we'll verify the path format is correct by checking the StoreDocument call
		mockStorage.On("GetDocument", mock.Anything, tc.tenantID, tc.documentID, tc.versionID).
			Return(io.NopCloser(bytes.NewReader([]byte("test"))), nil).Once()
		mockStorage.On("StoreDocument", mock.Anything, tc.tenantID, tc.documentID, tc.versionID, mock.Anything).
			Return(tc.expected, nil).Once()
		
		// Call a public method that uses the path generation
		path, err := thumbnailGenerator.GenerateThumbnail(
			context.Background(),
			tc.documentID,
			tc.versionID,
			tc.tenantID,
			"some/path",
		)
		
		// Check results
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, path)
		
		// Verify mocks
		mockStorage.AssertExpectations(t)
	}
}

func TestThumbnailGenerator_generatePDFThumbnail(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Create a test PDF content
	pdfContent := bytes.NewReader([]byte("%PDF-1.5\n...sample PDF content..."))
	
	// Test PDF thumbnail generation indirectly
	mockStorage.On("GetDocument", mock.Anything, "tenant-123", "doc-123", "ver-123").
		Return(io.NopCloser(pdfContent), nil).Once()
	mockStorage.On("StoreDocument", mock.Anything, "tenant-123", "doc-123", "ver-123", mock.Anything).
		Return("thumbnails/tenant-123/doc-123/ver-123", nil).Once()
	
	// Call GenerateThumbnail with PDF document
	thumbnailPath, err := thumbnailGenerator.GenerateThumbnail(
		context.Background(),
		"doc-123",
		"ver-123",
		"tenant-123",
		"tenants/tenant-123/documents/doc-123/ver-123",
	)
	
	// Check results
	assert.NoError(t, err)
	assert.Equal(t, "thumbnails/tenant-123/doc-123/ver-123", thumbnailPath)
	
	// Verify mocks
	mockStorage.AssertExpectations(t)
}

func TestThumbnailGenerator_generateImageThumbnail(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test cases for different image formats
	testCases := []struct {
		name        string
		contentType string
		content     []byte
	}{
		{
			name:        "JPEG Image",
			contentType: "image/jpeg",
			content:     []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}, // JPEG header
		},
		{
			name:        "PNG Image",
			contentType: "image/png",
			content:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
		},
		{
			name:        "GIF Image",
			contentType: "image/gif",
			content:     []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, // GIF header
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test image thumbnail generation indirectly
			mockStorage.On("GetDocument", mock.Anything, "tenant-123", "doc-img", "ver-img").
				Return(io.NopCloser(bytes.NewReader(tc.content)), nil).Once()
			mockStorage.On("StoreDocument", mock.Anything, "tenant-123", "doc-img", "ver-img", mock.Anything).
				Return("thumbnails/tenant-123/doc-img/ver-img", nil).Once()
			
			// Call GenerateThumbnail with image document
			thumbnailPath, err := thumbnailGenerator.GenerateThumbnail(
				context.Background(),
				"doc-img",
				"ver-img",
				"tenant-123",
				"tenants/tenant-123/documents/doc-img/ver-img",
			)
			
			// Check results
			assert.NoError(t, err)
			assert.Equal(t, "thumbnails/tenant-123/doc-img/ver-img", thumbnailPath)
			
			// Verify mocks
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestThumbnailGenerator_generateGenericThumbnail(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test cases for different content types
	testCases := []struct {
		name        string
		contentType string
		content     []byte
	}{
		{
			name:        "Word Document",
			contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			content:     []byte("word document content"),
		},
		{
			name:        "Excel Document",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			content:     []byte("excel document content"),
		},
		{
			name:        "Text Document",
			contentType: "text/plain",
			content:     []byte("text document content"),
		},
		{
			name:        "Unknown Type",
			contentType: "application/octet-stream",
			content:     []byte("unknown content"),
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test generic thumbnail generation indirectly
			mockStorage.On("GetDocument", mock.Anything, "tenant-123", "doc-gen", "ver-gen").
				Return(io.NopCloser(bytes.NewReader(tc.content)), nil).Once()
			mockStorage.On("StoreDocument", mock.Anything, "tenant-123", "doc-gen", "ver-gen", mock.Anything).
				Return("thumbnails/tenant-123/doc-gen/ver-gen", nil).Once()
			
			// Call GenerateThumbnail with generic document
			thumbnailPath, err := thumbnailGenerator.GenerateThumbnail(
				context.Background(),
				"doc-gen",
				"ver-gen",
				"tenant-123",
				"tenants/tenant-123/documents/doc-gen/ver-gen",
			)
			
			// Check results
			assert.NoError(t, err)
			assert.Equal(t, "thumbnails/tenant-123/doc-gen/ver-gen", thumbnailPath)
			
			// Verify mocks
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestThumbnailGenerator_ErrorHandling(t *testing.T) {
	// Create a mock storage service
	mockStorage := new(MockStorageService)
	
	// Create a test S3 configuration
	s3Config := config.S3Config{
		Region:            "us-east-1",
		Bucket:            "test-bucket",
		TempBucket:        "test-temp-bucket",
		QuarantineBucket:  "test-quarantine-bucket",
	}
	
	// Create a thumbnail generator
	thumbnailGenerator := NewThumbnailGenerator(mockStorage, &s3Config)
	
	// Test parameters
	documentID := "doc-123"
	versionID := "ver-123"
	tenantID := "tenant-123"
	storagePath := "tenants/tenant-123/documents/doc-123/ver-123"
	expirationSeconds := 3600
	
	// Setup error
	expectedError := errors.New("storage service error")
	
	// Test GenerateThumbnail error
	t.Run("GenerateThumbnail Error", func(t *testing.T) {
		mockStorage.On("GetDocument", mock.Anything, tenantID, documentID, versionID).
			Return(nil, expectedError).Once()
		
		// Call GenerateThumbnail
		_, err := thumbnailGenerator.GenerateThumbnail(
			context.Background(),
			documentID,
			versionID,
			tenantID,
			storagePath,
		)
		
		// Check results
		assert.Error(t, err)
		
		// Verify mocks
		mockStorage.AssertExpectations(t)
	})
	
	// Test GetThumbnail error
	t.Run("GetThumbnail Error", func(t *testing.T) {
		mockStorage.On("GetDocument", mock.Anything, tenantID, documentID, versionID).
			Return(nil, expectedError).Once()
		
		// Call GetThumbnail
		_, err := thumbnailGenerator.GetThumbnail(
			context.Background(),
			documentID,
			versionID,
			tenantID,
		)
		
		// Check results
		assert.Error(t, err)
		
		// Verify mocks
		mockStorage.AssertExpectations(t)
	})
	
	// Test GetThumbnailURL error
	t.Run("GetThumbnailURL Error", func(t *testing.T) {
		mockStorage.On("GetPresignedURL", mock.Anything, tenantID, documentID, versionID, expirationSeconds).
			Return("", expectedError).Once()
		
		// Call GetThumbnailURL
		_, err := thumbnailGenerator.GetThumbnailURL(
			context.Background(),
			documentID,
			versionID,
			tenantID,
			expirationSeconds,
		)
		
		// Check results
		assert.Error(t, err)
		
		// Verify mocks
		mockStorage.AssertExpectations(t)
	})
	
	// Test DeleteThumbnail error
	t.Run("DeleteThumbnail Error", func(t *testing.T) {
		mockStorage.On("DeleteDocument", mock.Anything, tenantID, documentID, versionID).
			Return(expectedError).Once()
		
		// Call DeleteThumbnail
		err := thumbnailGenerator.DeleteThumbnail(
			context.Background(),
			documentID,
			versionID,
			tenantID,
		)
		
		// Check results
		assert.Error(t, err)
		
		// Verify mocks
		mockStorage.AssertExpectations(t)
	})
}