package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid" // v1.3.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+
	"gorm.io/gorm" // v1.25.0+

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/utils"
)

// TestDocumentRepositorySuite runs the document repository test suite
func TestDocumentRepositorySuite(t *testing.T) {
	suite.Run(t, new(DocumentRepositorySuite))
}

// DocumentRepositorySuite is a test suite for testing the PostgreSQL implementation of DocumentRepository
type DocumentRepositorySuite struct {
	suite.Suite
	db           *gorm.DB
	repo         repositories.DocumentRepository
	testTenantID string
	testFolderID string
	testOwnerID  string
}

// SetupSuite initializes the test suite by setting up the database connection
func (s *DocumentRepositorySuite) SetupSuite() {
	// Initialize test database connection
	var err error
	s.db, err = setupTestDatabase()
	require.NoError(s.T(), err, "Failed to set up test database")

	// Set up test constants
	s.testTenantID = uuid.New().String()
	s.testFolderID = uuid.New().String()
	s.testOwnerID = uuid.New().String()
}

// TearDownSuite cleans up resources after all tests have run
func (s *DocumentRepositorySuite) TearDownSuite() {
	// Close database connection if needed
	sqlDB, err := s.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

// SetupTest prepares the test environment before each test
func (s *DocumentRepositorySuite) SetupTest() {
	// Clean up test data from previous tests
	err := cleanTestData(s.db)
	require.NoError(s.T(), err, "Failed to clean test data")

	// Initialize a new document repository
	s.repo = NewDocumentRepository(s.db)
}

// TearDownTest cleans up after each test
func (s *DocumentRepositorySuite) TearDownTest() {
	// Clean up test data created during the test
}

// createTestDocument is a helper function to create a test document
func (s *DocumentRepositorySuite) createTestDocument(name, contentType string, size int64) *models.Document {
	doc := models.NewDocument(
		name,
		contentType,
		size,
		s.testFolderID,
		s.testTenantID,
		s.testOwnerID,
	)
	return &doc
}

// TestCreate tests the Create method of the document repository
func (s *DocumentRepositorySuite) TestCreate() {
	// Create a test document
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)

	// Test document creation
	docID, err := s.repo.Create(context.Background(), doc)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), docID)

	// Verify document was created with correct data
	createdDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), doc.Name, createdDoc.Name)
	assert.Equal(s.T(), doc.ContentType, createdDoc.ContentType)
	assert.Equal(s.T(), doc.Size, createdDoc.Size)
	assert.Equal(s.T(), doc.FolderID, createdDoc.FolderID)
	assert.Equal(s.T(), doc.TenantID, createdDoc.TenantID)
	assert.Equal(s.T(), doc.OwnerID, createdDoc.OwnerID)
	assert.Equal(s.T(), models.DocumentStatusProcessing, createdDoc.Status)
}

// TestGetByID tests the GetByID method of the document repository
func (s *DocumentRepositorySuite) TestGetByID() {
	// Create and persist a test document
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), docID)

	// Test retrieving the document by ID
	retrievedDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), retrievedDoc)
	assert.Equal(s.T(), docID, retrievedDoc.ID)
	assert.Equal(s.T(), doc.Name, retrievedDoc.Name)
	assert.Equal(s.T(), s.testTenantID, retrievedDoc.TenantID)

	// Test with non-existent ID
	nonExistentDoc, err := s.repo.GetByID(context.Background(), "non-existent-id", s.testTenantID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), nonExistentDoc)

	// Test tenant isolation: Try to retrieve document with wrong tenant ID
	wrongTenantDoc, err := s.repo.GetByID(context.Background(), docID, "wrong-tenant-id")
	assert.Error(s.T(), err)
	assert.Nil(s.T(), wrongTenantDoc)
}

// TestUpdate tests the Update method of the document repository
func (s *DocumentRepositorySuite) TestUpdate() {
	// Create and persist a test document
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), docID)

	// Retrieve created document to get its ID
	createdDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	require.NoError(s.T(), err)

	// Modify document
	createdDoc.Name = "updated.pdf"
	createdDoc.Status = models.DocumentStatusAvailable

	// Test updating the document
	err = s.repo.Update(context.Background(), createdDoc)
	assert.NoError(s.T(), err)

	// Verify document was updated
	updatedDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "updated.pdf", updatedDoc.Name)
	assert.Equal(s.T(), models.DocumentStatusAvailable, updatedDoc.Status)

	// Test tenant isolation: Try to update document with wrong tenant ID
	createdDoc.TenantID = "wrong-tenant-id"
	err = s.repo.Update(context.Background(), createdDoc)
	assert.Error(s.T(), err)
}

// TestDelete tests the Delete method of the document repository
func (s *DocumentRepositorySuite) TestDelete() {
	// Create and persist a test document
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), docID)

	// Test deleting the document
	err = s.repo.Delete(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)

	// Verify document was deleted
	deletedDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), deletedDoc)

	// Test tenant isolation: Try to delete document with wrong tenant ID
	docID, err = s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)
	err = s.repo.Delete(context.Background(), docID, "wrong-tenant-id")
	assert.Error(s.T(), err)

	// Verify document still exists
	existingDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), existingDoc)
}

// TestListByFolder tests the ListByFolder method of the document repository
func (s *DocumentRepositorySuite) TestListByFolder() {
	// Create multiple test documents in the same folder
	for i := 0; i < 5; i++ {
		doc := s.createTestDocument(
			fmt.Sprintf("test%d.pdf", i),
			"application/pdf",
			1024,
		)
		_, err := s.repo.Create(context.Background(), doc)
		require.NoError(s.T(), err)
	}

	// Test listing documents by folder
	pagination := utils.NewPagination(1, 10)
	result, err := s.repo.ListByFolder(context.Background(), s.testFolderID, s.testTenantID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result.Items, 5)
	assert.Equal(s.T(), int64(5), result.Pagination.TotalItems)

	// Test tenant isolation: Try to list documents with wrong tenant ID
	wrongTenantResult, err := s.repo.ListByFolder(context.Background(), s.testFolderID, "wrong-tenant-id", pagination)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), wrongTenantResult.Items)

	// Test pagination
	// Create more documents to test pagination
	for i := 5; i < 15; i++ {
		doc := s.createTestDocument(
			fmt.Sprintf("test%d.pdf", i),
			"application/pdf",
			1024,
		)
		_, err := s.repo.Create(context.Background(), doc)
		require.NoError(s.T(), err)
	}

	// Test first page
	pagination = utils.NewPagination(1, 5)
	firstPageResult, err := s.repo.ListByFolder(context.Background(), s.testFolderID, s.testTenantID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), firstPageResult.Items, 5)
	assert.Equal(s.T(), int64(15), firstPageResult.Pagination.TotalItems)
	assert.True(s.T(), firstPageResult.Pagination.HasNext)
	assert.False(s.T(), firstPageResult.Pagination.HasPrevious)
}

// TestListByTenant tests the ListByTenant method of the document repository
func (s *DocumentRepositorySuite) TestListByTenant() {
	// Create multiple test documents for the test tenant
	for i := 0; i < 5; i++ {
		doc := s.createTestDocument(
			fmt.Sprintf("test%d.pdf", i),
			"application/pdf",
			1024,
		)
		_, err := s.repo.Create(context.Background(), doc)
		require.NoError(s.T(), err)
	}

	// Create documents for a different tenant
	otherTenantID := uuid.New().String()
	for i := 0; i < 3; i++ {
		doc := models.NewDocument(
			fmt.Sprintf("other%d.pdf", i),
			"application/pdf",
			1024,
			s.testFolderID,
			otherTenantID,
			s.testOwnerID,
		)
		_, err := s.repo.Create(context.Background(), &doc)
		require.NoError(s.T(), err)
	}

	// Test listing documents by tenant
	pagination := utils.NewPagination(1, 10)
	result, err := s.repo.ListByTenant(context.Background(), s.testTenantID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), result.Items, 5)
	assert.Equal(s.T(), int64(5), result.Pagination.TotalItems)

	// Test listing documents for other tenant
	otherResult, err := s.repo.ListByTenant(context.Background(), otherTenantID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), otherResult.Items, 3)
	assert.Equal(s.T(), int64(3), otherResult.Pagination.TotalItems)
}

// TestSearchByMetadata tests the SearchByMetadata method of the document repository
func (s *DocumentRepositorySuite) TestSearchByMetadata() {
	// Create test documents with different metadata
	for i := 0; i < 5; i++ {
		doc := s.createTestDocument(
			fmt.Sprintf("test%d.pdf", i),
			"application/pdf",
			1024,
		)
		docID, err := s.repo.Create(context.Background(), doc)
		require.NoError(s.T(), err)

		// Add metadata
		_, err = s.repo.AddMetadata(context.Background(), docID, "category", "finance", s.testTenantID)
		require.NoError(s.T(), err)
		
		if i < 3 {
			// First 3 documents also have department=accounting
			_, err = s.repo.AddMetadata(context.Background(), docID, "department", "accounting", s.testTenantID)
			require.NoError(s.T(), err)
		} else {
			// Last 2 documents have department=legal
			_, err = s.repo.AddMetadata(context.Background(), docID, "department", "legal", s.testTenantID)
			require.NoError(s.T(), err)
		}
	}

	// Test searching by single metadata field
	pagination := utils.NewPagination(1, 10)
	categoryMetadata := map[string]string{"category": "finance"}
	categoryResult, err := s.repo.SearchByMetadata(context.Background(), categoryMetadata, s.testTenantID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), categoryResult.Items, 5)

	// Test searching by multiple metadata fields
	accountingMetadata := map[string]string{
		"category":   "finance",
		"department": "accounting",
	}
	accountingResult, err := s.repo.SearchByMetadata(context.Background(), accountingMetadata, s.testTenantID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), accountingResult.Items, 3)

	// Test tenant isolation: No results for wrong tenant
	wrongTenantResult, err := s.repo.SearchByMetadata(context.Background(), categoryMetadata, "wrong-tenant-id", pagination)
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), wrongTenantResult.Items)
}

// TestAddVersion tests the AddVersion method of the document repository
func (s *DocumentRepositorySuite) TestAddVersion() {
	// Create and persist a test document
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)

	// Create a new document version
	version := models.NewDocumentVersion(
		docID,
		1,
		2048,
		"abcdef123456",
		"test/path/file.pdf",
		s.testOwnerID,
	)

	// Test adding version
	versionID, err := s.repo.AddVersion(context.Background(), &version)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), versionID)

	// Verify version was added
	retrievedVersion, err := s.repo.GetVersionByID(context.Background(), versionID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), docID, retrievedVersion.DocumentID)
	assert.Equal(s.T(), 1, retrievedVersion.VersionNumber)
	assert.Equal(s.T(), int64(2048), retrievedVersion.Size)

	// Test adding version to non-existent document
	invalidVersion := models.NewDocumentVersion(
		"non-existent-doc",
		1,
		1024,
		"abcdef123456",
		"test/path/file.pdf",
		s.testOwnerID,
	)
	_, err = s.repo.AddVersion(context.Background(), &invalidVersion)
	assert.Error(s.T(), err)

	// Test tenant isolation when retrieving version
	_, err = s.repo.GetVersionByID(context.Background(), versionID, "wrong-tenant-id")
	assert.Error(s.T(), err)
}

// TestGetVersionByID tests the GetVersionByID method of the document repository
func (s *DocumentRepositorySuite) TestGetVersionByID() {
	// Create a test document with a version
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)

	version := models.NewDocumentVersion(
		docID,
		1,
		2048,
		"abcdef123456",
		"test/path/file.pdf",
		s.testOwnerID,
	)
	versionID, err := s.repo.AddVersion(context.Background(), &version)
	require.NoError(s.T(), err)

	// Test retrieving the version
	retrievedVersion, err := s.repo.GetVersionByID(context.Background(), versionID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), retrievedVersion)
	assert.Equal(s.T(), versionID, retrievedVersion.ID)
	assert.Equal(s.T(), docID, retrievedVersion.DocumentID)

	// Test with non-existent version ID
	nonExistentVersion, err := s.repo.GetVersionByID(context.Background(), "non-existent-id", s.testTenantID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), nonExistentVersion)

	// Test tenant isolation
	_, err = s.repo.GetVersionByID(context.Background(), versionID, "wrong-tenant-id")
	assert.Error(s.T(), err)
}

// TestUpdateVersionStatus tests the UpdateVersionStatus method of the document repository
func (s *DocumentRepositorySuite) TestUpdateVersionStatus() {
	// Create a test document with a version
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)

	version := models.NewDocumentVersion(
		docID,
		1,
		2048,
		"abcdef123456",
		"test/path/file.pdf",
		s.testOwnerID,
	)
	versionID, err := s.repo.AddVersion(context.Background(), &version)
	require.NoError(s.T(), err)

	// Test updating version status
	err = s.repo.UpdateVersionStatus(context.Background(), versionID, models.VersionStatusAvailable, s.testTenantID)
	assert.NoError(s.T(), err)

	// Verify status was updated
	updatedVersion, err := s.repo.GetVersionByID(context.Background(), versionID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), models.VersionStatusAvailable, updatedVersion.Status)

	// Test with non-existent version ID
	err = s.repo.UpdateVersionStatus(context.Background(), "non-existent-id", models.VersionStatusAvailable, s.testTenantID)
	assert.Error(s.T(), err)

	// Test tenant isolation
	err = s.repo.UpdateVersionStatus(context.Background(), versionID, models.VersionStatusQuarantined, "wrong-tenant-id")
	assert.Error(s.T(), err)

	// Verify status wasn't changed
	unchangedVersion, err := s.repo.GetVersionByID(context.Background(), versionID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), models.VersionStatusAvailable, unchangedVersion.Status)
}

// TestAddMetadata tests the AddMetadata method of the document repository
func (s *DocumentRepositorySuite) TestAddMetadata() {
	// Create and persist a test document
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)

	// Test adding metadata
	metadataID, err := s.repo.AddMetadata(context.Background(), docID, "author", "Jane Doe", s.testTenantID)
	assert.NoError(s.T(), err)
	assert.NotEmpty(s.T(), metadataID)

	// Retrieve document and verify metadata was added
	retrievedDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), retrievedDoc.Metadata, 1)
	assert.Equal(s.T(), "author", retrievedDoc.Metadata[0].Key)
	assert.Equal(s.T(), "Jane Doe", retrievedDoc.Metadata[0].Value)

	// Test adding metadata to non-existent document
	_, err = s.repo.AddMetadata(context.Background(), "non-existent-id", "author", "Jane Doe", s.testTenantID)
	assert.Error(s.T(), err)

	// Test tenant isolation
	_, err = s.repo.AddMetadata(context.Background(), docID, "department", "Finance", "wrong-tenant-id")
	assert.Error(s.T(), err)
}

// TestUpdateMetadata tests the UpdateMetadata method of the document repository
func (s *DocumentRepositorySuite) TestUpdateMetadata() {
	// Create a test document with metadata
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)

	_, err = s.repo.AddMetadata(context.Background(), docID, "author", "Jane Doe", s.testTenantID)
	require.NoError(s.T(), err)

	// Test updating metadata
	err = s.repo.UpdateMetadata(context.Background(), docID, "author", "John Smith", s.testTenantID)
	assert.NoError(s.T(), err)

	// Verify metadata was updated
	updatedDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), updatedDoc.Metadata, 1)
	assert.Equal(s.T(), "author", updatedDoc.Metadata[0].Key)
	assert.Equal(s.T(), "John Smith", updatedDoc.Metadata[0].Value)

	// Test updating non-existent metadata
	err = s.repo.UpdateMetadata(context.Background(), docID, "non-existent", "value", s.testTenantID)
	assert.Error(s.T(), err)

	// Test tenant isolation
	err = s.repo.UpdateMetadata(context.Background(), docID, "author", "Impostor", "wrong-tenant-id")
	assert.Error(s.T(), err)
}

// TestDeleteMetadata tests the DeleteMetadata method of the document repository
func (s *DocumentRepositorySuite) TestDeleteMetadata() {
	// Create a test document with metadata
	doc := s.createTestDocument("test.pdf", "application/pdf", 1024)
	docID, err := s.repo.Create(context.Background(), doc)
	require.NoError(s.T(), err)

	_, err = s.repo.AddMetadata(context.Background(), docID, "author", "Jane Doe", s.testTenantID)
	require.NoError(s.T(), err)
	_, err = s.repo.AddMetadata(context.Background(), docID, "department", "HR", s.testTenantID)
	require.NoError(s.T(), err)

	// Test deleting metadata
	err = s.repo.DeleteMetadata(context.Background(), docID, "author", s.testTenantID)
	assert.NoError(s.T(), err)

	// Verify metadata was deleted
	updatedDoc, err := s.repo.GetByID(context.Background(), docID, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), updatedDoc.Metadata, 1)
	assert.Equal(s.T(), "department", updatedDoc.Metadata[0].Key)

	// Test deleting non-existent metadata
	err = s.repo.DeleteMetadata(context.Background(), docID, "non-existent", s.testTenantID)
	assert.Error(s.T(), err)

	// Test tenant isolation
	err = s.repo.DeleteMetadata(context.Background(), docID, "department", "wrong-tenant-id")
	assert.Error(s.T(), err)
}

// TestGetDocumentsByIDs tests the GetDocumentsByIDs method of the document repository
func (s *DocumentRepositorySuite) TestGetDocumentsByIDs() {
	// Create multiple test documents
	docIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		doc := s.createTestDocument(
			fmt.Sprintf("test%d.pdf", i),
			"application/pdf",
			1024,
		)
		docID, err := s.repo.Create(context.Background(), doc)
		require.NoError(s.T(), err)
		docIDs[i] = docID
	}

	// Test retrieving documents by IDs
	docs, err := s.repo.GetDocumentsByIDs(context.Background(), docIDs, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), docs, 5)

	// Test with some non-existent IDs
	mixedIDs := append(docIDs[:2], "non-existent-1", "non-existent-2")
	mixedDocs, err := s.repo.GetDocumentsByIDs(context.Background(), mixedIDs, s.testTenantID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), mixedDocs, 2)

	// Test tenant isolation
	wrongTenantDocs, err := s.repo.GetDocumentsByIDs(context.Background(), docIDs, "wrong-tenant-id")
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), wrongTenantDocs)
}

// TestTenantIsolation performs comprehensive tests for tenant isolation across all repository methods
func (s *DocumentRepositorySuite) TestTenantIsolation() {
	// Create documents for multiple tenants
	tenant1ID := uuid.New().String()
	tenant2ID := uuid.New().String()

	// Create documents for tenant 1
	tenant1Docs := make([]string, 3)
	for i := 0; i < 3; i++ {
		doc := models.NewDocument(
			fmt.Sprintf("tenant1-doc%d.pdf", i),
			"application/pdf",
			1024,
			s.testFolderID,
			tenant1ID,
			s.testOwnerID,
		)
		docID, err := s.repo.Create(context.Background(), &doc)
		require.NoError(s.T(), err)
		tenant1Docs[i] = docID
	}

	// Create documents for tenant 2
	tenant2Docs := make([]string, 3)
	for i := 0; i < 3; i++ {
		doc := models.NewDocument(
			fmt.Sprintf("tenant2-doc%d.pdf", i),
			"application/pdf",
			1024,
			s.testFolderID,
			tenant2ID,
			s.testOwnerID,
		)
		docID, err := s.repo.Create(context.Background(), &doc)
		require.NoError(s.T(), err)
		tenant2Docs[i] = docID
	}

	// Test GetByID with correct and incorrect tenant IDs
	doc1, err := s.repo.GetByID(context.Background(), tenant1Docs[0], tenant1ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), doc1)
	assert.Equal(s.T(), tenant1ID, doc1.TenantID)

	doc1Wrong, err := s.repo.GetByID(context.Background(), tenant1Docs[0], tenant2ID)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), doc1Wrong)

	// Test ListByTenant for each tenant
	pagination := utils.NewPagination(1, 10)
	tenant1List, err := s.repo.ListByTenant(context.Background(), tenant1ID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), tenant1List.Items, 3)
	for _, doc := range tenant1List.Items {
		assert.Equal(s.T(), tenant1ID, doc.TenantID)
		assert.Contains(s.T(), doc.Name, "tenant1-doc")
	}

	tenant2List, err := s.repo.ListByTenant(context.Background(), tenant2ID, pagination)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), tenant2List.Items, 3)
	for _, doc := range tenant2List.Items {
		assert.Equal(s.T(), tenant2ID, doc.TenantID)
		assert.Contains(s.T(), doc.Name, "tenant2-doc")
	}

	// Test Update with correct and incorrect tenant IDs
	doc1.Name = "tenant1-updated.pdf"
	err = s.repo.Update(context.Background(), doc1)
	assert.NoError(s.T(), err)

	doc1Copy := *doc1
	doc1Copy.TenantID = tenant2ID // Try to update with wrong tenant ID
	err = s.repo.Update(context.Background(), &doc1Copy)
	assert.Error(s.T(), err)

	// Test Delete with correct and incorrect tenant IDs
	err = s.repo.Delete(context.Background(), tenant1Docs[1], tenant1ID)
	assert.NoError(s.T(), err)

	err = s.repo.Delete(context.Background(), tenant1Docs[2], tenant2ID)
	assert.Error(s.T(), err)

	// Verify that document still exists
	doc2, err := s.repo.GetByID(context.Background(), tenant1Docs[2], tenant1ID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), doc2)

	// Test metadata operations with tenant isolation
	_, err = s.repo.AddMetadata(context.Background(), tenant2Docs[0], "key", "value", tenant2ID)
	assert.NoError(s.T(), err)

	err = s.repo.UpdateMetadata(context.Background(), tenant2Docs[0], "key", "new-value", tenant2ID)
	assert.NoError(s.T(), err)

	err = s.repo.UpdateMetadata(context.Background(), tenant2Docs[0], "key", "impostor-value", tenant1ID)
	assert.Error(s.T(), err)

	// Verify all operations maintained tenant isolation
	allTenant1Docs, err := s.repo.GetDocumentsByIDs(context.Background(), append(tenant1Docs, tenant2Docs...), tenant1ID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), allTenant1Docs, 2) // One was deleted, so only 2 remain

	allTenant2Docs, err := s.repo.GetDocumentsByIDs(context.Background(), append(tenant1Docs, tenant2Docs...), tenant2ID)
	assert.NoError(s.T(), err)
	assert.Len(s.T(), allTenant2Docs, 3)
}

// setupTestDatabase creates and initializes a test database for running repository tests
func setupTestDatabase() (*gorm.DB, error) {
	// In a real implementation, this would connect to a test database or use an in-memory database
	// For this test file, we'll assume there's an implementation that sets up the test database
	
	// This is a placeholder - actual implementation depends on project's test infrastructure
	var db *gorm.DB
	// db, err := setupTestPostgresDB() or setupInMemoryDB()
	
	// For the purpose of this test file, we'll return nil to simulate a successful connection
	// In a real implementation, this would return an actual database connection
	return db, nil
}

// cleanTestData removes all test data from the database between tests
func cleanTestData(db *gorm.DB) error {
	// In a real implementation, this would delete test data from the database
	// For this test file, we'll assume it works without implementing the details
	return nil
}