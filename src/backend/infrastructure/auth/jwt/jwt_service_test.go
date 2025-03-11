package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/golang-jwt/jwt/v5"

	"../../../domain/models"
	"../../../domain/services"
	"../../../pkg/config"
	"../../../pkg/errors"
)

// Test RSA keys in PEM format
const testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEU/wT8RDtn
SgFEZOQpHEgQ7JL38xUfU0Y3g6aYw9QT0hJ7mCpz9Er5qLaMXJwZxzHzAahlfA0i
cqabvJOMvQtzD6uQv6wPEyZtDTWiQi9AXwBpHssPnpYGIn20ZZuNlX2BrClciHhC
PUIIZOQn/MmqTD31jSyjoQoV7MhhMTATKJx2XrHhR+1DcKJzQBSTAGnpYVaqpsAR
ap+nwRipr3nUTuxyGohBTSmjJ2usSeQXHI3bODIRe1AuTyHceAbewn8b462yEWKA
Rdpd9AjQW5SIVPfdsz5B6GlYQ5LdYKtznTuy7wIDAQABAoIBAQCwia1k7+4oZ3Hd
mJyS1Ht7MCDsFoLp4QKPDJ5wJwpC7u5mA8l1fGdOs9NrYbI/amocDiJZ5AvPBx5V
15+CnXGZZ7uKqkjGS7EnKmA5xP0I9Hi94HGrZQHUmCDhboGJyCuEddnYP5OIrV/R
v9zLSWCP9fujlLVgrzuMtZyxUHQ5+Zf0XKS+BQmf6k3fJGDX9yTVGkRnzqj9iIo9
3BvdogKFLYZukYMmUYEwlcRe3mCb2WO9vXWo4odxoRdSHtZUdC8nQzfX5JL7yKkw
r6+QjKS9tK3GYEmTqQU/XvK0H0Ek7Sh3ODG/vUYzRKRJgcC965I3XdDrUJOYFQIk
V2tYDBJhAoGBAPe9AHN86/Ldt8QvqabeXzKo3UtSypRXQc3pIGi9JhMkrKnYZfY1
6sE5CowEKNV1SC3gH7YJoZTMnUdJISOETIoA4MpGUlbL3ZmKC1yVCIaitW+UIOqj
yfRQlV1qlVxyEK4+JPZKuJeGOz4WnDpYzlbzEuhxu+xtUhDnj/cVFL1/AoGBAOnf
sLzXkUOZlK8Rbmd8JgZFaJlZ8roE3gOK/VYJa3LMgW1I9lU7mixlHUwbURxYMKgE
U9R3J5yUdpwRD+jzU8l0NHudTtXdWwNKjkeOPzLPFQofQMXtCf/ivn1A0NFjA2Av
J/O1+uLjmcLH41oLtU+9oIl6UvSrPTCCEJsVKjBRAoGAVxCkJllDsNkzLXN5lIoI
vRPCpHEUdLNO4h7rnHU4rBgR+K4bNEGhUvTbhvn0PczIbSjKxhgJ8Ct9pP+vHGoy
lLwSuZIJSx8N6SznZp8sYV0J/0HHgCKQGbmTLdqX2O+gotj3tH6u1M3OvzTk1hbI
AKngtLsP2jDdZySY3ZE4KDsCgYEAn9XzNSVhq4jQDW1uskqxjZxB6Gsw6Wph1NU7
zJkyv4xOlfZ8lBAl9yzhZnLtDEMAS3WbZjnBrFFDgV9qm/yzjFn0CxDYjLwA7kxp
CV2vFJYLSvpnP2LW7+pnOVJLsG7mTNrvx72KG7INPyzCgZhECIiyW/6FKLBCbFbw
atVcxvECgYA6297Vsn0dR42c3O/jQZpkCJ8J1D0lLJ4vUoKo3UtdZLQvVwxT68vR
PpH/mK9LStJtU5n8jbfMPEzdZI2UZiIrKnxhYbR+6G8g0XcuFQkYKZWLmPHcZUzB
j7BDJ6GvLg1Xgj5LqSed5ObSgKMjYYvHKDzku2CQJAYZgmhYrEyP2g==
-----END RSA PRIVATE KEY-----`

const testPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41
fGnJm6gOdrj8ym3rFkEU/wT8RDtnSgFEZOQpHEgQ7JL38xUfU0Y3g6aYw9QT0hJ7
mCpz9Er5qLaMXJwZxzHzAahlfA0icqabvJOMvQtzD6uQv6wPEyZtDTWiQi9AXwBp
HssPnpYGIn20ZZuNlX2BrClciHhCPUIIZOQn/MmqTD31jSyjoQoV7MhhMTATKJx2
XrHhR+1DcKJzQBSTAGnpYVaqpsARap+nwRipr3nUTuxyGohBTSmjJ2usSeQXHI3b
ODIRe1AuTyHceAbewn8b462yEWKARdpd9AjQW5SIVPfdsz5B6GlYQ5LdYKtznTuy
7wIDAQAB
-----END PUBLIC KEY-----`

// must is a helper function that panics if err is non-nil
// It's used for operations that should never fail during testing
func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// Mock UserRepository for testing
type MockUserRepository struct {
	mock.Mock
}

// GetByID mocks the GetByID method of UserRepository
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetByUsernameOrEmail mocks the GetByUsernameOrEmail method of UserRepository
func (m *MockUserRepository) GetByUsernameOrEmail(ctx context.Context, tenantID, usernameOrEmail string) (*models.User, error) {
	args := m.Called(ctx, tenantID, usernameOrEmail)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Save mocks the Save method of UserRepository
func (m *MockUserRepository) Save(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

// Mock TenantRepository for testing
type MockTenantRepository struct {
	mock.Mock
}

// GetByID mocks the GetByID method of TenantRepository
func (m *MockTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Tenant), args.Error(1)
}

// Save mocks the Save method of TenantRepository
func (m *MockTenantRepository) Save(ctx context.Context, tenant *models.Tenant) error {
	return m.Called(ctx, tenant).Error(0)
}

// JWTServiceSuite is a test suite for JWTService
type JWTServiceSuite struct {
	suite.Suite
	jwtService services.AuthService
	jwtConfig  config.JWTConfig
	ctx        context.Context
}

// TestJWTServiceSuite runs the JWT service test suite
func TestJWTServiceSuite(t *testing.T) {
	suite.Run(t, new(JWTServiceSuite))
}

// SetupTest sets up the test environment before each test
func (s *JWTServiceSuite) SetupTest() {
	s.ctx = context.Background()
	
	// Set up JWT configuration with test keys
	s.jwtConfig = config.JWTConfig{
		PrivateKey: testPrivateKey,
		PublicKey:  testPublicKey,
		Algorithm:  "RS256",
		Issuer:     "document-management-platform-test",
		ExpirationTime: "1h",
	}
	
	// Create a new JWT service
	var err error
	s.jwtService, err = NewJWTService(s.jwtConfig)
	s.Require().NoError(err)
	s.Require().NotNil(s.jwtService)
	
	// Set default token expiration
	s.jwtService.SetTokenExpiration(time.Hour)
	s.jwtService.SetRefreshTokenExpiration(24 * time.Hour)
}

// TestNewJWTService tests the creation of a new JWT service
func (s *JWTServiceSuite) TestNewJWTService() {
	// Test with valid configuration
	service, err := NewJWTService(s.jwtConfig)
	s.Require().NoError(err)
	s.Require().NotNil(service)
	
	// Test with invalid private key
	invalidConfig := s.jwtConfig
	invalidConfig.PrivateKey = "invalid-key"
	service, err = NewJWTService(invalidConfig)
	s.Require().Error(err)
	s.Require().Nil(service)
	s.True(errors.IsValidationError(err))
	
	// Test with invalid public key
	invalidConfig = s.jwtConfig
	invalidConfig.PublicKey = "invalid-key"
	service, err = NewJWTService(invalidConfig)
	s.Require().Error(err)
	s.Require().Nil(service)
	s.True(errors.IsValidationError(err))
}

// TestAuthenticate tests the Authenticate method
func (s *JWTServiceSuite) TestAuthenticate() {
	// Create test tenant and user
	tenant := s.createTestTenant("tenant-123")
	user := s.createTestUser("user-123", "tenant-123", []string{"contributor"})
	
	// Mock repository calls
	userRepo := new(MockUserRepository)
	tenantRepo := new(MockTenantRepository)
	
	// Set up mock expectations
	tenantRepo.On("GetByID", mock.Anything, "tenant-123").Return(tenant, nil)
	tenantRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("tenant not found"))
	
	userRepo.On("GetByUsernameOrEmail", mock.Anything, "tenant-123", "testuser").Return(user, nil)
	userRepo.On("GetByUsernameOrEmail", mock.Anything, "tenant-123", "nonexistent").Return(nil, errors.NewResourceNotFoundError("user not found"))
	
	// Create JWT service with mocked repositories
	jwtService := &JWTService{
		config:      s.jwtConfig,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		privateKey:  must(jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))),
		publicKey:   must(jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))),
		tokenExp:    time.Hour,
		refreshExp:  24 * time.Hour,
	}
	
	// Test successful authentication
	token, err := jwtService.Authenticate(s.ctx, "tenant-123", "testuser", "password")
	s.Require().NoError(err)
	s.Require().NotEmpty(token)
	
	// Test tenant not found
	token, err = jwtService.Authenticate(s.ctx, "non-existent", "testuser", "password")
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.Empty(token)
	
	// Test user not found
	token, err = jwtService.Authenticate(s.ctx, "tenant-123", "nonexistent", "password")
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.Empty(token)
	
	// Test wrong password
	mockUser := s.createTestUser("user-456", "tenant-123", []string{"contributor"})
	// Override the SetPassword behavior to ensure wrong password verification
	mockUser.SetPassword("wrongpassword")
	userRepo.On("GetByUsernameOrEmail", mock.Anything, "tenant-123", "wrongpass").Return(mockUser, nil)
	
	token, err = jwtService.Authenticate(s.ctx, "tenant-123", "wrongpass", "password")
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(token)
	
	// Test inactive user
	inactiveUser := s.createTestUser("user-789", "tenant-123", []string{"contributor"})
	inactiveUser.Status = models.UserStatusInactive
	userRepo.On("GetByUsernameOrEmail", mock.Anything, "tenant-123", "inactive").Return(inactiveUser, nil)
	
	token, err = jwtService.Authenticate(s.ctx, "tenant-123", "inactive", "password")
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(token)
	
	// Test suspended user
	suspendedUser := s.createTestUser("user-101", "tenant-123", []string{"contributor"})
	suspendedUser.Status = models.UserStatusSuspended
	userRepo.On("GetByUsernameOrEmail", mock.Anything, "tenant-123", "suspended").Return(suspendedUser, nil)
	
	token, err = jwtService.Authenticate(s.ctx, "tenant-123", "suspended", "password")
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(token)
	
	// Verify all expectations were met
	userRepo.AssertExpectations(s.T())
	tenantRepo.AssertExpectations(s.T())
}

// TestValidateToken tests the ValidateToken method
func (s *JWTServiceSuite) TestValidateToken() {
	// Create test user and tenant
	tenant := s.createTestTenant("tenant-123")
	user := s.createTestUser("user-123", "tenant-123", []string{"contributor"})
	
	// Mock repository calls
	userRepo := new(MockUserRepository)
	tenantRepo := new(MockTenantRepository)
	
	userRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil)
	userRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("user not found"))
	tenantRepo.On("GetByID", mock.Anything, "tenant-123").Return(tenant, nil)
	
	// Create JWT service with mocked repositories
	jwtService := &JWTService{
		config:      s.jwtConfig,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		privateKey:  must(jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))),
		publicKey:   must(jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))),
		tokenExp:    time.Hour,
		refreshExp:  24 * time.Hour,
	}
	
	// Generate a valid token
	token, err := jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().NoError(err)
	s.Require().NotEmpty(token)
	
	// Test valid token
	userID, roles, err := jwtService.ValidateToken(s.ctx, token)
	s.Require().NoError(err)
	s.Equal("user-123", userID)
	s.ElementsMatch([]string{"contributor"}, roles)
	
	// Test invalid token format
	userID, roles, err = jwtService.ValidateToken(s.ctx, "invalid-token-format")
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(userID)
	s.Nil(roles)
	
	// Test expired token
	expiredToken, err := jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, -time.Hour)
	s.Require().NoError(err)
	
	userID, roles, err = jwtService.ValidateToken(s.ctx, expiredToken)
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(userID)
	s.Nil(roles)
	
	// Test token with non-existent user
	nonExistentUserToken, err := jwtService.GenerateToken(s.ctx, "non-existent", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().NoError(err)
	
	userID, roles, err = jwtService.ValidateToken(s.ctx, nonExistentUserToken)
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.Empty(userID)
	s.Nil(roles)
	
	// Test token with inactive user
	inactiveUser := s.createTestUser("inactive-user", "tenant-123", []string{"contributor"})
	inactiveUser.Status = models.UserStatusInactive
	userRepo.On("GetByID", mock.Anything, "inactive-user").Return(inactiveUser, nil)
	
	inactiveUserToken, err := jwtService.GenerateToken(s.ctx, "inactive-user", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().NoError(err)
	
	userID, roles, err = jwtService.ValidateToken(s.ctx, inactiveUserToken)
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(userID)
	s.Nil(roles)
	
	// Verify all expectations were met
	userRepo.AssertExpectations(s.T())
	tenantRepo.AssertExpectations(s.T())
}

// TestRefreshToken tests the RefreshToken method
func (s *JWTServiceSuite) TestRefreshToken() {
	// Create test user and tenant
	tenant := s.createTestTenant("tenant-123")
	user := s.createTestUser("user-123", "tenant-123", []string{"contributor"})
	
	// Mock repository calls
	userRepo := new(MockUserRepository)
	tenantRepo := new(MockTenantRepository)
	
	userRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil)
	userRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("user not found"))
	tenantRepo.On("GetByID", mock.Anything, "tenant-123").Return(tenant, nil)
	
	// Create JWT service with mocked repositories
	jwtService := &JWTService{
		config:      s.jwtConfig,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		privateKey:  must(jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))),
		publicKey:   must(jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))),
		tokenExp:    time.Hour,
		refreshExp:  24 * time.Hour,
	}
	
	// Generate a valid refresh token
	refreshToken, err := jwtService.GenerateRefreshToken(s.ctx, "user-123", "tenant-123", 24*time.Hour)
	s.Require().NoError(err)
	s.Require().NotEmpty(refreshToken)
	
	// Test valid refresh token
	newRefreshToken, err := jwtService.RefreshToken(s.ctx, refreshToken)
	s.Require().NoError(err)
	s.Require().NotEmpty(newRefreshToken)
	s.NotEqual(refreshToken, newRefreshToken) // New token should be different
	
	// Test invalid refresh token format
	newRefreshToken, err = jwtService.RefreshToken(s.ctx, "invalid-token-format")
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(newRefreshToken)
	
	// Test expired refresh token
	expiredToken, err := jwtService.GenerateRefreshToken(s.ctx, "user-123", "tenant-123", -time.Hour)
	s.Require().NoError(err)
	
	newRefreshToken, err = jwtService.RefreshToken(s.ctx, expiredToken)
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(newRefreshToken)
	
	// Test refresh token with non-existent user
	nonExistentUserToken, err := jwtService.GenerateRefreshToken(s.ctx, "non-existent", "tenant-123", time.Hour)
	s.Require().NoError(err)
	
	newRefreshToken, err = jwtService.RefreshToken(s.ctx, nonExistentUserToken)
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.Empty(newRefreshToken)
	
	// Test refresh token with inactive user
	inactiveUser := s.createTestUser("inactive-user", "tenant-123", []string{"contributor"})
	inactiveUser.Status = models.UserStatusInactive
	userRepo.On("GetByID", mock.Anything, "inactive-user").Return(inactiveUser, nil)
	
	inactiveUserToken, err := jwtService.GenerateRefreshToken(s.ctx, "inactive-user", "tenant-123", time.Hour)
	s.Require().NoError(err)
	
	newRefreshToken, err = jwtService.RefreshToken(s.ctx, inactiveUserToken)
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(newRefreshToken)
	
	// Test refresh token with wrong token type
	accessToken, err := jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().NoError(err)
	
	newRefreshToken, err = jwtService.RefreshToken(s.ctx, accessToken)
	s.Require().Error(err)
	s.True(errors.IsAuthenticationError(err))
	s.Empty(newRefreshToken)
	
	// Verify all expectations were met
	userRepo.AssertExpectations(s.T())
	tenantRepo.AssertExpectations(s.T())
}

// TestInvalidateToken tests the InvalidateToken method
func (s *JWTServiceSuite) TestInvalidateToken() {
	// Since JWT is stateless, this is essentially a no-op, but we should test it anyway
	token, err := s.jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().NoError(err)
	s.Require().NotEmpty(token)
	
	// Invalidate token should not return an error
	err = s.jwtService.InvalidateToken(s.ctx, token)
	s.Require().NoError(err)
	
	// Invalid token should still not return an error
	err = s.jwtService.InvalidateToken(s.ctx, "invalid-token")
	s.Require().NoError(err)
}

// TestVerifyPermission tests the VerifyPermission method
func (s *JWTServiceSuite) TestVerifyPermission() {
	// Create users with different roles
	readerUser := s.createTestUser("reader", "tenant-123", []string{"reader"})
	contributorUser := s.createTestUser("contributor", "tenant-123", []string{"contributor"})
	editorUser := s.createTestUser("editor", "tenant-123", []string{"editor"})
	adminUser := s.createTestUser("admin", "tenant-123", []string{"administrator"})
	
	// Mock repository calls
	userRepo := new(MockUserRepository)
	tenantRepo := new(MockTenantRepository)
	
	userRepo.On("GetByID", mock.Anything, "reader").Return(readerUser, nil)
	userRepo.On("GetByID", mock.Anything, "contributor").Return(contributorUser, nil)
	userRepo.On("GetByID", mock.Anything, "editor").Return(editorUser, nil)
	userRepo.On("GetByID", mock.Anything, "admin").Return(adminUser, nil)
	userRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("user not found"))
	
	tenant := s.createTestTenant("tenant-123")
	tenantRepo.On("GetByID", mock.Anything, "tenant-123").Return(tenant, nil)
	tenantRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("tenant not found"))
	
	// Create JWT service with mocked repositories
	jwtService := &JWTService{
		config:      s.jwtConfig,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		privateKey:  must(jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))),
		publicKey:   must(jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))),
		tokenExp:    time.Hour,
		refreshExp:  24 * time.Hour,
	}
	
	// Test read permission
	// All roles should have read permission
	hasPermission, err := jwtService.VerifyPermission(s.ctx, "reader", "tenant-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "contributor", "tenant-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "editor", "tenant-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "admin", "tenant-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	// Test write permission
	// Reader should not have write permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "reader", "tenant-123", services.PermissionWrite)
	s.Require().NoError(err)
	s.False(hasPermission)
	
	// Contributor, editor, and admin should have write permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "contributor", "tenant-123", services.PermissionWrite)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "editor", "tenant-123", services.PermissionWrite)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "admin", "tenant-123", services.PermissionWrite)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	// Test delete permission
	// Reader and contributor should not have delete permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "reader", "tenant-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.False(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "contributor", "tenant-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.False(hasPermission)
	
	// Editor and admin should have delete permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "editor", "tenant-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "admin", "tenant-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	// Test manage_folders permission
	// Reader, contributor, and editor should not have manage_folders permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "reader", "tenant-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.False(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "contributor", "tenant-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.False(hasPermission)
	
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "editor", "tenant-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.False(hasPermission)
	
	// Admin should have manage_folders permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "admin", "tenant-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.True(hasPermission)
	
	// Test with non-existent user
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "non-existent", "tenant-123", services.PermissionRead)
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.False(hasPermission)
	
	// Test with non-existent tenant
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "reader", "non-existent", services.PermissionRead)
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.False(hasPermission)
	
	// Test with invalid permission
	hasPermission, err = jwtService.VerifyPermission(s.ctx, "reader", "tenant-123", "invalid-permission")
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.False(hasPermission)
	
	// Verify all expectations were met
	userRepo.AssertExpectations(s.T())
	tenantRepo.AssertExpectations(s.T())
}

// TestVerifyResourceAccess tests the VerifyResourceAccess method
func (s *JWTServiceSuite) TestVerifyResourceAccess() {
	// Create users with different roles
	readerUser := s.createTestUser("reader", "tenant-123", []string{"reader"})
	contributorUser := s.createTestUser("contributor", "tenant-123", []string{"contributor"})
	editorUser := s.createTestUser("editor", "tenant-123", []string{"editor"})
	adminUser := s.createTestUser("admin", "tenant-123", []string{"administrator"})
	
	// Mock repository calls
	userRepo := new(MockUserRepository)
	tenantRepo := new(MockTenantRepository)
	
	userRepo.On("GetByID", mock.Anything, "reader").Return(readerUser, nil)
	userRepo.On("GetByID", mock.Anything, "contributor").Return(contributorUser, nil)
	userRepo.On("GetByID", mock.Anything, "editor").Return(editorUser, nil)
	userRepo.On("GetByID", mock.Anything, "admin").Return(adminUser, nil)
	userRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("user not found"))
	
	tenant := s.createTestTenant("tenant-123")
	tenantRepo.On("GetByID", mock.Anything, "tenant-123").Return(tenant, nil)
	tenantRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("tenant not found"))
	
	// Create JWT service with mocked repositories
	jwtService := &JWTService{
		config:      s.jwtConfig,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		privateKey:  must(jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))),
		publicKey:   must(jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))),
		tokenExp:    time.Hour,
		refreshExp:  24 * time.Hour,
	}
	
	// Test document resource access
	// All users should have read access to documents
	hasAccess, err := jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "contributor", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasAccess)
	
	// Only contributor, editor, and admin should have write access to documents
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionWrite)
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "contributor", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionWrite)
	s.Require().NoError(err)
	s.True(hasAccess)
	
	// Only editor and admin should have delete access to documents
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "contributor", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "editor", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionDelete)
	s.Require().NoError(err)
	s.True(hasAccess)
	
	// Test folder resource access
	// All users should have read access to folders
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", services.ResourceTypeFolder, "folder-123", services.PermissionRead)
	s.Require().NoError(err)
	s.True(hasAccess)
	
	// Only admin should have manage_folders access
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", services.ResourceTypeFolder, "folder-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "contributor", "tenant-123", services.ResourceTypeFolder, "folder-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "editor", "tenant-123", services.ResourceTypeFolder, "folder-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "admin", "tenant-123", services.ResourceTypeFolder, "folder-123", services.PermissionManageFolders)
	s.Require().NoError(err)
	s.True(hasAccess)
	
	// Test with non-existent user
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "non-existent", "tenant-123", services.ResourceTypeDocument, "doc-123", services.PermissionRead)
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.False(hasAccess)
	
	// Test with non-existent tenant
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "non-existent", services.ResourceTypeDocument, "doc-123", services.PermissionRead)
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.False(hasAccess)
	
	// Test with invalid resource type
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", "invalid-resource", "resource-123", services.PermissionRead)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.False(hasAccess)
	
	// Test with invalid permission
	hasAccess, err = jwtService.VerifyResourceAccess(s.ctx, "reader", "tenant-123", services.ResourceTypeDocument, "doc-123", "invalid-permission")
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.False(hasAccess)
	
	// Verify all expectations were met
	userRepo.AssertExpectations(s.T())
	tenantRepo.AssertExpectations(s.T())
}

// TestVerifyTenantAccess tests the VerifyTenantAccess method
func (s *JWTServiceSuite) TestVerifyTenantAccess() {
	// Create test users in different tenants
	user1 := s.createTestUser("user-123", "tenant-123", []string{"contributor"})
	user2 := s.createTestUser("user-456", "tenant-456", []string{"contributor"})
	
	// Mock repository calls
	userRepo := new(MockUserRepository)
	tenantRepo := new(MockTenantRepository)
	
	userRepo.On("GetByID", mock.Anything, "user-123").Return(user1, nil)
	userRepo.On("GetByID", mock.Anything, "user-456").Return(user2, nil)
	userRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("user not found"))
	
	tenant1 := s.createTestTenant("tenant-123")
	tenant2 := s.createTestTenant("tenant-456")
	tenantRepo.On("GetByID", mock.Anything, "tenant-123").Return(tenant1, nil)
	tenantRepo.On("GetByID", mock.Anything, "tenant-456").Return(tenant2, nil)
	tenantRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.NewResourceNotFoundError("tenant not found"))
	
	// Create JWT service with mocked repositories
	jwtService := &JWTService{
		config:      s.jwtConfig,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		privateKey:  must(jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivateKey))),
		publicKey:   must(jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))),
		tokenExp:    time.Hour,
		refreshExp:  24 * time.Hour,
	}
	
	// Test user access to correct tenant
	hasAccess, err := jwtService.VerifyTenantAccess(s.ctx, "user-123", "tenant-123")
	s.Require().NoError(err)
	s.True(hasAccess)
	
	// Test user access to incorrect tenant
	hasAccess, err = jwtService.VerifyTenantAccess(s.ctx, "user-123", "tenant-456")
	s.Require().NoError(err)
	s.False(hasAccess)
	
	hasAccess, err = jwtService.VerifyTenantAccess(s.ctx, "user-456", "tenant-123")
	s.Require().NoError(err)
	s.False(hasAccess)
	
	// Test with non-existent user
	hasAccess, err = jwtService.VerifyTenantAccess(s.ctx, "non-existent", "tenant-123")
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.False(hasAccess)
	
	// Test with non-existent tenant
	hasAccess, err = jwtService.VerifyTenantAccess(s.ctx, "user-123", "non-existent")
	s.Require().Error(err)
	s.True(errors.IsResourceNotFoundError(err))
	s.False(hasAccess)
	
	// Verify all expectations were met
	userRepo.AssertExpectations(s.T())
	tenantRepo.AssertExpectations(s.T())
}

// TestGenerateToken tests the GenerateToken method
func (s *JWTServiceSuite) TestGenerateToken() {
	// Generate a token with valid parameters
	token, err := s.jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().NoError(err)
	s.Require().NotEmpty(token)
	
	// Parse the token to verify claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		// Return the public key for verification
		return jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	})
	
	s.Require().NoError(err)
	s.Require().True(parsedToken.Valid)
	
	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	s.Require().True(ok)
	
	s.Equal("user-123", claims["sub"])
	s.Equal("tenant-123", claims["tenant_id"])
	s.Equal("access", claims["type"])
	
	// Check roles
	roles, ok := claims["roles"].([]interface{})
	s.Require().True(ok)
	s.Require().Len(roles, 1)
	s.Equal("contributor", roles[0])
	
	// Verify expiration (should be ~1 hour from now)
	exp, ok := claims["exp"].(float64)
	s.Require().True(ok)
	
	// The expiration should be roughly 1 hour from now (with some tolerance for test execution time)
	expectedExp := float64(time.Now().Add(time.Hour).Unix())
	s.InDelta(expectedExp, exp, 60) // Allow 60 seconds of tolerance
	
	// Test with empty user ID
	token, err = s.jwtService.GenerateToken(s.ctx, "", "tenant-123", []string{"contributor"}, time.Hour)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.Empty(token)
	
	// Test with empty tenant ID
	token, err = s.jwtService.GenerateToken(s.ctx, "user-123", "", []string{"contributor"}, time.Hour)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.Empty(token)
	
	// Test with invalid expiration
	token, err = s.jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, -time.Hour)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.Empty(token)
}

// TestGenerateRefreshToken tests the GenerateRefreshToken method
func (s *JWTServiceSuite) TestGenerateRefreshToken() {
	// Generate a refresh token with valid parameters
	token, err := s.jwtService.GenerateRefreshToken(s.ctx, "user-123", "tenant-123", 24*time.Hour)
	s.Require().NoError(err)
	s.Require().NotEmpty(token)
	
	// Parse the token to verify claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		// Return the public key for verification
		return jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	})
	
	s.Require().NoError(err)
	s.Require().True(parsedToken.Valid)
	
	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	s.Require().True(ok)
	
	s.Equal("user-123", claims["sub"])
	s.Equal("tenant-123", claims["tenant_id"])
	s.Equal("refresh", claims["type"])
	
	// Verify no roles in refresh token
	_, hasRoles := claims["roles"]
	s.False(hasRoles)
	
	// Verify expiration (should be ~24 hours from now)
	exp, ok := claims["exp"].(float64)
	s.Require().True(ok)
	
	// The expiration should be roughly 24 hours from now (with some tolerance for test execution time)
	expectedExp := float64(time.Now().Add(24 * time.Hour).Unix())
	s.InDelta(expectedExp, exp, 60) // Allow 60 seconds of tolerance
	
	// Test with empty user ID
	token, err = s.jwtService.GenerateRefreshToken(s.ctx, "", "tenant-123", 24*time.Hour)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.Empty(token)
	
	// Test with empty tenant ID
	token, err = s.jwtService.GenerateRefreshToken(s.ctx, "user-123", "", 24*time.Hour)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.Empty(token)
	
	// Test with invalid expiration
	token, err = s.jwtService.GenerateRefreshToken(s.ctx, "user-123", "tenant-123", -time.Hour)
	s.Require().Error(err)
	s.True(errors.IsValidationError(err))
	s.Empty(token)
}

// TestSetTokenExpiration tests the SetTokenExpiration method
func (s *JWTServiceSuite) TestSetTokenExpiration() {
	// Set a custom token expiration
	s.jwtService.SetTokenExpiration(2 * time.Hour)
	
	// Generate a token and verify the expiration
	token, err := s.jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, 0)
	s.Require().NoError(err)
	
	// Parse the token to verify claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	})
	
	s.Require().NoError(err)
	s.Require().True(parsedToken.Valid)
	
	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	s.Require().True(ok)
	
	// Verify expiration (should be ~2 hours from now)
	exp, ok := claims["exp"].(float64)
	s.Require().True(ok)
	
	expectedExp := float64(time.Now().Add(2 * time.Hour).Unix())
	s.InDelta(expectedExp, exp, 60) // Allow 60 seconds of tolerance
	
	// Test with negative duration (should not change expiration)
	s.jwtService.SetTokenExpiration(-time.Hour)
	
	// Token should still use the previous valid expiration (2 hours)
	token, err = s.jwtService.GenerateToken(s.ctx, "user-123", "tenant-123", []string{"contributor"}, 0)
	s.Require().NoError(err)
	
	parsedToken, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	})
	
	s.Require().NoError(err)
	s.Require().True(parsedToken.Valid)
	
	claims, ok = parsedToken.Claims.(jwt.MapClaims)
	s.Require().True(ok)
	
	exp, ok = claims["exp"].(float64)
	s.Require().True(ok)
	
	expectedExp = float64(time.Now().Add(2 * time.Hour).Unix())
	s.InDelta(expectedExp, exp, 60) // Allow 60 seconds of tolerance
}

// TestSetRefreshTokenExpiration tests the SetRefreshTokenExpiration method
func (s *JWTServiceSuite) TestSetRefreshTokenExpiration() {
	// Set a custom refresh token expiration
	s.jwtService.SetRefreshTokenExpiration(48 * time.Hour)
	
	// Generate a refresh token and verify the expiration
	token, err := s.jwtService.GenerateRefreshToken(s.ctx, "user-123", "tenant-123", 0)
	s.Require().NoError(err)
	
	// Parse the token to verify claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	})
	
	s.Require().NoError(err)
	s.Require().True(parsedToken.Valid)
	
	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	s.Require().True(ok)
	
	// Verify expiration (should be ~48 hours from now)
	exp, ok := claims["exp"].(float64)
	s.Require().True(ok)
	
	expectedExp := float64(time.Now().Add(48 * time.Hour).Unix())
	s.InDelta(expectedExp, exp, 60) // Allow 60 seconds of tolerance
	
	// Test with negative duration (should not change expiration)
	s.jwtService.SetRefreshTokenExpiration(-time.Hour)
	
	// Token should still use the previous valid expiration (48 hours)
	token, err = s.jwtService.GenerateRefreshToken(s.ctx, "user-123", "tenant-123", 0)
	s.Require().NoError(err)
	
	parsedToken, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(testPublicKey))
	})
	
	s.Require().NoError(err)
	s.Require().True(parsedToken.Valid)
	
	claims, ok = parsedToken.Claims.(jwt.MapClaims)
	s.Require().True(ok)
	
	exp, ok = claims["exp"].(float64)
	s.Require().True(ok)
	
	expectedExp = float64(time.Now().Add(48 * time.Hour).Unix())
	s.InDelta(expectedExp, exp, 60) // Allow 60 seconds of tolerance
}

// createTestUser creates a test user with the given ID, tenant ID, and roles
func (s *JWTServiceSuite) createTestUser(id string, tenantID string, roles []string) *models.User {
	user := &models.User{
		ID:       id,
		TenantID: tenantID,
		Username: "testuser",
		Email:    "test@example.com",
		Status:   models.UserStatusActive,
		Roles:    roles,
	}
	
	// Set a test password
	err := user.SetPassword("password")
	s.Require().NoError(err)
	
	return user
}

// createTestTenant creates a test tenant with the given ID
func (s *JWTServiceSuite) createTestTenant(id string) *models.Tenant {
	return &models.Tenant{
		ID:     id,
		Name:   "Test Tenant",
		Status: models.TenantStatusActive,
	}
}