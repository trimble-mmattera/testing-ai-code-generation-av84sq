// Package repositories defines repository interfaces for the document management system.
package repositories

import (
	"context" // standard library - For context propagation in repository operations

	"../models"       // For folder domain models
	"../../pkg/utils" // For pagination utilities
)

// FolderRepository defines the contract for folder persistence operations.
// It follows the repository pattern from Domain-Driven Design and ensures
// tenant isolation for all operations.
type FolderRepository interface {
	// Create creates a new folder in the repository.
	// It returns the ID of the created folder or an error if the operation fails.
	Create(ctx context.Context, folder *models.Folder) (string, error)

	// GetByID retrieves a folder by its ID with tenant isolation.
	// It returns the folder or an error if the folder is not found or the operation fails.
	GetByID(ctx context.Context, id string, tenantID string) (*models.Folder, error)

	// Update updates an existing folder with tenant isolation.
	// It returns an error if the operation fails.
	Update(ctx context.Context, folder *models.Folder) error

	// Delete deletes a folder by its ID with tenant isolation.
	// It returns an error if the operation fails.
	Delete(ctx context.Context, id string, tenantID string) error

	// GetChildren lists child folders of a parent folder with pagination and tenant isolation.
	// It returns a paginated list of child folders or an error if the operation fails.
	GetChildren(ctx context.Context, parentID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error)

	// GetRootFolders lists root folders for a tenant with pagination.
	// It returns a paginated list of root folders or an error if the operation fails.
	GetRootFolders(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error)

	// GetFolderPath retrieves the full path of a folder by its ID with tenant isolation.
	// It returns the full path of the folder or an error if the folder is not found or the operation fails.
	GetFolderPath(ctx context.Context, id string, tenantID string) (string, error)

	// GetByPath retrieves a folder by its path with tenant isolation.
	// It returns the folder or an error if the folder is not found or the operation fails.
	GetByPath(ctx context.Context, path string, tenantID string) (*models.Folder, error)

	// Move moves a folder to a new parent with tenant isolation.
	// It returns an error if the operation fails.
	Move(ctx context.Context, id string, newParentID string, tenantID string) error

	// Exists checks if a folder exists by its ID with tenant isolation.
	// It returns true if the folder exists, false otherwise, or an error if the operation fails.
	Exists(ctx context.Context, id string, tenantID string) (bool, error)

	// IsEmpty checks if a folder is empty (has no child folders) with tenant isolation.
	// It returns true if the folder is empty, false otherwise, or an error if the operation fails.
	IsEmpty(ctx context.Context, id string, tenantID string) (bool, error)

	// Search searches folders by name with tenant isolation.
	// It returns a paginated list of folders matching the search query or an error if the operation fails.
	Search(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error)
}