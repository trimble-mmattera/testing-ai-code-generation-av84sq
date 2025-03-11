package usecases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+

	"../../domain/models"
	"../../domain/services"
	appErrors "../../pkg/errors"
	"../../pkg/utils"
	"../usecases"
)

// MockSearchService is a mock implementation of the SearchService interface for testing
type MockSearchService struct {
	mock.Mock
}

// Implement SearchService interface methods for mocking
func (m *MockSearchService) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, query, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchService) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, metadata, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchService) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, contentQuery, metadata, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchService) SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	args := m.Called(ctx, folderID, query, tenantID, pagination)
	return args.Get(0).(utils.PaginatedResult[models.Document]), args.Error(1)
}

func (m *MockSearchService) IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error {
	args := m.Called(ctx, documentID, tenantID, content)
	return args.Error(0)
}

func (m *MockSearchService) RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error {
	args := m.Called(ctx, documentID, tenantID)
	return args.Error(0)
}

// SearchUseCaseTestSuite defines the test suite for the search use case
type SearchUseCaseTestSuite struct {
	suite.Suite
	mockSearchService *MockSearchService
	searchUseCase     *usecases.SearchUseCase
}

// SetupTest sets up the test environment before each test
func (s *SearchUseCaseTestSuite) SetupTest() {
	s.mockSearchService = new(MockSearchService)
	s.searchUseCase = usecases.NewSearchUseCase(s.mockSearchService)
}

// TestNewSearchUseCase_Success tests successful creation of a search use case
func (s *SearchUseCaseTestSuite) TestNewSearchUseCase_Success() {
	mockService := new(MockSearchService)
	useCase := usecases.NewSearchUseCase(mockService)
	
	assert.NotNil(s.T(), useCase)
}

// TestNewSearchUseCase_NilService tests that creating a search use case with nil service returns an error
func (s *SearchUseCaseTestSuite) TestNewSearchUseCase_NilService() {
	useCase, err := usecases.NewSearchUseCase(nil)
	
	assert.Nil(s.T(), useCase)
	assert.Error(s.T(), err)
	assert.True(s.T(), appErrors.IsValidationError(err))
}

// TestSearchByContent_Success tests successful content search
func (s *SearchUseCaseTestSuite) TestSearchByContent_Success() {
	// Setup test data
	ctx := context.Background()
	query := "test query"
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Create expected result
	doc := &models.Document{ID: "doc-123", Name: "Test Document", TenantID: tenantID}
	expectedResult := utils.PaginatedResult[models.Document]{
		Items: []*models.Document{doc},
		Pagination: utils.PageInfo{
			Page: 1,
			PageSize: 10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext: false,
			HasPrevious: false,
		},
	}
	
	// Set up mock search service to return expected result
	s.mockSearchService.On("SearchByContent", ctx, query, tenantID, pagination).
		Return(expectedResult, nil)
	
	// Call searchUseCase.SearchByContent with test data
	result, err := s.searchUseCase.SearchByContent(ctx, query, tenantID, pagination)
	
	// Assert that the returned result matches expected result
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedResult, result)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchByContent_EmptyQuery tests that content search with empty query returns an error
func (s *SearchUseCaseTestSuite) TestSearchByContent_EmptyQuery() {
	// Call searchUseCase.SearchByContent with empty query
	_, err := s.searchUseCase.SearchByContent(context.Background(), "", "tenant-123", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptySearchQuery
	assert.Equal(s.T(), services.ErrEmptySearchQuery, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchByContent")
}

// TestSearchByContent_EmptyTenantID tests that content search with empty tenant ID returns an error
func (s *SearchUseCaseTestSuite) TestSearchByContent_EmptyTenantID() {
	// Call searchUseCase.SearchByContent with empty tenant ID
	_, err := s.searchUseCase.SearchByContent(context.Background(), "test query", "", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyTenantID
	assert.Equal(s.T(), services.ErrEmptyTenantID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchByContent")
}

// TestSearchByContent_ServiceError tests handling of service errors during content search
func (s *SearchUseCaseTestSuite) TestSearchByContent_ServiceError() {
	// Create test data
	ctx := context.Background()
	query := "test query"
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock search service to return an error
	expectedError := errors.New("service error")
	s.mockSearchService.On("SearchByContent", ctx, query, tenantID, pagination).
		Return(utils.PaginatedResult[models.Document]{}, expectedError)
	
	// Call searchUseCase.SearchByContent with test data
	_, err := s.searchUseCase.SearchByContent(ctx, query, tenantID, pagination)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is the same as the service error
	assert.Equal(s.T(), expectedError, err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchByMetadata_Success tests successful metadata search
func (s *SearchUseCaseTestSuite) TestSearchByMetadata_Success() {
	// Create test data
	ctx := context.Background()
	metadata := map[string]string{"key": "value"}
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Create expected result with document list
	doc := &models.Document{ID: "doc-123", Name: "Test Document", TenantID: tenantID}
	expectedResult := utils.PaginatedResult[models.Document]{
		Items: []*models.Document{doc},
		Pagination: utils.PageInfo{
			Page: 1,
			PageSize: 10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext: false,
			HasPrevious: false,
		},
	}
	
	// Set up mock search service to return expected result
	s.mockSearchService.On("SearchByMetadata", ctx, metadata, tenantID, pagination).
		Return(expectedResult, nil)
	
	// Call searchUseCase.SearchByMetadata with test data
	result, err := s.searchUseCase.SearchByMetadata(ctx, metadata, tenantID, pagination)
	
	// Assert that the returned result matches expected result
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedResult, result)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchByMetadata_EmptyMetadata tests that metadata search with empty metadata returns an error
func (s *SearchUseCaseTestSuite) TestSearchByMetadata_EmptyMetadata() {
	// Call searchUseCase.SearchByMetadata with empty metadata map
	_, err := s.searchUseCase.SearchByMetadata(context.Background(), nil, "tenant-123", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyMetadataQuery
	assert.Equal(s.T(), services.ErrEmptyMetadataQuery, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchByMetadata")
}

// TestSearchByMetadata_EmptyTenantID tests that metadata search with empty tenant ID returns an error
func (s *SearchUseCaseTestSuite) TestSearchByMetadata_EmptyTenantID() {
	// Call searchUseCase.SearchByMetadata with empty tenant ID
	metadata := map[string]string{"key": "value"}
	_, err := s.searchUseCase.SearchByMetadata(context.Background(), metadata, "", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyTenantID
	assert.Equal(s.T(), services.ErrEmptyTenantID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchByMetadata")
}

// TestSearchByMetadata_ServiceError tests handling of service errors during metadata search
func (s *SearchUseCaseTestSuite) TestSearchByMetadata_ServiceError() {
	// Create test data
	ctx := context.Background()
	metadata := map[string]string{"key": "value"}
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock search service to return an error
	expectedError := errors.New("service error")
	s.mockSearchService.On("SearchByMetadata", ctx, metadata, tenantID, pagination).
		Return(utils.PaginatedResult[models.Document]{}, expectedError)
	
	// Call searchUseCase.SearchByMetadata with test data
	_, err := s.searchUseCase.SearchByMetadata(ctx, metadata, tenantID, pagination)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is the same as the service error
	assert.Equal(s.T(), expectedError, err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestCombinedSearch_Success tests successful combined search
func (s *SearchUseCaseTestSuite) TestCombinedSearch_Success() {
	// Create test data
	ctx := context.Background()
	contentQuery := "test query"
	metadata := map[string]string{"key": "value"}
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Create expected result with document list
	doc := &models.Document{ID: "doc-123", Name: "Test Document", TenantID: tenantID}
	expectedResult := utils.PaginatedResult[models.Document]{
		Items: []*models.Document{doc},
		Pagination: utils.PageInfo{
			Page: 1,
			PageSize: 10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext: false,
			HasPrevious: false,
		},
	}
	
	// Set up mock search service to return expected result
	s.mockSearchService.On("CombinedSearch", ctx, contentQuery, metadata, tenantID, pagination).
		Return(expectedResult, nil)
	
	// Call searchUseCase.CombinedSearch with test data
	result, err := s.searchUseCase.CombinedSearch(ctx, contentQuery, metadata, tenantID, pagination)
	
	// Assert that the returned result matches expected result
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedResult, result)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestCombinedSearch_NoSearchCriteria tests that combined search with no search criteria returns an error
func (s *SearchUseCaseTestSuite) TestCombinedSearch_NoSearchCriteria() {
	// Call searchUseCase.CombinedSearch with empty content query and empty metadata
	_, err := s.searchUseCase.CombinedSearch(context.Background(), "", nil, "tenant-123", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrNoSearchCriteria
	assert.Equal(s.T(), services.ErrNoSearchCriteria, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "CombinedSearch")
}

// TestCombinedSearch_EmptyTenantID tests that combined search with empty tenant ID returns an error
func (s *SearchUseCaseTestSuite) TestCombinedSearch_EmptyTenantID() {
	// Call searchUseCase.CombinedSearch with empty tenant ID
	contentQuery := "test query"
	metadata := map[string]string{"key": "value"}
	_, err := s.searchUseCase.CombinedSearch(context.Background(), contentQuery, metadata, "", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyTenantID
	assert.Equal(s.T(), services.ErrEmptyTenantID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "CombinedSearch")
}

// TestCombinedSearch_ServiceError tests handling of service errors during combined search
func (s *SearchUseCaseTestSuite) TestCombinedSearch_ServiceError() {
	// Create test data
	ctx := context.Background()
	contentQuery := "test query"
	metadata := map[string]string{"key": "value"}
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock search service to return an error
	expectedError := errors.New("service error")
	s.mockSearchService.On("CombinedSearch", ctx, contentQuery, metadata, tenantID, pagination).
		Return(utils.PaginatedResult[models.Document]{}, expectedError)
	
	// Call searchUseCase.CombinedSearch with test data
	_, err := s.searchUseCase.CombinedSearch(ctx, contentQuery, metadata, tenantID, pagination)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is the same as the service error
	assert.Equal(s.T(), expectedError, err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchInFolder_Success tests successful folder search
func (s *SearchUseCaseTestSuite) TestSearchInFolder_Success() {
	// Create test data
	ctx := context.Background()
	folderID := "folder-123"
	query := "test query"
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Create expected result with document list
	doc := &models.Document{ID: "doc-123", Name: "Test Document", TenantID: tenantID}
	expectedResult := utils.PaginatedResult[models.Document]{
		Items: []*models.Document{doc},
		Pagination: utils.PageInfo{
			Page: 1,
			PageSize: 10,
			TotalItems: 1,
			TotalPages: 1,
			HasNext: false,
			HasPrevious: false,
		},
	}
	
	// Set up mock search service to return expected result
	s.mockSearchService.On("SearchInFolder", ctx, folderID, query, tenantID, pagination).
		Return(expectedResult, nil)
	
	// Call searchUseCase.SearchInFolder with test data
	result, err := s.searchUseCase.SearchInFolder(ctx, folderID, query, tenantID, pagination)
	
	// Assert that the returned result matches expected result
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedResult, result)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchInFolder_EmptyFolderID tests that folder search with empty folder ID returns an error
func (s *SearchUseCaseTestSuite) TestSearchInFolder_EmptyFolderID() {
	// Call searchUseCase.SearchInFolder with empty folder ID
	_, err := s.searchUseCase.SearchInFolder(context.Background(), "", "test query", "tenant-123", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyFolderID
	assert.Equal(s.T(), services.ErrEmptyFolderID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchInFolder")
}

// TestSearchInFolder_EmptyQuery tests that folder search with empty query returns an error
func (s *SearchUseCaseTestSuite) TestSearchInFolder_EmptyQuery() {
	// Call searchUseCase.SearchInFolder with empty query
	_, err := s.searchUseCase.SearchInFolder(context.Background(), "folder-123", "", "tenant-123", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptySearchQuery
	assert.Equal(s.T(), services.ErrEmptySearchQuery, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchInFolder")
}

// TestSearchInFolder_EmptyTenantID tests that folder search with empty tenant ID returns an error
func (s *SearchUseCaseTestSuite) TestSearchInFolder_EmptyTenantID() {
	// Call searchUseCase.SearchInFolder with empty tenant ID
	_, err := s.searchUseCase.SearchInFolder(context.Background(), "folder-123", "test query", "", utils.NewPagination(1, 10))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyTenantID
	assert.Equal(s.T(), services.ErrEmptyTenantID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "SearchInFolder")
}

// TestSearchInFolder_ServiceError tests handling of service errors during folder search
func (s *SearchUseCaseTestSuite) TestSearchInFolder_ServiceError() {
	// Create test data
	ctx := context.Background()
	folderID := "folder-123"
	query := "test query"
	tenantID := "tenant-123"
	pagination := utils.NewPagination(1, 10)
	
	// Set up mock search service to return an error
	expectedError := errors.New("service error")
	s.mockSearchService.On("SearchInFolder", ctx, folderID, query, tenantID, pagination).
		Return(utils.PaginatedResult[models.Document]{}, expectedError)
	
	// Call searchUseCase.SearchInFolder with test data
	_, err := s.searchUseCase.SearchInFolder(ctx, folderID, query, tenantID, pagination)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is the same as the service error
	assert.Equal(s.T(), expectedError, err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestIndexDocument_Success tests successful document indexing
func (s *SearchUseCaseTestSuite) TestIndexDocument_Success() {
	// Create test data
	ctx := context.Background()
	documentID := "doc-123"
	tenantID := "tenant-123"
	content := []byte("test content")
	
	// Set up mock search service to return nil error
	s.mockSearchService.On("IndexDocument", ctx, documentID, tenantID, content).Return(nil)
	
	// Call searchUseCase.IndexDocument with test data
	err := s.searchUseCase.IndexDocument(ctx, documentID, tenantID, content)
	
	// Assert that no error is returned
	assert.NoError(s.T(), err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestIndexDocument_EmptyDocumentID tests that document indexing with empty document ID returns an error
func (s *SearchUseCaseTestSuite) TestIndexDocument_EmptyDocumentID() {
	// Call searchUseCase.IndexDocument with empty document ID
	err := s.searchUseCase.IndexDocument(context.Background(), "", "tenant-123", []byte("test content"))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is a validation error
	assert.True(s.T(), appErrors.IsValidationError(err))
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "IndexDocument")
}

// TestIndexDocument_EmptyTenantID tests that document indexing with empty tenant ID returns an error
func (s *SearchUseCaseTestSuite) TestIndexDocument_EmptyTenantID() {
	// Call searchUseCase.IndexDocument with empty tenant ID
	err := s.searchUseCase.IndexDocument(context.Background(), "doc-123", "", []byte("test content"))
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyTenantID
	assert.Equal(s.T(), services.ErrEmptyTenantID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "IndexDocument")
}

// TestIndexDocument_EmptyContent tests that document indexing with empty content returns an error
func (s *SearchUseCaseTestSuite) TestIndexDocument_EmptyContent() {
	// Call searchUseCase.IndexDocument with empty content
	err := s.searchUseCase.IndexDocument(context.Background(), "doc-123", "tenant-123", nil)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is a validation error
	assert.True(s.T(), appErrors.IsValidationError(err))
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "IndexDocument")
}

// TestIndexDocument_ServiceError tests handling of service errors during document indexing
func (s *SearchUseCaseTestSuite) TestIndexDocument_ServiceError() {
	// Create test data
	ctx := context.Background()
	documentID := "doc-123"
	tenantID := "tenant-123"
	content := []byte("test content")
	
	// Set up mock search service to return an error
	expectedError := errors.New("service error")
	s.mockSearchService.On("IndexDocument", ctx, documentID, tenantID, content).Return(expectedError)
	
	// Call searchUseCase.IndexDocument with test data
	err := s.searchUseCase.IndexDocument(ctx, documentID, tenantID, content)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is the same as the service error
	assert.Equal(s.T(), expectedError, err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestRemoveDocumentFromIndex_Success tests successful document removal from index
func (s *SearchUseCaseTestSuite) TestRemoveDocumentFromIndex_Success() {
	// Create test data
	ctx := context.Background()
	documentID := "doc-123"
	tenantID := "tenant-123"
	
	// Set up mock search service to return nil error
	s.mockSearchService.On("RemoveDocumentFromIndex", ctx, documentID, tenantID).Return(nil)
	
	// Call searchUseCase.RemoveDocumentFromIndex with test data
	err := s.searchUseCase.RemoveDocumentFromIndex(ctx, documentID, tenantID)
	
	// Assert that no error is returned
	assert.NoError(s.T(), err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestRemoveDocumentFromIndex_EmptyDocumentID tests that document removal with empty document ID returns an error
func (s *SearchUseCaseTestSuite) TestRemoveDocumentFromIndex_EmptyDocumentID() {
	// Call searchUseCase.RemoveDocumentFromIndex with empty document ID
	err := s.searchUseCase.RemoveDocumentFromIndex(context.Background(), "", "tenant-123")
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is a validation error
	assert.True(s.T(), appErrors.IsValidationError(err))
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "RemoveDocumentFromIndex")
}

// TestRemoveDocumentFromIndex_EmptyTenantID tests that document removal with empty tenant ID returns an error
func (s *SearchUseCaseTestSuite) TestRemoveDocumentFromIndex_EmptyTenantID() {
	// Call searchUseCase.RemoveDocumentFromIndex with empty tenant ID
	err := s.searchUseCase.RemoveDocumentFromIndex(context.Background(), "doc-123", "")
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is ErrEmptyTenantID
	assert.Equal(s.T(), services.ErrEmptyTenantID, err)
	
	// Verify that the mock was not called
	s.mockSearchService.AssertNotCalled(s.T(), "RemoveDocumentFromIndex")
}

// TestRemoveDocumentFromIndex_ServiceError tests handling of service errors during document removal from index
func (s *SearchUseCaseTestSuite) TestRemoveDocumentFromIndex_ServiceError() {
	// Create test data
	ctx := context.Background()
	documentID := "doc-123"
	tenantID := "tenant-123"
	
	// Set up mock search service to return an error
	expectedError := errors.New("service error")
	s.mockSearchService.On("RemoveDocumentFromIndex", ctx, documentID, tenantID).Return(expectedError)
	
	// Call searchUseCase.RemoveDocumentFromIndex with test data
	err := s.searchUseCase.RemoveDocumentFromIndex(ctx, documentID, tenantID)
	
	// Assert that an error is returned
	assert.Error(s.T(), err)
	// Assert that the error is the same as the service error
	assert.Equal(s.T(), expectedError, err)
	
	// Verify that the mock was called with correct parameters
	s.mockSearchService.AssertExpectations(s.T())
}

// TestSearchUseCaseSuite runs the test suite
func TestSearchUseCaseSuite(t *testing.T) {
	suite.Run(t, new(SearchUseCaseTestSuite))
}