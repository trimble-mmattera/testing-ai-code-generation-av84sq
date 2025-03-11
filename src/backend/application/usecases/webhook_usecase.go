// Package usecases implements the application layer of the Document Management Platform.
// It contains use case implementations that orchestrate domain models and services.
package usecases

import (
	"context"
	"fmt"

	"../../domain/models"
	"../../domain/services"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// WebhookUseCase defines the contract for webhook application use cases
type WebhookUseCase interface {
	// CreateWebhook creates a new webhook subscription
	CreateWebhook(ctx context.Context, webhook *models.Webhook) (string, error)
	
	// GetWebhook retrieves a webhook by its ID
	GetWebhook(ctx context.Context, id string, tenantID string) (*models.Webhook, error)
	
	// UpdateWebhook updates an existing webhook
	UpdateWebhook(ctx context.Context, webhook *models.Webhook) error
	
	// DeleteWebhook deletes a webhook
	DeleteWebhook(ctx context.Context, id string, tenantID string) error
	
	// ListWebhooks lists webhooks for a tenant with pagination
	ListWebhooks(ctx context.Context, tenantID string, page int, pageSize int) (utils.PaginatedResult[models.Webhook], error)
	
	// ProcessEvent processes an event and delivers it to relevant webhooks
	ProcessEvent(ctx context.Context, event *models.Event) error
	
	// GetDeliveryStatus gets the status of a webhook delivery
	GetDeliveryStatus(ctx context.Context, deliveryID string, tenantID string) (*models.WebhookDelivery, error)
	
	// ListDeliveries lists delivery attempts for a webhook with pagination
	ListDeliveries(ctx context.Context, webhookID string, tenantID string, page int, pageSize int) (utils.PaginatedResult[models.WebhookDelivery], error)
	
	// RetryDelivery retries a failed webhook delivery
	RetryDelivery(ctx context.Context, deliveryID string, tenantID string) error
}

// webhookUseCase implements the WebhookUseCase interface
type webhookUseCase struct {
	webhookService services.WebhookService
	eventService   services.EventServiceInterface
}

// NewWebhookUseCase creates a new WebhookUseCase instance
func NewWebhookUseCase(webhookService services.WebhookService, eventService services.EventServiceInterface) (WebhookUseCase, error) {
	if webhookService == nil {
		return nil, fmt.Errorf("webhook service cannot be nil")
	}
	
	if eventService == nil {
		return nil, fmt.Errorf("event service cannot be nil")
	}
	
	return &webhookUseCase{
		webhookService: webhookService,
		eventService:   eventService,
	}, nil
}

// CreateWebhook creates a new webhook subscription
func (u *webhookUseCase) CreateWebhook(ctx context.Context, webhook *models.Webhook) (string, error) {
	log := logger.WithContext(ctx)
	
	if webhook == nil {
		log.Error("webhook cannot be nil")
		return "", errors.NewValidationError("webhook cannot be nil")
	}
	
	id, err := u.webhookService.CreateWebhook(ctx, webhook)
	if err != nil {
		log.WithError(err).Error("failed to create webhook")
		return "", errors.Wrap(err, "failed to create webhook")
	}
	
	log.Info("webhook created successfully", "id", id)
	return id, nil
}

// GetWebhook retrieves a webhook by its ID
func (u *webhookUseCase) GetWebhook(ctx context.Context, id string, tenantID string) (*models.Webhook, error) {
	log := logger.WithContext(ctx)
	
	if err := u.validateInput(map[string]string{
		"webhook ID": id,
		"tenant ID": tenantID,
	}); err != nil {
		return nil, err
	}
	
	webhook, err := u.webhookService.GetWebhook(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("failed to get webhook", "id", id, "tenantID", tenantID)
		return nil, errors.Wrap(err, "failed to get webhook")
	}
	
	log.Info("webhook retrieved successfully", "id", id)
	return webhook, nil
}

// UpdateWebhook updates an existing webhook
func (u *webhookUseCase) UpdateWebhook(ctx context.Context, webhook *models.Webhook) error {
	log := logger.WithContext(ctx)
	
	if webhook == nil {
		log.Error("webhook cannot be nil")
		return errors.NewValidationError("webhook cannot be nil")
	}
	
	err := u.webhookService.UpdateWebhook(ctx, webhook)
	if err != nil {
		log.WithError(err).Error("failed to update webhook", "id", webhook.ID)
		return errors.Wrap(err, "failed to update webhook")
	}
	
	log.Info("webhook updated successfully", "id", webhook.ID)
	return nil
}

// DeleteWebhook deletes a webhook
func (u *webhookUseCase) DeleteWebhook(ctx context.Context, id string, tenantID string) error {
	log := logger.WithContext(ctx)
	
	if err := u.validateInput(map[string]string{
		"webhook ID": id,
		"tenant ID": tenantID,
	}); err != nil {
		return err
	}
	
	err := u.webhookService.DeleteWebhook(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("failed to delete webhook", "id", id, "tenantID", tenantID)
		return errors.Wrap(err, "failed to delete webhook")
	}
	
	log.Info("webhook deleted successfully", "id", id)
	return nil
}

// ListWebhooks lists webhooks for a tenant with pagination
func (u *webhookUseCase) ListWebhooks(ctx context.Context, tenantID string, page int, pageSize int) (utils.PaginatedResult[models.Webhook], error) {
	log := logger.WithContext(ctx)
	
	if tenantID == "" {
		log.Error("tenant ID cannot be empty")
		return utils.PaginatedResult[models.Webhook]{}, errors.NewValidationError("tenant ID is required")
	}
	
	pagination := utils.NewPagination(page, pageSize)
	
	result, err := u.webhookService.ListWebhooks(ctx, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("failed to list webhooks", "tenantID", tenantID)
		return utils.PaginatedResult[models.Webhook]{}, errors.Wrap(err, "failed to list webhooks")
	}
	
	log.Info("webhooks listed successfully", "tenantID", tenantID, "count", len(result.Items))
	return result, nil
}

// ProcessEvent processes an event and delivers it to relevant webhooks
func (u *webhookUseCase) ProcessEvent(ctx context.Context, event *models.Event) error {
	log := logger.WithContext(ctx)
	
	if event == nil {
		log.Error("event cannot be nil")
		return errors.NewValidationError("event cannot be nil")
	}
	
	err := u.webhookService.ProcessEvent(ctx, event)
	if err != nil {
		log.WithError(err).Error("failed to process event", "eventID", event.ID, "eventType", event.Type)
		return errors.Wrap(err, "failed to process event")
	}
	
	log.Info("event processed successfully", "eventID", event.ID, "eventType", event.Type)
	return nil
}

// GetDeliveryStatus gets the status of a webhook delivery
func (u *webhookUseCase) GetDeliveryStatus(ctx context.Context, deliveryID string, tenantID string) (*models.WebhookDelivery, error) {
	log := logger.WithContext(ctx)
	
	if err := u.validateInput(map[string]string{
		"delivery ID": deliveryID,
		"tenant ID": tenantID,
	}); err != nil {
		return nil, err
	}
	
	delivery, err := u.webhookService.GetDeliveryStatus(ctx, deliveryID, tenantID)
	if err != nil {
		log.WithError(err).Error("failed to get delivery status", "deliveryID", deliveryID, "tenantID", tenantID)
		return nil, errors.Wrap(err, "failed to get delivery status")
	}
	
	log.Info("delivery status retrieved successfully", "deliveryID", deliveryID)
	return delivery, nil
}

// ListDeliveries lists delivery attempts for a webhook with pagination
func (u *webhookUseCase) ListDeliveries(ctx context.Context, webhookID string, tenantID string, page int, pageSize int) (utils.PaginatedResult[models.WebhookDelivery], error) {
	log := logger.WithContext(ctx)
	
	if err := u.validateInput(map[string]string{
		"webhook ID": webhookID,
		"tenant ID": tenantID,
	}); err != nil {
		return utils.PaginatedResult[models.WebhookDelivery]{}, err
	}
	
	pagination := utils.NewPagination(page, pageSize)
	
	result, err := u.webhookService.ListDeliveries(ctx, webhookID, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("failed to list deliveries", "webhookID", webhookID, "tenantID", tenantID)
		return utils.PaginatedResult[models.WebhookDelivery]{}, errors.Wrap(err, "failed to list deliveries")
	}
	
	log.Info("deliveries listed successfully", "webhookID", webhookID, "count", len(result.Items))
	return result, nil
}

// RetryDelivery retries a failed webhook delivery
func (u *webhookUseCase) RetryDelivery(ctx context.Context, deliveryID string, tenantID string) error {
	log := logger.WithContext(ctx)
	
	if err := u.validateInput(map[string]string{
		"delivery ID": deliveryID,
		"tenant ID": tenantID,
	}); err != nil {
		return err
	}
	
	err := u.webhookService.RetryDelivery(ctx, deliveryID, tenantID)
	if err != nil {
		log.WithError(err).Error("failed to retry delivery", "deliveryID", deliveryID, "tenantID", tenantID)
		return errors.Wrap(err, "failed to retry delivery")
	}
	
	log.Info("delivery retry initiated successfully", "deliveryID", deliveryID)
	return nil
}

// validateInput validates input parameters
func (u *webhookUseCase) validateInput(params map[string]string) error {
	for name, value := range params {
		if value == "" {
			return errors.NewValidationError(fmt.Sprintf("%s cannot be empty", name))
		}
	}
	return nil
}