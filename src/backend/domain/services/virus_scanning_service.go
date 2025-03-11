// Package services contains domain service interfaces and types for the document management platform.
package services

import (
	"context" // v1.6.0
	"io"      // v1.0.0
)

// Scan result constants
const (
	ScanResultClean    = "clean"    // Document is clean
	ScanResultInfected = "infected" // Document is infected
	ScanResultError    = "error"    // Error during scanning
)

// ScanTask represents a document scanning task in the queue.
type ScanTask struct {
	DocumentID  string // Unique identifier of the document
	VersionID   string // Version identifier of the document
	TenantID    string // Tenant identifier
	StoragePath string // Path to the document in storage
	RetryCount  int    // Number of retry attempts
}

// ScannerClient is an interface for virus scanning implementations.
type ScannerClient interface {
	// ScanStream scans a document stream for viruses.
	// Returns scan result constant, additional details (virus name if infected), and error if scanning fails.
	ScanStream(ctx context.Context, content io.Reader) (string, string, error)
}

// ScanQueue is an interface for managing the document scanning queue.
type ScanQueue interface {
	// Enqueue adds a document to the scanning queue.
	Enqueue(ctx context.Context, task ScanTask) error
	
	// Dequeue retrieves the next document to scan from the queue.
	// Returns the next scan task or nil if queue is empty.
	Dequeue(ctx context.Context) (*ScanTask, error)
	
	// Complete marks a scan task as completed and removes it from the queue.
	Complete(ctx context.Context, task ScanTask) error
	
	// Retry requeues a scan task for retry after a failure.
	Retry(ctx context.Context, task ScanTask) error
	
	// DeadLetter moves a scan task to the dead letter queue after maximum retries.
	DeadLetter(ctx context.Context, task ScanTask, reason string) error
}

// VirusScanningService is an interface for virus scanning service operations.
type VirusScanningService interface {
	// QueueForScanning queues a document for virus scanning.
	QueueForScanning(ctx context.Context, documentID, versionID, tenantID, storagePath string) error
	
	// ProcessScanQueue processes the virus scanning queue.
	// Returns the number of documents processed and error if processing fails.
	ProcessScanQueue(ctx context.Context, batchSize int) (int, error)
	
	// ScanDocument scans a document for viruses.
	// Returns scan result constant, additional details (virus name if infected), and error if scanning fails.
	ScanDocument(ctx context.Context, storagePath string) (string, string, error)
	
	// MoveToQuarantine moves an infected document to quarantine storage.
	// Returns the quarantine storage path and error if quarantine fails.
	MoveToQuarantine(ctx context.Context, tenantID, documentID, versionID, sourcePath string) (string, error)
	
	// GetScanStatus gets the current scan status of a document.
	// Returns scan status, additional details, and error if status retrieval fails.
	GetScanStatus(ctx context.Context, documentID, versionID, tenantID string) (string, string, error)
}