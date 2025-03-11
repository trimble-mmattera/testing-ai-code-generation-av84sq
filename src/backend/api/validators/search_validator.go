// Package validators provides validation functions for search-related API requests.
// This file contains validators for content search, metadata search, combined search,
// and folder-scoped search requests to ensure data integrity and proper error handling.
package validators

import (
	"fmt"    // standard library
	"strings" // standard library

	"../dto"
	"../../pkg/errors"
	"../../pkg/validator"
)

// Constants for validation limits
const (
	// Pagination limits
	MaxPageSize = 100
	MinPageSize = 1
	MinPage     = 1

	// Metadata limits
	MaxMetadataCount       = 50
	MaxMetadataKeyLength   = 64
	MaxMetadataValueLength = 1024
)

// Valid sort fields and orders
var (
	ValidSortFields = []string{dto.SortByRelevance, dto.SortByName, dto.SortByCreatedAt, dto.SortByUpdatedAt, dto.SortBySize}
	ValidSortOrders = []string{dto.SortOrderAsc, dto.SortOrderDesc}
)

// ValidateContentSearchRequest validates a content-based search request
func ValidateContentSearchRequest(request *dto.ContentSearchRequest) error {
	// Check if request is nil
	if request == nil {
		return errors.NewValidationError("search request cannot be nil")
	}

	// Validate struct using validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate query is not empty
	if err := validator.ValidateRequired(request.Query, "query"); err != nil {
		return err
	}

	// Validate pagination parameters
	if err := validatePagination(request.Page, request.PageSize); err != nil {
		return err
	}

	// Validate sort parameters if provided
	if err := validateSortParameters(request.SortBy, request.SortOrder); err != nil {
		return err
	}

	return nil
}

// ValidateMetadataSearchRequest validates a metadata-based search request
func ValidateMetadataSearchRequest(request *dto.MetadataSearchRequest) error {
	// Check if request is nil
	if request == nil {
		return errors.NewValidationError("search request cannot be nil")
	}

	// Validate struct using validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate metadata
	if err := validateMetadata(request.Metadata); err != nil {
		return err
	}

	// Validate pagination parameters
	if err := validatePagination(request.Page, request.PageSize); err != nil {
		return err
	}

	// Validate sort parameters if provided
	if err := validateSortParameters(request.SortBy, request.SortOrder); err != nil {
		return err
	}

	return nil
}

// ValidateCombinedSearchRequest validates a combined content and metadata search request
func ValidateCombinedSearchRequest(request *dto.CombinedSearchRequest) error {
	// Check if request is nil
	if request == nil {
		return errors.NewValidationError("search request cannot be nil")
	}

	// Validate struct using validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate that at least one of query or metadata is provided
	if request.Query == "" && (request.Metadata == nil || len(request.Metadata) == 0) {
		return errors.NewValidationError("either query or metadata must be provided")
	}

	// If metadata is provided, validate it
	if request.Metadata != nil && len(request.Metadata) > 0 {
		if err := validateMetadata(request.Metadata); err != nil {
			return err
		}
	}

	// Validate pagination parameters
	if err := validatePagination(request.Page, request.PageSize); err != nil {
		return err
	}

	// Validate sort parameters if provided
	if err := validateSortParameters(request.SortBy, request.SortOrder); err != nil {
		return err
	}

	return nil
}

// ValidateFolderSearchRequest validates a folder-scoped search request
func ValidateFolderSearchRequest(request *dto.FolderSearchRequest) error {
	// Check if request is nil
	if request == nil {
		return errors.NewValidationError("search request cannot be nil")
	}

	// Validate struct using validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate folderID is not empty and is a valid UUID
	if err := validator.ValidateRequired(request.FolderID, "folder_id"); err != nil {
		return err
	}
	if err := validator.ValidateUUID(request.FolderID); err != nil {
		return err
	}

	// Validate query is not empty
	if err := validator.ValidateRequired(request.Query, "query"); err != nil {
		return err
	}

	// Validate pagination parameters
	if err := validatePagination(request.Page, request.PageSize); err != nil {
		return err
	}

	// Validate sort parameters if provided
	if err := validateSortParameters(request.SortBy, request.SortOrder); err != nil {
		return err
	}

	return nil
}

// validatePagination validates pagination parameters
func validatePagination(page, pageSize int) error {
	if page < MinPage {
		return errors.NewValidationError(fmt.Sprintf("page must be greater than or equal to %d", MinPage))
	}

	if pageSize < MinPageSize || pageSize > MaxPageSize {
		return errors.NewValidationError(fmt.Sprintf("page size must be between %d and %d", MinPageSize, MaxPageSize))
	}

	return nil
}

// validateSortParameters validates sorting parameters
func validateSortParameters(sortBy, sortOrder string) error {
	// If sortBy is empty, return nil (default sorting will be used)
	if sortBy == "" {
		return nil
	}

	// Validate sortBy is one of the valid options
	validSort := false
	for _, validField := range ValidSortFields {
		if sortBy == validField {
			validSort = true
			break
		}
	}
	if !validSort {
		return errors.NewValidationError(fmt.Sprintf("sort_by must be one of: %s", strings.Join(ValidSortFields, ", ")))
	}

	// If sortOrder is empty, return nil (default order will be used)
	if sortOrder == "" {
		return nil
	}

	// Validate sortOrder is one of the valid options
	validOrder := false
	for _, validOrderOption := range ValidSortOrders {
		if sortOrder == validOrderOption {
			validOrder = true
			break
		}
	}
	if !validOrder {
		return errors.NewValidationError(fmt.Sprintf("sort_order must be one of: %s", strings.Join(ValidSortOrders, ", ")))
	}

	return nil
}

// validateMetadata validates search metadata
func validateMetadata(metadata map[string]string) error {
	// Check if metadata is nil or empty
	if metadata == nil || len(metadata) == 0 {
		return errors.NewValidationError("at least one metadata field is required")
	}

	// Check if metadata count is within limit
	if len(metadata) > MaxMetadataCount {
		return errors.NewValidationError(fmt.Sprintf("maximum of %d metadata entries allowed", MaxMetadataCount))
	}

	// Validate each metadata entry
	for key, value := range metadata {
		// Validate key
		if key == "" {
			return errors.NewValidationError("metadata key cannot be empty")
		}
		if len(key) > MaxMetadataKeyLength {
			return errors.NewValidationError(fmt.Sprintf("metadata key length cannot exceed %d characters", MaxMetadataKeyLength))
		}

		// Validate value (can be empty, but length is limited)
		if len(value) > MaxMetadataValueLength {
			return errors.NewValidationError(fmt.Sprintf("metadata value length cannot exceed %d characters", MaxMetadataValueLength))
		}
	}

	return nil
}