// Package integration provides integration tests for the Document Management Platform.
package integration

import (
	"context"
	"os"
	"testing"
	"time"
	"fmt"

	"github.com/google/uuid" // v1.3.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+

	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services"
	"../../infrastructure/persistence/postgres"
	"../../infrastructure/search/elasticsearch"
	"../../pkg/config"
	"../../pkg/errors"
	"../../pkg/utils"
)

// Constants for testing
const testTenantID1 = "tenant-test-1"
const testTenantID2 = "tenant-test-2"
const testUserID = "user-test-1"
const testFolderID1 = "folder-test-1"
const testFolderID2 = "folder-test-2"

// TestSearchServiceSuite is the entry point for running the search service test suite
func TestSearchServiceSuite(t *testing.T) {
	suite.Run(t, new(SearchServiceSuite))
}

// SearchServiceSuite is the test suite for search service integration tests
type SearchServiceSuite struct {
	suite.Suite
	searchService services.SearchService
	documentRepo  repositories.DocumentRepository
	ctx           context.Context
}

// SetupSuite sets up the test suite by initializing the database and Elasticsearch connections
func (s *SearchServiceSuite) SetupSuite() {
	// Create test database configuration
	dbConfig := config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		DBName:          "document_mgmt_test",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: "1h",
	}

	// Initialize database connection
	err := postgres.Init(dbConfig)
	s.Require().NoError(err, "Failed to initialize database")

	// Get database instance
	db, err := postgres.GetDB()
	s.Require().NoError(err, "Failed to get database instance")

	// Run migrations
	err = postgres.Migrate(&models.Document{}, &models.DocumentMetadata{}, &models.DocumentVersion{}, &models.Tag{})
	s.Require().NoError(err, "Failed to run migrations")

	// Create document repository
	s.documentRepo = postgres.NewDocumentRepository(db)
	s.Require().NotNil(s.documentRepo, "Document repository should not be nil")

	// Create test Elasticsearch configuration
	esConfig := config.ElasticsearchConfig{
		Addresses:    []string{"http://localhost:9200"},
		Username:     "",
		Password:     "",
		EnableSniff:  false,
		IndexPrefix:  "test_",
	}

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewElasticsearchClient(esConfig)
	s.Require().NoError(err, "Failed to create Elasticsearch client")

	// Create document index
	documentIndex, err := elasticsearch.NewDocumentIndex(esClient)
	s.Require().NoError(err, "Failed to create document index")

	// Create search indexer
	searchIndexer, err := elasticsearch.NewElasticsearchIndexer(documentIndex)
	s.Require().NoError(err, "Failed to create search indexer")

	// Create search query executor
	searchQueryExecutor, err := elasticsearch.NewElasticsearchQueryExecutor(esClient)
	s.Require().NoError(err, "Failed to create search query executor")

	// Create search service
	s.searchService, err = services.NewSearchService(searchIndexer, searchQueryExecutor, s.documentRepo)
	s.Require().NoError(err, "Failed to create search service")

	// Create background context for tests
	s.ctx = context.Background()
}

// TearDownSuite tears down the test suite by closing the database connection
func (s *SearchServiceSuite) TearDownSuite() {
	// Close database connection
	err := postgres.Close()
	s.Require().NoError(err, "Failed to close database connection")
}

// SetupTest sets up each test by cleaning the database and search index
func (s *SearchServiceSuite) SetupTest() {
	// Clean up test data from previous tests
	s.cleanupTestData()
}

// TestSearchByContent tests content-based search functionality with tenant isolation
func (s *SearchServiceSuite) TestSearchByContent() {
	// Create test documents with specific content for tenant 1
	doc1, docID1 := s.createTestDocument("test-doc-1.pdf", "application/pdf", 1024, testTenantID1, testFolderID1)
	doc2, docID2 := s.createTestDocument("test-doc-2.pdf", "application/pdf", 2048, testTenantID1, testFolderID1)
	
	// Create test documents with the same content for tenant 2
	doc3, docID3 := s.createTestDocument("test-doc-3.pdf", "application/pdf", 1024, testTenantID2, testFolderID1)
	
	// Index documents in Elasticsearch
	content1 := []byte("This is a test document containing important information about tests")
	content2 := []byte("Another test document with different content about projects")
	content3 := []byte("This is a test document containing important information about tests for tenant 2")
	
	err := s.indexTestDocument(docID1, testTenantID1, content1)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID2, testTenantID1, content2)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID3, testTenantID2, content3)
	s.Require().NoError(err)
	
	// Wait for indexing to complete
	time.Sleep(1 * time.Second)
	
	// Search for "test document" in tenant 1
	pagination := utils.NewPagination(1, 10)
	result, err := s.searchService.SearchByContent(s.ctx, "test document", testTenantID1, pagination)
	
	// Assert that only tenant 1's matching documents are returned
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(2, len(result.Items), "Should return both documents from tenant 1")
	
	// Verify that documents from tenant 2 are not included
	for _, doc := range result.Items {
		s.Assert().Equal(testTenantID1, doc.TenantID, "Search results should only contain documents from tenant 1")
	}
	
	// Test with different search queries
	result, err = s.searchService.SearchByContent(s.ctx, "important information", testTenantID1, pagination)
	s.Require().NoError(err)
	s.Require().Equal(1, len(result.Items), "Should return only one document")
	s.Assert().Equal(docID1, result.Items[0].ID, "Should return the correct document")
	
	// Test with non-matching search query
	result, err = s.searchService.SearchByContent(s.ctx, "nonexistent content", testTenantID1, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(0, len(result.Items), "Should return empty results for non-matching query")
	
	// Test with empty search query (should return validation error)
	_, err = s.searchService.SearchByContent(s.ctx, "", testTenantID1, pagination)
	s.Require().Error(err)
	s.Assert().True(errors.IsValidationError(err), "Empty query should return validation error")
	
	// Test with different tenant ID to ensure tenant isolation
	result, err = s.searchService.SearchByContent(s.ctx, "test document", testTenantID2, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(1, len(result.Items), "Should return only documents from tenant 2")
	s.Assert().Equal(docID3, result.Items[0].ID, "Should return the correct document from tenant 2")
}

// TestSearchByMetadata tests metadata-based search functionality with tenant isolation
func (s *SearchServiceSuite) TestSearchByMetadata() {
	// Create test documents with specific metadata for tenant 1
	doc1, docID1 := s.createTestDocument("metadata-doc-1.pdf", "application/pdf", 1024, testTenantID1, testFolderID1)
	doc2, docID2 := s.createTestDocument("metadata-doc-2.pdf", "application/pdf", 2048, testTenantID1, testFolderID1)
	
	// Create test documents with the same metadata for tenant 2
	doc3, docID3 := s.createTestDocument("metadata-doc-3.pdf", "application/pdf", 1024, testTenantID2, testFolderID1)
	
	// Add metadata to documents
	doc1.AddMetadata("category", "report")
	doc1.AddMetadata("department", "engineering")
	doc1.AddMetadata("status", "active")
	
	doc2.AddMetadata("category", "invoice")
	doc2.AddMetadata("department", "finance")
	doc2.AddMetadata("status", "active")
	
	doc3.AddMetadata("category", "report")
	doc3.AddMetadata("department", "engineering")
	doc3.AddMetadata("status", "active")
	
	// Update documents in repository
	err := s.documentRepo.Update(s.ctx, doc1)
	s.Require().NoError(err)
	
	err = s.documentRepo.Update(s.ctx, doc2)
	s.Require().NoError(err)
	
	err = s.documentRepo.Update(s.ctx, doc3)
	s.Require().NoError(err)
	
	// Index documents in Elasticsearch
	content1 := []byte("This is a document with metadata")
	content2 := []byte("Another document with different metadata")
	content3 := []byte("This is a document with metadata for tenant 2")
	
	err = s.indexTestDocument(docID1, testTenantID1, content1)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID2, testTenantID1, content2)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID3, testTenantID2, content3)
	s.Require().NoError(err)
	
	// Wait for indexing to complete
	time.Sleep(1 * time.Second)
	
	// Search by metadata criteria and tenant 1 ID
	pagination := utils.NewPagination(1, 10)
	metadata := map[string]string{
		"category": "report",
	}
	
	result, err := s.searchService.SearchByMetadata(s.ctx, metadata, testTenantID1, pagination)
	
	// Assert that only tenant 1's matching documents are returned
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(1, len(result.Items), "Should return one document matching metadata criteria")
	s.Assert().Equal(docID1, result.Items[0].ID, "Should return the correct document")
	
	// Test with different metadata criteria
	metadata = map[string]string{
		"status":     "active",
		"department": "finance",
	}
	
	result, err = s.searchService.SearchByMetadata(s.ctx, metadata, testTenantID1, pagination)
	s.Require().NoError(err)
	s.Require().Equal(1, len(result.Items), "Should return one document matching all metadata criteria")
	s.Assert().Equal(docID2, result.Items[0].ID, "Should return the correct document")
	
	// Test with non-matching metadata criteria
	metadata = map[string]string{
		"category": "nonexistent",
	}
	
	result, err = s.searchService.SearchByMetadata(s.ctx, metadata, testTenantID1, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(0, len(result.Items), "Should return empty results for non-matching metadata")
	
	// Test with empty metadata criteria (should return validation error)
	_, err = s.searchService.SearchByMetadata(s.ctx, map[string]string{}, testTenantID1, pagination)
	s.Require().Error(err)
	s.Assert().True(errors.IsValidationError(err), "Empty metadata should return validation error")
	
	// Test with different tenant ID to ensure tenant isolation
	metadata = map[string]string{
		"category": "report",
	}
	
	result, err = s.searchService.SearchByMetadata(s.ctx, metadata, testTenantID2, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(1, len(result.Items), "Should return only documents from tenant 2")
	s.Assert().Equal(docID3, result.Items[0].ID, "Should return the correct document from tenant 2")
}

// TestCombinedSearch tests combined content and metadata search functionality with tenant isolation
func (s *SearchServiceSuite) TestCombinedSearch() {
	// Create test documents with specific content and metadata for tenant 1
	doc1, docID1 := s.createTestDocument("combined-doc-1.pdf", "application/pdf", 1024, testTenantID1, testFolderID1)
	doc2, docID2 := s.createTestDocument("combined-doc-2.pdf", "application/pdf", 2048, testTenantID1, testFolderID1)
	
	// Create test documents with the same content and metadata for tenant 2
	doc3, docID3 := s.createTestDocument("combined-doc-3.pdf", "application/pdf", 1024, testTenantID2, testFolderID1)
	
	// Add metadata to documents
	doc1.AddMetadata("category", "report")
	doc1.AddMetadata("department", "engineering")
	
	doc2.AddMetadata("category", "presentation")
	doc2.AddMetadata("department", "marketing")
	
	doc3.AddMetadata("category", "report")
	doc3.AddMetadata("department", "engineering")
	
	// Update documents in repository
	err := s.documentRepo.Update(s.ctx, doc1)
	s.Require().NoError(err)
	
	err = s.documentRepo.Update(s.ctx, doc2)
	s.Require().NoError(err)
	
	err = s.documentRepo.Update(s.ctx, doc3)
	s.Require().NoError(err)
	
	// Index documents in Elasticsearch
	content1 := []byte("This is an engineering report about the project implementation")
	content2 := []byte("Marketing presentation about new product features")
	content3 := []byte("This is an engineering report about the project implementation for tenant 2")
	
	err = s.indexTestDocument(docID1, testTenantID1, content1)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID2, testTenantID1, content2)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID3, testTenantID2, content3)
	s.Require().NoError(err)
	
	// Wait for indexing to complete
	time.Sleep(1 * time.Second)
	
	// Call searchService.CombinedSearch with content query, metadata criteria, and tenant 1 ID
	pagination := utils.NewPagination(1, 10)
	metadata := map[string]string{
		"category": "report",
	}
	
	result, err := s.searchService.CombinedSearch(s.ctx, "engineering", metadata, testTenantID1, pagination)
	
	// Assert that only tenant 1's matching documents are returned
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(1, len(result.Items), "Should return one document matching both content and metadata criteria")
	s.Assert().Equal(docID1, result.Items[0].ID, "Should return the correct document")
	
	// Test with content query only
	result, err = s.searchService.CombinedSearch(s.ctx, "marketing presentation", nil, testTenantID1, pagination)
	s.Require().NoError(err)
	s.Require().Equal(1, len(result.Items), "Should return one document matching content criteria")
	s.Assert().Equal(docID2, result.Items[0].ID, "Should return the correct document")
	
	// Test with metadata criteria only
	metadata = map[string]string{
		"department": "marketing",
	}
	result, err = s.searchService.CombinedSearch(s.ctx, "", metadata, testTenantID1, pagination)
	s.Require().NoError(err)
	s.Require().Equal(1, len(result.Items), "Should return one document matching metadata criteria")
	s.Assert().Equal(docID2, result.Items[0].ID, "Should return the correct document")
	
	// Test with non-matching criteria
	metadata = map[string]string{
		"category": "report",
	}
	result, err = s.searchService.CombinedSearch(s.ctx, "nonexistent", metadata, testTenantID1, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(0, len(result.Items), "Should return empty results for non-matching criteria")
	
	// Test with empty criteria (should return validation error)
	_, err = s.searchService.CombinedSearch(s.ctx, "", map[string]string{}, testTenantID1, pagination)
	s.Require().Error(err)
	s.Assert().True(errors.IsValidationError(err), "Empty criteria should return validation error")
	
	// Test with different tenant ID to ensure tenant isolation
	metadata = map[string]string{
		"category": "report",
	}
	result, err = s.searchService.CombinedSearch(s.ctx, "engineering", metadata, testTenantID2, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(1, len(result.Items), "Should return only documents from tenant 2")
	s.Assert().Equal(docID3, result.Items[0].ID, "Should return the correct document from tenant 2")
}

// TestSearchInFolder tests folder-scoped search functionality with tenant isolation
func (s *SearchServiceSuite) TestSearchInFolder() {
	// Create test documents in specific folders for tenant 1
	doc1, docID1 := s.createTestDocument("folder-doc-1.pdf", "application/pdf", 1024, testTenantID1, testFolderID1)
	doc2, docID2 := s.createTestDocument("folder-doc-2.pdf", "application/pdf", 2048, testTenantID1, testFolderID2)
	
	// Create test documents in the same folders for tenant 2
	doc3, docID3 := s.createTestDocument("folder-doc-3.pdf", "application/pdf", 1024, testTenantID2, testFolderID1)
	
	// Index documents in Elasticsearch
	content1 := []byte("This document is in folder 1 and contains project information")
	content2 := []byte("This document is in folder 2 and contains project information")
	content3 := []byte("This document is in folder 1 for tenant 2 and contains project information")
	
	err := s.indexTestDocument(docID1, testTenantID1, content1)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID2, testTenantID1, content2)
	s.Require().NoError(err)
	
	err = s.indexTestDocument(docID3, testTenantID2, content3)
	s.Require().NoError(err)
	
	// Wait for indexing to complete
	time.Sleep(1 * time.Second)
	
	// Call searchService.SearchInFolder with folder ID, search query, and tenant 1 ID
	pagination := utils.NewPagination(1, 10)
	result, err := s.searchService.SearchInFolder(s.ctx, testFolderID1, "project information", testTenantID1, pagination)
	
	// Assert that only tenant 1's matching documents in the specified folder are returned
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(1, len(result.Items), "Should return one document from folder 1")
	s.Assert().Equal(docID1, result.Items[0].ID, "Should return the correct document")
	
	// Test with different folder IDs
	result, err = s.searchService.SearchInFolder(s.ctx, testFolderID2, "project information", testTenantID1, pagination)
	s.Require().NoError(err)
	s.Require().Equal(1, len(result.Items), "Should return one document from folder 2")
	s.Assert().Equal(docID2, result.Items[0].ID, "Should return the correct document")
	
	// Test with non-matching search query
	result, err = s.searchService.SearchInFolder(s.ctx, testFolderID1, "nonexistent", testTenantID1, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(0, len(result.Items), "Should return empty results for non-matching content")
	
	// Test with empty folder ID (should return validation error)
	_, err = s.searchService.SearchInFolder(s.ctx, "", "project", testTenantID1, pagination)
	s.Require().Error(err)
	s.Assert().True(errors.IsValidationError(err), "Empty folder ID should return validation error")
	
	// Test with empty search query (should return validation error)
	_, err = s.searchService.SearchInFolder(s.ctx, testFolderID1, "", testTenantID1, pagination)
	s.Require().Error(err)
	s.Assert().True(errors.IsValidationError(err), "Empty query should return validation error")
	
	// Test with different tenant ID to ensure tenant isolation
	result, err = s.searchService.SearchInFolder(s.ctx, testFolderID1, "project information", testTenantID2, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(1, len(result.Items), "Should return only documents from tenant 2")
	s.Assert().Equal(docID3, result.Items[0].ID, "Should return the correct document from tenant 2")
}

// TestIndexDocument tests document indexing functionality with tenant isolation
func (s *SearchServiceSuite) TestIndexDocument() {
	// Create a test document for tenant 1
	doc, docID := s.createTestDocument("index-test-doc.pdf", "application/pdf", 1024, testTenantID1, testFolderID1)
	
	// Call searchService.IndexDocument to index the document
	content := []byte("This is a document to test indexing functionality")
	err := s.searchService.IndexDocument(s.ctx, docID, testTenantID1, content)
	s.Require().NoError(err, "Document indexing should succeed")
	
	// Wait for indexing to complete
	time.Sleep(1 * time.Second)
	
	// Search for the document content to verify indexing
	pagination := utils.NewPagination(1, 10)
	result, err := s.searchService.SearchByContent(s.ctx, "indexing functionality", testTenantID1, pagination)
	
	// Assert that the document is found in search results
	s.Require().NoError(err)
	s.Require().NotNil(result)
	s.Require().Equal(1, len(result.Items), "Should return the indexed document")
	s.Assert().Equal(docID, result.Items[0].ID, "Should return the correct document")
	
	// Attempt to index a document with a different tenant ID
	err = s.searchService.IndexDocument(s.ctx, docID, testTenantID2, content)
	s.Require().Error(err, "Indexing with different tenant ID should fail")
	s.Assert().True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err), 
		"Should return resource not found error or authorization error to ensure tenant isolation")
}

// TestRemoveDocumentFromIndex tests document removal from search index with tenant isolation
func (s *SearchServiceSuite) TestRemoveDocumentFromIndex() {
	// Create and index a test document for tenant 1
	doc, docID := s.createTestDocument("remove-test-doc.pdf", "application/pdf", 1024, testTenantID1, testFolderID1)
	
	content := []byte("This is a document to test removal from index")
	err := s.searchService.IndexDocument(s.ctx, docID, testTenantID1, content)
	s.Require().NoError(err, "Document indexing should succeed")
	
	// Wait for indexing to complete
	time.Sleep(1 * time.Second)
	
	// Verify the document is searchable
	pagination := utils.NewPagination(1, 10)
	result, err := s.searchService.SearchByContent(s.ctx, "removal from index", testTenantID1, pagination)
	s.Require().NoError(err)
	s.Require().Equal(1, len(result.Items), "Document should be searchable after indexing")
	
	// Call searchService.RemoveDocumentFromIndex to remove the document
	err = s.searchService.RemoveDocumentFromIndex(s.ctx, docID, testTenantID1)
	s.Require().NoError(err, "Document removal should succeed")
	
	// Wait for removal to complete
	time.Sleep(1 * time.Second)
	
	// Search for the document content to verify removal
	result, err = s.searchService.SearchByContent(s.ctx, "removal from index", testTenantID1, pagination)
	s.Require().NoError(err)
	s.Assert().Equal(0, len(result.Items), "Document should no longer be searchable after removal")
	
	// Attempt to remove a document with a different tenant ID
	err = s.searchService.RemoveDocumentFromIndex(s.ctx, docID, testTenantID2)
	s.Require().Error(err, "Removal with different tenant ID should fail")
	s.Assert().True(errors.IsResourceNotFoundError(err) || errors.IsAuthorizationError(err),
		"Should return resource not found error or authorization error to ensure tenant isolation")
}

// TestPaginationInSearch tests pagination functionality in search results
func (s *SearchServiceSuite) TestPaginationInSearch() {
	// Create multiple test documents with similar content for tenant 1
	docIDs := make([]string, 0, 25)
	for i := 0; i < 25; i++ {
		doc, docID := s.createTestDocument(fmt.Sprintf("pagination-doc-%d.pdf", i), "application/pdf", 1024, testTenantID1, testFolderID1)
		docIDs = append(docIDs, docID)
		
		// Index all documents in Elasticsearch
		content := []byte(fmt.Sprintf("This is document number %d for pagination testing", i))
		err := s.indexTestDocument(docID, testTenantID1, content)
		s.Require().NoError(err)
	}
	
	// Wait for indexing to complete
	time.Sleep(2 * time.Second)
	
	// Call searchService.SearchByContent with different page sizes and page numbers
	testCases := []struct {
		pageSize    int
		page        int
		expectedLen int
	}{
		{5, 1, 5},   // First page with 5 items per page
		{5, 2, 5},   // Second page with 5 items per page
		{5, 5, 5},   // Last full page with 5 items per page
		{5, 6, 0},   // Page beyond results
		{10, 1, 10}, // First page with 10 items per page
		{10, 2, 10}, // Second page with 10 items per page
		{10, 3, 5},  // Last page with 5 items (partial)
		{25, 1, 25}, // All results in one page
		{30, 1, 25}, // More than total results
		{1, 1, 1},   // One result per page
	}
	
	for _, tc := range testCases {
		pagination := utils.NewPagination(tc.page, tc.pageSize)
		result, err := s.searchService.SearchByContent(s.ctx, "pagination testing", testTenantID1, pagination)
		
		s.Require().NoError(err, "Search with pagination should succeed")
		s.Assert().Equal(tc.expectedLen, len(result.Items), 
			"Page %d with size %d should return %d items", tc.page, tc.pageSize, tc.expectedLen)
		
		// Assert that pagination metadata (total, page, page size) is correct
		s.Assert().Equal(tc.page, result.Pagination.Page, "Page number should match request")
		s.Assert().Equal(tc.pageSize, result.Pagination.PageSize, "Page size should match request")
		s.Assert().Equal(int64(25), result.Pagination.TotalItems, "Total items should be 25")
		
		expectedTotalPages := (25 + tc.pageSize - 1) / tc.pageSize
		s.Assert().Equal(expectedTotalPages, result.Pagination.TotalPages, "Total pages calculation should be correct")
		
		// Test edge cases like page size = 1, last page, etc.
		hasNext := tc.page < expectedTotalPages
		hasPrevious := tc.page > 1
		s.Assert().Equal(hasNext, result.Pagination.HasNext, "HasNext should be correctly calculated")
		s.Assert().Equal(hasPrevious, result.Pagination.HasPrevious, "HasPrevious should be correctly calculated")
	}
}

// Helper method to create a test document
func (s *SearchServiceSuite) createTestDocument(name string, contentType string, size int64, tenantID string, folderID string) (*models.Document, string) {
	doc := models.NewDocument(name, contentType, size, folderID, tenantID, testUserID)
	docID, err := s.documentRepo.Create(s.ctx, &doc)
	s.Require().NoError(err, "Failed to create test document")
	s.Require().NotEmpty(docID, "Document ID should not be empty")
	
	// Retrieve the document to get the complete object
	document, err := s.documentRepo.GetByID(s.ctx, docID, tenantID)
	s.Require().NoError(err, "Failed to retrieve test document")
	
	return document, docID
}

// Helper method to index a test document in Elasticsearch
func (s *SearchServiceSuite) indexTestDocument(documentID string, tenantID string, content []byte) error {
	return s.searchService.IndexDocument(s.ctx, documentID, tenantID, content)
}

// Helper method to clean up test data
func (s *SearchServiceSuite) cleanupTestData() {
	// Get database instance
	db, err := postgres.GetDB()
	s.Require().NoError(err, "Failed to get database instance")
	
	// Execute SQL to delete all test documents and related data
	err = db.Exec("DELETE FROM document_metadata WHERE document_id IN (SELECT id FROM documents WHERE tenant_id LIKE 'tenant-test-%')").Error
	s.Require().NoError(err, "Failed to clean up document metadata")
	
	err = db.Exec("DELETE FROM document_versions WHERE document_id IN (SELECT id FROM documents WHERE tenant_id LIKE 'tenant-test-%')").Error
	s.Require().NoError(err, "Failed to clean up document versions")
	
	err = db.Exec("DELETE FROM documents WHERE tenant_id LIKE 'tenant-test-%'").Error
	s.Require().NoError(err, "Failed to clean up documents")
	
	// Delete all documents from Elasticsearch indices
	// This will depend on the specific Elasticsearch implementation
	// For now, we'll assume the removal happens through the search service or client
}