package services

import (
	"context"
	"io"
)

// DefaultThumbnailWidth defines the default width for generated thumbnails
const DefaultThumbnailWidth = 256

// DefaultThumbnailHeight defines the default height for generated thumbnails
const DefaultThumbnailHeight = 256

// DefaultThumbnailFormat defines the default format for generated thumbnails
const DefaultThumbnailFormat = "png"

// ThumbnailService defines the interface for document thumbnail operations
// It provides methods for generating, retrieving, and managing document thumbnails
type ThumbnailService interface {
	// GenerateThumbnail generates a thumbnail for a document
	// It takes the document ID, version ID, tenant ID, and storage path
	// Returns the thumbnail storage path and any error encountered
	GenerateThumbnail(ctx context.Context, documentID, versionID, tenantID, storagePath string) (string, error)

	// GetThumbnail retrieves a document thumbnail
	// It takes the document ID, version ID, and tenant ID
	// Returns a stream containing the thumbnail content and any error encountered
	GetThumbnail(ctx context.Context, documentID, versionID, tenantID string) (io.ReadCloser, error)

	// GetThumbnailURL generates a URL for accessing a document thumbnail
	// It takes the document ID, version ID, tenant ID, and expiration in seconds
	// Returns a presigned URL for the thumbnail and any error encountered
	GetThumbnailURL(ctx context.Context, documentID, versionID, tenantID string, expirationSeconds int) (string, error)

	// DeleteThumbnail deletes a document thumbnail
	// It takes the document ID, version ID, and tenant ID
	// Returns any error encountered during deletion
	DeleteThumbnail(ctx context.Context, documentID, versionID, tenantID string) error
}