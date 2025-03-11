package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid" // v1.3.0
	"github.com/sirupsen/logrus" // v1.9.0

	"github.com/organization/document-management/internal/domain"
	"github.com/organization/document-management/internal/repository"
)

// DocumentService defines the interface for document management operations
type DocumentService interface {
	// UploadDocument handles the document upload process and returns a document ID
	UploadDocument(ctx context.Context, doc domain.Document, content io.Reader) (string, error)
	
	// GetDocument retrieves document metadata by ID
	GetDocument(ctx context.Context, id string) (*domain.Document, error)
	
	// DownloadDocument retrieves document content by ID
	DownloadDocument(ctx context.Context, id string) (io.ReadCloser, error)
	
	// DeleteDocument marks a document as deleted
	DeleteDocument(ctx context.Context, id string) error
	
	// ListDocuments lists documents in a folder with pagination
	ListDocuments(ctx context.Context, folderID string, page, pageSize int) ([]domain.Document, int, error)
	
	// SearchDocuments searches for documents based on criteria
	SearchDocuments(ctx context.Context, query string, filters map[string]interface{}, page, pageSize int) ([]domain.Document, int, error)
	
	// UpdateDocumentMetadata updates document metadata
	UpdateDocumentMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
	
	// ProcessDocumentCallback handles callback from virus scanning and processing
	ProcessDocumentCallback(ctx context.Context, id string, scanResult domain.ScanResult) error
}

// documentService implements DocumentService
type documentService struct {
	logger         *logrus.Logger
	docRepo        repository.DocumentRepository
	storageService repository.StorageService
	searchService  repository.SearchService
	virusScanner   repository.VirusScanService
	eventService   repository.EventService
	folderService  repository.FolderRepository
}

// NewDocumentService creates a new document service instance
func NewDocumentService(
	logger *logrus.Logger,
	docRepo repository.DocumentRepository,
	storageService repository.StorageService,
	searchService repository.SearchService,
	virusScanner repository.VirusScanService,
	eventService repository.EventService,
	folderService repository.FolderRepository,
) DocumentService {
	return &documentService{
		logger:         logger,
		docRepo:        docRepo,
		storageService: storageService,
		searchService:  searchService,
		virusScanner:   virusScanner,
		eventService:   eventService,
		folderService:  folderService,
	}
}

// UploadDocument handles the document upload process
func (s *documentService) UploadDocument(ctx context.Context, doc domain.Document, content io.Reader) (string, error) {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return "", domain.ErrUnauthorized
	}
	
	// Extract user ID from context
	userID, err := domain.ExtractUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract user ID from context")
		return "", domain.ErrUnauthorized
	}
	
	// Validate document
	if err := s.validateDocument(ctx, doc); err != nil {
		s.logger.WithError(err).WithField("document", doc.Name).Error("Document validation failed")
		return "", err
	}
	
	// Set document metadata
	doc.ID = uuid.New().String()
	doc.TenantID = tenantID
	doc.OwnerID = userID
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	doc.Status = domain.DocumentStatusProcessing
	
	s.logger.WithFields(logrus.Fields{
		"document_id": doc.ID,
		"tenant_id":   doc.TenantID,
		"folder_id":   doc.FolderID,
		"size":        doc.Size,
	}).Info("Uploading document")
	
	// Store document in temporary location
	tempLocation, err := s.storageService.StoreTemporary(ctx, doc.ID, content)
	if err != nil {
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to store document in temporary location")
		return "", fmt.Errorf("failed to store document: %w", err)
	}
	
	// Create a document version
	version := domain.DocumentVersion{
		ID:            uuid.New().String(),
		DocumentID:    doc.ID,
		VersionNumber: 1,
		Size:          doc.Size,
		Status:        domain.DocumentStatusProcessing,
		StoragePath:   tempLocation,
		CreatedAt:     time.Now(),
		CreatedBy:     userID,
	}
	
	// Set current version reference
	doc.CurrentVersionID = version.ID
	
	// Start a transaction
	tx, err := s.docRepo.BeginTransaction(ctx)
	if err != nil {
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to begin transaction")
		return "", fmt.Errorf("database error: %w", err)
	}
	
	// Save document metadata
	if err := s.docRepo.CreateWithTx(ctx, tx, doc); err != nil {
		_ = s.docRepo.RollbackTransaction(ctx, tx)
		_ = s.storageService.Delete(ctx, tempLocation)
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to save document metadata")
		return "", fmt.Errorf("failed to save document metadata: %w", err)
	}
	
	// Save document version
	if err := s.docRepo.CreateVersionWithTx(ctx, tx, version); err != nil {
		_ = s.docRepo.RollbackTransaction(ctx, tx)
		_ = s.storageService.Delete(ctx, tempLocation)
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to save document version")
		return "", fmt.Errorf("failed to save document version: %w", err)
	}
	
	// Commit transaction
	if err := s.docRepo.CommitTransaction(ctx, tx); err != nil {
		_ = s.storageService.Delete(ctx, tempLocation)
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to commit transaction")
		return "", fmt.Errorf("database error: %w", err)
	}
	
	// Queue document for virus scanning asynchronously
	if err := s.virusScanner.QueueForScanning(ctx, doc.ID, tempLocation); err != nil {
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to queue document for virus scanning")
		// We don't return an error here since the document is saved and can be processed later
	}
	
	// Publish document.uploaded event
	eventData := map[string]interface{}{
		"document_id": doc.ID,
		"tenant_id":   doc.TenantID,
		"folder_id":   doc.FolderID,
		"name":        doc.Name,
		"size":        doc.Size,
		"status":      doc.Status,
		"created_by":  doc.OwnerID,
	}
	if err := s.eventService.PublishEvent(ctx, domain.EventDocumentUploaded, eventData); err != nil {
		s.logger.WithError(err).WithField("document_id", doc.ID).Error("Failed to publish document.uploaded event")
		// We don't return an error here to not affect the upload success
	}
	
	s.logger.WithField("document_id", doc.ID).Info("Document upload accepted for processing")
	return doc.ID, nil
}

// GetDocument retrieves document metadata
func (s *documentService) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return nil, domain.ErrUnauthorized
	}
	
	s.logger.WithFields(logrus.Fields{
		"document_id": id,
		"tenant_id":   tenantID,
	}).Debug("Getting document metadata")
	
	// Get document from repository
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			s.logger.WithField("document_id", id).Debug("Document not found")
		} else {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document")
		}
		return nil, err
	}
	
	// Verify tenant access
	if doc.TenantID != tenantID {
		s.logger.WithFields(logrus.Fields{
			"document_id":   id,
			"tenant_id":     tenantID,
			"doc_tenant_id": doc.TenantID,
		}).Warn("Tenant access denied to document")
		return nil, domain.ErrForbidden
	}
	
	// Check if document is deleted
	if doc.Status == domain.DocumentStatusDeleted {
		s.logger.WithField("document_id", id).Debug("Attempted to access deleted document")
		return nil, domain.ErrDocumentNotFound
	}
	
	// Get document metadata
	metadata, err := s.docRepo.GetDocumentMetadata(ctx, id)
	if err != nil && !errors.Is(err, domain.ErrMetadataNotFound) {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document metadata")
		// Continue without metadata
	} else {
		doc.Metadata = metadata
	}
	
	return doc, nil
}

// DownloadDocument retrieves document content
func (s *documentService) DownloadDocument(ctx context.Context, id string) (io.ReadCloser, error) {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return nil, domain.ErrUnauthorized
	}
	
	// Extract user ID from context
	userID, err := domain.ExtractUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract user ID from context")
		return nil, domain.ErrUnauthorized
	}
	
	s.logger.WithFields(logrus.Fields{
		"document_id": id,
		"tenant_id":   tenantID,
		"user_id":     userID,
	}).Info("Downloading document")
	
	// Get document from repository
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			s.logger.WithField("document_id", id).Debug("Document not found for download")
		} else {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document for download")
		}
		return nil, err
	}
	
	// Verify tenant access
	if doc.TenantID != tenantID {
		s.logger.WithFields(logrus.Fields{
			"document_id":   id,
			"tenant_id":     tenantID,
			"doc_tenant_id": doc.TenantID,
		}).Warn("Tenant access denied for document download")
		return nil, domain.ErrForbidden
	}
	
	// Check document status
	if doc.Status != domain.DocumentStatusAvailable {
		s.logger.WithFields(logrus.Fields{
			"document_id": id,
			"status":      doc.Status,
		}).Warn("Document is not available for download")
		return nil, domain.ErrDocumentNotAvailable
	}
	
	// Get the current version
	version, err := s.docRepo.GetVersionByID(ctx, doc.CurrentVersionID)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"document_id": id,
			"version_id":  doc.CurrentVersionID,
		}).Error("Failed to get document version")
		return nil, err
	}
	
	// Get document content from storage
	content, err := s.storageService.GetContent(ctx, version.StoragePath)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"document_id": id,
			"storage_path": version.StoragePath,
		}).Error("Failed to get document content from storage")
		return nil, fmt.Errorf("failed to retrieve document content: %w", err)
	}
	
	// Publish document.downloaded event asynchronously
	go func() {
		eventCtx := context.Background()
		eventData := map[string]interface{}{
			"document_id": doc.ID,
			"tenant_id":   doc.TenantID,
			"user_id":     userID,
			"version_id":  doc.CurrentVersionID,
		}
		if err := s.eventService.PublishEvent(eventCtx, domain.EventDocumentDownloaded, eventData); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to publish document.downloaded event")
		}
	}()
	
	s.logger.WithField("document_id", id).Info("Document download successful")
	return content, nil
}

// DeleteDocument marks a document as deleted
func (s *documentService) DeleteDocument(ctx context.Context, id string) error {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return domain.ErrUnauthorized
	}
	
	// Extract user ID from context
	userID, err := domain.ExtractUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract user ID from context")
		return domain.ErrUnauthorized
	}
	
	s.logger.WithFields(logrus.Fields{
		"document_id": id,
		"tenant_id":   tenantID,
		"user_id":     userID,
	}).Info("Deleting document")
	
	// Get document from repository
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			s.logger.WithField("document_id", id).Debug("Document not found for deletion")
		} else {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document for deletion")
		}
		return err
	}
	
	// Verify tenant access
	if doc.TenantID != tenantID {
		s.logger.WithFields(logrus.Fields{
			"document_id":   id,
			"tenant_id":     tenantID,
			"doc_tenant_id": doc.TenantID,
		}).Warn("Tenant access denied for document deletion")
		return domain.ErrForbidden
	}
	
	// Check if document is already deleted
	if doc.Status == domain.DocumentStatusDeleted {
		s.logger.WithField("document_id", id).Debug("Document already deleted")
		return nil
	}
	
	// Mark document as deleted
	doc.Status = domain.DocumentStatusDeleted
	doc.UpdatedAt = time.Now()
	doc.DeletedAt = &time.Time{}
	*doc.DeletedAt = time.Now()
	
	if err := s.docRepo.Update(ctx, *doc); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to mark document as deleted")
		return fmt.Errorf("failed to delete document: %w", err)
	}
	
	// Get all document versions
	versions, err := s.docRepo.GetVersionsByDocumentID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document versions for deletion")
		// Continue with deletion even if we can't get versions
	} else {
		// Mark all versions for deletion in storage
		for _, version := range versions {
			if err := s.storageService.MarkDeleted(ctx, version.StoragePath); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"document_id": id,
					"version_id":  version.ID,
					"storage_path": version.StoragePath,
				}).Error("Failed to mark document version as deleted in storage")
				// Continue with other versions
			}
		}
	}
	
	// Remove from search index
	if err := s.searchService.RemoveFromIndex(ctx, id); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to remove document from search index")
		// Continue with deletion even if index removal fails
		// TODO: implement retry mechanism for index removal
	}
	
	// Publish document.deleted event
	eventData := map[string]interface{}{
		"document_id": doc.ID,
		"tenant_id":   doc.TenantID,
		"folder_id":   doc.FolderID,
		"deleted_by":  userID,
	}
	if err := s.eventService.PublishEvent(ctx, domain.EventDocumentDeleted, eventData); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to publish document.deleted event")
		// Continue with deletion even if event publishing fails
	}
	
	s.logger.WithField("document_id", id).Info("Document successfully deleted")
	return nil
}

// ListDocuments lists documents in a folder with pagination
func (s *documentService) ListDocuments(ctx context.Context, folderID string, page, pageSize int) ([]domain.Document, int, error) {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return nil, 0, domain.ErrUnauthorized
	}
	
	// Validate folder access
	if folderID != "" {
		folder, err := s.folderService.GetByID(ctx, folderID)
		if err != nil {
			if errors.Is(err, domain.ErrFolderNotFound) {
				s.logger.WithField("folder_id", folderID).Debug("Folder not found for listing documents")
				return nil, 0, domain.ErrFolderNotFound
			}
			s.logger.WithError(err).WithField("folder_id", folderID).Error("Failed to get folder for listing documents")
			return nil, 0, err
		}
		
		// Verify tenant access
		if folder.TenantID != tenantID {
			s.logger.WithFields(logrus.Fields{
				"folder_id":       folderID,
				"tenant_id":       tenantID,
				"folder_tenant_id": folder.TenantID,
			}).Warn("Tenant access denied to folder")
			return nil, 0, domain.ErrForbidden
		}
	}
	
	// Default pagination values
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	
	s.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"folder_id": folderID,
		"page":      page,
		"page_size": pageSize,
	}).Debug("Listing documents")
	
	// Get documents from repository
	docs, total, err := s.docRepo.ListByFolder(ctx, tenantID, folderID, page, pageSize)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"folder_id": folderID,
		}).Error("Failed to list documents")
		return nil, 0, fmt.Errorf("failed to list documents: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"folder_id": folderID,
		"count":     len(docs),
		"total":     total,
	}).Debug("Documents listed successfully")
	
	return docs, total, nil
}

// SearchDocuments searches for documents based on criteria
func (s *documentService) SearchDocuments(ctx context.Context, query string, filters map[string]interface{}, page, pageSize int) ([]domain.Document, int, error) {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return nil, 0, domain.ErrUnauthorized
	}
	
	// Default pagination values
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	
	// Ensure tenant filter is always applied
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["tenant_id"] = tenantID
	
	// Exclude deleted documents
	filters["status_ne"] = domain.DocumentStatusDeleted
	
	s.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"query":     query,
		"filters":   filters,
		"page":      page,
		"page_size": pageSize,
	}).Debug("Searching documents")
	
	// Search for documents
	docIDs, total, err := s.searchService.Search(ctx, query, filters, page, pageSize)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"query":     query,
		}).Error("Failed to search documents")
		return nil, 0, fmt.Errorf("search failed: %w", err)
	}
	
	if len(docIDs) == 0 {
		s.logger.WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"query":     query,
		}).Debug("No documents found in search")
		return []domain.Document{}, 0, nil
	}
	
	// Get documents by IDs
	docs, err := s.docRepo.GetByIDs(ctx, docIDs)
	if err != nil {
		s.logger.WithError(err).WithField("doc_ids", docIDs).Error("Failed to get documents by IDs")
		return nil, 0, fmt.Errorf("failed to retrieve search results: %w", err)
	}
	
	// Filter out documents from other tenants (extra safety check)
	var filteredDocs []domain.Document
	for _, doc := range docs {
		if doc.TenantID == tenantID && doc.Status != domain.DocumentStatusDeleted {
			filteredDocs = append(filteredDocs, doc)
		}
	}
	
	s.logger.WithFields(logrus.Fields{
		"tenant_id":      tenantID,
		"query":          query,
		"found_count":    len(filteredDocs),
		"original_count": len(docIDs),
		"total":          total,
	}).Debug("Documents searched successfully")
	
	return filteredDocs, total, nil
}

// UpdateDocumentMetadata updates document metadata
func (s *documentService) UpdateDocumentMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	// Extract tenant context from context
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract tenant ID from context")
		return domain.ErrUnauthorized
	}
	
	// Extract user ID from context
	userID, err := domain.ExtractUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to extract user ID from context")
		return domain.ErrUnauthorized
	}
	
	s.logger.WithFields(logrus.Fields{
		"document_id": id,
		"tenant_id":   tenantID,
		"user_id":     userID,
	}).Info("Updating document metadata")
	
	// Get document from repository
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrDocumentNotFound) {
			s.logger.WithField("document_id", id).Debug("Document not found for metadata update")
		} else {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document for metadata update")
		}
		return err
	}
	
	// Verify tenant access
	if doc.TenantID != tenantID {
		s.logger.WithFields(logrus.Fields{
			"document_id":   id,
			"tenant_id":     tenantID,
			"doc_tenant_id": doc.TenantID,
		}).Warn("Tenant access denied for document metadata update")
		return domain.ErrForbidden
	}
	
	// Check if document is deleted
	if doc.Status == domain.DocumentStatusDeleted {
		s.logger.WithField("document_id", id).Warn("Attempted to update metadata of deleted document")
		return domain.ErrDocumentNotFound
	}
	
	// Validate metadata
	if err := s.validateMetadata(metadata); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Metadata validation failed")
		return err
	}
	
	// Update document metadata
	if err := s.docRepo.UpdateMetadata(ctx, id, metadata); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document metadata")
		return fmt.Errorf("failed to update metadata: %w", err)
	}
	
	// Update document updated timestamp
	doc.UpdatedAt = time.Now()
	if err := s.docRepo.Update(ctx, *doc); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document timestamp")
		// Continue even if timestamp update fails
	}
	
	// Update search index
	if err := s.searchService.UpdateIndex(ctx, id, metadata); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to update search index")
		// Continue even if index update fails
		// TODO: implement retry mechanism for index updates
	}
	
	// Publish document.updated event
	eventData := map[string]interface{}{
		"document_id": doc.ID,
		"tenant_id":   doc.TenantID,
		"updated_by":  userID,
		"metadata":    metadata,
	}
	if err := s.eventService.PublishEvent(ctx, domain.EventDocumentUpdated, eventData); err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to publish document.updated event")
		// Continue even if event publishing fails
	}
	
	s.logger.WithField("document_id", id).Info("Document metadata updated successfully")
	return nil
}

// ProcessDocumentCallback handles callback from virus scanning and processing
func (s *documentService) ProcessDocumentCallback(ctx context.Context, id string, scanResult domain.ScanResult) error {
	s.logger.WithFields(logrus.Fields{
		"document_id": id,
		"scan_result": scanResult,
	}).Info("Processing document scan result")
	
	// Get document from repository
	doc, err := s.docRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("document_id", id).Error("Failed to get document for processing callback")
		return err
	}
	
	// Get current version
	version, err := s.docRepo.GetVersionByID(ctx, doc.CurrentVersionID)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"document_id": id,
			"version_id":  doc.CurrentVersionID,
		}).Error("Failed to get document version for processing callback")
		return err
	}
	
	// Process based on scan result
	switch scanResult.Status {
	case domain.ScanStatusClean:
		s.logger.WithField("document_id", id).Info("Document passed virus scan - processing")
		
		// Move document to permanent storage
		permanentPath, err := s.storageService.MoveToPermanent(ctx, version.StoragePath, doc.TenantID, doc.FolderID, id)
		if err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to move document to permanent storage")
			return fmt.Errorf("storage error: %w", err)
		}
		
		// Update version storage path
		version.StoragePath = permanentPath
		version.Status = domain.DocumentStatusAvailable
		
		if err := s.docRepo.UpdateVersion(ctx, version); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document version")
			return fmt.Errorf("database error: %w", err)
		}
		
		// Update document status
		doc.Status = domain.DocumentStatusAvailable
		doc.UpdatedAt = time.Now()
		
		if err := s.docRepo.Update(ctx, *doc); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document status")
			return fmt.Errorf("database error: %w", err)
		}
		
		// Index document content
		if err := s.searchService.IndexDocument(ctx, *doc, permanentPath); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to index document content")
			// Continue even if indexing fails
			// TODO: implement retry mechanism for indexing
		}
		
		// Generate thumbnail if applicable
		// TODO: Add thumbnail generation logic
		
		// Publish document.available event
		eventData := map[string]interface{}{
			"document_id": doc.ID,
			"tenant_id":   doc.TenantID,
			"status":      doc.Status,
		}
		if err := s.eventService.PublishEvent(ctx, domain.EventDocumentAvailable, eventData); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to publish document.available event")
			// Continue even if event publishing fails
		}
		
	case domain.ScanStatusInfected:
		s.logger.WithFields(logrus.Fields{
			"document_id": id,
			"threat_name": scanResult.ThreatName,
		}).Warn("Document failed virus scan - quarantining")
		
		// Move document to quarantine
		quarantinePath, err := s.storageService.MoveToQuarantine(ctx, version.StoragePath, doc.TenantID, id)
		if err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to move document to quarantine")
			return fmt.Errorf("storage error: %w", err)
		}
		
		// Update version storage path and status
		version.StoragePath = quarantinePath
		version.Status = domain.DocumentStatusQuarantined
		
		if err := s.docRepo.UpdateVersion(ctx, version); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document version for quarantine")
			return fmt.Errorf("database error: %w", err)
		}
		
		// Update document status
		doc.Status = domain.DocumentStatusQuarantined
		doc.UpdatedAt = time.Now()
		
		if err := s.docRepo.Update(ctx, *doc); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document status for quarantine")
			return fmt.Errorf("database error: %w", err)
		}
		
		// Save scan result as metadata
		scanMetadata := map[string]interface{}{
			"virus_scan_result": scanResult.Status,
			"threat_name":       scanResult.ThreatName,
			"scan_time":         time.Now().Format(time.RFC3339),
		}
		if err := s.docRepo.UpdateMetadata(ctx, id, scanMetadata); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to save scan result metadata")
			// Continue even if metadata update fails
		}
		
		// Publish document.quarantined event
		eventData := map[string]interface{}{
			"document_id": doc.ID,
			"tenant_id":   doc.TenantID,
			"status":      doc.Status,
			"threat_name": scanResult.ThreatName,
		}
		if err := s.eventService.PublishEvent(ctx, domain.EventDocumentQuarantined, eventData); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to publish document.quarantined event")
			// Continue even if event publishing fails
		}
		
	case domain.ScanStatusError:
		s.logger.WithFields(logrus.Fields{
			"document_id": id,
			"error":       scanResult.Error,
		}).Error("Document scanning failed")
		
		// Update version status
		version.Status = domain.DocumentStatusFailed
		
		if err := s.docRepo.UpdateVersion(ctx, version); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document version for failure")
			return fmt.Errorf("database error: %w", err)
		}
		
		// Update document status
		doc.Status = domain.DocumentStatusFailed
		doc.UpdatedAt = time.Now()
		
		if err := s.docRepo.Update(ctx, *doc); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to update document status for failure")
			return fmt.Errorf("database error: %w", err)
		}
		
		// Save error as metadata
		errorMetadata := map[string]interface{}{
			"processing_error": scanResult.Error,
			"error_time":       time.Now().Format(time.RFC3339),
		}
		if err := s.docRepo.UpdateMetadata(ctx, id, errorMetadata); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to save error metadata")
			// Continue even if metadata update fails
		}
		
		// Publish document.processing_failed event
		eventData := map[string]interface{}{
			"document_id": doc.ID,
			"tenant_id":   doc.TenantID,
			"status":      doc.Status,
			"error":       scanResult.Error,
		}
		if err := s.eventService.PublishEvent(ctx, domain.EventDocumentProcessingFailed, eventData); err != nil {
			s.logger.WithError(err).WithField("document_id", id).Error("Failed to publish document.processing_failed event")
			// Continue even if event publishing fails
		}
		
	default:
		s.logger.WithFields(logrus.Fields{
			"document_id": id,
			"scan_result": scanResult,
		}).Error("Unknown scan result status")
		return fmt.Errorf("unknown scan result status: %s", scanResult.Status)
	}
	
	s.logger.WithFields(logrus.Fields{
		"document_id": id,
		"status":      doc.Status,
	}).Info("Document processing completed")
	return nil
}

// validateDocument validates the document properties
func (s *documentService) validateDocument(ctx context.Context, doc domain.Document) error {
	if doc.Name == "" {
		return domain.ErrInvalidInput("document name is required")
	}
	
	if doc.Size <= 0 {
		return domain.ErrInvalidInput("document size must be greater than 0")
	}
	
	if doc.ContentType == "" {
		return domain.ErrInvalidInput("content type is required")
	}
	
	if doc.FolderID == "" {
		return domain.ErrInvalidInput("folder ID is required")
	}
	
	// Validate document size limit (100MB)
	const maxSize = 100 * 1024 * 1024 // 100MB
	if doc.Size > maxSize {
		return domain.ErrInvalidInput(fmt.Sprintf("document size exceeds the maximum limit of %d bytes", maxSize))
	}
	
	// Validate folder exists and belongs to tenant
	tenantID, err := domain.ExtractTenantIDFromContext(ctx)
	if err != nil {
		return domain.ErrUnauthorized
	}
	
	folder, err := s.folderService.GetByID(ctx, doc.FolderID)
	if err != nil {
		if errors.Is(err, domain.ErrFolderNotFound) {
			return domain.ErrInvalidInput("folder not found")
		}
		return fmt.Errorf("failed to validate folder: %w", err)
	}
	
	if folder.TenantID != tenantID {
		return domain.ErrForbidden
	}
	
	return nil
}

// validateMetadata validates document metadata
func (s *documentService) validateMetadata(metadata map[string]interface{}) error {
	if len(metadata) > 100 {
		return domain.ErrInvalidInput("too many metadata fields, maximum is 100")
	}
	
	for key, value := range metadata {
		if key == "" {
			return domain.ErrInvalidInput("metadata key cannot be empty")
		}
		
		if len(key) > 255 {
			return domain.ErrInvalidInput(fmt.Sprintf("metadata key '%s' is too long, maximum is 255 characters", key))
		}
		
		// Check the value based on its type
		switch v := value.(type) {
		case string:
			if len(v) > 1024 {
				return domain.ErrInvalidInput(fmt.Sprintf("metadata value for key '%s' is too long, maximum is 1024 characters", key))
			}
		case float64, int, int64, float32:
			// Numeric values are allowed
		case bool:
			// Boolean values are allowed
		case nil:
			// Null values are allowed
		default:
			return domain.ErrInvalidInput(fmt.Sprintf("unsupported metadata value type for key '%s'", key))
		}
	}
	
	return nil
}