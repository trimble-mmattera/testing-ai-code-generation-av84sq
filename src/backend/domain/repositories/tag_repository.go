// Package repositories defines interfaces for domain entity persistence operations.
package repositories

import (
	"context"

	"../models" // For tag domain model
	"../../pkg/utils" // v1.21+ (standard library) For pagination utilities
)

// TagRepository defines the interface for tag-related persistence operations.
// It follows the repository pattern from Domain-Driven Design to abstract
// storage details from the domain layer and enforce tenant isolation.
type TagRepository interface {
	// Create persists a new tag in the repository.
	// Returns the ID of the created tag or an error if the operation fails.
	// The tag must include a valid tenant ID for isolation.
	Create(ctx context.Context, tag *models.Tag) (string, error)

	// GetByID retrieves a tag by its ID with tenant isolation.
	// Returns the tag if found or an error if not found or operation fails.
	GetByID(ctx context.Context, id string, tenantID string) (*models.Tag, error)

	// GetByName retrieves a tag by its name with tenant isolation.
	// Returns the tag if found or an error if not found or operation fails.
	GetByName(ctx context.Context, name string, tenantID string) (*models.Tag, error)

	// Update modifies an existing tag with tenant isolation enforcement.
	// Returns an error if the operation fails or the tag doesn't exist.
	Update(ctx context.Context, tag *models.Tag) error

	// Delete removes a tag by its ID with tenant isolation.
	// Returns an error if the operation fails or the tag doesn't exist.
	// This should also remove all tag associations from documents.
	Delete(ctx context.Context, id string, tenantID string) error

	// ListByTenant retrieves all tags for a tenant with pagination.
	// Returns a paginated list of tags or an error if the operation fails.
	ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Tag], error)

	// SearchByName finds tags matching a name pattern with tenant isolation.
	// Returns a paginated list of matching tags or an error if the operation fails.
	SearchByName(ctx context.Context, namePattern string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Tag], error)

	// AddTagToDocument associates a tag with a document with tenant isolation.
	// Returns an error if the operation fails or if either the tag or document
	// doesn't exist within the specified tenant.
	AddTagToDocument(ctx context.Context, tagID string, documentID string, tenantID string) error

	// RemoveTagFromDocument removes a tag association from a document with tenant isolation.
	// Returns an error if the operation fails or if the association doesn't exist
	// within the specified tenant.
	RemoveTagFromDocument(ctx context.Context, tagID string, documentID string, tenantID string) error

	// GetTagsByDocumentID retrieves all tags associated with a document with tenant isolation.
	// Returns the list of tags or an error if the operation fails.
	GetTagsByDocumentID(ctx context.Context, documentID string, tenantID string) ([]*models.Tag, error)

	// GetDocumentsByTagID retrieves all document IDs associated with a tag with tenant isolation.
	// Returns a paginated list of document IDs or an error if the operation fails.
	GetDocumentsByTagID(ctx context.Context, tagID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[string], error)
}