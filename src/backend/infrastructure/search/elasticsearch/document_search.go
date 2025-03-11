// Package elasticsearch provides Elasticsearch implementations for the search interfaces
// of the Document Management Platform.
package elasticsearch

import (
	"context" // standard library
	"fmt"     // standard library
	"strings" // standard library

	"../../../domain/models"
	"../../../domain/services"
	"../../../pkg/errors"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

// NewElasticsearchIndexer creates a new ElasticsearchIndexer instance that implements the SearchIndexer interface
func NewElasticsearchIndexer(documentIndex *DocumentIndex) (services.SearchIndexer, error) {
	if documentIndex == nil {
		return nil, fmt.Errorf("documentIndex cannot be nil")
	}

	return &elasticsearchIndexer{
		documentIndex: documentIndex,
		logger:        logger.WithField("component", "elasticsearch_indexer"),
	}, nil
}

// NewElasticsearchQueryExecutor creates a new ElasticsearchQueryExecutor instance that implements the SearchQueryExecutor interface
func NewElasticsearchQueryExecutor(client *ElasticsearchClient) (services.SearchQueryExecutor, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	return &elasticsearchQueryExecutor{
		client: client,
		logger: logger.WithField("component", "elasticsearch_query_executor"),
	}, nil
}

// elasticsearchIndexer implements the SearchIndexer interface using Elasticsearch
type elasticsearchIndexer struct {
	documentIndex *DocumentIndex
	logger        logger.Logger
}

// IndexDocument indexes a document for search in Elasticsearch
func (e *elasticsearchIndexer) IndexDocument(ctx context.Context, document *models.Document, content []byte) error {
	e.logger.InfoContext(ctx, "Indexing document", 
		"documentID", document.ID,
		"documentName", document.Name,
		"tenantID", document.TenantID)

	// Validate document and content
	if document == nil {
		return errors.NewValidationError("document cannot be nil")
	}
	if content == nil || len(content) == 0 {
		return errors.NewValidationError("document content cannot be empty")
	}
	if document.TenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Call documentIndex to index the document with content
	err := e.documentIndex.IndexDocument(ctx, document, content)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to index document",
			"error", err,
			"documentID", document.ID,
			"tenantID", document.TenantID)
		return errors.NewDependencyError(fmt.Sprintf("failed to index document: %v", err))
	}

	e.logger.InfoContext(ctx, "Document indexed successfully",
		"documentID", document.ID,
		"tenantID", document.TenantID)
	return nil
}

// RemoveDocument removes a document from the Elasticsearch index
func (e *elasticsearchIndexer) RemoveDocument(ctx context.Context, documentID string, tenantID string) error {
	e.logger.InfoContext(ctx, "Removing document from index",
		"documentID", documentID,
		"tenantID", tenantID)

	// Validate document ID and tenant ID
	if documentID == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Call documentIndex to remove the document from the index
	err := e.documentIndex.RemoveDocument(ctx, documentID, tenantID)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to remove document from index",
			"error", err,
			"documentID", documentID,
			"tenantID", tenantID)
		return errors.NewDependencyError(fmt.Sprintf("failed to remove document from index: %v", err))
	}

	e.logger.InfoContext(ctx, "Document removed from index successfully",
		"documentID", documentID,
		"tenantID", tenantID)
	return nil
}

// elasticsearchQueryExecutor implements the SearchQueryExecutor interface using Elasticsearch
type elasticsearchQueryExecutor struct {
	client *ElasticsearchClient
	logger logger.Logger
}

// ExecuteContentSearch executes a content-based search query in Elasticsearch
func (e *elasticsearchQueryExecutor) ExecuteContentSearch(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) ([]string, int64, error) {
	e.logger.InfoContext(ctx, "Executing content search",
		"query", query,
		"tenantID", tenantID)

	// Validate query and tenant ID
	if strings.TrimSpace(query) == "" {
		return nil, 0, errors.NewValidationError("search query cannot be empty")
	}
	if tenantID == "" {
		return nil, 0, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get tenant-specific index name
	indexName := fmt.Sprintf("documents-%s", tenantID)

	// Build content search query
	searchQuery := e.client.BuildContentQuery(query)

	// Apply pagination parameters
	from := 0
	size := 10
	if pagination != nil {
		from = pagination.GetOffset()
		size = pagination.GetLimit()
	} else {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Execute search against Elasticsearch
	searchResults, err := e.client.Search(ctx, indexName, searchQuery, from, size)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to execute content search",
			"error", err,
			"query", query,
			"tenantID", tenantID)
		return nil, 0, errors.NewDependencyError(fmt.Sprintf("failed to execute content search: %v", err))
	}

	// Extract document IDs and total count from search results
	documentIDs, totalCount, err := e.extractDocumentIDs(searchResults)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to extract document IDs from search results", "error", err)
		return nil, 0, err
	}

	e.logger.InfoContext(ctx, "Content search executed successfully",
		"query", query,
		"tenantID", tenantID,
		"resultCount", len(documentIDs),
		"totalCount", totalCount)

	return documentIDs, totalCount, nil
}

// ExecuteMetadataSearch executes a metadata-based search query in Elasticsearch
func (e *elasticsearchQueryExecutor) ExecuteMetadataSearch(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) ([]string, int64, error) {
	e.logger.InfoContext(ctx, "Executing metadata search",
		"metadata", metadata,
		"tenantID", tenantID)

	// Validate metadata and tenant ID
	if metadata == nil || len(metadata) == 0 {
		return nil, 0, errors.NewValidationError("metadata search criteria cannot be empty")
	}
	if tenantID == "" {
		return nil, 0, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get tenant-specific index name
	indexName := fmt.Sprintf("documents-%s", tenantID)

	// Build metadata search query
	searchQuery := e.client.BuildMetadataQuery(metadata)

	// Apply pagination parameters
	from := 0
	size := 10
	if pagination != nil {
		from = pagination.GetOffset()
		size = pagination.GetLimit()
	} else {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Execute search against Elasticsearch
	searchResults, err := e.client.Search(ctx, indexName, searchQuery, from, size)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to execute metadata search",
			"error", err,
			"metadata", metadata,
			"tenantID", tenantID)
		return nil, 0, errors.NewDependencyError(fmt.Sprintf("failed to execute metadata search: %v", err))
	}

	// Extract document IDs and total count from search results
	documentIDs, totalCount, err := e.extractDocumentIDs(searchResults)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to extract document IDs from search results", "error", err)
		return nil, 0, err
	}

	e.logger.InfoContext(ctx, "Metadata search executed successfully",
		"metadata", metadata,
		"tenantID", tenantID,
		"resultCount", len(documentIDs),
		"totalCount", totalCount)

	return documentIDs, totalCount, nil
}

// ExecuteCombinedSearch executes a combined content and metadata search query in Elasticsearch
func (e *elasticsearchQueryExecutor) ExecuteCombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) ([]string, int64, error) {
	e.logger.InfoContext(ctx, "Executing combined search",
		"contentQuery", contentQuery,
		"metadata", metadata,
		"tenantID", tenantID)

	// Validate that at least one of contentQuery or metadata is provided
	contentQueryEmpty := strings.TrimSpace(contentQuery) == ""
	metadataEmpty := metadata == nil || len(metadata) == 0
	
	if contentQueryEmpty && metadataEmpty {
		return nil, 0, errors.NewValidationError("at least one search criteria (content or metadata) must be provided")
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return nil, 0, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get tenant-specific index name
	indexName := fmt.Sprintf("documents-%s", tenantID)

	// Build combined search query
	searchQuery := e.client.BuildCombinedQuery(contentQuery, metadata)

	// Apply pagination parameters
	from := 0
	size := 10
	if pagination != nil {
		from = pagination.GetOffset()
		size = pagination.GetLimit()
	} else {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Execute search against Elasticsearch
	searchResults, err := e.client.Search(ctx, indexName, searchQuery, from, size)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to execute combined search",
			"error", err,
			"contentQuery", contentQuery,
			"metadata", metadata,
			"tenantID", tenantID)
		return nil, 0, errors.NewDependencyError(fmt.Sprintf("failed to execute combined search: %v", err))
	}

	// Extract document IDs and total count from search results
	documentIDs, totalCount, err := e.extractDocumentIDs(searchResults)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to extract document IDs from search results", "error", err)
		return nil, 0, err
	}

	e.logger.InfoContext(ctx, "Combined search executed successfully",
		"contentQuery", contentQuery,
		"metadata", metadata,
		"tenantID", tenantID,
		"resultCount", len(documentIDs),
		"totalCount", totalCount)

	return documentIDs, totalCount, nil
}

// ExecuteFolderSearch executes a search query within a specific folder in Elasticsearch
func (e *elasticsearchQueryExecutor) ExecuteFolderSearch(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) ([]string, int64, error) {
	e.logger.InfoContext(ctx, "Executing folder search",
		"folderID", folderID,
		"query", query,
		"tenantID", tenantID)

	// Validate folder ID, query, and tenant ID
	if folderID == "" {
		return nil, 0, errors.NewValidationError("folder ID cannot be empty")
	}
	if strings.TrimSpace(query) == "" {
		return nil, 0, errors.NewValidationError("search query cannot be empty")
	}
	if tenantID == "" {
		return nil, 0, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get tenant-specific index name
	indexName := fmt.Sprintf("documents-%s", tenantID)

	// Build folder-scoped search query
	searchQuery := e.client.BuildFolderQuery(folderID, query)

	// Apply pagination parameters
	from := 0
	size := 10
	if pagination != nil {
		from = pagination.GetOffset()
		size = pagination.GetLimit()
	} else {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Execute search against Elasticsearch
	searchResults, err := e.client.Search(ctx, indexName, searchQuery, from, size)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to execute folder search",
			"error", err,
			"folderID", folderID,
			"query", query,
			"tenantID", tenantID)
		return nil, 0, errors.NewDependencyError(fmt.Sprintf("failed to execute folder search: %v", err))
	}

	// Extract document IDs and total count from search results
	documentIDs, totalCount, err := e.extractDocumentIDs(searchResults)
	if err != nil {
		e.logger.ErrorContext(ctx, "Failed to extract document IDs from search results", "error", err)
		return nil, 0, err
	}

	e.logger.InfoContext(ctx, "Folder search executed successfully",
		"folderID", folderID,
		"query", query,
		"tenantID", tenantID,
		"resultCount", len(documentIDs),
		"totalCount", totalCount)

	return documentIDs, totalCount, nil
}

// extractDocumentIDs extracts document IDs from Elasticsearch search results
func (e *elasticsearchQueryExecutor) extractDocumentIDs(searchResults map[string]interface{}) ([]string, int64, error) {
	// Extract hits array from search results
	hitsMap, ok := searchResults["hits"].(map[string]interface{})
	if !ok {
		return nil, 0, errors.NewDependencyError("invalid search results format: missing hits object")
	}
	
	// Extract total count from search results
	totalObj, ok := hitsMap["total"].(map[string]interface{})
	if !ok {
		return nil, 0, errors.NewDependencyError("invalid search results format: missing total object")
	}
	
	totalValue, ok := totalObj["value"].(float64)
	if !ok {
		return nil, 0, errors.NewDependencyError("invalid search results format: missing total value")
	}
	
	totalCount := int64(totalValue)
	
	// Extract hits array
	hitsArray, ok := hitsMap["hits"].([]interface{})
	if !ok {
		return nil, totalCount, nil // No results but valid query
	}
	
	// Initialize slice for document IDs
	documentIDs := make([]string, 0, len(hitsArray))
	
	// Iterate through hits and extract document ID from each hit
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		
		id, ok := hitMap["_id"].(string)
		if !ok {
			continue
		}
		
		documentIDs = append(documentIDs, id)
	}
	
	return documentIDs, totalCount, nil
}