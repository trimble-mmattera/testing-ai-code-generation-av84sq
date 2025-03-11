// Package dto provides Data Transfer Objects for webhook-related operations in the Document Management Platform API.
// This file defines the request and response structures for webhook management endpoints, including creation,
// retrieval, update, and listing of webhooks and their delivery statuses.
package dto

import (
	"time" // standard library

	"../../domain/models"
	"../../pkg/utils/pagination"
	timeutils "../../pkg/utils/time_utils"
)

// SupportedEventTypes lists all the supported event types for webhook subscriptions
var SupportedEventTypes = []string{
	"document.uploaded",
	"document.processed",
	"document.downloaded",
	"document.quarantined",
	"folder.created",
	"folder.updated",
}

// CreateWebhookRequest is a DTO for creating a new webhook
type CreateWebhookRequest struct {
	URL         string   `json:"url"`
	EventTypes  []string `json:"event_types"`
	Description string   `json:"description"`
	SecretKey   string   `json:"secret_key"`
}

// UpdateWebhookRequest is a DTO for updating an existing webhook
type UpdateWebhookRequest struct {
	URL         string   `json:"url"`
	EventTypes  []string `json:"event_types"`
	Description *string  `json:"description"`
	Status      string   `json:"status"`
	SecretKey   string   `json:"secret_key"`
}

// WebhookDTO is a DTO for webhook data
type WebhookDTO struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	EventTypes  []string `json:"event_types"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// WebhookDeliveryDTO is a DTO for webhook delivery data
type WebhookDeliveryDTO struct {
	ID             string `json:"id"`
	WebhookID      string `json:"webhook_id"`
	EventID        string `json:"event_id"`
	Status         string `json:"status"`
	AttemptCount   int    `json:"attempt_count"`
	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `json:"response_body"`
	ErrorMessage   string `json:"error_message"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	CompletedAt    string `json:"completed_at"`
}

// WebhookEventTypesResponse is a DTO for listing supported webhook event types
type WebhookEventTypesResponse struct {
	EventTypes []string `json:"event_types"`
}

// ResourceCreatedResponse is a DTO for resource creation operations
type ResourceCreatedResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	ID        string `json:"id"`
	Message   string `json:"message"`
}

// ResourceUpdatedResponse is a DTO for resource update operations
type ResourceUpdatedResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// ResourceDeletedResponse is a DTO for resource deletion operations
type ResourceDeletedResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// ToWebhookDTO converts a domain Webhook model to a WebhookDTO
func ToWebhookDTO(webhook *models.Webhook) WebhookDTO {
	return WebhookDTO{
		ID:          webhook.ID,
		URL:         webhook.URL,
		EventTypes:  webhook.EventTypes,
		Description: webhook.Description,
		Status:      webhook.Status,
		CreatedAt:   timeutils.FormatTime(webhook.CreatedAt, ""),
		UpdatedAt:   timeutils.FormatTime(webhook.UpdatedAt, ""),
	}
}

// ToWebhookListDTO converts a paginated list of domain Webhook models to WebhookDTOs
func ToWebhookListDTO(result pagination.PaginatedResult[models.Webhook]) []WebhookDTO {
	dtos := make([]WebhookDTO, len(result.Items))
	for i, webhook := range result.Items {
		dtos[i] = ToWebhookDTO(&webhook)
	}
	return dtos
}

// ToWebhookDomain converts a CreateWebhookRequest to a domain Webhook model
func ToWebhookDomain(request *CreateWebhookRequest, tenantID string) *models.Webhook {
	webhook := &models.Webhook{
		TenantID:    tenantID,
		URL:         request.URL,
		EventTypes:  request.EventTypes,
		Description: request.Description,
		Status:      models.WebhookStatusActive,
	}
	return webhook
}

// UpdateWebhookFromRequest updates a domain Webhook model with values from an UpdateWebhookRequest
func UpdateWebhookFromRequest(webhook *models.Webhook, request *UpdateWebhookRequest) *models.Webhook {
	if request.URL != "" {
		webhook.URL = request.URL
	}
	if request.EventTypes != nil && len(request.EventTypes) > 0 {
		webhook.EventTypes = request.EventTypes
	}
	if request.Description != nil {
		webhook.Description = *request.Description
	}
	if request.Status != "" {
		webhook.Status = request.Status
	}
	return webhook
}

// ToWebhookDeliveryDTO converts a domain WebhookDelivery model to a WebhookDeliveryDTO
func ToWebhookDeliveryDTO(delivery *models.WebhookDelivery) WebhookDeliveryDTO {
	dto := WebhookDeliveryDTO{
		ID:             delivery.ID,
		WebhookID:      delivery.WebhookID,
		EventID:        delivery.EventID,
		Status:         delivery.Status,
		AttemptCount:   delivery.AttemptCount,
		ResponseStatus: delivery.ResponseStatus,
		ResponseBody:   delivery.ResponseBody,
		ErrorMessage:   delivery.ErrorMessage,
		CreatedAt:      timeutils.FormatTime(delivery.CreatedAt, ""),
		UpdatedAt:      timeutils.FormatTime(delivery.UpdatedAt, ""),
	}

	if !delivery.CompletedAt.IsZero() {
		dto.CompletedAt = timeutils.FormatTime(delivery.CompletedAt, "")
	}

	return dto
}

// ToWebhookDeliveryListDTO converts a paginated list of domain WebhookDelivery models to WebhookDeliveryDTOs
func ToWebhookDeliveryListDTO(result pagination.PaginatedResult[models.WebhookDelivery]) []WebhookDeliveryDTO {
	dtos := make([]WebhookDeliveryDTO, len(result.Items))
	for i, delivery := range result.Items {
		dtos[i] = ToWebhookDeliveryDTO(&delivery)
	}
	return dtos
}

// NewSuccessResponse creates a new success response with a message
func NewSuccessResponse(message string) MessageResponse {
	return MessageResponse{
		Success:   true,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Message:   message,
	}
}

// NewEmptySuccessResponse creates a new empty success response
func NewEmptySuccessResponse() Response {
	return Response{
		Success:   true,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
	}
}

// Response is a base DTO for API responses
type Response struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
}

// MessageResponse is a DTO for API responses with a message
type MessageResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}