// Package validators provides validation functions for folder-related API requests.
// This file contains validators for folder creation, update, move, listing, and search requests
// to ensure data integrity and proper error handling.
package validators

import (
	"fmt"     // standard library
	"strings" // standard library

	"../dto"              // For folder DTOs
	"../../pkg/errors"    // For creating standardized validation errors
	"../../pkg/validator" // For validation utilities
)

// MaxFolderNameLength is the maximum allowed length for folder names
const MaxFolderNameLength = 255

// MinFolderNameLength is the minimum allowed length for folder names
const MinFolderNameLength = 1

// ValidSortFields defines the allowed fields for sorting folder listings
var ValidSortFields = []string{"name", "created_at", "updated_at"}

// ValidSortOrders defines the allowed sort orders
var ValidSortOrders = []string{"asc", "desc"}

// ValidateCreateFolderRequest validates a folder creation request
func ValidateCreateFolderRequest(request *dto.FolderCreateRequest) error {
	if request == nil {
		return errors.NewValidationError("create folder request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate folder name
	if err := validateFolderName(request.Name); err != nil {
		return err
	}

	// Validate parent folder ID if provided
	if request.ParentID != "" {
		if err := validator.ValidateUUID(request.ParentID); err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid parent folder ID: %s", err.Error()))
		}
	}

	return nil
}

// ValidateUpdateFolderRequest validates a folder update request
func ValidateUpdateFolderRequest(request *dto.FolderUpdateRequest) error {
	if request == nil {
		return errors.NewValidationError("update folder request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate folder name
	if err := validateFolderName(request.Name); err != nil {
		return err
	}

	return nil
}

// ValidateMoveFolderRequest validates a folder move request
func ValidateMoveFolderRequest(request *dto.FolderMoveRequest) error {
	if request == nil {
		return errors.NewValidationError("move folder request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate new parent folder ID
	if err := validator.ValidateUUID(request.NewParentID); err != nil {
		return errors.NewValidationError(fmt.Sprintf("invalid new parent folder ID: %s", err.Error()))
	}

	return nil
}

// ValidateFolderListRequest validates a folder listing request
func ValidateFolderListRequest(request *dto.FolderListRequest) error {
	if request == nil {
		return errors.NewValidationError("folder list request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate parent folder ID if provided
	if request.ParentID != "" {
		if err := validator.ValidateUUID(request.ParentID); err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid parent folder ID: %s", err.Error()))
		}
	}

	// Validate pagination parameters
	if request.Page <= 0 {
		return errors.NewValidationError("page number must be greater than 0")
	}
	
	if request.PageSize <= 0 {
		return errors.NewValidationError("page size must be greater than 0")
	}
	
	if request.PageSize > 100 {
		return errors.NewValidationError("page size cannot exceed 100")
	}

	// Validate sort parameters if provided
	if err := validateSortParameters(request.SortBy, request.SortOrder); err != nil {
		return err
	}

	return nil
}

// ValidateFolderSearchRequest validates a folder search request
func ValidateFolderSearchRequest(request *dto.FolderSearchRequest) error {
	if request == nil {
		return errors.NewValidationError("folder search request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate search query
	if err := validator.ValidateRequired(request.Query, "query"); err != nil {
		return err
	}
	
	if err := validator.ValidateMaxLength(request.Query, 100, "query"); err != nil {
		return err
	}

	// Validate pagination parameters
	if request.Page <= 0 {
		return errors.NewValidationError("page number must be greater than 0")
	}
	
	if request.PageSize <= 0 {
		return errors.NewValidationError("page size must be greater than 0")
	}
	
	if request.PageSize > 100 {
		return errors.NewValidationError("page size cannot exceed 100")
	}

	return nil
}

// validateFolderName validates a folder name against naming rules
func validateFolderName(name string) error {
	// Check if name is empty
	if err := validator.ValidateRequired(name, "name"); err != nil {
		return err
	}

	// Check if name length is within allowed limits
	if err := validator.ValidateMinLength(name, MinFolderNameLength, "name"); err != nil {
		return err
	}
	
	if err := validator.ValidateMaxLength(name, MaxFolderNameLength, "name"); err != nil {
		return err
	}

	// Check for invalid characters in folder name
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return errors.NewValidationError("folder name contains invalid characters")
	}

	return nil
}

// validateSortParameters validates sorting parameters for folder listing
func validateSortParameters(sortBy, sortOrder string) error {
	// If sortBy is empty, return nil (default sorting will be used)
	if sortBy == "" {
		return nil
	}

	// Convert string slice to interface slice for ValidateOneOf
	sortByValues := make([]interface{}, len(ValidSortFields))
	for i, v := range ValidSortFields {
		sortByValues[i] = v
	}
	
	if err := validator.ValidateOneOf(sortBy, sortByValues, "sort field"); err != nil {
		return err
	}

	// If sortOrder is empty, return nil (default order will be used)
	if sortOrder == "" {
		return nil
	}

	// Convert string slice to interface slice for ValidateOneOf
	sortOrderValues := make([]interface{}, len(ValidSortOrders))
	for i, v := range ValidSortOrders {
		sortOrderValues[i] = v
	}
	
	if err := validator.ValidateOneOf(sortOrder, sortOrderValues, "sort order"); err != nil {
		return err
	}

	return nil
}