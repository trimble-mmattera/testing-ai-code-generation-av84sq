// Package postgres provides PostgreSQL implementations of the domain repositories.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid" // v1.3.0+
	"gorm.io/gorm" // v1.25.0+

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/errors"
	"../../../pkg/utils"
)

// tagRepository implements the repositories.TagRepository interface using PostgreSQL.
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new PostgreSQL tag repository instance.
func NewTagRepository(db *gorm.DB) (repositories.TagRepository, error) {
	if db == nil {
		return nil, errors.NewValidationError("db cannot be nil")
	}
	return &tagRepository{db: db}, nil
}

// Create persists a new tag in the database.
func (r *tagRepository) Create(ctx context.Context, tag *models.Tag) (string, error) {
	if err := tag.Validate(); err != nil {
		return "", errors.NewValidationError(err.Error())
	}

	// Generate a new UUID if ID is empty
	if tag.ID == "" {
		tag.ID = uuid.New().String()
	}

	// Start a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if tag with same name already exists for this tenant
	var count int64
	if err := tx.Model(&models.Tag{}).Where("name = ? AND tenant_id = ?", tag.Name, tag.TenantID).Count(&count).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to check for existing tag")
	}

	if count > 0 {
		tx.Rollback()
		return "", errors.NewValidationError(fmt.Sprintf("tag with name '%s' already exists for this tenant", tag.Name))
	}

	// Try to create the tag
	if err := tx.Create(tag).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to create tag")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return tag.ID, nil
}

// GetByID retrieves a tag by its ID with tenant isolation.
func (r *tagRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.Tag, error) {
	if id == "" {
		return nil, errors.NewValidationError("id cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenantID cannot be empty")
	}

	var tag models.Tag
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("tag with ID '%s' not found", id))
		}
		return nil, errors.Wrap(err, "failed to get tag by ID")
	}

	return &tag, nil
}

// GetByName retrieves a tag by its name with tenant isolation.
func (r *tagRepository) GetByName(ctx context.Context, name string, tenantID string) (*models.Tag, error) {
	if name == "" {
		return nil, errors.NewValidationError("name cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenantID cannot be empty")
	}

	var tag models.Tag
	if err := r.db.WithContext(ctx).Where("name = ? AND tenant_id = ?", name, tenantID).First(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("tag with name '%s' not found", name))
		}
		return nil, errors.Wrap(err, "failed to get tag by name")
	}

	return &tag, nil
}

// Update modifies an existing tag with tenant isolation enforcement.
func (r *tagRepository) Update(ctx context.Context, tag *models.Tag) error {
	if err := tag.Validate(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	// Check if tag exists and belongs to the specified tenant
	existingTag, err := r.GetByID(ctx, tag.ID, tag.TenantID)
	if err != nil {
		return err
	}

	// Start a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if another tag with the same name already exists for this tenant
	var count int64
	if err := tx.Model(&models.Tag{}).Where("name = ? AND tenant_id = ? AND id != ?", tag.Name, tag.TenantID, tag.ID).Count(&count).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check for existing tag")
	}

	if count > 0 {
		tx.Rollback()
		return errors.NewValidationError(fmt.Sprintf("another tag with name '%s' already exists for this tenant", tag.Name))
	}

	// Preserve created time
	tag.CreatedAt = existingTag.CreatedAt 

	// Update the tag
	if err := tx.Where("id = ? AND tenant_id = ?", tag.ID, tag.TenantID).Save(tag).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update tag")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// Delete removes a tag by its ID with tenant isolation.
func (r *tagRepository) Delete(ctx context.Context, id string, tenantID string) error {
	if id == "" {
		return errors.NewValidationError("id cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenantID cannot be empty")
	}

	// Start a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if tag exists and belongs to the specified tenant
	var count int64
	if err := tx.Model(&models.Tag{}).Where("id = ? AND tenant_id = ?", id, tenantID).Count(&count).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check tag existence")
	}

	if count == 0 {
		tx.Rollback()
		return errors.NewResourceNotFoundError(fmt.Sprintf("tag with ID '%s' not found", id))
	}

	// Delete document-tag associations first
	if err := tx.Table("document_tags").Where("tag_id = ?", id).Delete(nil).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete document-tag associations")
	}

	// Delete the tag
	if err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Tag{}).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete tag")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// ListByTenant retrieves all tags for a tenant with pagination.
func (r *tagRepository) ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Tag], error) {
	if tenantID == "" {
		return utils.PaginatedResult[models.Tag]{}, errors.NewValidationError("tenantID cannot be empty")
	}

	// Use default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	var tags []models.Tag
	var totalItems int64

	// Count total items
	if err := r.db.WithContext(ctx).Model(&models.Tag{}).Where("tenant_id = ?", tenantID).Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Tag]{}, errors.Wrap(err, "failed to count tags")
	}

	// Retrieve items with pagination
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Limit(pagination.GetLimit()).
		Offset(pagination.GetOffset()).
		Find(&tags).Error; err != nil {
		return utils.PaginatedResult[models.Tag]{}, errors.Wrap(err, "failed to list tags")
	}

	// Create paginated result
	return utils.NewPaginatedResult(tags, pagination, totalItems), nil
}

// SearchByName finds tags matching a name pattern with tenant isolation.
func (r *tagRepository) SearchByName(ctx context.Context, namePattern string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Tag], error) {
	if namePattern == "" {
		return utils.PaginatedResult[models.Tag]{}, errors.NewValidationError("namePattern cannot be empty")
	}
	if tenantID == "" {
		return utils.PaginatedResult[models.Tag]{}, errors.NewValidationError("tenantID cannot be empty")
	}

	// Use default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Add wildcards to name pattern for LIKE query
	likePattern := "%" + namePattern + "%"

	var tags []models.Tag
	var totalItems int64

	// Count total items
	if err := r.db.WithContext(ctx).Model(&models.Tag{}).
		Where("tenant_id = ? AND name ILIKE ?", tenantID, likePattern).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Tag]{}, errors.Wrap(err, "failed to count tags by name pattern")
	}

	// Retrieve items with pagination
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name ILIKE ?", tenantID, likePattern).
		Order("name ASC").
		Limit(pagination.GetLimit()).
		Offset(pagination.GetOffset()).
		Find(&tags).Error; err != nil {
		return utils.PaginatedResult[models.Tag]{}, errors.Wrap(err, "failed to search tags by name pattern")
	}

	// Create paginated result
	return utils.NewPaginatedResult(tags, pagination, totalItems), nil
}

// AddTagToDocument associates a tag with a document with tenant isolation.
func (r *tagRepository) AddTagToDocument(ctx context.Context, tagID string, documentID string, tenantID string) error {
	if tagID == "" {
		return errors.NewValidationError("tagID cannot be empty")
	}
	if documentID == "" {
		return errors.NewValidationError("documentID cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenantID cannot be empty")
	}

	// Start a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if tag exists and belongs to the specified tenant
	var tagCount int64
	if err := tx.Model(&models.Tag{}).Where("id = ? AND tenant_id = ?", tagID, tenantID).Count(&tagCount).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check tag existence")
	}

	if tagCount == 0 {
		tx.Rollback()
		return errors.NewResourceNotFoundError(fmt.Sprintf("tag with ID '%s' not found", tagID))
	}

	// Check if document exists and belongs to the specified tenant
	var docCount int64
	if err := tx.Table("documents").Where("id = ? AND tenant_id = ?", documentID, tenantID).Count(&docCount).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check document existence")
	}

	if docCount == 0 {
		tx.Rollback()
		return errors.NewResourceNotFoundError(fmt.Sprintf("document with ID '%s' not found", documentID))
	}

	// Check if the association already exists
	var assocCount int64
	if err := tx.Table("document_tags").Where("document_id = ? AND tag_id = ?", documentID, tagID).Count(&assocCount).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check document-tag association existence")
	}

	if assocCount > 0 {
		tx.Commit() // No error, but no insert needed
		return nil
	}

	// Create the association
	if err := tx.Table("document_tags").Create(map[string]interface{}{
		"document_id": documentID,
		"tag_id":      tagID,
	}).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to create document-tag association")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// RemoveTagFromDocument removes a tag association from a document with tenant isolation.
func (r *tagRepository) RemoveTagFromDocument(ctx context.Context, tagID string, documentID string, tenantID string) error {
	if tagID == "" {
		return errors.NewValidationError("tagID cannot be empty")
	}
	if documentID == "" {
		return errors.NewValidationError("documentID cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenantID cannot be empty")
	}

	// Start a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if tag exists and belongs to the specified tenant
	var tagCount int64
	if err := tx.Model(&models.Tag{}).Where("id = ? AND tenant_id = ?", tagID, tenantID).Count(&tagCount).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check tag existence")
	}

	if tagCount == 0 {
		tx.Rollback()
		return errors.NewResourceNotFoundError(fmt.Sprintf("tag with ID '%s' not found", tagID))
	}

	// Check if document exists and belongs to the specified tenant
	var docCount int64
	if err := tx.Table("documents").Where("id = ? AND tenant_id = ?", documentID, tenantID).Count(&docCount).Error; err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to check document existence")
	}

	if docCount == 0 {
		tx.Rollback()
		return errors.NewResourceNotFoundError(fmt.Sprintf("document with ID '%s' not found", documentID))
	}

	// Delete the association
	result := tx.Table("document_tags").Where("document_id = ? AND tag_id = ?", documentID, tagID).Delete(nil)
	if result.Error != nil {
		tx.Rollback()
		return errors.Wrap(result.Error, "failed to delete document-tag association")
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// GetTagsByDocumentID retrieves all tags associated with a document with tenant isolation.
func (r *tagRepository) GetTagsByDocumentID(ctx context.Context, documentID string, tenantID string) ([]*models.Tag, error) {
	if documentID == "" {
		return nil, errors.NewValidationError("documentID cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenantID cannot be empty")
	}

	var tags []*models.Tag

	// Check if document exists and belongs to the specified tenant
	var docCount int64
	if err := r.db.WithContext(ctx).Table("documents").Where("id = ? AND tenant_id = ?", documentID, tenantID).Count(&docCount).Error; err != nil {
		return nil, errors.Wrap(err, "failed to check document existence")
	}

	if docCount == 0 {
		return nil, errors.NewResourceNotFoundError(fmt.Sprintf("document with ID '%s' not found", documentID))
	}

	// Get tags associated with the document
	err := r.db.WithContext(ctx).
		Joins("INNER JOIN document_tags ON document_tags.tag_id = tags.id").
		Where("document_tags.document_id = ? AND tags.tenant_id = ?", documentID, tenantID).
		Find(&tags).Error

	if err != nil {
		return nil, errors.Wrap(err, "failed to get tags for document")
	}

	return tags, nil
}

// GetDocumentsByTagID retrieves all document IDs associated with a tag with tenant isolation.
func (r *tagRepository) GetDocumentsByTagID(ctx context.Context, tagID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[string], error) {
	if tagID == "" {
		return utils.PaginatedResult[string]{}, errors.NewValidationError("tagID cannot be empty")
	}
	if tenantID == "" {
		return utils.PaginatedResult[string]{}, errors.NewValidationError("tenantID cannot be empty")
	}

	// Use default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Check if tag exists and belongs to the specified tenant
	var tagCount int64
	if err := r.db.WithContext(ctx).Model(&models.Tag{}).Where("id = ? AND tenant_id = ?", tagID, tenantID).Count(&tagCount).Error; err != nil {
		return utils.PaginatedResult[string]{}, errors.Wrap(err, "failed to check tag existence")
	}

	if tagCount == 0 {
		return utils.PaginatedResult[string]{}, errors.NewResourceNotFoundError(fmt.Sprintf("tag with ID '%s' not found", tagID))
	}

	var documentIDs []string
	var totalItems int64

	// Count total items
	if err := r.db.WithContext(ctx).
		Table("documents").
		Joins("INNER JOIN document_tags ON document_tags.document_id = documents.id").
		Where("document_tags.tag_id = ? AND documents.tenant_id = ?", tagID, tenantID).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[string]{}, errors.Wrap(err, "failed to count documents by tag")
	}

	// Retrieve items with pagination
	if err := r.db.WithContext(ctx).
		Table("documents").
		Select("documents.id").
		Joins("INNER JOIN document_tags ON document_tags.document_id = documents.id").
		Where("document_tags.tag_id = ? AND documents.tenant_id = ?", tagID, tenantID).
		Order("documents.created_at DESC").
		Limit(pagination.GetLimit()).
		Offset(pagination.GetOffset()).
		Pluck("documents.id", &documentIDs).Error; err != nil {
		return utils.PaginatedResult[string]{}, errors.Wrap(err, "failed to get document IDs by tag")
	}

	// Create paginated result
	return utils.NewPaginatedResult(documentIDs, pagination, totalItems), nil
}