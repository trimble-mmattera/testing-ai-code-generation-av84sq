// Package repositories defines interfaces for data persistence operations in the domain layer,
// following the repository pattern from Domain-Driven Design and Clean Architecture principles.
package repositories

import (
	"context" // standard library

	"../models"
	"../../pkg/utils"
)

// WebhookRepository defines the contract for webhook persistence and retrieval operations.
// This interface allows the domain layer to remain independent of the webhook storage implementation details.
type WebhookRepository interface {
	// Create persists a new webhook to the repository
	Create(ctx context.Context, webhook *models.Webhook) (string, error)

	// GetByID retrieves a webhook by its ID
	GetByID(ctx context.Context, id string, tenantID string) (*models.Webhook, error)

	// Update updates an existing webhook in the repository
	Update(ctx context.Context, webhook *models.Webhook) error

	// Delete deletes a webhook from the repository
	Delete(ctx context.Context, id string, tenantID string) error

	// ListByTenant lists all webhooks for a tenant with pagination
	ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Webhook], error)

	// ListByEventType lists webhooks that subscribe to a specific event type
	ListByEventType(ctx context.Context, eventType string, tenantID string) ([]*models.Webhook, error)

	// CreateDelivery creates a new webhook delivery record
	CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) (string, error)

	// UpdateDelivery updates an existing webhook delivery record
	UpdateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error

	// GetDeliveryByID retrieves a webhook delivery record by its ID
	GetDeliveryByID(ctx context.Context, id string, tenantID string) (*models.WebhookDelivery, error)

	// ListDeliveriesByWebhook lists delivery records for a specific webhook with pagination
	ListDeliveriesByWebhook(ctx context.Context, webhookID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.WebhookDelivery], error)

	// ListPendingDeliveries lists pending delivery records for processing
	ListPendingDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error)

	// ListFailedDeliveries lists failed delivery records for retry
	ListFailedDeliveries(ctx context.Context, limit int, maxAttempts int) ([]*models.WebhookDelivery, error)
}