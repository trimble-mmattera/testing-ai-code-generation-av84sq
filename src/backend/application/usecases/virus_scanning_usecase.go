// Package usecases implements application use cases for the Document Management Platform.
// This layer orchestrates the flow of data and business rules between the domain and infrastructure layers.
package usecases

import (
	"context" // standard library
	"fmt"     // standard library

	"../../domain/services"
	"../../pkg/errors"
	"../../pkg/logger"
)

// Error variables for validation failures
var (
	ErrInvalidDocumentID  = errors.NewValidationError("invalid document ID")
	ErrInvalidVersionID   = errors.NewValidationError("invalid version ID")
	ErrInvalidTenantID    = errors.NewValidationError("invalid tenant ID")
	ErrInvalidStoragePath = errors.NewValidationError("invalid storage path")
)

// Event type constants
const (
	DocumentEventScannedClean    = "document.scanned.clean"
	DocumentEventScannedInfected = "document.scanned.infected"
)

// VirusScanningUseCaseInterface defines the contract for virus scanning use cases.
type VirusScanningUseCaseInterface interface {
	// QueueDocumentForScanning queues a document for virus scanning.
	QueueDocumentForScanning(ctx context.Context, documentID, versionID, tenantID, storagePath string) error

	// ProcessScanQueue processes the virus scanning queue.
	// Returns the number of documents processed and error if processing fails.
	ProcessScanQueue(ctx context.Context, batchSize int) (int, error)

	// ScanDocument scans a document for viruses.
	// Returns true if document is clean, false if infected or error,
	// additional scan details (virus name if infected), and error if scanning fails.
	ScanDocument(ctx context.Context, documentID, versionID, tenantID, storagePath string) (bool, string, error)

	// ProcessScanResult processes the result of a virus scan.
	ProcessScanResult(ctx context.Context, documentID, versionID, tenantID, storagePath string, isClean bool, scanDetails string) error
}

// virusScanningUseCase implements the VirusScanningUseCaseInterface.
type virusScanningUseCase struct {
	virusScanningService services.VirusScanningService
	documentService      services.DocumentService
	eventService         services.EventServiceInterface
}

// NewVirusScanningUseCase creates a new VirusScanningUseCase instance with the provided dependencies.
func NewVirusScanningUseCase(
	virusScanningService services.VirusScanningService,
	documentService services.DocumentService,
	eventService services.EventServiceInterface,
) (VirusScanningUseCaseInterface, error) {
	// Validate that virusScanningService is not nil
	if virusScanningService == nil {
		return nil, errors.NewValidationError("virus scanning service cannot be nil")
	}
	
	// Validate that documentService is not nil
	if documentService == nil {
		return nil, errors.NewValidationError("document service cannot be nil")
	}
	
	// Validate that eventService is not nil
	if eventService == nil {
		return nil, errors.NewValidationError("event service cannot be nil")
	}

	// Create and return a new virusScanningUseCase with the provided dependencies
	return &virusScanningUseCase{
		virusScanningService: virusScanningService,
		documentService:      documentService,
		eventService:         eventService,
	}, nil
}

// QueueDocumentForScanning queues a document for virus scanning.
func (uc *virusScanningUseCase) QueueDocumentForScanning(
	ctx context.Context,
	documentID,
	versionID,
	tenantID,
	storagePath string,
) error {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate documentID is not empty, return ErrInvalidDocumentID if empty
	if documentID == "" {
		return ErrInvalidDocumentID
	}
	
	// Validate versionID is not empty, return ErrInvalidVersionID if empty
	if versionID == "" {
		return ErrInvalidVersionID
	}
	
	// Validate tenantID is not empty, return ErrInvalidTenantID if empty
	if tenantID == "" {
		return ErrInvalidTenantID
	}
	
	// Validate storagePath is not empty, return ErrInvalidStoragePath if empty
	if storagePath == "" {
		return ErrInvalidStoragePath
	}

	// Call virusScanningService.QueueForScanning with the provided parameters
	err := uc.virusScanningService.QueueForScanning(ctx, documentID, versionID, tenantID, storagePath)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to queue document for virus scanning",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID)
		return errors.Wrap(err, "failed to queue document for virus scanning")
	}

	// Log successful queueing
	log.Info("Document queued for virus scanning",
		"document_id", documentID,
		"version_id", versionID,
		"tenant_id", tenantID)

	return nil
}

// ProcessScanQueue processes the virus scanning queue.
func (uc *virusScanningUseCase) ProcessScanQueue(ctx context.Context, batchSize int) (int, error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Call virusScanningService.ProcessScanQueue with the provided batch size
	processed, err := uc.virusScanningService.ProcessScanQueue(ctx, batchSize)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to process virus scanning queue",
			"batch_size", batchSize)
		return 0, errors.Wrap(err, "failed to process virus scanning queue")
	}

	// Log number of documents processed
	log.Info("Processed documents from virus scanning queue",
		"processed_count", processed,
		"batch_size", batchSize)

	return processed, nil
}

// ScanDocument scans a document for viruses.
func (uc *virusScanningUseCase) ScanDocument(
	ctx context.Context,
	documentID,
	versionID,
	tenantID,
	storagePath string,
) (bool, string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate documentID is not empty, return ErrInvalidDocumentID if empty
	if documentID == "" {
		return false, "", ErrInvalidDocumentID
	}
	
	// Validate versionID is not empty, return ErrInvalidVersionID if empty
	if versionID == "" {
		return false, "", ErrInvalidVersionID
	}
	
	// Validate tenantID is not empty, return ErrInvalidTenantID if empty
	if tenantID == "" {
		return false, "", ErrInvalidTenantID
	}
	
	// Validate storagePath is not empty, return ErrInvalidStoragePath if empty
	if storagePath == "" {
		return false, "", ErrInvalidStoragePath
	}

	// Call virusScanningService.ScanDocument with the storage path
	scanResult, scanDetails, err := uc.virusScanningService.ScanDocument(ctx, storagePath)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to scan document for viruses",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID)
		return false, "", errors.Wrap(err, "failed to scan document for viruses")
	}

	// Determine if document is clean based on scan result
	isClean := scanResult == services.ScanResultClean

	// Log scan result with appropriate level based on outcome
	if isClean {
		log.Info("Document scan completed - no viruses detected",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID)
	} else {
		log.Error("Document scan completed - virus detected",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID,
			"scan_details", scanDetails)
	}

	return isClean, scanDetails, nil
}

// ProcessScanResult processes the result of a virus scan.
func (uc *virusScanningUseCase) ProcessScanResult(
	ctx context.Context,
	documentID,
	versionID,
	tenantID,
	storagePath string,
	isClean bool,
	scanDetails string,
) error {
	// Get logger with context
	log := logger.WithContext(ctx)

	// Validate documentID is not empty, return ErrInvalidDocumentID if empty
	if documentID == "" {
		return ErrInvalidDocumentID
	}
	
	// Validate versionID is not empty, return ErrInvalidVersionID if empty
	if versionID == "" {
		return ErrInvalidVersionID
	}
	
	// Validate tenantID is not empty, return ErrInvalidTenantID if empty
	if tenantID == "" {
		return ErrInvalidTenantID
	}
	
	// Validate storagePath is not empty, return ErrInvalidStoragePath if empty
	if storagePath == "" {
		return ErrInvalidStoragePath
	}

	// Call documentService.ProcessDocumentScanResult with scan results
	err := uc.documentService.ProcessDocumentScanResult(ctx, documentID, versionID, tenantID, isClean, scanDetails)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to process document scan result",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID,
			"is_clean", isClean)
		return errors.Wrap(err, "failed to process document scan result")
	}

	// Prepare event payload with document details and scan results
	payload := map[string]interface{}{
		"document_id":  documentID,
		"version_id":   versionID,
		"tenant_id":    tenantID,
		"scan_details": scanDetails,
	}

	// If document is clean:
	if isClean {
		// Publish DocumentEventScannedClean event
		eventType := DocumentEventScannedClean
		err = uc.eventService.PublishEvent(ctx, eventType, payload)
		if err != nil {
			logger.WithContext(ctx).WithError(err).Error("Failed to publish clean document scan event",
				"document_id", documentID,
				"version_id", versionID,
				"tenant_id", tenantID)
			return errors.Wrap(err, "failed to publish clean document scan event")
		}
		
		// Log clean document processing
		log.Info("Document processed as clean",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID)
	} else {
		// If document is infected:
		// Publish DocumentEventScannedInfected event
		eventType := DocumentEventScannedInfected
		err = uc.eventService.PublishEvent(ctx, eventType, payload)
		if err != nil {
			logger.WithContext(ctx).WithError(err).Error("Failed to publish infected document scan event",
				"document_id", documentID,
				"version_id", versionID,
				"tenant_id", tenantID)
			return errors.Wrap(err, "failed to publish infected document scan event")
		}
		
		// Log infected document processing
		log.Error("Document processed as infected",
			"document_id", documentID,
			"version_id", versionID,
			"tenant_id", tenantID,
			"scan_details", scanDetails)
	}

	return nil
}

// validateInput validates input parameters
func (uc *virusScanningUseCase) validateInput(params map[string]string) error {
	// Check each parameter in the map
	for key, value := range params {
		// If any parameter is empty, return appropriate validation error
		if value == "" {
			switch key {
			case "documentID":
				return ErrInvalidDocumentID
			case "versionID":
				return ErrInvalidVersionID
			case "tenantID":
				return ErrInvalidTenantID
			case "storagePath":
				return ErrInvalidStoragePath
			default:
				return errors.NewValidationError(fmt.Sprintf("invalid %s", key))
			}
		}
	}
	
	// Return nil if all parameters are valid
	return nil
}