// Package dto provides Data Transfer Objects for the Document Management Platform API.
// This file contains DTOs for document-related operations, including document creation, 
// retrieval, update, and listing. These DTOs serve as the interface between the API layer 
// and the domain layer, transforming domain models into client-friendly representations 
// and vice versa.
package dto

import (
	"mime/multipart" // standard library
	"time"           // standard library

	"../../domain/models"
	"../../pkg/errors"
	timeutils "../../pkg/utils/time_utils"
)

// DocumentDTO represents a document in API responses
type DocumentDTO struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	ContentType   string                `json:"content_type"`
	Size          int64                 `json:"size"`
	FolderID      string                `json:"folder_id"`
	Status        string                `json:"status"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
	CreatedBy     string                `json:"created_by"`
	Metadata      []DocumentMetadataDTO `json:"metadata,omitempty"`
	Tags          []TagDTO              `json:"tags,omitempty"`
	LatestVersion DocumentVersionDTO    `json:"latest_version,omitempty"`
}

// DocumentMetadataDTO represents document metadata in API responses
type DocumentMetadataDTO struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// DocumentVersionDTO represents a document version in API responses
type DocumentVersionDTO struct {
	ID            string `json:"id"`
	VersionNumber int    `json:"version_number"`
	Size          int64  `json:"size"`
	ContentHash   string `json:"content_hash"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	CreatedBy     string `json:"created_by"`
}

// TagDTO represents a tag in API responses
type TagDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateDocumentRequest represents a request to create a new document
type CreateDocumentRequest struct {
	Name     string                `form:"name" json:"name"`
	FolderID string                `form:"folder_id" json:"folder_id"`
	File     *multipart.FileHeader `form:"file" json:"-"`
	Metadata map[string]string     `form:"metadata" json:"metadata,omitempty"`
	Tags     []string              `form:"tags" json:"tags,omitempty"`
}

// Validate validates the create document request
func (r *CreateDocumentRequest) Validate() error {
	if r.Name == "" {
		return errors.NewValidationError("document name is required")
	}
	if r.FolderID == "" {
		return errors.NewValidationError("folder ID is required")
	}
	if r.File == nil {
		return errors.NewValidationError("file is required")
	}
	return nil
}

// UpdateDocumentRequest represents a request to update an existing document
type UpdateDocumentRequest struct {
	Name       string            `json:"name,omitempty"`
	FolderID   string            `json:"folder_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	AddTags    []string          `json:"add_tags,omitempty"`
	RemoveTags []string          `json:"remove_tags,omitempty"`
}

// Validate validates the update document request
func (r *UpdateDocumentRequest) Validate() error {
	// At least one field should be provided for update
	if r.Name == "" && r.FolderID == "" && len(r.Metadata) == 0 && len(r.AddTags) == 0 && len(r.RemoveTags) == 0 {
		return errors.NewValidationError("at least one field must be provided for update")
	}
	return nil
}

// DocumentUploadResponse represents a response to a document upload request
type DocumentUploadResponse struct {
	DocumentID string `json:"document_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// DocumentDownloadResponse represents a response to a document download request
type DocumentDownloadResponse struct {
	DocumentID  string `json:"document_id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
	ExpiresIn   int    `json:"expires_in,omitempty"` // in seconds
}

// BatchDownloadRequest represents a request to download multiple documents
type BatchDownloadRequest struct {
	DocumentIDs []string `json:"document_ids"`
	ArchiveName string   `json:"archive_name,omitempty"`
}

// Validate validates the batch download request
func (r *BatchDownloadRequest) Validate() error {
	if len(r.DocumentIDs) == 0 {
		return errors.NewValidationError("document IDs are required")
	}
	if len(r.DocumentIDs) > 100 {
		return errors.NewValidationError("maximum of 100 documents can be downloaded in a batch")
	}
	return nil
}

// BatchDownloadResponse represents a response to a batch document download request
type BatchDownloadResponse struct {
	ArchiveName   string `json:"archive_name"`
	DocumentCount int    `json:"document_count"`
	TotalSize     int64  `json:"total_size"`
	DownloadURL   string `json:"download_url"`
	ExpiresIn     int    `json:"expires_in,omitempty"` // in seconds
}

// DocumentListRequest represents a request to list documents
type DocumentListRequest struct {
	FolderID  string            `form:"folder_id" json:"folder_id"`
	Page      int               `form:"page" json:"page"`
	PageSize  int               `form:"page_size" json:"page_size"`
	SortBy    string            `form:"sort_by" json:"sort_by,omitempty"`
	SortOrder string            `form:"sort_order" json:"sort_order,omitempty"`
	Filters   map[string]string `form:"filters" json:"filters,omitempty"`
}

// Validate validates the document list request
func (r *DocumentListRequest) Validate() error {
	if r.FolderID == "" {
		return errors.NewValidationError("folder ID is required")
	}
	if r.Page < 1 {
		return errors.NewValidationError("page must be greater than 0")
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		return errors.NewValidationError("page size must be between 1 and 100")
	}
	if r.SortBy != "" && r.SortBy != "name" && r.SortBy != "created_at" && r.SortBy != "updated_at" && r.SortBy != "size" {
		return errors.NewValidationError("invalid sort_by parameter")
	}
	if r.SortOrder != "" && r.SortOrder != "asc" && r.SortOrder != "desc" {
		return errors.NewValidationError("sort_order must be 'asc' or 'desc'")
	}
	return nil
}

// DocumentStatusResponse represents a response to a document status check request
type DocumentStatusResponse struct {
	DocumentID         string `json:"document_id"`
	Status             string `json:"status"`
	Message            string `json:"message,omitempty"`
	ProcessingProgress int    `json:"processing_progress,omitempty"` // 0-100 percentage
}

// DocumentToDTO converts a domain Document model to a DocumentDTO
func DocumentToDTO(document models.Document) DocumentDTO {
	dto := DocumentDTO{
		ID:          document.ID,
		Name:        document.Name,
		ContentType: document.ContentType,
		Size:        document.Size,
		FolderID:    document.FolderID,
		Status:      document.Status,
		CreatedAt:   timeutils.FormatTimeDefault(document.CreatedAt),
		UpdatedAt:   timeutils.FormatTimeDefault(document.UpdatedAt),
		CreatedBy:   document.OwnerID,
		Metadata:    make([]DocumentMetadataDTO, 0, len(document.Metadata)),
		Tags:        make([]TagDTO, 0, len(document.Tags)),
	}

	// Convert metadata
	for _, metadata := range document.Metadata {
		dto.Metadata = append(dto.Metadata, DocumentMetadataToDTO(metadata))
	}

	// Convert tags
	for _, tag := range document.Tags {
		dto.Tags = append(dto.Tags, TagToDTO(tag))
	}

	// Add latest version if available
	latestVersion := document.GetLatestVersion()
	if latestVersion != nil {
		dto.LatestVersion = DocumentVersionToDTO(*latestVersion)
	}

	return dto
}

// DocumentsToDTOs converts a slice of domain Document models to DocumentDTOs
func DocumentsToDTOs(documents []models.Document) []DocumentDTO {
	dtos := make([]DocumentDTO, 0, len(documents))
	for _, doc := range documents {
		dtos = append(dtos, DocumentToDTO(doc))
	}
	return dtos
}

// DocumentVersionToDTO converts a domain DocumentVersion model to a DocumentVersionDTO
func DocumentVersionToDTO(version models.DocumentVersion) DocumentVersionDTO {
	return DocumentVersionDTO{
		ID:            version.ID,
		VersionNumber: version.VersionNumber,
		Size:          version.Size,
		ContentHash:   version.ContentHash,
		Status:        version.Status,
		CreatedAt:     timeutils.FormatTimeDefault(version.CreatedAt),
		CreatedBy:     version.CreatedBy,
	}
}

// DocumentMetadataToDTO converts a domain DocumentMetadata model to a DocumentMetadataDTO
func DocumentMetadataToDTO(metadata models.DocumentMetadata) DocumentMetadataDTO {
	return DocumentMetadataDTO{
		ID:        metadata.ID,
		Key:       metadata.Key,
		Value:     metadata.Value,
		CreatedAt: timeutils.FormatTimeDefault(metadata.CreatedAt),
		UpdatedAt: timeutils.FormatTimeDefault(metadata.UpdatedAt),
	}
}

// TagToDTO converts a domain Tag model to a TagDTO
func TagToDTO(tag models.Tag) TagDTO {
	return TagDTO{
		ID:   tag.ID,
		Name: tag.Name,
	}
}

// CreateDocumentRequestToModel converts a CreateDocumentRequest to a domain Document model
func CreateDocumentRequestToModel(request CreateDocumentRequest, tenantID, userID string) (models.Document, error) {
	// Create a new document with basic properties
	contentType := request.File.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Default content type if not provided
	}
	
	document := models.NewDocument(
		request.Name,
		contentType,
		request.File.Size,
		request.FolderID,
		tenantID,
		userID,
	)

	// Add metadata
	for key, value := range request.Metadata {
		document.AddMetadata(key, value)
	}

	return document, nil
}

// UpdateDocumentRequestToModel updates a domain Document model with values from an UpdateDocumentRequest
func UpdateDocumentRequestToModel(document models.Document, request UpdateDocumentRequest) error {
	// Update fields if provided
	if request.Name != "" {
		document.Name = request.Name
		document.UpdatedAt = time.Now()
	}

	if request.FolderID != "" {
		document.FolderID = request.FolderID
		document.UpdatedAt = time.Now()
	}

	// Update metadata
	if len(request.Metadata) > 0 {
		for key, value := range request.Metadata {
			document.AddMetadata(key, value)
		}
	}

	return nil
}