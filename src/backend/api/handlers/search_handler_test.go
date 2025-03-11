package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"../dto"
	"../../domain/models"
	"../../pkg/errors"
	"../../pkg/utils/pagination"
)

// Mock implementation of SearchUseCase
type MockSearchUseCase struct {
	mock.Mock
}

func (m *MockSearchUseCase) SearchByContent(ctx context.Context, query string, tenantID string, pagination *pagination.Pagination) (pagination.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, query, tenantID, pagination)
	return args.Get(0).(pagination.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchUseCase) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *pagination.Pagination) (pagination.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, metadata, tenantID, pagination)
	return args.Get(0).(pagination.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchUseCase) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *pagination.Pagination) (pagination.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, contentQuery, metadata, tenantID, pagination)
	return args.Get(0).(pagination.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchUseCase) SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *pagination.Pagination) (pagination.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, folderID, query, tenantID, pagination)
	return args.Get(0).(pagination.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchUseCase) IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error {
	args := m.Called(ctx, documentID, tenantID, content)
	return args.Error(0)
}

func (m *MockSearchUseCase) RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error {
	args := m.Called(ctx, documentID, tenantID)
	return args.Error(0)
}

// Test helper functions
func setupTest() (*MockSearchUseCase, *gin.Engine, *SearchHandler) {
	mockUseCase := new(MockSearchUseCase)
	handler := NewSearchHandler(mockUseCase)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	router.POST("/api/v1/search/content", handler.SearchByContent)
	router.POST("/api/v1/search/metadata", handler.SearchByMetadata)
	router.POST("/api/v1/search/combined", handler.CombinedSearch)
	router.POST("/api/v1/search/folder", handler.SearchInFolder)
	
	return mockUseCase, router, handler
}

func createTestDocument(id string) models.Document {
	now := time.Now()
	return models.Document{
		ID:          id,
		Name:        "Test Document",
		ContentType: "application/pdf",
		Size:        1024,
		FolderID:    "folder-123",
		TenantID:    "tenant-123",
		OwnerID:     "user-123",
		Status:      "available",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func createTestDocuments(count int) []models.Document {
	docs := make([]models.Document, count)
	for i := 0; i < count; i++ {
		docs[i] = createTestDocument(fmt.Sprintf("doc-%d", i+1))
	}
	return docs
}

func createTestContext(router *gin.Engine) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	return c, w
}

// Unit tests
func TestNewSearchHandler(t *testing.T) {
	// Test with valid use case
	mockUseCase := new(MockSearchUseCase)
	handler := NewSearchHandler(mockUseCase)
	assert.NotNil(t, handler)
	
	// Test with nil use case (should panic)
	assert.Panics(t, func() {
		NewSearchHandler(nil)
	})
}

func TestSearchHandler_SearchByContent(t *testing.T) {
	mockUseCase, _, handler := setupTest()
	
	// Success case
	contentReq := dto.ContentSearchRequest{
		Query:    "test",
		Page:     1,
		PageSize: 10,
	}
	
	testDocs := []models.Document{
		createTestDocument("doc-1"),
		createTestDocument("doc-2"),
	}
	
	// Create expected result
	expectedResult := pagination.PaginatedResult[models.Document]{
		Items: testDocs,
		Pagination: pagination.PageInfo{
			Page:        contentReq.Page,
			PageSize:    contentReq.PageSize,
			TotalPages:  1,
			TotalItems:  int64(len(testDocs)),
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	// Set up mock expectations
	mockUseCase.On("SearchByContent", mock.Anything, contentReq.Query, "tenant-123", mock.Anything).
		Return(expectedResult, nil)
	
	// Create request
	body, _ := json.Marshal(contentReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/search/content", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	// Set up response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("tenant_id", "tenant-123")
	
	// Call handler
	handler.SearchByContent(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response dto.DocumentSearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Equal(t, len(testDocs), len(response.Results))
	assert.Equal(t, contentReq.Page, response.Pagination.Page)
	assert.Equal(t, contentReq.PageSize, response.Pagination.PageSize)
	
	// Validation error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	invalidReq := dto.ContentSearchRequest{
		Query:    "", // Empty query should fail validation
		Page:     1,
		PageSize: 10,
	}
	
	body, _ = json.Marshal(invalidReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/content", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchByContent(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Authorization error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	authErrorReq := dto.ContentSearchRequest{
		Query:    "unauthorized",
		Page:     1,
		PageSize: 10,
	}
	
	mockUseCase.On("SearchByContent", mock.Anything, authErrorReq.Query, "tenant-123", mock.Anything).
		Return(pagination.PaginatedResult[models.Document]{}, errors.NewAuthorizationError("unauthorized access"))
	
	body, _ = json.Marshal(authErrorReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/content", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchByContent(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	// Internal error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	internalErrorReq := dto.ContentSearchRequest{
		Query:    "error",
		Page:     1,
		PageSize: 10,
	}
	
	mockUseCase.On("SearchByContent", mock.Anything, internalErrorReq.Query, "tenant-123", mock.Anything).
		Return(pagination.PaginatedResult[models.Document]{}, errors.NewInternalError("internal error"))
	
	body, _ = json.Marshal(internalErrorReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/content", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchByContent(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	mockUseCase.AssertExpectations(t)
}

func TestSearchHandler_SearchByMetadata(t *testing.T) {
	mockUseCase, _, handler := setupTest()
	
	// Success case
	metadataReq := dto.MetadataSearchRequest{
		Metadata: map[string]string{
			"author": "John Doe",
			"department": "Engineering",
		},
		Page:     1,
		PageSize: 10,
	}
	
	testDocs := []models.Document{
		createTestDocument("doc-1"),
		createTestDocument("doc-2"),
	}
	
	// Create expected result
	expectedResult := pagination.PaginatedResult[models.Document]{
		Items: testDocs,
		Pagination: pagination.PageInfo{
			Page:        metadataReq.Page,
			PageSize:    metadataReq.PageSize,
			TotalPages:  1,
			TotalItems:  int64(len(testDocs)),
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	// Set up mock expectations
	mockUseCase.On("SearchByMetadata", mock.Anything, metadataReq.Metadata, "tenant-123", mock.Anything).
		Return(expectedResult, nil)
	
	// Create request
	body, _ := json.Marshal(metadataReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/search/metadata", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	// Set up response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("tenant_id", "tenant-123")
	
	// Call handler
	handler.SearchByMetadata(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response dto.DocumentSearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Equal(t, len(testDocs), len(response.Results))
	assert.Equal(t, metadataReq.Page, response.Pagination.Page)
	assert.Equal(t, metadataReq.PageSize, response.Pagination.PageSize)
	
	// Validation error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	invalidReq := dto.MetadataSearchRequest{
		Metadata: map[string]string{}, // Empty metadata should fail validation
		Page:     1,
		PageSize: 10,
	}
	
	body, _ = json.Marshal(invalidReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/metadata", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchByMetadata(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Authorization error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	authErrorReq := dto.MetadataSearchRequest{
		Metadata: map[string]string{"restricted": "true"},
		Page:     1,
		PageSize: 10,
	}
	
	mockUseCase.On("SearchByMetadata", mock.Anything, authErrorReq.Metadata, "tenant-123", mock.Anything).
		Return(pagination.PaginatedResult[models.Document]{}, errors.NewAuthorizationError("unauthorized access"))
	
	body, _ = json.Marshal(authErrorReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/metadata", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchByMetadata(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	mockUseCase.AssertExpectations(t)
}

func TestSearchHandler_CombinedSearch(t *testing.T) {
	mockUseCase, _, handler := setupTest()
	
	// Success case
	combinedReq := dto.CombinedSearchRequest{
		Query: "test",
		Metadata: map[string]string{
			"author": "John Doe",
		},
		Page:     1,
		PageSize: 10,
	}
	
	testDocs := []models.Document{
		createTestDocument("doc-1"),
		createTestDocument("doc-2"),
	}
	
	// Create expected result
	expectedResult := pagination.PaginatedResult[models.Document]{
		Items: testDocs,
		Pagination: pagination.PageInfo{
			Page:        combinedReq.Page,
			PageSize:    combinedReq.PageSize,
			TotalPages:  1,
			TotalItems:  int64(len(testDocs)),
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	// Set up mock expectations
	mockUseCase.On("CombinedSearch", mock.Anything, combinedReq.Query, combinedReq.Metadata, "tenant-123", mock.Anything).
		Return(expectedResult, nil)
	
	// Create request
	body, _ := json.Marshal(combinedReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/search/combined", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	// Set up response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("tenant_id", "tenant-123")
	
	// Call handler
	handler.CombinedSearch(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response dto.DocumentSearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Equal(t, len(testDocs), len(response.Results))
	assert.Equal(t, combinedReq.Page, response.Pagination.Page)
	assert.Equal(t, combinedReq.PageSize, response.Pagination.PageSize)
	
	// Validation error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	invalidReq := dto.CombinedSearchRequest{
		Query:    "", // Both empty query and metadata should fail validation
		Metadata: map[string]string{},
		Page:     1,
		PageSize: 10,
	}
	
	body, _ = json.Marshal(invalidReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/combined", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.CombinedSearch(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Internal error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	errorReq := dto.CombinedSearchRequest{
		Query: "error",
		Metadata: map[string]string{
			"error": "true",
		},
		Page:     1,
		PageSize: 10,
	}
	
	mockUseCase.On("CombinedSearch", mock.Anything, errorReq.Query, errorReq.Metadata, "tenant-123", mock.Anything).
		Return(pagination.PaginatedResult[models.Document]{}, errors.NewInternalError("internal error"))
	
	body, _ = json.Marshal(errorReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/combined", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.CombinedSearch(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	mockUseCase.AssertExpectations(t)
}

func TestSearchHandler_SearchInFolder(t *testing.T) {
	mockUseCase, _, handler := setupTest()
	
	// Success case
	folderReq := dto.FolderSearchRequest{
		FolderID: "folder-123",
		Query:    "test",
		Page:     1,
		PageSize: 10,
	}
	
	testDocs := []models.Document{
		createTestDocument("doc-1"),
		createTestDocument("doc-2"),
	}
	
	// Create expected result
	expectedResult := pagination.PaginatedResult[models.Document]{
		Items: testDocs,
		Pagination: pagination.PageInfo{
			Page:        folderReq.Page,
			PageSize:    folderReq.PageSize,
			TotalPages:  1,
			TotalItems:  int64(len(testDocs)),
			HasNext:     false,
			HasPrevious: false,
		},
	}
	
	// Set up mock expectations
	mockUseCase.On("SearchInFolder", mock.Anything, folderReq.FolderID, folderReq.Query, "tenant-123", mock.Anything).
		Return(expectedResult, nil)
	
	// Create request
	body, _ := json.Marshal(folderReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/search/folder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	// Set up response recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("tenant_id", "tenant-123")
	
	// Call handler
	handler.SearchInFolder(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response dto.DocumentSearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.True(t, response.Success)
	assert.Equal(t, len(testDocs), len(response.Results))
	assert.Equal(t, folderReq.Page, response.Pagination.Page)
	assert.Equal(t, folderReq.PageSize, response.Pagination.PageSize)
	
	// Validation error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	invalidReq := dto.FolderSearchRequest{
		FolderID: "", // Empty folder ID should fail validation
		Query:    "test",
		Page:     1,
		PageSize: 10,
	}
	
	body, _ = json.Marshal(invalidReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/folder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchInFolder(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Folder not found case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	notFoundReq := dto.FolderSearchRequest{
		FolderID: "non-existent",
		Query:    "test",
		Page:     1,
		PageSize: 10,
	}
	
	mockUseCase.On("SearchInFolder", mock.Anything, notFoundReq.FolderID, notFoundReq.Query, "tenant-123", mock.Anything).
		Return(pagination.PaginatedResult[models.Document]{}, errors.NewResourceNotFoundError("folder not found"))
	
	body, _ = json.Marshal(notFoundReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/folder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchInFolder(c)
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	// Authorization error case
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Set("tenant_id", "tenant-123")
	
	authErrorReq := dto.FolderSearchRequest{
		FolderID: "unauthorized-folder",
		Query:    "test",
		Page:     1,
		PageSize: 10,
	}
	
	mockUseCase.On("SearchInFolder", mock.Anything, authErrorReq.FolderID, authErrorReq.Query, "tenant-123", mock.Anything).
		Return(pagination.PaginatedResult[models.Document]{}, errors.NewAuthorizationError("unauthorized access"))
	
	body, _ = json.Marshal(authErrorReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/search/folder", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	
	handler.SearchInFolder(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	mockUseCase.AssertExpectations(t)
}

func TestSearchHandler_handleSearchError(t *testing.T) {
	_, _, handler := setupTest()
	
	// Test validation error
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	validationErr := errors.NewValidationError("validation error")
	handler.handleSearchError(c, validationErr)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Test resource not found error
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	notFoundErr := errors.NewResourceNotFoundError("resource not found")
	handler.handleSearchError(c, notFoundErr)
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	// Test authorization error
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	authErr := errors.NewAuthorizationError("authorization error")
	handler.handleSearchError(c, authErr)
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	// Test generic error
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	genericErr := fmt.Errorf("generic error")
	handler.handleSearchError(c, genericErr)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSearchHandler_convertToSearchResults(t *testing.T) {
	_, _, handler := setupTest()
	
	// Create test documents
	testDocs := []models.Document{
		createTestDocument("doc-1"),
		createTestDocument("doc-2"),
		createTestDocument("doc-3"),
	}
	
	// Convert to search results
	results := handler.convertToSearchResults(testDocs)
	
	// Assert the conversion
	assert.Equal(t, len(testDocs), len(results))
	
	for i, doc := range testDocs {
		assert.Equal(t, doc.ID, results[i].ID)
		assert.Equal(t, doc.Name, results[i].Name)
		assert.Equal(t, doc.ContentType, results[i].ContentType)
		assert.Equal(t, doc.Size, results[i].Size)
		assert.Equal(t, doc.FolderID, results[i].FolderID)
		assert.Equal(t, doc.Status, results[i].Status)
		assert.Equal(t, doc.OwnerID, results[i].CreatedBy)
	}
}