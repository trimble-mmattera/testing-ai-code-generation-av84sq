package sns

import (
	"context"             // standard library
	"encoding/json"       // standard library
	"errors"              // standard library
	"testing"             // standard library

	"github.com/aws/aws-sdk-go/service/sns"      // latest
	"github.com/stretchr/testify/assert"         // v1.8.0+
	"github.com/stretchr/testify/mock"           // v1.8.0+
	"github.com/stretchr/testify/require"        // v1.8.0+

	"../../../domain/models"
	pkgerrors "../../../pkg/errors"
	"../../../pkg/logger"
)

// Define constants for testing
const testTenantID = "tenant-123"
const testDocumentID = "doc-123"
const testFolderID = "folder-123"

// Mock SNS client
type mockSNSClient struct {
	mock.Mock
}

func (m *mockSNSClient) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sns.PublishOutput), args.Error(1)
}

// Helper function to create test events
func createTestEvent(eventType string) *models.Event {
	payload := map[string]interface{}{
		"key": "value",
	}
	if eventType == models.EventTypeDocumentUploaded || eventType == models.EventTypeDocumentProcessed {
		payload["documentID"] = testDocumentID
	} else if eventType == models.EventTypeFolderCreated {
		payload["folderID"] = testFolderID
	}
	
	jsonPayload, _ := json.Marshal(payload)
	
	return &models.Event{
		Type:     eventType,
		TenantID: testTenantID,
		Payload:  jsonPayload,
	}
}

// Helper function for common test setup
func setupTest(t *testing.T) (*mockSNSClient, *EventPublisher) {
	// Initialize logger
	err := logger.Init(logger.LogConfig{
		Level:       "info",
		Format:      "json",
		Output:      "console",
		Development: true,
	})
	require.NoError(t, err)

	// Create mock SNS client
	mockSNS := new(mockSNSClient)
	
	// Create event publisher
	publisher, err := NewEventPublisher(mockSNS, logger.WithField("test", true))
	require.NoError(t, err)
	
	return mockSNS, publisher
}

// Test NewEventPublisher with valid input
func TestNewEventPublisher_ValidInput(t *testing.T) {
	// Initialize logger
	err := logger.Init(logger.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})
	assert.NoError(t, err)
	
	// Create mock SNS client
	mockSNS := new(mockSNSClient)
	
	// Call NewEventPublisher with valid inputs
	publisher, err := NewEventPublisher(mockSNS, logger.WithField("test", true))
	
	// Assert results
	assert.NotNil(t, publisher)
	assert.NoError(t, err)
}

// Test NewEventPublisher with nil SNS client
func TestNewEventPublisher_NilSNSClient(t *testing.T) {
	// Initialize logger
	err := logger.Init(logger.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "console",
	})
	assert.NoError(t, err)
	
	// Call NewEventPublisher with nil SNS client
	publisher, err := NewEventPublisher(nil, logger.WithField("test", true))
	
	// Assert results
	assert.Nil(t, publisher)
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsValidationError(err))
}

// Test NewEventPublisher with nil logger
func TestNewEventPublisher_NilLogger(t *testing.T) {
	// Create mock SNS client
	mockSNS := new(mockSNSClient)
	
	// Call NewEventPublisher with nil logger
	publisher, err := NewEventPublisher(mockSNS, nil)
	
	// Assert results
	assert.Nil(t, publisher)
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsValidationError(err))
}

// Test PublishEvent with successful SNS response
func TestEventPublisher_PublishEvent_Success(t *testing.T) {
	// Setup
	mockSNS, publisher := setupTest(t)
	
	// Create test event
	event := createTestEvent(models.EventTypeDocumentUploaded)
	
	// Set up mock expectation
	mockSNS.On("Publish", mock.Anything).Return(&sns.PublishOutput{}, nil)
	
	// Call PublishEvent
	err := publisher.PublishEvent(context.Background(), event)
	
	// Assert results
	assert.NoError(t, err)
	mockSNS.AssertExpectations(t)
}

// Test PublishEvent with nil event
func TestEventPublisher_PublishEvent_NilEvent(t *testing.T) {
	// Setup
	_, publisher := setupTest(t)
	
	// Call PublishEvent with nil event
	err := publisher.PublishEvent(context.Background(), nil)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsValidationError(err))
}

// Test PublishEvent with invalid event
func TestEventPublisher_PublishEvent_InvalidEvent(t *testing.T) {
	// Setup
	_, publisher := setupTest(t)
	
	// Create invalid event (missing required fields)
	invalidEvent := &models.Event{
		// Missing Type, TenantID, and Payload
	}
	
	// Call PublishEvent with invalid event
	err := publisher.PublishEvent(context.Background(), invalidEvent)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsValidationError(err))
}

// Test PublishEvent with SNS client error
func TestEventPublisher_PublishEvent_SNSError(t *testing.T) {
	// Setup
	mockSNS, publisher := setupTest(t)
	
	// Create test event
	event := createTestEvent(models.EventTypeDocumentUploaded)
	
	// Set up mock expectation to return error
	mockSNS.On("Publish", mock.Anything).Return(nil, errors.New("SNS error"))
	
	// Call PublishEvent
	err := publisher.PublishEvent(context.Background(), event)
	
	// Assert results
	assert.Error(t, err)
	mockSNS.AssertExpectations(t)
}

// Test that different event types are properly published
func TestEventPublisher_getTopicForEventType(t *testing.T) {
	// Setup
	mockSNS, publisher := setupTest(t)
	
	// We'll test three event types
	docEvent := createTestEvent(models.EventTypeDocumentUploaded)
	docProcessedEvent := createTestEvent(models.EventTypeDocumentProcessed)
	folderEvent := createTestEvent(models.EventTypeFolderCreated)
	
	// Set up mock to accept all publishes
	mockSNS.On("Publish", mock.Anything).Return(&sns.PublishOutput{}, nil).Times(3)
	
	// Publish all three event types
	err := publisher.PublishEvent(context.Background(), docEvent)
	assert.NoError(t, err)
	
	err = publisher.PublishEvent(context.Background(), docProcessedEvent)
	assert.NoError(t, err)
	
	err = publisher.PublishEvent(context.Background(), folderEvent)
	assert.NoError(t, err)
	
	// Verify the mock was called the expected number of times
	mockSNS.AssertNumberOfCalls(t, "Publish", 3)
}