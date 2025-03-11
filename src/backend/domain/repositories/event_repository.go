// Package repositories defines the repository interfaces for domain persistence operations
package repositories

import (
	"context" // standard library - For context propagation in repository operations
	"time"    // standard library - For time-based operations in repository methods

	"../models"       // To reference the Event domain model for repository operations
	"../../pkg/utils" // For pagination support in repository methods
)

// EventRepository defines the contract for event persistence and retrieval operations
type EventRepository interface {
	// Create persists a new event to the repository
	Create(ctx context.Context, event *models.Event) (string, error)

	// GetByID retrieves an event by its ID
	GetByID(ctx context.Context, id string, tenantID string) (*models.Event, error)

	// ListByType lists events of a specific type with pagination
	ListByType(ctx context.Context, eventType string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error)

	// ListByTenant lists all events for a tenant with pagination
	ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error)

	// ListDocumentEvents lists events related to a specific document
	ListDocumentEvents(ctx context.Context, documentID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error)

	// ListFolderEvents lists events related to a specific folder
	ListFolderEvents(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error)

	// DeleteOlderThan deletes events older than a specified time
	DeleteOlderThan(ctx context.Context, olderThan time.Time, tenantID string) (int, error)
}