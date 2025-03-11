// Package postgres provides PostgreSQL implementations of the repository interfaces.
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

// postgresqlFolderRepository is the PostgreSQL implementation of the FolderRepository interface
type postgresqlFolderRepository struct {
	db *gorm.DB
}

// NewFolderRepository creates a new PostgreSQL folder repository instance
func NewFolderRepository(db *gorm.DB) repositories.FolderRepository {
	if db == nil {
		panic("db cannot be nil")
	}
	return &postgresqlFolderRepository{db: db}
}

// Create creates a new folder in the database
func (r *postgresqlFolderRepository) Create(ctx context.Context, folder *models.Folder) (string, error) {
	if err := folder.Validate(); err != nil {
		return "", errors.NewValidationError(err.Error())
	}

	// Generate a new UUID for the folder ID
	folder.ID = uuid.New().String()

	// Set the folder path based on whether it's a root folder or has a parent
	if folder.IsRoot() {
		folder.SetPath(models.PathSeparator + folder.Name)
	} else {
		// Get parent folder to build the path
		var parentFolder models.Folder
		if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", folder.ParentID, folder.TenantID).First(&parentFolder).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return "", errors.NewNotFoundError(fmt.Sprintf("parent folder with ID %s not found", folder.ParentID))
			}
			return "", errors.NewInternalError(fmt.Sprintf("error fetching parent folder: %v", err))
		}
		folder.SetPath(folder.BuildPath(parentFolder.Path))
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Create the folder
	if err := tx.Create(folder).Error; err != nil {
		tx.Rollback()
		return "", errors.NewInternalError(fmt.Sprintf("failed to create folder: %v", err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return "", errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return folder.ID, nil
}

// GetByID retrieves a folder by its ID with tenant isolation
func (r *postgresqlFolderRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.Folder, error) {
	if id == "" {
		return nil, errors.NewValidationError("folder ID cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var folder models.Folder
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError(fmt.Sprintf("folder with ID %s not found", id))
		}
		return nil, errors.NewInternalError(fmt.Sprintf("error fetching folder: %v", err))
	}

	return &folder, nil
}

// Update updates an existing folder with tenant isolation
func (r *postgresqlFolderRepository) Update(ctx context.Context, folder *models.Folder) error {
	if err := folder.Validate(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	// Check if folder exists
	var existingFolder models.Folder
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", folder.ID, folder.TenantID).First(&existingFolder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError(fmt.Sprintf("folder with ID %s not found", folder.ID))
		}
		return errors.NewInternalError(fmt.Sprintf("error fetching folder: %v", err))
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Update the folder
	if err := tx.Model(&models.Folder{}).Where("id = ? AND tenant_id = ?", folder.ID, folder.TenantID).
		Updates(map[string]interface{}{
			"name":       folder.Name,
			"updated_at": folder.UpdatedAt,
		}).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to update folder: %v", err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// Delete deletes a folder by its ID with tenant isolation
func (r *postgresqlFolderRepository) Delete(ctx context.Context, id string, tenantID string) error {
	if id == "" {
		return errors.NewValidationError("folder ID cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Check if folder exists
	var folder models.Folder
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError(fmt.Sprintf("folder with ID %s not found", id))
		}
		return errors.NewInternalError(fmt.Sprintf("error fetching folder: %v", err))
	}

	// Check if folder has child folders or documents
	isEmpty, err := r.IsEmpty(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !isEmpty {
		return errors.NewConflictError("cannot delete folder that contains items")
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Delete the folder
	if err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Folder{}).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to delete folder: %v", err))
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// GetChildren lists child folders of a parent folder with pagination and tenant isolation
func (r *postgresqlFolderRepository) GetChildren(ctx context.Context, parentID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	if tenantID == "" {
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	// If parentID is provided, check if it exists
	if parentID != "" {
		var parent models.Folder
		if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", parentID, tenantID).First(&parent).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return utils.PaginatedResult[models.Folder]{}, errors.NewNotFoundError(fmt.Sprintf("parent folder with ID %s not found", parentID))
			}
			return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error fetching parent folder: %v", err))
		}
	}

	var folders []models.Folder
	query := r.db.WithContext(ctx).Where("parent_id = ? AND tenant_id = ?", parentID, tenantID).
		Order("name ASC").
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit())

	if err := query.Find(&folders).Error; err != nil {
		return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error fetching child folders: %v", err))
	}

	// Count total items for pagination
	var totalItems int64
	if err := r.db.WithContext(ctx).Model(&models.Folder{}).
		Where("parent_id = ? AND tenant_id = ?", parentID, tenantID).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error counting child folders: %v", err))
	}

	return utils.NewPaginatedResult(folders, pagination, totalItems), nil
}

// GetRootFolders lists root folders for a tenant with pagination
func (r *postgresqlFolderRepository) GetRootFolders(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	if tenantID == "" {
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	var folders []models.Folder
	query := r.db.WithContext(ctx).Where("parent_id = '' AND tenant_id = ?", tenantID).
		Order("name ASC").
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit())

	if err := query.Find(&folders).Error; err != nil {
		return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error fetching root folders: %v", err))
	}

	// Count total items for pagination
	var totalItems int64
	if err := r.db.WithContext(ctx).Model(&models.Folder{}).
		Where("parent_id = '' AND tenant_id = ?", tenantID).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error counting root folders: %v", err))
	}

	return utils.NewPaginatedResult(folders, pagination, totalItems), nil
}

// GetFolderPath retrieves the full path of a folder by its ID with tenant isolation
func (r *postgresqlFolderRepository) GetFolderPath(ctx context.Context, id string, tenantID string) (string, error) {
	if id == "" {
		return "", errors.NewValidationError("folder ID cannot be empty")
	}
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID cannot be empty")
	}

	var folder models.Folder
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.NewNotFoundError(fmt.Sprintf("folder with ID %s not found", id))
		}
		return "", errors.NewInternalError(fmt.Sprintf("error fetching folder: %v", err))
	}

	return folder.Path, nil
}

// GetByPath retrieves a folder by its path with tenant isolation
func (r *postgresqlFolderRepository) GetByPath(ctx context.Context, path string, tenantID string) (*models.Folder, error) {
	if path == "" {
		return nil, errors.NewValidationError("folder path cannot be empty")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var folder models.Folder
	if err := r.db.WithContext(ctx).Where("path = ? AND tenant_id = ?", path, tenantID).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError(fmt.Sprintf("folder with path %s not found", path))
		}
		return nil, errors.NewInternalError(fmt.Sprintf("error fetching folder by path: %v", err))
	}

	return &folder, nil
}

// Move moves a folder to a new parent with tenant isolation
func (r *postgresqlFolderRepository) Move(ctx context.Context, id string, newParentID string, tenantID string) error {
	if id == "" {
		return errors.NewValidationError("folder ID cannot be empty")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Check if folder exists
	var folder models.Folder
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError(fmt.Sprintf("folder with ID %s not found", id))
		}
		return errors.NewInternalError(fmt.Sprintf("error fetching folder: %v", err))
	}

	// Check if new parent exists (if provided)
	var newParentPath string
	if newParentID != "" {
		var parentFolder models.Folder
		if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", newParentID, tenantID).First(&parentFolder).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError(fmt.Sprintf("parent folder with ID %s not found", newParentID))
			}
			return errors.NewInternalError(fmt.Sprintf("error fetching parent folder: %v", err))
		}

		// Check for circular reference (cannot move folder to its own descendant)
		if parentFolder.IsDescendantOf(folder.Path) {
			return errors.NewValidationError("cannot move a folder to its own descendant")
		}

		newParentPath = parentFolder.Path
	} else {
		// Moving to root
		newParentPath = ""
	}

	// Begin a transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Store the old path for updating descendants
	oldPath := folder.Path

	// Update the folder's parent ID
	folder.SetParent(newParentID)

	// Calculate the new path
	if newParentID == "" {
		folder.SetPath(models.PathSeparator + folder.Name)
	} else {
		folder.SetPath(folder.BuildPath(newParentPath))
	}

	// Update the folder
	if err := tx.Model(&models.Folder{}).Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{
			"parent_id":  folder.ParentID,
			"path":       folder.Path,
			"updated_at": folder.UpdatedAt,
		}).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to update folder: %v", err))
	}

	// Update all descendant folders' paths
	if err := r.updateDescendantPaths(tx, id, oldPath, folder.Path, tenantID); err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// Exists checks if a folder exists by its ID with tenant isolation
func (r *postgresqlFolderRepository) Exists(ctx context.Context, id string, tenantID string) (bool, error) {
	if id == "" {
		return false, errors.NewValidationError("folder ID cannot be empty")
	}
	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID cannot be empty")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Folder{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Count(&count).Error; err != nil {
		return false, errors.NewInternalError(fmt.Sprintf("error checking folder existence: %v", err))
	}

	return count > 0, nil
}

// IsEmpty checks if a folder is empty (has no child folders or documents) with tenant isolation
func (r *postgresqlFolderRepository) IsEmpty(ctx context.Context, id string, tenantID string) (bool, error) {
	if id == "" {
		return false, errors.NewValidationError("folder ID cannot be empty")
	}
	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Check if folder exists
	exists, err := r.Exists(ctx, id, tenantID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, errors.NewNotFoundError(fmt.Sprintf("folder with ID %s not found", id))
	}

	// Check for child folders
	var folderCount int64
	if err := r.db.WithContext(ctx).Model(&models.Folder{}).
		Where("parent_id = ? AND tenant_id = ?", id, tenantID).
		Count(&folderCount).Error; err != nil {
		return false, errors.NewInternalError(fmt.Sprintf("error checking child folders: %v", err))
	}

	// Check for documents in the folder
	var documentCount int64
	if err := r.db.WithContext(ctx).Table("documents").
		Where("folder_id = ? AND tenant_id = ?", id, tenantID).
		Count(&documentCount).Error; err != nil {
		return false, errors.NewInternalError(fmt.Sprintf("error checking documents in folder: %v", err))
	}

	return folderCount == 0 && documentCount == 0, nil
}

// Search searches folders by name with tenant isolation
func (r *postgresqlFolderRepository) Search(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	if query == "" {
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("search query cannot be empty")
	}
	if tenantID == "" {
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Prepare the search pattern
	searchPattern := "%" + query + "%"

	var folders []models.Folder
	dbQuery := r.db.WithContext(ctx).Where("name LIKE ? AND tenant_id = ?", searchPattern, tenantID).
		Order("name ASC").
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit())

	if err := dbQuery.Find(&folders).Error; err != nil {
		return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error searching folders: %v", err))
	}

	// Count total items for pagination
	var totalItems int64
	if err := r.db.WithContext(ctx).Model(&models.Folder{}).
		Where("name LIKE ? AND tenant_id = ?", searchPattern, tenantID).
		Count(&totalItems).Error; err != nil {
		return utils.PaginatedResult[models.Folder]{}, errors.NewInternalError(fmt.Sprintf("error counting search results: %v", err))
	}

	return utils.NewPaginatedResult(folders, pagination, totalItems), nil
}

// updateDescendantPaths updates paths of all descendant folders recursively when a folder is moved
func (r *postgresqlFolderRepository) updateDescendantPaths(tx *gorm.DB, folderID, oldPath, newPath, tenantID string) error {
	var descendants []models.Folder
	
	// Find all descendant folders
	if err := tx.Where("parent_id = ? AND tenant_id = ?", folderID, tenantID).Find(&descendants).Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("error fetching descendant folders: %v", err))
	}

	for _, descendant := range descendants {
		// Calculate new path
		updatedPath := strings.Replace(descendant.Path, oldPath+models.PathSeparator, newPath+models.PathSeparator, 1)
		
		// Update the path
		if err := tx.Model(&models.Folder{}).Where("id = ? AND tenant_id = ?", descendant.ID, tenantID).
			Update("path", updatedPath).Error; err != nil {
			return errors.NewInternalError(fmt.Sprintf("error updating descendant folder path: %v", err))
		}
		
		// Recursively update descendants of this folder
		if err := r.updateDescendantPaths(tx, descendant.ID, descendant.Path, updatedPath, tenantID); err != nil {
			return err
		}
	}

	return nil
}