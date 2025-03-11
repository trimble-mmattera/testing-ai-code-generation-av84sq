package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin" // v1.9.0+
	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/suite" // v1.8.0+

	"../../domain/services/auth_service" // For mocking authentication service in tests
	"../../pkg/errors" // For verifying error types in tests
	"../../pkg/config" // For creating test configurations
)

// MockAuthService is a mock implementation of the AuthService interface for testing
type MockAuthService struct {
	mock.Mock
}

// ValidateToken mocks the ValidateToken method of AuthService
func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (string, []string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Get(1).([]string), args.Error(2)
}

// VerifyTenantAccess mocks the VerifyTenantAccess method of AuthService
func (m *MockAuthService) VerifyTenantAccess(ctx context.Context, userID, tenantID string) (bool, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Bool(0), args.Error(1)
}

// MiddlewareSuite is a test suite for middleware components
type MiddlewareSuite struct {
	suite.Suite
	mockAuthService *MockAuthService
	testConfig      config.Config
}

// SetupSuite initializes the test suite before running tests
func (s *MiddlewareSuite) SetupSuite() {
	s.mockAuthService = new(MockAuthService)
	
	// Create test configuration
	s.testConfig = config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		JWT: config.JWTConfig{
			Secret:         "test-secret",
			ExpirationTime: "1h",
			Issuer:         "document-management-platform",
		},
	}
}

// SetupTest is run before each test to reset the mock
func (s *MiddlewareSuite) SetupTest() {
	s.mockAuthService = new(MockAuthService)
}

// TestMiddlewareSuite runs the middleware test suite
func TestMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareSuite))
}

// setupTestRouter creates a test router with middleware
func setupTestRouter(s *MiddlewareSuite, middleware ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add middleware
	for _, m := range middleware {
		router.Use(m)
	}
	
	// Add a test endpoint
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	
	return router
}

// createTestRequest creates a test HTTP request
func createTestRequest(method, path string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	return req
}

// createTestJWT creates a test JWT token
func (s *MiddlewareSuite) createTestJWT(tenantID, userID string, roles []string) string {
	// In a real implementation, we'd generate an actual JWT
	// For testing, we return a dummy token
	return "test-jwt-token-" + tenantID
}

// TestAuthMiddleware_ValidToken tests that AuthMiddleware accepts valid tokens
func (s *MiddlewareSuite) TestAuthMiddleware_ValidToken() {
	// Arrange
	token := s.createTestJWT("tenant-123", "user-123", []string{"admin"})
	s.mockAuthService.On("ValidateToken", mock.Anything, token).
		Return("tenant-123", []string{"admin"}, nil)
	
	router := setupTestRouter(s, AuthMiddleware(s.mockAuthService))
	
	req := createTestRequest("GET", "/test", map[string]string{
		"Authorization": "Bearer " + token,
	})
	
	// Act
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)
	s.mockAuthService.AssertExpectations(s.T())
}

// TestAuthMiddleware_InvalidToken tests that AuthMiddleware rejects invalid tokens
func (s *MiddlewareSuite) TestAuthMiddleware_InvalidToken() {
	// Arrange
	s.mockAuthService.On("ValidateToken", mock.Anything, "invalid-token").
		Return("", []string{}, errors.NewAuthenticationError("invalid token"))
	
	router := setupTestRouter(s, AuthMiddleware(s.mockAuthService))
	
	req := createTestRequest("GET", "/test", map[string]string{
		"Authorization": "Bearer invalid-token",
	})
	
	// Act
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
	s.mockAuthService.AssertExpectations(s.T())
}

// TestAuthMiddleware_MissingToken tests that AuthMiddleware rejects requests without tokens
func (s *MiddlewareSuite) TestAuthMiddleware_MissingToken() {
	// Arrange
	router := setupTestRouter(s, AuthMiddleware(s.mockAuthService))
	req := createTestRequest("GET", "/test", nil)
	
	// Act
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// TestRequireAuthentication tests that RequireAuthentication middleware enforces authentication
func (s *MiddlewareSuite) TestRequireAuthentication() {
	// Arrange - test with user ID in context
	router := setupTestRouter(s, RequireAuthentication())
	
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set(UserIDKey, "user-123")
	
	// Act - execute middleware directly
	RequireAuthentication()(ctx)
	
	// Assert - should continue the chain
	assert.False(s.T(), ctx.IsAborted())
	
	// Arrange - test without user ID in context
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	
	// Act - execute middleware directly
	RequireAuthentication()(ctx)
	
	// Assert - should abort with 401
	assert.True(s.T(), ctx.IsAborted())
	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// TestRequireRole tests that RequireRole middleware enforces role requirements
func (s *MiddlewareSuite) TestRequireRole() {
	// Arrange - test with required role
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set(UserRolesKey, []string{"admin", "user"})
	
	// Act - execute middleware directly
	RequireRole("admin")(ctx)
	
	// Assert - should continue the chain
	assert.False(s.T(), ctx.IsAborted())
	
	// Arrange - test without required role
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Set(UserRolesKey, []string{"user"})
	
	// Act - execute middleware directly
	RequireRole("admin")(ctx)
	
	// Assert - should abort with 403
	assert.True(s.T(), ctx.IsAborted())
	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

// TestRequireAnyRole tests that RequireAnyRole middleware enforces role requirements
func (s *MiddlewareSuite) TestRequireAnyRole() {
	// Arrange - test with admin role
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set(UserRolesKey, []string{"admin", "user"})
	
	// Act - execute middleware directly
	RequireAnyRole("admin", "editor")(ctx)
	
	// Assert - should continue the chain
	assert.False(s.T(), ctx.IsAborted())
	
	// Arrange - test with editor role
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Set(UserRolesKey, []string{"editor", "user"})
	
	// Act - execute middleware directly
	RequireAnyRole("admin", "editor")(ctx)
	
	// Assert - should continue the chain
	assert.False(s.T(), ctx.IsAborted())
	
	// Arrange - test without any required role
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Set(UserRolesKey, []string{"user"})
	
	// Act - execute middleware directly
	RequireAnyRole("admin", "editor")(ctx)
	
	// Assert - should abort with 403
	assert.True(s.T(), ctx.IsAborted())
	assert.Equal(s.T(), http.StatusForbidden, w.Code)
}

// TestTenantMiddleware tests that TenantMiddleware enforces tenant isolation
func (s *MiddlewareSuite) TestTenantMiddleware() {
	// Arrange - test with matching tenant IDs
	s.mockAuthService.On("VerifyTenantAccess", mock.Anything, "user-123", "tenant-123").
		Return(true, nil)
	
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Set(UserIDKey, "user-123")
	ctx.Set(TenantIDKey, "tenant-123")
	ctx.Params = []gin.Param{{Key: "tenant_id", Value: "tenant-123"}}
	
	// Act - execute middleware directly
	TenantMiddleware(s.mockAuthService)(ctx)
	
	// Assert - should continue the chain
	assert.False(s.T(), ctx.IsAborted())
	s.mockAuthService.AssertExpectations(s.T())
	
	// Arrange - test with non-matching tenant IDs
	s.mockAuthService = new(MockAuthService)
	s.mockAuthService.On("VerifyTenantAccess", mock.Anything, "user-123", "tenant-456").
		Return(false, nil)
	
	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Set(UserIDKey, "user-123")
	ctx.Set(TenantIDKey, "tenant-123")
	ctx.Params = []gin.Param{{Key: "tenant_id", Value: "tenant-456"}}
	
	// Act - execute middleware directly
	TenantMiddleware(s.mockAuthService)(ctx)
	
	// Assert - should abort with 403
	assert.True(s.T(), ctx.IsAborted())
	assert.Equal(s.T(), http.StatusForbidden, w.Code)
	s.mockAuthService.AssertExpectations(s.T())
}

// TestCORSMiddleware tests that CORSMiddleware sets appropriate CORS headers
func (s *MiddlewareSuite) TestCORSMiddleware() {
	// Arrange - setup CORS config
	corsConfig := s.testConfig
	corsConfig.Server.CORSEnabled = true
	corsConfig.Server.CORSAllowOrigins = []string{"http://example.com"}
	corsConfig.Server.CORSAllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	corsConfig.Server.CORSAllowHeaders = []string{"Content-Type", "Authorization"}
	corsConfig.Server.CORSExposeHeaders = []string{"Content-Length"}
	corsConfig.Server.CORSAllowCredentials = true
	corsConfig.Server.CORSMaxAge = 86400
	
	router := setupTestRouter(s, CORSMiddleware(corsConfig))
	
	// Create a preflight request
	req := createTestRequest("OPTIONS", "/test", map[string]string{
		"Origin": "http://example.com",
		"Access-Control-Request-Method": "POST",
		"Access-Control-Request-Headers": "Content-Type, Authorization",
	})
	
	// Act
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Equal(s.T(), "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(s.T(), w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(s.T(), w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(s.T(), w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(s.T(), "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(s.T(), "86400", w.Header().Get("Access-Control-Max-Age"))
}

// TestLoggingMiddleware tests that LoggingMiddleware adds request ID and logs requests
func (s *MiddlewareSuite) TestLoggingMiddleware() {
	// Arrange
	router := setupTestRouter(s, LoggingMiddleware())
	req := createTestRequest("GET", "/test", nil)
	
	// Act
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.NotEmpty(s.T(), w.Header().Get("X-Request-ID"))
}

// TestRateLimiterMiddleware tests that RateLimiterMiddleware enforces rate limits
func (s *MiddlewareSuite) TestRateLimiterMiddleware() {
	// Arrange - create router with a low rate limit
	router := setupTestRouter(s, RateLimiterMiddleware(2, time.Minute))
	
	// Create multiple requests from same IP
	req1 := createTestRequest("GET", "/test", map[string]string{
		"X-Real-IP": "192.168.1.1",
	})
	req2 := createTestRequest("GET", "/test", map[string]string{
		"X-Real-IP": "192.168.1.1",
	})
	req3 := createTestRequest("GET", "/test", map[string]string{
		"X-Real-IP": "192.168.1.1",
	})
	
	// Act & Assert
	// First request - should pass
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(s.T(), http.StatusOK, w1.Code)
	assert.NotEmpty(s.T(), w1.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(s.T(), w1.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(s.T(), w1.Header().Get("X-RateLimit-Reset"))
	
	// Second request - should pass
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(s.T(), http.StatusOK, w2.Code)
	
	// Third request - should be rate limited
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(s.T(), http.StatusTooManyRequests, w3.Code)
}

// TestRecoveryMiddleware tests that RecoveryMiddleware catches panics
func (s *MiddlewareSuite) TestRecoveryMiddleware() {
	// Arrange - create router with recovery middleware and a handler that panics
	router := gin.New()
	router.Use(RecoveryMiddleware())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})
	
	req := createTestRequest("GET", "/panic", nil)
	
	// Act
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(s.T(), http.StatusInternalServerError, w.Code)
	
	// Verify error response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), response, "error")
}

// TestGetUserID tests that GetUserID extracts user ID from context
func (s *MiddlewareSuite) TestGetUserID() {
	// Arrange - context with user ID
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(UserIDKey, "user-123")
	
	// Act & Assert
	assert.Equal(s.T(), "user-123", GetUserID(c))
	
	// Test with missing user ID
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(s.T(), "", GetUserID(c))
}

// TestGetTenantID tests that GetTenantID extracts tenant ID from context
func (s *MiddlewareSuite) TestGetTenantID() {
	// Arrange - context with tenant ID
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(TenantIDKey, "tenant-123")
	
	// Act & Assert
	assert.Equal(s.T(), "tenant-123", GetTenantID(c))
	
	// Test with missing tenant ID
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(s.T(), "", GetTenantID(c))
}

// TestGetUserRoles tests that GetUserRoles extracts user roles from context
func (s *MiddlewareSuite) TestGetUserRoles() {
	// Arrange - context with roles
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(UserRolesKey, []string{"admin", "user"})
	
	// Act & Assert
	assert.Equal(s.T(), []string{"admin", "user"}, GetUserRoles(c))
	
	// Test with missing roles
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	assert.Empty(s.T(), GetUserRoles(c))
}

// TestHasRole tests that HasRole checks if a user has a specific role
func (s *MiddlewareSuite) TestHasRole() {
	// Arrange - context with roles
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(UserRolesKey, []string{"admin", "user"})
	
	// Act & Assert
	assert.True(s.T(), HasRole(c, "admin"))
	assert.True(s.T(), HasRole(c, "user"))
	assert.False(s.T(), HasRole(c, "editor"))
	
	// Test with missing roles
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	assert.False(s.T(), HasRole(c, "admin"))
}