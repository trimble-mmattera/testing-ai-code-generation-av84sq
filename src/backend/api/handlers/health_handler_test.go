package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"../dto"
	"../../pkg/errors"
)

// MockHealthChecker is a mock implementation of the HealthChecker interface for testing
type MockHealthChecker struct {
	mock.Mock
}

// Check implements the HealthChecker interface
func (m *MockHealthChecker) Check(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

// HealthHandlerSuite is a test suite for health handler endpoints
type HealthHandlerSuite struct {
	suite.Suite
	router              *gin.Engine
	mockDBChecker       *MockHealthChecker
	mockStorageChecker  *MockHealthChecker
	mockSearchChecker   *MockHealthChecker
	healthHandler       *HealthHandler
}

// SetupSuite sets up the test suite before running tests
func (s *HealthHandlerSuite) SetupSuite() {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize mock health checkers
	s.mockDBChecker = new(MockHealthChecker)
	s.mockStorageChecker = new(MockHealthChecker)
	s.mockSearchChecker = new(MockHealthChecker)

	// Create a map of checkers with database, storage, and search checkers
	checkers := map[string]HealthChecker{
		"database": s.mockDBChecker,
		"storage":  s.mockStorageChecker,
		"search":   s.mockSearchChecker,
	}

	// Initialize the health handler with the mock checkers
	s.healthHandler = NewHealthHandler(checkers)

	// Create a new gin router
	s.router = gin.New()

	// Register health check endpoints on the router
	s.healthHandler.RegisterRoutes(s.router)
}

// SetupTest sets up before each test
func (s *HealthHandlerSuite) SetupTest() {
	// Reset the mock health checkers
	s.mockDBChecker = new(MockHealthChecker)
	s.mockStorageChecker = new(MockHealthChecker)
	s.mockSearchChecker = new(MockHealthChecker)

	// Update the health handler with the fresh mocks
	checkers := map[string]HealthChecker{
		"database": s.mockDBChecker,
		"storage":  s.mockStorageChecker,
		"search":   s.mockSearchChecker,
	}

	s.healthHandler = NewHealthHandler(checkers)

	// Re-register routes with the fresh handler
	s.router = gin.New()
	s.healthHandler.RegisterRoutes(s.router)
}

// TestLivenessCheck tests that LivenessCheck returns 200 OK
func (s *HealthHandlerSuite) TestLivenessCheck() {
	// Create a test request to /health/live
	req, _ := http.NewRequest(http.MethodGet, "/health/live", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 200 OK
	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=true
	assert.True(s.T(), response.Success)
}

// TestReadinessCheck_AllDependenciesHealthy tests that ReadinessCheck returns 200 OK when all dependencies are healthy
func (s *HealthHandlerSuite) TestReadinessCheck_AllDependenciesHealthy() {
	// Configure mock checkers to return success
	s.mockDBChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)
	s.mockStorageChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)
	s.mockSearchChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)

	// Create a test request to /health/ready
	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 200 OK
	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=true
	assert.True(s.T(), response.Success)

	// Assert that the response contains status information for all dependencies
	responseData, ok := response.Data.(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Contains(s.T(), responseData, "database")
	assert.Contains(s.T(), responseData, "storage")
	assert.Contains(s.T(), responseData, "search")
}

// TestReadinessCheck_DatabaseUnhealthy tests that ReadinessCheck returns 503 Service Unavailable when database is unhealthy
func (s *HealthHandlerSuite) TestReadinessCheck_DatabaseUnhealthy() {
	// Configure database checker to return an error
	s.mockDBChecker.On("Check", mock.Anything).Return(nil, errors.NewDependencyError("database connection error"))

	// Configure other checkers to return success
	s.mockStorageChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)
	s.mockSearchChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)

	// Create a test request to /health/ready
	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 503 Service Unavailable
	assert.Equal(s.T(), http.StatusServiceUnavailable, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=false
	assert.False(s.T(), response.Success)

	// Assert that the response contains an error message about the database
	responseData, ok := response.Data.(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Contains(s.T(), responseData, "error")
	errorMsg, ok := responseData["error"].(string)
	assert.True(s.T(), ok)
	assert.Contains(s.T(), errorMsg, "database")
}

// TestReadinessCheck_StorageUnhealthy tests that ReadinessCheck returns 503 Service Unavailable when storage is unhealthy
func (s *HealthHandlerSuite) TestReadinessCheck_StorageUnhealthy() {
	// Configure storage checker to return an error
	s.mockStorageChecker.On("Check", mock.Anything).Return(nil, errors.NewDependencyError("storage connection error"))

	// Configure other checkers to return success
	s.mockDBChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)
	s.mockSearchChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)

	// Create a test request to /health/ready
	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 503 Service Unavailable
	assert.Equal(s.T(), http.StatusServiceUnavailable, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=false
	assert.False(s.T(), response.Success)

	// Assert that the response contains an error message about the storage
	responseData, ok := response.Data.(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Contains(s.T(), responseData, "error")
	errorMsg, ok := responseData["error"].(string)
	assert.True(s.T(), ok)
	assert.Contains(s.T(), errorMsg, "storage")
}

// TestReadinessCheck_SearchUnhealthy tests that ReadinessCheck returns 503 Service Unavailable when search is unhealthy
func (s *HealthHandlerSuite) TestReadinessCheck_SearchUnhealthy() {
	// Configure search checker to return an error
	s.mockSearchChecker.On("Check", mock.Anything).Return(nil, errors.NewDependencyError("search connection error"))

	// Configure other checkers to return success
	s.mockDBChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)
	s.mockStorageChecker.On("Check", mock.Anything).Return(map[string]string{"status": "up"}, nil)

	// Create a test request to /health/ready
	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 503 Service Unavailable
	assert.Equal(s.T(), http.StatusServiceUnavailable, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=false
	assert.False(s.T(), response.Success)

	// Assert that the response contains an error message about the search
	responseData, ok := response.Data.(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Contains(s.T(), responseData, "error")
	errorMsg, ok := responseData["error"].(string)
	assert.True(s.T(), ok)
	assert.Contains(s.T(), errorMsg, "search")
}

// TestDeepHealthCheck_AllDependenciesHealthy tests that DeepHealthCheck returns 200 OK with detailed status when all dependencies are healthy
func (s *HealthHandlerSuite) TestDeepHealthCheck_AllDependenciesHealthy() {
	// Configure mock checkers to return detailed status information
	s.mockDBChecker.On("Check", mock.Anything).Return(map[string]interface{}{
		"status": "up",
		"details": map[string]interface{}{
			"version":     "12.4",
			"connections": 5,
			"latency_ms":  2,
		},
	}, nil)

	s.mockStorageChecker.On("Check", mock.Anything).Return(map[string]interface{}{
		"status": "up",
		"details": map[string]interface{}{
			"region":     "us-west-2",
			"buckets":    3,
			"latency_ms": 15,
		},
	}, nil)

	s.mockSearchChecker.On("Check", mock.Anything).Return(map[string]interface{}{
		"status": "up",
		"details": map[string]interface{}{
			"version":    "7.10.0",
			"indices":    5,
			"latency_ms": 10,
		},
	}, nil)

	// Create a test request to /health/deep
	req, _ := http.NewRequest(http.MethodGet, "/health/deep", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 200 OK
	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=true
	assert.True(s.T(), response.Success)

	// Assert that the response contains detailed status information for all dependencies
	responseData, ok := response.Data.(map[string]interface{})
	assert.True(s.T(), ok)

	// Assert each dependency has detailed information
	dbData, ok := responseData["database"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "up", dbData["status"])
	assert.Contains(s.T(), dbData, "details")

	storageData, ok := responseData["storage"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "up", storageData["status"])
	assert.Contains(s.T(), storageData, "details")

	searchData, ok := responseData["search"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "up", searchData["status"])
	assert.Contains(s.T(), searchData, "details")
}

// TestDeepHealthCheck_DependencyUnhealthy tests that DeepHealthCheck returns 503 Service Unavailable with detailed error when a dependency is unhealthy
func (s *HealthHandlerSuite) TestDeepHealthCheck_DependencyUnhealthy() {
	// Configure one checker to return a detailed error
	s.mockDBChecker.On("Check", mock.Anything).Return(nil, errors.NewDependencyError("database connection timeout"))

	// Configure other checkers to return success
	s.mockStorageChecker.On("Check", mock.Anything).Return(map[string]interface{}{
		"status": "up",
		"details": map[string]interface{}{
			"region":  "us-west-2",
			"buckets": 3,
		},
	}, nil)

	s.mockSearchChecker.On("Check", mock.Anything).Return(map[string]interface{}{
		"status": "up",
		"details": map[string]interface{}{
			"version": "7.10.0",
			"indices": 5,
		},
	}, nil)

	// Create a test request to /health/deep
	req, _ := http.NewRequest(http.MethodGet, "/health/deep", nil)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Execute the request through the router
	s.router.ServeHTTP(w, req)

	// Assert that the response status is 503 Service Unavailable
	assert.Equal(s.T(), http.StatusServiceUnavailable, w.Code)

	// Parse the response body as JSON
	var response dto.DataResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)

	// Assert no parsing error
	assert.NoError(s.T(), err)

	// Assert that the response contains success=false
	assert.False(s.T(), response.Success)

	// Assert that the response contains detailed error information
	responseData, ok := response.Data.(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Contains(s.T(), responseData, "error")
	assert.Contains(s.T(), responseData, "details")

	details, ok := responseData["details"].(map[string]interface{})
	assert.True(s.T(), ok)

	// Check that database shows error
	dbStatus, ok := details["database"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Contains(s.T(), dbStatus, "error")
	assert.Contains(s.T(), dbStatus["error"], "database connection timeout")

	// Check that other dependencies show up status
	storageStatus, ok := details["storage"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "up", storageStatus["status"])

	searchStatus, ok := details["search"].(map[string]interface{})
	assert.True(s.T(), ok)
	assert.Equal(s.T(), "up", searchStatus["status"])
}

// TestHealthHandlerSuite runs the health handler test suite
func TestHealthHandlerSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerSuite))
}