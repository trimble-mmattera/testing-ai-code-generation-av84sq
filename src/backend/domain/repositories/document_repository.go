// Package repositories provides repository interfaces for the Document Management Platform.
package repositories

import (
	"context" // standard library

	"../models"
	"../../pkg/utils"
)

// DocumentRepository defines the contract for document persistence operations.
// It provides methods for storing, retrieving, updating, and deleting documents
// while ensuring tenant isolation and supporting the platform's core features.
type DocumentRepository interface {
	// Create stores a new document in the repository and returns its ID.
	// It ensures proper tenant isolation by using the tenant ID in the document.
	Create(ctx context.Context, document *models.Document) (string, error)

	// GetByID retrieves a document by its ID with tenant isolation.
	// Returns the document if found and belongs to the specified tenant, otherwise an error.
	GetByID(ctx context.Context, id string, tenantID string) (*models.Document, error)

	// Update modifies an existing document with tenant isolation.
	// Only updates documents that belong to the specified tenant in the document.
	Update(ctx context.Context, document *models.Document) error

	// Delete removes a document by its ID with tenant isolation.
	// Only deletes documents that belong to the specified tenant.
	Delete(ctx context.Context, id string, tenantID string) error

	// ListByFolder retrieves documents in a specific folder with pagination and tenant isolation.
	// Only returns documents that belong to the specified tenant.
	ListByFolder(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// ListByTenant lists all documents for a tenant with pagination.
	ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// SearchByContent searches documents by their content with tenant isolation.
	// Only returns documents that belong to the specified tenant.
	SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// SearchByMetadata searches documents by their metadata with tenant isolation.
	// Only returns documents that belong to the specified tenant.
	SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)

	// AddVersion adds a new version to an existing document with tenant isolation.
	// Validates that the document exists and belongs to the tenant.
	AddVersion(ctx context.Context, version *models.DocumentVersion) (string, error)

	// GetVersionByID retrieves a document version by its ID with tenant isolation.
	GetVersionByID(ctx context.Context, versionID string, tenantID string) (*models.DocumentVersion, error)

	// UpdateVersionStatus updates the status of a document version with tenant isolation.
	UpdateVersionStatus(ctx context.Context, versionID string, status string, tenantID string) error

	// AddMetadata adds metadata to a document with tenant isolation.
	// Validates that the document exists and belongs to the specified tenant.
	AddMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) (string, error)

	// UpdateMetadata updates existing document metadata with tenant isolation.
	// Validates that the document exists and belongs to the specified tenant.
	UpdateMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) error

	// DeleteMetadata deletes document metadata by key with tenant isolation.
	// Validates that the document exists and belongs to the specified tenant.
	DeleteMetadata(ctx context.Context, documentID string, key string, tenantID string) error

	// GetDocumentsByIDs retrieves multiple documents by their IDs with tenant isolation.
	// Only returns documents that belong to the specified tenant.
	GetDocumentsByIDs(ctx context.Context, ids []string, tenantID string) ([]*models.Document, error)
}