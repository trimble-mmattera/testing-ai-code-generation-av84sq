// Package models contains the domain models for the Document Management Platform
package models

import (
	"errors" // standard library
	"time"   // standard library
)

// Document status constants define the possible states of a document
const (
	// DocumentStatusProcessing represents a document that is being processed
	DocumentStatusProcessing = "processing"
	
	// DocumentStatusAvailable represents a document that is available for access
	DocumentStatusAvailable = "available"
	
	// DocumentStatusQuarantined represents a document that has been quarantined due to virus detection
	DocumentStatusQuarantined = "quarantined"
	
	// DocumentStatusFailed represents a document where processing has failed
	DocumentStatusFailed = "failed"
)

// Document represents a document in the system with its metadata and relationships.
// This is a core entity that encapsulates document metadata, status, and relationships
// to other entities like folders, versions, and tags.
type Document struct {
	ID          string              // Unique identifier for the document
	Name        string              // Document name (filename)
	ContentType string              // MIME type of the document
	Size        int64               // Size in bytes
	FolderID    string              // Reference to the folder containing this document
	TenantID    string              // Reference to the tenant this document belongs to (ensures tenant isolation)
	OwnerID     string              // Reference to the user who owns this document
	Status      string              // Current status of the document (processing, available, quarantined, failed)
	CreatedAt   time.Time           // Creation timestamp
	UpdatedAt   time.Time           // Last update timestamp
	Metadata    []DocumentMetadata  // Associated metadata key-value pairs
	Versions    []DocumentVersion   // Document versions history
	Tags        []Tag               // Associated tags for categorization
}

// NewDocument creates a new Document instance with the given parameters.
// The status is initialized to "processing" and timestamps are set to current time.
func NewDocument(name, contentType string, size int64, folderID, tenantID, ownerID string) Document {
	now := time.Now()
	return Document{
		Name:        name,
		ContentType: contentType,
		Size:        size,
		FolderID:    folderID,
		TenantID:    tenantID,
		OwnerID:     ownerID,
		Status:      DocumentStatusProcessing,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    []DocumentMetadata{},
		Versions:    []DocumentVersion{},
		Tags:        []Tag{},
	}
}

// Validate checks if the document has all required fields.
// Returns an error if validation fails, nil otherwise.
func (d *Document) Validate() error {
	if d.Name == "" {
		return errors.New("document name is required")
	}
	if d.ContentType == "" {
		return errors.New("content type is required")
	}
	if d.Size <= 0 {
		return errors.New("size must be greater than 0")
	}
	if d.TenantID == "" {
		return errors.New("tenant ID is required")
	}
	if d.FolderID == "" {
		return errors.New("folder ID is required")
	}
	if d.OwnerID == "" {
		return errors.New("owner ID is required")
	}
	return nil
}

// IsAvailable checks if the document is available for download
func (d *Document) IsAvailable() bool {
	return d.Status == DocumentStatusAvailable
}

// IsProcessing checks if the document is currently being processed
func (d *Document) IsProcessing() bool {
	return d.Status == DocumentStatusProcessing
}

// IsQuarantined checks if the document has been quarantined due to virus detection
func (d *Document) IsQuarantined() bool {
	return d.Status == DocumentStatusQuarantined
}

// IsFailed checks if the document processing has failed
func (d *Document) IsFailed() bool {
	return d.Status == DocumentStatusFailed
}

// MarkAsAvailable updates the status of the document to available
func (d *Document) MarkAsAvailable() {
	d.Status = DocumentStatusAvailable
	d.UpdatedAt = time.Now()
}

// MarkAsQuarantined updates the status of the document to quarantined
func (d *Document) MarkAsQuarantined() {
	d.Status = DocumentStatusQuarantined
	d.UpdatedAt = time.Now()
}

// MarkAsFailed updates the status of the document to failed
func (d *Document) MarkAsFailed() {
	d.Status = DocumentStatusFailed
	d.UpdatedAt = time.Now()
}

// AddMetadata adds metadata to the document
func (d *Document) AddMetadata(key, value string) {
	metadata := NewDocumentMetadata(d.ID, key, value)
	d.Metadata = append(d.Metadata, metadata)
	d.UpdatedAt = time.Now()
}

// GetMetadata gets metadata value by key
func (d *Document) GetMetadata(key string) string {
	for _, m := range d.Metadata {
		if m.Key == key {
			return m.Value
		}
	}
	return ""
}

// AddVersion adds a new version to the document
func (d *Document) AddVersion(version DocumentVersion) {
	d.Versions = append(d.Versions, version)
	d.UpdatedAt = time.Now()
}

// GetLatestVersion gets the latest version of the document
func (d *Document) GetLatestVersion() *DocumentVersion {
	if len(d.Versions) == 0 {
		return nil
	}
	
	var latest *DocumentVersion
	highestVersion := 0
	
	for i, v := range d.Versions {
		if v.VersionNumber > highestVersion {
			highestVersion = v.VersionNumber
			latest = &d.Versions[i]
		}
	}
	
	return latest
}

// AddTag adds a tag to the document
func (d *Document) AddTag(tag Tag) {
	d.Tags = append(d.Tags, tag)
	d.UpdatedAt = time.Now()
}

// RemoveTag removes a tag from the document
func (d *Document) RemoveTag(tagID string) bool {
	for i, tag := range d.Tags {
		if tag.ID == tagID {
			// Remove the tag by replacing it with the last element
			// and then reducing the slice length by 1
			d.Tags[i] = d.Tags[len(d.Tags)-1]
			d.Tags = d.Tags[:len(d.Tags)-1]
			d.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// HasTag checks if the document has a specific tag
func (d *Document) HasTag(tagID string) bool {
	for _, tag := range d.Tags {
		if tag.ID == tagID {
			return true
		}
	}
	return false
}