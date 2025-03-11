// Package models contains the domain models for the Document Management Platform
package models

import (
	"errors" // v1.0.0+ - For error handling in validation methods
	"time"   // standard library - For timestamp fields like CreatedAt
)

// Document version status constants
const (
	// VersionStatusProcessing represents a document version that is being processed
	VersionStatusProcessing = "processing"
	
	// VersionStatusAvailable represents a document version that is available for access
	VersionStatusAvailable = "available"
	
	// VersionStatusQuarantined represents a document version that has been quarantined due to virus detection
	VersionStatusQuarantined = "quarantined"
	
	// VersionStatusFailed represents a document version where processing has failed
	VersionStatusFailed = "failed"
)

// DocumentVersion represents a specific version of a document in the system.
// It tracks version-specific information such as version number, size, content hash,
// status, and storage location.
type DocumentVersion struct {
	ID            string    // Unique identifier for the version
	DocumentID    string    // Reference to the parent document
	VersionNumber int       // Sequential version number
	Size          int64     // Size in bytes
	ContentHash   string    // SHA-256 hash of content
	Status        string    // Current status of the version
	StoragePath   string    // S3 storage path
	CreatedAt     time.Time // Creation timestamp
	CreatedBy     string    // User who created this version
}

// NewDocumentVersion creates a new DocumentVersion instance with the given parameters.
// The status is initialized to "processing" and created timestamp is set to current time.
func NewDocumentVersion(documentID string, versionNumber int, size int64, contentHash string, storagePath string, createdBy string) DocumentVersion {
	return DocumentVersion{
		DocumentID:    documentID,
		VersionNumber: versionNumber,
		Size:          size,
		ContentHash:   contentHash,
		StoragePath:   storagePath,
		Status:        VersionStatusProcessing,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
	}
}

// Validate checks that the document version has all required fields and values.
// Returns an error if validation fails, nil otherwise.
func (v *DocumentVersion) Validate() error {
	if v.DocumentID == "" {
		return errors.New("document version must have a document ID")
	}
	
	if v.VersionNumber < 1 {
		return errors.New("version number must be greater than 0")
	}
	
	if v.Size <= 0 {
		return errors.New("size must be greater than 0")
	}
	
	if v.ContentHash == "" {
		return errors.New("content hash is required")
	}
	
	if v.StoragePath == "" {
		return errors.New("storage path is required")
	}
	
	if v.CreatedBy == "" {
		return errors.New("created by is required")
	}
	
	return nil
}

// IsAvailable checks if the document version is available for download
func (v *DocumentVersion) IsAvailable() bool {
	return v.Status == VersionStatusAvailable
}

// IsProcessing checks if the document version is currently being processed
func (v *DocumentVersion) IsProcessing() bool {
	return v.Status == VersionStatusProcessing
}

// IsQuarantined checks if the document version has been quarantined due to virus detection
func (v *DocumentVersion) IsQuarantined() bool {
	return v.Status == VersionStatusQuarantined
}

// IsFailed checks if the document version processing has failed
func (v *DocumentVersion) IsFailed() bool {
	return v.Status == VersionStatusFailed
}

// MarkAsAvailable updates the status of the document version to available
func (v *DocumentVersion) MarkAsAvailable() {
	v.Status = VersionStatusAvailable
}

// MarkAsQuarantined updates the status of the document version to quarantined
func (v *DocumentVersion) MarkAsQuarantined() {
	v.Status = VersionStatusQuarantined
}

// MarkAsFailed updates the status of the document version to failed
func (v *DocumentVersion) MarkAsFailed() {
	v.Status = VersionStatusFailed
}