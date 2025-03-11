// Package services provides domain services for the Document Management Platform.
package services

import (
	"context" // standard library
	"fmt"    // standard library
	"strings" // standard library

	"../models"
	"../repositories"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// Error variables for search-related operations
var ErrEmptySearchQuery = errors.NewValidationError("search query cannot be empty")
var ErrEmptyMetadataQuery = errors.NewValidationError("metadata search criteria cannot be empty")
var ErrEmptyTenantID = errors.NewValidationError("tenant ID cannot be empty")
var ErrEmptyDocumentID = errors.NewValidationError("document ID cannot be empty")
var ErrEmptyFolderID = errors.NewValidationError("folder ID cannot be empty")
var ErrEmptyContent = errors.NewValidationError("document content cannot be empty")
var ErrNoSearchCriteria = errors.NewValidationError("at least one search criteria (content or metadata) must be provided")

// SearchIndexer defines operations for indexing documents in the search engine
type SearchIndexer interface {
	// IndexDocument indexes a document for search
	IndexDocument(ctx context.Context, document *models.Document, content []byte) error
	
	// RemoveDocument removes a document from the search index
	RemoveDocument(ctx context.Context, documentID string, tenantID string) error
}

// SearchQueryExecutor defines operations for executing search queries
type SearchQueryExecutor interface {
	// ExecuteContentSearch executes a content-based search query
	ExecuteContentSearch(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
	
	// ExecuteMetadataSearch executes a metadata-based search query
	ExecuteMetadataSearch(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
	
	// ExecuteCombinedSearch executes a combined content and metadata search query
	ExecuteCombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
	
	// ExecuteFolderSearch executes a search query within a specific folder
	ExecuteFolderSearch(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
}

// SearchService defines the search service operations
type SearchService interface {
	// SearchByContent searches documents by their content
	SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
	
	// SearchByMetadata searches documents by their metadata
	SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
	
	// CombinedSearch performs a search using both content and metadata criteria
	CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
	
	// SearchInFolder searches documents within a specific folder
	SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
	
	// IndexDocument indexes a document for search
	IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error
	
	// RemoveDocumentFromIndex removes a document from the search index
	RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error
}

// NewSearchService creates a new SearchService instance with the provided dependencies
func NewSearchService(indexer SearchIndexer, queryExecutor SearchQueryExecutor, documentRepo repositories.DocumentRepository) (SearchService, error) {
	if indexer == nil {
		return nil, fmt.Errorf("indexer cannot be nil")
	}
	if queryExecutor == nil {
		return nil, fmt.Errorf("queryExecutor cannot be nil")
	}
	if documentRepo == nil {
		return nil, fmt.Errorf("documentRepo cannot be nil")
	}

	return &searchServiceImpl{
		indexer:       indexer,
		queryExecutor: queryExecutor,
		documentRepo:  documentRepo,
		logger:        logger.WithField("service", "search"),
	}, nil
}

// searchServiceImpl implements the SearchService interface
type searchServiceImpl struct {
	indexer       SearchIndexer
	queryExecutor SearchQueryExecutor
	documentRepo  repositories.DocumentRepository
	logger        *logger.Logger
}

// SearchByContent searches documents by their content
func (s *searchServiceImpl) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	logger.InfoContext(ctx, "SearchByContent request", "query", query, "tenantID", tenantID)
	
	// Validate query
	if strings.TrimSpace(query) == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptySearchQuery
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptyTenantID
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Execute content search query
	docIDs, totalCount, err := s.queryExecutor.ExecuteContentSearch(ctx, query, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute content search", "error", err, "query", query, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Retrieve documents
	documents, err := s.getDocumentsByIDs(ctx, docIDs, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve documents by IDs", "error", err, "docIDs", docIDs, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Create and return paginated result
	return utils.NewPaginatedResult(documents, pagination, totalCount), nil
}

// SearchByMetadata searches documents by their metadata
func (s *searchServiceImpl) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	logger.InfoContext(ctx, "SearchByMetadata request", "metadata", metadata, "tenantID", tenantID)
	
	// Validate metadata
	if metadata == nil || len(metadata) == 0 {
		return utils.PaginatedResult[models.Document]{}, ErrEmptyMetadataQuery
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptyTenantID
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Execute metadata search query
	docIDs, totalCount, err := s.queryExecutor.ExecuteMetadataSearch(ctx, metadata, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute metadata search", "error", err, "metadata", metadata, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Retrieve documents
	documents, err := s.getDocumentsByIDs(ctx, docIDs, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve documents by IDs", "error", err, "docIDs", docIDs, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Create and return paginated result
	return utils.NewPaginatedResult(documents, pagination, totalCount), nil
}

// CombinedSearch performs a search using both content and metadata criteria
func (s *searchServiceImpl) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	logger.InfoContext(ctx, "CombinedSearch request", "contentQuery", contentQuery, "metadata", metadata, "tenantID", tenantID)
	
	// Validate that at least one search criterion is provided
	contentQueryEmpty := strings.TrimSpace(contentQuery) == ""
	metadataEmpty := metadata == nil || len(metadata) == 0
	
	if contentQueryEmpty && metadataEmpty {
		return utils.PaginatedResult[models.Document]{}, ErrNoSearchCriteria
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptyTenantID
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Execute combined search query
	docIDs, totalCount, err := s.queryExecutor.ExecuteCombinedSearch(ctx, contentQuery, metadata, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute combined search", 
			"error", err, 
			"contentQuery", contentQuery, 
			"metadata", metadata, 
			"tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Retrieve documents
	documents, err := s.getDocumentsByIDs(ctx, docIDs, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve documents by IDs", "error", err, "docIDs", docIDs, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Create and return paginated result
	return utils.NewPaginatedResult(documents, pagination, totalCount), nil
}

// SearchInFolder searches documents within a specific folder
func (s *searchServiceImpl) SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	logger.InfoContext(ctx, "SearchInFolder request", "folderID", folderID, "query", query, "tenantID", tenantID)
	
	// Validate folder ID
	if folderID == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptyFolderID
	}
	
	// Validate query
	if strings.TrimSpace(query) == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptySearchQuery
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, ErrEmptyTenantID
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Execute folder search query
	docIDs, totalCount, err := s.queryExecutor.ExecuteFolderSearch(ctx, folderID, query, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute folder search", 
			"error", err, 
			"folderID", folderID,
			"query", query, 
			"tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Retrieve documents
	documents, err := s.getDocumentsByIDs(ctx, docIDs, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve documents by IDs", "error", err, "docIDs", docIDs, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, err
	}
	
	// Create and return paginated result
	return utils.NewPaginatedResult(documents, pagination, totalCount), nil
}

// IndexDocument indexes a document for search
func (s *searchServiceImpl) IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error {
	logger.InfoContext(ctx, "IndexDocument request", "documentID", documentID, "tenantID", tenantID)
	
	// Validate document ID
	if documentID == "" {
		return ErrEmptyDocumentID
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return ErrEmptyTenantID
	}
	
	// Validate content
	if content == nil || len(content) == 0 {
		return ErrEmptyContent
	}
	
	// Retrieve document
	document, err := s.documentRepo.GetByID(ctx, documentID, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to retrieve document for indexing", "error", err, "documentID", documentID, "tenantID", tenantID)
		return err
	}
	
	// Verify document belongs to tenant
	if document.TenantID != tenantID {
		logger.WarnContext(ctx, "Document does not belong to tenant", "documentID", documentID, "tenantID", tenantID, "documentTenantID", document.TenantID)
		return errors.NewAuthorizationError("document does not belong to tenant")
	}
	
	// Index document
	err = s.indexer.IndexDocument(ctx, document, content)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to index document", "error", err, "documentID", documentID, "tenantID", tenantID)
		return err
	}
	
	logger.InfoContext(ctx, "Document indexed successfully", "documentID", documentID, "tenantID", tenantID)
	return nil
}

// RemoveDocumentFromIndex removes a document from the search index
func (s *searchServiceImpl) RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error {
	logger.InfoContext(ctx, "RemoveDocumentFromIndex request", "documentID", documentID, "tenantID", tenantID)
	
	// Validate document ID
	if documentID == "" {
		return ErrEmptyDocumentID
	}
	
	// Validate tenant ID
	if tenantID == "" {
		return ErrEmptyTenantID
	}
	
	// Remove document from index
	err := s.indexer.RemoveDocument(ctx, documentID, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to remove document from index", "error", err, "documentID", documentID, "tenantID", tenantID)
		return err
	}
	
	logger.InfoContext(ctx, "Document removed from index successfully", "documentID", documentID, "tenantID", tenantID)
	return nil
}

// getDocumentsByIDs retrieves documents by their IDs with tenant isolation
func (s *searchServiceImpl) getDocumentsByIDs(ctx context.Context, documentIDs []string, tenantID string) ([]*models.Document, error) {
	if len(documentIDs) == 0 {
		return []*models.Document{}, nil
	}
	
	documents, err := s.documentRepo.GetDocumentsByIDs(ctx, documentIDs, tenantID)
	if err != nil {
		return nil, err
	}
	
	return documents, nil
}