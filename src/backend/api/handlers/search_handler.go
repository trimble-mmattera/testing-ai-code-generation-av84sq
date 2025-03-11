// Package handlers provides HTTP handlers for the Document Management Platform API.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"../dto"
	"../validators"
	"../../application/usecases"
	"../../domain/models"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
	searchUseCase usecases.SearchUseCase
}

// NewSearchHandler creates a new SearchHandler with the provided search use case
func NewSearchHandler(searchUseCase usecases.SearchUseCase) *SearchHandler {
	if searchUseCase == nil {
		logger.Error("searchUseCase cannot be nil")
		panic("searchUseCase cannot be nil")
	}
	return &SearchHandler{
		searchUseCase: searchUseCase,
	}
}

// SearchByContent handles content-based search requests
func (h *SearchHandler) SearchByContent(c *gin.Context) {
	// Log the incoming request
	logger.InfoContext(c, "Content search request received")

	// Extract tenant ID from context
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		logger.ErrorContext(c, "Missing tenant ID in context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Unauthorized: missing tenant context"))
		return
	}

	// Bind request
	var request dto.ContentSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.ErrorContext(c, "Failed to parse content search request", "error", err)
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request format"))
		return
	}

	// Validate request
	if err := validators.ValidateContentSearchRequest(&request); err != nil {
		logger.ErrorContext(c, "Invalid content search request", "error", err)
		if errors.IsValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse([]string{err.Error()}))
		} else {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse(err.Error()))
		}
		return
	}

	// Create pagination parameters
	pagination := utils.NewPagination(request.Page, request.PageSize)

	// Call searchUseCase.SearchByContent with query, tenant ID, and pagination
	result, err := h.searchUseCase.SearchByContent(c, request.Query, tenantID, pagination)
	if err != nil {
		h.handleSearchError(c, err)
		return
	}

	// Convert domain documents to DocumentSearchResult DTOs
	searchResults := h.convertToSearchResults(result.Items)

	// Create page info from pagination and total items
	pageInfo := utils.NewPageInfo(pagination, result.Pagination.TotalItems)

	// Return 200 OK with search results and pagination info
	c.JSON(http.StatusOK, dto.NewDocumentSearchResponse(searchResults, pageInfo))
}

// SearchByMetadata handles metadata-based search requests
func (h *SearchHandler) SearchByMetadata(c *gin.Context) {
	// Log the incoming request
	logger.InfoContext(c, "Metadata search request received")

	// Extract tenant ID from context
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		logger.ErrorContext(c, "Missing tenant ID in context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Unauthorized: missing tenant context"))
		return
	}

	// Bind request
	var request dto.MetadataSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.ErrorContext(c, "Failed to parse metadata search request", "error", err)
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request format"))
		return
	}

	// Validate request
	if err := validators.ValidateMetadataSearchRequest(&request); err != nil {
		logger.ErrorContext(c, "Invalid metadata search request", "error", err)
		if errors.IsValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse([]string{err.Error()}))
		} else {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse(err.Error()))
		}
		return
	}

	// Create pagination parameters
	pagination := utils.NewPagination(request.Page, request.PageSize)

	// Call searchUseCase.SearchByMetadata with metadata, tenant ID, and pagination
	result, err := h.searchUseCase.SearchByMetadata(c, request.Metadata, tenantID, pagination)
	if err != nil {
		h.handleSearchError(c, err)
		return
	}

	// Convert domain documents to DocumentSearchResult DTOs
	searchResults := h.convertToSearchResults(result.Items)

	// Create page info from pagination and total items
	pageInfo := utils.NewPageInfo(pagination, result.Pagination.TotalItems)

	// Return 200 OK with search results and pagination info
	c.JSON(http.StatusOK, dto.NewDocumentSearchResponse(searchResults, pageInfo))
}

// CombinedSearch handles combined content and metadata search requests
func (h *SearchHandler) CombinedSearch(c *gin.Context) {
	// Log the incoming request
	logger.InfoContext(c, "Combined search request received")

	// Extract tenant ID from context
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		logger.ErrorContext(c, "Missing tenant ID in context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Unauthorized: missing tenant context"))
		return
	}

	// Bind request
	var request dto.CombinedSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.ErrorContext(c, "Failed to parse combined search request", "error", err)
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request format"))
		return
	}

	// Validate request
	if err := validators.ValidateCombinedSearchRequest(&request); err != nil {
		logger.ErrorContext(c, "Invalid combined search request", "error", err)
		if errors.IsValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse([]string{err.Error()}))
		} else {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse(err.Error()))
		}
		return
	}

	// Create pagination parameters
	pagination := utils.NewPagination(request.Page, request.PageSize)

	// Call searchUseCase.CombinedSearch with query, metadata, tenant ID, and pagination
	result, err := h.searchUseCase.CombinedSearch(c, request.Query, request.Metadata, tenantID, pagination)
	if err != nil {
		h.handleSearchError(c, err)
		return
	}

	// Convert domain documents to DocumentSearchResult DTOs
	searchResults := h.convertToSearchResults(result.Items)

	// Create page info from pagination and total items
	pageInfo := utils.NewPageInfo(pagination, result.Pagination.TotalItems)

	// Return 200 OK with search results and pagination info
	c.JSON(http.StatusOK, dto.NewDocumentSearchResponse(searchResults, pageInfo))
}

// SearchInFolder handles folder-scoped search requests
func (h *SearchHandler) SearchInFolder(c *gin.Context) {
	// Log the incoming request
	logger.InfoContext(c, "Folder search request received")

	// Extract tenant ID from context
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		logger.ErrorContext(c, "Missing tenant ID in context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse("Unauthorized: missing tenant context"))
		return
	}

	// Bind request
	var request dto.FolderSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.ErrorContext(c, "Failed to parse folder search request", "error", err)
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request format"))
		return
	}

	// Validate request
	if err := validators.ValidateFolderSearchRequest(&request); err != nil {
		logger.ErrorContext(c, "Invalid folder search request", "error", err)
		if errors.IsValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse([]string{err.Error()}))
		} else {
			c.JSON(http.StatusBadRequest, dto.NewErrorResponse(err.Error()))
		}
		return
	}

	// Create pagination parameters
	pagination := utils.NewPagination(request.Page, request.PageSize)

	// Call searchUseCase.SearchInFolder with folder ID, query, tenant ID, and pagination
	result, err := h.searchUseCase.SearchInFolder(c, request.FolderID, request.Query, tenantID, pagination)
	if err != nil {
		h.handleSearchError(c, err)
		return
	}

	// Convert domain documents to DocumentSearchResult DTOs
	searchResults := h.convertToSearchResults(result.Items)

	// Create page info from pagination and total items
	pageInfo := utils.NewPageInfo(pagination, result.Pagination.TotalItems)

	// Return 200 OK with search results and pagination info
	c.JSON(http.StatusOK, dto.NewDocumentSearchResponse(searchResults, pageInfo))
}

// handleSearchError handles errors from search operations and returns appropriate HTTP responses
func (h *SearchHandler) handleSearchError(c *gin.Context, err error) {
	logger.ErrorContext(c, "Search error occurred", "error", err.Error())

	if errors.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse(err.Error()))
		return
	}

	if errors.IsResourceNotFoundError(err) {
		c.JSON(http.StatusNotFound, dto.NewErrorResponse(err.Error()))
		return
	}

	statusCode := errors.GetStatusCode(err)
	c.JSON(statusCode, dto.NewErrorResponse(err.Error()))
}

// convertToSearchResults converts domain documents to search result DTOs
func (h *SearchHandler) convertToSearchResults(documents []models.Document) []dto.DocumentSearchResult {
	results := make([]dto.DocumentSearchResult, 0, len(documents))
	for _, doc := range documents {
		results = append(results, dto.DocumentToSearchResult(doc))
	}
	return results
}