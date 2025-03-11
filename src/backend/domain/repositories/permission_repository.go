// Package repositories provides repository interfaces for the domain layer
// of the Document Management Platform. These interfaces follow the repository pattern
// from Domain-Driven Design and ensure that domain logic remains decoupled from
// persistence implementation details.
package repositories

import (
	"context" // standard library

	"../models"      // For Permission domain model
	"../../pkg/utils" // For pagination utilities
)

// PermissionRepository defines the contract for permission persistence operations.
// It follows the repository pattern from Domain-Driven Design and provides an abstraction
// for permission-related database operations while maintaining tenant isolation.
type PermissionRepository interface {
	// Create creates a new permission in the repository.
	// It returns the ID of the created permission or an error if the operation fails.
	Create(ctx context.Context, permission *models.Permission) (string, error)

	// GetByID retrieves a permission by its ID with tenant isolation.
	// It returns the permission or an error if not found or if the tenant doesn't match.
	GetByID(ctx context.Context, id, tenantID string) (*models.Permission, error)

	// Update updates an existing permission with tenant isolation.
	// It returns an error if the operation fails or if the tenant doesn't match.
	Update(ctx context.Context, permission *models.Permission) error

	// Delete deletes a permission by its ID with tenant isolation.
	// It returns an error if the operation fails or if the tenant doesn't match.
	Delete(ctx context.Context, id, tenantID string) error

	// GetByResourceID retrieves permissions for a specific resource with tenant isolation.
	// It returns a list of permissions for the resource or an error if the operation fails.
	GetByResourceID(ctx context.Context, resourceType, resourceID, tenantID string) ([]*models.Permission, error)

	// GetByRoleID retrieves permissions for a specific role with pagination and tenant isolation.
	// It returns a paginated list of permissions for the role or an error if the operation fails.
	GetByRoleID(ctx context.Context, roleID, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Permission], error)

	// GetByTenant retrieves all permissions for a tenant with pagination.
	// It returns a paginated list of permissions for the tenant or an error if the operation fails.
	GetByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Permission], error)

	// CreateBulk creates multiple permissions in a single operation with tenant isolation.
	// It returns the IDs of the created permissions or an error if the operation fails.
	CreateBulk(ctx context.Context, permissions []*models.Permission) ([]string, error)

	// DeleteByResourceID deletes all permissions for a specific resource with tenant isolation.
	// It returns an error if the operation fails or if the tenant doesn't match.
	DeleteByResourceID(ctx context.Context, resourceType, resourceID, tenantID string) error

	// DeleteByRoleID deletes all permissions for a specific role with tenant isolation.
	// It returns an error if the operation fails or if the tenant doesn't match.
	DeleteByRoleID(ctx context.Context, roleID, tenantID string) error

	// CheckPermission checks if a role has a specific permission on a resource with tenant isolation.
	// It returns true if the permission exists, false otherwise, or an error if the operation fails.
	CheckPermission(ctx context.Context, roleID, resourceType, resourceID, permissionType, tenantID string) (bool, error)

	// GetInheritedPermissions retrieves inherited permissions for a folder with tenant isolation.
	// It returns a list of inherited permissions for the folder or an error if the operation fails.
	GetInheritedPermissions(ctx context.Context, folderID, tenantID string) ([]*models.Permission, error)

	// PropagatePermissions propagates permissions from a folder to all its subfolders with tenant isolation.
	// It returns an error if the operation fails or if the tenant doesn't match.
	PropagatePermissions(ctx context.Context, folderID, tenantID string) error
}