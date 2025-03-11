package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid" // v1.3.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+

	"../../domain/models"
	"../../domain/repositories"
	"../../infrastructure/persistence/postgres"
	"../../pkg/config"
	"../../pkg/errors"
	"../../pkg/utils"
)

// Define test constants for tenant isolation testing
const (
	testTenantID1 = "tenant-test-1"
	testTenantID2 = "tenant-test-2"
	testUserID    = "user-test-1"
	testFolderID  = "folder-test-1"
)

// TestDocumentRepositorySuite runs the document repository test suite
func TestDocumentRepositorySuite(t *testing.T) {
	suite.Run(t, new(DocumentRepositorySuite))
}

// DocumentRepositorySuite is the test suite for document repository integration tests
type DocumentRepositorySuite struct {
	suite.Suite
	repo repositories.DocumentRepository
	ctx  context.Context
}

// SetupSuite sets up the test suite by initializing the database connection
func (s *DocumentRepositorySuite) SetupSuite() {
	// Create test database configuration
	dbConfig := config.DatabaseConfig{
		Host:            os.Getenv("TEST_DB_HOST"),
		Port:            5432, // Default PostgreSQL port
		User:            os.Getenv("TEST_DB_USER"),
		Password:        os.Getenv("TEST_DB_PASSWORD"),
		DBName:          os.Getenv("TEST_DB_NAME"),
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: "1h",
	}

	// Use default values if environment variables are not set
	if dbConfig.Host == "" {
		dbConfig.Host = "localhost"
	}
	if dbConfig.User == "" {
		dbConfig.User = "postgres"
	}
	if dbConfig.Password == "" {
		dbConfig.Password = "postgres"
	}
	if dbConfig.DBName == "" {
		dbConfig.DBName = "document_mgmt_test"
	}

	// Initialize database connection
	err := postgres.Init(dbConfig)
	s.Require().NoError(err, "Failed to initialize database connection")

	// Get database instance
	db, err := postgres.GetDB()
	s.Require().NoError(err, "Failed to get database instance")

	// Run migrations for required models
	err = postgres.Migrate(&models.Document{}, &models.DocumentMetadata{}, &models.DocumentVersion{}, &models.Tag{})
	s.Require().NoError(err, "Failed to run migrations")

	// Create document repository
	s.repo = postgres.NewDocumentRepository(db)
	s.ctx = context.Background()
}

// TearDownSuite tears down the test suite by closing the database connection
func (s *DocumentRepositorySuite) TearDownSuite() {
	// Close database connection
	err := postgres.Close()
	s.NoError(err, "Failed to close database connection")
}

// SetupTest sets up each test by cleaning the database
func (s *DocumentRepositorySuite) SetupTest() {
	// Clean up test data from previous tests
	s.cleanupTestData()
}

// TestCreateDocument tests document creation functionality
func (s *DocumentRepositorySuite) TestCreateDocument() {
	// Create a test document
	doc := models.NewDocument("test-doc.pdf", "application/pdf", 1024, testFolderID, testTenantID1, testUserID)
	
	// Create the document
	docID, err := s.repo.Create(s.ctx, &doc)
	s.NoError(err, "Document creation should succeed")
	s.NotEmpty(docID, "Document ID should not be empty")
	
	// Retrieve the document to verify it was created correctly
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.NotNil(retrievedDoc, "Retrieved document should not be nil")
	s.Equal("test-doc.pdf", retrievedDoc.Name, "Document name should match")
	s.Equal("application/pdf", retrievedDoc.ContentType, "Document content type should match")
	s.Equal(int64(1024), retrievedDoc.Size, "Document size should match")
	s.Equal(testFolderID, retrievedDoc.FolderID, "Document folder ID should match")
	s.Equal(testTenantID1, retrievedDoc.TenantID, "Document tenant ID should match")
	s.Equal(testUserID, retrievedDoc.OwnerID, "Document owner ID should match")
	s.Equal(models.DocumentStatusProcessing, retrievedDoc.Status, "Document status should be processing")
}

// TestCreateDocumentWithMetadata tests document creation with metadata
func (s *DocumentRepositorySuite) TestCreateDocumentWithMetadata() {
	// Create a test document with metadata
	doc := models.NewDocument("test-doc-with-metadata.pdf", "application/pdf", 2048, testFolderID, testTenantID1, testUserID)
	doc.AddMetadata("author", "Test Author")
	doc.AddMetadata("department", "Test Department")
	
	// Create the document
	docID, err := s.repo.Create(s.ctx, &doc)
	s.NoError(err, "Document creation with metadata should succeed")
	
	// Retrieve the document to verify it was created correctly with metadata
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.NotNil(retrievedDoc, "Retrieved document should not be nil")
	
	// Verify metadata
	s.Len(retrievedDoc.Metadata, 2, "Document should have 2 metadata entries")
	s.Equal("Test Author", retrievedDoc.GetMetadata("author"), "Author metadata should match")
	s.Equal("Test Department", retrievedDoc.GetMetadata("department"), "Department metadata should match")
}

// TestGetDocumentByID tests document retrieval by ID with tenant isolation
func (s *DocumentRepositorySuite) TestGetDocumentByID() {
	// Create a test document for tenant 1
	doc, docID := s.createTestDocument("tenant1-doc.pdf", "application/pdf", 1024, testTenantID1)
	
	// Retrieve the document with the correct tenant ID
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval with correct tenant ID should succeed")
	s.NotNil(retrievedDoc, "Retrieved document should not be nil")
	s.Equal(docID, retrievedDoc.ID, "Document ID should match")
	
	// Attempt to retrieve the document with a different tenant ID
	retrievedDoc, err = s.repo.GetByID(s.ctx, docID, testTenantID2)
	s.Error(err, "Document retrieval with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err), "Error should be ResourceNotFoundError")
	s.Nil(retrievedDoc, "Retrieved document should be nil")
}

// TestUpdateDocument tests document update functionality with tenant isolation
func (s *DocumentRepositorySuite) TestUpdateDocument() {
	// Create a test document
	doc, docID := s.createTestDocument("original-name.pdf", "application/pdf", 1024, testTenantID1)
	
	// Update the document properties
	doc.Name = "updated-name.pdf"
	doc.ContentType = "application/octet-stream"
	err := s.repo.Update(s.ctx, doc)
	s.NoError(err, "Document update should succeed")
	
	// Retrieve the updated document
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.Equal("updated-name.pdf", retrievedDoc.Name, "Document name should be updated")
	s.Equal("application/octet-stream", retrievedDoc.ContentType, "Document content type should be updated")
	
	// Attempt to update a document with a different tenant ID
	originalTenantID := doc.TenantID
	doc.TenantID = testTenantID2
	err = s.repo.Update(s.ctx, doc)
	s.Error(err, "Document update with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err), "Error should be ResourceNotFoundError")
	
	// Restore the original tenant ID for cleanup
	doc.TenantID = originalTenantID
}

// TestDeleteDocument tests document deletion functionality with tenant isolation
func (s *DocumentRepositorySuite) TestDeleteDocument() {
	// Create a test document
	_, docID := s.createTestDocument("to-be-deleted.pdf", "application/pdf", 1024, testTenantID1)
	
	// Delete the document
	err := s.repo.Delete(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document deletion should succeed")
	
	// Verify the document is deleted
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.Error(err, "Document retrieval after deletion should fail")
	s.True(errors.IsResourceNotFoundError(err), "Error should be ResourceNotFoundError")
	s.Nil(retrievedDoc, "Retrieved document should be nil")
	
	// Create another test document
	_, docID = s.createTestDocument("another-doc.pdf", "application/pdf", 1024, testTenantID1)
	
	// Attempt to delete with a different tenant ID
	err = s.repo.Delete(s.ctx, docID, testTenantID2)
	s.Error(err, "Document deletion with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err), "Error should be ResourceNotFoundError")
	
	// Verify the document still exists
	retrievedDoc, err = s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document should still exist")
	s.NotNil(retrievedDoc, "Retrieved document should not be nil")
}

// TestListDocumentsByFolder tests listing documents by folder with pagination and tenant isolation
func (s *DocumentRepositorySuite) TestListDocumentsByFolder() {
	// Create multiple test documents in the same folder for tenant 1
	s.createTestDocument("tenant1-doc1.pdf", "application/pdf", 1024, testTenantID1)
	s.createTestDocument("tenant1-doc2.pdf", "application/pdf", 2048, testTenantID1)
	s.createTestDocument("tenant1-doc3.pdf", "application/pdf", 3072, testTenantID1)
	
	// Create test documents in the same folder for tenant 2
	s.createTestDocument("tenant2-doc1.pdf", "application/pdf", 1024, testTenantID2)
	s.createTestDocument("tenant2-doc2.pdf", "application/pdf", 2048, testTenantID2)
	
	// List documents for tenant 1 with pagination
	pagination := utils.NewPagination(1, 2)
	result, err := s.repo.ListByFolder(s.ctx, testFolderID, testTenantID1, pagination)
	
	s.NoError(err, "Document listing should succeed")
	s.Len(result.Items, 2, "Should return 2 documents (page size)")
	s.Equal(int64(3), result.Pagination.TotalItems, "Total items should be 3")
	s.Equal(2, result.Pagination.TotalPages, "Total pages should be 2")
	s.True(result.Pagination.HasNext, "Should have next page")
	s.False(result.Pagination.HasPrevious, "Should not have previous page")
	
	// Check tenant isolation
	for _, doc := range result.Items {
		s.Equal(testTenantID1, doc.TenantID, "Document should belong to tenant 1")
	}
	
	// Test with page 2
	pagination = utils.NewPagination(2, 2)
	result, err = s.repo.ListByFolder(s.ctx, testFolderID, testTenantID1, pagination)
	
	s.NoError(err, "Document listing should succeed")
	s.Len(result.Items, 1, "Should return 1 document (remaining)")
	s.Equal(int64(3), result.Pagination.TotalItems, "Total items should be 3")
	s.Equal(2, result.Pagination.TotalPages, "Total pages should be 2")
	s.False(result.Pagination.HasNext, "Should not have next page")
	s.True(result.Pagination.HasPrevious, "Should have previous page")
}

// TestListDocumentsByTenant tests listing all documents for a tenant with pagination
func (s *DocumentRepositorySuite) TestListDocumentsByTenant() {
	// Create multiple test documents for tenant 1 in different folders
	s.createTestDocument("tenant1-doc1.pdf", "application/pdf", 1024, testTenantID1)
	s.createTestDocument("tenant1-doc2.pdf", "application/pdf", 2048, testTenantID1)
	s.createTestDocument("tenant1-doc3.pdf", "application/pdf", 3072, testTenantID1)
	s.createTestDocument("tenant1-doc4.pdf", "application/pdf", 4096, testTenantID1)
	
	// Create test documents for tenant 2
	s.createTestDocument("tenant2-doc1.pdf", "application/pdf", 1024, testTenantID2)
	s.createTestDocument("tenant2-doc2.pdf", "application/pdf", 2048, testTenantID2)
	
	// List documents for tenant 1 with pagination
	pagination := utils.NewPagination(1, 3)
	result, err := s.repo.ListByTenant(s.ctx, testTenantID1, pagination)
	
	s.NoError(err, "Document listing should succeed")
	s.Len(result.Items, 3, "Should return 3 documents (page size)")
	s.Equal(int64(4), result.Pagination.TotalItems, "Total items should be 4")
	s.Equal(2, result.Pagination.TotalPages, "Total pages should be 2")
	s.True(result.Pagination.HasNext, "Should have next page")
	s.False(result.Pagination.HasPrevious, "Should not have previous page")
	
	// Check tenant isolation
	for _, doc := range result.Items {
		s.Equal(testTenantID1, doc.TenantID, "Document should belong to tenant 1")
	}
	
	// Test with page 2
	pagination = utils.NewPagination(2, 3)
	result, err = s.repo.ListByTenant(s.ctx, testTenantID1, pagination)
	
	s.NoError(err, "Document listing should succeed")
	s.Len(result.Items, 1, "Should return 1 document (remaining)")
	s.Equal(int64(4), result.Pagination.TotalItems, "Total items should be 4")
	s.Equal(2, result.Pagination.TotalPages, "Total pages should be 2")
	s.False(result.Pagination.HasNext, "Should not have next page")
	s.True(result.Pagination.HasPrevious, "Should have previous page")
}

// TestSearchDocumentsByMetadata tests searching documents by metadata with tenant isolation
func (s *DocumentRepositorySuite) TestSearchDocumentsByMetadata() {
	// Create test documents with specific metadata for tenant 1
	doc1, _ := s.createTestDocument("tenant1-doc1.pdf", "application/pdf", 1024, testTenantID1)
	err := s.repo.AddMetadata(s.ctx, doc1.ID, "department", "HR", testTenantID1)
	s.NoError(err, "Adding metadata should succeed")
	
	doc2, _ := s.createTestDocument("tenant1-doc2.pdf", "application/pdf", 2048, testTenantID1)
	err = s.repo.AddMetadata(s.ctx, doc2.ID, "department", "IT", testTenantID1)
	s.NoError(err, "Adding metadata should succeed")
	
	doc3, _ := s.createTestDocument("tenant1-doc3.pdf", "application/pdf", 3072, testTenantID1)
	err = s.repo.AddMetadata(s.ctx, doc3.ID, "department", "HR", testTenantID1)
	s.NoError(err, "Adding metadata should succeed")
	
	// Create test documents with the same metadata for tenant 2
	doc4, _ := s.createTestDocument("tenant2-doc1.pdf", "application/pdf", 1024, testTenantID2)
	err = s.repo.AddMetadata(s.ctx, doc4.ID, "department", "HR", testTenantID2)
	s.NoError(err, "Adding metadata should succeed")
	
	// Search for documents with department=HR for tenant 1
	pagination := utils.NewPagination(1, 10)
	searchCriteria := map[string]string{"department": "HR"}
	result, err := s.repo.SearchByMetadata(s.ctx, searchCriteria, testTenantID1, pagination)
	
	s.NoError(err, "Document search should succeed")
	s.Len(result.Items, 2, "Should return 2 documents")
	s.Equal(int64(2), result.Pagination.TotalItems, "Total items should be 2")
	
	// Check tenant isolation and metadata match
	for _, doc := range result.Items {
		s.Equal(testTenantID1, doc.TenantID, "Document should belong to tenant 1")
		
		// Find the department metadata for each document
		var foundDept bool
		for _, metadata := range doc.Metadata {
			if metadata.Key == "department" && metadata.Value == "HR" {
				foundDept = true
				break
			}
		}
		s.True(foundDept, "Document should have department=HR metadata")
	}
}

// TestAddDocumentVersion tests adding a new version to an existing document with tenant isolation
func (s *DocumentRepositorySuite) TestAddDocumentVersion() {
	// Create a test document
	doc, docID := s.createTestDocument("version-test.pdf", "application/pdf", 1024, testTenantID1)
	
	// Create a new document version
	version := models.NewDocumentVersion(docID, 1, 2048, "abc123", "s3://test-bucket/document.pdf", testUserID)
	
	// Add the version to the document
	versionID, err := s.repo.AddVersion(s.ctx, &version)
	s.NoError(err, "Adding version should succeed")
	s.NotEmpty(versionID, "Version ID should not be empty")
	
	// Retrieve the document to verify the version was added
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.Len(retrievedDoc.Versions, 1, "Document should have 1 version")
	s.Equal(1, retrievedDoc.Versions[0].VersionNumber, "Version number should be 1")
	s.Equal(int64(2048), retrievedDoc.Versions[0].Size, "Version size should be 2048")
	
	// Attempt to add a version to a document with a different tenant ID
	version.ID = "" // Clear ID for new creation
	_, err = s.repo.AddVersion(s.ctx, &version)
	s.Error(err, "Adding version with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err), 
		"Error should be ResourceNotFoundError or AuthorizationError")
}

// TestUpdateVersionStatus tests updating the status of a document version with tenant isolation
func (s *DocumentRepositorySuite) TestUpdateVersionStatus() {
	// Create a test document with a version
	doc, docID := s.createTestDocument("status-test.pdf", "application/pdf", 1024, testTenantID1)
	
	// Add a version to the document
	version := models.NewDocumentVersion(docID, 1, 2048, "abc123", "s3://test-bucket/document.pdf", testUserID)
	versionID, err := s.repo.AddVersion(s.ctx, &version)
	s.NoError(err, "Adding version should succeed")
	
	// Update the version status
	err = s.repo.UpdateVersionStatus(s.ctx, versionID, models.VersionStatusAvailable, testTenantID1)
	s.NoError(err, "Updating version status should succeed")
	
	// Retrieve the document to verify the version status was updated
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.Equal(models.VersionStatusAvailable, retrievedDoc.Versions[0].Status, "Version status should be updated")
	s.Equal(models.DocumentStatusAvailable, retrievedDoc.Status, "Document status should also be updated")
	
	// Attempt to update a version with a different tenant ID
	err = s.repo.UpdateVersionStatus(s.ctx, versionID, models.VersionStatusFailed, testTenantID2)
	s.Error(err, "Updating version with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err), 
		"Error should be ResourceNotFoundError or AuthorizationError")
}

// TestAddAndUpdateMetadata tests adding and updating document metadata with tenant isolation
func (s *DocumentRepositorySuite) TestAddAndUpdateMetadata() {
	// Create a test document
	doc, docID := s.createTestDocument("metadata-test.pdf", "application/pdf", 1024, testTenantID1)
	
	// Add metadata to the document
	metaID, err := s.repo.AddMetadata(s.ctx, docID, "author", "Test Author", testTenantID1)
	s.NoError(err, "Adding metadata should succeed")
	s.NotEmpty(metaID, "Metadata ID should not be empty")
	
	// Retrieve the document to verify the metadata was added
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.Len(retrievedDoc.Metadata, 1, "Document should have 1 metadata entry")
	s.Equal("author", retrievedDoc.Metadata[0].Key, "Metadata key should be 'author'")
	s.Equal("Test Author", retrievedDoc.Metadata[0].Value, "Metadata value should be 'Test Author'")
	
	// Update the metadata
	err = s.repo.UpdateMetadata(s.ctx, docID, "author", "Updated Author", testTenantID1)
	s.NoError(err, "Updating metadata should succeed")
	
	// Retrieve the document again to verify the metadata was updated
	retrievedDoc, err = s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.Equal("Updated Author", retrievedDoc.GetMetadata("author"), "Metadata value should be updated")
	
	// Attempt to add/update metadata with a different tenant ID
	_, err = s.repo.AddMetadata(s.ctx, docID, "department", "HR", testTenantID2)
	s.Error(err, "Adding metadata with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err), 
		"Error should be ResourceNotFoundError or AuthorizationError")
	
	err = s.repo.UpdateMetadata(s.ctx, docID, "author", "Unauthorized Update", testTenantID2)
	s.Error(err, "Updating metadata with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err), 
		"Error should be ResourceNotFoundError or AuthorizationError")
}

// TestDeleteMetadata tests deleting document metadata with tenant isolation
func (s *DocumentRepositorySuite) TestDeleteMetadata() {
	// Create a test document with metadata
	doc, docID := s.createTestDocument("delete-metadata-test.pdf", "application/pdf", 1024, testTenantID1)
	
	// Add metadata to the document
	_, err := s.repo.AddMetadata(s.ctx, docID, "author", "Test Author", testTenantID1)
	s.NoError(err, "Adding metadata should succeed")
	_, err = s.repo.AddMetadata(s.ctx, docID, "department", "HR", testTenantID1)
	s.NoError(err, "Adding metadata should succeed")
	
	// Delete a metadata entry
	err = s.repo.DeleteMetadata(s.ctx, docID, "author", testTenantID1)
	s.NoError(err, "Deleting metadata should succeed")
	
	// Retrieve the document to verify the metadata was deleted
	retrievedDoc, err := s.repo.GetByID(s.ctx, docID, testTenantID1)
	s.NoError(err, "Document retrieval should succeed")
	s.Len(retrievedDoc.Metadata, 1, "Document should have 1 metadata entry remaining")
	s.Equal("department", retrievedDoc.Metadata[0].Key, "Remaining metadata key should be 'department'")
	s.Equal("HR", retrievedDoc.Metadata[0].Value, "Remaining metadata value should be 'HR'")
	
	// Attempt to delete metadata with a different tenant ID
	err = s.repo.DeleteMetadata(s.ctx, docID, "department", testTenantID2)
	s.Error(err, "Deleting metadata with incorrect tenant ID should fail")
	s.True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err), 
		"Error should be ResourceNotFoundError or AuthorizationError")
}

// TestGetDocumentsByIDs tests retrieving multiple documents by their IDs with tenant isolation
func (s *DocumentRepositorySuite) TestGetDocumentsByIDs() {
	// Create multiple test documents for tenant 1
	doc1, id1 := s.createTestDocument("tenant1-batch1.pdf", "application/pdf", 1024, testTenantID1)
	doc2, id2 := s.createTestDocument("tenant1-batch2.pdf", "application/pdf", 2048, testTenantID1)
	doc3, id3 := s.createTestDocument("tenant1-batch3.pdf", "application/pdf", 3072, testTenantID1)
	
	// Create test documents for tenant 2
	doc4, id4 := s.createTestDocument("tenant2-batch1.pdf", "application/pdf", 1024, testTenantID2)
	doc5, id5 := s.createTestDocument("tenant2-batch2.pdf", "application/pdf", 2048, testTenantID2)
	
	// Collect IDs for batch retrieval
	ids := []string{id1, id2, id3}
	
	// Retrieve documents by IDs for tenant 1
	docs, err := s.repo.GetDocumentsByIDs(s.ctx, ids, testTenantID1)
	s.NoError(err, "Batch document retrieval should succeed")
	s.Len(docs, 3, "Should return 3 documents")
	
	// Check that all documents belong to tenant 1
	for _, doc := range docs {
		s.Equal(testTenantID1, doc.TenantID, "Document should belong to tenant 1")
	}
	
	// Include IDs from tenant 2 in the batch
	mixedIDs := []string{id1, id4, id5}
	
	// Retrieve documents, which should only return tenant 1's documents
	docs, err = s.repo.GetDocumentsByIDs(s.ctx, mixedIDs, testTenantID1)
	s.NoError(err, "Batch document retrieval should succeed")
	s.Len(docs, 1, "Should return only 1 document belonging to tenant 1")
	s.Equal(id1, docs[0].ID, "Document ID should match the tenant 1 document")
}

// Helper method to create a test document
func (s *DocumentRepositorySuite) createTestDocument(name, contentType string, size int64, tenantID string) (*models.Document, string) {
	doc := models.NewDocument(name, contentType, size, testFolderID, tenantID, testUserID)
	docID, err := s.repo.Create(s.ctx, &doc)
	s.NoError(err, "Failed to create test document")
	s.NotEmpty(docID, "Document ID should not be empty")
	
	doc.ID = docID
	return &doc, docID
}

// Helper method to clean up test data
func (s *DocumentRepositorySuite) cleanupTestData() {
	// Get database instance
	db, err := postgres.GetDB()
	s.NoError(err, "Failed to get database instance")

	// Delete all test documents and related data
	db.Exec("DELETE FROM document_versions WHERE document_id IN (SELECT id FROM documents WHERE tenant_id LIKE 'tenant-test-%')")
	db.Exec("DELETE FROM document_metadata WHERE document_id IN (SELECT id FROM documents WHERE tenant_id LIKE 'tenant-test-%')")
	db.Exec("DELETE FROM documents WHERE tenant_id LIKE 'tenant-test-%'")
}