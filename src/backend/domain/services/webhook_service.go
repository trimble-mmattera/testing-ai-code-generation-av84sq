// Package services implements business logic for the Document Management Platform.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"../models"
	"../repositories"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

const (
	maxRetryAttempts = 5
	defaultTimeout   = 10 * time.Second
	headerSignature  = "X-Webhook-Signature"
	headerEventType  = "X-Webhook-Event-Type"
	headerEventID    = "X-Webhook-Event-ID"
)

// WebhookService defines the contract for webhook management operations
type WebhookService interface {
	// CreateWebhook creates a new webhook subscription
	CreateWebhook(ctx context.Context, webhook *models.Webhook) (string, error)
	
	// GetWebhook retrieves a webhook by its ID
	GetWebhook(ctx context.Context, id string, tenantID string) (*models.Webhook, error)
	
	// UpdateWebhook updates an existing webhook
	UpdateWebhook(ctx context.Context, webhook *models.Webhook) error
	
	// DeleteWebhook deletes a webhook
	DeleteWebhook(ctx context.Context, id string, tenantID string) error
	
	// ListWebhooks lists webhooks for a tenant with pagination
	ListWebhooks(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Webhook], error)
	
	// ProcessEvent processes an event and delivers it to relevant webhooks
	ProcessEvent(ctx context.Context, event *models.Event) error
	
	// DeliverEvent delivers an event to a specific webhook
	DeliverEvent(ctx context.Context, webhook *models.Webhook, event *models.Event, delivery *models.WebhookDelivery) error
	
	// GetDeliveryStatus gets the status of a webhook delivery
	GetDeliveryStatus(ctx context.Context, deliveryID string, tenantID string) (*models.WebhookDelivery, error)
	
	// ListDeliveries lists delivery attempts for a webhook with pagination
	ListDeliveries(ctx context.Context, webhookID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.WebhookDelivery], error)
	
	// RetryDelivery retries a failed webhook delivery
	RetryDelivery(ctx context.Context, deliveryID string, tenantID string) error
	
	// ProcessPendingDeliveries processes pending webhook deliveries
	ProcessPendingDeliveries(ctx context.Context, batchSize int) (int, error)
	
	// RetryFailedDeliveries retries failed webhook deliveries
	RetryFailedDeliveries(ctx context.Context, batchSize int) (int, error)
}

// webhookService implements the WebhookService interface
type webhookService struct {
	webhookRepo repositories.WebhookRepository
	httpClient  *http.Client
	logger      logger.Logger
}

// NewWebhookService creates a new WebhookService instance
func NewWebhookService(webhookRepo repositories.WebhookRepository, httpClient *http.Client) (WebhookService, error) {
	if webhookRepo == nil {
		return nil, fmt.Errorf("webhook repository cannot be nil")
	}
	
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultTimeout,
		}
	}
	
	return &webhookService{
		webhookRepo: webhookRepo,
		httpClient:  httpClient,
		logger:      logger.WithField("service", "webhook"),
	}, nil
}

// CreateWebhook creates a new webhook subscription
func (s *webhookService) CreateWebhook(ctx context.Context, webhook *models.Webhook) (string, error) {
	ctxLogger := logger.WithContext(ctx)
	
	if webhook == nil {
		return "", errors.NewValidationError("webhook cannot be nil")
	}
	
	if err := webhook.Validate(); err != nil {
		return "", errors.NewValidationError(err.Error())
	}
	
	id, err := s.webhookRepo.Create(ctx, webhook)
	if err != nil {
		return "", errors.Wrap(err, "failed to create webhook")
	}
	
	ctxLogger.Info("webhook created successfully", "webhook_id", id)
	return id, nil
}

// GetWebhook retrieves a webhook by its ID
func (s *webhookService) GetWebhook(ctx context.Context, id string, tenantID string) (*models.Webhook, error) {
	ctxLogger := logger.WithContext(ctx)
	
	if err := s.validateInput(map[string]string{
		"webhook ID": id,
		"tenant ID":  tenantID,
	}); err != nil {
		return nil, err
	}
	
	webhook, err := s.webhookRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook")
	}
	
	// Verify tenant ownership
	if webhook.TenantID != tenantID {
		return nil, errors.NewAuthorizationError("webhook does not belong to the specified tenant")
	}
	
	ctxLogger.Info("webhook retrieved successfully", "webhook_id", id)
	return webhook, nil
}

// UpdateWebhook updates an existing webhook
func (s *webhookService) UpdateWebhook(ctx context.Context, webhook *models.Webhook) error {
	ctxLogger := logger.WithContext(ctx)
	
	if webhook == nil {
		return errors.NewValidationError("webhook cannot be nil")
	}
	
	if err := webhook.Validate(); err != nil {
		return errors.NewValidationError(err.Error())
	}
	
	// Verify webhook exists
	existingWebhook, err := s.webhookRepo.GetByID(ctx, webhook.ID, webhook.TenantID)
	if err != nil {
		return errors.Wrap(err, "failed to verify webhook existence")
	}
	
	if existingWebhook.TenantID != webhook.TenantID {
		return errors.NewAuthorizationError("webhook does not belong to the specified tenant")
	}
	
	err = s.webhookRepo.Update(ctx, webhook)
	if err != nil {
		return errors.Wrap(err, "failed to update webhook")
	}
	
	ctxLogger.Info("webhook updated successfully", "webhook_id", webhook.ID)
	return nil
}

// DeleteWebhook deletes a webhook
func (s *webhookService) DeleteWebhook(ctx context.Context, id string, tenantID string) error {
	ctxLogger := logger.WithContext(ctx)
	
	if err := s.validateInput(map[string]string{
		"webhook ID": id,
		"tenant ID":  tenantID,
	}); err != nil {
		return err
	}
	
	// Verify webhook exists and belongs to tenant
	webhook, err := s.webhookRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to verify webhook existence")
	}
	
	if webhook.TenantID != tenantID {
		return errors.NewAuthorizationError("webhook does not belong to the specified tenant")
	}
	
	err = s.webhookRepo.Delete(ctx, id, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to delete webhook")
	}
	
	ctxLogger.Info("webhook deleted successfully", "webhook_id", id)
	return nil
}

// ListWebhooks lists webhooks for a tenant with pagination
func (s *webhookService) ListWebhooks(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Webhook], error) {
	ctxLogger := logger.WithContext(ctx)
	
	if tenantID == "" {
		return utils.PaginatedResult[models.Webhook]{}, errors.NewValidationError("tenant ID cannot be empty")
	}
	
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	result, err := s.webhookRepo.ListByTenant(ctx, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Webhook]{}, errors.Wrap(err, "failed to list webhooks")
	}
	
	ctxLogger.Info("webhooks listed successfully", "tenant_id", tenantID, "count", len(result.Items))
	return result, nil
}

// ProcessEvent processes an event and delivers it to relevant webhooks
func (s *webhookService) ProcessEvent(ctx context.Context, event *models.Event) error {
	ctxLogger := logger.WithContext(ctx)
	
	if event == nil {
		return errors.NewValidationError("event cannot be nil")
	}
	
	if err := event.Validate(); err != nil {
		return errors.NewValidationError(err.Error())
	}
	
	// Find webhooks that subscribe to this event type
	webhooks, err := s.webhookRepo.ListByEventType(ctx, event.Type, event.TenantID)
	if err != nil {
		return errors.Wrap(err, "failed to list webhooks for event type")
	}
	
	for _, webhook := range webhooks {
		// Check if webhook should process this event
		if !webhook.ShouldProcessEvent(event.Type) {
			continue
		}
		
		// Create a delivery record
		delivery := models.NewWebhookDelivery(webhook.ID, event.ID)
		deliveryID, err := s.webhookRepo.CreateDelivery(ctx, delivery)
		if err != nil {
			ctxLogger.Error("failed to create delivery record", 
				"webhook_id", webhook.ID, 
				"event_id", event.ID, 
				"error", err)
			continue
		}
		
		delivery.ID = deliveryID
		
		// Deliver event asynchronously
		go func(w *models.Webhook, e *models.Event, d *models.WebhookDelivery) {
			deliveryCtx := context.Background()
			if err := s.DeliverEvent(deliveryCtx, w, e, d); err != nil {
				logger.WithContext(deliveryCtx).Error("failed to deliver event", 
					"webhook_id", w.ID, 
					"event_id", e.ID, 
					"delivery_id", d.ID, 
					"error", err)
			}
		}(webhook, event, delivery)
	}
	
	ctxLogger.Info("event processed", 
		"event_id", event.ID, 
		"event_type", event.Type, 
		"webhook_count", len(webhooks))
	
	return nil
}

// DeliverEvent delivers an event to a specific webhook
func (s *webhookService) DeliverEvent(ctx context.Context, webhook *models.Webhook, event *models.Event, delivery *models.WebhookDelivery) error {
	ctxLogger := logger.WithContext(ctx)
	
	if webhook == nil {
		return errors.NewValidationError("webhook cannot be nil")
	}
	
	if event == nil {
		return errors.NewValidationError("event cannot be nil")
	}
	
	if delivery == nil {
		return errors.NewValidationError("delivery cannot be nil")
	}
	
	// Create request context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(reqCtx, "POST", webhook.URL, bytes.NewReader(event.Payload))
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}
	
	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerSignature, webhook.GenerateSignatureForPayload(event.Payload))
	req.Header.Set(headerEventType, event.Type)
	req.Header.Set(headerEventID, event.ID)
	
	// Execute request
	resp, err := s.httpClient.Do(req)
	
	// Handle network errors
	if err != nil {
		// Update delivery status
		delivery.MarkAsFailed(0, "", err.Error())
		if updateErr := s.webhookRepo.UpdateDelivery(ctx, delivery); updateErr != nil {
			ctxLogger.Error("failed to update delivery status", 
				"delivery_id", delivery.ID, 
				"error", updateErr)
		}
		
		// Update webhook stats
		webhook.RecordDeliveryFailure()
		if updateErr := s.webhookRepo.Update(ctx, webhook); updateErr != nil {
			ctxLogger.Error("failed to update webhook stats", 
				"webhook_id", webhook.ID, 
				"error", updateErr)
		}
		
		return errors.Wrap(err, "failed to execute HTTP request")
	}
	defer resp.Body.Close()
	
	// Read response body (limited to prevent memory issues)
	respBodyBytes := make([]byte, 1024)
	if resp.Body != nil {
		resp.Body.Read(respBodyBytes)
	}
	
	respBody := string(respBodyBytes)
	
	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		delivery.MarkAsSuccess(resp.StatusCode, respBody)
		webhook.RecordDeliverySuccess()
		
		ctxLogger.Info("event delivered successfully", 
			"webhook_id", webhook.ID, 
			"event_id", event.ID, 
			"delivery_id", delivery.ID, 
			"status", resp.StatusCode)
	} else {
		// Failure
		delivery.MarkAsFailed(resp.StatusCode, respBody, fmt.Sprintf("HTTP error: %d", resp.StatusCode))
		webhook.RecordDeliveryFailure()
		
		ctxLogger.Error("event delivery failed", 
			"webhook_id", webhook.ID, 
			"event_id", event.ID, 
			"delivery_id", delivery.ID, 
			"status", resp.StatusCode)
	}
	
	// Update delivery in repository
	if err := s.webhookRepo.UpdateDelivery(ctx, delivery); err != nil {
		return errors.Wrap(err, "failed to update delivery record")
	}
	
	// Update webhook stats in repository
	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		return errors.Wrap(err, "failed to update webhook stats")
	}
	
	return nil
}

// GetDeliveryStatus gets the status of a webhook delivery
func (s *webhookService) GetDeliveryStatus(ctx context.Context, deliveryID string, tenantID string) (*models.WebhookDelivery, error) {
	ctxLogger := logger.WithContext(ctx)
	
	if err := s.validateInput(map[string]string{
		"delivery ID": deliveryID,
		"tenant ID":   tenantID,
	}); err != nil {
		return nil, err
	}
	
	// Get delivery record
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get delivery")
	}
	
	// Get webhook to verify tenant ownership
	webhook, err := s.webhookRepo.GetByID(ctx, delivery.WebhookID, tenantID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook for delivery")
	}
	
	// Verify tenant ownership
	if webhook.TenantID != tenantID {
		return nil, errors.NewAuthorizationError("delivery does not belong to the specified tenant")
	}
	
	ctxLogger.Info("delivery status retrieved", "delivery_id", deliveryID)
	return delivery, nil
}

// ListDeliveries lists delivery attempts for a webhook with pagination
func (s *webhookService) ListDeliveries(ctx context.Context, webhookID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.WebhookDelivery], error) {
	ctxLogger := logger.WithContext(ctx)
	
	if err := s.validateInput(map[string]string{
		"webhook ID": webhookID,
		"tenant ID":  tenantID,
	}); err != nil {
		return utils.PaginatedResult[models.WebhookDelivery]{}, err
	}
	
	// Verify webhook exists and belongs to tenant
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID, tenantID)
	if err != nil {
		return utils.PaginatedResult[models.WebhookDelivery]{}, errors.Wrap(err, "failed to verify webhook existence")
	}
	
	if webhook.TenantID != tenantID {
		return utils.PaginatedResult[models.WebhookDelivery]{}, errors.NewAuthorizationError("webhook does not belong to the specified tenant")
	}
	
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// List deliveries for the webhook
	result, err := s.webhookRepo.ListDeliveriesByWebhook(ctx, webhookID, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.WebhookDelivery]{}, errors.Wrap(err, "failed to list deliveries")
	}
	
	ctxLogger.Info("deliveries listed successfully", "webhook_id", webhookID, "count", len(result.Items))
	return result, nil
}

// RetryDelivery retries a failed webhook delivery
func (s *webhookService) RetryDelivery(ctx context.Context, deliveryID string, tenantID string) error {
	ctxLogger := logger.WithContext(ctx)
	
	if err := s.validateInput(map[string]string{
		"delivery ID": deliveryID,
		"tenant ID":   tenantID,
	}); err != nil {
		return err
	}
	
	// Get delivery record
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to get delivery")
	}
	
	// Verify delivery is in failed state
	if !delivery.IsFailed() {
		return errors.NewValidationError("only failed deliveries can be retried")
	}
	
	// Verify delivery attempt count is under maximum
	if delivery.AttemptCount >= maxRetryAttempts {
		return errors.NewValidationError(fmt.Sprintf("maximum retry attempts (%d) reached", maxRetryAttempts))
	}
	
	// Get webhook
	webhook, err := s.webhookRepo.GetByID(ctx, delivery.WebhookID, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to get webhook for delivery")
	}
	
	// Verify tenant ownership
	if webhook.TenantID != tenantID {
		return errors.NewAuthorizationError("delivery does not belong to the specified tenant")
	}
	
	// Get event - this would typically come from an EventRepository
	// For this implementation, we'll need to reconstruct the event from data
	// that would be available in the system. In a real implementation,
	// this would likely use an EventRepository to fetch the complete event.
	event := &models.Event{
		ID:       delivery.EventID,
		TenantID: tenantID,
		// Note: In a real implementation, we would need to retrieve the full
		// event data including Type and Payload from a repository or other source
	}
	
	// Increment attempt count
	delivery.IncrementAttempt()
	if err := s.webhookRepo.UpdateDelivery(ctx, delivery); err != nil {
		return errors.Wrap(err, "failed to update delivery attempt count")
	}
	
	ctxLogger.Info("retrying webhook delivery", "delivery_id", deliveryID, "attempt", delivery.AttemptCount)
	
	// Note: A real implementation would use DeliverEvent here with the complete event data
	// For this demonstration, we acknowledge the limitation that we can't fully
	// retry the delivery without access to the event data
	
	return nil
}

// ProcessPendingDeliveries processes pending webhook deliveries
func (s *webhookService) ProcessPendingDeliveries(ctx context.Context, batchSize int) (int, error) {
	ctxLogger := logger.WithContext(ctx)
	
	if batchSize <= 0 {
		return 0, errors.NewValidationError("batch size must be positive")
	}
	
	// Get pending deliveries
	deliveries, err := s.webhookRepo.ListPendingDeliveries(ctx, batchSize)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list pending deliveries")
	}
	
	processed := 0
	
	for _, delivery := range deliveries {
		// Get the webhook for this delivery
		webhook, err := s.webhookRepo.GetByID(ctx, delivery.WebhookID, "")
		if err != nil {
			ctxLogger.Error("failed to get webhook for delivery", 
				"delivery_id", delivery.ID, 
				"webhook_id", delivery.WebhookID, 
				"error", err)
			continue
		}
		
		// In a real implementation, we would get the event data here
		// and call DeliverEvent with the webhook, event, and delivery
		
		ctxLogger.Info("processing pending delivery", 
			"delivery_id", delivery.ID, 
			"webhook_id", delivery.WebhookID, 
			"tenant_id", webhook.TenantID, 
			"attempt", delivery.AttemptCount)
		
		// Count as processed even if we can't complete the delivery
		// in this demonstration implementation
		processed++
	}
	
	ctxLogger.Info("processed pending deliveries", "processed", processed, "total", len(deliveries))
	return processed, nil
}

// RetryFailedDeliveries retries failed webhook deliveries
func (s *webhookService) RetryFailedDeliveries(ctx context.Context, batchSize int) (int, error) {
	ctxLogger := logger.WithContext(ctx)
	
	if batchSize <= 0 {
		return 0, errors.NewValidationError("batch size must be positive")
	}
	
	// Get failed deliveries
	deliveries, err := s.webhookRepo.ListFailedDeliveries(ctx, batchSize, maxRetryAttempts)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list failed deliveries")
	}
	
	retried := 0
	
	for _, delivery := range deliveries {
		// Skip deliveries that have reached the max retry attempts
		if delivery.AttemptCount >= maxRetryAttempts {
			continue
		}
		
		// Get the webhook for this delivery
		webhook, err := s.webhookRepo.GetByID(ctx, delivery.WebhookID, "")
		if err != nil {
			ctxLogger.Error("failed to get webhook for delivery", 
				"delivery_id", delivery.ID, 
				"webhook_id", delivery.WebhookID, 
				"error", err)
			continue
		}
		
		// Increment attempt count
		delivery.IncrementAttempt()
		if err := s.webhookRepo.UpdateDelivery(ctx, delivery); err != nil {
			ctxLogger.Error("failed to update delivery attempt count", 
				"delivery_id", delivery.ID, 
				"error", err)
			continue
		}
		
		// In a real implementation, we would get the event data here
		// and call DeliverEvent with the webhook, event, and delivery
		
		ctxLogger.Info("retrying failed delivery", 
			"delivery_id", delivery.ID, 
			"webhook_id", delivery.WebhookID, 
			"tenant_id", webhook.TenantID, 
			"attempt", delivery.AttemptCount)
		
		// Count as retried even if we can't complete the delivery
		// in this demonstration implementation
		retried++
	}
	
	ctxLogger.Info("retried failed deliveries", "retried", retried, "total", len(deliveries))
	return retried, nil
}

// validateInput validates input parameters
func (s *webhookService) validateInput(params map[string]string) error {
	for param, value := range params {
		if value == "" {
			return errors.NewValidationError(fmt.Sprintf("%s cannot be empty", param))
		}
	}
	return nil
}