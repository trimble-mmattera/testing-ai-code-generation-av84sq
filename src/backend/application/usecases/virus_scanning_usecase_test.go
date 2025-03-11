package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock"   // v1.8.0+

	"../../domain/services"
	apperrors "../../pkg/errors"
	"../usecases"
)

// MockVirusScanningService is a mock implementation of the VirusScanningService interface
type MockVirusScanningService struct {
	mock.Mock
}

func (m *MockVirusScanningService) QueueForScanning(ctx context.Context, documentID, versionID, tenantID, storagePath string) error {
	args := m.Called(ctx, documentID, versionID, tenantID, storagePath)
	return args.Error(0)
}

func (m *MockVirusScanningService) ProcessScanQueue(ctx context.Context, batchSize int) (int, error) {
	args := m.Called(ctx, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockVirusScanningService) ScanDocument(ctx context.Context, storagePath string) (string, string, error) {
	args := m.Called(ctx, storagePath)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockVirusScanningService) MoveToQuarantine(ctx context.Context, tenantID, documentID, versionID, sourcePath string) (string, error) {
	args := m.Called(ctx, tenantID, documentID, versionID, sourcePath)
	return args.String(0), args.Error(1)
}

func (m *MockVirusScanningService) GetScanStatus(ctx context.Context, documentID, versionID, tenantID string) (string, string, error) {
	args := m.Called(ctx, documentID, versionID, tenantID)
	return args.String(0), args.String(1), args.Error(2)
}

// MockDocumentService is a mock implementation of the DocumentService interface
type MockDocumentService struct {
	mock.Mock
}

func (m *MockDocumentService) ProcessDocumentScanResult(ctx context.Context, documentID, versionID, tenantID string, isClean bool, scanDetails string) error {
	args := m.Called(ctx, documentID, versionID, tenantID, isClean, scanDetails)
	return args.Error(0)
}

// MockEventService is a mock implementation of the EventServiceInterface interface
type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) PublishEvent(ctx context.Context, eventType string, eventData map[string]interface{}) error {
	args := m.Called(ctx, eventType, eventData)
	return args.Error(0)
}

// TestNewVirusScanningUseCase tests the creation of a new VirusScanningUseCase instance
func TestNewVirusScanningUseCase(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	// Act
	useCase, err := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, useCase)
}

// TestNewVirusScanningUseCase_NilServices tests the creation of a new VirusScanningUseCase with nil services
func TestNewVirusScanningUseCase_NilServices(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	testCases := []struct {
		name                 string
		virusScanningService services.VirusScanningService
		documentService      services.DocumentService
		eventService         services.EventServiceInterface
	}{
		{
			name:                 "Nil VirusScanningService",
			virusScanningService: nil,
			documentService:      mockDocumentService,
			eventService:         mockEventService,
		},
		{
			name:                 "Nil DocumentService",
			virusScanningService: mockVirusScanningService,
			documentService:      nil,
			eventService:         mockEventService,
		},
		{
			name:                 "Nil EventService",
			virusScanningService: mockVirusScanningService,
			documentService:      mockDocumentService,
			eventService:         nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			useCase, err := usecases.NewVirusScanningUseCase(tc.virusScanningService, tc.documentService, tc.eventService)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, useCase)
		})
	}
}

// TestVirusScanningUseCase_QueueDocumentForScanning tests the QueueDocumentForScanning method
func TestVirusScanningUseCase_QueueDocumentForScanning(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"

	ctx := context.Background()
	mockVirusScanningService.On("QueueForScanning", ctx, documentID, versionID, tenantID, storagePath).Return(nil)

	// Act
	err := useCase.QueueDocumentForScanning(ctx, documentID, versionID, tenantID, storagePath)

	// Assert
	assert.NoError(t, err)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_QueueDocumentForScanning_ValidationErrors tests validation errors in QueueDocumentForScanning method
func TestVirusScanningUseCase_QueueDocumentForScanning_ValidationErrors(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	ctx := context.Background()

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
			versionID:   "ver123",
			tenantID:    "tenant123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty VersionID",
			documentID:  "doc123",
			versionID:   "",
			tenantID:    "tenant123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty TenantID",
			documentID:  "doc123",
			versionID:   "ver123",
			tenantID:    "",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty StoragePath",
			documentID:  "doc123",
			versionID:   "ver123",
			tenantID:    "tenant123",
			storagePath: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := useCase.QueueDocumentForScanning(ctx, tc.documentID, tc.versionID, tc.tenantID, tc.storagePath)

			// Assert
			assert.Error(t, err)
			var validationErr *apperrors.AppError
			assert.True(t, errors.As(err, &validationErr))
		})
	}

	// Verify no calls were made to the service
	mockVirusScanningService.AssertNotCalled(t, "QueueForScanning")
}

// TestVirusScanningUseCase_QueueDocumentForScanning_ServiceError tests service errors in QueueDocumentForScanning method
func TestVirusScanningUseCase_QueueDocumentForScanning_ServiceError(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	serviceError := errors.New("queue error")

	ctx := context.Background()
	mockVirusScanningService.On("QueueForScanning", ctx, documentID, versionID, tenantID, storagePath).Return(serviceError)

	// Act
	err := useCase.QueueDocumentForScanning(ctx, documentID, versionID, tenantID, storagePath)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, serviceError, err)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ProcessScanQueue tests the ProcessScanQueue method
func TestVirusScanningUseCase_ProcessScanQueue(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	batchSize := 10
	expectedCount := 5

	ctx := context.Background()
	mockVirusScanningService.On("ProcessScanQueue", ctx, batchSize).Return(expectedCount, nil)

	// Act
	count, err := useCase.ProcessScanQueue(ctx, batchSize)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ProcessScanQueue_ServiceError tests service errors in ProcessScanQueue method
func TestVirusScanningUseCase_ProcessScanQueue_ServiceError(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	batchSize := 10
	serviceError := errors.New("processing error")

	ctx := context.Background()
	mockVirusScanningService.On("ProcessScanQueue", ctx, batchSize).Return(0, serviceError)

	// Act
	count, err := useCase.ProcessScanQueue(ctx, batchSize)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, serviceError, err)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ScanDocument tests the ScanDocument method for clean documents
func TestVirusScanningUseCase_ScanDocument(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	scanDetails := "Clean document"

	ctx := context.Background()
	mockVirusScanningService.On("ScanDocument", ctx, storagePath).Return(services.ScanResultClean, scanDetails, nil)

	// Act
	isClean, details, err := useCase.ScanDocument(ctx, documentID, versionID, tenantID, storagePath)

	// Assert
	assert.NoError(t, err)
	assert.True(t, isClean)
	assert.Equal(t, scanDetails, details)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ScanDocument_Infected tests the ScanDocument method for infected documents
func TestVirusScanningUseCase_ScanDocument_Infected(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	scanDetails := "Found virus: EICAR-Test-File"

	ctx := context.Background()
	mockVirusScanningService.On("ScanDocument", ctx, storagePath).Return(services.ScanResultInfected, scanDetails, nil)

	// Act
	isClean, details, err := useCase.ScanDocument(ctx, documentID, versionID, tenantID, storagePath)

	// Assert
	assert.NoError(t, err)
	assert.False(t, isClean)
	assert.Equal(t, scanDetails, details)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ScanDocument_ValidationErrors tests validation errors in ScanDocument method
func TestVirusScanningUseCase_ScanDocument_ValidationErrors(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	ctx := context.Background()

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
			versionID:   "ver123",
			tenantID:    "tenant123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty VersionID",
			documentID:  "doc123",
			versionID:   "",
			tenantID:    "tenant123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty TenantID",
			documentID:  "doc123",
			versionID:   "ver123",
			tenantID:    "",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty StoragePath",
			documentID:  "doc123",
			versionID:   "ver123",
			tenantID:    "tenant123",
			storagePath: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			isClean, details, err := useCase.ScanDocument(ctx, tc.documentID, tc.versionID, tc.tenantID, tc.storagePath)

			// Assert
			assert.Error(t, err)
			assert.False(t, isClean)
			assert.Empty(t, details)
			var validationErr *apperrors.AppError
			assert.True(t, errors.As(err, &validationErr))
		})
	}

	// Verify no calls were made to the service
	mockVirusScanningService.AssertNotCalled(t, "ScanDocument")
}

// TestVirusScanningUseCase_ScanDocument_ServiceError tests service errors in ScanDocument method
func TestVirusScanningUseCase_ScanDocument_ServiceError(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	serviceError := errors.New("scanning error")

	ctx := context.Background()
	mockVirusScanningService.On("ScanDocument", ctx, storagePath).Return("", "", serviceError)

	// Act
	isClean, details, err := useCase.ScanDocument(ctx, documentID, versionID, tenantID, storagePath)

	// Assert
	assert.Error(t, err)
	assert.False(t, isClean)
	assert.Empty(t, details)
	assert.Equal(t, serviceError, err)
	mockVirusScanningService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ProcessScanResult_Clean tests the ProcessScanResult method for clean documents
func TestVirusScanningUseCase_ProcessScanResult_Clean(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	scanDetails := "Clean document"
	isClean := true

	ctx := context.Background()
	mockDocumentService.On("ProcessDocumentScanResult", ctx, documentID, versionID, tenantID, isClean, scanDetails).Return(nil)
	mockEventService.On("PublishEvent", ctx, "document.scanned.clean", mock.Anything).Return(nil)

	// Act
	err := useCase.ProcessScanResult(ctx, documentID, versionID, tenantID, storagePath, isClean, scanDetails)

	// Assert
	assert.NoError(t, err)
	mockDocumentService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ProcessScanResult_Infected tests the ProcessScanResult method for infected documents
func TestVirusScanningUseCase_ProcessScanResult_Infected(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	scanDetails := "Found virus: EICAR-Test-File"
	isClean := false

	ctx := context.Background()
	mockDocumentService.On("ProcessDocumentScanResult", ctx, documentID, versionID, tenantID, isClean, scanDetails).Return(nil)
	mockEventService.On("PublishEvent", ctx, "document.scanned.infected", mock.Anything).Return(nil)

	// Act
	err := useCase.ProcessScanResult(ctx, documentID, versionID, tenantID, storagePath, isClean, scanDetails)

	// Assert
	assert.NoError(t, err)
	mockDocumentService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}

// TestVirusScanningUseCase_ProcessScanResult_ValidationErrors tests validation errors in ProcessScanResult method
func TestVirusScanningUseCase_ProcessScanResult_ValidationErrors(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	ctx := context.Background()
	isClean := true
	scanDetails := "Clean document"

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
			versionID:   "ver123",
			tenantID:    "tenant123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty VersionID",
			documentID:  "doc123",
			versionID:   "",
			tenantID:    "tenant123",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty TenantID",
			documentID:  "doc123",
			versionID:   "ver123",
			tenantID:    "",
			storagePath: "path/to/document",
		},
		{
			name:        "Empty StoragePath",
			documentID:  "doc123",
			versionID:   "ver123",
			tenantID:    "tenant123",
			storagePath: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := useCase.ProcessScanResult(ctx, tc.documentID, tc.versionID, tc.tenantID, tc.storagePath, isClean, scanDetails)

			// Assert
			assert.Error(t, err)
			var validationErr *apperrors.AppError
			assert.True(t, errors.As(err, &validationErr))
		})
	}

	// Verify no calls were made to the services
	mockDocumentService.AssertNotCalled(t, "ProcessDocumentScanResult")
	mockEventService.AssertNotCalled(t, "PublishEvent")
}

// TestVirusScanningUseCase_ProcessScanResult_DocumentServiceError tests document service errors in ProcessScanResult method
func TestVirusScanningUseCase_ProcessScanResult_DocumentServiceError(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	scanDetails := "Clean document"
	isClean := true
	serviceError := errors.New("processing error")

	ctx := context.Background()
	mockDocumentService.On("ProcessDocumentScanResult", ctx, documentID, versionID, tenantID, isClean, scanDetails).Return(serviceError)

	// Act
	err := useCase.ProcessScanResult(ctx, documentID, versionID, tenantID, storagePath, isClean, scanDetails)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, serviceError, err)
	mockDocumentService.AssertExpectations(t)
	mockEventService.AssertNotCalled(t, "PublishEvent")
}

// TestVirusScanningUseCase_ProcessScanResult_EventServiceError tests event service errors in ProcessScanResult method
func TestVirusScanningUseCase_ProcessScanResult_EventServiceError(t *testing.T) {
	// Arrange
	mockVirusScanningService := new(MockVirusScanningService)
	mockDocumentService := new(MockDocumentService)
	mockEventService := new(MockEventService)

	useCase, _ := usecases.NewVirusScanningUseCase(mockVirusScanningService, mockDocumentService, mockEventService)

	documentID := "doc123"
	versionID := "ver123"
	tenantID := "tenant123"
	storagePath := "path/to/document"
	scanDetails := "Clean document"
	isClean := true
	serviceError := errors.New("event publishing error")

	ctx := context.Background()
	mockDocumentService.On("ProcessDocumentScanResult", ctx, documentID, versionID, tenantID, isClean, scanDetails).Return(nil)
	mockEventService.On("PublishEvent", ctx, "document.scanned.clean", mock.Anything).Return(serviceError)

	// Act
	err := useCase.ProcessScanResult(ctx, documentID, versionID, tenantID, storagePath, isClean, scanDetails)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, serviceError, err)
	mockDocumentService.AssertExpectations(t)
	mockEventService.AssertExpectations(t)
}