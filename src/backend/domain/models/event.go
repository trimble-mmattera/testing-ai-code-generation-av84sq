// Package models defines the core domain models for the document management platform
package models

import (
	"encoding/json" // standard library - For handling JSON payloads in events
	"errors"        // standard library - For error handling in validation methods
	"strings"
	"time"          // standard library - For timestamp fields like CreatedAt and OccurredAt
)

// Event type constants
const (
	EventTypeDocumentUploaded    = "document.uploaded"
	EventTypeDocumentProcessed   = "document.processed"
	EventTypeDocumentQuarantined = "document.quarantined"
	EventTypeDocumentDownloaded  = "document.downloaded"
	EventTypeFolderCreated       = "folder.created"
	EventTypeFolderUpdated       = "folder.updated"
)

// Event represents a domain event in the system for document and folder operations
type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	TenantID   string          `json:"tenant_id"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Validate ensures that the event has all required fields
func (e *Event) Validate() error {
	if e.Type == "" {
		return errors.New("event type is required")
	}
	if e.TenantID == "" {
		return errors.New("tenant ID is required")
	}
	if e.Payload == nil || len(e.Payload) == 0 {
		return errors.New("payload is required")
	}
	return nil
}

// GetPayloadAsMap unmarshals the JSON payload into a map
func (e *Event) GetPayloadAsMap() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(e.Payload, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetDocumentID extracts the document ID from the payload if present
func (e *Event) GetDocumentID() (string, error) {
	payload, err := e.GetPayloadAsMap()
	if err != nil {
		return "", err
	}
	
	if docID, ok := payload["documentID"]; ok {
		if docIDStr, ok := docID.(string); ok {
			return docIDStr, nil
		}
	}
	
	return "", nil
}

// GetFolderID extracts the folder ID from the payload if present
func (e *Event) GetFolderID() (string, error) {
	payload, err := e.GetPayloadAsMap()
	if err != nil {
		return "", err
	}
	
	if folderID, ok := payload["folderID"]; ok {
		if folderIDStr, ok := folderID.(string); ok {
			return folderIDStr, nil
		}
	}
	
	return "", nil
}

// IsDocumentEvent checks if this is a document-related event
func (e *Event) IsDocumentEvent() bool {
	return strings.HasPrefix(e.Type, "document.")
}

// IsFolderEvent checks if this is a folder-related event
func (e *Event) IsFolderEvent() bool {
	return strings.HasPrefix(e.Type, "folder.")
}

// NewEvent creates a new Event instance with the given parameters
func NewEvent(eventType string, tenantID string, payload json.RawMessage) *Event {
	// Validate that eventType is not empty
	if eventType == "" {
		return nil
	}
	// Validate that tenantID is not empty
	if tenantID == "" {
		return nil
	}
	// Validate that payload is not nil or empty
	if payload == nil || len(payload) == 0 {
		return nil
	}
	
	now := time.Now().UTC()
	return &Event{
		Type:       eventType,
		TenantID:   tenantID,
		Payload:    payload,
		OccurredAt: now,
		CreatedAt:  now,
	}
}

// NewDocumentUploadedEvent creates a new document.uploaded event
func NewDocumentUploadedEvent(tenantID string, documentID string, metadata map[string]interface{}) (*Event, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if documentID == "" {
		return nil, errors.New("document ID is required")
	}
	
	// Create a payload map with documentID and metadata
	payload := map[string]interface{}{
		"documentID": documentID,
		"metadata":   metadata,
	}
	
	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	// Call NewEvent with EventTypeDocumentUploaded, tenantID, and the JSON payload
	event := NewEvent(EventTypeDocumentUploaded, tenantID, jsonPayload)
	if event == nil {
		return nil, errors.New("failed to create event")
	}
	
	return event, nil
}

// NewDocumentProcessedEvent creates a new document.processed event
func NewDocumentProcessedEvent(tenantID string, documentID string, status string) (*Event, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if documentID == "" {
		return nil, errors.New("document ID is required")
	}
	if status == "" {
		return nil, errors.New("status is required")
	}
	
	// Create a payload map with documentID and status
	payload := map[string]interface{}{
		"documentID": documentID,
		"status":     status,
	}
	
	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	// Call NewEvent with EventTypeDocumentProcessed, tenantID, and the JSON payload
	event := NewEvent(EventTypeDocumentProcessed, tenantID, jsonPayload)
	if event == nil {
		return nil, errors.New("failed to create event")
	}
	
	return event, nil
}

// NewDocumentQuarantinedEvent creates a new document.quarantined event
func NewDocumentQuarantinedEvent(tenantID string, documentID string, reason string) (*Event, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if documentID == "" {
		return nil, errors.New("document ID is required")
	}
	
	// Create a payload map with documentID and reason
	payload := map[string]interface{}{
		"documentID": documentID,
		"reason":     reason,
	}
	
	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	// Call NewEvent with EventTypeDocumentQuarantined, tenantID, and the JSON payload
	event := NewEvent(EventTypeDocumentQuarantined, tenantID, jsonPayload)
	if event == nil {
		return nil, errors.New("failed to create event")
	}
	
	return event, nil
}

// NewDocumentDownloadedEvent creates a new document.downloaded event
func NewDocumentDownloadedEvent(tenantID string, documentID string, userID string) (*Event, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if documentID == "" {
		return nil, errors.New("document ID is required")
	}
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	
	// Create a payload map with documentID and userID
	payload := map[string]interface{}{
		"documentID": documentID,
		"userID":     userID,
	}
	
	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	// Call NewEvent with EventTypeDocumentDownloaded, tenantID, and the JSON payload
	event := NewEvent(EventTypeDocumentDownloaded, tenantID, jsonPayload)
	if event == nil {
		return nil, errors.New("failed to create event")
	}
	
	return event, nil
}

// NewFolderCreatedEvent creates a new folder.created event
func NewFolderCreatedEvent(tenantID string, folderID string, metadata map[string]interface{}) (*Event, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if folderID == "" {
		return nil, errors.New("folder ID is required")
	}
	
	// Create a payload map with folderID and metadata
	payload := map[string]interface{}{
		"folderID": folderID,
		"metadata": metadata,
	}
	
	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	// Call NewEvent with EventTypeFolderCreated, tenantID, and the JSON payload
	event := NewEvent(EventTypeFolderCreated, tenantID, jsonPayload)
	if event == nil {
		return nil, errors.New("failed to create event")
	}
	
	return event, nil
}

// NewFolderUpdatedEvent creates a new folder.updated event
func NewFolderUpdatedEvent(tenantID string, folderID string, changes map[string]interface{}) (*Event, error) {
	if tenantID == "" {
		return nil, errors.New("tenant ID is required")
	}
	if folderID == "" {
		return nil, errors.New("folder ID is required")
	}
	
	// Create a payload map with folderID and changes
	payload := map[string]interface{}{
		"folderID": folderID,
		"changes":  changes,
	}
	
	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	
	// Call NewEvent with EventTypeFolderUpdated, tenantID, and the JSON payload
	event := NewEvent(EventTypeFolderUpdated, tenantID, jsonPayload)
	if event == nil {
		return nil, errors.New("failed to create event")
	}
	
	return event, nil
}