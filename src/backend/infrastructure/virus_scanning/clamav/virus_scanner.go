// Package clamav provides an implementation of the VirusScanningService interface using ClamAV for virus detection.
// This service is responsible for scanning uploaded documents for malicious content, managing the scanning queue,
// and handling infected documents by moving them to quarantine storage.
package clamav

import (
	"context"
	"fmt"
	"sync"
	"time"

	"src/backend/domain/services"
	"src/backend/pkg/errors"
	"src/backend/pkg/logger"
	"src/backend/pkg/metrics"
	"src/backend/pkg/config"
)

// Maximum number of retry attempts for scan tasks
const maxRetries = 3

// Metric constants for virus scanning
const scannerMetricPrefix = "virus_scanner"
const documentScannedCounter = scannerMetricPrefix + "_documents_scanned_total"
const documentInfectedCounter = scannerMetricPrefix + "_documents_infected_total"
const documentCleanCounter = scannerMetricPrefix + "_documents_clean_total"
const scanErrorCounter = scannerMetricPrefix + "_scan_errors_total"
const scanDurationHistogram = scannerMetricPrefix + "_scan_duration_seconds"

// VirusScanner implements the VirusScanningService interface using ClamAV.
type VirusScanner struct {
	scannerClient   services.ScannerClient
	scanQueue       services.ScanQueue
	storageService  services.StorageService
	eventService    services.EventServiceInterface
	logger          *logger.Logger
	mutex           sync.Mutex
	isProcessing    bool
	config          config.Config
}

// NewVirusScanner creates a new VirusScanner instance that implements the VirusScanningService interface
func NewVirusScanner(scannerClient services.ScannerClient, scanQueue services.ScanQueue, 
                     storageService services.StorageService, eventService services.EventServiceInterface, 
                     cfg config.Config) (services.VirusScanningService, error) {
	// Validate that scannerClient is not nil
	if scannerClient == nil {
		return nil, errors.NewValidationError("scannerClient cannot be nil")
	}
	
	// Validate that scanQueue is not nil
	if scanQueue == nil {
		return nil, errors.NewValidationError("scanQueue cannot be nil")
	}
	
	// Validate that storageService is not nil
	if storageService == nil {
		return nil, errors.NewValidationError("storageService cannot be nil")
	}
	
	// Validate that eventService is not nil
	if eventService == nil {
		return nil, errors.NewValidationError("eventService cannot be nil")
	}
	
	// Create and return a new VirusScanner instance
	return &VirusScanner{
		scannerClient:  scannerClient,
		scanQueue:      scanQueue,
		storageService: storageService,
		eventService:   eventService,
		logger:         logger.WithField("service", "virus_scanner"),
		isProcessing:   false,
		config:         cfg,
	}, nil
}

// QueueForScanning queues a document for virus scanning
func (v *VirusScanner) QueueForScanning(ctx context.Context, documentID, versionID, tenantID, storagePath string) error {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Validate input parameters
	params := map[string]string{
		"documentID": documentID,
		"versionID":  versionID,
		"tenantID":   tenantID,
		"storagePath": storagePath,
	}
	if err := v.validateInput(params); err != nil {
		log.WithError(err).Error("Invalid input parameters for queuing document scan")
		return err
	}
	
	// Create a scan task
	task := services.ScanTask{
		DocumentID:  documentID,
		VersionID:   versionID,
		TenantID:    tenantID,
		StoragePath: storagePath,
		RetryCount:  0,
	}
	
	// Enqueue the task
	err := v.scanQueue.Enqueue(ctx, task)
	if err != nil {
		log.WithError(err).Error("Failed to enqueue document for scanning", 
			"documentID", documentID, 
			"tenantID", tenantID)
		return errors.Wrap(err, "failed to enqueue document for scanning")
	}
	
	// Log successful queueing
	log.Info("Document queued for virus scanning", 
		"documentID", documentID, 
		"tenantID", tenantID,
		"storagePath", storagePath)
	return nil
}

// ProcessScanQueue processes the virus scanning queue
func (v *VirusScanner) ProcessScanQueue(ctx context.Context, batchSize int) (int, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Acquire mutex to prevent concurrent processing
	v.mutex.Lock()
	
	// Check if already processing, return 0 if true
	if v.isProcessing {
		v.mutex.Unlock()
		log.Info("Scan queue is already being processed")
		return 0, nil
	}
	
	// Set isProcessing to true
	v.isProcessing = true
	
	// Defer setting isProcessing to false and releasing mutex
	defer func() {
		v.isProcessing = false
		v.mutex.Unlock()
	}()
	
	log.Info("Starting to process virus scan queue", "batchSize", batchSize)
	
	// Initialize counter for processed documents
	processed := 0
	
	// Loop for batchSize iterations or until queue is empty
	for i := 0; i < batchSize; i++ {
		// Check for context cancellation
		if ctx.Err() != nil {
			log.Warn("Context cancelled, stopping queue processing", "processed", processed)
			return processed, ctx.Err()
		}
		
		// Dequeue a task from the scan queue
		task, err := v.scanQueue.Dequeue(ctx)
		if err != nil {
			log.WithError(err).Error("Failed to dequeue scan task")
			return processed, errors.Wrap(err, "failed to dequeue scan task")
		}
		
		// If no task, break the loop
		if task == nil {
			log.Info("No more tasks in queue, stopping processing", "processed", processed)
			break
		}
		
		// Process the task using processScanTask
		err = v.processScanTask(ctx, *task)
		if err != nil {
			log.WithError(err).Error("Failed to process scan task", 
				"documentID", task.DocumentID, 
				"tenantID", task.TenantID)
			// Continue processing other tasks despite error
		}
		
		// Increment processed counter
		processed++
	}
	
	log.Info("Completed processing virus scan queue", "processed", processed)
	return processed, nil
}

// ScanDocument scans a document for viruses
func (v *VirusScanner) ScanDocument(ctx context.Context, storagePath string) (string, string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Validate storage path
	if storagePath == "" {
		err := errors.NewValidationError("storage path is required")
		log.WithError(err).Error("Invalid storage path for scanning")
		return services.ScanResultError, "", err
	}
	
	log.Info("Scanning document for viruses", "storagePath", storagePath)
	
	// Start timer for scan duration metrics
	startTime := time.Now()
	
	// Get document content from storage service
	content, err := v.storageService.GetDocumentContent(ctx, storagePath)
	if err != nil {
		log.WithError(err).Error("Failed to get document content", "storagePath", storagePath)
		return services.ScanResultError, "", errors.Wrap(err, "failed to get document content")
	}
	defer content.Close()
	
	// Call scannerClient.ScanStream to scan the document
	result, details, err := v.scannerClient.ScanStream(ctx, content)
	
	// Record scan duration metric
	scanDuration := time.Since(startTime)
	metrics.ObserveHistogram(scanDurationHistogram, scanDuration.Seconds())
	
	// Increment appropriate counter based on scan result
	metrics.IncrementCounter(documentScannedCounter, 1)
	
	if err != nil {
		log.WithError(err).Error("Error scanning document", "storagePath", storagePath)
		metrics.IncrementCounter(scanErrorCounter, 1)
		return services.ScanResultError, fmt.Sprintf("scan error: %s", err.Error()), errors.Wrap(err, "failed to scan document")
	} else if result == services.ScanResultInfected {
		log.Warn("Virus detected in document", "storagePath", storagePath, "virusDetails", details)
		metrics.IncrementCounter(documentInfectedCounter, 1)
	} else {
		log.Info("Document scan completed successfully", "storagePath", storagePath, "result", result)
		metrics.IncrementCounter(documentCleanCounter, 1)
	}
	
	return result, details, nil
}

// MoveToQuarantine moves an infected document to quarantine storage
func (v *VirusScanner) MoveToQuarantine(ctx context.Context, tenantID, documentID, versionID, sourcePath string) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Validate input parameters
	params := map[string]string{
		"tenantID":   tenantID,
		"documentID": documentID,
		"versionID":  versionID,
		"sourcePath": sourcePath,
	}
	if err := v.validateInput(params); err != nil {
		log.WithError(err).Error("Invalid input parameters for quarantine operation")
		return "", err
	}
	
	log.Info("Moving infected document to quarantine", 
		"documentID", documentID, 
		"tenantID", tenantID,
		"sourcePath", sourcePath)
	
	// Call storageService.MoveToQuarantine to move the document
	quarantinePath, err := v.storageService.MoveToQuarantine(ctx, tenantID, documentID, versionID, sourcePath)
	if err != nil {
		log.WithError(err).Error("Failed to move document to quarantine", 
			"documentID", documentID, 
			"tenantID", tenantID)
		return "", errors.Wrap(err, "failed to move document to quarantine")
	}
	
	log.Info("Document moved to quarantine successfully", 
		"documentID", documentID, 
		"tenantID", tenantID,
		"quarantinePath", quarantinePath)
	
	return quarantinePath, nil
}

// GetScanStatus gets the current scan status of a document
func (v *VirusScanner) GetScanStatus(ctx context.Context, documentID, versionID, tenantID string) (string, string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Validate input parameters
	params := map[string]string{
		"documentID": documentID,
		"versionID":  versionID,
		"tenantID":   tenantID,
	}
	if err := v.validateInput(params); err != nil {
		log.WithError(err).Error("Invalid input parameters for scan status")
		return "", "", err
	}
	
	// This is a placeholder for future implementation
	// Currently returns 'unknown' status as scan results are not persisted
	log.Info("Getting scan status for document", 
		"documentID", documentID, 
		"tenantID", tenantID)
	
	return "unknown", "", nil
}

// processScanTask is an internal method to process a single scan task
func (v *VirusScanner) processScanTask(ctx context.Context, task services.ScanTask) error {
	// Get logger with context and task details
	log := logger.WithContext(ctx).
		WithField("documentID", task.DocumentID).
		WithField("tenantID", task.TenantID).
		WithField("retryCount", task.RetryCount)
	
	log.Info("Processing scan task")
	
	// Call ScanDocument to scan the document
	result, details, err := v.ScanDocument(ctx, task.StoragePath)
	
	// Handle scan result based on outcome
	if err != nil {
		// Check retry count against maxRetries
		if task.RetryCount < maxRetries {
			// Increment retry count and requeue task
			task.RetryCount++
			log.WithError(err).Warn("Scan failed, retrying", "retry", task.RetryCount)
			
			if retryErr := v.scanQueue.Retry(ctx, task); retryErr != nil {
				log.WithError(retryErr).Error("Failed to requeue scan task")
				return errors.Wrap(retryErr, "failed to requeue scan task after error")
			}
			return nil
		}
		
		// Max retries exceeded, move task to dead letter queue
		log.WithError(err).Error("Max retries exceeded for scan task")
		
		dlqErr := v.scanQueue.DeadLetter(ctx, task, fmt.Sprintf("Max retries exceeded: %s", err.Error()))
		if dlqErr != nil {
			log.WithError(dlqErr).Error("Failed to move scan task to dead letter queue")
			return errors.Wrap(dlqErr, "failed to move scan task to dead letter queue")
		}
		
		// Publish document.scan_failed event with error details
		_, pubErr := v.eventService.CreateAndPublishDocumentEvent(ctx, "document.scan_failed", 
			task.TenantID, task.DocumentID, map[string]interface{}{
				"error": err.Error(),
				"scanAttempts": task.RetryCount,
			})
		
		if pubErr != nil {
			log.WithError(pubErr).Error("Failed to publish scan failed event")
		}
		
		return nil
	}
	
	// Handle successful scan result
	if result == services.ScanResultClean {
		log.Info("Document scan clean, marking as complete")
		
		// Publish document.scanned event with clean status
		_, pubErr := v.eventService.CreateAndPublishDocumentEvent(ctx, "document.scanned", 
			task.TenantID, task.DocumentID, map[string]interface{}{
				"status": "clean",
			})
		
		if pubErr != nil {
			log.WithError(pubErr).Error("Failed to publish document scanned event")
		}
		
		// Mark task as complete in queue
		if completeErr := v.scanQueue.Complete(ctx, task); completeErr != nil {
			log.WithError(completeErr).Error("Failed to mark scan task as complete")
			return errors.Wrap(completeErr, "failed to mark scan task as complete")
		}
		
		log.Info("Document scan task completed successfully", "result", "clean")
		
	} else if result == services.ScanResultInfected {
		log.Warn("Document scan detected infection, quarantining", "virusDetails", details)
		
		// Move document to quarantine
		quarantinePath, quarErr := v.MoveToQuarantine(ctx, task.TenantID, task.DocumentID, task.VersionID, task.StoragePath)
		if quarErr != nil {
			log.WithError(quarErr).Error("Failed to move infected document to quarantine")
			return errors.Wrap(quarErr, "failed to move infected document to quarantine")
		}
		
		// Publish document.quarantined event with virus details
		_, pubErr := v.eventService.CreateAndPublishDocumentEvent(ctx, "document.quarantined", 
			task.TenantID, task.DocumentID, map[string]interface{}{
				"reason": details,
				"quarantinePath": quarantinePath,
			})
		
		if pubErr != nil {
			log.WithError(pubErr).Error("Failed to publish document quarantined event")
		}
		
		// Mark task as complete in queue
		if completeErr := v.scanQueue.Complete(ctx, task); completeErr != nil {
			log.WithError(completeErr).Error("Failed to mark scan task as complete")
			return errors.Wrap(completeErr, "failed to mark scan task as complete")
		}
		
		log.Info("Infected document quarantined successfully")
	}
	
	return nil
}

// validateInput validates input parameters
func (v *VirusScanner) validateInput(params map[string]string) error {
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