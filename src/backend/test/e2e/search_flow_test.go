// Package e2e contains end-to-end tests for the Document Management Platform.
package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid" // v1.3.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+
	"go.uber.org/zap" // v1.24.0+

	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services"
	"../../application/usecases/searchusecase"
	"../../pkg/errors"
	"../../pkg/config"
	"../../pkg/logger"
	"../../pkg/utils"
)

// TestSearchFlow is the entry point for the search flow end-to-end test suite
func TestSearchFlow(t *testing.T) {
	suite.Run(t, new(SearchFlowTestSuite))
}

// SearchFlowTestSuite is the test suite for search functionality end-to-end tests
type SearchFlowTestSuite struct {
	suite.Suite
	documentRepo    repositories.DocumentRepository
	documentService services.DocumentService
	storageService  services.StorageService
	searchService   services.SearchService
	searchUseCase   searchusecase.SearchUseCase
	testTenantID    string
	testUserID      string
	testFolderID    string
	logger          *zap.Logger
	testDocumentIDs map[string]string // Maps document name to ID for cleanup
}

// SetupSuite initializes the test suite before any tests run
func (s *SearchFlowTestSuite) SetupSuite() {
	// Load test configuration
	cfg := &config.Config{}
	err := config.Load(cfg)
	require.NoError(s.T(), err, "Failed to load test configuration")

	// Set up test tenant ID, user ID, and folder ID using UUID
	s.testTenantID = uuid.New().String()
	s.testUserID = uuid.New().String()
	s.testFolderID = uuid.New().String()

	// Initialize logger
	s.logger = logger.NewLogger()

	// Initialize testDocumentIDs map to track created documents
	s.testDocumentIDs = make(map[string]string)
}

// SetupTest sets up each test before it runs
func (s *SearchFlowTestSuite) SetupTest() {
	// Create mock document repository
	mockDocRepo := new(mockDocumentRepository)
	s.documentRepo = mockDocRepo

	// Create mock storage service
	mockStorageSvc := new(mockStorageService)
	s.storageService = mockStorageSvc

	// Create mock document service
	mockDocSvc := new(mockDocumentService)
	s.documentService = mockDocSvc

	// Create mock search service
	mockSearchSvc := new(mockSearchService)
	s.searchService = mockSearchSvc

	// Create search use case with dependencies
	var err error
	s.searchUseCase, err = searchusecase.NewSearchUseCase(s.searchService)
	require.NoError(s.T(), err, "Failed to create search use case")
}

// TearDownTest cleans up after each test
func (s *SearchFlowTestSuite) TearDownTest() {
	// Clean up any test documents from repositories
	for name, docID := range s.testDocumentIDs {
		s.documentRepo.(*mockDocumentRepository).On("Delete", mock.Anything, docID, s.testTenantID).Return(nil)
	}
}

// TestContentSearch tests searching documents by content
func (s *SearchFlowTestSuite) TestContentSearch() {
	// Create test context
	ctx := context.Background()
	
	// Create and index multiple test documents with different content
	doc1ID, err := s.createTestDocument("test1.pdf", "application/pdf", []byte("This is a test document with specific content"), nil, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 1")
	
	doc2ID, err := s.createTestDocument("test2.pdf", "application/pdf", []byte("This document has different content"), nil, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 2")
	
	// Create a document for another tenant (should not be included in results)
	otherTenantID := uuid.New().String()
	doc3ID, err := s.createTestDocument("test3.pdf", "application/pdf", []byte("This document belongs to another tenant"), nil, otherTenantID)
	require.NoError(s.T(), err, "Failed to create test document 3")
	
	// Index documents
	require.NoError(s.T(), s.indexTestDocument(doc1ID, []byte("This is a test document with specific content"), s.testTenantID))
	require.NoError(s.T(), s.indexTestDocument(doc2ID, []byte("This document has different content"), s.testTenantID))
	require.NoError(s.T(), s.indexTestDocument(doc3ID, []byte("This document belongs to another tenant"), otherTenantID))
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock expectations for content search
	expectedDocs := []models.Document{
		{
			ID:          doc1ID,
			Name:        "test1.pdf",
			ContentType: "application/pdf",
			Size:        44,
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult := utils.NewPaginatedResult(expectedDocs, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"specific content", 
		s.testTenantID, 
		pagination,
	).Return(expectedResult, nil)
	
	// Call searchUseCase.SearchByContent with a search query
	result, err := s.searchUseCase.SearchByContent(ctx, "specific content", s.testTenantID, pagination)
	
	// Assert that correct documents are returned in search results
	require.NoError(s.T(), err, "Search by content should not return an error")
	assert.Equal(s.T(), 1, len(result.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc1ID, result.Items[0].ID, "Search should return the correct document")
	
	// Verify that search results are properly paginated
	assert.Equal(s.T(), int64(1), result.Pagination.TotalItems, "Total items should be 1")
	assert.Equal(s.T(), 1, result.Pagination.Page, "Current page should be 1")
	assert.Equal(s.T(), 10, result.Pagination.PageSize, "Page size should be 10")
	
	// Verify that documents from other tenants are not included in results
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"specific content", 
		otherTenantID, 
		pagination,
	).Return(utils.PaginatedResult[models.Document]{}, nil)
	
	otherResult, err := s.searchUseCase.SearchByContent(ctx, "specific content", otherTenantID, pagination)
	require.NoError(s.T(), err, "Search in other tenant should not return an error")
	assert.Equal(s.T(), 0, len(otherResult.Items), "Search in other tenant should return 0 documents")
}

// TestMetadataSearch tests searching documents by metadata
func (s *SearchFlowTestSuite) TestMetadataSearch() {
	// Create test context
	ctx := context.Background()
	
	// Create and index multiple test documents with different metadata
	metadata1 := map[string]string{
		"department": "finance",
		"author":     "john.doe",
		"year":       "2023",
	}
	
	metadata2 := map[string]string{
		"department": "hr",
		"author":     "jane.smith",
		"year":       "2023",
	}
	
	doc1ID, err := s.createTestDocument("finance-report.pdf", "application/pdf", []byte("Finance report content"), metadata1, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 1")
	
	doc2ID, err := s.createTestDocument("hr-policy.pdf", "application/pdf", []byte("HR policy content"), metadata2, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 2")
	
	// Index documents
	require.NoError(s.T(), s.indexTestDocument(doc1ID, []byte("Finance report content"), s.testTenantID))
	require.NoError(s.T(), s.indexTestDocument(doc2ID, []byte("HR policy content"), s.testTenantID))
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock expectations for metadata search
	searchMetadata := map[string]string{
		"department": "finance",
	}
	
	expectedDocs := []models.Document{
		{
			ID:          doc1ID,
			Name:        "finance-report.pdf",
			ContentType: "application/pdf",
			Size:        22,
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult := utils.NewPaginatedResult(expectedDocs, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByMetadata", 
		mock.Anything, 
		searchMetadata, 
		s.testTenantID, 
		pagination,
	).Return(expectedResult, nil)
	
	// Call searchUseCase.SearchByMetadata with metadata criteria
	result, err := s.searchUseCase.SearchByMetadata(ctx, searchMetadata, s.testTenantID, pagination)
	
	// Assert that correct documents are returned in search results
	require.NoError(s.T(), err, "Search by metadata should not return an error")
	assert.Equal(s.T(), 1, len(result.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc1ID, result.Items[0].ID, "Search should return the correct document")
	
	// Verify that search results are properly paginated
	assert.Equal(s.T(), int64(1), result.Pagination.TotalItems, "Total items should be 1")
	
	// Verify that documents from other tenants are not included in results
	otherTenantID := uuid.New().String()
	s.searchService.(*mockSearchService).On(
		"SearchByMetadata", 
		mock.Anything, 
		searchMetadata, 
		otherTenantID, 
		pagination,
	).Return(utils.PaginatedResult[models.Document]{}, nil)
	
	otherResult, err := s.searchUseCase.SearchByMetadata(ctx, searchMetadata, otherTenantID, pagination)
	require.NoError(s.T(), err, "Search in other tenant should not return an error")
	assert.Equal(s.T(), 0, len(otherResult.Items), "Search in other tenant should return 0 documents")
}

// TestCombinedSearch tests searching documents by both content and metadata
func (s *SearchFlowTestSuite) TestCombinedSearch() {
	// Create test context
	ctx := context.Background()
	
	// Create and index multiple test documents with different content and metadata
	metadata1 := map[string]string{
		"department": "finance",
		"type":       "report",
	}
	
	metadata2 := map[string]string{
		"department": "finance",
		"type":       "invoice",
	}
	
	doc1ID, err := s.createTestDocument("finance-report.pdf", "application/pdf", []byte("This is a quarterly financial report"), metadata1, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 1")
	
	doc2ID, err := s.createTestDocument("finance-invoice.pdf", "application/pdf", []byte("This is an invoice for financial services"), metadata2, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 2")
	
	// Index documents
	require.NoError(s.T(), s.indexTestDocument(doc1ID, []byte("This is a quarterly financial report"), s.testTenantID))
	require.NoError(s.T(), s.indexTestDocument(doc2ID, []byte("This is an invoice for financial services"), s.testTenantID))
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up search criteria
	searchContent := "financial report"
	searchMetadata := map[string]string{
		"department": "finance",
		"type":       "report",
	}
	
	// Set up mock expectations for combined search
	expectedDocs := []models.Document{
		{
			ID:          doc1ID,
			Name:        "finance-report.pdf",
			ContentType: "application/pdf",
			Size:        36,
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult := utils.NewPaginatedResult(expectedDocs, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"CombinedSearch", 
		mock.Anything, 
		searchContent, 
		searchMetadata, 
		s.testTenantID, 
		pagination,
	).Return(expectedResult, nil)
	
	// Call searchUseCase.CombinedSearch with content query and metadata criteria
	result, err := s.searchUseCase.CombinedSearch(ctx, searchContent, searchMetadata, s.testTenantID, pagination)
	
	// Assert that correct documents are returned in search results
	require.NoError(s.T(), err, "Combined search should not return an error")
	assert.Equal(s.T(), 1, len(result.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc1ID, result.Items[0].ID, "Search should return the correct document")
	
	// Verify that search results are properly paginated
	assert.Equal(s.T(), int64(1), result.Pagination.TotalItems, "Total items should be 1")
	
	// Verify that documents from other tenants are not included in results
	otherTenantID := uuid.New().String()
	s.searchService.(*mockSearchService).On(
		"CombinedSearch", 
		mock.Anything, 
		searchContent, 
		searchMetadata, 
		otherTenantID, 
		pagination,
	).Return(utils.PaginatedResult[models.Document]{}, nil)
	
	otherResult, err := s.searchUseCase.CombinedSearch(ctx, searchContent, searchMetadata, otherTenantID, pagination)
	require.NoError(s.T(), err, "Search in other tenant should not return an error")
	assert.Equal(s.T(), 0, len(otherResult.Items), "Search in other tenant should return 0 documents")
}

// TestFolderSearch tests searching documents within a specific folder
func (s *SearchFlowTestSuite) TestFolderSearch() {
	// Create test context
	ctx := context.Background()
	
	// Create folder IDs
	folder1ID := s.testFolderID
	folder2ID := uuid.New().String()
	
	// Create and index multiple test documents in different folders
	doc1ID, err := s.createTestDocument("folder1-doc.pdf", "application/pdf", []byte("This document is in folder 1"), nil, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 1")
	
	doc2ID, err := s.createTestDocument("folder2-doc.pdf", "application/pdf", []byte("This document is in folder 2"), nil, s.testTenantID)
	require.NoError(s.T(), err, "Failed to create test document 2")
	
	// Update folder ID for second document
	s.documentRepo.(*mockDocumentRepository).On("Update", mock.Anything, mock.MatchedBy(func(d *models.Document) bool {
		return d.ID == doc2ID && d.FolderID == folder2ID
	})).Return(nil)
	
	// Index documents
	require.NoError(s.T(), s.indexTestDocument(doc1ID, []byte("This document is in folder 1"), s.testTenantID))
	require.NoError(s.T(), s.indexTestDocument(doc2ID, []byte("This document is in folder 2"), s.testTenantID))
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up search query
	searchQuery := "document"
	
	// Set up mock expectations for folder search
	expectedDocs := []models.Document{
		{
			ID:          doc1ID,
			Name:        "folder1-doc.pdf",
			ContentType: "application/pdf",
			FolderID:    folder1ID,
			Size:        28,
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult := utils.NewPaginatedResult(expectedDocs, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchInFolder", 
		mock.Anything, 
		folder1ID, 
		searchQuery, 
		s.testTenantID, 
		pagination,
	).Return(expectedResult, nil)
	
	// Call searchUseCase.SearchInFolder with folder ID and search query
	result, err := s.searchUseCase.SearchInFolder(ctx, folder1ID, searchQuery, s.testTenantID, pagination)
	
	// Assert that only documents in the specified folder are returned
	require.NoError(s.T(), err, "Folder search should not return an error")
	assert.Equal(s.T(), 1, len(result.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc1ID, result.Items[0].ID, "Search should return the correct document")
	assert.Equal(s.T(), folder1ID, result.Items[0].FolderID, "Document should be in the specified folder")
	
	// Verify that search results are properly paginated
	assert.Equal(s.T(), int64(1), result.Pagination.TotalItems, "Total items should be 1")
}

// TestSearchPagination tests pagination of search results
func (s *SearchFlowTestSuite) TestSearchPagination() {
	// Create test context
	ctx := context.Background()
	
	// Create and index a large number of test documents
	var docIDs []string
	for i := 0; i < 25; i++ {
		docID, err := s.createTestDocument(
			fmt.Sprintf("pagination-doc-%d.pdf", i),
			"application/pdf",
			[]byte(fmt.Sprintf("This is pagination test document %d", i)),
			nil,
			s.testTenantID,
		)
		require.NoError(s.T(), err, "Failed to create test document %d", i)
		docIDs = append(docIDs, docID)
		
		// Index the document
		require.NoError(
			s.T(),
			s.indexTestDocument(
				docID,
				[]byte(fmt.Sprintf("This is pagination test document %d", i)),
				s.testTenantID,
			),
		)
	}
	
	// Test first page
	pagination1 := utils.NewPagination(1, 10)
	expectedDocs1 := make([]models.Document, 10)
	for i := 0; i < 10; i++ {
		expectedDocs1[i] = models.Document{
			ID:          docIDs[i],
			Name:        fmt.Sprintf("pagination-doc-%d.pdf", i),
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		}
	}
	
	expectedResult1 := utils.NewPaginatedResult(expectedDocs1, pagination1, 25)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"pagination test", 
		s.testTenantID, 
		pagination1,
	).Return(expectedResult1, nil)
	
	// Call search methods with different pagination parameters
	result1, err := s.searchUseCase.SearchByContent(ctx, "pagination test", s.testTenantID, pagination1)
	
	// Verify that correct page of results is returned
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 10, len(result1.Items), "First page should return 10 documents")
	
	// Verify that pagination metadata is correct
	assert.Equal(s.T(), int64(25), result1.Pagination.TotalItems, "Total items should be 25")
	assert.Equal(s.T(), 3, result1.Pagination.TotalPages, "Total pages should be 3")
	assert.True(s.T(), result1.Pagination.HasNext, "First page should have next page")
	assert.False(s.T(), result1.Pagination.HasPrevious, "First page should not have previous page")
	
	// Test last page
	pagination3 := utils.NewPagination(3, 10)
	expectedDocs3 := make([]models.Document, 5)
	for i := 0; i < 5; i++ {
		expectedDocs3[i] = models.Document{
			ID:          docIDs[i+20],
			Name:        fmt.Sprintf("pagination-doc-%d.pdf", i+20),
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		}
	}
	
	expectedResult3 := utils.NewPaginatedResult(expectedDocs3, pagination3, 25)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"pagination test", 
		s.testTenantID, 
		pagination3,
	).Return(expectedResult3, nil)
	
	// Test edge cases like first page, last page, and invalid page parameters
	result3, err := s.searchUseCase.SearchByContent(ctx, "pagination test", s.testTenantID, pagination3)
	
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 5, len(result3.Items), "Last page should return 5 documents")
	assert.Equal(s.T(), 3, result3.Pagination.Page, "Current page should be 3")
	assert.Equal(s.T(), int64(25), result3.Pagination.TotalItems, "Total items should be 25")
	assert.Equal(s.T(), 3, result3.Pagination.TotalPages, "Total pages should be 3")
	assert.False(s.T(), result3.Pagination.HasNext, "Last page should not have next page")
	assert.True(s.T(), result3.Pagination.HasPrevious, "Last page should have previous page")
}

// TestEmptySearchResults tests behavior when search returns no results
func (s *SearchFlowTestSuite) TestEmptySearchResults() {
	// Create test context
	ctx := context.Background()
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock expectations for search with no matching results
	emptyResult := utils.NewPaginatedResult([]models.Document{}, pagination, 0)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"nonexistent", 
		s.testTenantID, 
		pagination,
	).Return(emptyResult, nil)
	
	// Call search methods with criteria that won't match any documents
	result, err := s.searchUseCase.SearchByContent(ctx, "nonexistent", s.testTenantID, pagination)
	
	// Verify that empty result set is returned with correct pagination metadata
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 0, len(result.Items), "Search should return 0 documents")
	assert.Equal(s.T(), int64(0), result.Pagination.TotalItems, "Total items should be 0")
	assert.Equal(s.T(), 0, result.Pagination.TotalPages, "Total pages should be 0")
	assert.False(s.T(), result.Pagination.HasNext, "No next page expected")
	assert.False(s.T(), result.Pagination.HasPrevious, "No previous page expected")
	
	// Verify that no error is returned for valid but non-matching search
	assert.NoError(s.T(), err, "Search with no results should not return an error")
}

// TestSearchValidationErrors tests validation errors in search parameters
func (s *SearchFlowTestSuite) TestSearchValidationErrors() {
	// Create test context
	ctx := context.Background()
	
	// Call searchUseCase.SearchByContent with empty query
	_, err := s.searchUseCase.SearchByContent(ctx, "", s.testTenantID, nil)
	assert.Error(s.T(), err, "Empty query should return an error")
	assert.True(s.T(), errors.IsValidationError(err), "Error should be a validation error")
	
	// Call searchUseCase.SearchByMetadata with empty metadata
	_, err = s.searchUseCase.SearchByMetadata(ctx, nil, s.testTenantID, nil)
	assert.Error(s.T(), err, "Empty metadata should return an error")
	assert.True(s.T(), errors.IsValidationError(err), "Error should be a validation error")
	
	// Call searchUseCase.CombinedSearch with empty query and metadata
	_, err = s.searchUseCase.CombinedSearch(ctx, "", nil, s.testTenantID, nil)
	assert.Error(s.T(), err, "Empty combined criteria should return an error")
	assert.True(s.T(), errors.IsValidationError(err), "Error should be a validation error")
	
	// Call search methods with empty tenant ID
	_, err = s.searchUseCase.SearchByContent(ctx, "test", "", nil)
	assert.Error(s.T(), err, "Empty tenant ID should return an error")
	assert.True(s.T(), errors.IsValidationError(err), "Error should be a validation error")
}

// TestTenantIsolation tests that search results are properly isolated between tenants
func (s *SearchFlowTestSuite) TestTenantIsolation() {
	// Create test context
	ctx := context.Background()
	
	// Create and index test documents for multiple tenants
	tenant1ID := s.testTenantID
	tenant2ID := uuid.New().String()
	
	doc1ID, err := s.createTestDocument(
		"tenant1-doc.pdf",
		"application/pdf",
		[]byte("This is a document for tenant 1"),
		nil,
		tenant1ID,
	)
	require.NoError(s.T(), err, "Failed to create test document for tenant 1")
	
	doc2ID, err := s.createTestDocument(
		"tenant2-doc.pdf",
		"application/pdf",
		[]byte("This is a document for tenant 2"),
		nil,
		tenant2ID,
	)
	require.NoError(s.T(), err, "Failed to create test document for tenant 2")
	
	// Index documents
	require.NoError(s.T(), s.indexTestDocument(doc1ID, []byte("This is a document for tenant 1"), tenant1ID))
	require.NoError(s.T(), s.indexTestDocument(doc2ID, []byte("This is a document for tenant 2"), tenant2ID))
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up search query
	searchQuery := "document"
	
	// Set up mock expectations for tenant 1 search
	expectedDocs1 := []models.Document{
		{
			ID:          doc1ID,
			Name:        "tenant1-doc.pdf",
			ContentType: "application/pdf",
			TenantID:    tenant1ID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult1 := utils.NewPaginatedResult(expectedDocs1, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		searchQuery, 
		tenant1ID, 
		pagination,
	).Return(expectedResult1, nil)
	
	// Call search methods with first tenant ID
	result1, err := s.searchUseCase.SearchByContent(ctx, searchQuery, tenant1ID, pagination)
	
	// Verify that only documents for first tenant are returned
	require.NoError(s.T(), err, "Search for tenant 1 should not return an error")
	assert.Equal(s.T(), 1, len(result1.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc1ID, result1.Items[0].ID, "Search should return the correct document")
	assert.Equal(s.T(), tenant1ID, result1.Items[0].TenantID, "Document should belong to tenant 1")
	
	// Set up mock expectations for tenant 2 search
	expectedDocs2 := []models.Document{
		{
			ID:          doc2ID,
			Name:        "tenant2-doc.pdf",
			ContentType: "application/pdf",
			TenantID:    tenant2ID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult2 := utils.NewPaginatedResult(expectedDocs2, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		searchQuery, 
		tenant2ID, 
		pagination,
	).Return(expectedResult2, nil)
	
	// Call search methods with second tenant ID
	result2, err := s.searchUseCase.SearchByContent(ctx, searchQuery, tenant2ID, pagination)
	
	// Verify that only documents for second tenant are returned
	require.NoError(s.T(), err, "Search for tenant 2 should not return an error")
	assert.Equal(s.T(), 1, len(result2.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc2ID, result2.Items[0].ID, "Search should return the correct document")
	assert.Equal(s.T(), tenant2ID, result2.Items[0].TenantID, "Document should belong to tenant 2")
	
	// Verify that no cross-tenant document access occurs
	assert.NotEqual(s.T(), result1.Items[0].ID, result2.Items[0].ID, "Documents from different tenants should be different")
}

// TestSearchPermissions tests that search respects document permissions
func (s *SearchFlowTestSuite) TestSearchPermissions() {
	// Create test context
	ctx := context.Background()
	
	// Create user IDs
	user1ID := s.testUserID
	user2ID := uuid.New().String()
	
	// Create and index test documents with different permission settings
	doc1ID, err := s.createTestDocument(
		"public-doc.pdf",
		"application/pdf",
		[]byte("This is a public document"),
		nil,
		s.testTenantID,
	)
	require.NoError(s.T(), err, "Failed to create public document")
	
	doc2ID, err := s.createTestDocument(
		"restricted-doc.pdf",
		"application/pdf",
		[]byte("This is a restricted document"),
		nil,
		s.testTenantID,
	)
	require.NoError(s.T(), err, "Failed to create restricted document")
	
	// Index documents
	require.NoError(s.T(), s.indexTestDocument(doc1ID, []byte("This is a public document"), s.testTenantID))
	require.NoError(s.T(), s.indexTestDocument(doc2ID, []byte("This is a restricted document"), s.testTenantID))
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up search query
	searchQuery := "document"
	
	// Set up context with user 1 (limited permissions)
	ctx1 := context.WithValue(ctx, "user_id", user1ID)
	
	// Set up mock expectations for search with user 1 having limited permissions
	expectedDocs1 := []models.Document{
		{
			ID:          doc1ID,
			Name:        "public-doc.pdf",
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult1 := utils.NewPaginatedResult(expectedDocs1, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.MatchedBy(func(ctx context.Context) bool {
			uid, ok := ctx.Value("user_id").(string)
			return ok && uid == user1ID
		}), 
		searchQuery, 
		s.testTenantID, 
		pagination,
	).Return(expectedResult1, nil)
	
	// Call search methods with user having limited permissions
	result1, err := s.searchUseCase.SearchByContent(ctx1, searchQuery, s.testTenantID, pagination)
	
	// Verify that only documents the user has access to are returned
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 1, len(result1.Items), "Search should return 1 document")
	assert.Equal(s.T(), doc1ID, result1.Items[0].ID, "Search should return the public document")
	
	// Set up context with user 2 (broader permissions)
	ctx2 := context.WithValue(ctx, "user_id", user2ID)
	
	// Set up mock expectations for search with user 2 having broader permissions
	expectedDocs2 := []models.Document{
		{
			ID:          doc1ID,
			Name:        "public-doc.pdf",
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
		{
			ID:          doc2ID,
			Name:        "restricted-doc.pdf",
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult2 := utils.NewPaginatedResult(expectedDocs2, pagination, 2)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.MatchedBy(func(ctx context.Context) bool {
			uid, ok := ctx.Value("user_id").(string)
			return ok && uid == user2ID
		}), 
		searchQuery, 
		s.testTenantID, 
		pagination,
	).Return(expectedResult2, nil)
	
	// Call search methods with user having broader permissions
	result2, err := s.searchUseCase.SearchByContent(ctx2, searchQuery, s.testTenantID, pagination)
	
	// Verify that more documents are returned based on permissions
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 2, len(result2.Items), "Search should return 2 documents")
	assert.Contains(s.T(), []string{doc1ID, doc2ID}, result2.Items[0].ID, "Search results should contain both documents")
	assert.Contains(s.T(), []string{doc1ID, doc2ID}, result2.Items[1].ID, "Search results should contain both documents")
	assert.NotEqual(s.T(), result2.Items[0].ID, result2.Items[1].ID, "Search results should return different documents")
}

// TestDocumentIndexing tests document indexing functionality
func (s *SearchFlowTestSuite) TestDocumentIndexing() {
	// Create test context
	ctx := context.Background()
	
	// Create test document content
	docID := uuid.New().String()
	docContent := []byte("This is a test document for indexing")
	
	// Set up mock expectations for document indexing
	s.searchService.(*mockSearchService).On(
		"IndexDocument",
		mock.Anything,
		docID,
		s.testTenantID,
		docContent,
	).Return(nil)
	
	// Call searchUseCase.IndexDocument
	err := s.searchUseCase.IndexDocument(ctx, docID, s.testTenantID, docContent)
	
	// Verify that document is properly indexed
	require.NoError(s.T(), err, "Document indexing should not return an error")
	
	// Set up mock expectations for search
	expectedDocs := []models.Document{
		{
			ID:          docID,
			Name:        "test-doc.pdf",
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	pagination := utils.NewPagination(1, 10)
	expectedResult := utils.NewPaginatedResult(expectedDocs, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"test document", 
		s.testTenantID, 
		pagination,
	).Return(expectedResult, nil)
	
	// Search for the indexed document
	result, err := s.searchUseCase.SearchByContent(ctx, "test document", s.testTenantID, pagination)
	
	// Verify that document appears in search results
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 1, len(result.Items), "Search should return 1 document")
	assert.Equal(s.T(), docID, result.Items[0].ID, "Search should return the indexed document")
}

// TestDocumentRemovalFromIndex tests removing documents from the search index
func (s *SearchFlowTestSuite) TestDocumentRemovalFromIndex() {
	// Create test context
	ctx := context.Background()
	
	// Create and index test document
	docID, err := s.createTestDocument(
		"removal-test.pdf",
		"application/pdf",
		[]byte("This document will be removed from the index"),
		nil,
		s.testTenantID,
	)
	require.NoError(s.T(), err, "Failed to create test document")
	
	require.NoError(
		s.T(),
		s.indexTestDocument(
			docID,
			[]byte("This document will be removed from the index"),
			s.testTenantID,
		),
	)
	
	// Set up pagination
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock expectations for document search (before removal)
	expectedDocs := []models.Document{
		{
			ID:          docID,
			Name:        "removal-test.pdf",
			ContentType: "application/pdf",
			TenantID:    s.testTenantID,
			Status:      models.DocumentStatusAvailable,
		},
	}
	
	expectedResult := utils.NewPaginatedResult(expectedDocs, pagination, 1)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"removed", 
		s.testTenantID, 
		pagination,
	).Return(expectedResult, nil).Once()
	
	// Verify document appears in search results
	result, err := s.searchUseCase.SearchByContent(ctx, "removed", s.testTenantID, pagination)
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 1, len(result.Items), "Search should return 1 document before removal")
	
	// Set up mock expectations for document removal from index
	s.searchService.(*mockSearchService).On(
		"RemoveDocumentFromIndex",
		mock.Anything,
		docID,
		s.testTenantID,
	).Return(nil)
	
	// Call searchUseCase.RemoveDocumentFromIndex
	err = s.searchUseCase.RemoveDocumentFromIndex(ctx, docID, s.testTenantID)
	
	// Verify that document is properly removed
	require.NoError(s.T(), err, "Document removal should not return an error")
	
	// Set up mock expectations for document search (after removal)
	emptyResult := utils.NewPaginatedResult([]models.Document{}, pagination, 0)
	
	s.searchService.(*mockSearchService).On(
		"SearchByContent", 
		mock.Anything, 
		"removed", 
		s.testTenantID, 
		pagination,
	).Return(emptyResult, nil).Once()
	
	// Search for the removed document
	result, err = s.searchUseCase.SearchByContent(ctx, "removed", s.testTenantID, pagination)
	
	// Verify that document no longer appears in search results
	require.NoError(s.T(), err, "Search should not return an error")
	assert.Equal(s.T(), 0, len(result.Items), "Search should return 0 documents after removal")
}

// Helper function to create a test document
func (s *SearchFlowTestSuite) createTestDocument(name, contentType string, content []byte, metadata map[string]string, tenantID string) (string, error) {
	// Create a new document model with the provided parameters
	doc := models.Document{
		Name:        name,
		ContentType: contentType,
		Size:        int64(len(content)),
		FolderID:    s.testFolderID,
		TenantID:    tenantID,
		OwnerID:     s.testUserID,
		Status:      models.DocumentStatusAvailable,
	}
	
	// Set up mock expectations for document creation in repository
	docID := uuid.New().String()
	s.documentRepo.(*mockDocumentRepository).On(
		"Create",
		mock.Anything,
		mock.MatchedBy(func(d *models.Document) bool {
			return d.Name == name && d.TenantID == tenantID
		}),
	).Return(docID, nil)
	
	// Store document ID in testDocumentIDs map if successful
	s.testDocumentIDs[name] = docID
	
	// Add metadata if provided
	if metadata != nil {
		for key, value := range metadata {
			// Set up mock expectations for document metadata
			metadataID := uuid.New().String()
			s.documentRepo.(*mockDocumentRepository).On(
				"AddMetadata",
				mock.Anything,
				docID,
				key,
				value,
				tenantID,
			).Return(metadataID, nil)
		}
	}
	
	return docID, nil
}

// Helper function to index a test document
func (s *SearchFlowTestSuite) indexTestDocument(documentID string, content []byte, tenantID string) error {
	// Set up mock expectations for document indexing
	s.searchService.(*mockSearchService).On(
		"IndexDocument",
		mock.Anything,
		documentID,
		tenantID,
		content,
	).Return(nil)
	
	// Call searchUseCase.IndexDocument with the provided parameters
	return s.searchUseCase.IndexDocument(context.Background(), documentID, tenantID, content)
}

// Helper function to load test files from testdata directory
func (s *SearchFlowTestSuite) loadTestFile(filename string) ([]byte, error) {
	// Construct path to test file in testdata directory
	filePath := filepath.Join("testdata", filename)
	
	// Read file content
	return ioutil.ReadFile(filePath)
}

// Mock implementations used for testing

type mockDocumentRepository struct {
	mock.Mock
}

func (m *mockDocumentRepository) Create(ctx context.Context, document *models.Document) (string, error) {
	args := m.Called(ctx, document)
	return args.String(0), args.Error(1)
}

func (m *mockDocumentRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.Document, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Document), args.Error(1)
}

func (m *mockDocumentRepository) Update(ctx context.Context, document *models.Document) error {
	args := m.Called(ctx, document)
	return args.Error(0)
}

func (m *mockDocumentRepository) Delete(ctx context.Context, id string, tenantID string) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *mockDocumentRepository) ListByFolder(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, folderID, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockDocumentRepository) ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockDocumentRepository) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, query, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockDocumentRepository) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, metadata, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockDocumentRepository) AddVersion(ctx context.Context, version *models.DocumentVersion) (string, error) {
	args := m.Called(ctx, version)
	return args.String(0), args.Error(1)
}

func (m *mockDocumentRepository) GetVersionByID(ctx context.Context, versionID string, tenantID string) (*models.DocumentVersion, error) {
	args := m.Called(ctx, versionID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DocumentVersion), args.Error(1)
}

func (m *mockDocumentRepository) UpdateVersionStatus(ctx context.Context, versionID string, status string, tenantID string) error {
	args := m.Called(ctx, versionID, status, tenantID)
	return args.Error(0)
}

func (m *mockDocumentRepository) AddMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) (string, error) {
	args := m.Called(ctx, documentID, key, value, tenantID)
	return args.String(0), args.Error(1)
}

func (m *mockDocumentRepository) UpdateMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) error {
	args := m.Called(ctx, documentID, key, value, tenantID)
	return args.Error(0)
}

func (m *mockDocumentRepository) DeleteMetadata(ctx context.Context, documentID string, key string, tenantID string) error {
	args := m.Called(ctx, documentID, key, tenantID)
	return args.Error(0)
}

func (m *mockDocumentRepository) GetDocumentsByIDs(ctx context.Context, ids []string, tenantID string) ([]*models.Document, error) {
	args := m.Called(ctx, ids, tenantID)
	return args.Get(0).([]*models.Document), args.Error(1)
}

type mockStorageService struct {
	mock.Mock
}

type mockDocumentService struct {
	mock.Mock
}

type mockSearchService struct {
	mock.Mock
}

func (m *mockSearchService) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, query, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockSearchService) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, metadata, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockSearchService) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, contentQuery, metadata, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockSearchService) SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, folderID, query, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *mockSearchService) IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error {
	args := m.Called(ctx, documentID, tenantID, content)
	return args.Error(0)
}

func (m *mockSearchService) RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error {
	args := m.Called(ctx, documentID, tenantID)
	return args.Error(0)
}