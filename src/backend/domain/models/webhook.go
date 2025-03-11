package models

import (
	"crypto/hmac"     // v1.0.0+ - For generating HMAC signatures for webhook payloads
	"crypto/sha256"   // v1.0.0+ - For SHA-256 hashing in signature generation
	"encoding/hex"    // v1.0.0+ - For encoding binary signatures to hexadecimal strings
	"errors"          // v1.0.0+ - For error handling in validation methods
	"strings"         // v1.0.0+ - For string manipulation operations
	"time"            // v1.0.0+ - For timestamp fields like CreatedAt and UpdatedAt
)

// Webhook status constants
const (
	WebhookStatusActive   = "active"
	WebhookStatusInactive = "inactive"
)

// WebhookDelivery status constants
const (
	WebhookDeliveryStatusPending = "pending"
	WebhookDeliveryStatusSuccess = "success"
	WebhookDeliveryStatusFailed  = "failed"
)

// Error variables for webhook validation
var (
	ErrWebhookURLEmpty         = errors.New("webhook URL cannot be empty")
	ErrWebhookNoEventTypes     = errors.New("webhook must subscribe to at least one event type")
	ErrWebhookInvalidEventType = errors.New("webhook contains invalid event type")
)

// Webhook represents a webhook subscription for receiving event notifications
type Webhook struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	URL            string     `json:"url"`
	EventTypes     []string   `json:"event_types"`
	SecretKey      string     `json:"secret_key"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FailureCount   int        `json:"failure_count"`
	LastFailureTime time.Time `json:"last_failure_time"`
}

// Validate validates that the webhook has all required fields
func (w *Webhook) Validate() error {
	if strings.TrimSpace(w.URL) == "" {
		return ErrWebhookURLEmpty
	}
	
	if strings.TrimSpace(w.TenantID) == "" {
		return errors.New("tenant ID cannot be empty")
	}
	
	if len(w.EventTypes) == 0 {
		return ErrWebhookNoEventTypes
	}
	
	// In a real implementation, we would validate event types against a list of known types
	// For now, we'll assume all event types are valid but this would be implementation-specific
	
	return nil
}

// IsActive checks if the webhook is active
func (w *Webhook) IsActive() bool {
	return w.Status == WebhookStatusActive
}

// Activate activates the webhook
func (w *Webhook) Activate() {
	w.Status = WebhookStatusActive
	w.UpdatedAt = time.Now()
}

// Deactivate deactivates the webhook
func (w *Webhook) Deactivate() {
	w.Status = WebhookStatusInactive
	w.UpdatedAt = time.Now()
}

// ShouldProcessEvent checks if this webhook should process a given event type
func (w *Webhook) ShouldProcessEvent(eventType string) bool {
	if !w.IsActive() {
		return false
	}
	
	for _, et := range w.EventTypes {
		if et == eventType {
			return true
		}
	}
	
	return false
}

// GenerateSignatureForPayload generates an HMAC-SHA256 signature for a payload
func (w *Webhook) GenerateSignatureForPayload(payload []byte) string {
	h := hmac.New(sha256.New, []byte(w.SecretKey))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// RecordDeliverySuccess records a successful delivery attempt
func (w *Webhook) RecordDeliverySuccess() {
	w.FailureCount = 0
	w.UpdatedAt = time.Now()
}

// RecordDeliveryFailure records a failed delivery attempt
func (w *Webhook) RecordDeliveryFailure() {
	w.FailureCount++
	w.LastFailureTime = time.Now()
	w.UpdatedAt = time.Now()
	
	// If the webhook has failed too many times, deactivate it
	// In a real implementation, this threshold would likely be configurable
	if w.FailureCount >= 10 {
		w.Deactivate()
	}
}

// WebhookDelivery represents a webhook delivery attempt for an event
type WebhookDelivery struct {
	ID             string    `json:"id"`
	WebhookID      string    `json:"webhook_id"`
	EventID        string    `json:"event_id"`
	Status         string    `json:"status"`
	AttemptCount   int       `json:"attempt_count"`
	ResponseStatus int       `json:"response_status"`
	ResponseBody   string    `json:"response_body"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

// MarkAsSuccess marks the delivery as successful
func (d *WebhookDelivery) MarkAsSuccess(statusCode int, responseBody string) {
	d.Status = WebhookDeliveryStatusSuccess
	d.ResponseStatus = statusCode
	d.ResponseBody = responseBody
	d.CompletedAt = time.Now()
	d.UpdatedAt = time.Now()
}

// MarkAsFailed marks the delivery as failed
func (d *WebhookDelivery) MarkAsFailed(statusCode int, responseBody, errorMessage string) {
	d.Status = WebhookDeliveryStatusFailed
	d.ResponseStatus = statusCode
	d.ResponseBody = responseBody
	d.ErrorMessage = errorMessage
	d.CompletedAt = time.Now()
	d.UpdatedAt = time.Now()
}

// IncrementAttempt increments the attempt count for retries
func (d *WebhookDelivery) IncrementAttempt() {
	d.AttemptCount++
	d.UpdatedAt = time.Now()
}

// IsCompleted checks if the delivery is completed (success or failure)
func (d *WebhookDelivery) IsCompleted() bool {
	return d.Status == WebhookDeliveryStatusSuccess || d.Status == WebhookDeliveryStatusFailed
}

// IsPending checks if the delivery is still pending
func (d *WebhookDelivery) IsPending() bool {
	return d.Status == WebhookDeliveryStatusPending
}

// IsSuccess checks if the delivery was successful
func (d *WebhookDelivery) IsSuccess() bool {
	return d.Status == WebhookDeliveryStatusSuccess
}

// IsFailed checks if the delivery failed
func (d *WebhookDelivery) IsFailed() bool {
	return d.Status == WebhookDeliveryStatusFailed
}

// NewWebhook creates a new Webhook instance with the given parameters
func NewWebhook(url, tenantID string, eventTypes []string) (*Webhook, error) {
	if strings.TrimSpace(url) == "" {
		return nil, ErrWebhookURLEmpty
	}
	
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenant ID cannot be empty")
	}
	
	if len(eventTypes) == 0 {
		return nil, ErrWebhookNoEventTypes
	}
	
	// In a real implementation, we would validate event types against a list of known types
	// and return ErrWebhookInvalidEventType if any are invalid
	
	// In a real implementation, this would generate a secure random key
	// For example, using crypto/rand to generate a random string
	secretKey := "secure-random-key" // Placeholder for demonstration
	
	now := time.Now()
	
	return &Webhook{
		URL:        url,
		TenantID:   tenantID,
		EventTypes: eventTypes,
		SecretKey:  secretKey,
		Status:     WebhookStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// NewWebhookDelivery creates a new WebhookDelivery instance for tracking a delivery attempt
func NewWebhookDelivery(webhookID, eventID string) *WebhookDelivery {
	now := time.Now()
	
	return &WebhookDelivery{
		WebhookID:    webhookID,
		EventID:      eventID,
		Status:       WebhookDeliveryStatusPending,
		AttemptCount: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}