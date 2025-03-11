// Package services provides domain-level services for the Document Management Platform
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"../models"
	"../repositories"
	"../../infrastructure/messaging/sns"
	"../../pkg/logger"
	"../../pkg/errors"
	"../../pkg/utils"
)

// EventServiceInterface defines the contract for event management operations
type EventServiceInterface interface {
	// CreateEvent creates a new event in the system
	CreateEvent(ctx context.Context, event *models.Event) (string, error)

	// GetEvent retrieves an event by its ID with tenant isolation
	GetEvent(ctx context.Context, id string, tenantID string) (*models.Event, error)

	// ListEventsByType lists events of a specific type with pagination and tenant isolation
	ListEventsByType(ctx context.Context, eventType string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error)

	// ListEventsByTenant lists all events for a tenant with pagination
	ListEventsByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error)

	// PublishEvent publishes an event to the messaging system
	PublishEvent(ctx context.Context, event *models.Event) error

	// CreateAndPublishDocumentEvent creates and publishes a document-related event
	CreateAndPublishDocumentEvent(ctx context.Context, eventType string, tenantID string, documentID string, additionalData map[string]interface{}) (string, error)

	// CreateAndPublishFolderEvent creates and publishes a folder-related event
	CreateAndPublishFolderEvent(ctx context.Context, eventType string, tenantID string, folderID string, additionalData map[string]interface{}) (string, error)
}

// eventService implements the EventServiceInterface
type eventService struct {
	eventRepo      repositories.EventRepository
	eventPublisher sns.EventPublisherInterface
	logger         *logger.Logger
}

// NewEventService creates a new EventService instance
func NewEventService(eventRepo repositories.EventRepository, eventPublisher sns.EventPublisherInterface) EventServiceInterface {
	// Validate that eventRepo is not nil
	if eventRepo == nil {
		panic("eventRepo cannot be nil")
	}

	// Validate that eventPublisher is not nil
	if eventPublisher == nil {
		panic("eventPublisher cannot be nil")
	}

	// Initialize logger
	return &eventService{
		eventRepo:      eventRepo,
		eventPublisher: eventPublisher,
		logger:         logger.WithField("service", "event_service"),
	}
}

// CreateEvent creates a new event in the system
func (s *eventService) CreateEvent(ctx context.Context, event *models.Event) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate event is not nil
	if event == nil {
		log.Error("Event cannot be nil")
		return "", errors.NewValidationError("event cannot be nil")
	}

	// Validate event fields using event.Validate()
	if err := event.Validate(); err != nil {
		log.WithError(err).Error("Invalid event")
		return "", errors.Wrap(err, "invalid event")
	}

	// Call eventRepo.Create to persist the event
	eventID, err := s.eventRepo.Create(ctx, event)
	if err != nil {
		log.WithError(err).Error("Failed to create event")
		return "", errors.Wrap(err, "failed to create event")
	}

	// Log successful creation
	log.Info("Event created successfully", "eventID", eventID)
	return eventID, nil
}

// GetEvent retrieves an event by its ID with tenant isolation
func (s *eventService) GetEvent(ctx context.Context, id string, tenantID string) (*models.Event, error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate ID is not empty
	if id == "" {
		log.Error("Event ID cannot be empty")
		return nil, errors.NewValidationError("event ID is required")
	}

	// Validate tenantID is not empty
	if tenantID == "" {
		log.Error("Tenant ID cannot be empty")
		return nil, errors.NewValidationError("tenant ID is required")
	}

	// Call eventRepo.GetByID to retrieve the event
	event, err := s.eventRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get event", "eventID", id, "tenantID", tenantID)
		return nil, errors.Wrap(err, "failed to get event")
	}

	// Verify event belongs to the specified tenant
	if event == nil {
		log.Error("Event not found", "eventID", id, "tenantID", tenantID)
		return nil, errors.NewResourceNotFoundError(fmt.Sprintf("event with ID %s not found", id))
	}

	// Log successful retrieval
	log.Info("Event retrieved successfully", "eventID", id, "tenantID", tenantID)
	return event, nil
}

// ListEventsByType lists events of a specific type with pagination and tenant isolation
func (s *eventService) ListEventsByType(ctx context.Context, eventType string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate eventType is not empty
	if eventType == "" {
		log.Error("Event type cannot be empty")
		return utils.PaginatedResult[models.Event]{}, errors.NewValidationError("event type is required")
	}

	// Validate tenantID is not empty
	if tenantID == "" {
		log.Error("Tenant ID cannot be empty")
		return utils.PaginatedResult[models.Event]{}, errors.NewValidationError("tenant ID is required")
	}

	// If pagination is nil, initialize with default values
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Call eventRepo.ListByType to retrieve events
	result, err := s.eventRepo.ListByType(ctx, eventType, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("Failed to list events by type", "eventType", eventType, "tenantID", tenantID)
		return utils.PaginatedResult[models.Event]{}, errors.Wrap(err, "failed to list events by type")
	}

	// Log successful listing
	log.Info("Events listed successfully", "eventType", eventType, "tenantID", tenantID, "count", len(result.Items))
	return result, nil
}

// ListEventsByTenant lists all events for a tenant with pagination
func (s *eventService) ListEventsByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Event], error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate tenantID is not empty
	if tenantID == "" {
		log.Error("Tenant ID cannot be empty")
		return utils.PaginatedResult[models.Event]{}, errors.NewValidationError("tenant ID is required")
	}

	// If pagination is nil, initialize with default values
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	// Call eventRepo.ListByTenant to retrieve events
	result, err := s.eventRepo.ListByTenant(ctx, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("Failed to list events by tenant", "tenantID", tenantID)
		return utils.PaginatedResult[models.Event]{}, errors.Wrap(err, "failed to list events by tenant")
	}

	// Log successful listing
	log.Info("Events listed successfully", "tenantID", tenantID, "count", len(result.Items))
	return result, nil
}

// PublishEvent publishes an event to the messaging system
func (s *eventService) PublishEvent(ctx context.Context, event *models.Event) error {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate event is not nil
	if event == nil {
		log.Error("Event cannot be nil")
		return errors.NewValidationError("event cannot be nil")
	}

	// Validate event fields using event.Validate()
	if err := event.Validate(); err != nil {
		log.WithError(err).Error("Invalid event")
		return errors.Wrap(err, "invalid event")
	}

	// If event ID is empty, persist it first using eventRepo.Create
	if event.ID == "" {
		eventID, err := s.CreateEvent(ctx, event)
		if err != nil {
			log.WithError(err).Error("Failed to create event before publishing")
			return errors.Wrap(err, "failed to create event before publishing")
		}
		event.ID = eventID
	}

	// Call eventPublisher.PublishEvent to publish the event
	err := s.eventPublisher.PublishEvent(ctx, event)
	if err != nil {
		log.WithError(err).Error("Failed to publish event", "eventID", event.ID, "eventType", event.Type)
		return errors.Wrap(err, "failed to publish event")
	}

	// Log successful publishing
	log.Info("Event published successfully", "eventID", event.ID, "eventType", event.Type)
	return nil
}

// CreateAndPublishDocumentEvent creates and publishes a document-related event
func (s *eventService) CreateAndPublishDocumentEvent(ctx context.Context, eventType string, tenantID string, documentID string, additionalData map[string]interface{}) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate eventType is not empty
	if eventType == "" {
		log.Error("Event type cannot be empty")
		return "", errors.NewValidationError("event type is required")
	}

	// Validate tenantID is not empty
	if tenantID == "" {
		log.Error("Tenant ID cannot be empty")
		return "", errors.NewValidationError("tenant ID is required")
	}

	// Validate documentID is not empty
	if documentID == "" {
		log.Error("Document ID cannot be empty")
		return "", errors.NewValidationError("document ID is required")
	}

	// Create payload map with documentID
	payload := map[string]interface{}{
		"documentID": documentID,
	}

	// Add additionalData to payload if provided
	if additionalData != nil {
		for k, v := range additionalData {
			payload[k] = v
		}
	}

	// Marshal payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.WithError(err).Error("Failed to marshal payload")
		return "", errors.Wrap(err, "failed to marshal payload")
	}

	// Create new Event with eventType, tenantID, and payload
	event := models.NewEvent(eventType, tenantID, payloadJSON)
	if event == nil {
		log.Error("Failed to create event")
		return "", errors.NewInternalError("failed to create event")
	}

	// Call PublishEvent to persist and publish the event
	err = s.PublishEvent(ctx, event)
	if err != nil {
		log.WithError(err).Error("Failed to publish document event")
		return "", errors.Wrap(err, "failed to publish document event")
	}

	// Log successful event creation and publishing
	log.Info("Document event created and published successfully", 
		"eventID", event.ID, 
		"eventType", eventType, 
		"documentID", documentID)
	return event.ID, nil
}

// CreateAndPublishFolderEvent creates and publishes a folder-related event
func (s *eventService) CreateAndPublishFolderEvent(ctx context.Context, eventType string, tenantID string, folderID string, additionalData map[string]interface{}) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate eventType is not empty
	if eventType == "" {
		log.Error("Event type cannot be empty")
		return "", errors.NewValidationError("event type is required")
	}

	// Validate tenantID is not empty
	if tenantID == "" {
		log.Error("Tenant ID cannot be empty")
		return "", errors.NewValidationError("tenant ID is required")
	}

	// Validate folderID is not empty
	if folderID == "" {
		log.Error("Folder ID cannot be empty")
		return "", errors.NewValidationError("folder ID is required")
	}

	// Create payload map with folderID
	payload := map[string]interface{}{
		"folderID": folderID,
	}

	// Add additionalData to payload if provided
	if additionalData != nil {
		for k, v := range additionalData {
			payload[k] = v
		}
	}

	// Marshal payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.WithError(err).Error("Failed to marshal payload")
		return "", errors.Wrap(err, "failed to marshal payload")
	}

	// Create new Event with eventType, tenantID, and payload
	event := models.NewEvent(eventType, tenantID, payloadJSON)
	if event == nil {
		log.Error("Failed to create event")
		return "", errors.NewInternalError("failed to create event")
	}

	// Call PublishEvent to persist and publish the event
	err = s.PublishEvent(ctx, event)
	if err != nil {
		log.WithError(err).Error("Failed to publish folder event")
		return "", errors.Wrap(err, "failed to publish folder event")
	}

	// Log successful event creation and publishing
	log.Info("Folder event created and published successfully", 
		"eventID", event.ID, 
		"eventType", eventType, 
		"folderID", folderID)
	return event.ID, nil
}

// validateInput validates input parameters
func (s *eventService) validateInput(params map[string]string) error {
	// Check each parameter in the map
	for key, value := range params {
		// If any parameter is empty, return validation error
		if value == "" {
			return errors.NewValidationError(fmt.Sprintf("%s is required", key))
		}
	}
	// Return nil if all parameters are valid
	return nil
}