// Package validators provides validation functions for document-related API requests.
// This file contains validators for document creation, update, download, batch download, and listing requests 
// to ensure data integrity and proper error handling.
package validators

import (
	"fmt"
	"path/filepath"
	"strings"

	"../dto"
	"../../domain/models"
	"../../pkg/errors"
	"../../pkg/validator"
)

// MaxDocumentNameLength defines the maximum allowed length for document names
const MaxDocumentNameLength = 255

// MinDocumentNameLength defines the minimum allowed length for document names
const MinDocumentNameLength = 1

// MaxDocumentSize defines the maximum allowed document size in bytes (100MB)
const MaxDocumentSize = 100 * 1024 * 1024

// MaxBatchDownloadCount defines the maximum number of documents in a batch download
const MaxBatchDownloadCount = 100

// MaxMetadataKeyLength defines the maximum allowed length for metadata keys
const MaxMetadataKeyLength = 64

// MaxMetadataValueLength defines the maximum allowed length for metadata values
const MaxMetadataValueLength = 1024

// MaxMetadataCount defines the maximum number of metadata entries per document
const MaxMetadataCount = 50

// ValidSortFields defines the allowed fields for sorting document lists
var ValidSortFields = []string{"name", "created_at", "updated_at", "size", "content_type"}

// ValidSortOrders defines the allowed sort orders
var ValidSortOrders = []string{"asc", "desc"}

// ValidDocumentStatuses defines the allowed document statuses
var ValidDocumentStatuses = []string{
	models.DocumentStatusProcessing,
	models.DocumentStatusAvailable,
	models.DocumentStatusQuarantined,
	models.DocumentStatusFailed,
}

// AllowedContentTypes defines the content types that are allowed for document uploads
var AllowedContentTypes = map[string]bool{
	"application/pdf": true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/vnd.ms-powerpoint": true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain": true,
	"text/csv": true,
	"image/jpeg": true,
	"image/png": true,
	"image/gif": true,
	"image/tiff": true,
}

// ValidateCreateDocumentRequest validates a document creation request
func ValidateCreateDocumentRequest(request *dto.CreateDocumentRequest) error {
	if request == nil {
		return errors.NewValidationError("request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate document name
	if err := validateDocumentName(request.Name); err != nil {
		return err
	}

	// Validate folder ID
	if err := validator.ValidateUUID(request.FolderID); err != nil {
		return errors.NewValidationError("invalid folder ID: " + err.Error())
	}

	// Validate file
	if request.File == nil {
		return errors.NewValidationError("file is required")
	}

	// Validate file size
	if err := validateFileSize(request.File.Size); err != nil {
		return err
	}

	// Validate content type
	contentType := request.File.Header.Get("Content-Type")
	if err := validateContentType(contentType); err != nil {
		return err
	}

	// Validate metadata
	if len(request.Metadata) > 0 {
		if err := validateMetadata(request.Metadata); err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdateDocumentRequest validates a document update request
func ValidateUpdateDocumentRequest(request *dto.UpdateDocumentRequest) error {
	if request == nil {
		return errors.NewValidationError("request cannot be nil")
	}

	// Check if at least one field is provided for update
	if request.Name == "" && request.FolderID == "" && len(request.Metadata) == 0 &&
		len(request.AddTags) == 0 && len(request.RemoveTags) == 0 {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	// Validate document name if provided
	if request.Name != "" {
		if err := validateDocumentName(request.Name); err != nil {
			return err
		}
	}

	// Validate folder ID if provided
	if request.FolderID != "" {
		if err := validator.ValidateUUID(request.FolderID); err != nil {
			return errors.NewValidationError("invalid folder ID: " + err.Error())
		}
	}

	// Validate metadata if provided
	if len(request.Metadata) > 0 {
		if err := validateMetadata(request.Metadata); err != nil {
			return err
		}
	}

	// Validate tags if provided
	if len(request.AddTags) > 0 {
		for _, tag := range request.AddTags {
			if tag == "" {
				return errors.NewValidationError("tag name cannot be empty")
			}
			if len(tag) > 50 {
				return errors.NewValidationError("tag name cannot exceed 50 characters")
			}
		}
	}

	if len(request.RemoveTags) > 0 {
		for _, tag := range request.RemoveTags {
			if tag == "" {
				return errors.NewValidationError("tag ID cannot be empty")
			}
			if err := validator.ValidateUUID(tag); err != nil {
				return errors.NewValidationError("invalid tag ID: " + err.Error())
			}
		}
	}

	return nil
}

// ValidateBatchDownloadRequest validates a batch document download request
func ValidateBatchDownloadRequest(request *dto.BatchDownloadRequest) error {
	if request == nil {
		return errors.NewValidationError("request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Check if DocumentIDs is not empty
	if len(request.DocumentIDs) == 0 {
		return errors.NewValidationError("document IDs are required")
	}

	// Check if DocumentIDs count is within MaxBatchDownloadCount limit
	if len(request.DocumentIDs) > MaxBatchDownloadCount {
		return errors.NewValidationError(fmt.Sprintf("maximum of %d documents can be downloaded in a batch", MaxBatchDownloadCount))
	}

	// Validate each document ID
	for _, id := range request.DocumentIDs {
		if err := validator.ValidateUUID(id); err != nil {
			return errors.NewValidationError("invalid document ID: " + err.Error())
		}
	}

	// Validate archive name if provided
	if request.ArchiveName != "" {
		if err := validateDocumentName(request.ArchiveName); err != nil {
			return errors.NewValidationError("invalid archive name: " + err.Error())
		}
	}

	return nil
}

// ValidateDocumentListRequest validates a document listing request
func ValidateDocumentListRequest(request *dto.DocumentListRequest) error {
	if request == nil {
		return errors.NewValidationError("request cannot be nil")
	}

	// Validate the request struct using the validator package
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate FolderID
	if err := validator.ValidateUUID(request.FolderID); err != nil {
		return errors.NewValidationError("invalid folder ID: " + err.Error())
	}

	// Validate pagination parameters
	if request.Page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}

	if request.PageSize < 1 || request.PageSize > 100 {
		return errors.NewValidationError("page size must be between 1 and 100")
	}

	// Validate sort parameters
	if err := validateSortParameters(request.SortBy, request.SortOrder); err != nil {
		return err
	}

	// Validate filters if provided
	if len(request.Filters) > 0 {
		for key, value := range request.Filters {
			// Validate filter key
			if key == "" {
				return errors.NewValidationError("filter key cannot be empty")
			}

			// Validate specific filters
			switch key {
			case "content_type":
				if !AllowedContentTypes[value] {
					return errors.NewValidationError("invalid content type filter")
				}
			case "status":
				validStatus := false
				for _, status := range ValidDocumentStatuses {
					if status == value {
						validStatus = true
						break
					}
				}
				if !validStatus {
					return errors.NewValidationError("invalid status filter")
				}
			}
		}
	}

	return nil
}

// ValidateDocumentStatusRequest validates a document status check request
func ValidateDocumentStatusRequest(documentID string) error {
	// Check if documentID is not empty
	if documentID == "" {
		return errors.NewValidationError("document ID is required")
	}

	// Validate documentID format
	if err := validator.ValidateUUID(documentID); err != nil {
		return errors.NewValidationError("invalid document ID: " + err.Error())
	}

	return nil
}

// validateDocumentName validates a document name against naming rules
func validateDocumentName(name string) error {
	// Check if name is empty
	if name == "" {
		return errors.NewValidationError("document name is required")
	}

	// Check if name length is within allowed limits
	if len(name) < MinDocumentNameLength || len(name) > MaxDocumentNameLength {
		return errors.NewValidationError(fmt.Sprintf("document name must be between %d and %d characters",
			MinDocumentNameLength, MaxDocumentNameLength))
	}

	// Check if name contains invalid characters
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return errors.NewValidationError(fmt.Sprintf("document name contains invalid character: %s", char))
		}
	}

	return nil
}

// validateContentType validates that a content type is allowed
func validateContentType(contentType string) error {
	// Check if contentType is empty
	if contentType == "" {
		return errors.NewValidationError("content type is required")
	}

	// Check if contentType is in the AllowedContentTypes map
	if !AllowedContentTypes[contentType] {
		return errors.NewValidationError(fmt.Sprintf("content type '%s' is not allowed", contentType))
	}

	return nil
}

// validateFileSize validates that a file size is within allowed limits
func validateFileSize(size int64) error {
	// Check if size is greater than 0
	if size <= 0 {
		return errors.NewValidationError("file size must be greater than 0")
	}

	// Check if size is less than or equal to MaxDocumentSize
	if size > MaxDocumentSize {
		return errors.NewValidationError(fmt.Sprintf("file size cannot exceed %d bytes (%d MB)",
			MaxDocumentSize, MaxDocumentSize/(1024*1024)))
	}

	return nil
}

// validateMetadata validates document metadata
func validateMetadata(metadata map[string]string) error {
	// Check if metadata count is within MaxMetadataCount limit
	if len(metadata) > MaxMetadataCount {
		return errors.NewValidationError(fmt.Sprintf("maximum of %d metadata entries are allowed", MaxMetadataCount))
	}

	// Iterate through metadata entries
	for key, value := range metadata {
		// Validate key
		if key == "" {
			return errors.NewValidationError("metadata key cannot be empty")
		}

		if len(key) > MaxMetadataKeyLength {
			return errors.NewValidationError(fmt.Sprintf("metadata key cannot exceed %d characters", MaxMetadataKeyLength))
		}

		// Validate value
		if len(value) > MaxMetadataValueLength {
			return errors.NewValidationError(fmt.Sprintf("metadata value for key '%s' cannot exceed %d characters",
				key, MaxMetadataValueLength))
		}
	}

	return nil
}

// validateSortParameters validates sorting parameters for document listing
func validateSortParameters(sortBy, sortOrder string) error {
	// If sortBy is empty, return nil (default sorting will be used)
	if sortBy == "" {
		return nil
	}

	// Check if sortBy is in the list of ValidSortFields
	validField := false
	for _, field := range ValidSortFields {
		if sortBy == field {
			validField = true
			break
		}
	}

	if !validField {
		return errors.NewValidationError(fmt.Sprintf("invalid sort field: %s", sortBy))
	}

	// If sortOrder is empty, return nil (default order will be used)
	if sortOrder == "" {
		return nil
	}

	// Check if sortOrder is in the list of ValidSortOrders
	validOrder := false
	for _, order := range ValidSortOrders {
		if sortOrder == order {
			validOrder = true
			break
		}
	}

	if !validOrder {
		return errors.NewValidationError(fmt.Sprintf("invalid sort order: %s", sortOrder))
	}

	return nil
}