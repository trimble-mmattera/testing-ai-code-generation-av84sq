// Package handlers implements HTTP handlers for webhook management in the Document Management Platform.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin" // v1.9.0+

	"../dto"
	"../validators"
	"../middleware"
	"../../domain/models"
	"../../application/usecases"
	"../../pkg/errors"
	"../../pkg/logger"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// WebhookHandler handles HTTP requests for webhook management
type WebhookHandler struct {
	webhookUseCase usecases.WebhookUseCase
}

// NewWebhookHandler creates a new WebhookHandler instance
func NewWebhookHandler(webhookUseCase usecases.WebhookUseCase) (*WebhookHandler, error) {
	if webhookUseCase == nil {
		return nil, errors.NewValidationError("webhook use case cannot be nil")
	}

	return &WebhookHandler{
		webhookUseCase: webhookUseCase,
	}, nil
}

// RegisterRoutes registers webhook routes with the provided router group
func (h *WebhookHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/webhooks", h.CreateWebhook)
	router.GET("/webhooks", h.ListWebhooks)
	router.GET("/webhooks/:id", h.GetWebhook)
	router.PUT("/webhooks/:id", h.UpdateWebhook)
	router.DELETE("/webhooks/:id", h.DeleteWebhook)
	router.GET("/webhooks/event-types", h.GetEventTypes)
	router.GET("/webhooks/:id/deliveries", h.ListWebhookDeliveries)
	router.GET("/webhooks/deliveries/:id", h.GetDeliveryStatus)
	router.POST("/webhooks/deliveries/:id/retry", h.RetryDelivery)
}

// CreateWebhook handles webhook creation requests
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Bind request body to DTO
	var req dto.CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Error("failed to bind request body")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("invalid request format"),
			map[string]string{"request": err.Error()},
		))
		return
	}

	// Validate request
	if err := validators.ValidateCreateWebhookRequest(&req); err != nil {
		log.WithError(err).Error("webhook validation failed")
		h.handleError(c, err)
		return
	}

	// Convert DTO to domain model
	webhook := dto.ToWebhookDomain(&req, tenantID)

	// Call use case to create webhook
	webhookID, err := h.webhookUseCase.CreateWebhook(c.Request.Context(), webhook)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, dto.NewDataResponse(map[string]string{
		"id":      webhookID,
		"message": "Webhook created successfully",
	}))
}

// GetWebhook handles webhook retrieval requests
func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get webhook ID from URL
	webhookID := c.Param("id")
	if webhookID == "" {
		log.Error("webhook ID missing in request path")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("webhook ID is required"),
			map[string]string{"id": "required"},
		))
		return
	}

	// Call use case to get webhook
	webhook, err := h.webhookUseCase.GetWebhook(c.Request.Context(), webhookID, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert domain model to DTO and return
	c.JSON(http.StatusOK, dto.NewDataResponse(dto.ToWebhookDTO(webhook)))
}

// ListWebhooks handles webhook listing requests with pagination
func (h *WebhookHandler) ListWebhooks(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get pagination parameters
	page, pageSize := h.getPaginationParams(c)

	// Call use case to list webhooks
	result, err := h.webhookUseCase.ListWebhooks(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert domain models to DTOs and return paginated response
	webhooks := dto.ToWebhookListDTO(result)
	c.JSON(http.StatusOK, dto.NewPaginatedResponse(webhooks, result.Pagination))
}

// UpdateWebhook handles webhook update requests
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get webhook ID from URL
	webhookID := c.Param("id")
	if webhookID == "" {
		log.Error("webhook ID missing in request path")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("webhook ID is required"),
			map[string]string{"id": "required"},
		))
		return
	}

	// Bind request body to DTO
	var req dto.UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Error("failed to bind request body")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("invalid request format"),
			map[string]string{"request": err.Error()},
		))
		return
	}

	// Validate request
	if err := validators.ValidateUpdateWebhookRequest(&req); err != nil {
		log.WithError(err).Error("webhook validation failed")
		h.handleError(c, err)
		return
	}

	// Get existing webhook
	webhook, err := h.webhookUseCase.GetWebhook(c.Request.Context(), webhookID, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Update webhook with request data
	webhook = dto.UpdateWebhookFromRequest(webhook, &req)

	// Call use case to update webhook
	err = h.webhookUseCase.UpdateWebhook(c.Request.Context(), webhook)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return success response
	c.JSON(http.StatusOK, dto.NewMessageResponse("Webhook updated successfully"))
}

// DeleteWebhook handles webhook deletion requests
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get webhook ID from URL
	webhookID := c.Param("id")
	if webhookID == "" {
		log.Error("webhook ID missing in request path")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("webhook ID is required"),
			map[string]string{"id": "required"},
		))
		return
	}

	// Call use case to delete webhook
	err := h.webhookUseCase.DeleteWebhook(c.Request.Context(), webhookID, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return success response
	c.JSON(http.StatusOK, dto.NewMessageResponse("Webhook deleted successfully"))
}

// GetEventTypes handles requests for supported webhook event types
func (h *WebhookHandler) GetEventTypes(c *gin.Context) {
	// Return list of supported event types
	response := dto.WebhookEventTypesResponse{
		EventTypes: dto.SupportedEventTypes,
	}
	c.JSON(http.StatusOK, dto.NewDataResponse(response))
}

// ListWebhookDeliveries handles webhook delivery listing requests with pagination
func (h *WebhookHandler) ListWebhookDeliveries(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get webhook ID from URL
	webhookID := c.Param("id")
	if webhookID == "" {
		log.Error("webhook ID missing in request path")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("webhook ID is required"),
			map[string]string{"id": "required"},
		))
		return
	}

	// Get pagination parameters
	page, pageSize := h.getPaginationParams(c)

	// Call use case to list webhook deliveries
	result, err := h.webhookUseCase.ListDeliveries(c.Request.Context(), webhookID, tenantID, page, pageSize)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert domain models to DTOs and return
	deliveries := dto.ToWebhookDeliveryListDTO(result)
	c.JSON(http.StatusOK, dto.NewPaginatedResponse(deliveries, result.Pagination))
}

// GetDeliveryStatus handles webhook delivery status retrieval requests
func (h *WebhookHandler) GetDeliveryStatus(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get delivery ID from URL
	deliveryID := c.Param("id")
	if deliveryID == "" {
		log.Error("delivery ID missing in request path")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("delivery ID is required"),
			map[string]string{"id": "required"},
		))
		return
	}

	// Call use case to get delivery status
	delivery, err := h.webhookUseCase.GetDeliveryStatus(c.Request.Context(), deliveryID, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert domain model to DTO and return
	c.JSON(http.StatusOK, dto.NewDataResponse(dto.ToWebhookDeliveryDTO(delivery)))
}

// RetryDelivery handles webhook delivery retry requests
func (h *WebhookHandler) RetryDelivery(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())

	// Extract tenant ID from request context
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		log.Error("tenant ID missing in request context")
		c.JSON(http.StatusUnauthorized, dto.NewErrorResponse(
			errors.NewAuthenticationError("tenant context required"),
		))
		return
	}

	// Get delivery ID from URL
	deliveryID := c.Param("id")
	if deliveryID == "" {
		log.Error("delivery ID missing in request path")
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			errors.NewValidationError("delivery ID is required"),
			map[string]string{"id": "required"},
		))
		return
	}

	// Call use case to retry delivery
	err := h.webhookUseCase.RetryDelivery(c.Request.Context(), deliveryID, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return success response
	c.JSON(http.StatusAccepted, dto.NewMessageResponse("Webhook delivery retry initiated"))
}

// getPaginationParams extracts and validates pagination parameters from the request
func (h *WebhookHandler) getPaginationParams(c *gin.Context) (int, int) {
	// Extract page parameter
	pageStr := c.Query("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Extract pageSize parameter
	pageSizeStr := c.Query("pageSize")
	pageSize := defaultPageSize
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// Cap pageSize at maximum
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

// handleError handles errors and returns appropriate HTTP responses
func (h *WebhookHandler) handleError(c *gin.Context, err error) {
	if errors.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, dto.NewValidationErrorResponse(
			err,
			map[string]string{}, // In a real implementation, we would extract validation details
		))
		return
	}

	if errors.IsResourceNotFoundError(err) {
		c.JSON(http.StatusNotFound, dto.NewResourceNotFoundErrorResponse(err))
		return
	}

	// Default to internal server error
	logger.WithError(err).Error("internal server error")
	c.JSON(http.StatusInternalServerError, dto.NewInternalErrorResponse(err))
}