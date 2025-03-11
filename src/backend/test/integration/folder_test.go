package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"../../domain/models"
	"../../domain/repositories"
	"../../infrastructure/persistence/postgres"
	"../../pkg/config"
	"../../pkg/utils"
)

// FolderTestSuite is the test suite for folder repository integration tests
type FolderTestSuite struct {
	suite.Suite
	folderRepo   repositories.FolderRepository
	testTenantID string
	testUserID   string
	ctx          context.Context
}

// SetupSuite sets up the test suite before any tests run
func (s *FolderTestSuite) SetupSuite() {
	// Get database configuration from environment or use test defaults
	dbConfig := config.DatabaseConfig{
		Host:            getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            getEnvOrDefaultInt("TEST_DB_PORT", 5432),
		User:            getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		DBName:          getEnvOrDefault("TEST_DB_NAME", "document_mgmt_test"),
		SSLMode:         getEnvOrDefault("TEST_DB_SSL_MODE", "disable"),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: "1h",
	}

	// Initialize database connection
	err := postgres.Init(dbConfig)
	require.NoError(s.T(), err, "Failed to initialize database connection")

	// Run migrations to ensure schema is up to date
	err = postgres.Migrate(&models.Folder{})
	require.NoError(s.T(), err, "Failed to migrate database schema")

	// Create folder repository instance
	s.folderRepo = postgres.NewFolderRepository()
	require.NotNil(s.T(), s.folderRepo, "Failed to create folder repository")

	// Generate test tenant ID and user ID
	s.testTenantID = uuid.New().String()
	s.testUserID = uuid.New().String()

	// Initialize context for tests
	s.ctx = context.Background()
}

// TearDownSuite cleans up after all tests have run
func (s *FolderTestSuite) TearDownSuite() {
	// Close database connection
	err := postgres.Close()
	require.NoError(s.T(), err, "Failed to close database connection")
}

// SetupTest sets up each test before it runs
func (s *FolderTestSuite) SetupTest() {
	// Clean up any folders created by previous tests
	s.cleanupTestFolders()
}

// TearDownTest cleans up after each test
func (s *FolderTestSuite) TearDownTest() {
	// Clean up any folders created during the test
	s.cleanupTestFolders()
}

// Helper functions for environment variables
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// cleanupTestFolders removes all test folders created for the test tenant
func (s *FolderTestSuite) cleanupTestFolders() {
	db, err := postgres.GetDB()
	require.NoError(s.T(), err, "Failed to get database connection")

	result := db.Exec("DELETE FROM folders WHERE tenant_id = ?", s.testTenantID)
	require.NoError(s.T(), result.Error, "Failed to cleanup test folders")
}

// createTestFolder creates a test folder with the given name and parent ID
func (s *FolderTestSuite) createTestFolder(name, parentID string) (*models.Folder, string) {
	folder := models.NewFolder(name, parentID, s.testTenantID, s.testUserID)
	folderID, err := s.folderRepo.Create(s.ctx, folder)
	require.NoError(s.T(), err, "Failed to create test folder")
	require.NotEmpty(s.T(), folderID, "Folder ID should not be empty")

	createdFolder, err := s.folderRepo.GetByID(s.ctx, folderID, s.testTenantID)
	require.NoError(s.T(), err, "Failed to retrieve created folder")
	require.NotNil(s.T(), createdFolder, "Created folder should not be nil")

	return createdFolder, folderID
}

// TestCreateFolder tests folder creation functionality
func (s *FolderTestSuite) TestCreateFolder() {
	// Create a new folder using the repository
	folderName := "Test Folder"
	folder := models.NewFolder(folderName, "", s.testTenantID, s.testUserID)
	
	folderID, err := s.folderRepo.Create(s.ctx, folder)
	require.NoError(s.T(), err, "Should create folder without error")
	require.NotEmpty(s.T(), folderID, "Folder ID should not be empty")
	
	// Retrieve the created folder
	createdFolder, err := s.folderRepo.GetByID(s.ctx, folderID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve folder without error")
	require.NotNil(s.T(), createdFolder, "Retrieved folder should not be nil")
	
	// Verify folder properties
	assert.Equal(s.T(), folderName, createdFolder.Name, "Folder name should match")
	assert.Equal(s.T(), "", createdFolder.ParentID, "Parent ID should be empty for root folder")
	assert.Equal(s.T(), s.testTenantID, createdFolder.TenantID, "Tenant ID should match")
	assert.Equal(s.T(), s.testUserID, createdFolder.OwnerID, "Owner ID should match")
	assert.NotEmpty(s.T(), createdFolder.Path, "Path should not be empty")
	assert.Equal(s.T(), models.PathSeparator+folderName, createdFolder.Path, "Path should be correctly generated")
}

// TestCreateNestedFolder tests creation of nested folders
func (s *FolderTestSuite) TestCreateNestedFolder() {
	// Create a parent folder
	parentFolder, parentID := s.createTestFolder("Parent Folder", "")
	
	// Create a child folder with parent ID
	childName := "Child Folder"
	childFolder := models.NewFolder(childName, parentID, s.testTenantID, s.testUserID)
	
	childID, err := s.folderRepo.Create(s.ctx, childFolder)
	require.NoError(s.T(), err, "Should create child folder without error")
	require.NotEmpty(s.T(), childID, "Child folder ID should not be empty")
	
	// Retrieve the child folder
	retrievedChild, err := s.folderRepo.GetByID(s.ctx, childID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve child folder without error")
	require.NotNil(s.T(), retrievedChild, "Retrieved child folder should not be nil")
	
	// Verify child folder properties
	assert.Equal(s.T(), childName, retrievedChild.Name, "Child folder name should match")
	assert.Equal(s.T(), parentID, retrievedChild.ParentID, "Parent ID should match")
	assert.Equal(s.T(), s.testTenantID, retrievedChild.TenantID, "Tenant ID should match")
	assert.Equal(s.T(), s.testUserID, retrievedChild.OwnerID, "Owner ID should match")
	assert.NotEmpty(s.T(), retrievedChild.Path, "Path should not be empty")
	assert.True(s.T(), strings.HasPrefix(retrievedChild.Path, parentFolder.Path), "Child path should start with parent path")
}

// TestGetByID tests retrieving a folder by ID with tenant isolation
func (s *FolderTestSuite) TestGetByID() {
	// Create a test folder
	_, folderID := s.createTestFolder("Get By ID Folder", "")
	
	// Retrieve the folder by ID with correct tenant ID
	folder, err := s.folderRepo.GetByID(s.ctx, folderID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve folder with correct tenant ID")
	require.NotNil(s.T(), folder, "Retrieved folder should not be nil")
	
	// Attempt to retrieve the folder with incorrect tenant ID
	wrongTenantID := uuid.New().String()
	_, err = s.folderRepo.GetByID(s.ctx, folderID, wrongTenantID)
	assert.Error(s.T(), err, "Should not retrieve folder with incorrect tenant ID")
}

// TestGetByPath tests retrieving a folder by path with tenant isolation
func (s *FolderTestSuite) TestGetByPath() {
	// Create a test folder with a specific path
	folderName := "Path Test Folder"
	folder, _ := s.createTestFolder(folderName, "")
	folderPath := folder.Path
	
	// Retrieve the folder by path with correct tenant ID
	retrievedFolder, err := s.folderRepo.GetByPath(s.ctx, folderPath, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve folder with correct tenant ID")
	require.NotNil(s.T(), retrievedFolder, "Retrieved folder should not be nil")
	assert.Equal(s.T(), folderName, retrievedFolder.Name, "Folder name should match")
	
	// Attempt to retrieve the folder with incorrect tenant ID
	wrongTenantID := uuid.New().String()
	_, err = s.folderRepo.GetByPath(s.ctx, folderPath, wrongTenantID)
	assert.Error(s.T(), err, "Should not retrieve folder with incorrect tenant ID")
}

// TestUpdateFolder tests updating a folder's metadata
func (s *FolderTestSuite) TestUpdateFolder() {
	// Create a test folder
	folder, _ := s.createTestFolder("Update Test Folder", "")
	
	// Update the folder's name
	newName := "Updated Folder Name"
	folder.Update(newName)
	
	err := s.folderRepo.Update(s.ctx, folder)
	require.NoError(s.T(), err, "Should update folder without error")
	
	// Retrieve the updated folder
	updatedFolder, err := s.folderRepo.GetByID(s.ctx, folder.ID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve updated folder without error")
	require.NotNil(s.T(), updatedFolder, "Updated folder should not be nil")
	
	// Verify folder properties
	assert.Equal(s.T(), newName, updatedFolder.Name, "Folder name should be updated")
	assert.Equal(s.T(), folder.ParentID, updatedFolder.ParentID, "Parent ID should remain unchanged")
	assert.Equal(s.T(), folder.TenantID, updatedFolder.TenantID, "Tenant ID should remain unchanged")
	assert.Equal(s.T(), folder.OwnerID, updatedFolder.OwnerID, "Owner ID should remain unchanged")
}

// TestDeleteFolder tests folder deletion with tenant isolation
func (s *FolderTestSuite) TestDeleteFolder() {
	// Create a test folder
	_, folderID := s.createTestFolder("Delete Test Folder", "")
	
	// Delete the folder with correct tenant ID
	err := s.folderRepo.Delete(s.ctx, folderID, s.testTenantID)
	require.NoError(s.T(), err, "Should delete folder with correct tenant ID")
	
	// Attempt to retrieve the deleted folder
	_, err = s.folderRepo.GetByID(s.ctx, folderID, s.testTenantID)
	assert.Error(s.T(), err, "Should not find deleted folder")
	
	// Create another test folder
	_, folderID = s.createTestFolder("Delete Test Folder 2", "")
	
	// Attempt to delete with incorrect tenant ID
	wrongTenantID := uuid.New().String()
	err = s.folderRepo.Delete(s.ctx, folderID, wrongTenantID)
	assert.Error(s.T(), err, "Should not delete folder with incorrect tenant ID")
	
	// Verify folder still exists
	existingFolder, err := s.folderRepo.GetByID(s.ctx, folderID, s.testTenantID)
	require.NoError(s.T(), err, "Folder should still exist")
	require.NotNil(s.T(), existingFolder, "Folder should still exist")
}

// TestDeleteNonEmptyFolder tests deletion of non-empty folders
func (s *FolderTestSuite) TestDeleteNonEmptyFolder() {
	// Create a parent folder
	_, parentID := s.createTestFolder("Parent For Delete", "")
	
	// Create a child folder with parent ID
	_, childID := s.createTestFolder("Child For Delete", parentID)
	
	// Attempt to delete the parent folder
	err := s.folderRepo.Delete(s.ctx, parentID, s.testTenantID)
	assert.Error(s.T(), err, "Should not delete non-empty folder")
	
	// Delete the child folder first
	err = s.folderRepo.Delete(s.ctx, childID, s.testTenantID)
	require.NoError(s.T(), err, "Should delete child folder")
	
	// Attempt to delete the parent folder again
	err = s.folderRepo.Delete(s.ctx, parentID, s.testTenantID)
	require.NoError(s.T(), err, "Should delete empty parent folder")
}

// TestGetChildren tests retrieving child folders with pagination
func (s *FolderTestSuite) TestGetChildren() {
	// Create a parent folder
	_, parentID := s.createTestFolder("Parent For Children", "")
	
	// Create multiple child folders (more than one page)
	childCount := 25 // More than one page with default pagination
	for i := 0; i < childCount; i++ {
		s.createTestFolder(fmt.Sprintf("Child %d", i), parentID)
	}
	
	// Retrieve first page of child folders
	pagination := utils.NewPagination(1, 10)
	result, err := s.folderRepo.GetChildren(s.ctx, parentID, s.testTenantID, pagination)
	require.NoError(s.T(), err, "Should retrieve children without error")
	assert.Len(s.T(), result.Items, 10, "Should return correct number of items")
	assert.Equal(s.T(), int64(childCount), result.Pagination.TotalItems, "Should return correct total count")
	assert.True(s.T(), result.Pagination.HasNext, "Should have next page")
	
	// Retrieve second page of child folders
	pagination = utils.NewPagination(2, 10)
	result, err = s.folderRepo.GetChildren(s.ctx, parentID, s.testTenantID, pagination)
	require.NoError(s.T(), err, "Should retrieve second page without error")
	assert.Len(s.T(), result.Items, 10, "Should return correct number of items")
	assert.Equal(s.T(), int64(childCount), result.Pagination.TotalItems, "Should return correct total count")
	
	// Verify all child folders have correct parent ID
	for _, folder := range result.Items {
		assert.Equal(s.T(), parentID, folder.ParentID, "All children should have correct parent ID")
	}
}

// TestGetRootFolders tests retrieving root folders with tenant isolation
func (s *FolderTestSuite) TestGetRootFolders() {
	// Create multiple root folders for test tenant
	rootCount := 5
	for i := 0; i < rootCount; i++ {
		s.createTestFolder(fmt.Sprintf("Root %d", i), "")
	}
	
	// Create root folders for different tenant
	otherTenantID := uuid.New().String()
	otherUserID := uuid.New().String()
	for i := 0; i < 3; i++ {
		folder := models.NewFolder(fmt.Sprintf("Other Tenant Root %d", i), "", otherTenantID, otherUserID)
		_, err := s.folderRepo.Create(s.ctx, folder)
		require.NoError(s.T(), err, "Should create folder for other tenant")
	}
	
	// Retrieve root folders for test tenant
	pagination := utils.NewPagination(1, 20)
	result, err := s.folderRepo.GetRootFolders(s.ctx, s.testTenantID, pagination)
	require.NoError(s.T(), err, "Should retrieve root folders without error")
	assert.Len(s.T(), result.Items, rootCount, "Should return correct number of root folders")
	assert.Equal(s.T(), int64(rootCount), result.Pagination.TotalItems, "Should return correct total count")
	
	// Verify all returned folders have empty parent ID
	for _, folder := range result.Items {
		assert.Equal(s.T(), "", folder.ParentID, "All root folders should have empty parent ID")
	}
}

// TestMoveFolder tests moving a folder to a new parent
func (s *FolderTestSuite) TestMoveFolder() {
	// Create two parent folders
	_, parentID1 := s.createTestFolder("Parent 1", "")
	_, parentID2 := s.createTestFolder("Parent 2", "")
	
	// Create a child folder under first parent
	childFolder, childID := s.createTestFolder("Child Folder", parentID1)
	
	// Move the child folder to second parent
	err := s.folderRepo.Move(s.ctx, childID, parentID2, s.testTenantID)
	require.NoError(s.T(), err, "Should move folder without error")
	
	// Retrieve the moved folder
	movedFolder, err := s.folderRepo.GetByID(s.ctx, childID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve moved folder without error")
	
	// Verify folder has new parent ID
	assert.Equal(s.T(), parentID2, movedFolder.ParentID, "Folder should have new parent ID")
	assert.NotEqual(s.T(), childFolder.Path, movedFolder.Path, "Folder path should be updated")
	
	// Get the new parent folder
	parent2, err := s.folderRepo.GetByID(s.ctx, parentID2, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve parent 2 without error")
	assert.True(s.T(), strings.HasPrefix(movedFolder.Path, parent2.Path), "Moved folder path should start with new parent path")
}

// TestMoveFolderWithChildren tests moving a folder with its children
func (s *FolderTestSuite) TestMoveFolderWithChildren() {
	// Create a folder structure with multiple levels
	_, rootID := s.createTestFolder("Root", "")
	parentFolder, parentID := s.createTestFolder("Parent", rootID)
	_, childID := s.createTestFolder("Child", parentID)
	
	// Create a new parent folder
	_, newParentID := s.createTestFolder("New Parent", "")
	
	// Move a middle-level folder to a new parent
	err := s.folderRepo.Move(s.ctx, parentID, newParentID, s.testTenantID)
	require.NoError(s.T(), err, "Should move folder without error")
	
	// Retrieve the moved folder
	movedParent, err := s.folderRepo.GetByID(s.ctx, parentID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve moved parent without error")
	
	// Verify folder's parent ID is updated
	assert.Equal(s.T(), newParentID, movedParent.ParentID, "Parent should have new parent ID")
	assert.NotEqual(s.T(), parentFolder.Path, movedParent.Path, "Parent path should be updated")
	
	// Retrieve all descendant folders
	movedChild, err := s.folderRepo.GetByID(s.ctx, childID, s.testTenantID)
	require.NoError(s.T(), err, "Should retrieve child folder without error")
	
	// Verify all descendant folders have updated paths
	assert.True(s.T(), strings.HasPrefix(movedChild.Path, movedParent.Path), "Child path should start with new parent path")
}

// TestCircularReference tests prevention of circular references in folder hierarchy
func (s *FolderTestSuite) TestCircularReference() {
	// Create a parent folder
	_, parentID := s.createTestFolder("Parent", "")
	
	// Create a child folder under parent
	_, childID := s.createTestFolder("Child", parentID)
	
	// Create a grandchild folder under child
	_, grandchildID := s.createTestFolder("Grandchild", childID)
	
	// Attempt to move parent folder under grandchild
	err := s.folderRepo.Move(s.ctx, parentID, grandchildID, s.testTenantID)
	assert.Error(s.T(), err, "Should prevent circular reference")
}

// TestFolderExists tests checking if a folder exists with tenant isolation
func (s *FolderTestSuite) TestFolderExists() {
	// Create a test folder
	_, folderID := s.createTestFolder("Exists Test Folder", "")
	
	// Check if folder exists with correct tenant ID
	exists, err := s.folderRepo.Exists(s.ctx, folderID, s.testTenantID)
	require.NoError(s.T(), err, "Should check existence without error")
	assert.True(s.T(), exists, "Folder should exist")
	
	// Check if folder exists with incorrect tenant ID
	wrongTenantID := uuid.New().String()
	exists, err = s.folderRepo.Exists(s.ctx, folderID, wrongTenantID)
	require.NoError(s.T(), err, "Should check existence without error")
	assert.False(s.T(), exists, "Folder should not exist for wrong tenant")
	
	// Check if non-existent folder exists
	exists, err = s.folderRepo.Exists(s.ctx, uuid.New().String(), s.testTenantID)
	require.NoError(s.T(), err, "Should check existence without error")
	assert.False(s.T(), exists, "Non-existent folder should not exist")
}

// TestIsEmpty tests checking if a folder is empty
func (s *FolderTestSuite) TestIsEmpty() {
	// Create a parent folder
	_, parentID := s.createTestFolder("Empty Test Parent", "")
	
	// Check if folder is empty
	isEmpty, err := s.folderRepo.IsEmpty(s.ctx, parentID, s.testTenantID)
	require.NoError(s.T(), err, "Should check if empty without error")
	assert.True(s.T(), isEmpty, "Folder should be empty")
	
	// Create a child folder under parent
	s.createTestFolder("Empty Test Child", parentID)
	
	// Check if parent folder is empty
	isEmpty, err = s.folderRepo.IsEmpty(s.ctx, parentID, s.testTenantID)
	require.NoError(s.T(), err, "Should check if empty without error")
	assert.False(s.T(), isEmpty, "Folder should not be empty")
}

// TestGetFolderPath tests retrieving the full path of a folder by its ID
func (s *FolderTestSuite) TestGetFolderPath() {
	// Create a folder structure
	rootFolder, rootID := s.createTestFolder("PathRoot", "")
	parentFolder, parentID := s.createTestFolder("PathParent", rootID)
	childFolder, childID := s.createTestFolder("PathChild", parentID)
	
	// Get path for root folder
	rootPath, err := s.folderRepo.GetFolderPath(s.ctx, rootID, s.testTenantID)
	require.NoError(s.T(), err, "Should get root path without error")
	assert.Equal(s.T(), rootFolder.Path, rootPath, "Root path should match")
	
	// Get path for parent folder
	parentPath, err := s.folderRepo.GetFolderPath(s.ctx, parentID, s.testTenantID)
	require.NoError(s.T(), err, "Should get parent path without error")
	assert.Equal(s.T(), parentFolder.Path, parentPath, "Parent path should match")
	
	// Get path for child folder
	childPath, err := s.folderRepo.GetFolderPath(s.ctx, childID, s.testTenantID)
	require.NoError(s.T(), err, "Should get child path without error")
	assert.Equal(s.T(), childFolder.Path, childPath, "Child path should match")
	
	// Try with incorrect tenant ID
	wrongTenantID := uuid.New().String()
	_, err = s.folderRepo.GetFolderPath(s.ctx, childID, wrongTenantID)
	assert.Error(s.T(), err, "Should not get path with incorrect tenant ID")
}

// TestSearch tests searching folders by name with tenant isolation
func (s *FolderTestSuite) TestSearch() {
	// Create multiple folders with different names
	s.createTestFolder("SearchTest Alpha", "")
	s.createTestFolder("SearchTest Beta", "")
	s.createTestFolder("SearchTest Gamma", "")
	s.createTestFolder("DifferentName", "")
	
	// Create folders with similar names for different tenant
	otherTenantID := uuid.New().String()
	otherUserID := uuid.New().String()
	searchFolder := models.NewFolder("SearchTest Delta", "", otherTenantID, otherUserID)
	_, err := s.folderRepo.Create(s.ctx, searchFolder)
	require.NoError(s.T(), err, "Should create folder for other tenant")
	
	// Search for folders with specific name pattern
	pagination := utils.NewPagination(1, 10)
	result, err := s.folderRepo.Search(s.ctx, "SearchTest", s.testTenantID, pagination)
	require.NoError(s.T(), err, "Should search without error")
	
	// Verify search results
	assert.Len(s.T(), result.Items, 3, "Should find 3 matching folders")
	assert.Equal(s.T(), int64(3), result.Pagination.TotalItems, "Total count should be 3")
	
	// Verify all results are for correct tenant
	for _, folder := range result.Items {
		assert.Equal(s.T(), s.testTenantID, folder.TenantID, "Result should be for correct tenant")
		assert.Contains(s.T(), folder.Name, "SearchTest", "Result should match search term")
	}
}

// TestFolderSuite runs the test suite
func TestFolderSuite(t *testing.T) {
	suite.Run(t, new(FolderTestSuite))
}