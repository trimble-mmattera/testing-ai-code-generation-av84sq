package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid" // v1.3.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+
	"gorm.io/gorm" // v1.25.0+
	"gorm.io/driver/sqlite" // v1.5.0+

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/errors"
	"../../../pkg/utils"
)

// TestFolderRepositoryIntegration is the entry point for the folder repository integration test suite
func TestFolderRepositoryIntegration(t *testing.T) {
	suite.Run(t, new(FolderRepositoryTestSuite))
}

// FolderRepositoryTestSuite is a test suite for folder repository integration tests
type FolderRepositoryTestSuite struct {
	suite.Suite
	db           *gorm.DB
	repository   repositories.FolderRepository
	testTenantID string
	testOwnerID  string
}

// SetupSuite sets up the test suite before any tests run
func (s *FolderRepositoryTestSuite) SetupSuite() {
	// Initialize in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(s.T(), err, "Failed to create in-memory database")

	// Run migrations to create necessary tables
	err = db.AutoMigrate(&FolderModel{})
	require.NoError(s.T(), err, "Failed to run migrations")

	s.db = db
	s.repository = NewFolderRepository(db)
	
	// Generate test tenant ID and owner ID for isolation testing
	s.testTenantID = uuid.New().String()
	s.testOwnerID = uuid.New().String()
}

// TearDownSuite cleans up after all tests have run
func (s *FolderRepositoryTestSuite) TearDownSuite() {
	// Close database connection
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// SetupTest sets up each test before it runs
func (s *FolderRepositoryTestSuite) SetupTest() {
	// Clean up database tables to ensure test isolation
	s.db.Exec("DELETE FROM folder_models")
}

// TestCreateFolder tests folder creation functionality
func (s *FolderRepositoryTestSuite) TestCreateFolder() {
	// Create a test folder with valid data
	folder := models.NewFolder("Test Folder", "", s.testTenantID, s.testOwnerID)
	
	folderID, err := s.repository.Create(context.Background(), folder)
	
	// Assert that folder ID is returned and not empty
	require.NoError(s.T(), err, "Failed to create folder")
	assert.NotEmpty(s.T(), folderID, "Folder ID should not be empty")
	
	// Retrieve the folder from the database
	var folderModel FolderModel
	result := s.db.Where("id = ?", folderID).First(&folderModel)
	assert.NoError(s.T(), result.Error, "Failed to retrieve created folder")
	assert.Equal(s.T(), "Test Folder", folderModel.Name, "Folder name should match")
	assert.Equal(s.T(), s.testTenantID, folderModel.TenantID, "Tenant ID should match")
	
	// Test creating a folder with invalid data (empty name)
	invalidFolder := models.NewFolder("", "", s.testTenantID, s.testOwnerID)
	_, err = s.repository.Create(context.Background(), invalidFolder)
	assert.True(s.T(), errors.IsValidationError(err), "Should return validation error for empty name")
	
	// Test creating a folder with invalid tenant ID
	invalidFolder = models.NewFolder("Test Folder", "", "", s.testOwnerID)
	_, err = s.repository.Create(context.Background(), invalidFolder)
	assert.True(s.T(), errors.IsValidationError(err), "Should return validation error for empty tenant ID")
}

// TestGetFolderByID tests retrieving a folder by ID with tenant isolation
func (s *FolderRepositoryTestSuite) TestGetFolderByID() {
	// Create a test folder in the database
	folder, folderID, err := s.createTestFolder("Test Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create test folder")
	
	// Call repository.GetByID with the folder ID and correct tenant ID
	retrievedFolder, err := s.repository.GetByID(context.Background(), folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to retrieve folder")
	assert.Equal(s.T(), folder.Name, retrievedFolder.Name, "Folder name should match")
	assert.Equal(s.T(), folderID, retrievedFolder.ID, "Folder ID should match")
	
	// Call repository.GetByID with the folder ID and incorrect tenant ID
	differentTenantID := uuid.New().String()
	_, err = s.repository.GetByID(context.Background(), folderID, differentTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Call repository.GetByID with non-existent folder ID
	nonExistentID := uuid.New().String()
	_, err = s.repository.GetByID(context.Background(), nonExistentID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent ID")
}

// TestUpdateFolder tests updating a folder with tenant isolation
func (s *FolderRepositoryTestSuite) TestUpdateFolder() {
	// Create a test folder in the database
	folder, folderID, err := s.createTestFolder("Test Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create test folder")
	
	// Modify the folder name
	folder.Update("Updated Folder")
	err = s.repository.Update(context.Background(), folder)
	require.NoError(s.T(), err, "Failed to update folder")
	
	// Retrieve the folder from the database
	updatedFolder, err := s.repository.GetByID(context.Background(), folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to retrieve updated folder")
	assert.Equal(s.T(), "Updated Folder", updatedFolder.Name, "Folder name should be updated")
	
	// Create a folder for a different tenant
	differentTenantID := uuid.New().String()
	differentFolder, differentFolderID, err := s.createTestFolder("Different Tenant Folder", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create folder for different tenant")
	
	// Try to update the folder with incorrect tenant ID
	differentFolder.Update("Should not update")
	differentFolder.TenantID = s.testTenantID // Change tenant ID
	err = s.repository.Update(context.Background(), differentFolder)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Try to update a non-existent folder
	nonExistentFolder := models.NewFolder("Non-existent", "", s.testTenantID, s.testOwnerID)
	nonExistentFolder.ID = uuid.New().String()
	err = s.repository.Update(context.Background(), nonExistentFolder)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent ID")
}

// TestDeleteFolder tests deleting a folder with tenant isolation
func (s *FolderRepositoryTestSuite) TestDeleteFolder() {
	// Create a test folder in the database
	_, folderID, err := s.createTestFolder("Test Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create test folder")
	
	// Call repository.Delete with the folder ID and correct tenant ID
	err = s.repository.Delete(context.Background(), folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to delete folder")
	
	// Try to retrieve the deleted folder
	_, err = s.repository.GetByID(context.Background(), folderID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Folder should be deleted")
	
	// Create a folder for a different tenant
	differentTenantID := uuid.New().String()
	_, differentFolderID, err := s.createTestFolder("Different Tenant Folder", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create folder for different tenant")
	
	// Try to delete the folder with incorrect tenant ID
	err = s.repository.Delete(context.Background(), differentFolderID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Create a folder with child folders
	parentFolder, parentID, err := s.createTestFolder("Parent Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create parent folder")
	
	_, _, err = s.createTestFolder("Child Folder", parentID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create child folder")
	
	// Try to delete the parent folder
	err = s.repository.Delete(context.Background(), parentID, s.testTenantID)
	assert.Error(s.T(), err, "Should not be able to delete folder with children")
}

// TestGetChildFolders tests retrieving child folders with pagination and tenant isolation
func (s *FolderRepositoryTestSuite) TestGetChildFolders() {
	// Create a parent folder in the database
	_, parentID, err := s.createTestFolder("Parent Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create parent folder")
	
	// Create multiple child folders under the parent folder
	childNames := []string{"Child 1", "Child 2", "Child 3", "Child 4", "Child 5"}
	for _, name := range childNames {
		_, _, err := s.createTestFolder(name, parentID, s.testTenantID, s.testOwnerID)
		require.NoError(s.T(), err, "Failed to create child folder")
	}
	
	// Call repository.GetChildren with parent ID, tenant ID, and pagination
	pagination := utils.NewPagination(1, 2)
	result, err := s.repository.GetChildren(context.Background(), parentID, s.testTenantID, pagination)
	require.NoError(s.T(), err, "Failed to get child folders")
	assert.Len(s.T(), result.Items, 2, "Should return 2 folders")
	assert.Equal(s.T(), int64(5), result.Pagination.TotalItems, "Total should be 5")
	assert.Equal(s.T(), 3, result.Pagination.TotalPages, "Should have 3 pages")
	assert.True(s.T(), result.Pagination.HasNext, "Should have next page")
	assert.False(s.T(), result.Pagination.HasPrevious, "Should not have previous page")
	
	// Create folders for a different tenant
	differentTenantID := uuid.New().String()
	_, differentParentID, err := s.createTestFolder("Different Parent", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create parent folder for different tenant")
	
	_, _, err = s.createTestFolder("Different Child", differentParentID, differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create child folder for different tenant")
	
	// Call repository.GetChildren with parent ID and incorrect tenant ID
	_, err = s.repository.GetChildren(context.Background(), differentParentID, s.testTenantID, pagination)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Test with non-existent parent folder
	nonExistentID := uuid.New().String()
	_, err = s.repository.GetChildren(context.Background(), nonExistentID, s.testTenantID, pagination)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent ID")
}

// TestGetRootFolders tests retrieving root folders with pagination and tenant isolation
func (s *FolderRepositoryTestSuite) TestGetRootFolders() {
	// Create multiple root folders (no parent ID) for the test tenant
	rootNames := []string{"Root 1", "Root 2", "Root 3", "Root 4", "Root 5"}
	for _, name := range rootNames {
		_, _, err := s.createTestFolder(name, "", s.testTenantID, s.testOwnerID)
		require.NoError(s.T(), err, "Failed to create root folder")
	}
	
	// Call repository.GetRootFolders with tenant ID and pagination
	pagination := utils.NewPagination(1, 3)
	result, err := s.repository.GetRootFolders(context.Background(), s.testTenantID, pagination)
	require.NoError(s.T(), err, "Failed to get root folders")
	assert.Len(s.T(), result.Items, 3, "Should return 3 folders")
	assert.Equal(s.T(), int64(5), result.Pagination.TotalItems, "Total should be 5")
	assert.Equal(s.T(), 2, result.Pagination.TotalPages, "Should have 2 pages")
	
	// Create root folders for a different tenant
	differentTenantID := uuid.New().String()
	_, _, err = s.createTestFolder("Different Root", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create root folder for different tenant")
	
	// Call repository.GetRootFolders with incorrect tenant ID
	result, err = s.repository.GetRootFolders(context.Background(), differentTenantID, pagination)
	require.NoError(s.T(), err, "Failed to get root folders for different tenant")
	assert.Len(s.T(), result.Items, 1, "Should return 1 folder")
	assert.Equal(s.T(), "Different Root", result.Items[0].Name, "Name should match")
	
	// Ensure tenant isolation works
	for _, folder := range result.Items {
		assert.Equal(s.T(), differentTenantID, folder.TenantID, "Folder should belong to the correct tenant")
	}
}

// TestGetFolderPath tests retrieving the full path of a folder with tenant isolation
func (s *FolderRepositoryTestSuite) TestGetFolderPath() {
	// Create a folder hierarchy (parent/child/grandchild)
	_, rootID, err := s.createTestFolder("Root", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create root folder")
	
	_, childID, err := s.createTestFolder("Child", rootID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create child folder")
	
	_, grandchildID, err := s.createTestFolder("Grandchild", childID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create grandchild folder")
	
	// Call repository.GetFolderPath with grandchild ID and correct tenant ID
	path, err := s.repository.GetFolderPath(context.Background(), grandchildID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to get folder path")
	expectedPath := models.PathSeparator + "Root" + models.PathSeparator + "Child" + models.PathSeparator + "Grandchild"
	assert.Equal(s.T(), expectedPath, path, "Path should match")
	
	// Create a folder for a different tenant
	differentTenantID := uuid.New().String()
	_, differentFolderID, err := s.createTestFolder("Different Folder", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create folder for different tenant")
	
	// Call repository.GetFolderPath with folder ID and incorrect tenant ID
	_, err = s.repository.GetFolderPath(context.Background(), differentFolderID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Test with non-existent folder ID
	nonExistentID := uuid.New().String()
	_, err = s.repository.GetFolderPath(context.Background(), nonExistentID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent ID")
}

// TestGetFolderByPath tests retrieving a folder by its path with tenant isolation
func (s *FolderRepositoryTestSuite) TestGetFolderByPath() {
	// Create a folder hierarchy with known paths
	_, rootID, err := s.createTestFolder("Root", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create root folder")
	
	_, childID, err := s.createTestFolder("Child", rootID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create child folder")
	
	_, grandchildID, err := s.createTestFolder("Grandchild", childID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create grandchild folder")
	
	// Define the path
	testPath := models.PathSeparator + "Root" + models.PathSeparator + "Child" + models.PathSeparator + "Grandchild"
	
	// Call repository.GetByPath with folder path and correct tenant ID
	folder, err := s.repository.GetByPath(context.Background(), testPath, s.testTenantID)
	require.NoError(s.T(), err, "Failed to get folder by path")
	assert.Equal(s.T(), grandchildID, folder.ID, "Folder ID should match")
	assert.Equal(s.T(), "Grandchild", folder.Name, "Folder name should match")
	
	// Create a folder with the same path for a different tenant
	differentTenantID := uuid.New().String()
	folders, err := s.createFolderHierarchy(differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create folder hierarchy for different tenant")
	
	// Call repository.GetByPath with folder path and incorrect tenant ID
	_, err = s.repository.GetByPath(context.Background(), testPath, differentTenantID)
	assert.Error(s.T(), err, "Should return error for different tenant path")
	
	// Test with non-existent path
	nonExistentPath := models.PathSeparator + "NonExistent" + models.PathSeparator + "Path"
	_, err = s.repository.GetByPath(context.Background(), nonExistentPath, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent path")
}

// TestMoveFolder tests moving a folder to a new parent with tenant isolation
func (s *FolderRepositoryTestSuite) TestMoveFolder() {
	// Create a folder hierarchy (parent1/child, parent2)
	_, parent1ID, err := s.createTestFolder("Parent1", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create parent1 folder")
	
	child, childID, err := s.createTestFolder("Child", parent1ID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create child folder")
	
	_, parent2ID, err := s.createTestFolder("Parent2", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create parent2 folder")
	
	// Call repository.Move to move child from parent1 to parent2
	err = s.repository.Move(context.Background(), childID, parent2ID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to move folder")
	
	// Retrieve the moved folder
	movedFolder, err := s.repository.GetByID(context.Background(), childID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to get moved folder")
	assert.Equal(s.T(), parent2ID, movedFolder.ParentID, "Parent ID should be updated")
	
	// Assert that folder path is updated
	path, err := s.repository.GetFolderPath(context.Background(), childID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to get folder path")
	expectedPath := models.PathSeparator + "Parent2" + models.PathSeparator + "Child"
	assert.Equal(s.T(), expectedPath, path, "Path should be updated")
	
	// Create folders for a different tenant
	differentTenantID := uuid.New().String()
	_, differentFolderID, err := s.createTestFolder("Different Folder", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create folder for different tenant")
	
	// Try to move a folder with incorrect tenant ID
	err = s.repository.Move(context.Background(), differentFolderID, parent2ID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Try to move a folder to a non-existent parent
	nonExistentID := uuid.New().String()
	err = s.repository.Move(context.Background(), childID, nonExistentID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent parent")
	
	// Try to move a folder to its own descendant (circular reference)
	_, grandchildID, err := s.createTestFolder("Grandchild", childID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create grandchild folder")
	
	err = s.repository.Move(context.Background(), parent2ID, grandchildID, s.testTenantID)
	assert.Error(s.T(), err, "Should not be able to move folder to its own descendant")
}

// TestFolderExists tests checking if a folder exists with tenant isolation
func (s *FolderRepositoryTestSuite) TestFolderExists() {
	// Create a test folder in the database
	_, folderID, err := s.createTestFolder("Test Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create test folder")
	
	// Call repository.Exists with folder ID and correct tenant ID
	exists, err := s.repository.Exists(context.Background(), folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to check if folder exists")
	assert.True(s.T(), exists, "Folder should exist")
	
	// Call repository.Exists with folder ID and incorrect tenant ID
	differentTenantID := uuid.New().String()
	exists, err = s.repository.Exists(context.Background(), folderID, differentTenantID)
	require.NoError(s.T(), err, "Failed to check if folder exists with different tenant")
	assert.False(s.T(), exists, "Folder should not exist for different tenant")
	
	// Call repository.Exists with non-existent folder ID
	nonExistentID := uuid.New().String()
	exists, err = s.repository.Exists(context.Background(), nonExistentID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to check if non-existent folder exists")
	assert.False(s.T(), exists, "Non-existent folder should not exist")
}

// TestIsFolderEmpty tests checking if a folder is empty with tenant isolation
func (s *FolderRepositoryTestSuite) TestIsFolderEmpty() {
	// Create a test folder in the database
	_, folderID, err := s.createTestFolder("Test Folder", "", s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create test folder")
	
	// Call repository.IsEmpty with folder ID and correct tenant ID
	isEmpty, err := s.repository.IsEmpty(context.Background(), folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to check if folder is empty")
	assert.True(s.T(), isEmpty, "New folder should be empty")
	
	// Create a child folder under the test folder
	_, _, err = s.createTestFolder("Child Folder", folderID, s.testTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create child folder")
	
	// Call repository.IsEmpty with parent folder ID
	isEmpty, err = s.repository.IsEmpty(context.Background(), folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to check if folder is empty")
	assert.False(s.T(), isEmpty, "Folder with child should not be empty")
	
	// Call repository.IsEmpty with folder ID and incorrect tenant ID
	differentTenantID := uuid.New().String()
	_, err = s.repository.IsEmpty(context.Background(), folderID, differentTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for different tenant")
	
	// Call repository.IsEmpty with non-existent folder ID
	nonExistentID := uuid.New().String()
	_, err = s.repository.IsEmpty(context.Background(), nonExistentID, s.testTenantID)
	assert.True(s.T(), errors.IsResourceNotFoundError(err), "Should return not found for non-existent ID")
}

// TestSearchFolders tests searching folders by name with tenant isolation
func (s *FolderRepositoryTestSuite) TestSearchFolders() {
	// Create multiple folders with different names
	folderNames := []string{"Alpha Folder", "Beta Folder", "Alpha Test", "Gamma Folder", "Alpha Beta"}
	for _, name := range folderNames {
		_, _, err := s.createTestFolder(name, "", s.testTenantID, s.testOwnerID)
		require.NoError(s.T(), err, "Failed to create folder")
	}
	
	// Call repository.Search with search query matching some folders
	pagination := utils.NewPagination(1, 10)
	result, err := s.repository.Search(context.Background(), "Alpha", s.testTenantID, pagination)
	require.NoError(s.T(), err, "Failed to search folders")
	assert.Len(s.T(), result.Items, 3, "Should find 3 folders")
	
	// Verify that correct folders are returned in search results
	folderMap := make(map[string]bool)
	for _, folder := range result.Items {
		folderMap[folder.Name] = true
	}
	assert.True(s.T(), folderMap["Alpha Folder"], "Should find 'Alpha Folder'")
	assert.True(s.T(), folderMap["Alpha Test"], "Should find 'Alpha Test'")
	assert.True(s.T(), folderMap["Alpha Beta"], "Should find 'Alpha Beta'")
	
	// Create folders with similar names for a different tenant
	differentTenantID := uuid.New().String()
	_, _, err = s.createTestFolder("Alpha Different", "", differentTenantID, s.testOwnerID)
	require.NoError(s.T(), err, "Failed to create folder for different tenant")
	
	// Call repository.Search with same query but different tenant ID
	result, err = s.repository.Search(context.Background(), "Alpha", differentTenantID, pagination)
	require.NoError(s.T(), err, "Failed to search folders for different tenant")
	assert.Len(s.T(), result.Items, 1, "Should find 1 folder")
	assert.Equal(s.T(), "Alpha Different", result.Items[0].Name, "Should find 'Alpha Different'")
	
	// Test with query that matches no folders
	result, err = s.repository.Search(context.Background(), "NonExistent", s.testTenantID, pagination)
	require.NoError(s.T(), err, "Failed to search folders with no matches")
	assert.Len(s.T(), result.Items, 0, "Should find 0 folders")
}

// Helper function to create a test folder
func (s *FolderRepositoryTestSuite) createTestFolder(name, parentID, tenantID, ownerID string) (*models.Folder, string, error) {
	// Create a new folder using models.NewFolder
	folder := models.NewFolder(name, parentID, tenantID, ownerID)
	
	// Call repository.Create to persist the folder
	folderID, err := s.repository.Create(context.Background(), folder)
	if err != nil {
		return nil, "", err
	}
	
	folder.ID = folderID
	return folder, folderID, nil
}

// Helper function to create a folder hierarchy for testing
func (s *FolderRepositoryTestSuite) createFolderHierarchy(tenantID, ownerID string) (map[string]*models.Folder, error) {
	// Create root folder
	folders := make(map[string]*models.Folder)
	
	root, rootID, err := s.createTestFolder("Root", "", tenantID, ownerID)
	if err != nil {
		return nil, err
	}
	folders["Root"] = root
	
	// Create multiple child folders under root
	child, childID, err := s.createTestFolder("Child", rootID, tenantID, ownerID)
	if err != nil {
		return nil, err
	}
	folders["Child"] = child
	
	// Create grandchild folders under some children
	grandchild, _, err := s.createTestFolder("Grandchild", childID, tenantID, ownerID)
	if err != nil {
		return nil, err
	}
	folders["Grandchild"] = grandchild
	
	return folders, nil
}

// FolderModel is the database model for folders
type FolderModel struct {
	ID        string `gorm:"primaryKey"`
	Name      string `gorm:"size:255;not null"`
	ParentID  string `gorm:"index"`
	Path      string `gorm:"index"`
	TenantID  string `gorm:"index;not null"`
	OwnerID   string `gorm:"not null"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

// TableName returns the table name for the FolderModel
func (FolderModel) TableName() string {
	return "folder_models"
}