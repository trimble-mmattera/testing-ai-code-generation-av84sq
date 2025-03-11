package clamav

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+

	"../../../domain/services"
	"../../../pkg/config"
	"../../../test/mockery"
)

// TestNewVirusScanner tests the creation of a new VirusScanner instance
func TestNewVirusScanner(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)

	// Assert expectations
	assert.NoError(t, err)
	assert.NotNil(t, scanner)
}

// TestNewVirusScanner_ValidationErrors tests validation errors when creating a new VirusScanner
func TestNewVirusScanner_ValidationErrors(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)
	
	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Test cases for validation errors
	testCases := []struct {
		name          string
		scannerClient services.ScannerClient
		scanQueue     services.ScanQueue
		storageService interface{}
		eventService  interface{}
	}{
		{
			name:          "Nil Scanner Client",
			scannerClient: nil,
			scanQueue:     mockScanQueue,
			storageService: mockStorageService,
			eventService:  mockEventService,
		},
		{
			name:          "Nil Scan Queue",
			scannerClient: mockScannerClient,
			scanQueue:     nil,
			storageService: mockStorageService,
			eventService:  mockEventService,
		},
		{
			name:          "Nil Storage Service",
			scannerClient: mockScannerClient,
			scanQueue:     mockScanQueue,
			storageService: nil,
			eventService:  mockEventService,
		},
		{
			name:          "Nil Event Service",
			scannerClient: mockScannerClient,
			scanQueue:     mockScanQueue,
			storageService: mockStorageService,
			eventService:  nil,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner, err := NewVirusScanner(tc.scannerClient, tc.scanQueue, tc.storageService, tc.eventService, testConfig)
			assert.Error(t, err)
			assert.Nil(t, scanner)
		})
	}
}

// TestVirusScanner_QueueForScanning tests queueing a document for virus scanning
func TestVirusScanner_QueueForScanning(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations
	mockScanQueue.On("Enqueue", mock.Anything, mock.MatchedBy(func(task services.ScanTask) bool {
		return task.DocumentID == "doc-123" &&
			   task.VersionID == "ver-123" &&
			   task.TenantID == "tenant-123" &&
			   task.StoragePath == "path/to/document" &&
			   task.RetryCount == 0
	})).Return(nil)

	// Call QueueForScanning
	err = scanner.QueueForScanning(context.Background(), "doc-123", "ver-123", "tenant-123", "path/to/document")
	
	// Assert expectations
	assert.NoError(t, err)
	mockScanQueue.AssertExpectations(t)
}

// TestVirusScanner_QueueForScanning_ValidationErrors tests validation errors when queueing a document
func TestVirusScanner_QueueForScanning_ValidationErrors(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Test cases for validation errors
	testCases := []struct {
		name        string
		documentID  string
		versionID   string
		tenantID    string
		storagePath string
	}{
		{
			name:        "Empty DocumentID",
			documentID:  "",
			versionID:   "ver-123",
			tenantID:    "tenant-123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty VersionID",
			documentID:  "doc-123",
			versionID:   "",
			tenantID:    "tenant-123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty TenantID",
			documentID:  "doc-123",
			versionID:   "ver-123",
			tenantID:    "",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty StoragePath",
			documentID:  "doc-123",
			versionID:   "ver-123",
			tenantID:    "tenant-123",
			storagePath: "",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = scanner.QueueForScanning(context.Background(), tc.documentID, tc.versionID, tc.tenantID, tc.storagePath)
			assert.Error(t, err)
		})
	}
}

// TestVirusScanner_QueueForScanning_QueueError tests handling of queue errors when queueing a document
func TestVirusScanner_QueueForScanning_QueueError(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for queue error
	mockScanQueue.On("Enqueue", mock.Anything, mock.Anything).Return(errors.New("queue error"))

	// Call QueueForScanning
	err = scanner.QueueForScanning(context.Background(), "doc-123", "ver-123", "tenant-123", "path/to/document")
	
	// Assert expectations
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue error")
	mockScanQueue.AssertExpectations(t)
}

// TestVirusScanner_ProcessScanQueue tests processing the scan queue
func TestVirusScanner_ProcessScanQueue(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Create test tasks
	task1 := &services.ScanTask{
		DocumentID:  "doc-1",
		VersionID:   "ver-1",
		TenantID:    "tenant-1",
		StoragePath: "path/to/doc-1",
		RetryCount:  0,
	}
	task2 := &services.ScanTask{
		DocumentID:  "doc-2",
		VersionID:   "ver-2",
		TenantID:    "tenant-2",
		StoragePath: "path/to/doc-2",
		RetryCount:  0,
	}

	// Set up expectations for dequeue
	mockScanQueue.On("Dequeue", mock.Anything).Return(task1, nil).Once()
	mockScanQueue.On("Dequeue", mock.Anything).Return(task2, nil).Once()
	mockScanQueue.On("Dequeue", mock.Anything).Return(nil, nil).Once() // End of queue

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, task1.StoragePath).Return(bytes.NewReader([]byte("test content 1")), nil)
	mockStorageService.On("GetDocument", mock.Anything, task2.StoragePath).Return(bytes.NewReader([]byte("test content 2")), nil)

	// Set up expectations for scanning
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return(services.ScanResultClean, "", nil).Twice()

	// Set up expectations for completing tasks
	mockScanQueue.On("Complete", mock.Anything, *task1).Return(nil)
	mockScanQueue.On("Complete", mock.Anything, *task2).Return(nil)

	// Set up expectations for publishing events
	mockEventService.On("CreateAndPublishDocumentEvent", mock.Anything, "document.scanned", mock.Anything).Return(nil).Twice()

	// Call ProcessScanQueue
	count, err := scanner.ProcessScanQueue(context.Background(), 10)
	
	// Assert expectations
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	mockScanQueue.AssertExpectations(t)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

// TestVirusScanner_ProcessScanQueue_EmptyQueue tests processing an empty scan queue
func TestVirusScanner_ProcessScanQueue_EmptyQueue(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for empty queue
	mockScanQueue.On("Dequeue", mock.Anything).Return(nil, nil)

	// Call ProcessScanQueue
	count, err := scanner.ProcessScanQueue(context.Background(), 10)
	
	// Assert expectations
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockScanQueue.AssertExpectations(t)
}

// TestVirusScanner_ProcessScanQueue_QueueError tests handling queue errors during processing
func TestVirusScanner_ProcessScanQueue_QueueError(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for queue error
	mockScanQueue.On("Dequeue", mock.Anything).Return(nil, errors.New("queue error"))

	// Call ProcessScanQueue
	count, err := scanner.ProcessScanQueue(context.Background(), 10)
	
	// Assert expectations
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue error")
	assert.Equal(t, 0, count)
	mockScanQueue.AssertExpectations(t)
}

// TestVirusScanner_ScanDocument_Clean tests scanning a clean document
func TestVirusScanner_ScanDocument_Clean(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, "path/to/document").Return(bytes.NewReader([]byte("test content")), nil)

	// Set up expectations for scanning
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return(services.ScanResultClean, "", nil)

	// Call ScanDocument
	result, details, err := scanner.ScanDocument(context.Background(), "path/to/document")
	
	// Assert expectations
	assert.NoError(t, err)
	assert.Equal(t, services.ScanResultClean, result)
	assert.Empty(t, details)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
}

// TestVirusScanner_ScanDocument_Infected tests scanning an infected document
func TestVirusScanner_ScanDocument_Infected(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, "path/to/document").Return(bytes.NewReader([]byte("infected content")), nil)

	// Set up expectations for scanning with virus detection
	virusDetails := "EICAR-Test-Signature"
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return(services.ScanResultInfected, virusDetails, nil)

	// Call ScanDocument
	result, details, err := scanner.ScanDocument(context.Background(), "path/to/document")
	
	// Assert expectations
	assert.NoError(t, err)
	assert.Equal(t, services.ScanResultInfected, result)
	assert.Equal(t, virusDetails, details)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
}

// TestVirusScanner_ScanDocument_Error tests handling errors during document scanning
func TestVirusScanner_ScanDocument_Error(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, "path/to/document").Return(bytes.NewReader([]byte("test content")), nil)

	// Set up expectations for scanning error
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return("", "", errors.New("scanning error"))

	// Call ScanDocument
	result, details, err := scanner.ScanDocument(context.Background(), "path/to/document")
	
	// Assert expectations
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scanning error")
	assert.Equal(t, services.ScanResultError, result)
	assert.Empty(t, details)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
}

// TestVirusScanner_ScanDocument_StorageError tests handling storage errors during document scanning
func TestVirusScanner_ScanDocument_StorageError(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Set up expectations for document content error
	mockStorageService.On("GetDocument", mock.Anything, "path/to/document").Return(nil, errors.New("storage error"))

	// Call ScanDocument
	result, details, err := scanner.ScanDocument(context.Background(), "path/to/document")
	
	// Assert expectations
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
	assert.Equal(t, services.ScanResultError, result)
	assert.Empty(t, details)
	mockStorageService.AssertExpectations(t)
}

// TestVirusScanner_MoveToQuarantine tests moving an infected document to quarantine
func TestVirusScanner_MoveToQuarantine(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Test parameters
	tenantID := "tenant-123"
	documentID := "doc-123"
	versionID := "ver-123"
	sourcePath := "path/to/document"
	quarantinePath := "quarantine/tenant-123/doc-123/ver-123"

	// Set up expectations for moving to quarantine
	mockStorageService.On("MoveToQuarantine", mock.Anything, tenantID, documentID, versionID, sourcePath).
		Return(quarantinePath, nil)

	// Call MoveToQuarantine
	resultPath, err := scanner.MoveToQuarantine(context.Background(), tenantID, documentID, versionID, sourcePath)
	
	// Assert expectations
	assert.NoError(t, err)
	assert.Equal(t, quarantinePath, resultPath)
	mockStorageService.AssertExpectations(t)
}

// TestVirusScanner_MoveToQuarantine_ValidationErrors tests validation errors when moving to quarantine
func TestVirusScanner_MoveToQuarantine_ValidationErrors(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Test cases for validation errors
	testCases := []struct {
		name       string
		tenantID   string
		documentID string
		versionID  string
		sourcePath string
	}{
		{
			name:       "Empty TenantID",
			tenantID:   "",
			documentID: "doc-123",
			versionID:  "ver-123",
			sourcePath: "path/to/document",
		},
		{
			name:       "Empty DocumentID",
			tenantID:   "tenant-123",
			documentID: "",
			versionID:  "ver-123",
			sourcePath: "path/to/document",
		},
		{
			name:       "Empty VersionID",
			tenantID:   "tenant-123",
			documentID: "doc-123",
			versionID:  "",
			sourcePath: "path/to/document",
		},
		{
			name:       "Empty SourcePath",
			tenantID:   "tenant-123",
			documentID: "doc-123",
			versionID:  "ver-123",
			sourcePath: "",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err = scanner.MoveToQuarantine(context.Background(), tc.tenantID, tc.documentID, tc.versionID, tc.sourcePath)
			assert.Error(t, err)
		})
	}
}

// TestVirusScanner_MoveToQuarantine_StorageError tests handling storage errors when moving to quarantine
func TestVirusScanner_MoveToQuarantine_StorageError(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Test parameters
	tenantID := "tenant-123"
	documentID := "doc-123"
	versionID := "ver-123"
	sourcePath := "path/to/document"

	// Set up expectations for storage error
	mockStorageService.On("MoveToQuarantine", mock.Anything, tenantID, documentID, versionID, sourcePath).
		Return("", errors.New("storage error"))

	// Call MoveToQuarantine
	_, err = scanner.MoveToQuarantine(context.Background(), tenantID, documentID, versionID, sourcePath)
	
	// Assert expectations
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
	mockStorageService.AssertExpectations(t)
}

// TestVirusScanner_GetScanStatus tests getting the scan status of a document
func TestVirusScanner_GetScanStatus(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Call GetScanStatus
	status, details, err := scanner.GetScanStatus(context.Background(), "doc-123", "ver-123", "tenant-123")
	
	// Assert expectations - assuming this is a placeholder method that returns a default value
	assert.NoError(t, err)
	assert.NotEmpty(t, status)
}

// TestVirusScanner_GetScanStatus_ValidationErrors tests validation errors when getting scan status
func TestVirusScanner_GetScanStatus_ValidationErrors(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Test cases for validation errors
	testCases := []struct {
		name       string
		documentID string
		versionID  string
		tenantID   string
	}{
		{
			name:       "Empty DocumentID",
			documentID: "",
			versionID:  "ver-123",
			tenantID:   "tenant-123",
		},
		{
			name:       "Empty VersionID",
			documentID: "doc-123",
			versionID:  "",
			tenantID:   "tenant-123",
		},
		{
			name:       "Empty TenantID",
			documentID: "doc-123",
			versionID:  "ver-123",
			tenantID:   "",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err = scanner.GetScanStatus(context.Background(), tc.documentID, tc.versionID, tc.tenantID)
			assert.Error(t, err)
		})
	}
}

// TestVirusScanner_processScanTask_Clean tests processing a scan task with a clean result
func TestVirusScanner_processScanTask_Clean(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Create a test task
	task := &services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, task.StoragePath).Return(bytes.NewReader([]byte("test content")), nil)

	// Set up expectations for scanning with clean result
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return(services.ScanResultClean, "", nil)

	// Set up expectations for task completion
	mockScanQueue.On("Complete", mock.Anything, *task).Return(nil)

	// Set up expectations for event publishing
	mockEventService.On("CreateAndPublishDocumentEvent", mock.Anything, "document.scanned", mock.Anything).Return(nil)

	// Call processScanTask - assuming it's exported for testing
	// If processScanTask is not exported, we would test this through ProcessScanQueue
	err = scanner.processScanTask(context.Background(), task)
	
	// Assert expectations
	assert.NoError(t, err)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
	mockScanQueue.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

// TestVirusScanner_processScanTask_Infected tests processing a scan task with an infected result
func TestVirusScanner_processScanTask_Infected(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Create a test task
	task := &services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  0,
	}

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, task.StoragePath).Return(bytes.NewReader([]byte("infected content")), nil)

	// Set up expectations for scanning with infected result
	virusDetails := "EICAR-Test-Signature"
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return(services.ScanResultInfected, virusDetails, nil)

	// Set up expectations for moving to quarantine
	mockStorageService.On("MoveToQuarantine", mock.Anything, task.TenantID, task.DocumentID, task.VersionID, task.StoragePath).
		Return("quarantine/path", nil)

	// Set up expectations for task completion
	mockScanQueue.On("Complete", mock.Anything, *task).Return(nil)

	// Set up expectations for event publishing
	mockEventService.On("CreateAndPublishDocumentEvent", mock.Anything, "document.quarantined", mock.Anything).Return(nil)

	// Call processScanTask
	err = scanner.processScanTask(context.Background(), task)
	
	// Assert expectations
	assert.NoError(t, err)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
	mockScanQueue.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

// TestVirusScanner_processScanTask_Error_Retry tests processing a scan task with an error that triggers retry
func TestVirusScanner_processScanTask_Error_Retry(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Create a test task with retry count less than max retries (assuming max is 3)
	task := &services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  1, // Less than max retries
	}

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, task.StoragePath).Return(bytes.NewReader([]byte("test content")), nil)

	// Set up expectations for scanning with error
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return("", "", errors.New("scan error"))

	// Set up expectations for task retry
	retryTask := *task
	retryTask.RetryCount++
	mockScanQueue.On("Retry", mock.Anything, retryTask).Return(nil)

	// Call processScanTask
	err = scanner.processScanTask(context.Background(), task)
	
	// Assert expectations
	assert.NoError(t, err)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
	mockScanQueue.AssertExpectations(t)
}

// TestVirusScanner_processScanTask_Error_DeadLetter tests processing a scan task with an error that exceeds max retries
func TestVirusScanner_processScanTask_Error_DeadLetter(t *testing.T) {
	// Create mock dependencies
	mockScannerClient := new(mockery.ScannerClient)
	mockScanQueue := new(mockery.ScanQueue)
	mockStorageService := new(mockery.StorageService)
	mockEventService := new(mockery.EventServiceInterface)

	// Create test configuration
	testConfig := &config.Config{
		ClamAV: config.ClamAVConfig{
			Host:    "localhost",
			Port:    3310,
			Timeout: 60,
		},
	}

	// Create a new VirusScanner
	scanner, err := NewVirusScanner(mockScannerClient, mockScanQueue, mockStorageService, mockEventService, testConfig)
	require.NoError(t, err)
	require.NotNil(t, scanner)

	// Create a test task with retry count equal to max retries (assuming max is 3)
	task := &services.ScanTask{
		DocumentID:  "doc-123",
		VersionID:   "ver-123",
		TenantID:    "tenant-123",
		StoragePath: "path/to/document",
		RetryCount:  3, // Equal to max retries
	}

	// Set up expectations for document content
	mockStorageService.On("GetDocument", mock.Anything, task.StoragePath).Return(bytes.NewReader([]byte("test content")), nil)

	// Set up expectations for scanning with error
	scanError := errors.New("scan error")
	mockScannerClient.On("ScanStream", mock.Anything, mock.Anything).Return("", "", scanError)

	// Set up expectations for dead letter
	mockScanQueue.On("DeadLetter", mock.Anything, *task, mock.Anything).Return(nil)

	// Set up expectations for event publishing
	mockEventService.On("CreateAndPublishDocumentEvent", mock.Anything, "document.scan_failed", mock.Anything).Return(nil)

	// Call processScanTask
	err = scanner.processScanTask(context.Background(), task)
	
	// Assert expectations
	assert.NoError(t, err)
	mockStorageService.AssertExpectations(t)
	mockScannerClient.AssertExpectations(t)
	mockScanQueue.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

// TestVirusScanner_validateInput tests the input validation helper function
func TestVirusScanner_validateInput(t *testing.T) {
	// Test cases for validation
	testCases := []struct {
		name      string
		input     map[string]string
		expectErr bool
	}{
		{
			name: "Valid Input",
			input: map[string]string{
				"documentID":  "doc-123",
				"versionID":   "ver-123",
				"tenantID":    "tenant-123",
				"storagePath": "path/to/document",
			},
			expectErr: false,
		},
		{
			name: "Empty DocumentID",
			input: map[string]string{
				"documentID":  "",
				"versionID":   "ver-123",
				"tenantID":    "tenant-123",
				"storagePath": "path/to/document",
			},
			expectErr: true,
		},
		{
			name: "Empty VersionID",
			input: map[string]string{
				"documentID":  "doc-123",
				"versionID":   "",
				"tenantID":    "tenant-123",
				"storagePath": "path/to/document",
			},
			expectErr: true,
		},
		{
			name: "Empty TenantID",
			input: map[string]string{
				"documentID":  "doc-123",
				"versionID":   "ver-123",
				"tenantID":    "",
				"storagePath": "path/to/document",
			},
			expectErr: true,
		},
		{
			name: "Empty StoragePath",
			input: map[string]string{
				"documentID":  "doc-123",
				"versionID":   "ver-123",
				"tenantID":    "tenant-123",
				"storagePath": "",
			},
			expectErr: true,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInput(
				tc.input["documentID"],
				tc.input["versionID"],
				tc.input["tenantID"],
				tc.input["storagePath"],
			)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}