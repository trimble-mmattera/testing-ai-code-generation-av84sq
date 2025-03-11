package elasticsearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock"   // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+

	"../../../domain/models"
	"../../../domain/services"
	"../../../pkg/config"
	"../../../pkg/utils"
)

// Constants for testing
const testTenantID = "tenant-123"
const testDocumentID = "doc-123"
const testFolderID = "folder-123"
var testContent = []byte("This is a test document content for search testing.")

// MockDocumentIndex is a mock implementation of DocumentIndex for testing
type MockDocumentIndex struct {
	mock.Mock
}

// GetTenantIndex mock implementation of GetTenantIndex
func (m *MockDocumentIndex) GetTenantIndex(tenantID string) string {
	return m.Called(tenantID).String(0)
}

// EnsureTenantIndex mock implementation of EnsureTenantIndex
func (m *MockDocumentIndex) EnsureTenantIndex(ctx context.Context, tenantID string) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

// IndexDocument mock implementation of IndexDocument
func (m *MockDocumentIndex) IndexDocument(ctx context.Context, document *models.Document, content []byte) error {
	return m.Called(ctx, document, content).Error(0)
}

// RemoveDocument mock implementation of RemoveDocument
func (m *MockDocumentIndex) RemoveDocument(ctx context.Context, documentID string, tenantID string) error {
	return m.Called(ctx, documentID, tenantID).Error(0)
}

// MockElasticsearchClient is a mock implementation of ElasticsearchClient for testing
type MockElasticsearchClient struct {
	mock.Mock
}

// Search mock implementation of Search
func (m *MockElasticsearchClient) Search(ctx context.Context, index string, query map[string]interface{}, from, size int) (map[string]interface{}, error) {
	args := m.Called(ctx, index, query, from, size)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// BuildContentQuery mock implementation of BuildContentQuery
func (m *MockElasticsearchClient) BuildContentQuery(query string) map[string]interface{} {
	return m.Called(query).Get(0).(map[string]interface{})
}

// BuildMetadataQuery mock implementation of BuildMetadataQuery
func (m *MockElasticsearchClient) BuildMetadataQuery(metadata map[string]string) map[string]interface{} {
	return m.Called(metadata).Get(0).(map[string]interface{})
}

// BuildCombinedQuery mock implementation of BuildCombinedQuery
func (m *MockElasticsearchClient) BuildCombinedQuery(contentQuery string, metadata map[string]string) map[string]interface{} {
	return m.Called(contentQuery, metadata).Get(0).(map[string]interface{})
}

// BuildFolderQuery mock implementation of BuildFolderQuery
func (m *MockElasticsearchClient) BuildFolderQuery(folderID string, query string) map[string]interface{} {
	return m.Called(folderID, query).Get(0).(map[string]interface{})
}

// TestNewElasticsearchIndexer tests the creation of a new ElasticsearchIndexer instance
func TestNewElasticsearchIndexer(t *testing.T) {
	// Create a mock DocumentIndex
	mockIndex := new(MockDocumentIndex)
	
	// Call NewElasticsearchIndexer with the mock
	indexer, err := NewElasticsearchIndexer(mockIndex)
	
	// Assert that the returned indexer is not nil
	assert.NotNil(t, indexer)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Test with nil DocumentIndex and assert error is returned
	indexer, err = NewElasticsearchIndexer(nil)
	assert.Nil(t, indexer)
	assert.Error(t, err)
}

// TestNewElasticsearchQueryExecutor tests the creation of a new ElasticsearchQueryExecutor instance
func TestNewElasticsearchQueryExecutor(t *testing.T) {
	// Create a mock ElasticsearchClient
	mockClient := new(MockElasticsearchClient)
	
	// Call NewElasticsearchQueryExecutor with the mock
	executor, err := NewElasticsearchQueryExecutor(mockClient)
	
	// Assert that the returned executor is not nil
	assert.NotNil(t, executor)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Test with nil ElasticsearchClient and assert error is returned
	executor, err = NewElasticsearchQueryExecutor(nil)
	assert.Nil(t, executor)
	assert.Error(t, err)
}

// TestElasticsearchIndexer_IndexDocument tests the IndexDocument method of elasticsearchIndexer
func TestElasticsearchIndexer_IndexDocument(t *testing.T) {
	// Create a mock DocumentIndex
	mockIndex := new(MockDocumentIndex)
	
	// Set up the mock to expect IndexDocument call with appropriate parameters
	mockIndex.On("IndexDocument", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	// Create an elasticsearchIndexer with the mock
	indexer, err := NewElasticsearchIndexer(mockIndex)
	require.NoError(t, err)
	
	// Create a test document and content
	doc := createTestDocument()
	
	// Call IndexDocument on the indexer
	err = indexer.IndexDocument(context.Background(), doc, testContent)
	
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Verify that the mock expectations were met
	mockIndex.AssertExpectations(t)
	
	// Test error cases: nil document
	err = indexer.IndexDocument(context.Background(), nil, testContent)
	assert.Error(t, err)
	
	// Test error cases: empty content
	err = indexer.IndexDocument(context.Background(), doc, nil)
	assert.Error(t, err)
	
	// Test error cases: DocumentIndex error
	mockErrorIndex := new(MockDocumentIndex)
	mockErrorIndex.On("IndexDocument", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
	errorIndexer, _ := NewElasticsearchIndexer(mockErrorIndex)
	
	err = errorIndexer.IndexDocument(context.Background(), doc, testContent)
	assert.Error(t, err)
}

// TestElasticsearchIndexer_RemoveDocument tests the RemoveDocument method of elasticsearchIndexer
func TestElasticsearchIndexer_RemoveDocument(t *testing.T) {
	// Create a mock DocumentIndex
	mockIndex := new(MockDocumentIndex)
	
	// Set up the mock to expect RemoveDocument call with appropriate parameters
	mockIndex.On("RemoveDocument", mock.Anything, testDocumentID, testTenantID).Return(nil)
	
	// Create an elasticsearchIndexer with the mock
	indexer, err := NewElasticsearchIndexer(mockIndex)
	require.NoError(t, err)
	
	// Call RemoveDocument on the indexer with test document ID and tenant ID
	err = indexer.RemoveDocument(context.Background(), testDocumentID, testTenantID)
	
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Verify that the mock expectations were met
	mockIndex.AssertExpectations(t)
	
	// Test error cases: empty document ID
	err = indexer.RemoveDocument(context.Background(), "", testTenantID)
	assert.Error(t, err)
	
	// Test error cases: empty tenant ID
	err = indexer.RemoveDocument(context.Background(), testDocumentID, "")
	assert.Error(t, err)
	
	// Test error cases: DocumentIndex error
	mockErrorIndex := new(MockDocumentIndex)
	mockErrorIndex.On("RemoveDocument", mock.Anything, testDocumentID, testTenantID).Return(assert.AnError)
	errorIndexer, _ := NewElasticsearchIndexer(mockErrorIndex)
	
	err = errorIndexer.RemoveDocument(context.Background(), testDocumentID, testTenantID)
	assert.Error(t, err)
}

// TestElasticsearchQueryExecutor_ExecuteContentSearch tests the ExecuteContentSearch method of elasticsearchQueryExecutor
func TestElasticsearchQueryExecutor_ExecuteContentSearch(t *testing.T) {
	// Create a mock ElasticsearchClient
	mockClient := new(MockElasticsearchClient)
	
	// Create test query
	query := "test query"
	expectedDocIDs := []string{testDocumentID, "doc-456"}
	expectedTotal := int64(2)
	mockResponse := createMockSearchResponse(expectedDocIDs, expectedTotal)
	
	// Set up the mock to expect BuildContentQuery and Search calls with appropriate parameters
	mockClient.On("BuildContentQuery", query).Return(map[string]interface{}{"query": "test"})
	mockClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(mockResponse, nil)
	
	// Create an elasticsearchQueryExecutor with the mock
	executor, err := NewElasticsearchQueryExecutor(mockClient)
	require.NoError(t, err)
	
	// Call ExecuteContentSearch on the executor with test query and tenant ID
	docIDs, total, err := executor.ExecuteContentSearch(context.Background(), query, testTenantID, utils.NewPagination(1, 20))
	
	// Assert that the returned document IDs match expected values
	assert.Equal(t, expectedDocIDs, docIDs)
	// Assert that the returned total count matches expected value
	assert.Equal(t, expectedTotal, total)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Verify that the mock expectations were met
	mockClient.AssertExpectations(t)
	
	// Test error cases: empty query
	docIDs, total, err = executor.ExecuteContentSearch(context.Background(), "", testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: empty tenant ID
	docIDs, total, err = executor.ExecuteContentSearch(context.Background(), query, "", utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: search error
	mockErrorClient := new(MockElasticsearchClient)
	mockErrorClient.On("BuildContentQuery", query).Return(map[string]interface{}{"query": "test"})
	mockErrorClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(map[string]interface{}{}, assert.AnError)
	errorExecutor, _ := NewElasticsearchQueryExecutor(mockErrorClient)
	
	docIDs, total, err = errorExecutor.ExecuteContentSearch(context.Background(), query, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
}

// TestElasticsearchQueryExecutor_ExecuteMetadataSearch tests the ExecuteMetadataSearch method of elasticsearchQueryExecutor
func TestElasticsearchQueryExecutor_ExecuteMetadataSearch(t *testing.T) {
	// Create a mock ElasticsearchClient
	mockClient := new(MockElasticsearchClient)
	
	// Set up the mock to expect BuildMetadataQuery and Search calls with appropriate parameters
	metadata := map[string]string{"key": "value"}
	expectedDocIDs := []string{testDocumentID, "doc-456"}
	expectedTotal := int64(2)
	mockResponse := createMockSearchResponse(expectedDocIDs, expectedTotal)
	
	mockClient.On("BuildMetadataQuery", metadata).Return(map[string]interface{}{"query": "test"})
	mockClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(mockResponse, nil)
	
	// Create an elasticsearchQueryExecutor with the mock
	executor, err := NewElasticsearchQueryExecutor(mockClient)
	require.NoError(t, err)
	
	// Call ExecuteMetadataSearch on the executor with test metadata and tenant ID
	docIDs, total, err := executor.ExecuteMetadataSearch(context.Background(), metadata, testTenantID, utils.NewPagination(1, 20))
	
	// Assert that the returned document IDs match expected values
	assert.Equal(t, expectedDocIDs, docIDs)
	// Assert that the returned total count matches expected value
	assert.Equal(t, expectedTotal, total)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Verify that the mock expectations were met
	mockClient.AssertExpectations(t)
	
	// Test error cases: empty metadata
	docIDs, total, err = executor.ExecuteMetadataSearch(context.Background(), nil, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: empty tenant ID
	docIDs, total, err = executor.ExecuteMetadataSearch(context.Background(), metadata, "", utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: search error
	mockErrorClient := new(MockElasticsearchClient)
	mockErrorClient.On("BuildMetadataQuery", metadata).Return(map[string]interface{}{"query": "test"})
	mockErrorClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(map[string]interface{}{}, assert.AnError)
	errorExecutor, _ := NewElasticsearchQueryExecutor(mockErrorClient)
	
	docIDs, total, err = errorExecutor.ExecuteMetadataSearch(context.Background(), metadata, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
}

// TestElasticsearchQueryExecutor_ExecuteCombinedSearch tests the ExecuteCombinedSearch method of elasticsearchQueryExecutor
func TestElasticsearchQueryExecutor_ExecuteCombinedSearch(t *testing.T) {
	// Create a mock ElasticsearchClient
	mockClient := new(MockElasticsearchClient)
	
	// Set up the mock to expect BuildCombinedQuery and Search calls with appropriate parameters
	query := "test query"
	metadata := map[string]string{"key": "value"}
	expectedDocIDs := []string{testDocumentID, "doc-456"}
	expectedTotal := int64(2)
	mockResponse := createMockSearchResponse(expectedDocIDs, expectedTotal)
	
	mockClient.On("BuildCombinedQuery", query, metadata).Return(map[string]interface{}{"query": "test"})
	mockClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(mockResponse, nil)
	
	// Create an elasticsearchQueryExecutor with the mock
	executor, err := NewElasticsearchQueryExecutor(mockClient)
	require.NoError(t, err)
	
	// Call ExecuteCombinedSearch on the executor with test query, metadata, and tenant ID
	docIDs, total, err := executor.ExecuteCombinedSearch(context.Background(), query, metadata, testTenantID, utils.NewPagination(1, 20))
	
	// Assert that the returned document IDs match expected values
	assert.Equal(t, expectedDocIDs, docIDs)
	// Assert that the returned total count matches expected value
	assert.Equal(t, expectedTotal, total)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Verify that the mock expectations were met
	mockClient.AssertExpectations(t)
	
	// Test error cases: empty query and metadata
	docIDs, total, err = executor.ExecuteCombinedSearch(context.Background(), "", nil, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: empty tenant ID
	docIDs, total, err = executor.ExecuteCombinedSearch(context.Background(), query, metadata, "", utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: search error
	mockErrorClient := new(MockElasticsearchClient)
	mockErrorClient.On("BuildCombinedQuery", query, metadata).Return(map[string]interface{}{"query": "test"})
	mockErrorClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(map[string]interface{}{}, assert.AnError)
	errorExecutor, _ := NewElasticsearchQueryExecutor(mockErrorClient)
	
	docIDs, total, err = errorExecutor.ExecuteCombinedSearch(context.Background(), query, metadata, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
}

// TestElasticsearchQueryExecutor_ExecuteFolderSearch tests the ExecuteFolderSearch method of elasticsearchQueryExecutor
func TestElasticsearchQueryExecutor_ExecuteFolderSearch(t *testing.T) {
	// Create a mock ElasticsearchClient
	mockClient := new(MockElasticsearchClient)
	
	// Set up the mock to expect BuildFolderQuery and Search calls with appropriate parameters
	query := "test query"
	expectedDocIDs := []string{testDocumentID, "doc-456"}
	expectedTotal := int64(2)
	mockResponse := createMockSearchResponse(expectedDocIDs, expectedTotal)
	
	mockClient.On("BuildFolderQuery", testFolderID, query).Return(map[string]interface{}{"query": "test"})
	mockClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(mockResponse, nil)
	
	// Create an elasticsearchQueryExecutor with the mock
	executor, err := NewElasticsearchQueryExecutor(mockClient)
	require.NoError(t, err)
	
	// Call ExecuteFolderSearch on the executor with test folder ID, query, and tenant ID
	docIDs, total, err := executor.ExecuteFolderSearch(context.Background(), testFolderID, query, testTenantID, utils.NewPagination(1, 20))
	
	// Assert that the returned document IDs match expected values
	assert.Equal(t, expectedDocIDs, docIDs)
	// Assert that the returned total count matches expected value
	assert.Equal(t, expectedTotal, total)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Verify that the mock expectations were met
	mockClient.AssertExpectations(t)
	
	// Test error cases: empty folder ID
	docIDs, total, err = executor.ExecuteFolderSearch(context.Background(), "", query, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: empty query
	docIDs, total, err = executor.ExecuteFolderSearch(context.Background(), testFolderID, "", testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: empty tenant ID
	docIDs, total, err = executor.ExecuteFolderSearch(context.Background(), testFolderID, query, "", utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: search error
	mockErrorClient := new(MockElasticsearchClient)
	mockErrorClient.On("BuildFolderQuery", testFolderID, query).Return(map[string]interface{}{"query": "test"})
	mockErrorClient.On("Search", mock.Anything, testTenantID+"-documents", mock.Anything, 0, 20).Return(map[string]interface{}{}, assert.AnError)
	errorExecutor, _ := NewElasticsearchQueryExecutor(mockErrorClient)
	
	docIDs, total, err = errorExecutor.ExecuteFolderSearch(context.Background(), testFolderID, query, testTenantID, utils.NewPagination(1, 20))
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
}

// TestElasticsearchQueryExecutor_extractDocumentIDs tests the extractDocumentIDs method of elasticsearchQueryExecutor
func TestElasticsearchQueryExecutor_extractDocumentIDs(t *testing.T) {
	// Create a sample Elasticsearch search response with document IDs
	expectedDocIDs := []string{testDocumentID, "doc-456", "doc-789"}
	expectedTotal := int64(3)
	searchResponse := createMockSearchResponse(expectedDocIDs, expectedTotal)
	
	// Create an elasticsearchQueryExecutor
	mockClient := new(MockElasticsearchClient)
	executor, err := NewElasticsearchQueryExecutor(mockClient)
	require.NoError(t, err)
	
	// Call extractDocumentIDs on the executor with the sample response
	docIDs, total, err := executor.extractDocumentIDs(searchResponse)
	
	// Assert that the returned document IDs match expected values
	assert.Equal(t, expectedDocIDs, docIDs)
	// Assert that the returned total count matches expected value
	assert.Equal(t, expectedTotal, total)
	// Assert that no error is returned
	assert.NoError(t, err)
	
	// Test error cases: malformed response
	malformedResponse := map[string]interface{}{"not_hits": "something"}
	docIDs, total, err = executor.extractDocumentIDs(malformedResponse)
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
	
	// Test error cases: missing total
	missingTotalResponse := map[string]interface{}{
		"hits": map[string]interface{}{
			"hits": []map[string]interface{}{
				{"_id": testDocumentID},
			},
		},
	}
	docIDs, total, err = executor.extractDocumentIDs(missingTotalResponse)
	assert.Error(t, err)
	assert.Empty(t, docIDs)
	assert.Zero(t, total)
}

// Helper function to create a test document for use in tests
func createTestDocument() *models.Document {
	return &models.Document{
		ID:          testDocumentID,
		Name:        "test-document.pdf",
		ContentType: "application/pdf",
		Size:        1024,
		FolderID:    testFolderID,
		TenantID:    testTenantID,
		OwnerID:     "user-123",
		Status:      models.DocumentStatusAvailable,
	}
}

// Helper function to create a mock Elasticsearch search response
func createMockSearchResponse(documentIDs []string, total int64) map[string]interface{} {
	hits := make([]map[string]interface{}, len(documentIDs))
	for i, id := range documentIDs {
		hits[i] = map[string]interface{}{
			"_id": id,
		}
	}
	
	return map[string]interface{}{
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value": total,
			},
			"hits": hits,
		},
	}
}