// Package dto provides Data Transfer Objects for the Document Management Platform API.
// This file contains DTOs for search-related operations, including content search,
// metadata search, combined search, and folder-scoped search. These DTOs serve as the
// interface between the API layer and the domain layer, facilitating document search
// functionality.
package dto

import (
	"time" // standard library

	"../../domain/models"
	"../../pkg/errors"
	"../../pkg/utils/pagination"
	timeutils "../../pkg/utils/time_utils"
	"./document_dto"
	"./response_dto"
)

// Sort constants for search requests
const (
	SortByRelevance = "relevance"
	SortByName      = "name"
	SortByCreatedAt = "created_at"
	SortByUpdatedAt = "updated_at"
	SortBySize      = "size"
	SortOrderAsc    = "asc"
	SortOrderDesc   = "desc"
)

// ContentSearchRequest represents a request for content-based document search
type ContentSearchRequest struct {
	Query     string `json:"query"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// Validate validates the content search request
func (r *ContentSearchRequest) Validate() error {
	if r.Query == "" {
		return errors.NewValidationError("search query is required")
	}
	
	if r.Page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}
	
	if r.PageSize < 1 || r.PageSize > 100 {
		return errors.NewValidationError("page size must be between 1 and 100")
	}
	
	if r.SortBy != "" && r.SortBy != SortByRelevance && 
	   r.SortBy != SortByName && r.SortBy != SortByCreatedAt && 
	   r.SortBy != SortByUpdatedAt && r.SortBy != SortBySize {
		return errors.NewValidationError("invalid sort_by parameter")
	}
	
	if r.SortOrder != "" && r.SortOrder != SortOrderAsc && r.SortOrder != SortOrderDesc {
		return errors.NewValidationError("sort_order must be 'asc' or 'desc'")
	}
	
	return nil
}

// MetadataSearchRequest represents a request for metadata-based document search
type MetadataSearchRequest struct {
	Metadata  map[string]string `json:"metadata"`
	Page      int               `json:"page"`
	PageSize  int               `json:"page_size"`
	SortBy    string            `json:"sort_by,omitempty"`
	SortOrder string            `json:"sort_order,omitempty"`
}

// Validate validates the metadata search request
func (r *MetadataSearchRequest) Validate() error {
	if r.Metadata == nil || len(r.Metadata) == 0 {
		return errors.NewValidationError("at least one metadata field is required")
	}
	
	if r.Page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}
	
	if r.PageSize < 1 || r.PageSize > 100 {
		return errors.NewValidationError("page size must be between 1 and 100")
	}
	
	if r.SortBy != "" && r.SortBy != SortByRelevance && 
	   r.SortBy != SortByName && r.SortBy != SortByCreatedAt && 
	   r.SortBy != SortByUpdatedAt && r.SortBy != SortBySize {
		return errors.NewValidationError("invalid sort_by parameter")
	}
	
	if r.SortOrder != "" && r.SortOrder != SortOrderAsc && r.SortOrder != SortOrderDesc {
		return errors.NewValidationError("sort_order must be 'asc' or 'desc'")
	}
	
	return nil
}

// CombinedSearchRequest represents a request for combined content and metadata search
type CombinedSearchRequest struct {
	Query     string            `json:"query,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Page      int               `json:"page"`
	PageSize  int               `json:"page_size"`
	SortBy    string            `json:"sort_by,omitempty"`
	SortOrder string            `json:"sort_order,omitempty"`
}

// Validate validates the combined search request
func (r *CombinedSearchRequest) Validate() error {
	if r.Query == "" && (r.Metadata == nil || len(r.Metadata) == 0) {
		return errors.NewValidationError("either query or metadata must be provided")
	}
	
	if r.Page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}
	
	if r.PageSize < 1 || r.PageSize > 100 {
		return errors.NewValidationError("page size must be between 1 and 100")
	}
	
	if r.SortBy != "" && r.SortBy != SortByRelevance && 
	   r.SortBy != SortByName && r.SortBy != SortByCreatedAt && 
	   r.SortBy != SortByUpdatedAt && r.SortBy != SortBySize {
		return errors.NewValidationError("invalid sort_by parameter")
	}
	
	if r.SortOrder != "" && r.SortOrder != SortOrderAsc && r.SortOrder != SortOrderDesc {
		return errors.NewValidationError("sort_order must be 'asc' or 'desc'")
	}
	
	return nil
}

// FolderSearchRequest represents a request for folder-scoped document search
type FolderSearchRequest struct {
	FolderID  string `json:"folder_id"`
	Query     string `json:"query"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// Validate validates the folder search request
func (r *FolderSearchRequest) Validate() error {
	if r.FolderID == "" {
		return errors.NewValidationError("folder ID is required")
	}
	
	if r.Query == "" {
		return errors.NewValidationError("search query is required")
	}
	
	if r.Page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}
	
	if r.PageSize < 1 || r.PageSize > 100 {
		return errors.NewValidationError("page size must be between 1 and 100")
	}
	
	if r.SortBy != "" && r.SortBy != SortByRelevance && 
	   r.SortBy != SortByName && r.SortBy != SortByCreatedAt && 
	   r.SortBy != SortByUpdatedAt && r.SortBy != SortBySize {
		return errors.NewValidationError("invalid sort_by parameter")
	}
	
	if r.SortOrder != "" && r.SortOrder != SortOrderAsc && r.SortOrder != SortOrderDesc {
		return errors.NewValidationError("sort_order must be 'asc' or 'desc'")
	}
	
	return nil
}

// DocumentSearchResult represents a document in search results
type DocumentSearchResult struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ContentType string  `json:"content_type"`
	Size        int64   `json:"size"`
	FolderID    string  `json:"folder_id"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	CreatedBy   string  `json:"created_by"`
	Relevance   float64 `json:"relevance,omitempty"`
}

// DocumentSearchResponse represents a response to a document search request
type DocumentSearchResponse struct {
	Success    bool                   `json:"success"`
	Timestamp  string                 `json:"timestamp"`
	Results    []DocumentSearchResult `json:"results"`
	Pagination pagination.PageInfo    `json:"pagination"`
}

// ErrorResponse represents an error response for search operations
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// ValidationErrorResponse represents a validation error response for search operations
type ValidationErrorResponse struct {
	Success   bool     `json:"success"`
	Timestamp string   `json:"timestamp"`
	Errors    []string `json:"errors"`
}

// NewDocumentSearchResponse creates a new DocumentSearchResponse with the given search results and pagination info
func NewDocumentSearchResponse(results []DocumentSearchResult, pageInfo pagination.PageInfo) DocumentSearchResponse {
	return DocumentSearchResponse{
		Success:    true,
		Timestamp:  timeutils.FormatTimeDefault(time.Now()),
		Results:    results,
		Pagination: pageInfo,
	}
}

// NewErrorResponse creates a new ErrorResponse with the given error message
func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTimeDefault(time.Now()),
		Message:   message,
	}
}

// NewValidationErrorResponse creates a new ValidationErrorResponse with the given validation errors
func NewValidationErrorResponse(errors []string) ValidationErrorResponse {
	return ValidationErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTimeDefault(time.Now()),
		Errors:    errors,
	}
}

// DocumentToSearchResult converts a domain Document model to a DocumentSearchResult
func DocumentToSearchResult(document models.Document) DocumentSearchResult {
	return DocumentSearchResult{
		ID:          document.ID,
		Name:        document.Name,
		ContentType: document.ContentType,
		Size:        document.Size,
		FolderID:    document.FolderID,
		Status:      document.Status,
		CreatedAt:   timeutils.FormatTimeDefault(document.CreatedAt),
		UpdatedAt:   timeutils.FormatTimeDefault(document.UpdatedAt),
		CreatedBy:   document.OwnerID,
	}
}