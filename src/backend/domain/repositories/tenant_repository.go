// Package repositories contains the repository interfaces for the Document Management Platform.
package repositories

import (
	"context" // standard library - For context propagation in repository operations

	"../models"        // For tenant domain model
	"../../pkg/utils"  // For pagination utilities
)

// TenantRepository defines the contract for tenant persistence operations.
// It follows the repository pattern from Domain-Driven Design and Clean Architecture principles,
// allowing the domain layer to remain independent of the persistence implementation details.
type TenantRepository interface {
	// Create creates a new tenant in the repository
	// It returns the ID of the newly created tenant or an error if creation fails
	Create(ctx context.Context, tenant *models.Tenant) (string, error)

	// GetByID retrieves a tenant by its ID
	// It returns the tenant if found, or an error if not found or retrieval fails
	GetByID(ctx context.Context, id string) (*models.Tenant, error)

	// GetByName retrieves a tenant by its name
	// It returns the tenant if found, or an error if not found or retrieval fails
	GetByName(ctx context.Context, name string) (*models.Tenant, error)

	// Update updates an existing tenant
	// It returns an error if the update fails
	Update(ctx context.Context, tenant *models.Tenant) error

	// Delete deletes a tenant by its ID
	// It returns an error if the deletion fails
	Delete(ctx context.Context, id string) error

	// List lists all tenants with pagination
	// It returns a paginated result of tenants or an error if listing fails
	List(ctx context.Context, pagination *utils.Pagination) (utils.PaginatedResult[models.Tenant], error)

	// ListByStatus lists tenants by status with pagination
	// It returns a paginated result of tenants with the specified status or an error if listing fails
	ListByStatus(ctx context.Context, status string, pagination *utils.Pagination) (utils.PaginatedResult[models.Tenant], error)

	// UpdateStatus updates the status of a tenant
	// It returns an error if the update fails
	UpdateStatus(ctx context.Context, id string, status string) error

	// UpdateSettings updates the settings of a tenant
	// It returns an error if the update fails
	UpdateSettings(ctx context.Context, id string, settings map[string]string) error

	// GetSetting gets a specific setting of a tenant
	// It returns the setting value or an error if the setting doesn't exist or retrieval fails
	GetSetting(ctx context.Context, id string, key string) (string, error)

	// SetSetting sets a specific setting of a tenant
	// It returns an error if the operation fails
	SetSetting(ctx context.Context, id string, key string, value string) error

	// DeleteSetting deletes a specific setting of a tenant
	// It returns an error if the operation fails
	DeleteSetting(ctx context.Context, id string, key string) error

	// Exists checks if a tenant exists by ID
	// It returns true if the tenant exists, false otherwise, or an error if the check fails
	Exists(ctx context.Context, id string) (bool, error)

	// ExistsByName checks if a tenant exists by name
	// It returns true if the tenant exists, false otherwise, or an error if the check fails
	ExistsByName(ctx context.Context, name string) (bool, error)

	// Count counts the total number of tenants
	// It returns the count or an error if counting fails
	Count(ctx context.Context) (int64, error)

	// CountByStatus counts the number of tenants with a specific status
	// It returns the count or an error if counting fails
	CountByStatus(ctx context.Context, status string) (int64, error)
}