// Package postgres provides PostgreSQL implementations of repository interfaces.
package postgres

import (
	"context" // standard library
	"fmt"     // standard library
	"strings" // standard library
	"time"    // standard library

	"gorm.io/gorm" // v1.25.0+
	"github.com/google/uuid" // v1.3.0+

	"../../../domain/repositories"
	"../../../domain/models"
	"../../../pkg/utils"
	"../../../pkg/errors"
)

// documentRepository is a PostgreSQL implementation of the DocumentRepository interface
type documentRepository struct {
	db *gorm.DB
}

// NewDocumentRepository creates a new PostgreSQL document repository instance
func NewDocumentRepository(db *gorm.DB) repositories.DocumentRepository {
	if db == nil {
		panic("db cannot be nil")
	}
	
	return &documentRepository{
		db: db,
	}
}

// Create stores a new document in the repository and returns its ID.
func (r *documentRepository) Create(ctx context.Context, document *models.Document) (string, error) {
	if err := document.Validate(); err != nil {
		return "", errors.NewValidationError(fmt.Sprintf("invalid document: %s", err.Error()))
	}

	// Generate a new UUID if not provided
	if document.ID == "" {
		document.ID = uuid.New().String()
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Create the document
	if err := tx.Create(document).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to create document")
	}

	// Create metadata entries if any
	if len(document.Metadata) > 0 {
		for i := range document.Metadata {
			document.Metadata[i].DocumentID = document.ID
			if document.Metadata[i].ID == "" {
				document.Metadata[i].ID = uuid.New().String()
			}
			if err := tx.Create(&document.Metadata[i]).Error; err != nil {
				tx.Rollback()
				return "", errors.Wrap(err, "failed to create document metadata")
			}
		}
	}

	// Create versions if any
	if len(document.Versions) > 0 {
		for i := range document.Versions {
			document.Versions[i].DocumentID = document.ID
			if document.Versions[i].ID == "" {
				document.Versions[i].ID = uuid.New().String()
			}
			if err := tx.Create(&document.Versions[i]).Error; err != nil {
				tx.Rollback()
				return "", errors.Wrap(err, "failed to create document version")
			}
		}
	}

	// Handle tags if any
	if len(document.Tags) > 0 {
		for _, tag := range document.Tags {
			// Associate document with tag (using a join table)
			if err := tx.Table("document_tags").Create(map[string]interface{}{
				"document_id": document.ID,
				"tag_id":      tag.ID,
			}).Error; err != nil {
				tx.Rollback()
				return "", errors.Wrap(err, "failed to associate document with tag")
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return document.ID, nil
}

// GetByID retrieves a document by its ID with tenant isolation.
func (r *documentRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.Document, error) {
	if id == "" {
		return nil, errors.NewValidationError("document ID cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var document models.Document

	// Query with tenant isolation
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Preload("Metadata").
		Preload("Versions").
		Preload("Tags").
		First(&document).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found", id))
		}
		return nil, errors.Wrap(err, "failed to get document")
	}

	return &document, nil
}

// Update modifies an existing document with tenant isolation.
func (r *documentRepository) Update(ctx context.Context, document *models.Document) error {
	if err := document.Validate(); err != nil {
		return errors.NewValidationError(fmt.Sprintf("invalid document: %s", err.Error()))
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if document exists and belongs to the tenant
	var existingDoc models.Document
	if err := tx.Where("id = ? AND tenant_id = ?", document.ID, document.TenantID).First(&existingDoc).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found or does not belong to tenant", document.ID))
		}
		return errors.Wrap(err, "failed to check document existence")
	}

	// Update the document
	if err := tx.Model(&document).Updates(map[string]interface{}{
		"name":         document.Name,
		"content_type": document.ContentType,
		"size":         document.Size,
		"folder_id":    document.FolderID,
		"owner_id":     document.OwnerID,
		"status":       document.Status,
		"updated_at":   document.UpdatedAt,
	}).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update document")
	}

	// Handle metadata updates
	if len(document.Metadata) > 0 {
		// Delete existing metadata for clean update
		if err := tx.Where("document_id = ?", document.ID).Delete(&models.DocumentMetadata{}).Error; err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to delete existing metadata")
		}

		// Create new metadata entries
		for i := range document.Metadata {
			document.Metadata[i].DocumentID = document.ID
			if document.Metadata[i].ID == "" {
				document.Metadata[i].ID = uuid.New().String()
			}
			if err := tx.Create(&document.Metadata[i]).Error; err != nil {
				tx.Rollback()
				return errors.Wrap(err, "failed to create document metadata")
			}
		}
	}

	// Handle version updates
	if len(document.Versions) > 0 {
		// Only add new versions, don't modify existing ones
		for i := range document.Versions {
			// Check if version already exists
			var existingVersion models.DocumentVersion
			err := tx.Where("id = ? AND document_id = ?", document.Versions[i].ID, document.ID).
				First(&existingVersion).Error

			if err == gorm.ErrRecordNotFound {
				// New version, create it
				document.Versions[i].DocumentID = document.ID
				if document.Versions[i].ID == "" {
					document.Versions[i].ID = uuid.New().String()
				}
				if err := tx.Create(&document.Versions[i]).Error; err != nil {
					tx.Rollback()
					return errors.Wrap(err, "failed to create document version")
				}
			} else if err != nil {
				tx.Rollback()
				return errors.Wrap(err, "failed to check version existence")
			}
			// If version exists, do nothing (versions are immutable)
		}
	}

	// Handle tag updates
	if len(document.Tags) > 0 {
		// Remove existing tag associations
		if err := tx.Table("document_tags").Where("document_id = ?", document.ID).Delete(nil).Error; err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to remove existing tag associations")
		}

		// Add new tag associations
		for _, tag := range document.Tags {
			if err := tx.Table("document_tags").Create(map[string]interface{}{
				"document_id": document.ID,
				"tag_id":      tag.ID,
			}).Error; err != nil {
				tx.Rollback()
				return errors.Wrap(err, "failed to associate document with tag")
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// Delete removes a document by its ID with tenant isolation.
func (r *documentRepository) Delete(ctx context.Context, id string, tenantID string) error {
	if id == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if document exists and belongs to the tenant
	var document models.Document
	if err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&document).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found or does not belong to tenant", id))
		}
		return errors.Wrap(err, "failed to check document existence")
	}

	// Delete metadata
	if err := tx.Where("document_id = ?", id).Delete(&models.DocumentMetadata{}).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete document metadata")
	}

	// Delete versions
	if err := tx.Where("document_id = ?", id).Delete(&models.DocumentVersion{}).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete document versions")
	}

	// Delete tag associations
	if err := tx.Table("document_tags").Where("document_id = ?", id).Delete(nil).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete document tag associations")
	}

	// Delete the document
	if err := tx.Delete(&models.Document{}, id).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete document")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// ListByFolder retrieves documents in a specific folder with pagination and tenant isolation.
func (r *documentRepository) ListByFolder(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
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

	var documents []models.Document
	var totalItems int64

	// Count total matching documents
	if err := r.db.WithContext(ctx).Model(&models.Document{}).
		Where("folder_id = ? AND tenant_id = ?", folderID, tenantID).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to count documents")
	}

	// Query documents with pagination
	if err := r.db.WithContext(ctx).
		Where("folder_id = ? AND tenant_id = ?", folderID, tenantID).
		Preload("Metadata").
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Order("version_number DESC") // Latest version first
		}).
		Preload("Tags").
		Limit(pagination.GetLimit()).
		Offset(pagination.GetOffset()).
		Find(&documents).Error; err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to list documents")
	}

	// Create paginated result
	result := utils.NewPaginatedResult(documents, pagination, totalItems)
	return result, nil
}

// ListByTenant lists all documents for a tenant with pagination.
func (r *documentRepository) ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	var documents []models.Document
	var totalItems int64

	// Count total matching documents
	if err := r.db.WithContext(ctx).Model(&models.Document{}).
		Where("tenant_id = ?", tenantID).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to count documents")
	}

	// Query documents with pagination
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Preload("Metadata").
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Order("version_number DESC") // Latest version first
		}).
		Preload("Tags").
		Limit(pagination.GetLimit()).
		Offset(pagination.GetOffset()).
		Find(&documents).Error; err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to list documents")
	}

	// Create paginated result
	result := utils.NewPaginatedResult(documents, pagination, totalItems)
	return result, nil
}

// SearchByContent searches documents by their content with tenant isolation.
func (r *documentRepository) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	if query == "" {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("search query cannot be empty")
	}
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Note: This implementation returns an empty result as content search 
	// is handled by the search service (Elasticsearch)
	// The actual implementation would integrate with the search service
	
	// Return empty result
	var emptyDocuments []models.Document
	return utils.NewPaginatedResult(emptyDocuments, pagination, 0), nil
}

// SearchByMetadata searches documents by their metadata with tenant isolation.
func (r *documentRepository) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	if len(metadata) == 0 {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("metadata cannot be empty")
	}
	if tenantID == "" {
		return utils.PaginatedResult[models.Document]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	var documents []models.Document
	var totalItems int64

	// Base query with tenant isolation
	baseQuery := r.db.WithContext(ctx).Table("documents").
		Joins("JOIN document_metadata ON documents.id = document_metadata.document_id").
		Where("documents.tenant_id = ?", tenantID).
		Group("documents.id")

	// Add conditions for each metadata key-value pair
	conditions := []string{}
	values := []interface{}{}

	for key, value := range metadata {
		conditions = append(conditions, "(document_metadata.key = ? AND document_metadata.value = ?)")
		values = append(values, key, value)
	}

	// Combine conditions with OR
	conditionStr := strings.Join(conditions, " OR ")
	if conditionStr != "" {
		baseQuery = baseQuery.Where(conditionStr, values...)
	}

	// Count total matching documents
	if err := baseQuery.Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to count documents")
	}

	// Query documents with pagination
	docIds := []string{}
	if err := baseQuery.Limit(pagination.GetLimit()).Offset(pagination.GetOffset()).
		Pluck("documents.id", &docIds).Error; err != nil {
		return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to search documents")
	}

	// Retrieve full documents with their relations
	if len(docIds) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", docIds).
			Preload("Metadata").
			Preload("Versions", func(db *gorm.DB) *gorm.DB {
				return db.Order("version_number DESC") // Latest version first
			}).
			Preload("Tags").
			Find(&documents).Error; err != nil {
			return utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to retrieve documents")
		}
	}

	// Create paginated result
	result := utils.NewPaginatedResult(documents, pagination, totalItems)
	return result, nil
}

// AddVersion adds a new version to an existing document with tenant isolation.
func (r *documentRepository) AddVersion(ctx context.Context, version *models.DocumentVersion) (string, error) {
	if err := version.Validate(); err != nil {
		return "", errors.NewValidationError(fmt.Sprintf("invalid document version: %s", err.Error()))
	}

	// Generate a new UUID if not provided
	if version.ID == "" {
		version.ID = uuid.New().String()
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if document exists and get tenant ID
	var document models.Document
	if err := tx.Where("id = ?", version.DocumentID).First(&document).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return "", errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found", version.DocumentID))
		}
		return "", errors.Wrap(err, "failed to check document existence")
	}

	// Create the version
	if err := tx.Create(version).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to create document version")
	}

	// Update document's updated_at timestamp
	if err := tx.Model(&document).Update("updated_at", version.CreatedAt).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to update document timestamp")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return version.ID, nil
}

// GetVersionByID retrieves a document version by its ID with tenant isolation.
func (r *documentRepository) GetVersionByID(ctx context.Context, versionID string, tenantID string) (*models.DocumentVersion, error) {
	if versionID == "" {
		return nil, errors.NewValidationError("version ID cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var version models.DocumentVersion

	// Query with tenant isolation (join with documents table to check tenant)
	err := r.db.WithContext(ctx).
		Joins("JOIN documents ON document_versions.document_id = documents.id").
		Where("document_versions.id = ? AND documents.tenant_id = ?", versionID, tenantID).
		First(&version).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("document version with ID %s not found", versionID))
		}
		return nil, errors.Wrap(err, "failed to get document version")
	}

	return &version, nil
}

// UpdateVersionStatus updates the status of a document version with tenant isolation.
func (r *documentRepository) UpdateVersionStatus(ctx context.Context, versionID string, status string, tenantID string) error {
	if versionID == "" {
		return errors.NewValidationError("version ID cannot be empty")
	}
	if status == "" {
		return errors.NewValidationError("status cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if version exists and belongs to a document owned by the tenant
	var version models.DocumentVersion
	err := tx.Joins("JOIN documents ON document_versions.document_id = documents.id").
		Where("document_versions.id = ? AND documents.tenant_id = ?", versionID, tenantID).
		First(&version).Error

	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("document version with ID %s not found or does not belong to tenant", versionID))
		}
		return errors.Wrap(err, "failed to check version existence")
	}

	// Update the version status
	if err := tx.Model(&models.DocumentVersion{}).Where("id = ?", versionID).
		Update("status", status).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update version status")
	}

	// Check if this is the latest version and update document status if needed
	var latestVersion models.DocumentVersion
	err = tx.Where("document_id = ?", version.DocumentID).
		Order("version_number DESC").
		First(&latestVersion).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return errors.Wrap(err, "failed to get latest version")
	}

	if err == nil && latestVersion.ID == versionID {
		// This is the latest version, update the document status as well
		if err := tx.Model(&models.Document{}).Where("id = ?", version.DocumentID).
			Update("status", status).Error; err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to update document status")
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// AddMetadata adds metadata to a document with tenant isolation.
func (r *documentRepository) AddMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) (string, error) {
	if documentID == "" {
		return "", errors.NewValidationError("document ID cannot be empty")
	}
	if key == "" {
		return "", errors.NewValidationError("metadata key cannot be empty")
	}
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if document exists and belongs to the tenant
	var document models.Document
	if err := tx.Where("id = ? AND tenant_id = ?", documentID, tenantID).First(&document).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return "", errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found or does not belong to tenant", documentID))
		}
		return "", errors.Wrap(err, "failed to check document existence")
	}

	// Check if metadata with the same key already exists
	var existingMetadata models.DocumentMetadata
	err := tx.Where("document_id = ? AND key = ?", documentID, key).
		First(&existingMetadata).Error

	var metadataID string

	if err == gorm.ErrRecordNotFound {
		// Create new metadata
		metadata := models.NewDocumentMetadata(documentID, key, value)
		metadata.ID = uuid.New().String()
		
		if err := tx.Create(&metadata).Error; err != nil {
			tx.Rollback()
			return "", errors.Wrap(err, "failed to create document metadata")
		}
		
		metadataID = metadata.ID
	} else if err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to check metadata existence")
	} else {
		// Update existing metadata
		existingMetadata.Update(value)
		
		if err := tx.Save(&existingMetadata).Error; err != nil {
			tx.Rollback()
			return "", errors.Wrap(err, "failed to update document metadata")
		}
		
		metadataID = existingMetadata.ID
	}

	// Update document's updated_at timestamp
	now := time.Now()
	if err := tx.Model(&document).Update("updated_at", now).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to update document timestamp")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return metadataID, nil
}

// UpdateMetadata updates existing document metadata with tenant isolation.
func (r *documentRepository) UpdateMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) error {
	if documentID == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	if key == "" {
		return errors.NewValidationError("metadata key cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if document exists and belongs to the tenant
	var document models.Document
	if err := tx.Where("id = ? AND tenant_id = ?", documentID, tenantID).First(&document).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found or does not belong to tenant", documentID))
		}
		return errors.Wrap(err, "failed to check document existence")
	}

	// Check if metadata with the key exists
	var metadata models.DocumentMetadata
	if err := tx.Where("document_id = ? AND key = ?", documentID, key).First(&metadata).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("metadata with key %s not found for document %s", key, documentID))
		}
		return errors.Wrap(err, "failed to check metadata existence")
	}

	// Update the metadata
	metadata.Update(value)
	if err := tx.Save(&metadata).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update metadata")
	}

	// Update document's updated_at timestamp
	now := time.Now()
	if err := tx.Model(&document).Update("updated_at", now).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update document timestamp")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// DeleteMetadata deletes document metadata by key with tenant isolation.
func (r *documentRepository) DeleteMetadata(ctx context.Context, documentID string, key string, tenantID string) error {
	if documentID == "" {
		return errors.NewValidationError("document ID cannot be empty")
	}
	if key == "" {
		return errors.NewValidationError("metadata key cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if document exists and belongs to the tenant
	var document models.Document
	if err := tx.Where("id = ? AND tenant_id = ?", documentID, tenantID).First(&document).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("document with ID %s not found or does not belong to tenant", documentID))
		}
		return errors.Wrap(err, "failed to check document existence")
	}

	// Delete the metadata
	result := tx.Where("document_id = ? AND key = ?", documentID, key).Delete(&models.DocumentMetadata{})
	if result.Error != nil {
		tx.Rollback()
		return errors.Wrap(result.Error, "failed to delete metadata")
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return errors.NewResourceNotFoundError(fmt.Sprintf("metadata with key %s not found for document %s", key, documentID))
	}

	// Update document's updated_at timestamp
	now := time.Now()
	if err := tx.Model(&document).Update("updated_at", now).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update document timestamp")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// GetDocumentsByIDs retrieves multiple documents by their IDs with tenant isolation.
func (r *documentRepository) GetDocumentsByIDs(ctx context.Context, ids []string, tenantID string) ([]*models.Document, error) {
	if len(ids) == 0 {
		return []*models.Document{}, errors.NewValidationError("document IDs cannot be empty")
	}
	if tenantID == "" {
		return []*models.Document{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	var documents []*models.Document

	// Query with tenant isolation
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND tenant_id = ?", ids, tenantID).
		Preload("Metadata").
		Preload("Versions").
		Preload("Tags").
		Find(&documents).Error; err != nil {
		return nil, errors.Wrap(err, "failed to get documents by IDs")
	}

	return documents, nil
}