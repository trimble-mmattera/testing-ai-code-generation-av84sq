package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"mocks"
	"../../../domain/models"
	"../../../domain/services"
	"../../../pkg/utils"
	"../../../pkg/errors"
)

// FolderUseCaseTestSuite is a test suite for FolderUseCase implementation
type FolderUseCaseTestSuite struct {
	suite.Suite
	mockFolderService *mocks.FolderService
	mockEventService  *mocks.EventServiceInterface
	useCase           FolderUseCase
	ctx               context.Context
}

// SetupTest sets up the test environment before each test
func (s *FolderUseCaseTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockFolderService = new(mocks.FolderService)
	s.mockEventService = new(mocks.EventServiceInterface)
	s.useCase = NewFolderUseCase(s.mockFolderService, s.mockEventService)
}

// TestCreateFolder_Success tests successful folder creation
func (s *FolderUseCaseTestSuite) TestCreateFolder_Success() {
	// Test data
	name := "Test Folder"
	parentID := "parent-123"
	tenantID := "tenant-123"
	userID := "user-123"
	folderID := "folder-123"

	// Setup mock expectations
	s.mockFolderService.On("CreateFolder", mock.Anything, name, parentID, tenantID, userID).Return(folderID, nil)

	// Call the method under test
	result, err := s.useCase.CreateFolder(s.ctx, name, parentID, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), folderID, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestCreateFolder_ValidationError tests folder creation with validation errors
func (s *FolderUseCaseTestSuite) TestCreateFolder_ValidationError() {
	testCases := []struct {
		name     string
		parentID string
		tenantID string
		userID   string
		errorMsg string
	}{
		{
			name:     "",
			parentID: "parent-123",
			tenantID: "tenant-123",
			userID:   "user-123",
			errorMsg: "folder name is required",
		},
		{
			name:     "Test Folder",
			parentID: "parent-123",
			tenantID: "",
			userID:   "user-123",
			errorMsg: "tenant ID is required",
		},
		{
			name:     "Test Folder",
			parentID: "parent-123",
			tenantID: "tenant-123",
			userID:   "",
			errorMsg: "user ID is required",
		},
	}

	for _, tc := range testCases {
		// Call the method under test
		result, err := s.useCase.CreateFolder(s.ctx, tc.name, tc.parentID, tc.tenantID, tc.userID)

		// Assertions
		assert.Empty(s.T(), result)
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.IsValidationError(err), "Expected validation error")
		assert.Contains(s.T(), err.Error(), tc.errorMsg)
	}

	// Assert that no calls were made to the folder service
	s.mockFolderService.AssertNotCalled(s.T(), "CreateFolder")
}

// TestCreateFolder_ServiceError tests folder creation with service error
func (s *FolderUseCaseTestSuite) TestCreateFolder_ServiceError() {
	// Test data
	name := "Test Folder"
	parentID := "parent-123"
	tenantID := "tenant-123"
	userID := "user-123"
	serviceErr := errors.New("service error")

	// Setup mock expectations
	s.mockFolderService.On("CreateFolder", mock.Anything, name, parentID, tenantID, userID).Return("", serviceErr)

	// Call the method under test
	result, err := s.useCase.CreateFolder(s.ctx, name, parentID, tenantID, userID)

	// Assertions
	assert.Empty(s.T(), result)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), serviceErr, err)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestGetFolder_Success tests successful folder retrieval
func (s *FolderUseCaseTestSuite) TestGetFolder_Success() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test folder to be returned
	folder := s.createTestFolder(folderID, "Test Folder", "parent-123", "/parent/Test Folder", tenantID, userID)

	// Setup mock expectations
	s.mockFolderService.On("GetFolder", mock.Anything, folderID, tenantID, userID).Return(folder, nil)

	// Call the method under test
	result, err := s.useCase.GetFolder(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), folder, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestGetFolder_NotFound tests folder retrieval when folder is not found
func (s *FolderUseCaseTestSuite) TestGetFolder_NotFound() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	notFoundErr := errors.NewResourceNotFoundError("folder not found")

	// Setup mock expectations
	s.mockFolderService.On("GetFolder", mock.Anything, folderID, tenantID, userID).Return(nil, notFoundErr)

	// Call the method under test
	result, err := s.useCase.GetFolder(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.Nil(s.T(), result)
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Expected resource not found error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestGetFolder_PermissionDenied tests folder retrieval with permission denied
func (s *FolderUseCaseTestSuite) TestGetFolder_PermissionDenied() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	permDeniedErr := errors.NewPermissionDeniedError("permission denied for folder operation")

	// Setup mock expectations
	s.mockFolderService.On("GetFolder", mock.Anything, folderID, tenantID, userID).Return(nil, permDeniedErr)

	// Call the method under test
	result, err := s.useCase.GetFolder(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.Nil(s.T(), result)
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsPermissionDeniedError(err), "Expected permission denied error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestUpdateFolder_Success tests successful folder update
func (s *FolderUseCaseTestSuite) TestUpdateFolder_Success() {
	// Test data
	folderID := "folder-123"
	name := "Updated Folder"
	tenantID := "tenant-123"
	userID := "user-123"

	// Setup mock expectations
	s.mockFolderService.On("UpdateFolder", mock.Anything, folderID, name, tenantID, userID).Return(nil)

	// Call the method under test
	err := s.useCase.UpdateFolder(s.ctx, folderID, name, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestUpdateFolder_ValidationError tests folder update with validation errors
func (s *FolderUseCaseTestSuite) TestUpdateFolder_ValidationError() {
	testCases := []struct {
		folderID string
		name     string
		tenantID string
		userID   string
		errorMsg string
	}{
		{
			folderID: "",
			name:     "Test Folder",
			tenantID: "tenant-123",
			userID:   "user-123",
			errorMsg: "folder ID is required",
		},
		{
			folderID: "folder-123",
			name:     "",
			tenantID: "tenant-123",
			userID:   "user-123",
			errorMsg: "folder name is required",
		},
		{
			folderID: "folder-123",
			name:     "Test Folder",
			tenantID: "",
			userID:   "user-123",
			errorMsg: "tenant ID is required",
		},
		{
			folderID: "folder-123",
			name:     "Test Folder",
			tenantID: "tenant-123",
			userID:   "",
			errorMsg: "user ID is required",
		},
	}

	for _, tc := range testCases {
		// Call the method under test
		err := s.useCase.UpdateFolder(s.ctx, tc.folderID, tc.name, tc.tenantID, tc.userID)

		// Assertions
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.IsValidationError(err), "Expected validation error")
		assert.Contains(s.T(), err.Error(), tc.errorMsg)
	}

	// Assert that no calls were made to the folder service
	s.mockFolderService.AssertNotCalled(s.T(), "UpdateFolder")
}

// TestUpdateFolder_NotFound tests folder update when folder is not found
func (s *FolderUseCaseTestSuite) TestUpdateFolder_NotFound() {
	// Test data
	folderID := "folder-123"
	name := "Updated Folder"
	tenantID := "tenant-123"
	userID := "user-123"
	notFoundErr := errors.NewResourceNotFoundError("folder not found")

	// Setup mock expectations
	s.mockFolderService.On("UpdateFolder", mock.Anything, folderID, name, tenantID, userID).Return(notFoundErr)

	// Call the method under test
	err := s.useCase.UpdateFolder(s.ctx, folderID, name, tenantID, userID)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Expected resource not found error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestUpdateFolder_PermissionDenied tests folder update with permission denied
func (s *FolderUseCaseTestSuite) TestUpdateFolder_PermissionDenied() {
	// Test data
	folderID := "folder-123"
	name := "Updated Folder"
	tenantID := "tenant-123"
	userID := "user-123"
	permDeniedErr := errors.NewPermissionDeniedError("permission denied for folder operation")

	// Setup mock expectations
	s.mockFolderService.On("UpdateFolder", mock.Anything, folderID, name, tenantID, userID).Return(permDeniedErr)

	// Call the method under test
	err := s.useCase.UpdateFolder(s.ctx, folderID, name, tenantID, userID)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsPermissionDeniedError(err), "Expected permission denied error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestDeleteFolder_Success tests successful folder deletion
func (s *FolderUseCaseTestSuite) TestDeleteFolder_Success() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"

	// Setup mock expectations
	s.mockFolderService.On("DeleteFolder", mock.Anything, folderID, tenantID, userID).Return(nil)

	// Call the method under test
	err := s.useCase.DeleteFolder(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestDeleteFolder_ValidationError tests folder deletion with validation errors
func (s *FolderUseCaseTestSuite) TestDeleteFolder_ValidationError() {
	testCases := []struct {
		folderID string
		tenantID string
		userID   string
		errorMsg string
	}{
		{
			folderID: "",
			tenantID: "tenant-123",
			userID:   "user-123",
			errorMsg: "folder ID is required",
		},
		{
			folderID: "folder-123",
			tenantID: "",
			userID:   "user-123",
			errorMsg: "tenant ID is required",
		},
		{
			folderID: "folder-123",
			tenantID: "tenant-123",
			userID:   "",
			errorMsg: "user ID is required",
		},
	}

	for _, tc := range testCases {
		// Call the method under test
		err := s.useCase.DeleteFolder(s.ctx, tc.folderID, tc.tenantID, tc.userID)

		// Assertions
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.IsValidationError(err), "Expected validation error")
		assert.Contains(s.T(), err.Error(), tc.errorMsg)
	}

	// Assert that no calls were made to the folder service
	s.mockFolderService.AssertNotCalled(s.T(), "DeleteFolder")
}

// TestDeleteFolder_NotFound tests folder deletion when folder is not found
func (s *FolderUseCaseTestSuite) TestDeleteFolder_NotFound() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	notFoundErr := errors.NewResourceNotFoundError("folder not found")

	// Setup mock expectations
	s.mockFolderService.On("DeleteFolder", mock.Anything, folderID, tenantID, userID).Return(notFoundErr)

	// Call the method under test
	err := s.useCase.DeleteFolder(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Expected resource not found error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestDeleteFolder_PermissionDenied tests folder deletion with permission denied
func (s *FolderUseCaseTestSuite) TestDeleteFolder_PermissionDenied() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	permDeniedErr := errors.NewPermissionDeniedError("permission denied for folder operation")

	// Setup mock expectations
	s.mockFolderService.On("DeleteFolder", mock.Anything, folderID, tenantID, userID).Return(permDeniedErr)

	// Call the method under test
	err := s.useCase.DeleteFolder(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsPermissionDeniedError(err), "Expected permission denied error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestListFolderContents_Success tests successful listing of folder contents
func (s *FolderUseCaseTestSuite) TestListFolderContents_Success() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 10)

	// Create test folders
	folders := []models.Folder{
		*s.createTestFolder("folder-1", "Folder 1", folderID, "/folder-123/Folder 1", tenantID, userID),
		*s.createTestFolder("folder-2", "Folder 2", folderID, "/folder-123/Folder 2", tenantID, userID),
	}
	folderResult := utils.NewPaginatedResult(folders, pagination, 2)

	// Create test documents
	documents := []models.Document{
		{
			ID:          "doc-1",
			Name:        "Document 1",
			ContentType: "application/pdf",
			Size:        1024,
			FolderID:    folderID,
			TenantID:    tenantID,
			OwnerID:     userID,
			Status:      models.DocumentStatusAvailable,
		},
		{
			ID:          "doc-2",
			Name:        "Document 2",
			ContentType: "application/pdf",
			Size:        2048,
			FolderID:    folderID,
			TenantID:    tenantID,
			OwnerID:     userID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	documentResult := utils.NewPaginatedResult(documents, pagination, 2)

	// Setup mock expectations
	s.mockFolderService.On("ListFolderContents", mock.Anything, folderID, tenantID, userID, pagination).
		Return(folderResult, documentResult, nil)

	// Call the method under test
	resultFolders, resultDocuments, err := s.useCase.ListFolderContents(s.ctx, folderID, tenantID, userID, pagination)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), folderResult, resultFolders)
	assert.Equal(s.T(), documentResult, resultDocuments)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestListFolderContents_NotFound tests listing folder contents when folder is not found
func (s *FolderUseCaseTestSuite) TestListFolderContents_NotFound() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 10)
	notFoundErr := errors.NewResourceNotFoundError("folder not found")

	// Setup mock expectations
	s.mockFolderService.On("ListFolderContents", mock.Anything, folderID, tenantID, userID, pagination).
		Return(utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, notFoundErr)

	// Call the method under test
	resultFolders, resultDocuments, err := s.useCase.ListFolderContents(s.ctx, folderID, tenantID, userID, pagination)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Expected resource not found error")
	assert.Equal(s.T(), utils.PaginatedResult[models.Folder]{}, resultFolders)
	assert.Equal(s.T(), utils.PaginatedResult[models.Document]{}, resultDocuments)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestListFolderContents_PermissionDenied tests listing folder contents with permission denied
func (s *FolderUseCaseTestSuite) TestListFolderContents_PermissionDenied() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 10)
	permDeniedErr := errors.NewPermissionDeniedError("permission denied for folder operation")

	// Setup mock expectations
	s.mockFolderService.On("ListFolderContents", mock.Anything, folderID, tenantID, userID, pagination).
		Return(utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, permDeniedErr)

	// Call the method under test
	resultFolders, resultDocuments, err := s.useCase.ListFolderContents(s.ctx, folderID, tenantID, userID, pagination)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsPermissionDeniedError(err), "Expected permission denied error")
	assert.Equal(s.T(), utils.PaginatedResult[models.Folder]{}, resultFolders)
	assert.Equal(s.T(), utils.PaginatedResult[models.Document]{}, resultDocuments)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestListRootFolders_Success tests successful listing of root folders
func (s *FolderUseCaseTestSuite) TestListRootFolders_Success() {
	// Test data
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 10)

	// Create test folders
	folders := []models.Folder{
		*s.createTestFolder("folder-1", "Folder 1", "", "/Folder 1", tenantID, userID),
		*s.createTestFolder("folder-2", "Folder 2", "", "/Folder 2", tenantID, userID),
	}
	folderResult := utils.NewPaginatedResult(folders, pagination, 2)

	// Setup mock expectations
	s.mockFolderService.On("ListRootFolders", mock.Anything, tenantID, userID, pagination).
		Return(folderResult, nil)

	// Call the method under test
	result, err := s.useCase.ListRootFolders(s.ctx, tenantID, userID, pagination)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), folderResult, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestListRootFolders_ValidationError tests listing root folders with validation errors
func (s *FolderUseCaseTestSuite) TestListRootFolders_ValidationError() {
	testCases := []struct {
		tenantID string
		userID   string
		errorMsg string
	}{
		{
			tenantID: "",
			userID:   "user-123",
			errorMsg: "tenant ID is required",
		},
		{
			tenantID: "tenant-123",
			userID:   "",
			errorMsg: "user ID is required",
		},
	}

	for _, tc := range testCases {
		// Call the method under test
		result, err := s.useCase.ListRootFolders(s.ctx, tc.tenantID, tc.userID, utils.NewPagination(1, 10))

		// Assertions
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.IsValidationError(err), "Expected validation error")
		assert.Contains(s.T(), err.Error(), tc.errorMsg)
		assert.Equal(s.T(), utils.PaginatedResult[models.Folder]{}, result)
	}

	// Assert that no calls were made to the folder service
	s.mockFolderService.AssertNotCalled(s.T(), "ListRootFolders")
}

// TestMoveFolder_Success tests successful folder move
func (s *FolderUseCaseTestSuite) TestMoveFolder_Success() {
	// Test data
	folderID := "folder-123"
	newParentID := "parent-123"
	tenantID := "tenant-123"
	userID := "user-123"

	// Setup mock expectations
	s.mockFolderService.On("MoveFolder", mock.Anything, folderID, newParentID, tenantID, userID).Return(nil)

	// Call the method under test
	err := s.useCase.MoveFolder(s.ctx, folderID, newParentID, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestMoveFolder_ValidationError tests folder move with validation errors
func (s *FolderUseCaseTestSuite) TestMoveFolder_ValidationError() {
	testCases := []struct {
		folderID    string
		newParentID string
		tenantID    string
		userID      string
		errorMsg    string
	}{
		{
			folderID:    "",
			newParentID: "parent-123",
			tenantID:    "tenant-123",
			userID:      "user-123",
			errorMsg:    "folder ID is required",
		},
		{
			folderID:    "folder-123",
			newParentID: "parent-123",
			tenantID:    "",
			userID:      "user-123",
			errorMsg:    "tenant ID is required",
		},
		{
			folderID:    "folder-123",
			newParentID: "parent-123",
			tenantID:    "tenant-123",
			userID:      "",
			errorMsg:    "user ID is required",
		},
	}

	for _, tc := range testCases {
		// Call the method under test
		err := s.useCase.MoveFolder(s.ctx, tc.folderID, tc.newParentID, tc.tenantID, tc.userID)

		// Assertions
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.IsValidationError(err), "Expected validation error")
		assert.Contains(s.T(), err.Error(), tc.errorMsg)
	}

	// Assert that no calls were made to the folder service
	s.mockFolderService.AssertNotCalled(s.T(), "MoveFolder")
}

// TestMoveFolder_NotFound tests folder move when folder is not found
func (s *FolderUseCaseTestSuite) TestMoveFolder_NotFound() {
	// Test data
	folderID := "folder-123"
	newParentID := "parent-123"
	tenantID := "tenant-123"
	userID := "user-123"
	notFoundErr := errors.NewResourceNotFoundError("folder not found")

	// Setup mock expectations
	s.mockFolderService.On("MoveFolder", mock.Anything, folderID, newParentID, tenantID, userID).Return(notFoundErr)

	// Call the method under test
	err := s.useCase.MoveFolder(s.ctx, folderID, newParentID, tenantID, userID)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Expected resource not found error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestMoveFolder_PermissionDenied tests folder move with permission denied
func (s *FolderUseCaseTestSuite) TestMoveFolder_PermissionDenied() {
	// Test data
	folderID := "folder-123"
	newParentID := "parent-123"
	tenantID := "tenant-123"
	userID := "user-123"
	permDeniedErr := errors.NewPermissionDeniedError("permission denied for folder operation")

	// Setup mock expectations
	s.mockFolderService.On("MoveFolder", mock.Anything, folderID, newParentID, tenantID, userID).Return(permDeniedErr)

	// Call the method under test
	err := s.useCase.MoveFolder(s.ctx, folderID, newParentID, tenantID, userID)

	// Assertions
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsPermissionDeniedError(err), "Expected permission denied error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestSearchFolders_Success tests successful folder search
func (s *FolderUseCaseTestSuite) TestSearchFolders_Success() {
	// Test data
	query := "test"
	tenantID := "tenant-123"
	userID := "user-123"
	pagination := utils.NewPagination(1, 10)

	// Create test folders
	folders := []models.Folder{
		*s.createTestFolder("folder-1", "Test Folder 1", "", "/Test Folder 1", tenantID, userID),
		*s.createTestFolder("folder-2", "Test Folder 2", "", "/Test Folder 2", tenantID, userID),
	}
	folderResult := utils.NewPaginatedResult(folders, pagination, 2)

	// Setup mock expectations
	s.mockFolderService.On("SearchFolders", mock.Anything, query, tenantID, userID, pagination).
		Return(folderResult, nil)

	// Call the method under test
	result, err := s.useCase.SearchFolders(s.ctx, query, tenantID, userID, pagination)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), folderResult, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestSearchFolders_ValidationError tests folder search with validation errors
func (s *FolderUseCaseTestSuite) TestSearchFolders_ValidationError() {
	testCases := []struct {
		query    string
		tenantID string
		userID   string
		errorMsg string
	}{
		{
			query:    "",
			tenantID: "tenant-123",
			userID:   "user-123",
			errorMsg: "search query is required",
		},
		{
			query:    "test",
			tenantID: "",
			userID:   "user-123",
			errorMsg: "tenant ID is required",
		},
		{
			query:    "test",
			tenantID: "tenant-123",
			userID:   "",
			errorMsg: "user ID is required",
		},
	}

	for _, tc := range testCases {
		// Call the method under test
		result, err := s.useCase.SearchFolders(s.ctx, tc.query, tc.tenantID, tc.userID, utils.NewPagination(1, 10))

		// Assertions
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.IsValidationError(err), "Expected validation error")
		assert.Contains(s.T(), err.Error(), tc.errorMsg)
		assert.Equal(s.T(), utils.PaginatedResult[models.Folder]{}, result)
	}

	// Assert that no calls were made to the folder service
	s.mockFolderService.AssertNotCalled(s.T(), "SearchFolders")
}

// TestGetFolderByPath_Success tests successful folder retrieval by path
func (s *FolderUseCaseTestSuite) TestGetFolderByPath_Success() {
	// Test data
	path := "/parent/folder"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create a test folder to be returned
	folder := s.createTestFolder("folder-123", "folder", "parent-123", path, tenantID, userID)

	// Setup mock expectations
	s.mockFolderService.On("GetFolderByPath", mock.Anything, path, tenantID, userID).Return(folder, nil)

	// Call the method under test
	result, err := s.useCase.GetFolderByPath(s.ctx, path, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), folder, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestGetFolderByPath_NotFound tests folder retrieval by path when folder is not found
func (s *FolderUseCaseTestSuite) TestGetFolderByPath_NotFound() {
	// Test data
	path := "/parent/folder"
	tenantID := "tenant-123"
	userID := "user-123"
	notFoundErr := errors.NewResourceNotFoundError("folder not found")

	// Setup mock expectations
	s.mockFolderService.On("GetFolderByPath", mock.Anything, path, tenantID, userID).Return(nil, notFoundErr)

	// Call the method under test
	result, err := s.useCase.GetFolderByPath(s.ctx, path, tenantID, userID)

	// Assertions
	assert.Nil(s.T(), result)
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Expected resource not found error")
	s.mockFolderService.AssertExpectations(s.T())
}

// TestCreateFolderPermission_Success tests successful folder permission creation
func (s *FolderUseCaseTestSuite) TestCreateFolderPermission_Success() {
	// Test data
	folderID := "folder-123"
	roleID := "role-123"
	permissionType := "read"
	tenantID := "tenant-123"
	userID := "user-123"
	permissionID := "perm-123"

	// Setup mock expectations
	s.mockFolderService.On("CreateFolderPermission", mock.Anything, folderID, roleID, permissionType, tenantID, userID).Return(permissionID, nil)

	// Call the method under test
	result, err := s.useCase.CreateFolderPermission(s.ctx, folderID, roleID, permissionType, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), permissionID, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestDeleteFolderPermission_Success tests successful folder permission deletion
func (s *FolderUseCaseTestSuite) TestDeleteFolderPermission_Success() {
	// Test data
	permissionID := "perm-123"
	tenantID := "tenant-123"
	userID := "user-123"

	// Setup mock expectations
	s.mockFolderService.On("DeleteFolderPermission", mock.Anything, permissionID, tenantID, userID).Return(nil)

	// Call the method under test
	err := s.useCase.DeleteFolderPermission(s.ctx, permissionID, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	s.mockFolderService.AssertExpectations(s.T())
}

// TestGetFolderPermissions_Success tests successful retrieval of folder permissions
func (s *FolderUseCaseTestSuite) TestGetFolderPermissions_Success() {
	// Test data
	folderID := "folder-123"
	tenantID := "tenant-123"
	userID := "user-123"
	
	// Create test permissions
	permissions := []*models.Permission{
		s.createTestPermission("perm-1", folderID, "role-1", "read", tenantID),
		s.createTestPermission("perm-2", folderID, "role-2", "write", tenantID),
	}

	// Setup mock expectations
	s.mockFolderService.On("GetFolderPermissions", mock.Anything, folderID, tenantID, userID).Return(permissions, nil)

	// Call the method under test
	result, err := s.useCase.GetFolderPermissions(s.ctx, folderID, tenantID, userID)

	// Assertions
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), permissions, result)
	s.mockFolderService.AssertExpectations(s.T())
}

// Helper function to create a test folder
func (s *FolderUseCaseTestSuite) createTestFolder(id, name, parentID, path, tenantID, ownerID string) *models.Folder {
	folder := models.NewFolder(name, parentID, tenantID, ownerID)
	folder.ID = id
	folder.Path = path
	return folder
}

// Helper function to create a test permission
func (s *FolderUseCaseTestSuite) createTestPermission(id, folderID, roleID, permissionType, tenantID string) *models.Permission {
	permission := &models.Permission{
		ID:             id,
		ResourceType:   models.ResourceTypeFolder,
		ResourceID:     folderID,
		RoleID:         roleID,
		PermissionType: permissionType,
		TenantID:       tenantID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return permission
}

// TestFolderUseCaseSuite runs the test suite
func TestFolderUseCaseSuite(t *testing.T) {
	suite.Run(t, new(FolderUseCaseTestSuite))
}