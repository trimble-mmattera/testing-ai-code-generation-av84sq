// Package e2e provides end-to-end tests for the Document Management Platform.
package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid" // v1.3.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+
	"go.uber.org/zap" // v1.24.0+

	"../../application/usecases/folderusecase"
	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services"
	"../../pkg/config"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// TestFolderFlow runs the folder flow test suite
func TestFolderFlow(t *testing.T) {
	suite.Run(t, new(FolderFlowTestSuite))
}

// FolderFlowTestSuite is the test suite for folder management
type FolderFlowTestSuite struct {
	suite.Suite
	folderRepo    *MockFolderRepository
	folderService services.FolderService
	eventService  *MockEventService
	folderUseCase folderusecase.FolderUseCase
	testTenantID  string
	testUserID    string
	logger        *zap.Logger
	testFolderIDs map[string]string
}

// SetupSuite sets up the test suite
func (s *FolderFlowTestSuite) SetupSuite() {
	// Setup test tenant and user IDs
	s.testTenantID = uuid.New().String()
	s.testUserID = uuid.New().String()

	// Initialize logger
	s.logger = logger.NewLogger()

	// Initialize test folder IDs map
	s.testFolderIDs = make(map[string]string)
}

// SetupTest sets up each test
func (s *FolderFlowTestSuite) SetupTest() {
	// Create mock folder repository
	s.folderRepo = new(MockFolderRepository)

	// Create mock event service
	s.eventService = new(MockEventService)

	// Create mock document repository
	mockDocumentRepo = new(MockDocumentRepository)

	// Create mock permission repository
	mockPermissionRepo = new(MockPermissionRepository)

	// Create mock auth service
	mockAuthService = new(MockAuthService)

	// Create folder service with dependencies
	s.folderService = services.NewFolderService(
		s.folderRepo,
		mockDocumentRepo,
		mockPermissionRepo,
		mockAuthService,
		s.eventService,
	)

	// Create folder use case with dependencies
	s.folderUseCase = folderusecase.NewFolderUseCase(
		s.folderService,
		s.eventService,
	)
}

// TearDownTest cleans up after each test
func (s *FolderFlowTestSuite) TearDownTest() {
	// Clean up any test folders from repositories
}

// TestFolderCreationAndRetrieval tests folder creation and retrieval
func (s *FolderFlowTestSuite) TestFolderCreationAndRetrieval() {
	// Arrange
	ctx := context.Background()
	folderName := "Test Folder"
	parentID := ""
	expectedFolder := models.NewFolder(folderName, parentID, s.testTenantID, s.testUserID)
	expectedFolder.ID = uuid.New().String()
	expectedFolder.Path = "/" + folderName

	// Set up mock expectations for folder creation
	mockAuthService.On("VerifyPermission", mock.Anything, s.testUserID, s.testTenantID, services.PermissionManageFolders).Return(true, nil)
	
	s.folderRepo.On("Create", mock.Anything, mock.MatchedBy(func(folder *models.Folder) bool {
		return folder.Name == folderName && 
			folder.ParentID == parentID && 
			folder.TenantID == s.testTenantID && 
			folder.OwnerID == s.testUserID
	})).Return(expectedFolder.ID, nil)

	mockPermissionRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.New().String(), nil)
	mockPermissionRepo.On("PropagatePermissions", mock.Anything, expectedFolder.ID, s.testTenantID).Return(nil)
	
	s.eventService.On("CreateAndPublishFolderEvent", mock.Anything, services.FolderEventCreated, s.testTenantID, expectedFolder.ID, mock.Anything).
		Return(uuid.New().String(), nil)

	// Act - Create folder
	folderID, err := s.folderUseCase.CreateFolder(ctx, folderName, parentID, s.testTenantID, s.testUserID)

	// Assert
	s.Require().NoError(err)
	s.Require().NotEmpty(folderID)
	s.testFolderIDs["root"] = folderID

	// Set up mock expectations for folder retrieval
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(expectedFolder, nil)

	// Act - Get folder
	folder, err := s.folderUseCase.GetFolder(ctx, folderID, s.testTenantID, s.testUserID)

	// Assert
	s.Require().NoError(err)
	s.Require().NotNil(folder)
	s.Equal(folderName, folder.Name)
	s.Equal(parentID, folder.ParentID)
	s.Equal(s.testTenantID, folder.TenantID)
	s.Equal(s.testUserID, folder.OwnerID)
}

// TestFolderHierarchy tests folder hierarchy creation and navigation
func (s *FolderFlowTestSuite) TestFolderHierarchy() {
	// Arrange
	ctx := context.Background()
	
	// Create a root folder
	rootFolderID, err := s.createTestFolder("Root Folder", "")
	s.Require().NoError(err)
	s.testFolderIDs["root"] = rootFolderID
	
	// Create multiple child folders under the root folder
	child1ID, err := s.createTestFolder("Child Folder 1", rootFolderID)
	s.Require().NoError(err)
	s.testFolderIDs["child1"] = child1ID
	
	child2ID, err := s.createTestFolder("Child Folder 2", rootFolderID)
	s.Require().NoError(err)
	s.testFolderIDs["child2"] = child2ID
	
	// Create nested folders (grandchildren)
	grandchildID, err := s.createTestFolder("Grandchild Folder", child1ID)
	s.Require().NoError(err)
	s.testFolderIDs["grandchild"] = grandchildID
	
	// Set up mock expectations for folder retrieval
	rootFolder := models.NewFolder("Root Folder", "", s.testTenantID, s.testUserID)
	rootFolder.ID = rootFolderID
	rootFolder.Path = "/Root Folder"
	
	child1Folder := models.NewFolder("Child Folder 1", rootFolderID, s.testTenantID, s.testUserID)
	child1Folder.ID = child1ID
	child1Folder.Path = "/Root Folder/Child Folder 1"
	
	child2Folder := models.NewFolder("Child Folder 2", rootFolderID, s.testTenantID, s.testUserID)
	child2Folder.ID = child2ID
	child2Folder.Path = "/Root Folder/Child Folder 2"
	
	grandchildFolder := models.NewFolder("Grandchild Folder", child1ID, s.testTenantID, s.testUserID)
	grandchildFolder.ID = grandchildID
	grandchildFolder.Path = "/Root Folder/Child Folder 1/Grandchild Folder"
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, rootFolderID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, child1ID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, child2ID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, grandchildID, services.PermissionRead).Return(true, nil)
	
	s.folderRepo.On("GetByID", mock.Anything, rootFolderID, s.testTenantID).Return(rootFolder, nil)
	s.folderRepo.On("GetByID", mock.Anything, child1ID, s.testTenantID).Return(child1Folder, nil)
	s.folderRepo.On("GetByID", mock.Anything, child2ID, s.testTenantID).Return(child2Folder, nil)
	s.folderRepo.On("GetByID", mock.Anything, grandchildID, s.testTenantID).Return(grandchildFolder, nil)
	
	// Act & Assert - Verify root folder
	retrievedRoot, err := s.folderUseCase.GetFolder(ctx, rootFolderID, s.testTenantID, s.testUserID)
	s.Require().NoError(err)
	s.Equal("Root Folder", retrievedRoot.Name)
	s.Equal("", retrievedRoot.ParentID)
	s.Equal("/Root Folder", retrievedRoot.Path)
	
	// Act & Assert - Verify child folders
	retrievedChild1, err := s.folderUseCase.GetFolder(ctx, child1ID, s.testTenantID, s.testUserID)
	s.Require().NoError(err)
	s.Equal("Child Folder 1", retrievedChild1.Name)
	s.Equal(rootFolderID, retrievedChild1.ParentID)
	s.Equal("/Root Folder/Child Folder 1", retrievedChild1.Path)
	
	retrievedChild2, err := s.folderUseCase.GetFolder(ctx, child2ID, s.testTenantID, s.testUserID)
	s.Require().NoError(err)
	s.Equal("Child Folder 2", retrievedChild2.Name)
	s.Equal(rootFolderID, retrievedChild2.ParentID)
	s.Equal("/Root Folder/Child Folder 2", retrievedChild2.Path)
	
	// Act & Assert - Verify grandchild folder
	retrievedGrandchild, err := s.folderUseCase.GetFolder(ctx, grandchildID, s.testTenantID, s.testUserID)
	s.Require().NoError(err)
	s.Equal("Grandchild Folder", retrievedGrandchild.Name)
	s.Equal(child1ID, retrievedGrandchild.ParentID)
	s.Equal("/Root Folder/Child Folder 1/Grandchild Folder", retrievedGrandchild.Path)
}

// TestFolderUpdate tests folder update functionality
func (s *FolderFlowTestSuite) TestFolderUpdate() {
	// Arrange
	ctx := context.Background()
	folderName := "Update Test Folder"
	newFolderName := "Updated Folder Name"
	
	// Create a test folder
	folderID, err := s.createTestFolder(folderName, "")
	s.Require().NoError(err)
	
	// Set up mock for folder update
	originalFolder := models.NewFolder(folderName, "", s.testTenantID, s.testUserID)
	originalFolder.ID = folderID
	originalFolder.Path = "/" + folderName
	
	updatedFolder := models.NewFolder(newFolderName, "", s.testTenantID, s.testUserID)
	updatedFolder.ID = folderID
	updatedFolder.Path = "/" + newFolderName
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionWrite).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(originalFolder, nil).Once()
	s.folderRepo.On("Update", mock.Anything, mock.MatchedBy(func(folder *models.Folder) bool {
		return folder.ID == folderID && folder.Name == newFolderName
	})).Return(nil)
	
	s.eventService.On("CreateAndPublishFolderEvent", mock.Anything, services.FolderEventUpdated, s.testTenantID, folderID, mock.Anything).
		Return(uuid.New().String(), nil)
	
	// Set up mock for folder retrieval after update
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(updatedFolder, nil).Once()
	
	// Act - Update folder
	err = s.folderUseCase.UpdateFolder(ctx, folderID, newFolderName, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	
	// Act - Get updated folder
	updatedResult, err := s.folderUseCase.GetFolder(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(newFolderName, updatedResult.Name)
}

// TestFolderMove tests moving a folder to a new parent
func (s *FolderFlowTestSuite) TestFolderMove() {
	// Arrange
	ctx := context.Background()
	
	// Create a root folder
	rootFolder1ID, err := s.createTestFolder("Move Test Root 1", "")
	s.Require().NoError(err)
	
	// Create a second root folder
	rootFolder2ID, err := s.createTestFolder("Move Test Root 2", "")
	s.Require().NoError(err)
	
	// Create a child folder under the first root folder
	childFolderID, err := s.createTestFolder("Move Test Child", rootFolder1ID)
	s.Require().NoError(err)
	
	// Set up mock for folder move
	rootFolder1 := models.NewFolder("Move Test Root 1", "", s.testTenantID, s.testUserID)
	rootFolder1.ID = rootFolder1ID
	rootFolder1.Path = "/Move Test Root 1"
	
	rootFolder2 := models.NewFolder("Move Test Root 2", "", s.testTenantID, s.testUserID)
	rootFolder2.ID = rootFolder2ID
	rootFolder2.Path = "/Move Test Root 2"
	
	childFolder := models.NewFolder("Move Test Child", rootFolder1ID, s.testTenantID, s.testUserID)
	childFolder.ID = childFolderID
	childFolder.Path = "/Move Test Root 1/Move Test Child"
	
	movedChildFolder := models.NewFolder("Move Test Child", rootFolder2ID, s.testTenantID, s.testUserID)
	movedChildFolder.ID = childFolderID
	movedChildFolder.Path = "/Move Test Root 2/Move Test Child"
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, childFolderID, services.PermissionWrite).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, rootFolder2ID, services.PermissionWrite).Return(true, nil)
	
	s.folderRepo.On("GetByID", mock.Anything, childFolderID, s.testTenantID).Return(childFolder, nil).Once()
	s.folderRepo.On("GetByID", mock.Anything, rootFolder2ID, s.testTenantID).Return(rootFolder2, nil)
	s.folderRepo.On("GetFolderPath", mock.Anything, childFolderID, s.testTenantID).Return(childFolder.Path, nil)
	s.folderRepo.On("GetFolderPath", mock.Anything, rootFolder2ID, s.testTenantID).Return(rootFolder2.Path, nil)
	s.folderRepo.On("Move", mock.Anything, childFolderID, rootFolder2ID, s.testTenantID).Return(nil)
	
	s.eventService.On("CreateAndPublishFolderEvent", mock.Anything, services.FolderEventMoved, s.testTenantID, childFolderID, mock.Anything).
		Return(uuid.New().String(), nil)
	
	// Set up mock for folder retrieval after move
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, childFolderID, services.PermissionRead).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, childFolderID, s.testTenantID).Return(movedChildFolder, nil).Once()
	
	// Act - Move folder
	err = s.folderUseCase.MoveFolder(ctx, childFolderID, rootFolder2ID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	
	// Act - Get moved folder
	movedResult, err := s.folderUseCase.GetFolder(ctx, childFolderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(rootFolder2ID, movedResult.ParentID)
	s.Equal("/Move Test Root 2/Move Test Child", movedResult.Path)
}

// TestFolderDeletion tests folder deletion functionality
func (s *FolderFlowTestSuite) TestFolderDeletion() {
	// Arrange
	ctx := context.Background()
	folderName := "Delete Test Folder"
	
	// Create a test folder
	folderID, err := s.createTestFolder(folderName, "")
	s.Require().NoError(err)
	
	// Set up mock for folder deletion
	folder := models.NewFolder(folderName, "", s.testTenantID, s.testUserID)
	folder.ID = folderID
	folder.Path = "/" + folderName
	
	// Mock folder is empty checks
	emptyResult := utils.PaginatedResult[models.Folder]{
		Items: []models.Folder{},
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  0,
			TotalItems:  0,
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	emptyDocResult := utils.PaginatedResult[models.Document]{
		Items: []models.Document{},
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  0,
			TotalItems:  0,
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionDelete).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(folder, nil).Once()
	s.folderRepo.On("GetChildren", mock.Anything, folderID, s.testTenantID, mock.Anything).Return(emptyResult, nil)
	mockDocumentRepo.On("ListByFolder", mock.Anything, folderID, s.testTenantID, mock.Anything).Return(emptyDocResult, nil)
	s.folderRepo.On("Delete", mock.Anything, folderID, s.testTenantID).Return(nil)
	mockPermissionRepo.On("DeleteByResourceID", mock.Anything, models.ResourceTypeFolder, folderID, s.testTenantID).Return(nil)
	
	s.eventService.On("CreateAndPublishFolderEvent", mock.Anything, services.FolderEventDeleted, s.testTenantID, folderID, mock.Anything).
		Return(uuid.New().String(), nil)
	
	// Set up mock for folder retrieval after deletion (should return not found)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(nil, errors.NewResourceNotFoundError("folder not found")).Once()
	
	// Act - Delete folder
	err = s.folderUseCase.DeleteFolder(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	
	// Act - Try to get deleted folder
	_, err = s.folderUseCase.GetFolder(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
}

// TestFolderListing tests listing folders with pagination
func (s *FolderFlowTestSuite) TestFolderListing() {
	// Arrange
	ctx := context.Background()
	
	// Create a parent folder
	parentFolderID, err := s.createTestFolder("Listing Parent Folder", "")
	s.Require().NoError(err)
	
	// Create multiple child folders under the parent
	var childFolders []models.Folder
	for i := 1; i <= 5; i++ {
		childID, err := s.createTestFolder(fmt.Sprintf("Child Folder %d", i), parentFolderID)
		s.Require().NoError(err)
		
		childFolder := models.NewFolder(fmt.Sprintf("Child Folder %d", i), parentFolderID, s.testTenantID, s.testUserID)
		childFolder.ID = childID
		childFolder.Path = fmt.Sprintf("/Listing Parent Folder/Child Folder %d", i)
		
		childFolders = append(childFolders, *childFolder)
	}
	
	// Set up mock for parent folder
	parentFolder := models.NewFolder("Listing Parent Folder", "", s.testTenantID, s.testUserID)
	parentFolder.ID = parentFolderID
	parentFolder.Path = "/Listing Parent Folder"
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, parentFolderID, services.PermissionRead).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, parentFolderID, s.testTenantID).Return(parentFolder, nil)
	
	// Set up mock for listing all child folders
	allChildrenResult := utils.PaginatedResult[models.Folder]{
		Items: childFolders,
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
			TotalItems:  int64(len(childFolders)),
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	emptyDocResult := utils.PaginatedResult[models.Document]{
		Items: []models.Document{},
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  0,
			TotalItems:  0,
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("GetChildren", mock.Anything, parentFolderID, s.testTenantID, mock.MatchedBy(func(p *utils.Pagination) bool {
		return p.Page == 1 && p.PageSize == 10
	})).Return(allChildrenResult, nil).Once()
	
	mockDocumentRepo.On("ListByFolder", mock.Anything, parentFolderID, s.testTenantID, mock.Anything).Return(emptyDocResult, nil)
	
	// Set up mock for paginated results (page 1, 2 items per page)
	paginatedResult1 := utils.PaginatedResult[models.Folder]{
		Items: childFolders[0:2],
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    2,
			TotalPages:  3,
			TotalItems:  int64(len(childFolders)),
			HasNext:     true,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("GetChildren", mock.Anything, parentFolderID, s.testTenantID, mock.MatchedBy(func(p *utils.Pagination) bool {
		return p.Page == 1 && p.PageSize == 2
	})).Return(paginatedResult1, nil).Once()
	
	// Act - Get all child folders
	pagination := utils.NewPagination(1, 10)
	folders, docs, err := s.folderUseCase.ListFolderContents(ctx, parentFolderID, s.testTenantID, s.testUserID, pagination)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(5, len(folders.Items))
	s.Equal(0, len(docs.Items))
	
	// Act - Get paginated results (page 1, 2 items per page)
	pagination = utils.NewPagination(1, 2)
	folders, _, err = s.folderUseCase.ListFolderContents(ctx, parentFolderID, s.testTenantID, s.testUserID, pagination)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(2, len(folders.Items))
	s.Equal(1, folders.Pagination.Page)
	s.Equal(2, folders.Pagination.PageSize)
	s.Equal(3, folders.Pagination.TotalPages)
	s.Equal(int64(5), folders.Pagination.TotalItems)
	s.True(folders.Pagination.HasNext)
	s.False(folders.Pagination.HasPrevious)
}

// TestRootFolderListing tests listing root folders for a tenant
func (s *FolderFlowTestSuite) TestRootFolderListing() {
	// Arrange
	ctx := context.Background()
	
	// Create multiple root folders
	var rootFolders []models.Folder
	for i := 1; i <= 5; i++ {
		rootID, err := s.createTestFolder(fmt.Sprintf("Root Folder %d", i), "")
		s.Require().NoError(err)
		
		rootFolder := models.NewFolder(fmt.Sprintf("Root Folder %d", i), "", s.testTenantID, s.testUserID)
		rootFolder.ID = rootID
		rootFolder.Path = fmt.Sprintf("/Root Folder %d", i)
		
		rootFolders = append(rootFolders, *rootFolder)
	}
	
	// Set up mock for listing all root folders
	mockAuthService.On("VerifyTenantAccess", mock.Anything, s.testUserID, s.testTenantID).Return(true, nil)
	
	allRootsResult := utils.PaginatedResult[models.Folder]{
		Items: rootFolders,
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
			TotalItems:  int64(len(rootFolders)),
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("GetRootFolders", mock.Anything, s.testTenantID, mock.MatchedBy(func(p *utils.Pagination) bool {
		return p.Page == 1 && p.PageSize == 10
	})).Return(allRootsResult, nil).Once()
	
	// Set up mock for paginated results (page 1, 2 items per page)
	paginatedResult1 := utils.PaginatedResult[models.Folder]{
		Items: rootFolders[0:2],
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    2,
			TotalPages:  3,
			TotalItems:  int64(len(rootFolders)),
			HasNext:     true,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("GetRootFolders", mock.Anything, s.testTenantID, mock.MatchedBy(func(p *utils.Pagination) bool {
		return p.Page == 1 && p.PageSize == 2
	})).Return(paginatedResult1, nil).Once()
	
	// Act - Get all root folders
	pagination := utils.NewPagination(1, 10)
	folders, err := s.folderUseCase.ListRootFolders(ctx, s.testTenantID, s.testUserID, pagination)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(5, len(folders.Items))
	
	// Act - Get paginated results (page 1, 2 items per page)
	pagination = utils.NewPagination(1, 2)
	folders, err = s.folderUseCase.ListRootFolders(ctx, s.testTenantID, s.testUserID, pagination)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(2, len(folders.Items))
	s.Equal(1, folders.Pagination.Page)
	s.Equal(2, folders.Pagination.PageSize)
	s.Equal(3, folders.Pagination.TotalPages)
	s.Equal(int64(5), folders.Pagination.TotalItems)
	s.True(folders.Pagination.HasNext)
	s.False(folders.Pagination.HasPrevious)
}

// TestFolderSearch tests searching folders by name
func (s *FolderFlowTestSuite) TestFolderSearch() {
	// Arrange
	ctx := context.Background()
	
	// Create folders with different names
	searchFolder1ID, err := s.createTestFolder("Project X Documentation", "")
	s.Require().NoError(err)
	
	searchFolder2ID, err := s.createTestFolder("Project Y Plans", "")
	s.Require().NoError(err)
	
	searchFolder3ID, err := s.createTestFolder("Project X Plans", "")
	s.Require().NoError(err)
	
	// Create folder objects for search results
	folder1 := models.NewFolder("Project X Documentation", "", s.testTenantID, s.testUserID)
	folder1.ID = searchFolder1ID
	folder1.Path = "/Project X Documentation"
	
	folder2 := models.NewFolder("Project Y Plans", "", s.testTenantID, s.testUserID)
	folder2.ID = searchFolder2ID
	folder2.Path = "/Project Y Plans"
	
	folder3 := models.NewFolder("Project X Plans", "", s.testTenantID, s.testUserID)
	folder3.ID = searchFolder3ID
	folder3.Path = "/Project X Plans"
	
	// Set up mock for searching folders
	mockAuthService.On("VerifyTenantAccess", mock.Anything, s.testUserID, s.testTenantID).Return(true, nil)
	
	// Search for "Project X"
	projectXResults := utils.PaginatedResult[models.Folder]{
		Items: []models.Folder{*folder1, *folder3},
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
			TotalItems:  2,
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("Search", mock.Anything, "Project X", s.testTenantID, mock.MatchedBy(func(p *utils.Pagination) bool {
		return p.Page == 1 && p.PageSize == 10
	})).Return(projectXResults, nil).Once()
	
	// Search for "Plans"
	plansResults := utils.PaginatedResult[models.Folder]{
		Items: []models.Folder{*folder2, *folder3},
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
			TotalItems:  2,
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("Search", mock.Anything, "Plans", s.testTenantID, mock.MatchedBy(func(p *utils.Pagination) bool {
		return p.Page == 1 && p.PageSize == 10
	})).Return(plansResults, nil).Once()
	
	// Act - Search for "Project X"
	pagination := utils.NewPagination(1, 10)
	results, err := s.folderUseCase.SearchFolders(ctx, "Project X", s.testTenantID, s.testUserID, pagination)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(2, len(results.Items))
	s.Equal("Project X Documentation", results.Items[0].Name)
	s.Equal("Project X Plans", results.Items[1].Name)
	
	// Act - Search for "Plans"
	results, err = s.folderUseCase.SearchFolders(ctx, "Plans", s.testTenantID, s.testUserID, pagination)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(2, len(results.Items))
	s.Equal("Project Y Plans", results.Items[0].Name)
	s.Equal("Project X Plans", results.Items[1].Name)
}

// TestFolderPermissions tests folder permission management
func (s *FolderFlowTestSuite) TestFolderPermissions() {
	// Arrange
	ctx := context.Background()
	folderName := "Permission Test Folder"
	roleID := "editor"
	permissionType := models.PermissionTypeWrite
	
	// Create a test folder
	folderID, err := s.createTestFolder(folderName, "")
	s.Require().NoError(err)
	
	// Set up mock for folder
	folder := models.NewFolder(folderName, "", s.testTenantID, s.testUserID)
	folder.ID = folderID
	folder.Path = "/" + folderName
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, models.PermissionTypeAdmin).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(folder, nil)
	
	// Set up mock for creating permission
	permissionID := uuid.New().String()
	mockPermissionRepo.On("Create", mock.Anything, mock.MatchedBy(func(permission *models.Permission) bool {
		return permission.ResourceType == models.ResourceTypeFolder &&
			permission.ResourceID == folderID &&
			permission.RoleID == roleID &&
			permission.PermissionType == permissionType &&
			permission.TenantID == s.testTenantID
	})).Return(permissionID, nil)
	
	mockPermissionRepo.On("PropagatePermissions", mock.Anything, folderID, s.testTenantID).Return(nil)
	
	// Set up mock for retrieving permissions
	permission := models.NewPermission(roleID, models.ResourceTypeFolder, folderID, permissionType, s.testTenantID, s.testUserID)
	permission.ID = permissionID
	
	mockPermissionRepo.On("GetByResourceID", mock.Anything, models.ResourceTypeFolder, folderID, s.testTenantID).Return([]*models.Permission{permission}, nil).Once()
	mockPermissionRepo.On("GetInheritedPermissions", mock.Anything, folderID, s.testTenantID).Return([]*models.Permission{}, nil)
	
	// Set up mock for deleting permission
	mockPermissionRepo.On("GetByID", mock.Anything, permissionID, s.testTenantID).Return(permission, nil)
	mockPermissionRepo.On("Delete", mock.Anything, permissionID, s.testTenantID).Return(nil)
	
	// Set up mock for retrieving permissions after deletion
	mockPermissionRepo.On("GetByResourceID", mock.Anything, models.ResourceTypeFolder, folderID, s.testTenantID).Return([]*models.Permission{}, nil).Once()
	
	// Act - Create permission
	createdPermissionID, err := s.folderUseCase.CreateFolderPermission(ctx, folderID, roleID, permissionType, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(permissionID, createdPermissionID)
	
	// Act - Get permissions
	permissions, err := s.folderUseCase.GetFolderPermissions(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Require().Len(permissions, 1)
	s.Equal(permissionID, permissions[0].ID)
	s.Equal(roleID, permissions[0].RoleID)
	s.Equal(permissionType, permissions[0].PermissionType)
	
	// Act - Delete permission
	err = s.folderUseCase.DeleteFolderPermission(ctx, permissionID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	
	// Act - Get permissions after deletion
	permissions, err = s.folderUseCase.GetFolderPermissions(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Empty(permissions)
}

// TestTenantIsolation tests that folders are properly isolated between tenants
func (s *FolderFlowTestSuite) TestTenantIsolation() {
	// Arrange
	ctx := context.Background()
	folderName := "Tenant Isolation Test Folder"
	differentTenantID := uuid.New().String()
	
	// Create a test folder for test tenant
	folderID, err := s.createTestFolder(folderName, "")
	s.Require().NoError(err)
	
	// Set up mock for folder
	folder := models.NewFolder(folderName, "", s.testTenantID, s.testUserID)
	folder.ID = folderID
	folder.Path = "/" + folderName
	
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyTenantAccess", mock.Anything, s.testUserID, differentTenantID).Return(true, nil)
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(folder, nil)
	
	// Set up mock for folder retrieval with different tenant ID (should fail)
	s.folderRepo.On("GetByID", mock.Anything, folderID, differentTenantID).Return(nil, errors.NewResourceNotFoundError("folder not found"))
	
	// Set up mock for folder listing with different tenant ID (should return empty list)
	emptyResult := utils.PaginatedResult[models.Folder]{
		Items: []models.Folder{},
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  0,
			TotalItems:  0,
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	s.folderRepo.On("GetRootFolders", mock.Anything, differentTenantID, mock.Anything).Return(emptyResult, nil)
	
	// Act - Get folder with correct tenant ID
	retrievedFolder, err := s.folderUseCase.GetFolder(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(folderName, retrievedFolder.Name)
	s.Equal(s.testTenantID, retrievedFolder.TenantID)
	
	// Act - Get folder with different tenant ID
	_, err = s.folderUseCase.GetFolder(ctx, folderID, differentTenantID, s.testUserID)
	
	// Assert
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	
	// Act - List root folders with different tenant ID
	rootFolders, err := s.folderUseCase.ListRootFolders(ctx, differentTenantID, s.testUserID, utils.NewPagination(1, 10))
	
	// Assert
	s.Require().NoError(err)
	s.Empty(rootFolders.Items)
}

// TestPermissionChecks tests that permission checks are enforced
func (s *FolderFlowTestSuite) TestPermissionChecks() {
	// Arrange
	ctx := context.Background()
	folderName := "Permission Check Test Folder"
	unauthorizedUserID := uuid.New().String()
	
	// Create a test folder
	folderID, err := s.createTestFolder(folderName, "")
	s.Require().NoError(err)
	
	// Set up mock for folder
	folder := models.NewFolder(folderName, "", s.testTenantID, s.testUserID)
	folder.ID = folderID
	folder.Path = "/" + folderName
	
	s.folderRepo.On("GetByID", mock.Anything, folderID, s.testTenantID).Return(folder, nil)
	
	// Set up mock for permission checks
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, unauthorizedUserID, s.testTenantID, services.ResourceTypeFolder, folderID, services.PermissionRead).Return(false, nil)
	
	// Act - Authorized user can get folder
	_, err = s.folderUseCase.GetFolder(ctx, folderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	
	// Act - Unauthorized user cannot get folder
	_, err = s.folderUseCase.GetFolder(ctx, folderID, s.testTenantID, unauthorizedUserID)
	
	// Assert
	s.Require().Error(err)
	s.True(errors.IsAuthorizationError(err) || errors.IsPermissionDeniedError(err))
}

// TestFolderPathOperations tests retrieving folders by path
func (s *FolderFlowTestSuite) TestFolderPathOperations() {
	// Arrange
	ctx := context.Background()
	
	// Create a folder hierarchy with known paths
	rootFolderID, err := s.createTestFolder("Path Test Root", "")
	s.Require().NoError(err)
	
	childFolderID, err := s.createTestFolder("Path Test Child", rootFolderID)
	s.Require().NoError(err)
	
	grandchildFolderID, err := s.createTestFolder("Path Test Grandchild", childFolderID)
	s.Require().NoError(err)
	
	// Set up mock folders
	rootFolder := models.NewFolder("Path Test Root", "", s.testTenantID, s.testUserID)
	rootFolder.ID = rootFolderID
	rootFolder.Path = "/Path Test Root"
	
	childFolder := models.NewFolder("Path Test Child", rootFolderID, s.testTenantID, s.testUserID)
	childFolder.ID = childFolderID
	childFolder.Path = "/Path Test Root/Path Test Child"
	
	grandchildFolder := models.NewFolder("Path Test Grandchild", childFolderID, s.testTenantID, s.testUserID)
	grandchildFolder.ID = grandchildFolderID
	grandchildFolder.Path = "/Path Test Root/Path Test Child/Path Test Grandchild"
	
	// Set up mock for folder retrieval by path
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, rootFolderID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, childFolderID, services.PermissionRead).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, grandchildFolderID, services.PermissionRead).Return(true, nil)
	
	s.folderRepo.On("GetByPath", mock.Anything, "/Path Test Root", s.testTenantID).Return(rootFolder, nil)
	s.folderRepo.On("GetByPath", mock.Anything, "/Path Test Root/Path Test Child", s.testTenantID).Return(childFolder, nil)
	s.folderRepo.On("GetByPath", mock.Anything, "/Path Test Root/Path Test Child/Path Test Grandchild", s.testTenantID).Return(grandchildFolder, nil)
	s.folderRepo.On("GetByPath", mock.Anything, "/Invalid/Path", s.testTenantID).Return(nil, errors.NewResourceNotFoundError("folder not found"))
	
	// Act - Get root folder by path
	folder, err := s.folderUseCase.GetFolderByPath(ctx, "/Path Test Root", s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(rootFolderID, folder.ID)
	
	// Act - Get child folder by path
	folder, err = s.folderUseCase.GetFolderByPath(ctx, "/Path Test Root/Path Test Child", s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(childFolderID, folder.ID)
	
	// Act - Get grandchild folder by path
	folder, err = s.folderUseCase.GetFolderByPath(ctx, "/Path Test Root/Path Test Child/Path Test Grandchild", s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().NoError(err)
	s.Equal(grandchildFolderID, folder.ID)
	
	// Act - Get folder with invalid path
	_, err = s.folderUseCase.GetFolderByPath(ctx, "/Invalid/Path", s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
}

// TestCircularReferenceDetection tests that circular references are detected and prevented
func (s *FolderFlowTestSuite) TestCircularReferenceDetection() {
	// Arrange
	ctx := context.Background()
	
	// Create a folder hierarchy
	rootFolderID, err := s.createTestFolder("Circular Root", "")
	s.Require().NoError(err)
	
	childFolderID, err := s.createTestFolder("Circular Child", rootFolderID)
	s.Require().NoError(err)
	
	grandchildFolderID, err := s.createTestFolder("Circular Grandchild", childFolderID)
	s.Require().NoError(err)
	
	// Set up mock folders
	rootFolder := models.NewFolder("Circular Root", "", s.testTenantID, s.testUserID)
	rootFolder.ID = rootFolderID
	rootFolder.Path = "/Circular Root"
	
	childFolder := models.NewFolder("Circular Child", rootFolderID, s.testTenantID, s.testUserID)
	childFolder.ID = childFolderID
	childFolder.Path = "/Circular Root/Circular Child"
	
	grandchildFolder := models.NewFolder("Circular Grandchild", childFolderID, s.testTenantID, s.testUserID)
	grandchildFolder.ID = grandchildFolderID
	grandchildFolder.Path = "/Circular Root/Circular Child/Circular Grandchild"
	
	// Set up mocks for folder retrieval
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, rootFolderID, services.PermissionWrite).Return(true, nil)
	mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, grandchildFolderID, services.PermissionWrite).Return(true, nil)
	
	s.folderRepo.On("GetByID", mock.Anything, rootFolderID, s.testTenantID).Return(rootFolder, nil)
	s.folderRepo.On("GetByID", mock.Anything, childFolderID, s.testTenantID).Return(childFolder, nil)
	s.folderRepo.On("GetByID", mock.Anything, grandchildFolderID, s.testTenantID).Return(grandchildFolder, nil)
	
	// Set up mock for checking circular reference
	s.folderRepo.On("GetFolderPath", mock.Anything, rootFolderID, s.testTenantID).Return("/Circular Root", nil)
	s.folderRepo.On("GetFolderPath", mock.Anything, grandchildFolderID, s.testTenantID).Return("/Circular Root/Circular Child/Circular Grandchild", nil)
	
	// Act - Try to move root folder under its grandchild (should detect circular reference)
	err = s.folderUseCase.MoveFolder(ctx, rootFolderID, grandchildFolderID, s.testTenantID, s.testUserID)
	
	// Assert
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
}

// createTestFolder is a helper function to create a test folder
func (s *FolderFlowTestSuite) createTestFolder(name, parentID string) (string, error) {
	// Set up mocks for folder creation
	folderID := uuid.New().String()
	
	if parentID == "" {
		mockAuthService.On("VerifyPermission", mock.Anything, s.testUserID, s.testTenantID, services.PermissionManageFolders).Return(true, nil).Maybe()
	} else {
		parentFolder := models.NewFolder("Parent", "", s.testTenantID, s.testUserID)
		parentFolder.ID = parentID
		parentFolder.Path = "/Parent" // Simplified for the test
		
		mockAuthService.On("VerifyPermission", mock.Anything, s.testUserID, s.testTenantID, services.PermissionManageFolders).Return(true, nil).Maybe()
		mockAuthService.On("VerifyResourceAccess", mock.Anything, s.testUserID, s.testTenantID, services.ResourceTypeFolder, parentID, services.PermissionWrite).Return(true, nil).Maybe()
		s.folderRepo.On("GetByID", mock.Anything, parentID, s.testTenantID).Return(parentFolder, nil).Maybe()
	}
	
	s.folderRepo.On("Create", mock.Anything, mock.MatchedBy(func(f *models.Folder) bool {
		return f.Name == name && f.ParentID == parentID && f.TenantID == s.testTenantID
	})).Return(folderID, nil).Maybe()
	
	mockPermissionRepo.On("Create", mock.Anything, mock.Anything).Return(uuid.New().String(), nil).Maybe()
	mockPermissionRepo.On("PropagatePermissions", mock.Anything, folderID, s.testTenantID).Return(nil).Maybe()
	
	s.eventService.On("CreateAndPublishFolderEvent", mock.Anything, services.FolderEventCreated, s.testTenantID, folderID, mock.Anything).Return(uuid.New().String(), nil).Maybe()
	
	// Call the folder creation method
	ctx := context.Background()
	return s.folderUseCase.CreateFolder(ctx, name, parentID, s.testTenantID, s.testUserID)
}

// Mock interfaces

// MockFolderRepository mocks the folder repository interface
type MockFolderRepository struct {
	mock.Mock
}

func (m *MockFolderRepository) Create(ctx context.Context, folder *models.Folder) (string, error) {
	args := m.Called(ctx, folder)
	return args.String(0), args.Error(1)
}

func (m *MockFolderRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.Folder, error) {
	args := m.Called(ctx, id, tenantID)
	folder, _ := args.Get(0).(*models.Folder)
	return folder, args.Error(1)
}

func (m *MockFolderRepository) Update(ctx context.Context, folder *models.Folder) error {
	args := m.Called(ctx, folder)
	return args.Error(0)
}

func (m *MockFolderRepository) Delete(ctx context.Context, id string, tenantID string) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockFolderRepository) GetChildren(ctx context.Context, parentID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	args := m.Called(ctx, parentID, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Folder]), args.Error(1)
}

func (m *MockFolderRepository) GetRootFolders(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	args := m.Called(ctx, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Folder]), args.Error(1)
}

func (m *MockFolderRepository) GetFolderPath(ctx context.Context, id string, tenantID string) (string, error) {
	args := m.Called(ctx, id, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockFolderRepository) GetByPath(ctx context.Context, path string, tenantID string) (*models.Folder, error) {
	args := m.Called(ctx, path, tenantID)
	folder, _ := args.Get(0).(*models.Folder)
	return folder, args.Error(1)
}

func (m *MockFolderRepository) Move(ctx context.Context, id string, newParentID string, tenantID string) error {
	args := m.Called(ctx, id, newParentID, tenantID)
	return args.Error(0)
}

func (m *MockFolderRepository) Exists(ctx context.Context, id string, tenantID string) (bool, error) {
	args := m.Called(ctx, id, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockFolderRepository) IsEmpty(ctx context.Context, id string, tenantID string) (bool, error) {
	args := m.Called(ctx, id, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockFolderRepository) Search(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	args := m.Called(ctx, query, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Folder]), args.Error(1)
}

// MockDocumentRepository mocks the document repository interface
type MockDocumentRepository struct {
	mock.Mock
}

func (m *MockDocumentRepository) ListByFolder(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, folderID, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

// MockPermissionRepository mocks the permission repository interface
type MockPermissionRepository struct {
	mock.Mock
}

func (m *MockPermissionRepository) Create(ctx context.Context, permission *models.Permission) (string, error) {
	args := m.Called(ctx, permission)
	return args.String(0), args.Error(1)
}

func (m *MockPermissionRepository) GetByID(ctx context.Context, id, tenantID string) (*models.Permission, error) {
	args := m.Called(ctx, id, tenantID)
	permission, _ := args.Get(0).(*models.Permission)
	return permission, args.Error(1)
}

func (m *MockPermissionRepository) Delete(ctx context.Context, id, tenantID string) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockPermissionRepository) GetByResourceID(ctx context.Context, resourceType, resourceID, tenantID string) ([]*models.Permission, error) {
	args := m.Called(ctx, resourceType, resourceID, tenantID)
	permissions, _ := args.Get(0).([]*models.Permission)
	return permissions, args.Error(1)
}

func (m *MockPermissionRepository) DeleteByResourceID(ctx context.Context, resourceType, resourceID, tenantID string) error {
	args := m.Called(ctx, resourceType, resourceID, tenantID)
	return args.Error(0)
}

func (m *MockPermissionRepository) PropagatePermissions(ctx context.Context, folderID, tenantID string) error {
	args := m.Called(ctx, folderID, tenantID)
	return args.Error(0)
}

func (m *MockPermissionRepository) GetInheritedPermissions(ctx context.Context, folderID, tenantID string) ([]*models.Permission, error) {
	args := m.Called(ctx, folderID, tenantID)
	permissions, _ := args.Get(0).([]*models.Permission)
	return permissions, args.Error(1)
}

// MockAuthService mocks the auth service interface
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) VerifyPermission(ctx context.Context, userID, tenantID, permission string) (bool, error) {
	args := m.Called(ctx, userID, tenantID, permission)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthService) VerifyResourceAccess(ctx context.Context, userID, tenantID, resourceType, resourceID, accessType string) (bool, error) {
	args := m.Called(ctx, userID, tenantID, resourceType, resourceID, accessType)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthService) VerifyTenantAccess(ctx context.Context, userID, tenantID string) (bool, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Bool(0), args.Error(1)
}

// MockEventService mocks the event service interface
type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) CreateAndPublishFolderEvent(ctx context.Context, eventType string, tenantID string, folderID string, additionalData map[string]interface{}) (string, error) {
	args := m.Called(ctx, eventType, tenantID, folderID, additionalData)
	return args.String(0), args.Error(1)
}

func (m *MockEventService) CreateAndPublishDocumentEvent(ctx context.Context, eventType string, tenantID string, documentID string, additionalData map[string]interface{}) (string, error) {
	args := m.Called(ctx, eventType, tenantID, documentID, additionalData)
	return args.String(0), args.Error(1)
}

func (m *MockEventService) CreateEvent(ctx context.Context, event *models.Event) (string, error) {
	args := m.Called(ctx, event)
	return args.String(0), args.Error(1)
}

func (m *MockEventService) PublishEvent(ctx context.Context, event *models.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// Global mocks for dependencies
var (
	mockDocumentRepo   *MockDocumentRepository
	mockPermissionRepo *MockPermissionRepository
	mockAuthService    *MockAuthService
)