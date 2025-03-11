// Package sns provides an implementation of the EventPublisher using AWS SNS
// for the Document Management Platform's event-driven architecture.
package sns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"../../../domain/models"
	"../../../pkg/errors"
	"../../../pkg/logger"
)

// Constants for SNS topics
const (
	documentEventsTopic = "document-events"
	folderEventsTopic   = "folder-events"
	defaultEventsTopic  = "general-events"
)

// EventPublisherInterface defines the contract for publishing domain events
type EventPublisherInterface interface {
	// PublishEvent publishes a domain event to the appropriate messaging topic
	PublishEvent(ctx context.Context, event *models.Event) error
}

// EventPublisher implements EventPublisherInterface using AWS SNS
type EventPublisher struct {
	snsClient SNSClientInterface
	logger    *logger.Logger
}

// NewEventPublisher creates a new EventPublisher with the provided SNS client and logger
func NewEventPublisher(snsClient SNSClientInterface, logger *logger.Logger) EventPublisherInterface {
	// Validate that snsClient is not nil
	if snsClient == nil {
		return nil
	}

	// Validate that logger is not nil
	if logger == nil {
		return nil
	}

	return &EventPublisher{
		snsClient: snsClient,
		logger:    logger,
	}
}

// PublishEvent publishes a domain event to the appropriate SNS topic
func (p *EventPublisher) PublishEvent(ctx context.Context, event *models.Event) error {
	// Use the instance logger
	log := p.logger

	// Validate that event is not nil
	if event == nil {
		return errors.NewValidationError("event cannot be nil")
	}

	// Add event details to logger
	log = log.WithField("eventType", event.Type)
	log = log.WithField("tenantID", event.TenantID)

	// Validate event fields
	if err := event.Validate(); err != nil {
		log.WithError(err).Error("Invalid event")
		return errors.Wrap(err, "invalid event")
	}

	// Determine the appropriate SNS topic for the event type
	topic := p.getTopicForEventType(event.Type)
	log = log.WithField("topic", topic)

	// Marshal the event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.WithError(err).Error("Failed to marshal event to JSON")
		return errors.Wrap(err, "failed to marshal event to JSON")
	}

	// Call snsClient.Publish with the topic and JSON payload
	messageID, err := p.snsClient.Publish(ctx, topic, string(eventJSON))
	if err != nil {
		log.WithError(err).Error("Failed to publish event to SNS")
		return errors.Wrap(err, "failed to publish event to SNS")
	}

	// Log successful publishing
	log.WithField("messageId", messageID).Info("Successfully published event to SNS")

	return nil
}

// getTopicForEventType maps event types to appropriate SNS topics
func (p *EventPublisher) getTopicForEventType(eventType string) string {
	// Check if eventType starts with 'document.'
	if strings.HasPrefix(eventType, "document.") {
		return documentEventsTopic
	}

	// Check if eventType starts with 'folder.'
	if strings.HasPrefix(eventType, "folder.") {
		return folderEventsTopic
	}

	// Otherwise, return defaultEventsTopic
	return defaultEventsTopic
}