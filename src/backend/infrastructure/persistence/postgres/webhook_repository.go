package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid" // v1.3.0+ - For generating unique IDs for webhooks and deliveries
	"gorm.io/gorm" // v1.25.0+ - ORM library for database operations

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/errors"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

// webhookRepository implements the WebhookRepository interface using PostgreSQL
type webhookRepository struct{}

// NewWebhookRepository creates a new instance of the PostgreSQL implementation of WebhookRepository
func NewWebhookRepository() repositories.WebhookRepository {
	return &webhookRepository{}
}

// Create persists a new webhook to the database
func (r *webhookRepository) Create(ctx context.Context, webhook *models.Webhook) (string, error) {
	// Validate the webhook
	if err := webhook.Validate(); err != nil {
		return "", err
	}

	// Generate a new UUID for the webhook ID if not provided
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	// Set timestamps if not already set
	now := time.Now()
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = now
	}
	if webhook.UpdatedAt.IsZero() {
		webhook.UpdatedAt = now
	}

	// Set default status if not provided
	if webhook.Status == "" {
		webhook.Status = models.WebhookStatusActive
	}

	// Get database connection
	db, err := GetDB()
	if err != nil {
		return "", err
	}

	// Create the webhook record
	if err := db.WithContext(ctx).Create(webhook).Error; err != nil {
		logger.Error("Failed to create webhook", "error", err, "webhook_id", webhook.ID, "tenant_id", webhook.TenantID)
		return "", errors.NewInternalError("Failed to create webhook: " + err.Error())
	}

	return webhook.ID, nil
}

// GetByID retrieves a webhook by its ID
func (r *webhookRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.Webhook, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	var webhook models.Webhook
	if err := db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&webhook).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError("Webhook not found")
		}
		logger.Error("Failed to get webhook", "error", err, "id", id, "tenant_id", tenantID)
		return nil, errors.NewInternalError("Failed to get webhook: " + err.Error())
	}

	return &webhook, nil
}

// Update updates an existing webhook in the database
func (r *webhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	// Validate the webhook
	if err := webhook.Validate(); err != nil {
		return err
	}

	// Update timestamp
	webhook.UpdatedAt = time.Now()

	db, err := GetDB()
	if err != nil {
		return err
	}

	// Ensure tenant isolation by including tenant_id in the update condition
	result := db.WithContext(ctx).Model(&models.Webhook{}).
		Where("id = ? AND tenant_id = ?", webhook.ID, webhook.TenantID).
		Updates(webhook)

	if result.Error != nil {
		logger.Error("Failed to update webhook", "error", result.Error, "id", webhook.ID, "tenant_id", webhook.TenantID)
		return errors.NewInternalError("Failed to update webhook: " + result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return errors.NewResourceNotFoundError("Webhook not found")
	}

	return nil
}

// Delete deletes a webhook from the database
func (r *webhookRepository) Delete(ctx context.Context, id string, tenantID string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	// Ensure tenant isolation by including tenant_id in the delete condition
	result := db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).Delete(&models.Webhook{})

	if result.Error != nil {
		logger.Error("Failed to delete webhook", "error", result.Error, "id", id, "tenant_id", tenantID)
		return errors.NewInternalError("Failed to delete webhook: " + result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return errors.NewResourceNotFoundError("Webhook not found")
	}

	return nil
}

// ListByTenant lists all webhooks for a tenant with pagination
func (r *webhookRepository) ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Webhook], error) {
	db, err := GetDB()
	if err != nil {
		return utils.PaginatedResult[models.Webhook]{}, err
	}

	var webhooks []models.Webhook
	var totalItems int64

	// Count total items for pagination
	if err := db.WithContext(ctx).Model(&models.Webhook{}).
		Where("tenant_id = ?", tenantID).
		Count(&totalItems).Error; err != nil {
		logger.Error("Failed to count webhooks", "error", err, "tenant_id", tenantID)
		return utils.PaginatedResult[models.Webhook]{}, errors.NewInternalError("Failed to count webhooks: " + err.Error())
	}

	// Get paginated results
	if err := db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Order("created_at DESC").
		Find(&webhooks).Error; err != nil {
		logger.Error("Failed to list webhooks", "error", err, "tenant_id", tenantID)
		return utils.PaginatedResult[models.Webhook]{}, errors.NewInternalError("Failed to list webhooks: " + err.Error())
	}

	return utils.NewPaginatedResult(webhooks, pagination, totalItems), nil
}

// ListByEventType lists webhooks that subscribe to a specific event type
func (r *webhookRepository) ListByEventType(ctx context.Context, eventType string, tenantID string) ([]*models.Webhook, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	var webhooks []*models.Webhook

	// Using PostgreSQL's array operators to find webhooks with the event type
	// This assumes event_types is stored as a string array in PostgreSQL
	query := db.WithContext(ctx).
		Where("tenant_id = ? AND status = ? AND ? = ANY(event_types)", 
			tenantID, models.WebhookStatusActive, eventType).
		Find(&webhooks)

	if query.Error != nil {
		logger.Error("Failed to list webhooks by event type", 
			"error", query.Error, "event_type", eventType, "tenant_id", tenantID)
		return nil, errors.NewInternalError("Failed to list webhooks by event type: " + query.Error.Error())
	}

	return webhooks, nil
}

// CreateDelivery creates a new webhook delivery record
func (r *webhookRepository) CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) (string, error) {
	// Generate a new UUID if one isn't provided
	if delivery.ID == "" {
		delivery.ID = uuid.New().String()
	}

	// Set timestamps if not already set
	now := time.Now()
	if delivery.CreatedAt.IsZero() {
		delivery.CreatedAt = now
	}
	if delivery.UpdatedAt.IsZero() {
		delivery.UpdatedAt = now
	}

	// Set default status if not provided
	if delivery.Status == "" {
		delivery.Status = models.WebhookDeliveryStatusPending
	}

	db, err := GetDB()
	if err != nil {
		return "", err
	}

	if err := db.WithContext(ctx).Create(delivery).Error; err != nil {
		logger.Error("Failed to create webhook delivery", 
			"error", err, "delivery_id", delivery.ID, "webhook_id", delivery.WebhookID)
		return "", errors.NewInternalError("Failed to create webhook delivery: " + err.Error())
	}

	return delivery.ID, nil
}

// UpdateDelivery updates an existing webhook delivery record
func (r *webhookRepository) UpdateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	// Update timestamp
	delivery.UpdatedAt = time.Now()

	db, err := GetDB()
	if err != nil {
		return err
	}

	result := db.WithContext(ctx).Model(&models.WebhookDelivery{}).
		Where("id = ?", delivery.ID).
		Updates(delivery)

	if result.Error != nil {
		logger.Error("Failed to update webhook delivery", 
			"error", result.Error, "delivery_id", delivery.ID)
		return errors.NewInternalError("Failed to update webhook delivery: " + result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return errors.NewResourceNotFoundError("Webhook delivery not found")
	}

	return nil
}

// GetDeliveryByID retrieves a webhook delivery record by its ID
func (r *webhookRepository) GetDeliveryByID(ctx context.Context, id string, tenantID string) (*models.WebhookDelivery, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	var delivery models.WebhookDelivery

	// Join with webhooks table to ensure tenant isolation
	if err := db.WithContext(ctx).
		Joins("JOIN webhooks ON webhook_deliveries.webhook_id = webhooks.id").
		Where("webhook_deliveries.id = ? AND webhooks.tenant_id = ?", id, tenantID).
		First(&delivery).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError("Webhook delivery not found")
		}
		logger.Error("Failed to get webhook delivery", 
			"error", err, "id", id, "tenant_id", tenantID)
		return nil, errors.NewInternalError("Failed to get webhook delivery: " + err.Error())
	}

	return &delivery, nil
}

// ListDeliveriesByWebhook lists delivery records for a specific webhook with pagination
func (r *webhookRepository) ListDeliveriesByWebhook(ctx context.Context, webhookID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.WebhookDelivery], error) {
	db, err := GetDB()
	if err != nil {
		return utils.PaginatedResult[models.WebhookDelivery]{}, err
	}

	var deliveries []models.WebhookDelivery
	var totalItems int64

	// Join with webhooks table to ensure tenant isolation
	baseQuery := db.WithContext(ctx).
		Table("webhook_deliveries").
		Joins("JOIN webhooks ON webhook_deliveries.webhook_id = webhooks.id").
		Where("webhook_deliveries.webhook_id = ? AND webhooks.tenant_id = ?", webhookID, tenantID)

	// Count total items for pagination
	if err := baseQuery.Count(&totalItems).Error; err != nil {
		logger.Error("Failed to count webhook deliveries", 
			"error", err, "webhook_id", webhookID, "tenant_id", tenantID)
		return utils.PaginatedResult[models.WebhookDelivery]{}, 
			errors.NewInternalError("Failed to count webhook deliveries: " + err.Error())
	}

	// Get paginated results
	if err := baseQuery.
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Order("webhook_deliveries.created_at DESC").
		Find(&deliveries).Error; err != nil {
		logger.Error("Failed to list webhook deliveries", 
			"error", err, "webhook_id", webhookID, "tenant_id", tenantID)
		return utils.PaginatedResult[models.WebhookDelivery]{}, 
			errors.NewInternalError("Failed to list webhook deliveries: " + err.Error())
	}

	return utils.NewPaginatedResult(deliveries, pagination, totalItems), nil
}

// ListPendingDeliveries lists pending delivery records for processing
func (r *webhookRepository) ListPendingDeliveries(ctx context.Context, limit int) ([]*models.WebhookDelivery, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	var deliveries []*models.WebhookDelivery

	if err := db.WithContext(ctx).
		Where("status = ?", models.WebhookDeliveryStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		logger.Error("Failed to list pending webhook deliveries", "error", err)
		return nil, errors.NewInternalError("Failed to list pending webhook deliveries: " + err.Error())
	}

	return deliveries, nil
}

// ListFailedDeliveries lists failed delivery records for retry
func (r *webhookRepository) ListFailedDeliveries(ctx context.Context, limit int, maxAttempts int) ([]*models.WebhookDelivery, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	var deliveries []*models.WebhookDelivery

	if err := db.WithContext(ctx).
		Where("status = ? AND attempt_count < ?", models.WebhookDeliveryStatusFailed, maxAttempts).
		Order("updated_at ASC"). // Order by last attempt time to allow for exponential backoff
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		logger.Error("Failed to list failed webhook deliveries", 
			"error", err, "max_attempts", maxAttempts)
		return nil, errors.NewInternalError("Failed to list failed webhook deliveries: " + err.Error())
	}

	return deliveries, nil
}