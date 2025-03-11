// Package usecases implements the business logic for the Document Management Platform.
package usecases

import (
	"context" // standard library
	"fmt"    // standard library
	"io"      // standard library
	"strings" // standard library

	"time"

	"github.com/google/uuid"

	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// Global error variables for document use cases
var (
	ErrDocumentNotFound     = errors.NewResourceNotFoundError("document not found")
	ErrInvalidDocumentID    = errors.NewValidationError("invalid document ID")
	ErrInvalidTenantID      = errors.NewValidationError("invalid tenant ID")
	ErrInvalidUserID        = errors.NewValidationError("invalid user ID")
	ErrInvalidFolderID      = errors.NewValidationError("invalid folder ID")
	ErrEmptyContent         = errors.NewValidationError("document content cannot be empty")
	ErrDocumentNotAvailable = errors.NewValidationError("document is not available for download")
	ErrPermissionDenied     = errors.NewAuthorizationError("permission denied for document operation")
)

// Global event type constants for document events
const (
	DocumentEventUploaded    = "document.uploaded"
	DocumentEventProcessed   = "document.processed"
	DocumentEventDownloaded  = "document.downloaded"
	DocumentEventDeleted     = "document.deleted"
	DocumentEventQuarantined = "document.quarantined"
)

// DocumentUseCase defines the contract for document use cases
type DocumentUseCase interface {
	// UploadDocument uploads a new document to the system
	UploadDocument(ctx context.Context, name string, contentType string, size int64, folderID string, tenantID string, userID string, content io.Reader, metadata map[string]string) (string, error)

	// GetDocument retrieves a document by its ID with tenant isolation and permission checks
	GetDocument(ctx context.Context, id string, tenantID string, userID string) (*models.Document, error)

	// DownloadDocument downloads a document by its ID with tenant isolation and permission checks
	DownloadDocument(ctx context.Context, id string, tenantID string, userID string) (io.ReadCloser, string, error)

	// GetDocumentPresignedURL generates a presigned URL for document download with tenant isolation and permission checks
	GetDocumentPresignedURL(ctx context.Context, id string, tenantID string, userID string, expirationSeconds int) (string, error)

	// BatchDownloadDocuments downloads multiple documents as a compressed archive with tenant isolation and permission checks
	BatchDownloadDocuments(ctx context.Context, ids []string, tenantID string, userID string) (io.ReadCloser, error)

	// GetBatchDownloadPresignedURL generates a presigned URL for batch document download with tenant isolation and permission checks
	GetBatchDownloadPresignedURL(ctx context.Context, ids []string, tenantID string, userID string, expirationSeconds int) (string, error)

	// DeleteDocument deletes a document by its ID with tenant isolation and permission checks
	DeleteDocument(ctx context.Context, id string, tenantID string, userID string) error

	// ListDocumentsByFolder lists documents in a folder with pagination, tenant isolation, and permission checks
	ListDocumentsByFolder(ctx context.Context, folderID string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// SearchDocumentsByContent searches documents by their content with tenant isolation and permission checks
	SearchDocumentsByContent(ctx context.Context, query string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// SearchDocumentsByMetadata searches documents by their metadata with tenant isolation and permission checks
	SearchDocumentsByMetadata(ctx context.Context, metadata map[string]string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// CombinedSearch performs a search using both content and metadata criteria with tenant isolation and permission checks
	CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// UpdateDocumentMetadata updates document metadata with tenant isolation and permission checks
	UpdateDocumentMetadata(ctx context.Context, id string, key string, value string, tenantID string, userID string) error

	// DeleteDocumentMetadata deletes document metadata with tenant isolation and permission checks
	DeleteDocumentMetadata(ctx context.Context, id string, key string, tenantID string, userID string) error

	// GetDocumentThumbnail retrieves a document thumbnail with tenant isolation and permission checks
	GetDocumentThumbnail(ctx context.Context, id string, tenantID string, userID string) (io.ReadCloser, error)

	// GetDocumentThumbnailURL generates a presigned URL for document thumbnail with tenant isolation and permission checks
	GetDocumentThumbnailURL(ctx context.Context, id string, tenantID string, userID string, expirationSeconds int) (string, error)

	// GetDocumentStatus gets the current status of a document with tenant isolation and permission checks
	GetDocumentStatus(ctx context.Context, id string, tenantID string, userID string) (string, error)
}

// documentUseCase implements the DocumentUseCase interface
type documentUseCase struct {
	documentRepo      repositories.DocumentRepository
	storageService    services.StorageService
	virusScanningService services.VirusScanningService
	searchService     services.SearchService
	folderService     services.FolderService
	eventService      services.EventServiceInterface
	authService       services.AuthService
	thumbnailService  services.ThumbnailService
	logger            *logger.Logger
}

// NewDocumentUseCase creates a new DocumentUseCase instance
func NewDocumentUseCase(
	documentRepo repositories.DocumentRepository,
	storageService services.StorageService,
	virusScanningService services.VirusScanningService,
	searchService services.SearchService,
	folderService services.FolderService,
	eventService services.EventServiceInterface,
	authService services.AuthService,
	thumbnailService services.ThumbnailService,
) (DocumentUseCase, error) {
	// Validate that documentRepo is not nil
	if documentRepo == nil {
		return nil, fmt.Errorf("documentRepo cannot be nil")
	}

	// Validate that storageService is not nil
	if storageService == nil {
		return nil, fmt.Errorf("storageService cannot be nil")
	}

	// Validate that virusScanningService is not nil
	if virusScanningService == nil {
		return nil, fmt.Errorf("virusScanningService cannot be nil")
	}

	// Validate that searchService is not nil
	if searchService == nil {
		return nil, fmt.Errorf("searchService cannot be nil")
	}

	// Validate that folderService is not nil
	if folderService == nil {
		return nil, fmt.Errorf("folderService cannot be nil")
	}

	// Validate that eventService is not nil
	if eventService == nil {
		return nil, fmt.Errorf("eventService cannot be nil")
	}

	if authService == nil {
		return nil, fmt.Errorf("authService cannot be nil")
	}

	if thumbnailService == nil {
		return nil, fmt.Errorf("thumbnailService cannot be nil")
	}

	// Create and return a new documentUseCase with the provided dependencies
	return &documentUseCase{
		documentRepo:      documentRepo,
		storageService:    storageService,
		virusScanningService: virusScanningService,
		searchService:     searchService,
		folderService:     folderService,
		eventService:      eventService,
		authService:       authService,
		thumbnailService:  thumbnailService,
		logger:            logger.WithField("usecase", "document"),
	}, nil
}

// UploadDocument uploads a new document to the system
func (uc *documentUseCase) UploadDocument(ctx context.Context, name string, contentType string, size int64, folderID string, tenantID string, userID string, content io.Reader, metadata map[string]string) (string, error) {
	// Get logger with context
	log := uc.logger.WithContext(ctx)

	// Validate name is not empty
	if strings.TrimSpace(name) == "" {
		log.Error("Document name cannot be empty")
		return "", errors.NewValidationError("document name is required")
	}

	// Validate contentType is not empty
	if strings.TrimSpace(contentType) == "" {
		log.Error("Content type cannot be empty")
		return "", errors.NewValidationError("content type is required")
	}

	// Validate size is greater than 0
	if size <= 0 {
		log.Error("Document size must be greater than 0")
		return "", errors.NewValidationError("document size must be greater than 0")
	}

	// Validate folderID is not empty
	if strings.TrimSpace(folderID) == "" {
		log.Error("Folder ID cannot be empty")
		return "", errors.NewValidationError("folder ID is required")
	}

	// Validate tenantID is not empty
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return "", errors.NewValidationError("tenant ID is required")
	}

	// Validate userID is not empty
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return "", errors.NewValidationError("user ID is required")
	}

	// Validate content is not nil
	if content == nil {
		log.Error("Document content cannot be nil")
		return "", errors.NewValidationError("document content is required")
	}

	// Check if folder exists and user has write permission
	_, err := uc.folderService.GetFolder(ctx, folderID, tenantID, userID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder or verify permissions")
		return "", errors.Wrap(err, "failed to get folder or verify permissions")
	}

	// Create a new document using models.NewDocument
	document := models.NewDocument(name, contentType, size, folderID, tenantID, userID)
	document.ID = uuid.New().String()

	// Store document content in temporary storage using storageService.StoreTemporary
	tempPath, err := uc.storageService.StoreTemporary(ctx, tenantID, document.ID, content, size, contentType)
	if err != nil {
		log.WithError(err).Error("Failed to store document in temporary storage")
		return "", errors.Wrap(err, "failed to store document in temporary storage")
	}

	// Add metadata to the document if provided
	if metadata != nil {
		for key, value := range metadata {
			document.AddMetadata(key, value)
		}
	}

	// Persist the document to the repository using documentRepo.Create
	documentID, err := uc.documentRepo.Create(ctx, &document)
	if err != nil {
		log.WithError(err).Error("Failed to persist document to repository")
		return "", errors.Wrap(err, "failed to persist document to repository")
	}

	// Create initial document version
	versionID := uuid.New().String()
	version := models.DocumentVersion{
		ID:            versionID,
		DocumentID:    documentID,
		VersionNumber: 1, // Initial version
		Size:          size,
		ContentHash:   "N/A", // TODO: Calculate content hash
		Status:        models.VersionStatusProcessing,
		StoragePath:   tempPath,
		CreatedAt:     time.Now(),
		CreatedBy:     userID,
	}

	_, err = uc.documentRepo.AddVersion(ctx, &version)
	if err != nil {
		log.WithError(err).Error("Failed to create initial document version")
		return "", errors.Wrap(err, "failed to create initial document version")
	}

	// Queue document for virus scanning using virusScanningService.QueueForScanning
	err = uc.virusScanningService.QueueForScanning(ctx, documentID, versionID, tenantID, tempPath)
	if err != nil {
		log.WithError(err).Error("Failed to queue document for virus scanning")
		return "", errors.Wrap(err, "failed to queue document for virus scanning")
	}

	// Publish document.uploaded event using eventService
	additionalData := map[string]interface{}{
		"name":      name,
		"folderID":  folderID,
		"size":      size,
		"contentType": contentType,
		"userID":    userID,
	}

	_, err = uc.eventService.CreateAndPublishDocumentEvent(ctx, DocumentEventUploaded, tenantID, documentID, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish document.uploaded event")
		// Do not return error, continue processing even if event publishing fails
	}

	// Log successful document upload
	log.Info("Document uploaded successfully", "documentID", documentID, "name", name, "size", size, "contentType", contentType)

	// Return document ID or wrap error with context
	return documentID, nil
}

// GetDocument retrieves a document by its ID with tenant isolation and permission checks
func (uc *documentUseCase) GetDocument(ctx context.Context, id string, tenantID string, userID string) (*models.Document, error) {
	// Get logger with context
	log := uc.logger.WithContext(ctx)

	// Validate document ID is not empty, return ErrInvalidDocumentID if empty
	if strings.TrimSpace(id) == "" {
		log.Error("Document ID cannot be empty")
		return nil, ErrInvalidDocumentID
	}

	// Validate tenant ID is not empty, return ErrInvalidTenantID if empty
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return nil, ErrInvalidTenantID
	}

	// Validate user ID is not empty, return ErrInvalidUserID if empty
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return nil, ErrInvalidUserID
	}

	// Retrieve the document from the repository using documentRepo.GetByID
	document, err := uc.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get document", "documentID", id, "tenantID", tenantID)
		return nil, errors.Wrap(err, "failed to get document")
	}

	// If document not found, return ErrDocumentNotFound
	if document == nil {
		log.Error("Document not found", "documentID", id, "tenantID", tenantID)
		return nil, ErrDocumentNotFound
	}

	// Verify the document belongs to the specified tenant
	if document.TenantID != tenantID {
		log.Error("Document tenant mismatch", "documentID", id, "documentTenantID", document.TenantID, "requestTenantID", tenantID)
		return nil, ErrDocumentNotFound
	}

	// Check if user has read permission for the document using authService.VerifyResourceAccess
	hasAccess, err := uc.authService.VerifyResourceAccess(ctx, userID, tenantID, services.ResourceTypeDocument, id, services.PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify document access", "documentID", id, "tenantID", tenantID, "userID", userID)
		return nil, errors.Wrap(err, "failed to verify document access")
	}

	if !hasAccess {
		log.Error("User does not have read permission for document", "documentID", id, "tenantID", tenantID, "userID", userID)
		return nil, ErrPermissionDenied
	}

	// Log successful document retrieval
	log.Info("Document retrieved successfully", "documentID", id, "tenantID", tenantID)

	// Return the document or wrap error with context
	return document, nil
}

// DownloadDocument downloads a document by its ID with tenant isolation and permission checks
func (uc *documentUseCase) DownloadDocument(ctx context.Context, id string, tenantID string, userID string) (io.ReadCloser, string, error) {
	// Get logger with context
	log := uc.logger.WithContext(ctx)

	// Validate document ID is not empty, return ErrInvalidDocumentID if empty
	if strings.TrimSpace(id) == "" {
		log.Error("Document ID cannot be empty")
		return nil, "", ErrInvalidDocumentID
	}

	// Validate tenant ID is not empty, return ErrInvalidTenantID if empty
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return nil, "", ErrInvalidTenantID
	}

	// Validate user ID is not empty, return ErrInvalidUserID if empty
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return nil, "", ErrInvalidUserID
	}

	// Retrieve the document from the repository using documentRepo.GetByID
	document, err := uc.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get document", "documentID", id, "tenantID", tenantID)
		return nil, "", errors.Wrap(err, "failed to get document")
	}

	// If document not found, return ErrDocumentNotFound
	if document == nil {
		log.Error("Document not found", "documentID", id, "tenantID", tenantID)
		return nil, "", ErrDocumentNotFound
	}

	// Verify the document belongs to the specified tenant
	if document.TenantID != tenantID {
		log.Error("Document tenant mismatch", "documentID", id, "documentTenantID", document.TenantID, "requestTenantID", tenantID)
		return nil, "", ErrDocumentNotFound
	}

	// Check if user has read permission for the document using authService.VerifyResourceAccess
	hasAccess, err := uc.authService.VerifyResourceAccess(ctx, userID, tenantID, services.ResourceTypeDocument, id, services.PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify document access", "documentID", id, "tenantID", tenantID, "userID", userID)
		return nil, "", errors.Wrap(err, "failed to verify document access")
	}

	if !hasAccess {
		log.Error("User does not have read permission for document", "documentID", id, "tenantID", tenantID, "userID", userID)
		return nil, "", ErrPermissionDenied
	}

	// Check if document is available for download (status is DocumentStatusAvailable)
	if !document.IsAvailable() {
		log.Error("Document is not available for download", "documentID", id, "status", document.Status)
		return nil, "", ErrDocumentNotAvailable
	}

	// Get the latest document version
	latestVersion := document.GetLatestVersion()
	if latestVersion == nil {
		log.Error("No versions found for document", "documentID", id)
		return nil, "", errors.NewResourceNotFoundError("no versions found for document")
	}

	// Retrieve document content from storage using storageService.GetDocument
	contentStream, err := uc.storageService.GetDocument(ctx, latestVersion.StoragePath)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve document content from storage", "documentID", id, "storagePath", latestVersion.StoragePath)
		return nil, "", errors.Wrap(err, "failed to retrieve document content from storage")
	}

	// Publish document.downloaded event using eventService
	additionalData := map[string]interface{}{
		"name":   document.Name,
		"userID": userID,
	}

	_, err = uc.eventService.CreateAndPublishDocumentEvent(ctx, DocumentEventDownloaded, tenantID, id, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish document.downloaded event")
		// Do not return error, continue processing even if event publishing fails
	}

	// Log successful document download
	log.Info("Document downloaded successfully", "documentID", id, "tenantID", tenantID)

	// Return document content stream, content type, file name, or wrap error with context
	return contentStream, document.Name, nil
}

// GetDocumentPresignedURL generates a presigned URL for document download with tenant isolation and permission checks
func (uc *documentUseCase) GetDocumentPresignedURL(ctx context.Context, id string, tenantID string, userID string, expirationSeconds int) (string, error) {
	// Get logger with context
	log := uc.logger.WithContext(ctx)

	// Validate document ID is not empty, return ErrInvalidDocumentID if empty
	if strings.TrimSpace(id) == "" {
		log.Error("Document ID cannot be empty")
		return "", ErrInvalidDocumentID
	}

	// Validate tenant ID is not empty, return ErrInvalidTenantID if empty
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return "", ErrInvalidTenantID
	}

	// Validate user ID is not empty, return ErrInvalidUserID if empty
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return "", ErrInvalidUserID
	}

	// Validate expirationSeconds is greater than 0
	if expirationSeconds <= 0 {
		log.Error("Expiration seconds must be greater than 0")
		return "", errors.NewValidationError("expiration seconds must be greater than 0")
	}

	// Retrieve the document from the repository using documentRepo.GetByID
	document, err := uc.documentRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get document", "documentID", id, "tenantID", tenantID)
		return "", errors.Wrap(err, "failed to get document")
	}

	// If document not found, return ErrDocumentNotFound
	if document == nil {
		log.Error("Document not found", "documentID", id, "tenantID", tenantID)
		return "", ErrDocumentNotFound
	}

	// Verify the document belongs to the specified tenant
	if document.TenantID != tenantID {
		log.Error("Document tenant mismatch", "documentID", id, "documentTenantID", document.TenantID, "requestTenantID", tenantID)
		return "", ErrDocumentNotFound
	}

	// Check if user has read permission for the document using authService.VerifyResourceAccess
	hasAccess, err := uc.authService.VerifyResourceAccess(ctx, userID, tenantID, services.ResourceTypeDocument, id, services.PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify document access", "documentID", id, "tenantID", tenantID, "userID", userID)
		return "", errors.Wrap(err, "failed to verify document access")
	}

	if !hasAccess {
		log.Error("User does not have read permission for document", "documentID", id, "tenantID", tenantID, "userID", userID)
		return "", ErrPermissionDenied
	}

	// Check if document is available for download (status is DocumentStatusAvailable)
	if !document.IsAvailable() {
		log.Error("Document is not available for download", "documentID", id, "status", document.Status)
		return "", ErrDocumentNotAvailable
	}

	// Get the latest document version
	latestVersion := document.GetLatestVersion()
	if latestVersion == nil {
		log.Error("No versions found for document", "documentID", id)
		return "", errors.NewResourceNotFoundError("no versions found for document")
	}

	// Generate presigned URL for document content using storageService.GetPresignedURL
	presignedURL, err := uc.storageService.GetPresignedURL(ctx, latestVersion.StoragePath, document.Name, expirationSeconds)
	if err != nil {
		log.WithError(err).Error("Failed to generate presigned URL", "documentID", id, "storagePath", latestVersion.StoragePath)
		return "", errors.Wrap(err, "failed to generate presigned URL")
	}

	// Publish document.downloaded event using eventService
	additionalData := map[string]interface{}{
		"name":   document.Name,
		"userID": userID,
	}

	_, err = uc.eventService.CreateAndPublishDocumentEvent(ctx, DocumentEventDownloaded, tenantID, id, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish document.downloaded event")
		// Do not return error, continue processing even if event publishing fails
	}

	// Log successful presigned URL generation
	log.Info("Presigned URL generated successfully", "documentID", id, "tenantID", tenantID)

	// Return presigned URL or wrap error with context
	return presignedURL, nil
}

// BatchDownloadDocuments downloads multiple documents as a compressed archive with tenant isolation and permission checks
func (uc *documentUseCase) BatchDownloadDocuments(ctx context.Context, ids []string, tenantID string, userID string) (io.ReadCloser, error) {
	panic("implement me")
}

// GetBatchDownloadPresignedURL generates a presigned URL for batch document download with tenant isolation and permission checks
func (uc *documentUseCase) GetBatchDownloadPresignedURL(ctx context.Context, ids []string, tenantID string, userID string, expirationSeconds int) (string, error) {
	panic("implement me")
}

// DeleteDocument deletes a document by its ID with tenant isolation and permission checks
func (uc *documentUseCase) DeleteDocument(ctx context.Context, id string, tenantID string, userID string) error {
	panic("implement me")
}

// ListDocumentsByFolder lists documents in a folder with pagination, tenant isolation, and permission checks
func (uc *documentUseCase) ListDocumentsByFolder(ctx context.Context, folderID string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	panic("implement me")
}

// SearchDocumentsByContent searches documents by their content with tenant isolation and permission checks
func (uc *documentUseCase) SearchDocumentsByContent(ctx context.Context, query string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	panic("implement me")
}

// SearchDocumentsByMetadata searches documents by their metadata with tenant isolation and permission checks
func (uc *documentUseCase) SearchDocumentsByMetadata(ctx context.Context, metadata map[string]string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	panic("implement me")
}

// CombinedSearch performs a search using both content and metadata criteria with tenant isolation and permission checks
func (uc *documentUseCase) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	panic("implement me")
}

// UpdateDocumentMetadata updates document metadata with tenant isolation and permission checks
func (uc *documentUseCase) UpdateDocumentMetadata(ctx context.Context, id string, key string, value string, tenantID string, userID string) error {
	panic("implement me")
}

// DeleteDocumentMetadata deletes document metadata with tenant isolation and permission checks
func (uc *documentUseCase) DeleteDocumentMetadata(ctx context.Context, id string, key string, tenantID string, userID string) error {
	panic("implement me")
}

// GetDocumentThumbnail retrieves a document thumbnail with tenant isolation and permission checks
func (uc *documentUseCase) GetDocumentThumbnail(ctx context.Context, id string, tenantID string, userID string) (io.ReadCloser, error) {
	panic("implement me")
}

// GetDocumentThumbnailURL generates a presigned URL for document thumbnail with tenant isolation and permission checks
func (uc *documentUseCase) GetDocumentThumbnailURL(ctx context.Context, id string, tenantID string, userID string, expirationSeconds int) (string, error) {
	panic("implement me")
}

// GetDocumentStatus gets the current status of a document with tenant isolation and permission checks
func (uc *documentUseCase) GetDocumentStatus(ctx context.Context, id string, tenantID string, userID string) (string, error) {
	panic("implement me")
}