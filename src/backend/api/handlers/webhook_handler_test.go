package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"../../api/dto"
	"../../application/usecases"
	"../../domain/models"
	apperrors "../../pkg/errors"
	"../../pkg/utils/pagination"
)

// MockWebhookUseCase is a mock implementation of the WebhookUseCase interface
type MockWebhookUseCase struct {
	mock.Mock
}

// Implement all the methods from the WebhookUseCase interface
func (m *MockWebhookUseCase) CreateWebhook(ctx context.Context, webhook *models.Webhook) (string, error) {
	args := m.Called(ctx, webhook)
	return args.String(0), args.Error(1)
}

func (m *MockWebhookUseCase) GetWebhook(ctx context.Context, id string, tenantID string) (*models.Webhook, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *MockWebhookUseCase) UpdateWebhook(ctx context.Context, webhook *models.Webhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *MockWebhookUseCase) DeleteWebhook(ctx context.Context, id string, tenantID string) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockWebhookUseCase) ListWebhooks(ctx context.Context, tenantID string, page int, pageSize int) (pagination.PaginatedResult[models.Webhook], error) {
	args := m.Called(ctx, tenantID, page, pageSize)
	return args.Get(0).(pagination.PaginatedResult[models.Webhook]), args.Error(1)
}

func (m *MockWebhookUseCase) ProcessEvent(ctx context.Context, event *models.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockWebhookUseCase) GetDeliveryStatus(ctx context.Context, deliveryID string, tenantID string) (*models.WebhookDelivery, error) {
	args := m.Called(ctx, deliveryID, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookUseCase) ListDeliveries(ctx context.Context, webhookID string, tenantID string, page int, pageSize int) (pagination.PaginatedResult[models.WebhookDelivery], error) {
	args := m.Called(ctx, webhookID, tenantID, page, pageSize)
	return args.Get(0).(pagination.PaginatedResult[models.WebhookDelivery]), args.Error(1)
}

func (m *MockWebhookUseCase) RetryDelivery(ctx context.Context, deliveryID string, tenantID string) error {
	args := m.Called(ctx, deliveryID, tenantID)
	return args.Error(0)
}

// WebhookHandlerSuite defines the test suite
type WebhookHandlerSuite struct {
	suite.Suite
	router         *gin.Engine
	recorder       *httptest.ResponseRecorder
	webhookUseCase *MockWebhookUseCase
	webhookHandler *WebhookHandler
}

// SetupTest is called before each test
func (s *WebhookHandlerSuite) SetupTest() {
	// Create a gin router in test mode
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.recorder = httptest.NewRecorder()
	
	// Create a mock webhook use case
	s.webhookUseCase = new(MockWebhookUseCase)
	
	// Create the webhook handler with the mock use case
	s.webhookHandler = NewWebhookHandler(s.webhookUseCase)
	
	// Set up a router group with the webhook handler routes
	group := s.router.Group("/api/v1")
	s.webhookHandler.RegisterRoutes(group)
}

// Helper function to create a test webhook model
func (s *WebhookHandlerSuite) createTestWebhook() *models.Webhook {
	return &models.Webhook{
		ID:          "webhook-123",
		TenantID:    "tenant-123",
		URL:         "https://example.com/webhook",
		EventTypes:  []string{"document.uploaded", "document.processed"},
		Description: "Test webhook",
		Status:      models.WebhookStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// Helper function to create a test webhook delivery model
func (s *WebhookHandlerSuite) createTestWebhookDelivery() *models.WebhookDelivery {
	return &models.WebhookDelivery{
		ID:             "delivery-123",
		WebhookID:      "webhook-123",
		EventID:        "event-123",
		Status:         models.WebhookDeliveryStatusSuccess,
		AttemptCount:   1,
		ResponseStatus: 200,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// Helper function to set up authentication context for tests
func (s *WebhookHandlerSuite) setupAuthContext(c *gin.Context) {
	c.Set("tenant_id", "tenant-123")
	c.Set("user_id", "user-123")
	c.Set("roles", []string{"admin"})
}

// Helper function to get a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// TestCreateWebhook_Success tests successful webhook creation
func (s *WebhookHandlerSuite) TestCreateWebhook_Success() {
	// Create a test webhook request
	reqBody := dto.CreateWebhookRequest{
		URL:         "https://example.com/webhook",
		EventTypes:  []string{"document.uploaded", "document.processed"},
		Description: "Test webhook",
		SecretKey:   "secret123",
	}
	
	// Marshal the request to JSON
	jsonBody, err := json.Marshal(reqBody)
	s.NoError(err)
	
	// Expect the use case to return a webhook ID
	s.webhookUseCase.On("CreateWebhook", mock.Anything, mock.AnythingOfType("*models.Webhook")).Return("webhook-123", nil)
	
	// Create a request
	req, _ := http.NewRequest("POST", "/api/v1/webhooks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusCreated, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Equal("webhook-123", response["id"])
	s.Contains(response, "timestamp")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestCreateWebhook_ValidationError tests webhook creation with validation errors
func (s *WebhookHandlerSuite) TestCreateWebhook_ValidationError() {
	// Create an invalid request (missing required fields)
	reqBody := dto.CreateWebhookRequest{
		// Missing URL and EventTypes
		Description: "Test webhook",
	}
	
	// Marshal the request to JSON
	jsonBody, err := json.Marshal(reqBody)
	s.NoError(err)
	
	// Create a request
	req, _ := http.NewRequest("POST", "/api/v1/webhooks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusBadRequest, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
}

// TestCreateWebhook_UseCaseError tests webhook creation when the use case returns an error
func (s *WebhookHandlerSuite) TestCreateWebhook_UseCaseError() {
	// Create a test webhook request
	reqBody := dto.CreateWebhookRequest{
		URL:         "https://example.com/webhook",
		EventTypes:  []string{"document.uploaded", "document.processed"},
		Description: "Test webhook",
		SecretKey:   "secret123",
	}
	
	// Marshal the request to JSON
	jsonBody, err := json.Marshal(reqBody)
	s.NoError(err)
	
	// Expect the use case to return an error
	s.webhookUseCase.On("CreateWebhook", mock.Anything, mock.AnythingOfType("*models.Webhook")).Return("", errors.New("internal error"))
	
	// Create a request
	req, _ := http.NewRequest("POST", "/api/v1/webhooks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusInternalServerError, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestGetWebhook_Success tests successful webhook retrieval
func (s *WebhookHandlerSuite) TestGetWebhook_Success() {
	// Create a test webhook
	webhook := s.createTestWebhook()
	
	// Expect the use case to return the webhook
	s.webhookUseCase.On("GetWebhook", mock.Anything, "webhook-123", "tenant-123").Return(webhook, nil)
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/webhook-123", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-123"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Contains(response, "webhook")
	webhookData := response["webhook"].(map[string]interface{})
	s.Equal("webhook-123", webhookData["id"])
	s.Equal("https://example.com/webhook", webhookData["url"])
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestGetWebhook_NotFound tests webhook retrieval when the webhook is not found
func (s *WebhookHandlerSuite) TestGetWebhook_NotFound() {
	// Expect the use case to return a not found error
	s.webhookUseCase.On("GetWebhook", mock.Anything, "webhook-999", "tenant-123").Return(nil, apperrors.NewResourceNotFoundError("webhook not found"))
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/webhook-999", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-999"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusNotFound, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestListWebhooks_Success tests successful webhook listing
func (s *WebhookHandlerSuite) TestListWebhooks_Success() {
	// Create test webhooks
	webhook1 := s.createTestWebhook()
	webhook2 := s.createTestWebhook()
	webhook2.ID = "webhook-456"
	webhook2.URL = "https://example.com/webhook2"
	
	// Create a paginated result
	webhooks := []models.Webhook{*webhook1, *webhook2}
	pageInfo := pagination.PageInfo{
		Page:        1,
		PageSize:    10,
		TotalPages:  1,
		TotalItems:  2,
		HasNext:     false,
		HasPrevious: false,
	}
	paginatedResult := pagination.PaginatedResult[models.Webhook]{
		Items:      webhooks,
		Pagination: pageInfo,
	}
	
	// Expect the use case to return the paginated result
	s.webhookUseCase.On("ListWebhooks", mock.Anything, "tenant-123", 1, 10).Return(paginatedResult, nil)
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks?page=1&page_size=10", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Contains(response, "webhooks")
	s.Contains(response, "pagination")
	
	webhooksData := response["webhooks"].([]interface{})
	s.Equal(2, len(webhooksData))
	
	pagination := response["pagination"].(map[string]interface{})
	s.Equal(float64(1), pagination["page"])
	s.Equal(float64(10), pagination["pageSize"])
	s.Equal(float64(2), pagination["totalItems"])
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestListWebhooks_Empty tests webhook listing when no webhooks exist
func (s *WebhookHandlerSuite) TestListWebhooks_Empty() {
	// Create an empty paginated result
	webhooks := []models.Webhook{}
	pageInfo := pagination.PageInfo{
		Page:        1,
		PageSize:    10,
		TotalPages:  0,
		TotalItems:  0,
		HasNext:     false,
		HasPrevious: false,
	}
	paginatedResult := pagination.PaginatedResult[models.Webhook]{
		Items:      webhooks,
		Pagination: pageInfo,
	}
	
	// Expect the use case to return the empty paginated result
	s.webhookUseCase.On("ListWebhooks", mock.Anything, "tenant-123", 1, 10).Return(paginatedResult, nil)
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks?page=1&page_size=10", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Contains(response, "webhooks")
	s.Contains(response, "pagination")
	
	webhooksData := response["webhooks"].([]interface{})
	s.Equal(0, len(webhooksData))
	
	pagination := response["pagination"].(map[string]interface{})
	s.Equal(float64(0), pagination["totalItems"])
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestUpdateWebhook_Success tests successful webhook update
func (s *WebhookHandlerSuite) TestUpdateWebhook_Success() {
	// Create a test webhook
	webhook := s.createTestWebhook()
	
	// Create an update request
	reqBody := dto.UpdateWebhookRequest{
		URL:         "https://example.com/webhook-updated",
		EventTypes:  []string{"document.uploaded", "document.processed", "document.downloaded"},
		Description: stringPtr("Updated webhook"),
		Status:      "active",
		SecretKey:   "newsecret123",
	}
	
	// Marshal the request to JSON
	jsonBody, err := json.Marshal(reqBody)
	s.NoError(err)
	
	// Expect the use case to return the webhook and a successful update
	s.webhookUseCase.On("GetWebhook", mock.Anything, "webhook-123", "tenant-123").Return(webhook, nil)
	s.webhookUseCase.On("UpdateWebhook", mock.Anything, mock.AnythingOfType("*models.Webhook")).Return(nil)
	
	// Create a request
	req, _ := http.NewRequest("PUT", "/api/v1/webhooks/webhook-123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-123"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains success message
	s.Equal(true, response["success"])
	s.Contains(response, "message")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestUpdateWebhook_ValidationError tests webhook update with validation errors
func (s *WebhookHandlerSuite) TestUpdateWebhook_ValidationError() {
	// Create an invalid update request (empty URL)
	reqBody := dto.UpdateWebhookRequest{
		URL:        "",  // Invalid: cannot be empty
		EventTypes: []string{},  // Invalid: cannot be empty
		Status:     "invalid",  // Invalid: not a valid status
	}
	
	// Marshal the request to JSON
	jsonBody, err := json.Marshal(reqBody)
	s.NoError(err)
	
	// Create a request
	req, _ := http.NewRequest("PUT", "/api/v1/webhooks/webhook-123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-123"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusBadRequest, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
}

// TestUpdateWebhook_NotFound tests webhook update when the webhook is not found
func (s *WebhookHandlerSuite) TestUpdateWebhook_NotFound() {
	// Create a valid update request
	reqBody := dto.UpdateWebhookRequest{
		URL:         "https://example.com/webhook-updated",
		EventTypes:  []string{"document.uploaded", "document.processed"},
		Description: stringPtr("Updated webhook"),
		Status:      "active",
	}
	
	// Marshal the request to JSON
	jsonBody, err := json.Marshal(reqBody)
	s.NoError(err)
	
	// Expect the use case to return a not found error
	s.webhookUseCase.On("GetWebhook", mock.Anything, "webhook-999", "tenant-123").Return(nil, apperrors.NewResourceNotFoundError("webhook not found"))
	
	// Create a request
	req, _ := http.NewRequest("PUT", "/api/v1/webhooks/webhook-999", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-999"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusNotFound, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err = json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestDeleteWebhook_Success tests successful webhook deletion
func (s *WebhookHandlerSuite) TestDeleteWebhook_Success() {
	// Expect the use case to return a successful deletion
	s.webhookUseCase.On("DeleteWebhook", mock.Anything, "webhook-123", "tenant-123").Return(nil)
	
	// Create a request
	req, _ := http.NewRequest("DELETE", "/api/v1/webhooks/webhook-123", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-123"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains success message
	s.Equal(true, response["success"])
	s.Contains(response, "message")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestDeleteWebhook_NotFound tests webhook deletion when the webhook is not found
func (s *WebhookHandlerSuite) TestDeleteWebhook_NotFound() {
	// Expect the use case to return a not found error
	s.webhookUseCase.On("DeleteWebhook", mock.Anything, "webhook-999", "tenant-123").Return(apperrors.NewResourceNotFoundError("webhook not found"))
	
	// Create a request
	req, _ := http.NewRequest("DELETE", "/api/v1/webhooks/webhook-999", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-999"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusNotFound, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestGetEventTypes_Success tests successful retrieval of supported event types
func (s *WebhookHandlerSuite) TestGetEventTypes_Success() {
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/event-types", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Contains(response, "event_types")
	
	// Check that the event types match the ones from dto.SupportedEventTypes
	eventTypes := response["event_types"].([]interface{})
	s.Equal(len(dto.SupportedEventTypes), len(eventTypes))
	for i, eventType := range dto.SupportedEventTypes {
		s.Equal(eventType, eventTypes[i])
	}
}

// TestListWebhookDeliveries_Success tests successful listing of webhook deliveries
func (s *WebhookHandlerSuite) TestListWebhookDeliveries_Success() {
	// Create test webhook deliveries
	delivery1 := s.createTestWebhookDelivery()
	delivery2 := s.createTestWebhookDelivery()
	delivery2.ID = "delivery-456"
	
	// Create a paginated result
	deliveries := []models.WebhookDelivery{*delivery1, *delivery2}
	pageInfo := pagination.PageInfo{
		Page:        1,
		PageSize:    10,
		TotalPages:  1,
		TotalItems:  2,
		HasNext:     false,
		HasPrevious: false,
	}
	paginatedResult := pagination.PaginatedResult[models.WebhookDelivery]{
		Items:      deliveries,
		Pagination: pageInfo,
	}
	
	// Expect the use case to return the paginated result
	s.webhookUseCase.On("ListDeliveries", mock.Anything, "webhook-123", "tenant-123", 1, 10).Return(paginatedResult, nil)
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/webhook-123/deliveries?page=1&page_size=10", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-123"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Contains(response, "deliveries")
	s.Contains(response, "pagination")
	
	deliveriesData := response["deliveries"].([]interface{})
	s.Equal(2, len(deliveriesData))
	
	pagination := response["pagination"].(map[string]interface{})
	s.Equal(float64(1), pagination["page"])
	s.Equal(float64(10), pagination["pageSize"])
	s.Equal(float64(2), pagination["totalItems"])
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestListWebhookDeliveries_NotFound tests delivery listing when the webhook is not found
func (s *WebhookHandlerSuite) TestListWebhookDeliveries_NotFound() {
	// Expect the use case to return a not found error
	s.webhookUseCase.On("ListDeliveries", mock.Anything, "webhook-999", "tenant-123", 1, 10).Return(
		pagination.PaginatedResult[models.WebhookDelivery]{},
		apperrors.NewResourceNotFoundError("webhook not found"))
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/webhook-999/deliveries?page=1&page_size=10", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "webhook-999"}}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusNotFound, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestGetDeliveryStatus_Success tests successful retrieval of delivery status
func (s *WebhookHandlerSuite) TestGetDeliveryStatus_Success() {
	// Create a test webhook delivery
	delivery := s.createTestWebhookDelivery()
	
	// Expect the use case to return the delivery
	s.webhookUseCase.On("GetDeliveryStatus", mock.Anything, "delivery-123", "tenant-123").Return(delivery, nil)
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/webhook-123/deliveries/delivery-123", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{
		gin.Param{Key: "id", Value: "webhook-123"},
		gin.Param{Key: "deliveryId", Value: "delivery-123"},
	}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusOK, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains the expected fields
	s.Equal(true, response["success"])
	s.Contains(response, "delivery")
	deliveryData := response["delivery"].(map[string]interface{})
	s.Equal("delivery-123", deliveryData["id"])
	s.Equal("webhook-123", deliveryData["webhook_id"])
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestGetDeliveryStatus_NotFound tests delivery status retrieval when the delivery is not found
func (s *WebhookHandlerSuite) TestGetDeliveryStatus_NotFound() {
	// Expect the use case to return a not found error
	s.webhookUseCase.On("GetDeliveryStatus", mock.Anything, "delivery-999", "tenant-123").Return(nil, apperrors.NewResourceNotFoundError("delivery not found"))
	
	// Create a request
	req, _ := http.NewRequest("GET", "/api/v1/webhooks/webhook-123/deliveries/delivery-999", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{
		gin.Param{Key: "id", Value: "webhook-123"},
		gin.Param{Key: "deliveryId", Value: "delivery-999"},
	}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusNotFound, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestRetryDelivery_Success tests successful retry of a webhook delivery
func (s *WebhookHandlerSuite) TestRetryDelivery_Success() {
	// Expect the use case to return a successful retry
	s.webhookUseCase.On("RetryDelivery", mock.Anything, "delivery-123", "tenant-123").Return(nil)
	
	// Create a request
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/webhook-123/deliveries/delivery-123/retry", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{
		gin.Param{Key: "id", Value: "webhook-123"},
		gin.Param{Key: "deliveryId", Value: "delivery-123"},
	}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusAccepted, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains success message
	s.Equal(true, response["success"])
	s.Contains(response, "message")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestRetryDelivery_NotFound tests delivery retry when the delivery is not found
func (s *WebhookHandlerSuite) TestRetryDelivery_NotFound() {
	// Expect the use case to return a not found error
	s.webhookUseCase.On("RetryDelivery", mock.Anything, "delivery-999", "tenant-123").Return(apperrors.NewResourceNotFoundError("delivery not found"))
	
	// Create a request
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/webhook-123/deliveries/delivery-999/retry", nil)
	
	// Create a gin context with the request
	c, _ := gin.CreateTestContext(s.recorder)
	c.Request = req
	s.setupAuthContext(c)
	c.Params = gin.Params{
		gin.Param{Key: "id", Value: "webhook-123"},
		gin.Param{Key: "deliveryId", Value: "delivery-999"},
	}
	
	// Call the handler
	s.router.ServeHTTP(s.recorder, req)
	
	// Assert the response
	s.Equal(http.StatusNotFound, s.recorder.Code)
	
	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(s.recorder.Body.Bytes(), &response)
	s.NoError(err)
	
	// Assert the response contains error information
	s.Equal(false, response["success"])
	s.Contains(response, "error")
	
	// Assert that the use case was called with the expected parameters
	s.webhookUseCase.AssertExpectations(s.T())
}

// TestWebhookHandlerSuite is the entry point for the test suite
func TestWebhookHandlerSuite(t *testing.T) {
	suite.Run(t, new(WebhookHandlerSuite))
}