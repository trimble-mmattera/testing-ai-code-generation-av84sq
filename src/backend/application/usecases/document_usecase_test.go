package usecases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/org/project/test/mocks"
	"github.com/org/project/domain/models"
	"github.com/org/project/pkg/utils"
	apperrors "github.com/org/project/pkg/errors"
)

// DocumentUseCaseTestSuite is a test suite for DocumentUseCase implementation
type DocumentUseCaseTestSuite struct {
	suite.Suite
	mockDocRepo          *mocks.DocumentRepository
	mockStorageService   *mocks.StorageService
	mockVirusScanService *mocks.VirusScanningService
	mockSearchService    *mocks.SearchService
	mockFolderService    *mocks.FolderService
	mockEventService     *mocks.EventServiceInterface
	mockAuthService      *mocks.AuthService
	mockThumbnailService *mocks.ThumbnailService
	useCase              DocumentUseCase
	ctx                  context.Context
}

// SetupTest sets up the test environment before each test
func (s *DocumentUseCaseTestSuite) SetupTest() {
	s.ctx = context.Background()
	
	// Create mock instances
	s.mockDocRepo = new(mocks.DocumentRepository)
	s.mockStorageService = new(mocks.StorageService)
	s.mockVirusScanService = new(mocks.VirusScanningService)
	s.mockSearchService = new(mocks.SearchService)
	s.mockFolderService = new(mocks.FolderService)
	s.mockEventService = new(mocks.EventServiceInterface)
	s.mockAuthService = new(mocks.AuthService)
	s.mockThumbnailService = new(mocks.ThumbnailService)
	
	// Initialize the use case with mocks
	s.useCase = NewDocumentUseCase(
		s.mockDocRepo,
		s.mockStorageService,
		s.mockVirusScanService,
		s.mockSearchService,
		s.mockFolderService,
		s.mockEventService,
		s.mockAuthService,
		s.mockThumbnailService,
	)
}

// TestUploadDocument_Success tests successful document upload
func (s *DocumentUseCaseTestSuite) TestUploadDocument_Success() {
	// Test data
	name := "test.pdf"
	contentType := "application/pdf"
	size := int64(1024)
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	content := bytes.NewReader([]byte("test content"))
	
	// Expecting a document ID to be returned
	expectedDocID := "doc-123"
	
	// Mock folder permission check
	s.mockFolderService.On("CheckFolderPermission", s.ctx, folderID, tenantID, userID, "write").Return(nil)
	
	// Mock temporary storage
	tempLocation := "temp/location/path"
	s.mockStorageService.On("StoreTemporary", s.ctx, mock.AnythingOfType("io.Reader"), mock.AnythingOfType("*models.Document")).Return(tempLocation, nil)
	
	// Mock document creation
	s.mockDocRepo.On("Create", s.ctx, mock.AnythingOfType("*models.Document")).Run(func(args mock.Arguments) {
		doc := args.Get(1).(*models.Document)
		doc.ID = expectedDocID
	}).Return(expectedDocID, nil)
	
	// Mock virus scanning queue
	s.mockVirusScanService.On("QueueForScanning", s.ctx, mock.AnythingOfType("*models.Document"), tempLocation).Return(nil)
	
	// Mock event publishing
	s.mockEventService.On("PublishDocumentUploadedEvent", s.ctx, mock.AnythingOfType("*models.Document")).Return(nil)
	
	// Call the use case method
	docID, err := s.useCase.UploadDocument(s.ctx, name, contentType, size, folderID, tenantID, userID, content)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedDocID, docID)
	
	// Verify mocks
	s.mockFolderService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockVirusScanService.AssertExpectations(s.T())
	s.mockEventService.AssertExpectations(s.T())
}

// TestUploadDocument_ValidationError tests document upload with validation errors
func (s *DocumentUseCaseTestSuite) TestUploadDocument_ValidationError() {
	// Test cases for validation errors
	testCases := []struct {
		name        string
		contentType string
		size        int64
		folderID    string
		tenantID    string
		userID      string
		content     io.Reader
		errorMsg    string
	}{
		{
			name:        "",
			contentType: "application/pdf",
			size:        1024,
			folderID:    "folder-123",
			tenantID:    "tenant-123",
			userID:      "user-123",
			content:     bytes.NewReader([]byte("test")),
			errorMsg:    "document name is required",
		},
		{
			name:        "test.pdf",
			contentType: "",
			size:        1024,
			folderID:    "folder-123",
			tenantID:    "tenant-123",
			userID:      "user-123",
			content:     bytes.NewReader([]byte("test")),
			errorMsg:    "content type is required",
		},
		{
			name:        "test.pdf",
			contentType: "application/pdf",
			size:        0,
			folderID:    "folder-123",
			tenantID:    "tenant-123",
			userID:      "user-123",
			content:     bytes.NewReader([]byte("test")),
			errorMsg:    "size must be greater than 0",
		},
		{
			name:        "test.pdf",
			contentType: "application/pdf",
			size:        1024,
			folderID:    "",
			tenantID:    "tenant-123",
			userID:      "user-123",
			content:     bytes.NewReader([]byte("test")),
			errorMsg:    "folder ID is required",
		},
		{
			name:        "test.pdf",
			contentType: "application/pdf",
			size:        1024,
			folderID:    "folder-123",
			tenantID:    "",
			userID:      "user-123",
			content:     bytes.NewReader([]byte("test")),
			errorMsg:    "tenant ID is required",
		},
		{
			name:        "test.pdf",
			contentType: "application/pdf",
			size:        1024,
			folderID:    "folder-123",
			tenantID:    "tenant-123",
			userID:      "",
			content:     bytes.NewReader([]byte("test")),
			errorMsg:    "owner ID is required",
		},
		{
			name:        "test.pdf",
			contentType: "application/pdf",
			size:        1024,
			folderID:    "folder-123",
			tenantID:    "tenant-123",
			userID:      "user-123",
			content:     nil,
			errorMsg:    "content cannot be nil",
		},
	}
	
	for _, tc := range testCases {
		// Call the use case method with invalid data
		_, err := s.useCase.UploadDocument(s.ctx, tc.name, tc.contentType, tc.size, tc.folderID, tc.tenantID, tc.userID, tc.content)
		
		// Assert that a validation error is returned with the expected message
		s.True(apperrors.IsValidationError(err))
		s.Contains(err.Error(), tc.errorMsg)
	}
}

// TestUploadDocument_FolderPermissionDenied tests document upload with folder permission denied
func (s *DocumentUseCaseTestSuite) TestUploadDocument_FolderPermissionDenied() {
	// Test data
	name := "test.pdf"
	contentType := "application/pdf"
	size := int64(1024)
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	content := bytes.NewReader([]byte("test content"))
	
	// Mock folder permission check to return permission denied
	permError := apperrors.NewAuthorizationError("permission denied to write to folder")
	s.mockFolderService.On("CheckFolderPermission", s.ctx, folderID, tenantID, userID, "write").Return(permError)
	
	// Call the use case method
	_, err := s.useCase.UploadDocument(s.ctx, name, contentType, size, folderID, tenantID, userID, content)
	
	// Assert expectations
	s.True(apperrors.IsAuthorizationError(err))
	s.Contains(err.Error(), "permission denied")
	
	// Verify mocks
	s.mockFolderService.AssertExpectations(s.T())
}

// TestUploadDocument_StorageError tests document upload with storage error
func (s *DocumentUseCaseTestSuite) TestUploadDocument_StorageError() {
	// Test data
	name := "test.pdf"
	contentType := "application/pdf"
	size := int64(1024)
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	content := bytes.NewReader([]byte("test content"))
	
	// Mock folder permission check
	s.mockFolderService.On("CheckFolderPermission", s.ctx, folderID, tenantID, userID, "write").Return(nil)
	
	// Mock temporary storage to return an error
	storageError := errors.New("storage error")
	s.mockStorageService.On("StoreTemporary", s.ctx, mock.AnythingOfType("io.Reader"), mock.AnythingOfType("*models.Document")).Return("", storageError)
	
	// Call the use case method
	_, err := s.useCase.UploadDocument(s.ctx, name, contentType, size, folderID, tenantID, userID, content)
	
	// Assert expectations
	s.Error(err)
	s.Contains(err.Error(), "storage error")
	
	// Verify mocks
	s.mockFolderService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
}

// TestUploadDocument_RepositoryError tests document upload with repository error
func (s *DocumentUseCaseTestSuite) TestUploadDocument_RepositoryError() {
	// Test data
	name := "test.pdf"
	contentType := "application/pdf"
	size := int64(1024)
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	content := bytes.NewReader([]byte("test content"))
	
	// Mock folder permission check
	s.mockFolderService.On("CheckFolderPermission", s.ctx, folderID, tenantID, userID, "write").Return(nil)
	
	// Mock temporary storage
	tempLocation := "temp/location/path"
	s.mockStorageService.On("StoreTemporary", s.ctx, mock.AnythingOfType("io.Reader"), mock.AnythingOfType("*models.Document")).Return(tempLocation, nil)
	
	// Mock document creation to return an error
	repoError := errors.New("repository error")
	s.mockDocRepo.On("Create", s.ctx, mock.AnythingOfType("*models.Document")).Return("", repoError)
	
	// Call the use case method
	_, err := s.useCase.UploadDocument(s.ctx, name, contentType, size, folderID, tenantID, userID, content)
	
	// Assert expectations
	s.Error(err)
	s.Contains(err.Error(), "repository error")
	
	// Verify mocks
	s.mockFolderService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockDocRepo.AssertExpectations(s.T())
}

// TestGetDocument_Success tests successful document retrieval
func (s *DocumentUseCaseTestSuite) TestGetDocument_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document to be returned by the repository
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Call the use case method
	doc, err := s.useCase.GetDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(testDoc, doc)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestGetDocument_NotFound tests document retrieval when document is not found
func (s *DocumentUseCaseTestSuite) TestGetDocument_NotFound() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Mock document retrieval from repository to return not found error
	notFoundErr := apperrors.NewResourceNotFoundError("document not found")
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(nil, notFoundErr)
	
	// Call the use case method
	_, err := s.useCase.GetDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.True(apperrors.IsResourceNotFoundError(err))
	s.Contains(err.Error(), "document not found")
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
}

// TestGetDocument_WrongTenant tests document retrieval with wrong tenant ID
func (s *DocumentUseCaseTestSuite) TestGetDocument_WrongTenant() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document with a different tenant ID
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", "different-tenant", "folder-123", models.DocumentStatusAvailable)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Call the use case method
	_, err := s.useCase.GetDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.True(apperrors.IsAuthorizationError(err))
	s.Contains(err.Error(), "tenant isolation")
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
}

// TestGetDocument_PermissionDenied tests document retrieval with permission denied
func (s *DocumentUseCaseTestSuite) TestGetDocument_PermissionDenied() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document to be returned by the repository
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check to return permission denied
	permError := apperrors.NewAuthorizationError("permission denied to read document")
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(permError)
	
	// Call the use case method
	_, err := s.useCase.GetDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.True(apperrors.IsAuthorizationError(err))
	s.Contains(err.Error(), "permission denied")
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestDownloadDocument_Success tests successful document download
func (s *DocumentUseCaseTestSuite) TestDownloadDocument_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document with status 'available'
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Create a test document version
	testVersion := s.createTestDocumentVersion("ver-123", documentID, 1, models.VersionStatusAvailable, "storage/path")
	testDoc.Versions = append(testDoc.Versions, testVersion)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Mock document content retrieval from storage
	content := io.NopCloser(bytes.NewReader([]byte("document content")))
	s.mockStorageService.On("GetContent", s.ctx, testVersion.StoragePath).Return(content, nil)
	
	// Mock event publishing
	s.mockEventService.On("PublishDocumentDownloadedEvent", s.ctx, testDoc, userID).Return(nil)
	
	// Call the use case method
	resultContent, filename, err := s.useCase.DownloadDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	s.Equal("test.pdf", filename)
	
	// Read the content to verify it
	contentBytes, err := io.ReadAll(resultContent)
	s.NoError(err)
	s.Equal([]byte("document content"), contentBytes)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockEventService.AssertExpectations(s.T())
}

// TestDownloadDocument_NotAvailable tests document download when document is not available
func (s *DocumentUseCaseTestSuite) TestDownloadDocument_NotAvailable() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document with status 'processing'
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusProcessing)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Call the use case method
	_, _, err := s.useCase.DownloadDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.True(apperrors.IsValidationError(err))
	s.Contains(err.Error(), "not available")
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestGetDocumentPresignedURL_Success tests successful generation of presigned URL for document download
func (s *DocumentUseCaseTestSuite) TestGetDocumentPresignedURL_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	expirationSeconds := 3600
	
	// Create a test document with status 'available'
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Create a test document version
	testVersion := s.createTestDocumentVersion("ver-123", documentID, 1, models.VersionStatusAvailable, "storage/path")
	testDoc.Versions = append(testDoc.Versions, testVersion)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Mock presigned URL generation from storage
	expectedURL := "https://presigned-url.example.com/document"
	s.mockStorageService.On("GetPresignedURL", s.ctx, testVersion.StoragePath, expirationSeconds).Return(expectedURL, nil)
	
	// Mock event publishing
	s.mockEventService.On("PublishDocumentDownloadedEvent", s.ctx, testDoc, userID).Return(nil)
	
	// Call the use case method
	url, err := s.useCase.GetDocumentPresignedURL(s.ctx, documentID, tenantID, userID, expirationSeconds)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedURL, url)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockEventService.AssertExpectations(s.T())
}

// TestBatchDownloadDocuments_Success tests successful batch document download
func (s *DocumentUseCaseTestSuite) TestBatchDownloadDocuments_Success() {
	// Test data
	documentIDs := []string{"doc-123", "doc-456"}
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create test documents with status 'available'
	testDoc1 := s.createTestDocument(documentIDs[0], "test1.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument(documentIDs[1], "test2.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Create test document versions
	testVersion1 := s.createTestDocumentVersion("ver-123", documentIDs[0], 1, models.VersionStatusAvailable, "storage/path1")
	testVersion2 := s.createTestDocumentVersion("ver-456", documentIDs[1], 1, models.VersionStatusAvailable, "storage/path2")
	testDoc1.Versions = append(testDoc1.Versions, testVersion1)
	testDoc2.Versions = append(testDoc2.Versions, testVersion2)
	
	// Documents map for GetByIDs
	docsMap := map[string]*models.Document{
		documentIDs[0]: testDoc1,
		documentIDs[1]: testDoc2,
	}
	
	// Mock documents retrieval from repository
	s.mockDocRepo.On("GetByIDs", s.ctx, documentIDs).Return(docsMap, nil)
	
	// Mock permission checks
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc1, userID, "read").Return(nil)
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc2, userID, "read").Return(nil)
	
	// Mock batch archive creation from storage
	archiveContent := io.NopCloser(bytes.NewReader([]byte("archive content")))
	s.mockStorageService.On("CreateBatchDownloadArchive", s.ctx, mock.AnythingOfType("map[string]string")).Return(archiveContent, nil)
	
	// Mock event publishing for each document
	s.mockEventService.On("PublishDocumentDownloadedEvent", s.ctx, testDoc1, userID).Return(nil)
	s.mockEventService.On("PublishDocumentDownloadedEvent", s.ctx, testDoc2, userID).Return(nil)
	
	// Call the use case method
	resultContent, err := s.useCase.BatchDownloadDocuments(s.ctx, documentIDs, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	
	// Read the content to verify it
	contentBytes, err := io.ReadAll(resultContent)
	s.NoError(err)
	s.Equal([]byte("archive content"), contentBytes)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockEventService.AssertExpectations(s.T())
}

// TestBatchDownloadDocuments_PermissionDenied tests batch document download with permission denied for some documents
func (s *DocumentUseCaseTestSuite) TestBatchDownloadDocuments_PermissionDenied() {
	// Test data
	documentIDs := []string{"doc-123", "doc-456"}
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create test documents
	testDoc1 := s.createTestDocument(documentIDs[0], "test1.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument(documentIDs[1], "test2.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Documents map for GetByIDs
	docsMap := map[string]*models.Document{
		documentIDs[0]: testDoc1,
		documentIDs[1]: testDoc2,
	}
	
	// Mock documents retrieval from repository
	s.mockDocRepo.On("GetByIDs", s.ctx, documentIDs).Return(docsMap, nil)
	
	// Mock permission checks with one document failing
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc1, userID, "read").Return(nil)
	permError := apperrors.NewAuthorizationError("permission denied to read document")
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc2, userID, "read").Return(permError)
	
	// Call the use case method
	_, err := s.useCase.BatchDownloadDocuments(s.ctx, documentIDs, tenantID, userID)
	
	// Assert expectations
	s.True(apperrors.IsAuthorizationError(err))
	s.Contains(err.Error(), "permission denied")
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestGetBatchDownloadPresignedURL_Success tests successful generation of presigned URL for batch document download
func (s *DocumentUseCaseTestSuite) TestGetBatchDownloadPresignedURL_Success() {
	// Test data
	documentIDs := []string{"doc-123", "doc-456"}
	tenantID := "tenant-123"
	userID := "user-123"
	expirationSeconds := 3600
	
	// Create test documents with status 'available'
	testDoc1 := s.createTestDocument(documentIDs[0], "test1.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument(documentIDs[1], "test2.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Create test document versions
	testVersion1 := s.createTestDocumentVersion("ver-123", documentIDs[0], 1, models.VersionStatusAvailable, "storage/path1")
	testVersion2 := s.createTestDocumentVersion("ver-456", documentIDs[1], 1, models.VersionStatusAvailable, "storage/path2")
	testDoc1.Versions = append(testDoc1.Versions, testVersion1)
	testDoc2.Versions = append(testDoc2.Versions, testVersion2)
	
	// Documents map for GetByIDs
	docsMap := map[string]*models.Document{
		documentIDs[0]: testDoc1,
		documentIDs[1]: testDoc2,
	}
	
	// Mock documents retrieval from repository
	s.mockDocRepo.On("GetByIDs", s.ctx, documentIDs).Return(docsMap, nil)
	
	// Mock permission checks
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc1, userID, "read").Return(nil)
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc2, userID, "read").Return(nil)
	
	// Mock batch archive creation and presigned URL generation from storage
	expectedURL := "https://presigned-url.example.com/batch-archive"
	s.mockStorageService.On("GetBatchDownloadPresignedURL", s.ctx, mock.AnythingOfType("map[string]string"), expirationSeconds).Return(expectedURL, nil)
	
	// Mock event publishing for each document
	s.mockEventService.On("PublishDocumentDownloadedEvent", s.ctx, testDoc1, userID).Return(nil)
	s.mockEventService.On("PublishDocumentDownloadedEvent", s.ctx, testDoc2, userID).Return(nil)
	
	// Call the use case method
	url, err := s.useCase.GetBatchDownloadPresignedURL(s.ctx, documentIDs, tenantID, userID, expirationSeconds)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedURL, url)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockEventService.AssertExpectations(s.T())
}

// TestDeleteDocument_Success tests successful document deletion
func (s *DocumentUseCaseTestSuite) TestDeleteDocument_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document with versions
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testVersion := s.createTestDocumentVersion("ver-123", documentID, 1, models.VersionStatusAvailable, "storage/path")
	testDoc.Versions = append(testDoc.Versions, testVersion)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check for delete permission
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "delete").Return(nil)
	
	// Mock document content deletion from storage for each version
	s.mockStorageService.On("DeleteContent", s.ctx, testVersion.StoragePath).Return(nil)
	
	// Mock document removal from search index
	s.mockSearchService.On("RemoveDocumentFromIndex", s.ctx, documentID).Return(nil)
	
	// Mock thumbnail deletion
	s.mockThumbnailService.On("DeleteThumbnail", s.ctx, documentID).Return(nil)
	
	// Mock document deletion from repository
	s.mockDocRepo.On("Delete", s.ctx, documentID).Return(nil)
	
	// Mock event publishing
	s.mockEventService.On("PublishDocumentDeletedEvent", s.ctx, testDoc, userID).Return(nil)
	
	// Call the use case method
	err := s.useCase.DeleteDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockStorageService.AssertExpectations(s.T())
	s.mockSearchService.AssertExpectations(s.T())
	s.mockThumbnailService.AssertExpectations(s.T())
	s.mockEventService.AssertExpectations(s.T())
}

// TestDeleteDocument_PermissionDenied tests document deletion with permission denied
func (s *DocumentUseCaseTestSuite) TestDeleteDocument_PermissionDenied() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check to return permission denied
	permError := apperrors.NewAuthorizationError("permission denied to delete document")
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "delete").Return(permError)
	
	// Call the use case method
	err := s.useCase.DeleteDocument(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.True(apperrors.IsAuthorizationError(err))
	s.Contains(err.Error(), "permission denied")
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestListDocumentsByFolder_Success tests successful listing of documents by folder
func (s *DocumentUseCaseTestSuite) TestListDocumentsByFolder_Success() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 20)
	
	// Create test documents
	testDoc1 := s.createTestDocument("doc-123", "test1.pdf", "application/pdf", tenantID, folderID, models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument("doc-456", "test2.pdf", "application/pdf", tenantID, folderID, models.DocumentStatusAvailable)
	testDocs := []*models.Document{testDoc1, testDoc2}
	expectedResult := utils.NewPaginatedResult(testDocs, pagination, int64(len(testDocs)))
	
	// Mock folder permission check
	s.mockFolderService.On("CheckFolderPermission", s.ctx, folderID, tenantID, userID, "read").Return(nil)
	
	// Mock document listing from repository
	s.mockDocRepo.On("ListByFolder", s.ctx, folderID, tenantID, pagination).Return(expectedResult, nil)
	
	// Call the use case method
	result, err := s.useCase.ListDocumentsByFolder(s.ctx, folderID, tenantID, userID, pagination)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedResult, result)
	
	// Verify mocks
	s.mockFolderService.AssertExpectations(s.T())
	s.mockDocRepo.AssertExpectations(s.T())
}

// TestListDocumentsByFolder_FolderPermissionDenied tests listing documents by folder with folder permission denied
func (s *DocumentUseCaseTestSuite) TestListDocumentsByFolder_FolderPermissionDenied() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 20)
	
	// Mock folder permission check to return permission denied
	permError := apperrors.NewAuthorizationError("permission denied to read folder")
	s.mockFolderService.On("CheckFolderPermission", s.ctx, folderID, tenantID, userID, "read").Return(permError)
	
	// Call the use case method
	_, err := s.useCase.ListDocumentsByFolder(s.ctx, folderID, tenantID, userID, pagination)
	
	// Assert expectations
	s.True(apperrors.IsAuthorizationError(err))
	s.Contains(err.Error(), "permission denied")
	
	// Verify mocks
	s.mockFolderService.AssertExpectations(s.T())
}

// TestSearchDocumentsByContent_Success tests successful search of documents by content
func (s *DocumentUseCaseTestSuite) TestSearchDocumentsByContent_Success() {
	// Test data
	query := "test query"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 20)
	
	// Create test documents
	testDoc1 := s.createTestDocument("doc-123", "test1.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument("doc-456", "test2.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDocs := []*models.Document{testDoc1, testDoc2}
	expectedResult := utils.NewPaginatedResult(testDocs, pagination, int64(len(testDocs)))
	
	// Mock tenant access verification
	s.mockAuthService.On("VerifyTenantAccess", s.ctx, tenantID, userID).Return(nil)
	
	// Mock content search
	s.mockSearchService.On("SearchByContent", s.ctx, query, tenantID, pagination).Return(expectedResult, nil)
	
	// Call the use case method
	result, err := s.useCase.SearchDocumentsByContent(s.ctx, query, tenantID, userID, pagination)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedResult, result)
	
	// Verify mocks
	s.mockAuthService.AssertExpectations(s.T())
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchDocumentsByMetadata_Success tests successful search of documents by metadata
func (s *DocumentUseCaseTestSuite) TestSearchDocumentsByMetadata_Success() {
	// Test data
	metadata := map[string]string{"author": "John Doe", "department": "HR"}
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 20)
	
	// Create test documents
	testDoc1 := s.createTestDocument("doc-123", "test1.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument("doc-456", "test2.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDocs := []*models.Document{testDoc1, testDoc2}
	expectedResult := utils.NewPaginatedResult(testDocs, pagination, int64(len(testDocs)))
	
	// Mock tenant access verification
	s.mockAuthService.On("VerifyTenantAccess", s.ctx, tenantID, userID).Return(nil)
	
	// Mock metadata search
	s.mockSearchService.On("SearchByMetadata", s.ctx, metadata, tenantID, pagination).Return(expectedResult, nil)
	
	// Call the use case method
	result, err := s.useCase.SearchDocumentsByMetadata(s.ctx, metadata, tenantID, userID, pagination)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedResult, result)
	
	// Verify mocks
	s.mockAuthService.AssertExpectations(s.T())
	s.mockSearchService.AssertExpectations(s.T())
}

// TestCombinedSearch_Success tests successful combined search of documents by content and metadata
func (s *DocumentUseCaseTestSuite) TestCombinedSearch_Success() {
	// Test data
	contentQuery := "test query"
	metadata := map[string]string{"author": "John Doe"}
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 20)
	
	// Create test documents
	testDoc1 := s.createTestDocument("doc-123", "test1.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDoc2 := s.createTestDocument("doc-456", "test2.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testDocs := []*models.Document{testDoc1, testDoc2}
	expectedResult := utils.NewPaginatedResult(testDocs, pagination, int64(len(testDocs)))
	
	// Mock tenant access verification
	s.mockAuthService.On("VerifyTenantAccess", s.ctx, tenantID, userID).Return(nil)
	
	// Mock combined search
	s.mockSearchService.On("CombinedSearch", s.ctx, contentQuery, metadata, tenantID, pagination).Return(expectedResult, nil)
	
	// Call the use case method
	result, err := s.useCase.CombinedSearch(s.ctx, contentQuery, metadata, tenantID, userID, pagination)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedResult, result)
	
	// Verify mocks
	s.mockAuthService.AssertExpectations(s.T())
	s.mockSearchService.AssertExpectations(s.T())
}

// TestCombinedSearch_NoSearchCriteria tests combined search with no search criteria provided
func (s *DocumentUseCaseTestSuite) TestCombinedSearch_NoSearchCriteria() {
	// Test data with empty contentQuery and metadata
	var contentQuery string
	metadata := map[string]string{}
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 20)
	
	// Call the use case method
	_, err := s.useCase.CombinedSearch(s.ctx, contentQuery, metadata, tenantID, userID, pagination)
	
	// Assert expectations
	s.True(apperrors.IsValidationError(err))
	s.Contains(err.Error(), "search criteria")
}

// TestUpdateDocumentMetadata_Success tests successful update of document metadata
func (s *DocumentUseCaseTestSuite) TestUpdateDocumentMetadata_Success() {
	// Test data
	documentID := "doc-123"
	key := "author"
	value := "John Doe"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check for write permission
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "write").Return(nil)
	
	// Mock metadata update in repository
	s.mockDocRepo.On("UpdateMetadata", s.ctx, documentID, key, value).Return(nil)
	
	// Call the use case method
	err := s.useCase.UpdateDocumentMetadata(s.ctx, documentID, key, value, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestDeleteDocumentMetadata_Success tests successful deletion of document metadata
func (s *DocumentUseCaseTestSuite) TestDeleteDocumentMetadata_Success() {
	// Test data
	documentID := "doc-123"
	key := "author"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check for write permission
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "write").Return(nil)
	
	// Mock metadata deletion from repository
	s.mockDocRepo.On("DeleteMetadata", s.ctx, documentID, key).Return(nil)
	
	// Call the use case method
	err := s.useCase.DeleteDocumentMetadata(s.ctx, documentID, key, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// TestGetDocumentThumbnail_Success tests successful retrieval of document thumbnail
func (s *DocumentUseCaseTestSuite) TestGetDocumentThumbnail_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test document with a version
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testVersion := s.createTestDocumentVersion("ver-123", documentID, 1, models.VersionStatusAvailable, "storage/path")
	testDoc.Versions = append(testDoc.Versions, testVersion)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Mock thumbnail retrieval from thumbnail service
	thumbnailContent := io.NopCloser(bytes.NewReader([]byte("thumbnail content")))
	s.mockThumbnailService.On("GetThumbnail", s.ctx, documentID).Return(thumbnailContent, nil)
	
	// Call the use case method
	resultContent, err := s.useCase.GetDocumentThumbnail(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	
	// Read the content to verify it
	contentBytes, err := io.ReadAll(resultContent)
	s.NoError(err)
	s.Equal([]byte("thumbnail content"), contentBytes)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockThumbnailService.AssertExpectations(s.T())
}

// TestGetDocumentThumbnailURL_Success tests successful generation of URL for document thumbnail
func (s *DocumentUseCaseTestSuite) TestGetDocumentThumbnailURL_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	expirationSeconds := 3600
	
	// Create a test document with a version
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", models.DocumentStatusAvailable)
	testVersion := s.createTestDocumentVersion("ver-123", documentID, 1, models.VersionStatusAvailable, "storage/path")
	testDoc.Versions = append(testDoc.Versions, testVersion)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Mock thumbnail URL generation from thumbnail service
	expectedURL := "https://presigned-url.example.com/thumbnail"
	s.mockThumbnailService.On("GetThumbnailURL", s.ctx, documentID, expirationSeconds).Return(expectedURL, nil)
	
	// Call the use case method
	url, err := s.useCase.GetDocumentThumbnailURL(s.ctx, documentID, tenantID, userID, expirationSeconds)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(expectedURL, url)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
	s.mockThumbnailService.AssertExpectations(s.T())
}

// TestGetDocumentStatus_Success tests successful retrieval of document status
func (s *DocumentUseCaseTestSuite) TestGetDocumentStatus_Success() {
	// Test data
	documentID := "doc-123"
	tenantID := "tenant-123"
	userID := "user-123"
	status := models.DocumentStatusAvailable
	
	// Create a test document with a specific status
	testDoc := s.createTestDocument(documentID, "test.pdf", "application/pdf", tenantID, "folder-123", status)
	
	// Mock document retrieval from repository
	s.mockDocRepo.On("GetByID", s.ctx, documentID).Return(testDoc, nil)
	
	// Mock permission check
	s.mockAuthService.On("CheckDocumentPermission", s.ctx, testDoc, userID, "read").Return(nil)
	
	// Call the use case method
	result, err := s.useCase.GetDocumentStatus(s.ctx, documentID, tenantID, userID)
	
	// Assert expectations
	s.NoError(err)
	s.Equal(status, result)
	
	// Verify mocks
	s.mockDocRepo.AssertExpectations(s.T())
	s.mockAuthService.AssertExpectations(s.T())
}

// Helper function to create a test document
func (s *DocumentUseCaseTestSuite) createTestDocument(id, name, contentType, tenantID, folderID, status string) *models.Document {
	doc := models.NewDocument(name, contentType, 1024, folderID, tenantID, "user-123")
	doc.ID = id
	doc.Status = status
	return &doc
}

// Helper function to create a test document version
func (s *DocumentUseCaseTestSuite) createTestDocumentVersion(id, documentID string, versionNumber int, status, storagePath string) models.DocumentVersion {
	return models.DocumentVersion{
		ID:            id,
		DocumentID:    documentID,
		VersionNumber: versionNumber,
		Size:          1024,
		ContentHash:   "hash123",
		Status:        status,
		StoragePath:   storagePath,
		CreatedAt:     time.Now(),
		CreatedBy:     "user-123",
	}
}

// TestDocumentUseCaseSuite is the entry point for running the test suite
func TestDocumentUseCaseSuite(t *testing.T) {
	suite.Run(t, new(DocumentUseCaseTestSuite))
}