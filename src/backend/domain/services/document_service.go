// Package services provides service interfaces and implementations for the Document Management Platform.
package services

import (
	"context" // standard library
	"fmt"     // standard library
	"io"      // standard library
	"time"    // standard library

	"../models"                 // For document domain models
	"../repositories"           // For document repository interface
	"../../pkg/errors"          // For standardized error handling
	"../../pkg/logger"          // For structured logging
	"../../pkg/utils"           // For pagination utilities
)

// StorageService defines the contract for document storage operations
type StorageService interface {
	// StoreTemporary stores a document in temporary storage and returns its location
	StoreTemporary(ctx context.Context, content io.Reader, filename string, tenantID string) (string, error)
	
	// StorePermanent moves a document from temporary storage to permanent storage
	StorePermanent(ctx context.Context, tempLocation string, documentID string, versionID string, tenantID string) (string, error)
	
	// GetDocument retrieves document content from storage
	GetDocument(ctx context.Context, storagePath string) (io.ReadCloser, error)
	
	// GetPresignedURL generates a presigned URL for direct document download
	GetPresignedURL(ctx context.Context, storagePath string, expirationSeconds int) (string, error)
	
	// DeleteDocument removes a document from storage
	DeleteDocument(ctx context.Context, storagePath string) error
	
	// MoveToQuarantine moves a document from temporary storage to quarantine storage
	MoveToQuarantine(ctx context.Context, tempLocation string, documentID string, versionID string, tenantID string) (string, error)
	
	// CreateBatchArchive creates a compressed archive containing multiple documents
	CreateBatchArchive(ctx context.Context, documents map[string]string) (io.ReadCloser, error)
}

// VirusScanningService defines the contract for virus scanning operations
type VirusScanningService interface {
	// QueueForScanning queues a document for virus scanning
	QueueForScanning(ctx context.Context, documentID string, versionID string, tempLocation string, tenantID string) error
}

// SearchService defines the contract for document search operations
type SearchService interface {
	// IndexDocument indexes a document for searching
	IndexDocument(ctx context.Context, document *models.Document) error
	
	// RemoveDocumentFromIndex removes a document from the search index
	RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error
}

// EventServiceInterface defines the contract for event publishing
type EventServiceInterface interface {
	// PublishEvent publishes an event to the event bus
	PublishEvent(ctx context.Context, eventType string, payload interface{}) error
}

// DocumentService defines the contract for document management operations
type DocumentService interface {
	// UploadDocument uploads a new document to the system
	UploadDocument(ctx context.Context, document *models.Document, content io.Reader) (string, error)
	
	// GetDocument retrieves a document by its ID with tenant isolation
	GetDocument(ctx context.Context, id string, tenantID string) (*models.Document, error)
	
	// DownloadDocument downloads a document's content
	DownloadDocument(ctx context.Context, id string, tenantID string) (io.ReadCloser, string, error)
	
	// GetDocumentPresignedURL generates a presigned URL for direct document download
	GetDocumentPresignedURL(ctx context.Context, id string, tenantID string, expirationSeconds int) (string, error)
	
	// DeleteDocument deletes a document by its ID with tenant isolation
	DeleteDocument(ctx context.Context, id string, tenantID string) error
	
	// UpdateDocumentMetadata updates document metadata
	UpdateDocumentMetadata(ctx context.Context, id string, metadata map[string]string, tenantID string) error
	
	// ListDocumentsByFolder lists documents in a specific folder with pagination and tenant isolation
	ListDocumentsByFolder(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
	
	// BatchDownloadDocuments creates a compressed archive of multiple documents
	BatchDownloadDocuments(ctx context.Context, documentIDs []string, tenantID string) (io.ReadCloser, error)
	
	// ProcessDocumentScanResult processes the result of a virus scan on a document
	ProcessDocumentScanResult(ctx context.Context, documentID string, versionID string, tenantID string, isClean bool, scanDetails string) error
}

// documentService implements the DocumentService interface
type documentService struct {
	documentRepo         repositories.DocumentRepository
	storageService       StorageService
	virusScanningService VirusScanningService
	searchService        SearchService
	eventService         EventServiceInterface
	logger               *logger.Logger
}

// NewDocumentService creates a new DocumentService instance
func NewDocumentService(
	documentRepo repositories.DocumentRepository,
	storageService StorageService,
	virusScanningService VirusScanningService,
	searchService SearchService,
	eventService EventServiceInterface,
) DocumentService {
	// Validate dependencies
	if documentRepo == nil {
		panic("documentRepo is required")
	}
	if storageService == nil {
		panic("storageService is required")
	}
	if virusScanningService == nil {
		panic("virusScanningService is required")
	}
	if searchService == nil {
		panic("searchService is required")
	}
	if eventService == nil {
		panic("eventService is required")
	}

	return &documentService{
		documentRepo:         documentRepo,
		storageService:       storageService,
		virusScanningService: virusScanningService,
		searchService:        searchService,
		eventService:         eventService,
		logger:               &logger.Logger{},
	}
}

// UploadDocument uploads a new document to the system
func (s *documentService) UploadDocument(ctx context.Context, document *models.Document, content io.Reader) (string, error) {
	log := logger.WithContext(ctx)
	
	// Validate document
	if document == nil {
		return "", errors.NewValidationError("document cannot be nil")
	}
	
	if err := document.Validate(); err != nil {
		return "", errors.NewValidationError(fmt.Sprintf("invalid document: %v", err))
	}
	
	// Create document in repository to get ID
	docID, err := s.documentRepo.Create(ctx, document)
	if err != nil {
		return "", errors.Wrap(err, "failed to create document")
	}
	
	document.ID = docID
	
	// Store document content in temporary storage
	tempLocation, err := s.storageService.StoreTemporary(ctx, content, document.Name, document.TenantID)
	if err != nil {
		// Attempt to clean up the document metadata since content storage failed
		_ = s.documentRepo.Delete(ctx, docID, document.TenantID)
		return "", errors.Wrap(err, "failed to store document content")
	}
	
	// Create initial document version
	version := models.NewDocumentVersion(
		docID,
		1, // First version
		document.Size,
		"", // Content hash will be updated during processing
		tempLocation,
		document.OwnerID,
	)
	
	// Add version to document
	document.AddVersion(version)
	
	// Update document in repository
	err = s.documentRepo.Update(ctx, document)
	if err != nil {
		return "", errors.Wrap(err, "failed to update document with version")
	}
	
	// Queue document for virus scanning
	err = s.virusScanningService.QueueForScanning(ctx, docID, version.ID, tempLocation, document.TenantID)
	if err != nil {
		return "", errors.Wrap(err, "failed to queue document for virus scanning")
	}
	
	// Publish document.uploaded event
	err = s.eventService.PublishEvent(ctx, "document.uploaded", map[string]interface{}{
		"document_id": docID,
		"tenant_id":   document.TenantID,
		"owner_id":    document.OwnerID,
		"name":        document.Name,
		"size":        document.Size,
		"status":      document.Status,
	})
	if err != nil {
		log.Warn("failed to publish document.uploaded event", "error", err.Error())
	}
	
	log.Info("document uploaded successfully", "document_id", docID, "tenant_id", document.TenantID)
	
	return docID, nil
}

// GetDocument retrieves a document by its ID with tenant isolation
func (s *documentService) GetDocument(ctx context.Context, id string, tenantID string) (*models.Document, error) {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if id == "" {
		return nil, errors.NewValidationError("document ID cannot be empty")
	}
	
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}
	
	// Retrieve document from repository
	document, err := s.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve document")
	}
	
	log.Debug("document retrieved successfully", "document_id", id, "tenant_id", tenantID)
	
	return document, nil
}

// DownloadDocument downloads a document's content
func (s *documentService) DownloadDocument(ctx context.Context, id string, tenantID string) (io.ReadCloser, string, error) {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if id == "" {
		return nil, "", errors.NewValidationError("document ID cannot be empty")
	}
	
	if tenantID == "" {
		return nil, "", errors.NewValidationError("tenant ID cannot be empty")
	}
	
	// Retrieve document from repository
	document, err := s.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to retrieve document")
	}
	
	// Check if document is available for download
	if !document.IsAvailable() {
		if document.IsProcessing() {
			return nil, "", errors.NewValidationError("document is still being processed")
		} else if document.IsQuarantined() {
			return nil, "", errors.NewSecurityError("document has been quarantined due to security concerns")
		} else {
			return nil, "", errors.NewValidationError("document is not available for download")
		}
	}
	
	// Get latest document version
	version := document.GetLatestVersion()
	if version == nil {
		return nil, "", errors.NewInternalError("document has no versions")
	}
	
	// Get document content from storage
	content, err := s.storageService.GetDocument(ctx, version.StoragePath)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to retrieve document content")
	}
	
	// Publish document.downloaded event
	err = s.eventService.PublishEvent(ctx, "document.downloaded", map[string]interface{}{
		"document_id": id,
		"tenant_id":   tenantID,
		"version_id":  version.ID,
	})
	if err != nil {
		log.Warn("failed to publish document.downloaded event", "error", err.Error())
	}
	
	log.Info("document downloaded successfully", "document_id", id, "tenant_id", tenantID)
	
	return content, document.Name, nil
}

// GetDocumentPresignedURL generates a presigned URL for direct document download
func (s *documentService) GetDocumentPresignedURL(ctx context.Context, id string, tenantID string, expirationSeconds int) (string, error) {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if id == "" {
		return "", errors.NewValidationError("document ID cannot be empty")
	}
	
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID cannot be empty")
	}
	
	if expirationSeconds <= 0 {
		return "", errors.NewValidationError("expiration seconds must be greater than 0")
	}
	
	// Retrieve document from repository
	document, err := s.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve document")
	}
	
	// Check if document is available
	if !document.IsAvailable() {
		if document.IsProcessing() {
			return "", errors.NewValidationError("document is still being processed")
		} else if document.IsQuarantined() {
			return "", errors.NewSecurityError("document has been quarantined due to security concerns")
		} else {
			return "", errors.NewValidationError("document is not available for download")
		}
	}
	
	// Get latest document version
	version := document.GetLatestVersion()
	if version == nil {
		return "", errors.NewInternalError("document has no versions")
	}
	
	// Generate presigned URL
	url, err := s.storageService.GetPresignedURL(ctx, version.StoragePath, expirationSeconds)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate presigned URL")
	}
	
	// Publish document.downloaded event
	err = s.eventService.PublishEvent(ctx, "document.downloaded", map[string]interface{}{
		"document_id": id,
		"tenant_id":   tenantID,
		"version_id":  version.ID,
		"presigned":   true,
	})
	if err != nil {
		log.Warn("failed to publish document.downloaded event", "error", err.Error())
	}
	
	log.Info("presigned URL generated successfully", "document_id", id, "tenant_id", tenantID, "expiration_seconds", expirationSeconds)
	
	return url, nil
}

// DeleteDocument deletes a document by its ID with tenant isolation
func (s *documentService) DeleteDocument(ctx context.Context, id string, tenantID string) error {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if id == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}
	
	// Retrieve document from repository
	document, err := s.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve document")
	}
	
	// Delete document content for each version
	for _, version := range document.Versions {
		err = s.storageService.DeleteDocument(ctx, version.StoragePath)
		if err != nil {
			log.Warn("failed to delete document content", "document_id", id, "version_id", version.ID, "error", err.Error())
			// Continue with other versions rather than failing completely
		}
	}
	
	// Delete document metadata from repository
	err = s.documentRepo.Delete(ctx, id, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to delete document metadata")
	}
	
	// Remove document from search index
	err = s.searchService.RemoveDocumentFromIndex(ctx, id, tenantID)
	if err != nil {
		log.Warn("failed to remove document from search index", "document_id", id, "error", err.Error())
		// Continue rather than failing the delete operation
	}
	
	// Publish document.deleted event
	err = s.eventService.PublishEvent(ctx, "document.deleted", map[string]interface{}{
		"document_id": id,
		"tenant_id":   tenantID,
	})
	if err != nil {
		log.Warn("failed to publish document.deleted event", "error", err.Error())
	}
	
	log.Info("document deleted successfully", "document_id", id, "tenant_id", tenantID)
	
	return nil
}

// UpdateDocumentMetadata updates document metadata
func (s *documentService) UpdateDocumentMetadata(ctx context.Context, id string, metadata map[string]string, tenantID string) error {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if id == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}
	
	if metadata == nil || len(metadata) == 0 {
		return errors.NewValidationError("metadata cannot be empty")
	}
	
	// Retrieve document from repository
	document, err := s.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve document")
	}
	
	// Update each metadata field
	for key, value := range metadata {
		// Check if metadata already exists
		exists := false
		for i, m := range document.Metadata {
			if m.Key == key {
				// Update existing metadata
				document.Metadata[i].Update(value)
				exists = true
				
				// Update in repository
				err = s.documentRepo.UpdateMetadata(ctx, id, key, value, tenantID)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("failed to update metadata %s", key))
				}
				
				break
			}
		}
		
		if !exists {
			// Add new metadata
			document.AddMetadata(key, value)
			
			// Add in repository
			_, err = s.documentRepo.AddMetadata(ctx, id, key, value, tenantID)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to add metadata %s", key))
			}
		}
	}
	
	// Update document in repository
	err = s.documentRepo.Update(ctx, document)
	if err != nil {
		return errors.Wrap(err, "failed to update document")
	}
	
	// Update search index
	err = s.searchService.IndexDocument(ctx, document)
	if err != nil {
		log.Warn("failed to update document in search index", "document_id", id, "error", err.Error())
		// Continue rather than failing the metadata update operation
	}
	
	// Publish document.metadata_updated event
	err = s.eventService.PublishEvent(ctx, "document.metadata_updated", map[string]interface{}{
		"document_id": id,
		"tenant_id":   tenantID,
		"metadata":    metadata,
	})
	if err != nil {
		log.Warn("failed to publish document.metadata_updated event", "error", err.Error())
	}
	
	log.Info("document metadata updated successfully", "document_id", id, "tenant_id", tenantID)
	
	return nil
}

// ListDocumentsByFolder lists documents in a specific folder with pagination and tenant isolation
func (s *documentService) ListDocumentsByFolder(
	ctx context.Context,
	folderID string,
	tenantID string,
	pagination *utils.Pagination,
) (utils.PaginatedResult[models.Document], error) {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if folderID == "" {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("folder ID cannot be empty")
	}
	
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("tenant ID cannot be empty")
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Retrieve documents from repository
	result, err := s.documentRepo.ListByFolder(ctx, folderID, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to list documents")
	}
	
	log.Debug("documents listed successfully", "folder_id", folderID, "tenant_id", tenantID, "count", len(result.Items))
	
	return result, nil
}

// BatchDownloadDocuments creates a compressed archive of multiple documents
func (s *documentService) BatchDownloadDocuments(ctx context.Context, documentIDs []string, tenantID string) (io.ReadCloser, error) {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if documentIDs == nil || len(documentIDs) == 0 {
		return nil, errors.NewValidationError("document IDs cannot be empty")
	}
	
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}
	
	// Retrieve documents from repository
	documents, err := s.documentRepo.GetDocumentsByIDs(ctx, documentIDs, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve documents")
	}
	
	if len(documents) == 0 {
		return nil, errors.NewResourceNotFoundError("no documents found for the provided IDs")
	}
	
	// Check if all documents are available
	unavailableDocs := make([]string, 0)
	for _, doc := range documents {
		if !doc.IsAvailable() {
			unavailableDocs = append(unavailableDocs, doc.ID)
		}
	}
	
	if len(unavailableDocs) > 0 {
		return nil, errors.NewValidationError(fmt.Sprintf("some documents are not available for download: %v", unavailableDocs))
	}
	
	// Collect storage paths and filenames for available documents
	pathsToFilenames := make(map[string]string)
	for _, doc := range documents {
		version := doc.GetLatestVersion()
		if version != nil && version.IsAvailable() {
			pathsToFilenames[version.StoragePath] = doc.Name
		}
	}
	
	// Create compressed archive
	archive, err := s.storageService.CreateBatchArchive(ctx, pathsToFilenames)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create batch archive")
	}
	
	// Publish document.batch_downloaded event
	err = s.eventService.PublishEvent(ctx, "document.batch_downloaded", map[string]interface{}{
		"document_ids": documentIDs,
		"tenant_id":    tenantID,
		"count":        len(pathsToFilenames),
	})
	if err != nil {
		log.Warn("failed to publish document.batch_downloaded event", "error", err.Error())
	}
	
	log.Info("batch download created successfully", "document_count", len(pathsToFilenames), "tenant_id", tenantID)
	
	return archive, nil
}

// ProcessDocumentScanResult processes the result of a virus scan on a document
func (s *documentService) ProcessDocumentScanResult(
	ctx context.Context,
	documentID string,
	versionID string,
	tenantID string,
	isClean bool,
	scanDetails string,
) error {
	log := logger.WithContext(ctx)
	
	// Validate inputs
	if documentID == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	
	if versionID == "" {
		return errors.NewValidationError("version ID cannot be empty")
	}
	
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}
	
	// Retrieve document from repository
	document, err := s.documentRepo.GetByID(ctx, documentID, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve document")
	}
	
	// Find the version
	var version *models.DocumentVersion
	for i, v := range document.Versions {
		if v.ID == versionID {
			version = &document.Versions[i]
			break
		}
	}
	
	if version == nil {
		return errors.NewResourceNotFoundError(fmt.Sprintf("version %s not found for document %s", versionID, documentID))
	}
	
	// Process scan result
	if isClean {
		// Move document from temporary to permanent storage
		permanentPath, err := s.storageService.StorePermanent(ctx, version.StoragePath, documentID, versionID, tenantID)
		if err != nil {
			return errors.Wrap(err, "failed to move document to permanent storage")
		}
		
		// Update document status and storage path
		version.StoragePath = permanentPath
		version.MarkAsAvailable()
		document.MarkAsAvailable()
		
		// Update version status in repository
		err = s.documentRepo.UpdateVersionStatus(ctx, versionID, models.VersionStatusAvailable, tenantID)
		if err != nil {
			return errors.Wrap(err, "failed to update version status")
		}
		
		// Index document content for search
		err = s.searchService.IndexDocument(ctx, document)
		if err != nil {
			log.Warn("failed to index document", "document_id", documentID, "error", err.Error())
			// Continue rather than failing the process
		}
		
		// Publish document.available event
		err = s.eventService.PublishEvent(ctx, "document.available", map[string]interface{}{
			"document_id":  documentID,
			"tenant_id":    tenantID,
			"version_id":   versionID,
			"scan_details": scanDetails,
		})
		if err != nil {
			log.Warn("failed to publish document.available event", "error", err.Error())
		}
	} else {
		// Move document to quarantine storage
		quarantinePath, err := s.storageService.MoveToQuarantine(ctx, version.StoragePath, documentID, versionID, tenantID)
		if err != nil {
			return errors.Wrap(err, "failed to move document to quarantine")
		}
		
		// Update document status and storage path
		version.StoragePath = quarantinePath
		version.MarkAsQuarantined()
		document.MarkAsQuarantined()
		
		// Update version status in repository
		err = s.documentRepo.UpdateVersionStatus(ctx, versionID, models.VersionStatusQuarantined, tenantID)
		if err != nil {
			return errors.Wrap(err, "failed to update version status")
		}
		
		// Publish document.quarantined event
		err = s.eventService.PublishEvent(ctx, "document.quarantined", map[string]interface{}{
			"document_id":  documentID,
			"tenant_id":    tenantID,
			"version_id":   versionID,
			"scan_details": scanDetails,
		})
		if err != nil {
			log.Warn("failed to publish document.quarantined event", "error", err.Error())
		}
	}
	
	// Update document in repository
	err = s.documentRepo.Update(ctx, document)
	if err != nil {
		return errors.Wrap(err, "failed to update document")
	}
	
	log.Info("document scan result processed", 
		"document_id", documentID, 
		"tenant_id", tenantID, 
		"is_clean", isClean,
		"status", document.Status)
	
	return nil
}

// validateInput is a helper function to validate input parameters
func (s *documentService) validateInput(params map[string]string) error {
	for key, value := range params {
		if value == "" {
			return errors.NewValidationError(fmt.Sprintf("%s cannot be empty", key))
		}
	}
	return nil
}