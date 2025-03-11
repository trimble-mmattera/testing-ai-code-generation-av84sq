// Package integration contains integration tests for the Document Management Platform.
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"../../api/middleware"
	auth "../../domain/services"
	"../../infrastructure/auth/jwt"
	"../../domain/models"
	"../../pkg/config"
	"../../pkg/errors"
)

// AuthTestSuite is a test suite for authentication service integration tests
type AuthTestSuite struct {
	suite.Suite
	authService auth.AuthService
	jwtConfig   config.JWTConfig
	testUserID  string
	testTenantID string
	testRoles   []string
	userRepo    *mockUserRepository
	tenantRepo  *mockTenantRepository
}

// mockUserRepository is a mock implementation of the UserRepository interface
type mockUserRepository struct {
	mock.Mock
}

// GetByID is a mock implementation of UserRepository.GetByID
func (m *mockUserRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.User, error) {
	args := m.Called(ctx, id, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// mockTenantRepository is a mock implementation of the TenantRepository interface
type mockTenantRepository struct {
	mock.Mock
}

// GetByID is a mock implementation of TenantRepository.GetByID
func (m *mockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

// SetupSuite sets up the test suite before any tests run
func (s *AuthTestSuite) SetupSuite() {
	// Set up test data
	s.testUserID = "test-user-123"
	s.testTenantID = "test-tenant-456"
	s.testRoles = []string{"reader", "contributor"}

	// Set up JWT configuration with test keys
	s.jwtConfig = config.JWTConfig{
		// These are test RSA keys, not for production use
		PrivateKey: `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAvxM9hy1kJ5iCjv2wkKQZaioTQRVlxW9nEdOTYbOzDEZlL+QM
vGh5TiCFDN8Z2uQMQJ44HZ5EUKxGPJ8jPHGQVqyTWaPasgMGhJ7CZ0xgQFk7NErb
pZkE8W2lt5PSc5PXOjjNnlasgGHh/DryCWZ/0ZAyf5mVnF0kzZBXABGfzJ8SaCk5
gJPqCdFvWvE0vOLyVgJPQp7FUJ4W/pIlIad3jDpzs0tTI8UvxKqFIp4Y4dAhZH1v
hj+eT5SyvWvRbQCBABEwFQBCKvY9FYnYgYk6XVwH3YdX3PUlX7cYK0iHnZQgvPH5
hjhgGjwVbh2xW0xVUQJR+YCsw5j5ZfkKRQOxlwIDAQABAoIBAFPMX3JrD8RWPnkz
JQgpD9xUcQEhnXC7zEFt64Cz5LPzJmHxK+y+T3L+QqwwIRzHGiKAjHc+v/zfQ6jA
gWNVvHPxq7z0ZT8Hbha08/TPbZrYXtHGKrAV3j2kF+ZfFeXlF8NqrBU0HU8eFZ/t
Dj95Qeo3mgsEprt+ncDvFQpQFaLO0cLvRYuMwRbSY4Z3sRyvXtFBYXaBZS/opSO2
9meLNDYPtfIYz2KjdI+CuCrZdqPLRYJX9wKw0jGZhzjIw8zT6u9/rUxXOUEkP24N
ELQBdz9MpHD2CxBTNxcBEHsCmD9x+tSuJA3VwlnDvdmatSKjBdkjE4EKz7TSv1Dw
T9ArHSkCgYEA+z9XvTErPKhM4aOL0T5bOLCc6z0exQsHtL8mxnJy/9CtEUqLnFJZ
KPljyj4sRQnWiQFgaSkGgJ81NiUFbYEJLQ6iDmTfJK9KoNnWB9kb8vAQSvtmF7j5
lscSuR9XZ5eBxtHvDRb1FGGF0dIXxGn+xtpEE25Q9Hg+7tZIWcnTZhMCgYEAwviX
UVDgbS0wYiYtCfnYQiugRGkYB9hxAP6S4aBQqK0/qWRx6i9dzOdxRQVGeeKNiQVj
JtKEOikc4+9OL0eQCmygEvHYFGQA+5pVz8P7IRp22Q47nLUYWOm3jRo8mXNfrL9j
4yW8nUTUs5CrxFHEjA+c5vaqNPQZxknO4fBXfz0CgYEAoFQzLnFRl5rAvh3ZlmJG
RLTiwcV3vkcWJJzMvbGLCsHcxG6MOJBp2Lnq2cSoeGYWcaWJpMIXfvbiKzXkdS7c
0T5YgshEG1Sc7kOk+El8Po8EO5+POKsiJ81YUKz+JJIfXJBgYF46J3fP7iJZzQcI
ZyKUB5dpK3CLvIXrS0qSjzkCgYA6z0bF7Y5DrXOG9Py7PHYjNLnV7+1wGj9z2XYX
5FtQjAkIdGI1kHJTFHCzH3ZHIwxgIZHU8Pq4pVejWs7PJupnSszXfnLzrHFWTnNO
eGJxJCDSWDb1jOfLNWZuKKm8/X/76wsmkQhqX8EC1vYo5i4QlrK252qy7dUSqQOO
YNrGLQKBgGgr7QJ5igbTq8b18SAbZQAnYQNnTZCVu3tcCYCy8pECXCCczJy5qKL+
eagFefTuHH31BLYsxm+0Y5LZdHATvbKEBpbJkUWJ5T1NJY2R9RY//Sdtq4oRjkpY
/ATCB41YtNi1fgXftBmVJrKYaJhLLNjHokMzN3KYFR2nZOx3xYEk
-----END RSA PRIVATE KEY-----`,
		PublicKey: `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvxM9hy1kJ5iCjv2wkKQZ
aioTQRVlxW9nEdOTYbOzDEZlL+QMvGh5TiCFDN8Z2uQMQJ44HZ5EUKxGPJ8jPHGQ
VqyTWaPasgMGhJ7CZ0xgQFk7NErb
pZkE8W2lt5PSc5PXOjjNnlasgGHh/DryCWZ/0ZAyf5mVnF0kzZBXABGfzJ8SaCk5
gJPqCdFvWvE0vOLyVgJPQp7FUJ4W/pIlIad3jDpzs0tTI8UvxKqFIp4Y4dAhZH1v
hj+eT5SyvWvRbQCBABEwFQBCKvY9FYnYgYk6XVwH3YdX3PUlX7cYK0iHnZQgvPH5
hjhgGjwVbh2xW0xVUQJR+YCsw5j5ZfkKRQOxlwIDAQAB
-----END PUBLIC KEY-----`,
		Issuer: "test-issuer",
		Algorithm: "RS256",
		ExpirationTime: "1h",
	}

	// Create mock repositories
	s.userRepo = new(mockUserRepository)
	s.tenantRepo = new(mockTenantRepository)

	// Create JWT auth service
	var err error
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	require.NoError(s.T(), err, "Failed to create JWT auth service")
}

// SetupTest sets up each test before it runs
func (s *AuthTestSuite) SetupTest() {
	// Reset mocks before each test
	s.userRepo = new(mockUserRepository)
	s.tenantRepo = new(mockTenantRepository)

	// Create auth service with fresh mocks
	var err error
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	require.NoError(s.T(), err, "Failed to create JWT auth service")

	// Set up common mock behaviors
	user := s.createTestUser()
	tenant := s.createTestTenant()

	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(user, nil)
	s.tenantRepo.On("GetByID", mock.Anything, s.testTenantID).Return(tenant, nil)
}

// TearDownTest cleans up after each test
func (s *AuthTestSuite) TearDownTest() {
	// Verify that all expected calls were made
	s.userRepo.AssertExpectations(s.T())
	s.tenantRepo.AssertExpectations(s.T())
}

// TestGenerateToken tests token generation functionality
func (s *AuthTestSuite) TestGenerateToken() {
	// Generate a token
	token, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, s.testRoles, time.Hour)
	
	// Assert token was generated successfully
	assert.NoError(s.T(), err, "Token generation should not fail")
	assert.NotEmpty(s.T(), token, "Generated token should not be empty")

	// Validate the generated token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), token)
	assert.NoError(s.T(), err, "Generated token should be valid")
	assert.Equal(s.T(), s.testTenantID, tenantID, "Token should contain correct tenant ID")
	assert.ElementsMatch(s.T(), s.testRoles, roles, "Token should contain correct roles")
}

// TestValidateToken tests token validation functionality
func (s *AuthTestSuite) TestValidateToken() {
	// Generate a valid token
	token, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Validate the token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), token)
	
	// Assert validation succeeds
	assert.NoError(s.T(), err, "Token validation should not fail")
	assert.Equal(s.T(), s.testTenantID, tenantID, "Extracted tenant ID should match")
	assert.ElementsMatch(s.T(), s.testRoles, roles, "Extracted roles should match")
}

// TestValidateToken_Invalid tests validation of invalid tokens
func (s *AuthTestSuite) TestValidateToken_Invalid() {
	// Test with empty token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), "")
	assert.Error(s.T(), err, "Empty token should fail validation")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty for invalid token")
	assert.Empty(s.T(), roles, "Roles should be empty for invalid token")

	// Test with malformed token
	tenantID, roles, err = s.authService.ValidateToken(context.Background(), "not.a.valid.jwt")
	assert.Error(s.T(), err, "Malformed token should fail validation")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty for invalid token")
	assert.Empty(s.T(), roles, "Roles should be empty for invalid token")

	// Test with expired token
	expiredToken, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, s.testRoles, -time.Hour)
	assert.NoError(s.T(), err, "Generating expired token should not fail")
	tenantID, roles, err = s.authService.ValidateToken(context.Background(), expiredToken)
	assert.Error(s.T(), err, "Expired token should fail validation")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty for expired token")
	assert.Empty(s.T(), roles, "Roles should be empty for expired token")

	// Test with token signed by different key
	// Since we don't have access to another key, we'll just ensure a modified token fails
	token, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")
	modifiedToken := token[:len(token)-5] + "12345" // Tamper with the signature
	tenantID, roles, err = s.authService.ValidateToken(context.Background(), modifiedToken)
	assert.Error(s.T(), err, "Modified token should fail validation")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty for invalid token")
	assert.Empty(s.T(), roles, "Roles should be empty for invalid token")
}

// TestValidateToken_UserNotFound tests validation when user is not found
func (s *AuthTestSuite) TestValidateToken_UserNotFound() {
	// Generate a valid token
	token, err := s.authService.GenerateToken(context.Background(), "unknown-user", s.testTenantID, s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Mock user repository to return not found error
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, "unknown-user", s.testTenantID).Return(nil, errors.NewResourceNotFoundError("user not found"))
	s.tenantRepo.On("GetByID", mock.Anything, s.testTenantID).Return(s.createTestTenant(), nil)
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Validate the token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), token)
	
	// Assert validation fails
	assert.Error(s.T(), err, "Validation should fail for unknown user")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty when validation fails")
	assert.Empty(s.T(), roles, "Roles should be empty when validation fails")
}

// TestValidateToken_UserInactive tests validation when user is inactive
func (s *AuthTestSuite) TestValidateToken_UserInactive() {
	// Generate a valid token
	token, err := s.authService.GenerateToken(context.Background(), "inactive-user", s.testTenantID, s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Create inactive user
	inactiveUser := s.createTestUser()
	inactiveUser.ID = "inactive-user"
	inactiveUser.Status = models.UserStatusInactive

	// Set up mocks
	s.userRepo = new(mockUserRepository)
	s.tenantRepo = new(mockTenantRepository)
	s.userRepo.On("GetByID", mock.Anything, "inactive-user", s.testTenantID).Return(inactiveUser, nil)
	s.tenantRepo.On("GetByID", mock.Anything, s.testTenantID).Return(s.createTestTenant(), nil)
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Validate the token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), token)
	
	// Assert validation fails
	assert.Error(s.T(), err, "Validation should fail for inactive user")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty when validation fails")
	assert.Empty(s.T(), roles, "Roles should be empty when validation fails")
}

// TestValidateToken_TenantNotFound tests validation when tenant is not found
func (s *AuthTestSuite) TestValidateToken_TenantNotFound() {
	// Generate a valid token
	token, err := s.authService.GenerateToken(context.Background(), s.testUserID, "unknown-tenant", s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Set up mocks
	s.userRepo = new(mockUserRepository)
	s.tenantRepo = new(mockTenantRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, "unknown-tenant").Return(s.createTestUser(), nil)
	s.tenantRepo.On("GetByID", mock.Anything, "unknown-tenant").Return(nil, errors.NewResourceNotFoundError("tenant not found"))
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Validate the token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), token)
	
	// Assert validation fails
	assert.Error(s.T(), err, "Validation should fail for unknown tenant")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty when validation fails")
	assert.Empty(s.T(), roles, "Roles should be empty when validation fails")
}

// TestValidateToken_TenantInactive tests validation when tenant is inactive
func (s *AuthTestSuite) TestValidateToken_TenantInactive() {
	// Generate a valid token
	token, err := s.authService.GenerateToken(context.Background(), s.testUserID, "inactive-tenant", s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Create inactive tenant
	inactiveTenant := s.createTestTenant()
	inactiveTenant.ID = "inactive-tenant"
	inactiveTenant.Status = "inactive"

	// Set up mocks
	s.userRepo = new(mockUserRepository)
	s.tenantRepo = new(mockTenantRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, "inactive-tenant").Return(s.createTestUser(), nil)
	s.tenantRepo.On("GetByID", mock.Anything, "inactive-tenant").Return(inactiveTenant, nil)
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Validate the token
	tenantID, roles, err := s.authService.ValidateToken(context.Background(), token)
	
	// Assert validation fails
	assert.Error(s.T(), err, "Validation should fail for inactive tenant")
	assert.Empty(s.T(), tenantID, "Tenant ID should be empty when validation fails")
	assert.Empty(s.T(), roles, "Roles should be empty when validation fails")
}

// TestRefreshToken tests refresh token functionality
func (s *AuthTestSuite) TestRefreshToken() {
	// Generate a refresh token
	refreshToken, err := s.authService.GenerateRefreshToken(context.Background(), s.testUserID, s.testTenantID, time.Hour)
	assert.NoError(s.T(), err, "Refresh token generation should not fail")

	// Refresh the token
	newRefreshToken, err := s.authService.RefreshToken(context.Background(), refreshToken)
	
	// Assert refresh succeeds
	assert.NoError(s.T(), err, "Token refresh should not fail")
	assert.NotEmpty(s.T(), newRefreshToken, "New refresh token should not be empty")
	assert.NotEqual(s.T(), refreshToken, newRefreshToken, "New refresh token should be different from old one")
}

// TestVerifyPermission tests permission verification functionality
func (s *AuthTestSuite) TestVerifyPermission() {
	// Set up user with specific roles
	user := s.createTestUser()
	user.Roles = []string{"reader", "contributor"}
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(user, nil)
	
	var err error
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Test read permission (all users have read permission)
	hasPermission, err := s.authService.VerifyPermission(context.Background(), s.testUserID, s.testTenantID, auth.PermissionRead)
	assert.NoError(s.T(), err, "Permission verification should not fail")
	assert.True(s.T(), hasPermission, "User should have read permission")

	// Test write permission (contributor role has write permission)
	hasPermission, err = s.authService.VerifyPermission(context.Background(), s.testUserID, s.testTenantID, auth.PermissionWrite)
	assert.NoError(s.T(), err, "Permission verification should not fail")
	assert.True(s.T(), hasPermission, "User with contributor role should have write permission")

	// Test delete permission (contributor role doesn't have delete permission)
	hasPermission, err = s.authService.VerifyPermission(context.Background(), s.testUserID, s.testTenantID, auth.PermissionDelete)
	assert.NoError(s.T(), err, "Permission verification should not fail")
	assert.False(s.T(), hasPermission, "User with contributor role should not have delete permission")

	// Test manage_folders permission (only admin role has this permission)
	hasPermission, err = s.authService.VerifyPermission(context.Background(), s.testUserID, s.testTenantID, auth.PermissionManageFolders)
	assert.NoError(s.T(), err, "Permission verification should not fail")
	assert.False(s.T(), hasPermission, "User without admin role should not have manage_folders permission")

	// Add admin role to user and test again
	user.Roles = append(user.Roles, "administrator")
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(user, nil)
	
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Test manage_folders permission again with admin role
	hasPermission, err = s.authService.VerifyPermission(context.Background(), s.testUserID, s.testTenantID, auth.PermissionManageFolders)
	assert.NoError(s.T(), err, "Permission verification should not fail")
	assert.True(s.T(), hasPermission, "User with admin role should have manage_folders permission")
}

// TestVerifyResourceAccess tests resource access verification functionality
func (s *AuthTestSuite) TestVerifyResourceAccess() {
	// Set up user with specific roles
	user := s.createTestUser()
	user.Roles = []string{"reader", "contributor"}
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(user, nil)
	
	var err error
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Test read access to document
	hasAccess, err := s.authService.VerifyResourceAccess(context.Background(), s.testUserID, s.testTenantID, auth.ResourceTypeDocument, "doc-123", "read")
	assert.NoError(s.T(), err, "Resource access verification should not fail")
	assert.True(s.T(), hasAccess, "User should have read access to document")

	// Test write access to document
	hasAccess, err = s.authService.VerifyResourceAccess(context.Background(), s.testUserID, s.testTenantID, auth.ResourceTypeDocument, "doc-123", "write")
	assert.NoError(s.T(), err, "Resource access verification should not fail")
	assert.True(s.T(), hasAccess, "User with contributor role should have write access to document")

	// Test delete access to document
	hasAccess, err = s.authService.VerifyResourceAccess(context.Background(), s.testUserID, s.testTenantID, auth.ResourceTypeDocument, "doc-123", "delete")
	assert.NoError(s.T(), err, "Resource access verification should not fail")
	assert.False(s.T(), hasAccess, "User with contributor role should not have delete access to document")

	// Test manage_folders access to folder
	hasAccess, err = s.authService.VerifyResourceAccess(context.Background(), s.testUserID, s.testTenantID, auth.ResourceTypeFolder, "folder-123", "manage_folders")
	assert.NoError(s.T(), err, "Resource access verification should not fail")
	assert.False(s.T(), hasAccess, "User without admin role should not have manage_folders access to folder")
}

// TestVerifyTenantAccess tests tenant access verification functionality
func (s *AuthTestSuite) TestVerifyTenantAccess() {
	// Test with matching tenant ID
	user := s.createTestUser()
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(user, nil)
	
	var err error
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	hasAccess, err := s.authService.VerifyTenantAccess(context.Background(), s.testUserID, s.testTenantID)
	assert.NoError(s.T(), err, "Tenant access verification should not fail")
	assert.True(s.T(), hasAccess, "User should have access to their own tenant")

	// Test with non-matching tenant ID
	otherTenantUser := s.createTestUser()
	otherTenantUser.TenantID = "different-tenant" // This user belongs to a different tenant
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, "different-tenant").Return(otherTenantUser, nil)
	
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	hasAccess, err = s.authService.VerifyTenantAccess(context.Background(), s.testUserID, "different-tenant")
	assert.NoError(s.T(), err, "Tenant access verification should not fail")
	assert.False(s.T(), hasAccess, "User should not have access to a different tenant")
}

// TestAuthMiddleware tests the authentication middleware
func (s *AuthTestSuite) TestAuthMiddleware() {
	// Set up Gin test router with auth middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware(s.authService))

	// Add a test handler
	router.GET("/test", func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		tenantID := middleware.GetTenantID(c)
		roles := middleware.GetUserRoles(c)

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"tenant_id": tenantID,
			"roles": roles,
		})
	})

	// Generate a valid token
	token, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, s.testRoles, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Test with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code, "Request with valid token should succeed")
	assert.Contains(s.T(), w.Body.String(), s.testUserID, "Response should include user ID")
	assert.Contains(s.T(), w.Body.String(), s.testTenantID, "Response should include tenant ID")

	// Test with missing Authorization header
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code, "Request without token should fail")

	// Test with invalid token format
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code, "Request with invalid token format should fail")

	// Test with invalid token
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.format")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code, "Request with invalid token should fail")
}

// TestRequireRole tests the role requirement middleware
func (s *AuthTestSuite) TestRequireRole() {
	// Set up Gin test router with auth and role middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware(s.authService))

	// Add test handlers with role requirements
	router.GET("/admin", middleware.RequireRole("administrator"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.GET("/contributor", middleware.RequireRole("contributor"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Generate tokens for different roles
	adminToken, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, []string{"administrator"}, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	contributorToken, err := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, []string{"contributor"}, time.Hour)
	assert.NoError(s.T(), err, "Token generation should not fail")

	// Set up user with admin role
	adminUser := s.createTestUser()
	adminUser.Roles = []string{"administrator"}
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(adminUser, nil)
	
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Test admin endpoint with admin role
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code, "Admin should be able to access admin endpoint")

	// Set up user with contributor role
	contribUser := s.createTestUser()
	contribUser.Roles = []string{"contributor"}
	
	s.userRepo = new(mockUserRepository)
	s.userRepo.On("GetByID", mock.Anything, s.testUserID, s.testTenantID).Return(contribUser, nil)
	
	s.authService, err = jwtauth.NewJWTService(s.userRepo, s.tenantRepo, s.jwtConfig)
	assert.NoError(s.T(), err, "Failed to create JWT auth service")

	// Test admin endpoint with contributor role
	req = httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+contributorToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusForbidden, w.Code, "Contributor should not be able to access admin endpoint")

	// Test contributor endpoint with contributor role
	req = httptest.NewRequest("GET", "/contributor", nil)
	req.Header.Set("Authorization", "Bearer "+contributorToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code, "Contributor should be able to access contributor endpoint")
}

// createTestUser is a helper method to create a test user
func (s *AuthTestSuite) createTestUser() *models.User {
	user := &models.User{
		ID:        s.testUserID,
		TenantID:  s.testTenantID,
		Username:  "testuser",
		Email:     "test@example.com",
		Status:    models.UserStatusActive,
		Roles:     s.testRoles,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return user
}

// createTestTenant is a helper method to create a test tenant
func (s *AuthTestSuite) createTestTenant() *models.Tenant {
	tenant := &models.Tenant{
		ID:      s.testTenantID,
		Name:    "Test Tenant",
		Status:  "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return tenant
}

// generateTestToken is a helper method to generate a test token
func (s *AuthTestSuite) generateTestToken() string {
	token, _ := s.authService.GenerateToken(context.Background(), s.testUserID, s.testTenantID, s.testRoles, time.Hour)
	return token
}

// TestAuthSuite runs the auth test suite
func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}