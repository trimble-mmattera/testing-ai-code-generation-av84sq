package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert" // v1.8.0+
	"github.com/stretchr/testify/mock" // v1.8.0+
	"github.com/stretchr/testify/require" // v1.8.0+

	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services/auth"
	apperrors "../../pkg/errors"
	"testing/mocks" // v1.0.0+
)

// Helper function to set up an AuthUseCase instance with mocked dependencies
func setupAuthUseCase(t *testing.T) (*mocks.AuthService, *mocks.UserRepository, *mocks.TenantRepository, *AuthUseCase) {
	mockAuthService := new(mocks.AuthService)
	mockUserRepo := new(mocks.UserRepository)
	mockTenantRepo := new(mocks.TenantRepository)
	
	useCase := NewAuthUseCase(mockAuthService, mockUserRepo, mockTenantRepo)
	require.NotNil(t, useCase)
	
	return mockAuthService, mockUserRepo, mockTenantRepo, useCase
}

// Helper function to create a test user with specified parameters
func createTestUser(id, username, email, tenantID string, roles []string) *models.User {
	user := models.NewUser(username, email, tenantID)
	user.ID = id
	user.Roles = roles
	user.Status = models.UserStatusActive
	return user
}

// Helper function to create a test tenant with specified parameters
func createTestTenant(id, name string) *models.Tenant {
	tenant := models.NewTenant(name)
	tenant.ID = id
	tenant.Status = models.TenantStatusActive
	return tenant
}

// Tests the creation of a new AuthUseCase
func TestNewAuthUseCase(t *testing.T) {
	mockAuthService := new(mocks.AuthService)
	mockUserRepo := new(mocks.UserRepository)
	mockTenantRepo := new(mocks.TenantRepository)
	
	useCase := NewAuthUseCase(mockAuthService, mockUserRepo, mockTenantRepo)
	
	assert.NotNil(t, useCase)
	assert.Equal(t, mockAuthService, useCase.authService)
	assert.Equal(t, mockUserRepo, useCase.userRepo)
	assert.Equal(t, mockTenantRepo, useCase.tenantRepo)
}

// Tests successful login with valid credentials
func TestLogin_Success(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	userID := "user-123"
	username := "testuser"
	email := "test@example.com"
	password := "password123"
	roles := []string{"reader"}
	user := createTestUser(userID, username, email, tenantID, roles)
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("GetByUsername", mock.Anything, username, tenantID).Return(user, nil)
	
	// Mock password verification
	user.SetPassword(password)
	
	// Mock token generation
	accessToken := "access_token"
	refreshToken := "refresh_token"
	mockAuthService.On("GenerateToken", mock.Anything, userID, tenantID, roles, mock.Anything).Return(accessToken, nil)
	mockAuthService.On("GenerateRefreshToken", mock.Anything, userID, tenantID, mock.Anything).Return(refreshToken, nil)
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, username, password)
	
	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, accessToken, resultAccess)
	assert.Equal(t, refreshToken, resultRefresh)
	
	// Verify expectations
	mockTenantRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockAuthService.AssertExpectations(t)
}

// Tests login with invalid tenant ID
func TestLogin_InvalidTenant(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Set up expectations
	tenantID := "invalid-tenant"
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(nil, errors.New("tenant not found"))
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, "username", "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccess)
	assert.Empty(t, resultRefresh)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify user repository was not called
	mockUserRepo.AssertNotCalled(t, "GetByUsername")
}

// Tests login with inactive tenant
func TestLogin_InactiveTenant(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	tenant.Status = models.TenantStatusInactive
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, "username", "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccess)
	assert.Empty(t, resultRefresh)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify user repository was not called
	mockUserRepo.AssertNotCalled(t, "GetByUsername")
}

// Tests login with non-existent user
func TestLogin_UserNotFound(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	username := "nonexistent"
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("GetByUsername", mock.Anything, username, tenantID).Return(nil, errors.New("user not found"))
	mockUserRepo.On("GetByEmail", mock.Anything, username, tenantID).Return(nil, errors.New("user not found"))
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, username, "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccess)
	assert.Empty(t, resultRefresh)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify auth service was not called
	mockAuthService.AssertNotCalled(t, "GenerateToken")
}

// Tests login with user from different tenant
func TestLogin_WrongTenant(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-1"
	tenant := createTestTenant(tenantID, "Test Tenant 1")
	
	userID := "user-123"
	username := "testuser"
	email := "test@example.com"
	wrongTenantID := "tenant-2"
	user := createTestUser(userID, username, email, wrongTenantID, []string{"reader"})
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("GetByUsername", mock.Anything, username, tenantID).Return(user, nil)
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, username, "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccess)
	assert.Empty(t, resultRefresh)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify auth service was not called
	mockAuthService.AssertNotCalled(t, "GenerateToken")
}

// Tests login with inactive user
func TestLogin_InactiveUser(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	userID := "user-123"
	username := "testuser"
	email := "test@example.com"
	user := createTestUser(userID, username, email, tenantID, []string{"reader"})
	user.Status = models.UserStatusInactive
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("GetByUsername", mock.Anything, username, tenantID).Return(user, nil)
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, username, "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccess)
	assert.Empty(t, resultRefresh)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify auth service was not called
	mockAuthService.AssertNotCalled(t, "GenerateToken")
}

// Tests login with invalid password
func TestLogin_InvalidPassword(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	userID := "user-123"
	username := "testuser"
	email := "test@example.com"
	correctPassword := "password123"
	wrongPassword := "wrongpassword"
	user := createTestUser(userID, username, email, tenantID, []string{"reader"})
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("GetByUsername", mock.Anything, username, tenantID).Return(user, nil)
	
	// Set up correct password for the user
	user.SetPassword(correctPassword)
	
	// Call the method being tested
	resultAccess, resultRefresh, err := useCase.Login(context.Background(), tenantID, username, wrongPassword)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccess)
	assert.Empty(t, resultRefresh)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify auth service was not called
	mockAuthService.AssertNotCalled(t, "GenerateToken")
}

// Tests successful user registration
func TestRegister_Success(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	username := "newuser"
	email := "newuser@example.com"
	password := "password123"
	userID := "new-user-123"
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("ExistsByUsername", mock.Anything, username, tenantID).Return(false, nil)
	mockUserRepo.On("ExistsByEmail", mock.Anything, email, tenantID).Return(false, nil)
	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(userID, nil)
	
	// Call the method being tested
	resultUserID, err := useCase.Register(context.Background(), tenantID, username, email, password)
	
	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, userID, resultUserID)
	
	// Verify all methods were called
	mockTenantRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// Tests registration with invalid tenant ID
func TestRegister_InvalidTenant(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Set up expectations
	tenantID := "invalid-tenant"
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(nil, errors.New("tenant not found"))
	
	// Call the method being tested
	resultUserID, err := useCase.Register(context.Background(), tenantID, "username", "email@example.com", "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultUserID)
	assert.True(t, apperrors.IsValidationError(err))
	
	// Verify user repo was not called
	mockUserRepo.AssertNotCalled(t, "ExistsByUsername")
}

// Tests registration with inactive tenant
func TestRegister_InactiveTenant(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	tenant.Status = models.TenantStatusInactive
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	
	// Call the method being tested
	resultUserID, err := useCase.Register(context.Background(), tenantID, "username", "email@example.com", "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultUserID)
	assert.True(t, apperrors.IsValidationError(err))
	
	// Verify user repo was not called
	mockUserRepo.AssertNotCalled(t, "ExistsByUsername")
}

// Tests registration with already taken username
func TestRegister_UsernameTaken(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	username := "takenusername"
	email := "email@example.com"
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("ExistsByUsername", mock.Anything, username, tenantID).Return(true, nil)
	
	// Call the method being tested
	resultUserID, err := useCase.Register(context.Background(), tenantID, username, email, "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultUserID)
	assert.True(t, apperrors.IsValidationError(err))
	
	// Verify email check was not called
	mockUserRepo.AssertNotCalled(t, "ExistsByEmail")
	mockUserRepo.AssertNotCalled(t, "Create")
}

// Tests registration with already taken email
func TestRegister_EmailTaken(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	username := "newusername"
	email := "taken@example.com"
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	mockUserRepo.On("ExistsByUsername", mock.Anything, username, tenantID).Return(false, nil)
	mockUserRepo.On("ExistsByEmail", mock.Anything, email, tenantID).Return(true, nil)
	
	// Call the method being tested
	resultUserID, err := useCase.Register(context.Background(), tenantID, username, email, "password")
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultUserID)
	assert.True(t, apperrors.IsValidationError(err))
	
	// Verify create was not called
	mockUserRepo.AssertNotCalled(t, "Create")
}

// Tests registration with invalid user data
func TestRegister_InvalidData(t *testing.T) {
	mockAuthService, mockUserRepo, mockTenantRepo, useCase := setupAuthUseCase(t)
	
	// Create test data
	tenantID := "tenant-123"
	tenant := createTestTenant(tenantID, "Test Tenant")
	
	// Set up expectations
	mockTenantRepo.On("GetByID", mock.Anything, tenantID).Return(tenant, nil)
	
	// Test cases
	testCases := []struct {
		name     string
		username string
		email    string
		password string
	}{
		{"Empty username", "", "valid@example.com", "validpass123"},
		{"Invalid email", "validuser", "invalid-email", "validpass123"},
		{"Short password", "validuser", "valid@example.com", "short"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the method being tested
			resultUserID, err := useCase.Register(context.Background(), tenantID, tc.username, tc.email, tc.password)
			
			// Assert results
			assert.Error(t, err)
			assert.Empty(t, resultUserID)
			assert.True(t, apperrors.IsValidationError(err))
		})
	}
	
	// Verify existence checks were not called
	mockUserRepo.AssertNotCalled(t, "ExistsByUsername")
	mockUserRepo.AssertNotCalled(t, "ExistsByEmail")
	mockUserRepo.AssertNotCalled(t, "Create")
}

// Tests successful token validation
func TestValidateToken_Success(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	token := "valid_token"
	userID := "user-123"
	tenantID := "tenant-123"
	roles := []string{"reader", "contributor"}
	
	// Set up expectations
	mockAuthService.On("ValidateToken", mock.Anything, token).Return(userID, tenantID, roles, nil)
	
	// Call the method being tested
	resultUserID, resultTenantID, resultRoles, err := useCase.ValidateToken(context.Background(), token)
	
	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, userID, resultUserID)
	assert.Equal(t, tenantID, resultTenantID)
	assert.Equal(t, roles, resultRoles)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests validation of invalid token
func TestValidateToken_Invalid(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	token := "invalid_token"
	
	// Set up expectations
	mockAuthService.On("ValidateToken", mock.Anything, token).Return("", "", []string{}, errors.New("invalid token"))
	
	// Call the method being tested
	resultUserID, resultTenantID, resultRoles, err := useCase.ValidateToken(context.Background(), token)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultUserID)
	assert.Empty(t, resultTenantID)
	assert.Empty(t, resultRoles)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests successful token refresh
func TestRefreshToken_Success(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	refreshToken := "valid_refresh_token"
	newAccessToken := "new_access_token"
	newRefreshToken := "new_refresh_token"
	
	// Set up expectations
	mockAuthService.On("RefreshToken", mock.Anything, refreshToken).Return(newAccessToken, newRefreshToken, nil)
	
	// Call the method being tested
	resultAccessToken, resultRefreshToken, err := useCase.RefreshToken(context.Background(), refreshToken)
	
	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, newAccessToken, resultAccessToken)
	assert.Equal(t, newRefreshToken, resultRefreshToken)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests refresh with invalid refresh token
func TestRefreshToken_Invalid(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	refreshToken := "invalid_refresh_token"
	
	// Set up expectations
	mockAuthService.On("RefreshToken", mock.Anything, refreshToken).Return("", "", errors.New("invalid refresh token"))
	
	// Call the method being tested
	resultAccessToken, resultRefreshToken, err := useCase.RefreshToken(context.Background(), refreshToken)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultAccessToken)
	assert.Empty(t, resultRefreshToken)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests successful logout
func TestLogout_Success(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	token := "valid_token"
	
	// Set up expectations
	mockAuthService.On("InvalidateToken", mock.Anything, token).Return(nil)
	
	// Call the method being tested
	err := useCase.Logout(context.Background(), token)
	
	// Assert results
	assert.NoError(t, err)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests logout with error from auth service
func TestLogout_Error(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	token := "token_with_error"
	
	// Set up expectations
	mockAuthService.On("InvalidateToken", mock.Anything, token).Return(errors.New("invalidation error"))
	
	// Call the method being tested
	err := useCase.Logout(context.Background(), token)
	
	// Assert results
	assert.Error(t, err)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests successful password change
func TestChangePassword_Success(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	currentPassword := "current123"
	newPassword := "new123"
	user := createTestUser(userID, "username", "email@example.com", tenantID, []string{"reader"})
	user.SetPassword(currentPassword)
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	mockUserRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string"), tenantID).Return(nil)
	
	// Call the method being tested
	err := useCase.ChangePassword(context.Background(), userID, tenantID, currentPassword, newPassword)
	
	// Assert results
	assert.NoError(t, err)
	
	// Verify repo methods were called
	mockUserRepo.AssertExpectations(t)
}

// Tests password change for non-existent user
func TestChangePassword_UserNotFound(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "nonexistent-user"
	tenantID := "tenant-123"
	currentPassword := "current123"
	newPassword := "new123"
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(nil, errors.New("user not found"))
	
	// Call the method being tested
	err := useCase.ChangePassword(context.Background(), userID, tenantID, currentPassword, newPassword)
	
	// Assert results
	assert.Error(t, err)
	
	// Verify UpdatePassword was not called
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
}

// Tests password change with user from different tenant
func TestChangePassword_WrongTenant(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	userTenantID := "tenant-1"
	requestTenantID := "tenant-2"
	currentPassword := "current123"
	newPassword := "new123"
	user := createTestUser(userID, "username", "email@example.com", userTenantID, []string{"reader"})
	user.SetPassword(currentPassword)
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, requestTenantID).Return(user, nil)
	
	// Call the method being tested
	err := useCase.ChangePassword(context.Background(), userID, requestTenantID, currentPassword, newPassword)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, apperrors.IsAuthorizationError(err))
	
	// Verify UpdatePassword was not called
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
}

// Tests password change with incorrect current password
func TestChangePassword_IncorrectCurrentPassword(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	correctPassword := "correct123"
	wrongPassword := "wrong123"
	newPassword := "new123"
	user := createTestUser(userID, "username", "email@example.com", tenantID, []string{"reader"})
	user.SetPassword(correctPassword)
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	
	// Call the method being tested
	err := useCase.ChangePassword(context.Background(), userID, tenantID, wrongPassword, newPassword)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, apperrors.IsAuthenticationError(err))
	
	// Verify UpdatePassword was not called
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
}

// Tests successful password reset by admin
func TestResetPassword_Success(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "admin-123"
	userID := "user-123"
	tenantID := "tenant-123"
	newPassword := "new123"
	
	admin := createTestUser(adminID, "admin", "admin@example.com", tenantID, []string{"administrator"})
	user := createTestUser(userID, "user", "user@example.com", tenantID, []string{"reader"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(admin, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	mockUserRepo.On("UpdatePassword", mock.Anything, userID, mock.AnythingOfType("string"), tenantID).Return(nil)
	
	// Call the method being tested
	err := useCase.ResetPassword(context.Background(), adminID, userID, tenantID, newPassword)
	
	// Assert results
	assert.NoError(t, err)
	
	// Verify repo methods were called
	mockUserRepo.AssertExpectations(t)
}

// Tests password reset with non-existent admin
func TestResetPassword_AdminNotFound(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "nonexistent-admin"
	userID := "user-123"
	tenantID := "tenant-123"
	newPassword := "new123"
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(nil, errors.New("admin not found"))
	
	// Call the method being tested
	err := useCase.ResetPassword(context.Background(), adminID, userID, tenantID, newPassword)
	
	// Assert results
	assert.Error(t, err)
	
	// Verify target user was not queried
	mockUserRepo.AssertNotCalled(t, "GetByID", mock.Anything, userID, tenantID)
}

// Tests password reset with admin from different tenant
func TestResetPassword_AdminWrongTenant(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "admin-123"
	userID := "user-123"
	adminTenantID := "tenant-1"
	requestTenantID := "tenant-2"
	newPassword := "new123"
	
	admin := createTestUser(adminID, "admin", "admin@example.com", adminTenantID, []string{"administrator"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, requestTenantID).Return(admin, nil)
	
	// Call the method being tested
	err := useCase.ResetPassword(context.Background(), adminID, userID, requestTenantID, newPassword)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, apperrors.IsAuthorizationError(err))
	
	// Verify target user was not queried
	mockUserRepo.AssertNotCalled(t, "GetByID", mock.Anything, userID, requestTenantID)
}

// Tests password reset by non-admin user
func TestResetPassword_NotAdmin(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	nonAdminID := "user-456"
	userID := "user-123"
	tenantID := "tenant-123"
	newPassword := "new123"
	
	nonAdmin := createTestUser(nonAdminID, "nonAdmin", "nonadmin@example.com", tenantID, []string{"reader"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, nonAdminID, tenantID).Return(nonAdmin, nil)
	
	// Call the method being tested
	err := useCase.ResetPassword(context.Background(), nonAdminID, userID, tenantID, newPassword)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, apperrors.IsAuthorizationError(err))
	
	// Verify target user was not queried
	mockUserRepo.AssertNotCalled(t, "GetByID", mock.Anything, userID, tenantID)
}

// Tests password reset for non-existent target user
func TestResetPassword_TargetUserNotFound(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "admin-123"
	userID := "nonexistent-user"
	tenantID := "tenant-123"
	newPassword := "new123"
	
	admin := createTestUser(adminID, "admin", "admin@example.com", tenantID, []string{"administrator"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(admin, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(nil, errors.New("user not found"))
	
	// Call the method being tested
	err := useCase.ResetPassword(context.Background(), adminID, userID, tenantID, newPassword)
	
	// Assert results
	assert.Error(t, err)
	
	// Verify user update was not called
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
}

// Tests password reset for user from different tenant
func TestResetPassword_TargetUserWrongTenant(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "admin-123"
	userID := "user-123"
	tenantID := "tenant-1"
	userTenantID := "tenant-2"
	newPassword := "new123"
	
	admin := createTestUser(adminID, "admin", "admin@example.com", tenantID, []string{"administrator"})
	user := createTestUser(userID, "user", "user@example.com", userTenantID, []string{"reader"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(admin, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	
	// Call the method being tested
	err := useCase.ResetPassword(context.Background(), adminID, userID, tenantID, newPassword)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, apperrors.IsAuthorizationError(err))
	
	// Verify user update was not called
	mockUserRepo.AssertNotCalled(t, "UpdatePassword")
}

// Tests successful permission verification
func TestVerifyPermission_Success(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	permission := auth.PermissionRead
	
	// Set up expectations
	mockAuthService.On("VerifyPermission", mock.Anything, userID, tenantID, permission).Return(true, nil)
	
	// Call the method being tested
	result, err := useCase.VerifyPermission(context.Background(), userID, tenantID, permission)
	
	// Assert results
	assert.NoError(t, err)
	assert.True(t, result)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests permission verification when permission is denied
func TestVerifyPermission_Denied(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	permission := auth.PermissionWrite
	
	// Set up expectations
	mockAuthService.On("VerifyPermission", mock.Anything, userID, tenantID, permission).Return(false, nil)
	
	// Call the method being tested
	result, err := useCase.VerifyPermission(context.Background(), userID, tenantID, permission)
	
	// Assert results
	assert.NoError(t, err)
	assert.False(t, result)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests successful resource access verification
func TestVerifyResourceAccess_Success(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	resourceType := auth.ResourceTypeDocument
	resourceID := "doc-123"
	accessType := auth.PermissionRead
	
	// Set up expectations
	mockAuthService.On("VerifyResourceAccess", mock.Anything, userID, tenantID, resourceType, resourceID, accessType).Return(true, nil)
	
	// Call the method being tested
	result, err := useCase.VerifyResourceAccess(context.Background(), userID, tenantID, resourceType, resourceID, accessType)
	
	// Assert results
	assert.NoError(t, err)
	assert.True(t, result)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests resource access verification when access is denied
func TestVerifyResourceAccess_Denied(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	resourceType := auth.ResourceTypeDocument
	resourceID := "doc-123"
	accessType := auth.PermissionDelete
	
	// Set up expectations
	mockAuthService.On("VerifyResourceAccess", mock.Anything, userID, tenantID, resourceType, resourceID, accessType).Return(false, nil)
	
	// Call the method being tested
	result, err := useCase.VerifyResourceAccess(context.Background(), userID, tenantID, resourceType, resourceID, accessType)
	
	// Assert results
	assert.NoError(t, err)
	assert.False(t, result)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests successful retrieval of user roles
func TestGetUserRoles_Success(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	tenantID := "tenant-123"
	roles := []string{"reader", "contributor"}
	user := createTestUser(userID, "username", "email@example.com", tenantID, roles)
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	
	// Call the method being tested
	resultRoles, err := useCase.GetUserRoles(context.Background(), userID, tenantID)
	
	// Assert results
	assert.NoError(t, err)
	assert.Equal(t, roles, resultRoles)
	
	// Verify user repo was called
	mockUserRepo.AssertExpectations(t)
}

// Tests role retrieval for non-existent user
func TestGetUserRoles_UserNotFound(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "nonexistent-user"
	tenantID := "tenant-123"
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(nil, errors.New("user not found"))
	
	// Call the method being tested
	resultRoles, err := useCase.GetUserRoles(context.Background(), userID, tenantID)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultRoles)
	
	// Verify user repo was called
	mockUserRepo.AssertExpectations(t)
}

// Tests role retrieval for user from different tenant
func TestGetUserRoles_WrongTenant(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	userID := "user-123"
	userTenantID := "tenant-1"
	requestTenantID := "tenant-2"
	roles := []string{"reader"}
	user := createTestUser(userID, "username", "email@example.com", userTenantID, roles)
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, userID, requestTenantID).Return(user, nil)
	
	// Call the method being tested
	resultRoles, err := useCase.GetUserRoles(context.Background(), userID, requestTenantID)
	
	// Assert results
	assert.Error(t, err)
	assert.Empty(t, resultRoles)
	assert.True(t, apperrors.IsAuthorizationError(err))
	
	// Verify user repo was called
	mockUserRepo.AssertExpectations(t)
}

// Tests successful addition of user role
func TestAddUserRole_Success(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "admin-123"
	userID := "user-123"
	tenantID := "tenant-123"
	role := "contributor"
	
	admin := createTestUser(adminID, "admin", "admin@example.com", tenantID, []string{"administrator"})
	user := createTestUser(userID, "user", "user@example.com", tenantID, []string{"reader"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(admin, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	mockUserRepo.On("AddRole", mock.Anything, userID, role, tenantID).Return(nil)
	
	// Call the method being tested
	err := useCase.AddUserRole(context.Background(), adminID, userID, tenantID, role)
	
	// Assert results
	assert.NoError(t, err)
	
	// Verify repo methods were called
	mockUserRepo.AssertExpectations(t)
}

// Tests role addition with non-existent admin
func TestAddUserRole_AdminNotFound(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "nonexistent-admin"
	userID := "user-123"
	tenantID := "tenant-123"
	role := "contributor"
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(nil, errors.New("admin not found"))
	
	// Call the method being tested
	err := useCase.AddUserRole(context.Background(), adminID, userID, tenantID, role)
	
	// Assert results
	assert.Error(t, err)
	
	// Verify target user was not queried
	mockUserRepo.AssertNotCalled(t, "GetByID", mock.Anything, userID, tenantID)
}

// Tests role addition by non-admin user
func TestAddUserRole_NotAdmin(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	nonAdminID := "user-456"
	userID := "user-123"
	tenantID := "tenant-123"
	role := "contributor"
	
	nonAdmin := createTestUser(nonAdminID, "nonAdmin", "nonadmin@example.com", tenantID, []string{"reader"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, nonAdminID, tenantID).Return(nonAdmin, nil)
	
	// Call the method being tested
	err := useCase.AddUserRole(context.Background(), nonAdminID, userID, tenantID, role)
	
	// Assert results
	assert.Error(t, err)
	assert.True(t, apperrors.IsAuthorizationError(err))
	
	// Verify target user was not queried
	mockUserRepo.AssertNotCalled(t, "GetByID", mock.Anything, userID, tenantID)
}

// Tests successful removal of user role
func TestRemoveUserRole_Success(t *testing.T) {
	_, mockUserRepo, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	adminID := "admin-123"
	userID := "user-123"
	tenantID := "tenant-123"
	role := "contributor"
	
	admin := createTestUser(adminID, "admin", "admin@example.com", tenantID, []string{"administrator"})
	user := createTestUser(userID, "user", "user@example.com", tenantID, []string{"reader", "contributor"})
	
	// Set up expectations
	mockUserRepo.On("GetByID", mock.Anything, adminID, tenantID).Return(admin, nil)
	mockUserRepo.On("GetByID", mock.Anything, userID, tenantID).Return(user, nil)
	mockUserRepo.On("RemoveRole", mock.Anything, userID, role, tenantID).Return(nil)
	
	// Call the method being tested
	err := useCase.RemoveUserRole(context.Background(), adminID, userID, tenantID, role)
	
	// Assert results
	assert.NoError(t, err)
	
	// Verify repo methods were called
	mockUserRepo.AssertExpectations(t)
}

// Tests setting token expiration
func TestSetTokenExpiration(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	expiration := 30 * time.Minute
	
	// Set up expectations
	mockAuthService.On("SetTokenExpiration", expiration).Return()
	
	// Call the method being tested
	useCase.SetTokenExpiration(expiration)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}

// Tests setting refresh token expiration
func TestSetRefreshTokenExpiration(t *testing.T) {
	mockAuthService, _, _, useCase := setupAuthUseCase(t)
	
	// Create test data
	expiration := 7 * 24 * time.Hour
	
	// Set up expectations
	mockAuthService.On("SetRefreshTokenExpiration", expiration).Return()
	
	// Call the method being tested
	useCase.SetRefreshTokenExpiration(expiration)
	
	// Verify auth service was called
	mockAuthService.AssertExpectations(t)
}