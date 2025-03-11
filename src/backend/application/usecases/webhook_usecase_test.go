package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock"   // v1.8.0+
	"github.com/stretchr/testify/suite"  // v1.8.0+

	"../../domain/models"
	pkgErrors "../../pkg/errors"
	"../../pkg/utils"
)

// MockWebhookService is a mock implementation of the WebhookService interface for testing
type MockWebhookService struct {
	mock.Mock
}

// CreateWebhook mock implementation for creating a webhook
func (m *MockWebhookService) CreateWebhook(ctx context.Context, webhook *models.Webhook) (string, error) {
	args := m.Called(ctx, webhook)
	return args.String(0), args.Error(1)
}

// GetWebhook mock implementation for retrieving a webhook
func (m *MockWebhookService) GetWebhook(ctx context.Context, id string, tenantID string) (*models.Webhook, error) {
	args := m.Called(ctx, id, tenantID)
	if webhook := args.Get(0); webhook != nil {
		return webhook.(*models.Webhook), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateWebhook mock implementation for updating a webhook
func (m *MockWebhookService) UpdateWebhook(ctx context.Context, webhook *models.Webhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

// DeleteWebhook mock implementation for deleting a webhook
func (m *MockWebhookService) DeleteWebhook(ctx context.Context, id string, tenantID string) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

// ListWebhooks mock implementation for listing webhooks
func (m *MockWebhookService) ListWebhooks(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Webhook], error) {
	args := m.Called(ctx, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Webhook]), args.Error(1)
}

// ProcessEvent mock implementation for processing an event
func (m *MockWebhookService) ProcessEvent(ctx context.Context, event *models.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// DeliverEvent mock implementation for delivering an event to a webhook
func (m *MockWebhookService) DeliverEvent(ctx context.Context, webhook *models.Webhook, event *models.Event, delivery *models.WebhookDelivery) error {
	args := m.Called(ctx, webhook, event, delivery)
	return args.Error(0)
}

// GetDeliveryStatus mock implementation for getting a delivery status
func (m *MockWebhookService) GetDeliveryStatus(ctx context.Context, deliveryID string, tenantID string) (*models.WebhookDelivery, error) {
	args := m.Called(ctx, deliveryID, tenantID)
	if delivery := args.Get(0); delivery != nil {
		return delivery.(*models.WebhookDelivery), args.Error(1)
	}
	return nil, args.Error(1)
}

// ListDeliveries mock implementation for listing deliveries
func (m *MockWebhookService) ListDeliveries(ctx context.Context, webhookID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.WebhookDelivery], error) {
	args := m.Called(ctx, webhookID, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.WebhookDelivery]), args.Error(1)
}

// RetryDelivery mock implementation for retrying a failed delivery
func (m *MockWebhookService) RetryDelivery(ctx context.Context, deliveryID string, tenantID string) error {
	args := m.Called(ctx, deliveryID, tenantID)
	return args.Error(0)
}

// Additional methods required by the WebhookService interface
func (m *MockWebhookService) ProcessPendingDeliveries(ctx context.Context, batchSize int) (int, error) {
	args := m.Called(ctx, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockWebhookService) RetryFailedDeliveries(ctx context.Context, batchSize int) (int, error) {
	args := m.Called(ctx, batchSize)
	return args.Int(0), args.Error(1)
}

// MockEventService is a mock implementation of the EventServiceInterface for testing
type MockEventService struct {
	mock.Mock
}

// GetEvent mock implementation for retrieving an event
func (m *MockEventService) GetEvent(ctx context.Context, id string, tenantID string) (*models.Event, error) {
	args := m.Called(ctx, id, tenantID)
	if event := args.Get(0); event != nil {
		return event.(*models.Event), args.Error(1)
	}
	return nil, args.Error(1)
}

// WebhookUseCaseTestSuite defines a test suite for WebhookUseCase
type WebhookUseCaseTestSuite struct {
	suite.Suite
	mockWebhookService *MockWebhookService
	mockEventService   *MockEventService
	webhookUseCase     WebhookUseCase
}

// SetupTest sets up the test environment before each test
func (s *WebhookUseCaseTestSuite) SetupTest() {
	s.mockWebhookService = new(MockWebhookService)
	s.mockEventService = new(MockEventService)
	
	var err error
	s.webhookUseCase, err = NewWebhookUseCase(s.mockWebhookService, s.mockEventService)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), s.webhookUseCase)
}

// TestNewWebhookUseCase tests the creation of a new WebhookUseCase
func (s *WebhookUseCaseTestSuite) TestNewWebhookUseCase() {
	// Test with valid services
	useCase, err := NewWebhookUseCase(s.mockWebhookService, s.mockEventService)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), useCase)

	// Test with nil webhook service
	useCase, err = NewWebhookUseCase(nil, s.mockEventService)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), useCase)

	// Test with nil event service
	useCase, err = NewWebhookUseCase(s.mockWebhookService, nil)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), useCase)
}

// TestCreateWebhook_Success tests successful webhook creation
func (s *WebhookUseCaseTestSuite) TestCreateWebhook_Success() {
	// Create test webhook
	webhook := &models.Webhook{
		TenantID:   "tenant123",
		URL:        "https://example.com/webhook",
		EventTypes: []string{models.EventTypeDocumentUploaded},
		Status:     models.WebhookStatusActive,
	}

	// Set up mock expectation for CreateWebhook
	expectedID := "webhook123"
	s.mockWebhookService.On("CreateWebhook", mock.Anything, webhook).Return(expectedID, nil)

	// Call webhookUseCase.CreateWebhook
	id, err := s.webhookUseCase.CreateWebhook(context.Background(), webhook)

	// Assert that returned ID matches expected
	assert.Equal(s.T(), expectedID, id)
	assert.Nil(s.T(), err)

	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestCreateWebhook_ValidationError tests webhook creation with validation error
func (s *WebhookUseCaseTestSuite) TestCreateWebhook_ValidationError() {
	// Create test webhook with invalid data
	id, err := s.webhookUseCase.CreateWebhook(context.Background(), nil)
	
	// Assert that error is returned
	assert.Equal(s.T(), "", id)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "CreateWebhook")
}

// TestCreateWebhook_ServiceError tests webhook creation with service error
func (s *WebhookUseCaseTestSuite) TestCreateWebhook_ServiceError() {
	// Create test webhook
	webhook := &models.Webhook{
		TenantID:   "tenant123",
		URL:        "https://example.com/webhook",
		EventTypes: []string{models.EventTypeDocumentUploaded},
		Status:     models.WebhookStatusActive,
	}

	// Set up mock expectation for CreateWebhook to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("CreateWebhook", mock.Anything, webhook).Return("", serviceErr)

	// Call webhookUseCase.CreateWebhook
	id, err := s.webhookUseCase.CreateWebhook(context.Background(), webhook)

	// Assert that error is returned
	assert.Equal(s.T(), "", id)
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestGetWebhook_Success tests successful webhook retrieval
func (s *WebhookUseCaseTestSuite) TestGetWebhook_Success() {
	// Create test webhook
	expectedWebhook := &models.Webhook{
		ID:         "webhook123",
		TenantID:   "tenant123",
		URL:        "https://example.com/webhook",
		EventTypes: []string{models.EventTypeDocumentUploaded},
		Status:     models.WebhookStatusActive,
	}

	// Set up mock expectation for GetWebhook
	s.mockWebhookService.On("GetWebhook", mock.Anything, "webhook123", "tenant123").Return(expectedWebhook, nil)

	// Call webhookUseCase.GetWebhook
	webhook, err := s.webhookUseCase.GetWebhook(context.Background(), "webhook123", "tenant123")

	// Assert that returned webhook matches expected
	assert.Equal(s.T(), expectedWebhook, webhook)
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestGetWebhook_ValidationError tests webhook retrieval with validation error
func (s *WebhookUseCaseTestSuite) TestGetWebhook_ValidationError() {
	// Call webhookUseCase.GetWebhook with empty ID
	webhook, err := s.webhookUseCase.GetWebhook(context.Background(), "", "tenant123")
	assert.Nil(s.T(), webhook)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))

	// Call webhookUseCase.GetWebhook with empty tenantID
	webhook, err = s.webhookUseCase.GetWebhook(context.Background(), "webhook123", "")
	assert.Nil(s.T(), webhook)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "GetWebhook")
}

// TestGetWebhook_NotFound tests webhook retrieval when webhook not found
func (s *WebhookUseCaseTestSuite) TestGetWebhook_NotFound() {
	// Set up mock expectation for GetWebhook to return not found error
	notFoundErr := pkgErrors.NewResourceNotFoundError("webhook not found")
	s.mockWebhookService.On("GetWebhook", mock.Anything, "webhook123", "tenant123").Return(nil, notFoundErr)

	// Call webhookUseCase.GetWebhook
	webhook, err := s.webhookUseCase.GetWebhook(context.Background(), "webhook123", "tenant123")

	// Assert that error is returned
	assert.Nil(s.T(), webhook)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsResourceNotFoundError(err))
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestUpdateWebhook_Success tests successful webhook update
func (s *WebhookUseCaseTestSuite) TestUpdateWebhook_Success() {
	// Create test webhook
	webhook := &models.Webhook{
		ID:         "webhook123",
		TenantID:   "tenant123",
		URL:        "https://example.com/webhook",
		EventTypes: []string{models.EventTypeDocumentUploaded},
		Status:     models.WebhookStatusActive,
	}

	// Set up mock expectation for UpdateWebhook
	s.mockWebhookService.On("UpdateWebhook", mock.Anything, webhook).Return(nil)

	// Call webhookUseCase.UpdateWebhook
	err := s.webhookUseCase.UpdateWebhook(context.Background(), webhook)

	// Assert that no error is returned
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestUpdateWebhook_ValidationError tests webhook update with validation error
func (s *WebhookUseCaseTestSuite) TestUpdateWebhook_ValidationError() {
	// Call webhookUseCase.UpdateWebhook with nil webhook
	err := s.webhookUseCase.UpdateWebhook(context.Background(), nil)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))

	// Create test webhook with invalid data
	invalidWebhook := &models.Webhook{
		// Missing required fields
	}
	err = s.webhookUseCase.UpdateWebhook(context.Background(), invalidWebhook)
	assert.NotNil(s.T(), err)
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "UpdateWebhook")
}

// TestUpdateWebhook_ServiceError tests webhook update with service error
func (s *WebhookUseCaseTestSuite) TestUpdateWebhook_ServiceError() {
	// Create test webhook
	webhook := &models.Webhook{
		ID:         "webhook123",
		TenantID:   "tenant123",
		URL:        "https://example.com/webhook",
		EventTypes: []string{models.EventTypeDocumentUploaded},
		Status:     models.WebhookStatusActive,
	}

	// Set up mock expectation for UpdateWebhook to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("UpdateWebhook", mock.Anything, webhook).Return(serviceErr)

	// Call webhookUseCase.UpdateWebhook
	err := s.webhookUseCase.UpdateWebhook(context.Background(), webhook)

	// Assert that error is returned
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestDeleteWebhook_Success tests successful webhook deletion
func (s *WebhookUseCaseTestSuite) TestDeleteWebhook_Success() {
	// Set up mock expectation for DeleteWebhook
	s.mockWebhookService.On("DeleteWebhook", mock.Anything, "webhook123", "tenant123").Return(nil)

	// Call webhookUseCase.DeleteWebhook
	err := s.webhookUseCase.DeleteWebhook(context.Background(), "webhook123", "tenant123")

	// Assert that no error is returned
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestDeleteWebhook_ValidationError tests webhook deletion with validation error
func (s *WebhookUseCaseTestSuite) TestDeleteWebhook_ValidationError() {
	// Call webhookUseCase.DeleteWebhook with empty ID
	err := s.webhookUseCase.DeleteWebhook(context.Background(), "", "tenant123")
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))

	// Call webhookUseCase.DeleteWebhook with empty tenantID
	err = s.webhookUseCase.DeleteWebhook(context.Background(), "webhook123", "")
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "DeleteWebhook")
}

// TestDeleteWebhook_ServiceError tests webhook deletion with service error
func (s *WebhookUseCaseTestSuite) TestDeleteWebhook_ServiceError() {
	// Set up mock expectation for DeleteWebhook to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("DeleteWebhook", mock.Anything, "webhook123", "tenant123").Return(serviceErr)

	// Call webhookUseCase.DeleteWebhook
	err := s.webhookUseCase.DeleteWebhook(context.Background(), "webhook123", "tenant123")

	// Assert that error is returned
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestListWebhooks_Success tests successful webhook listing
func (s *WebhookUseCaseTestSuite) TestListWebhooks_Success() {
	// Create test webhooks
	webhooks := []models.Webhook{
		{
			ID:         "webhook123",
			TenantID:   "tenant123",
			URL:        "https://example.com/webhook1",
			EventTypes: []string{models.EventTypeDocumentUploaded},
			Status:     models.WebhookStatusActive,
		},
		{
			ID:         "webhook456",
			TenantID:   "tenant123",
			URL:        "https://example.com/webhook2",
			EventTypes: []string{models.EventTypeDocumentProcessed},
			Status:     models.WebhookStatusActive,
		},
	}

	// Create paginated result
	expectedResult := utils.PaginatedResult[models.Webhook]{
		Items: webhooks,
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
			TotalItems:  2,
			HasNext:     false,
			HasPrevious: false,
		},
	}

	// Set up mock expectation for ListWebhooks
	s.mockWebhookService.On("ListWebhooks", mock.Anything, "tenant123", mock.AnythingOfType("*utils.Pagination")).Return(expectedResult, nil)

	// Call webhookUseCase.ListWebhooks
	result, err := s.webhookUseCase.ListWebhooks(context.Background(), "tenant123", 1, 10)

	// Assert that returned webhooks match expected
	assert.Equal(s.T(), expectedResult, result)
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestListWebhooks_ValidationError tests webhook listing with validation error
func (s *WebhookUseCaseTestSuite) TestListWebhooks_ValidationError() {
	// Call webhookUseCase.ListWebhooks with empty tenantID
	result, err := s.webhookUseCase.ListWebhooks(context.Background(), "", 1, 10)
	
	// Assert that error is returned
	assert.Equal(s.T(), utils.PaginatedResult[models.Webhook]{}, result)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "ListWebhooks")
}

// TestListWebhooks_ServiceError tests webhook listing with service error
func (s *WebhookUseCaseTestSuite) TestListWebhooks_ServiceError() {
	// Set up mock expectation for ListWebhooks to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("ListWebhooks", mock.Anything, "tenant123", mock.AnythingOfType("*utils.Pagination")).Return(utils.PaginatedResult[models.Webhook]{}, serviceErr)

	// Call webhookUseCase.ListWebhooks
	result, err := s.webhookUseCase.ListWebhooks(context.Background(), "tenant123", 1, 10)

	// Assert that error is returned
	assert.Equal(s.T(), utils.PaginatedResult[models.Webhook]{}, result)
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestProcessEvent_Success tests successful event processing
func (s *WebhookUseCaseTestSuite) TestProcessEvent_Success() {
	// Create test event
	event := &models.Event{
		ID:       "event123",
		Type:     models.EventTypeDocumentUploaded,
		TenantID: "tenant123",
	}

	// Set up mock expectation for ProcessEvent
	s.mockWebhookService.On("ProcessEvent", mock.Anything, event).Return(nil)

	// Call webhookUseCase.ProcessEvent
	err := s.webhookUseCase.ProcessEvent(context.Background(), event)

	// Assert that no error is returned
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestProcessEvent_ValidationError tests event processing with validation error
func (s *WebhookUseCaseTestSuite) TestProcessEvent_ValidationError() {
	// Call webhookUseCase.ProcessEvent with nil event
	err := s.webhookUseCase.ProcessEvent(context.Background(), nil)
	
	// Assert that error is returned
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "ProcessEvent")
}

// TestProcessEvent_ServiceError tests event processing with service error
func (s *WebhookUseCaseTestSuite) TestProcessEvent_ServiceError() {
	// Create test event
	event := &models.Event{
		ID:       "event123",
		Type:     models.EventTypeDocumentUploaded,
		TenantID: "tenant123",
	}

	// Set up mock expectation for ProcessEvent to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("ProcessEvent", mock.Anything, event).Return(serviceErr)

	// Call webhookUseCase.ProcessEvent
	err := s.webhookUseCase.ProcessEvent(context.Background(), event)

	// Assert that error is returned
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestGetDeliveryStatus_Success tests successful delivery status retrieval
func (s *WebhookUseCaseTestSuite) TestGetDeliveryStatus_Success() {
	// Create test delivery
	expectedDelivery := &models.WebhookDelivery{
		ID:        "delivery123",
		WebhookID: "webhook123",
		Status:    "success",
	}

	// Set up mock expectation for GetDeliveryStatus
	s.mockWebhookService.On("GetDeliveryStatus", mock.Anything, "delivery123", "tenant123").Return(expectedDelivery, nil)

	// Call webhookUseCase.GetDeliveryStatus
	delivery, err := s.webhookUseCase.GetDeliveryStatus(context.Background(), "delivery123", "tenant123")

	// Assert that returned delivery matches expected
	assert.Equal(s.T(), expectedDelivery, delivery)
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestGetDeliveryStatus_ValidationError tests delivery status retrieval with validation error
func (s *WebhookUseCaseTestSuite) TestGetDeliveryStatus_ValidationError() {
	// Call webhookUseCase.GetDeliveryStatus with empty deliveryID
	delivery, err := s.webhookUseCase.GetDeliveryStatus(context.Background(), "", "tenant123")
	assert.Nil(s.T(), delivery)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))

	// Call webhookUseCase.GetDeliveryStatus with empty tenantID
	delivery, err = s.webhookUseCase.GetDeliveryStatus(context.Background(), "delivery123", "")
	assert.Nil(s.T(), delivery)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "GetDeliveryStatus")
}

// TestGetDeliveryStatus_NotFound tests delivery status retrieval when delivery not found
func (s *WebhookUseCaseTestSuite) TestGetDeliveryStatus_NotFound() {
	// Set up mock expectation for GetDeliveryStatus to return not found error
	notFoundErr := pkgErrors.NewResourceNotFoundError("delivery not found")
	s.mockWebhookService.On("GetDeliveryStatus", mock.Anything, "delivery123", "tenant123").Return(nil, notFoundErr)

	// Call webhookUseCase.GetDeliveryStatus
	delivery, err := s.webhookUseCase.GetDeliveryStatus(context.Background(), "delivery123", "tenant123")

	// Assert that error is returned
	assert.Nil(s.T(), delivery)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsResourceNotFoundError(err))
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestListDeliveries_Success tests successful delivery listing
func (s *WebhookUseCaseTestSuite) TestListDeliveries_Success() {
	// Create test deliveries
	deliveries := []models.WebhookDelivery{
		{
			ID:        "delivery123",
			WebhookID: "webhook123",
			Status:    "success",
		},
		{
			ID:        "delivery456",
			WebhookID: "webhook123",
			Status:    "failed",
		},
	}

	// Create paginated result
	expectedResult := utils.PaginatedResult[models.WebhookDelivery]{
		Items: deliveries,
		Pagination: utils.PageInfo{
			Page:        1,
			PageSize:    10,
			TotalPages:  1,
			TotalItems:  2,
			HasNext:     false,
			HasPrevious: false,
		},
	}

	// Set up mock expectation for ListDeliveries
	s.mockWebhookService.On("ListDeliveries", mock.Anything, "webhook123", "tenant123", mock.AnythingOfType("*utils.Pagination")).Return(expectedResult, nil)

	// Call webhookUseCase.ListDeliveries
	result, err := s.webhookUseCase.ListDeliveries(context.Background(), "webhook123", "tenant123", 1, 10)

	// Assert that returned deliveries match expected
	assert.Equal(s.T(), expectedResult, result)
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestListDeliveries_ValidationError tests delivery listing with validation error
func (s *WebhookUseCaseTestSuite) TestListDeliveries_ValidationError() {
	// Call webhookUseCase.ListDeliveries with empty webhookID
	result, err := s.webhookUseCase.ListDeliveries(context.Background(), "", "tenant123", 1, 10)
	assert.Equal(s.T(), utils.PaginatedResult[models.WebhookDelivery]{}, result)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))

	// Call webhookUseCase.ListDeliveries with empty tenantID
	result, err = s.webhookUseCase.ListDeliveries(context.Background(), "webhook123", "", 1, 10)
	assert.Equal(s.T(), utils.PaginatedResult[models.WebhookDelivery]{}, result)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "ListDeliveries")
}

// TestListDeliveries_ServiceError tests delivery listing with service error
func (s *WebhookUseCaseTestSuite) TestListDeliveries_ServiceError() {
	// Set up mock expectation for ListDeliveries to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("ListDeliveries", mock.Anything, "webhook123", "tenant123", mock.AnythingOfType("*utils.Pagination")).Return(utils.PaginatedResult[models.WebhookDelivery]{}, serviceErr)

	// Call webhookUseCase.ListDeliveries
	result, err := s.webhookUseCase.ListDeliveries(context.Background(), "webhook123", "tenant123", 1, 10)

	// Assert that error is returned
	assert.Equal(s.T(), utils.PaginatedResult[models.WebhookDelivery]{}, result)
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestRetryDelivery_Success tests successful delivery retry
func (s *WebhookUseCaseTestSuite) TestRetryDelivery_Success() {
	// Set up mock expectation for RetryDelivery
	s.mockWebhookService.On("RetryDelivery", mock.Anything, "delivery123", "tenant123").Return(nil)

	// Call webhookUseCase.RetryDelivery
	err := s.webhookUseCase.RetryDelivery(context.Background(), "delivery123", "tenant123")

	// Assert that no error is returned
	assert.Nil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestRetryDelivery_ValidationError tests delivery retry with validation error
func (s *WebhookUseCaseTestSuite) TestRetryDelivery_ValidationError() {
	// Call webhookUseCase.RetryDelivery with empty deliveryID
	err := s.webhookUseCase.RetryDelivery(context.Background(), "", "tenant123")
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))

	// Call webhookUseCase.RetryDelivery with empty tenantID
	err = s.webhookUseCase.RetryDelivery(context.Background(), "delivery123", "")
	assert.NotNil(s.T(), err)
	assert.True(s.T(), pkgErrors.IsValidationError(err))
	
	// Verify that no service methods were called
	s.mockWebhookService.AssertNotCalled(s.T(), "RetryDelivery")
}

// TestRetryDelivery_ServiceError tests delivery retry with service error
func (s *WebhookUseCaseTestSuite) TestRetryDelivery_ServiceError() {
	// Set up mock expectation for RetryDelivery to return error
	serviceErr := errors.New("service error")
	s.mockWebhookService.On("RetryDelivery", mock.Anything, "delivery123", "tenant123").Return(serviceErr)

	// Call webhookUseCase.RetryDelivery
	err := s.webhookUseCase.RetryDelivery(context.Background(), "delivery123", "tenant123")

	// Assert that error is returned
	assert.NotNil(s.T(), err)
	
	// Verify that all expectations were met
	s.mockWebhookService.AssertExpectations(s.T())
}

// TestWebhookUseCaseSuite entry point for running the WebhookUseCase test suite
func TestWebhookUseCaseSuite(t *testing.T) {
	suite.Run(t, new(WebhookUseCaseTestSuite))
}