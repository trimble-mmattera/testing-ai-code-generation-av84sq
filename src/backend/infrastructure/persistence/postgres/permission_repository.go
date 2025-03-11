// Package postgres provides PostgreSQL implementations of repository interfaces
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid" // v1.3.0+
	"gorm.io/gorm"           // v1.25.0+

	"src/backend/domain/models"
	"src/backend/domain/repositories"
	"src/backend/pkg/errors"
	"src/backend/pkg/utils"
)

// postgresqlPermissionRepository is a PostgreSQL implementation of the PermissionRepository interface
type postgresqlPermissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository creates a new PostgreSQL permission repository instance
func NewPermissionRepository(db *gorm.DB) (repositories.PermissionRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	return &postgresqlPermissionRepository{
		db: db,
	}, nil
}

// Create creates a new permission in the repository
func (r *postgresqlPermissionRepository) Create(ctx context.Context, permission *models.Permission) (string, error) {
	if err := permission.Validate(); err != nil {
		return "", errors.NewValidationError(err.Error())
	}

	// Generate UUID for the permission ID if not provided
	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Create the permission record
	if err := tx.Create(permission).Error; err != nil {
		tx.Rollback()
		return "", errors.NewInternalError(fmt.Sprintf("failed to create permission: %v", err))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return "", errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return permission.ID, nil
}

// GetByID retrieves a permission by its ID with tenant isolation
func (r *postgresqlPermissionRepository) GetByID(ctx context.Context, id, tenantID string) (*models.Permission, error) {
	if id == "" {
		return nil, errors.NewValidationError("permission ID cannot be empty")
	}

	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var permission models.Permission
	result := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&permission)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("permission with ID %s not found", id))
		}
		return nil, errors.NewInternalError(fmt.Sprintf("failed to get permission: %v", result.Error))
	}

	return &permission, nil
}

// Update updates an existing permission with tenant isolation
func (r *postgresqlPermissionRepository) Update(ctx context.Context, permission *models.Permission) error {
	if err := permission.Validate(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	if permission.ID == "" {
		return errors.NewValidationError("permission ID cannot be empty")
	}

	// Check if permission exists with matching ID and tenant ID
	var existingPermission models.Permission
	result := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", permission.ID, permission.TenantID).First(&existingPermission)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("permission with ID %s not found", permission.ID))
		}
		return errors.NewInternalError(fmt.Sprintf("failed to check permission existence: %v", result.Error))
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Update the permission
	if err := tx.Model(&models.Permission{}).Where("id = ? AND tenant_id = ?", permission.ID, permission.TenantID).Updates(permission).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to update permission: %v", err))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// Delete deletes a permission by its ID with tenant isolation
func (r *postgresqlPermissionRepository) Delete(ctx context.Context, id, tenantID string) error {
	if id == "" {
		return errors.NewValidationError("permission ID cannot be empty")
	}

	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Check if permission exists with matching ID and tenant ID
	var existingPermission models.Permission
	result := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&existingPermission)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("permission with ID %s not found", id))
		}
		return errors.NewInternalError(fmt.Sprintf("failed to check permission existence: %v", result.Error))
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Delete the permission
	if err := tx.Delete(&models.Permission{}, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to delete permission: %v", err))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// GetByResourceID retrieves permissions for a specific resource with tenant isolation
func (r *postgresqlPermissionRepository) GetByResourceID(ctx context.Context, resourceType, resourceID, tenantID string) ([]*models.Permission, error) {
	if resourceType == "" {
		return nil, errors.NewValidationError("resource type cannot be empty")
	}

	if resourceID == "" {
		return nil, errors.NewValidationError("resource ID cannot be empty")
	}

	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var permissions []*models.Permission
	result := r.db.WithContext(ctx).Where(
		"resource_type = ? AND resource_id = ? AND tenant_id = ?",
		resourceType, resourceID, tenantID,
	).Find(&permissions)

	if result.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to get permissions by resource: %v", result.Error))
	}

	return permissions, nil
}

// GetByRoleID retrieves permissions for a specific role with pagination and tenant isolation
func (r *postgresqlPermissionRepository) GetByRoleID(ctx context.Context, roleID, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Permission], error) {
	if roleID == "" {
		return utils.PaginatedResult[models.Permission]{}, errors.NewValidationError("role ID cannot be empty")
	}

	if tenantID == "" {
		return utils.PaginatedResult[models.Permission]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	var permissions []models.Permission
	query := r.db.WithContext(ctx).Where("role_id = ? AND tenant_id = ?", roleID, tenantID)

	// Count total matching records
	var total int64
	if err := query.Model(&models.Permission{}).Count(&total).Error; err != nil {
		return utils.PaginatedResult[models.Permission]{}, errors.NewInternalError(fmt.Sprintf("failed to count permissions: %v", err))
	}

	// Apply pagination if provided
	if pagination != nil {
		query = query.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())
	}

	// Execute the query
	if err := query.Find(&permissions).Error; err != nil {
		return utils.PaginatedResult[models.Permission]{}, errors.NewInternalError(fmt.Sprintf("failed to get permissions by role: %v", err))
	}

	// Return paginated result
	return utils.NewPaginatedResult(permissions, pagination, total), nil
}

// GetByTenant retrieves all permissions for a tenant with pagination
func (r *postgresqlPermissionRepository) GetByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Permission], error) {
	if tenantID == "" {
		return utils.PaginatedResult[models.Permission]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	var permissions []models.Permission
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Count total matching records
	var total int64
	if err := query.Model(&models.Permission{}).Count(&total).Error; err != nil {
		return utils.PaginatedResult[models.Permission]{}, errors.NewInternalError(fmt.Sprintf("failed to count permissions: %v", err))
	}

	// Apply pagination if provided
	if pagination != nil {
		query = query.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())
	}

	// Execute the query
	if err := query.Find(&permissions).Error; err != nil {
		return utils.PaginatedResult[models.Permission]{}, errors.NewInternalError(fmt.Sprintf("failed to get permissions by tenant: %v", err))
	}

	// Return paginated result
	return utils.NewPaginatedResult(permissions, pagination, total), nil
}

// CreateBulk creates multiple permissions in a single operation with tenant isolation
func (r *postgresqlPermissionRepository) CreateBulk(ctx context.Context, permissions []*models.Permission) ([]string, error) {
	if len(permissions) == 0 {
		return []string{}, nil
	}

	// Validate all permissions and generate IDs if needed
	ids := make([]string, len(permissions))
	for i, permission := range permissions {
		if err := permission.Validate(); err != nil {
			return nil, errors.NewValidationError(err.Error())
		}

		// Generate UUID for the permission ID if not provided
		if permission.ID == "" {
			permission.ID = uuid.New().String()
		}

		ids[i] = permission.ID
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Create all permissions
	if err := tx.Create(permissions).Error; err != nil {
		tx.Rollback()
		return nil, errors.NewInternalError(fmt.Sprintf("failed to create permissions in bulk: %v", err))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return ids, nil
}

// DeleteByResourceID deletes all permissions for a specific resource with tenant isolation
func (r *postgresqlPermissionRepository) DeleteByResourceID(ctx context.Context, resourceType, resourceID, tenantID string) error {
	if resourceType == "" {
		return errors.NewValidationError("resource type cannot be empty")
	}

	if resourceID == "" {
		return errors.NewValidationError("resource ID cannot be empty")
	}

	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Delete permissions for the resource
	if err := tx.Where(
		"resource_type = ? AND resource_id = ? AND tenant_id = ?",
		resourceType, resourceID, tenantID,
	).Delete(&models.Permission{}).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to delete permissions by resource: %v", err))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// DeleteByRoleID deletes all permissions for a specific role with tenant isolation
func (r *postgresqlPermissionRepository) DeleteByRoleID(ctx context.Context, roleID, tenantID string) error {
	if roleID == "" {
		return errors.NewValidationError("role ID cannot be empty")
	}

	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// Delete permissions for the role
	if err := tx.Where(
		"role_id = ? AND tenant_id = ?",
		roleID, tenantID,
	).Delete(&models.Permission{}).Error; err != nil {
		tx.Rollback()
		return errors.NewInternalError(fmt.Sprintf("failed to delete permissions by role: %v", err))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// CheckPermission checks if a role has a specific permission on a resource with tenant isolation
func (r *postgresqlPermissionRepository) CheckPermission(ctx context.Context, roleID, resourceType, resourceID, permissionType, tenantID string) (bool, error) {
	if roleID == "" {
		return false, errors.NewValidationError("role ID cannot be empty")
	}

	if resourceType == "" {
		return false, errors.NewValidationError("resource type cannot be empty")
	}

	if resourceID == "" {
		return false, errors.NewValidationError("resource ID cannot be empty")
	}

	if permissionType == "" {
		return false, errors.NewValidationError("permission type cannot be empty")
	}

	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Check direct permission
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Permission{}).Where(
		"role_id = ? AND resource_type = ? AND resource_id = ? AND permission_type = ? AND tenant_id = ?",
		roleID, resourceType, resourceID, permissionType, tenantID,
	).Count(&count).Error; err != nil {
		return false, errors.NewInternalError(fmt.Sprintf("failed to check permission: %v", err))
	}

	if count > 0 {
		return true, nil
	}

	// If checking for folder permissions, also check for admin permission
	if permissionType != models.PermissionTypeAdmin && resourceType == models.ResourceTypeFolder {
		if err := r.db.WithContext(ctx).Model(&models.Permission{}).Where(
			"role_id = ? AND resource_type = ? AND resource_id = ? AND permission_type = ? AND tenant_id = ?",
			roleID, resourceType, resourceID, models.PermissionTypeAdmin, tenantID,
		).Count(&count).Error; err != nil {
			return false, errors.NewInternalError(fmt.Sprintf("failed to check admin permission: %v", err))
		}

		if count > 0 {
			return true, nil
		}

		// Check for inherited permissions if it's a folder resource
		permissions, err := r.GetInheritedPermissions(ctx, resourceID, tenantID)
		if err != nil {
			return false, err
		}

		for _, perm := range permissions {
			if perm.RoleID == roleID && (perm.PermissionType == permissionType || perm.PermissionType == models.PermissionTypeAdmin) {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetInheritedPermissions retrieves inherited permissions for a folder with tenant isolation
func (r *postgresqlPermissionRepository) GetInheritedPermissions(ctx context.Context, folderID, tenantID string) ([]*models.Permission, error) {
	if folderID == "" {
		return nil, errors.NewValidationError("folder ID cannot be empty")
	}

	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get the folder's path
	type Folder struct {
		Path string
	}
	var folder Folder

	if err := r.db.WithContext(ctx).Table("folders").
		Select("path").
		Where("id = ? AND tenant_id = ?", folderID, tenantID).
		First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("folder with ID %s not found", folderID))
		}
		return nil, errors.NewInternalError(fmt.Sprintf("failed to get folder path: %v", err))
	}

	// Extract parent folder paths from the path
	parentPaths := extractParentPaths(folder.Path)
	if len(parentPaths) == 0 {
		return []*models.Permission{}, nil
	}

	// Get parent folder IDs from paths
	type ParentFolder struct {
		ID string
	}
	var parentFolders []ParentFolder

	if err := r.db.WithContext(ctx).Table("folders").
		Select("id").
		Where("path IN ? AND tenant_id = ?", parentPaths, tenantID).
		Find(&parentFolders).Error; err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to get parent folders: %v", err))
	}

	if len(parentFolders) == 0 {
		return []*models.Permission{}, nil
	}

	// Extract parent folder IDs
	parentIDs := make([]string, len(parentFolders))
	for i, folder := range parentFolders {
		parentIDs[i] = folder.ID
	}

	// Get permissions for parent folders
	var permissions []*models.Permission
	if err := r.db.WithContext(ctx).Where(
		"resource_type = ? AND resource_id IN ? AND tenant_id = ?",
		models.ResourceTypeFolder, parentIDs, tenantID,
	).Find(&permissions).Error; err != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("failed to get inherited permissions: %v", err))
	}

	// Mark each permission as inherited
	for _, perm := range permissions {
		perm.MarkAsInherited()
	}

	return permissions, nil
}

// PropagatePermissions propagates permissions from a folder to all its subfolders with tenant isolation
func (r *postgresqlPermissionRepository) PropagatePermissions(ctx context.Context, folderID, tenantID string) error {
	if folderID == "" {
		return errors.NewValidationError("folder ID cannot be empty")
	}

	if tenantID == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get the folder's permissions
	permissions, err := r.GetByResourceID(ctx, models.ResourceTypeFolder, folderID, tenantID)
	if err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to get folder permissions: %v", err))
	}

	if len(permissions) == 0 {
		return nil // No permissions to propagate
	}

	// Get the folder's path
	type Folder struct {
		Path string
	}
	var folder Folder

	if err := r.db.WithContext(ctx).Table("folders").
		Select("path").
		Where("id = ? AND tenant_id = ?", folderID, tenantID).
		First(&folder).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("folder with ID %s not found", folderID))
		}
		return errors.NewInternalError(fmt.Sprintf("failed to get folder path: %v", err))
	}

	// Get all subfolders
	type Subfolder struct {
		ID string
	}
	var subfolders []Subfolder

	if err := r.db.WithContext(ctx).Table("folders").
		Select("id").
		Where("path LIKE ? AND tenant_id = ? AND id != ?", folder.Path+"%", tenantID, folderID).
		Find(&subfolders).Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to get subfolders: %v", err))
	}

	if len(subfolders) == 0 {
		return nil // No subfolders to propagate to
	}

	// Begin transaction
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to begin transaction: %v", tx.Error))
	}

	// For each subfolder, create inherited permissions
	for _, subfolder := range subfolders {
		for _, perm := range permissions {
			// Clone the permission for the subfolder
			inherited := perm.Clone(subfolder.ID)

			// Check if permission already exists
			var count int64
			if err := tx.Model(&models.Permission{}).Where(
				"role_id = ? AND resource_type = ? AND resource_id = ? AND permission_type = ? AND tenant_id = ?",
				inherited.RoleID, inherited.ResourceType, inherited.ResourceID, inherited.PermissionType, inherited.TenantID,
			).Count(&count).Error; err != nil {
				tx.Rollback()
				return errors.NewInternalError(fmt.Sprintf("failed to check existing permission: %v", err))
			}

			// Skip if permission already exists
			if count > 0 {
				continue
			}

			// Create the inherited permission
			if err := tx.Create(inherited).Error; err != nil {
				tx.Rollback()
				return errors.NewInternalError(fmt.Sprintf("failed to create inherited permission: %v", err))
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return errors.NewInternalError(fmt.Sprintf("failed to commit transaction: %v", err))
	}

	return nil
}

// extractParentPaths extracts parent folder paths from a given path
// For example, for a path "/tenant1/folder1/folder2/folder3",
// it would return ["/tenant1", "/tenant1/folder1", "/tenant1/folder1/folder2"]
func extractParentPaths(path string) []string {
	if path == "" || path == "/" {
		return []string{}
	}

	path = strings.TrimRight(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) <= 2 {
		return []string{}
	}

	paths := make([]string, 0, len(parts)-2)
	currentPath := ""

	for i := 1; i < len(parts)-1; i++ {
		if currentPath == "" {
			currentPath = "/" + parts[i]
		} else {
			currentPath = currentPath + "/" + parts[i]
		}
		paths = append(paths, currentPath)
	}

	return paths
}