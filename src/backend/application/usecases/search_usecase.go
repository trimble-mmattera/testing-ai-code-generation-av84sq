// Package usecases provides application-level use cases for the Document Management Platform.
package usecases

import (
	"context" // standard library
	"fmt"     // standard library
	"strings" // standard library

	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// Error variables for search-related validations
var ErrEmptySearchQuery = errors.NewValidationError("search query cannot be empty")
var ErrEmptyMetadataQuery = errors.NewValidationError("metadata search criteria cannot be empty")
var ErrEmptyTenantID = errors.NewValidationError("tenant ID cannot be empty")
var ErrEmptyFolderID = errors.NewValidationError("folder ID cannot be empty")
var ErrNoSearchCriteria = errors.NewValidationError("at least one search criteria (content or metadata) must be provided")

// SearchUseCase defines the interface for search-related use cases.
type SearchUseCase interface {
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

// searchUseCaseImpl implements the SearchUseCase interface.
type searchUseCaseImpl struct {
	searchService services.SearchService
}

// NewSearchUseCase creates a new SearchUseCase instance with the provided dependencies.
func NewSearchUseCase(searchService services.SearchService) (SearchUseCase, error) {
	if searchService == nil {
		return nil, fmt.Errorf("searchService cannot be nil")
	}

	return &searchUseCaseImpl{
		searchService: searchService,
	}, nil
}

// SearchByContent searches documents by their content.
func (u *searchUseCaseImpl) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
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

	// Call the domain service to perform the search
	result, err := u.searchService.SearchByContent(ctx, query, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to perform content search", "error", err, "query", query, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to perform content search")
	}

	return result, nil
}

// SearchByMetadata searches documents by their metadata.
func (u *searchUseCaseImpl) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
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

	// Call the domain service to perform the search
	result, err := u.searchService.SearchByMetadata(ctx, metadata, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to perform metadata search", "error", err, "metadata", metadata, "tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to perform metadata search")
	}

	return result, nil
}

// CombinedSearch performs a search using both content and metadata criteria.
func (u *searchUseCaseImpl) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
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

	// Call the domain service to perform the search
	result, err := u.searchService.CombinedSearch(ctx, contentQuery, metadata, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to perform combined search",
			"error", err,
			"contentQuery", contentQuery,
			"metadata", metadata,
			"tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to perform combined search")
	}

	return result, nil
}

// SearchInFolder searches documents within a specific folder.
func (u *searchUseCaseImpl) SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
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

	// Call the domain service to perform the search
	result, err := u.searchService.SearchInFolder(ctx, folderID, query, tenantID, pagination)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to perform folder search",
			"error", err,
			"folderID", folderID,
			"query", query,
			"tenantID", tenantID)
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to perform folder search")
	}

	return result, nil
}

// IndexDocument indexes a document for search.
func (u *searchUseCaseImpl) IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error {
	logger.InfoContext(ctx, "IndexDocument request", "documentID", documentID, "tenantID", tenantID)

	// Validate document ID
	if documentID == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}

	// Validate tenant ID
	if tenantID == "" {
		return ErrEmptyTenantID
	}

	// Validate content
	if content == nil || len(content) == 0 {
		return errors.NewValidationError("document content cannot be empty")
	}

	// Call the domain service to index the document
	err := u.searchService.IndexDocument(ctx, documentID, tenantID, content)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to index document", "error", err, "documentID", documentID, "tenantID", tenantID)
		return errors.Wrap(err, "failed to index document")
	}

	logger.InfoContext(ctx, "Document indexed successfully", "documentID", documentID, "tenantID", tenantID)
	return nil
}

// RemoveDocumentFromIndex removes a document from the search index.
func (u *searchUseCaseImpl) RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error {
	logger.InfoContext(ctx, "RemoveDocumentFromIndex request", "documentID", documentID, "tenantID", tenantID)

	// Validate document ID
	if documentID == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}

	// Validate tenant ID
	if tenantID == "" {
		return ErrEmptyTenantID
	}

	// Call the domain service to remove the document from the index
	err := u.searchService.RemoveDocumentFromIndex(ctx, documentID, tenantID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to remove document from index", "error", err, "documentID", documentID, "tenantID", tenantID)
		return errors.Wrap(err, "failed to remove document from index")
	}

	logger.InfoContext(ctx, "Document removed from index successfully", "documentID", documentID, "tenantID", tenantID)
	return nil
}