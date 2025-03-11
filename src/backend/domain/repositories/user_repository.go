// Package repositories provides repository interfaces for domain models.
package repositories

import (
	"context"

	"../models"
	"../../pkg/utils"
)

// UserRepository defines the interface for user persistence operations.
// It follows the repository pattern from Domain-Driven Design and Clean Architecture
// principles, allowing the domain layer to remain independent of the persistence
// implementation details.
type UserRepository interface {
	// Create creates a new user in the repository.
	// It returns the ID of the created user or an error if the operation fails.
	Create(ctx context.Context, user *models.User) (string, error)

	// GetByID retrieves a user by its ID with tenant isolation.
	// It returns the user or an error if not found or if the operation fails.
	GetByID(ctx context.Context, id string, tenantID string) (*models.User, error)

	// GetByUsername retrieves a user by username with tenant isolation.
	// It returns the user or an error if not found or if the operation fails.
	GetByUsername(ctx context.Context, username string, tenantID string) (*models.User, error)

	// GetByEmail retrieves a user by email with tenant isolation.
	// It returns the user or an error if not found or if the operation fails.
	GetByEmail(ctx context.Context, email string, tenantID string) (*models.User, error)

	// Update updates an existing user with tenant isolation.
	// It returns an error if the operation fails.
	Update(ctx context.Context, user *models.User) error

	// Delete deletes a user by its ID with tenant isolation.
	// It returns an error if the operation fails.
	Delete(ctx context.Context, id string, tenantID string) error

	// ListByTenant lists all users for a tenant with pagination.
	// It returns a paginated list of users or an error if the operation fails.
	ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.User], error)

	// ListByRole lists users with a specific role with tenant isolation and pagination.
	// It returns a paginated list of users or an error if the operation fails.
	ListByRole(ctx context.Context, role string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.User], error)

	// ListByStatus lists users with a specific status with tenant isolation and pagination.
	// It returns a paginated list of users or an error if the operation fails.
	ListByStatus(ctx context.Context, status string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.User], error)

	// UpdateStatus updates the status of a user with tenant isolation.
	// It returns an error if the operation fails.
	UpdateStatus(ctx context.Context, id string, status string, tenantID string) error

	// AddRole adds a role to a user with tenant isolation.
	// It returns an error if the operation fails.
	AddRole(ctx context.Context, id string, role string, tenantID string) error

	// RemoveRole removes a role from a user with tenant isolation.
	// It returns an error if the operation fails.
	RemoveRole(ctx context.Context, id string, role string, tenantID string) error

	// UpdatePassword updates the password of a user with tenant isolation.
	// It returns an error if the operation fails.
	UpdatePassword(ctx context.Context, id string, passwordHash string, tenantID string) error

	// SetSetting sets a user setting with tenant isolation.
	// It returns an error if the operation fails.
	SetSetting(ctx context.Context, id string, key string, value string, tenantID string) error

	// DeleteSetting deletes a user setting with tenant isolation.
	// It returns an error if the operation fails.
	DeleteSetting(ctx context.Context, id string, key string, tenantID string) error

	// GetSetting gets a user setting with tenant isolation.
	// It returns the setting value or an error if not found or if the operation fails.
	GetSetting(ctx context.Context, id string, key string, tenantID string) (string, error)

	// Exists checks if a user exists by ID with tenant isolation.
	// It returns true if the user exists, false otherwise, or an error if the operation fails.
	Exists(ctx context.Context, id string, tenantID string) (bool, error)

	// ExistsByUsername checks if a user exists by username with tenant isolation.
	// It returns true if the user exists, false otherwise, or an error if the operation fails.
	ExistsByUsername(ctx context.Context, username string, tenantID string) (bool, error)

	// ExistsByEmail checks if a user exists by email with tenant isolation.
	// It returns true if the user exists, false otherwise, or an error if the operation fails.
	ExistsByEmail(ctx context.Context, email string, tenantID string) (bool, error)

	// Count counts the total number of users for a tenant.
	// It returns the count or an error if the operation fails.
	Count(ctx context.Context, tenantID string) (int64, error)

	// CountByStatus counts the number of users with a specific status for a tenant.
	// It returns the count or an error if the operation fails.
	CountByStatus(ctx context.Context, status string, tenantID string) (int64, error)

	// CountByRole counts the number of users with a specific role for a tenant.
	// It returns the count or an error if the operation fails.
	CountByRole(ctx context.Context, role string, tenantID string) (int64, error)
}