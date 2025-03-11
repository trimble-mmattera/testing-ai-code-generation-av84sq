// Package services defines domain service interfaces for the document management platform.
package services

import (
	"context" // standard library
	"io"      // standard library
)

// StorageService defines the contract for document storage operations.
// It provides methods for storing, retrieving, and managing documents
// across different storage locations while maintaining tenant isolation and security.
type StorageService interface {
	// StoreTemporary stores a document in temporary storage during processing.
	// It ensures tenant isolation by using tenantID in the storage path.
	// Returns the storage path where the document is stored or an error if storage fails.
	StoreTemporary(ctx context.Context, tenantID string, documentID string, content io.Reader, size int64, contentType string) (string, error)

	// StorePermanent moves a document from temporary to permanent storage after processing.
	// It ensures tenant isolation by using tenantID in the storage path.
	// Returns the permanent storage path or an error if the move fails.
	StorePermanent(ctx context.Context, tenantID string, documentID string, versionID string, folderID string, tempPath string) (string, error)

	// MoveToQuarantine moves a document from temporary to quarantine storage when a virus is detected.
	// It ensures tenant isolation by using tenantID in the storage path.
	// Returns the quarantine storage path or an error if the move fails.
	MoveToQuarantine(ctx context.Context, tenantID string, documentID string, tempPath string) (string, error)

	// GetDocument retrieves a document from storage.
	// Returns a content stream or an error if retrieval fails.
	GetDocument(ctx context.Context, storagePath string) (io.ReadCloser, error)

	// GetPresignedURL generates a presigned URL for direct document download.
	// Returns a presigned URL or an error if URL generation fails.
	GetPresignedURL(ctx context.Context, storagePath string, fileName string, expirationSeconds int) (string, error)

	// DeleteDocument deletes a document from storage.
	// Returns an error if deletion fails.
	DeleteDocument(ctx context.Context, storagePath string) error

	// CreateBatchArchive creates a compressed archive of multiple documents.
	// Returns an archive stream or an error if archive creation fails.
	CreateBatchArchive(ctx context.Context, storagePaths []string, filenames []string) (io.ReadCloser, error)
}