package models

import (
	"errors"
	"time" // standard library
)

// DocumentMetadata represents metadata associated with a document in the system.
// It enables storing and retrieving key-value pairs of metadata for documents,
// supporting the document search and filtering capabilities.
type DocumentMetadata struct {
	// ID is the unique identifier for the metadata record
	ID string

	// DocumentID is a reference to the document this metadata belongs to
	DocumentID string

	// Key is the metadata key (e.g. "author", "department", "category")
	Key string

	// Value is the metadata value associated with the key
	Value string

	// CreatedAt is when the metadata was created
	CreatedAt time.Time

	// UpdatedAt is when the metadata was last updated
	UpdatedAt time.Time
}

// NewDocumentMetadata creates a new DocumentMetadata instance with the given document ID, key, and value.
// The CreatedAt and UpdatedAt fields are automatically set to the current time.
func NewDocumentMetadata(documentID, key, value string) DocumentMetadata {
	now := time.Now()
	return DocumentMetadata{
		DocumentID: documentID,
		Key:        key,
		Value:      value,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Validate checks if the DocumentMetadata has all required fields.
// Returns an error if any required field is missing, nil otherwise.
func (m *DocumentMetadata) Validate() error {
	if m.DocumentID == "" {
		return errors.New("document ID is required")
	}
	if m.Key == "" {
		return errors.New("key is required")
	}
	return nil
}

// Update changes the value of the metadata and updates the UpdatedAt timestamp.
// This method should be used instead of directly modifying the Value field to
// ensure the UpdatedAt timestamp is correctly maintained.
func (m *DocumentMetadata) Update(newValue string) {
	m.Value = newValue
	m.UpdatedAt = time.Now()
}