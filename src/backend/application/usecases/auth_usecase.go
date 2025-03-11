// Package usecases implements the application-specific use cases for the Document Management Platform.
package usecases

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt" // v0.0.0-20220622213112-05595931fe9d

	"../../domain/models"
	"../../domain/repositories"
	"../../domain/services"
	"../../pkg/errors"
)

// Default token expiration durations
var (
	defaultTokenExpiration        = time.Hour
	defaultRefreshTokenExpiration = time.Hour * 24 * 7
)

// AuthUseCase provides authentication and authorization functionality for the application
type AuthUseCase struct {
	authService           services.AuthService
	userRepo              repositories.UserRepository
	tenantRepo            repositories.TenantRepository
	tokenExpiration       time.Duration
	refreshTokenExpiration time.Duration
}

// NewAuthUseCase creates a new authentication use case with the given dependencies
func NewAuthUseCase(authService services.AuthService, userRepo repositories.UserRepository, tenantRepo repositories.TenantRepository) (*AuthUseCase, error) {
	// Validate input parameters
	if authService == nil {
		return nil, errors.NewValidationError("auth service is required")
	}
	if userRepo == nil {
		return nil, errors.NewValidationError("user repository is required")
	}
	if tenantRepo == nil {
		return nil, errors.NewValidationError("tenant repository is required")
	}

	// Create a new AuthUseCase instance with the provided dependencies
	return &AuthUseCase{
		authService:           authService,
		userRepo:              userRepo,
		tenantRepo:            tenantRepo,
		tokenExpiration:       defaultTokenExpiration,
		refreshTokenExpiration: defaultRefreshTokenExpiration,
	}, nil
}

// Login authenticates a user with username/email and password
func (a *AuthUseCase) Login(ctx context.Context, tenantID, usernameOrEmail, password string) (string, error) {
	// Validate input parameters
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID is required")
	}
	if usernameOrEmail == "" {
		return "", errors.NewValidationError("username or email is required")
	}
	if password == "" {
		return "", errors.NewValidationError("password is required")
	}

	// Check if tenant exists and is active
	tenant, err := a.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", errors.NewAuthenticationError("invalid tenant ID")
		}
		return "", errors.Wrap(err, "failed to retrieve tenant")
	}

	// We need to verify tenant is active
	// Assuming Tenant has an IsActive method similar to User
	if !strings.EqualFold(tenant.Status, "active") {
		return "", errors.NewAuthenticationError("tenant is not active")
	}

	// Try to get user by username
	var user *models.User
	user, err = a.userRepo.GetByUsername(ctx, usernameOrEmail, tenantID)
	if err != nil && !errors.IsResourceNotFoundError(err) {
		return "", errors.Wrap(err, "failed to retrieve user by username")
	}

	// If not found by username, try by email
	if user == nil || errors.IsResourceNotFoundError(err) {
		user, err = a.userRepo.GetByEmail(ctx, usernameOrEmail, tenantID)
		if err != nil {
			if errors.IsResourceNotFoundError(err) {
				return "", errors.NewAuthenticationError("invalid credentials")
			}
			return "", errors.Wrap(err, "failed to retrieve user by email")
		}
	}

	// Verify user belongs to the specified tenant
	if user.TenantID != tenantID {
		return "", errors.NewAuthenticationError("invalid credentials")
	}

	// Verify user is active
	if !user.IsActive() {
		return "", errors.NewAuthenticationError("user account is not active")
	}

	// Verify password
	match, err := user.VerifyPassword(password)
	if err != nil {
		return "", errors.Wrap(err, "password verification failed")
	}
	if !match {
		return "", errors.NewAuthenticationError("invalid credentials")
	}

	// Generate access token with user ID, tenant ID, and roles
	token, err := a.authService.GenerateToken(ctx, user.ID, user.TenantID, user.Roles, a.tokenExpiration)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate access token")
	}

	// Generate refresh token
	refreshToken, err := a.authService.GenerateRefreshToken(ctx, user.ID, user.TenantID, a.refreshTokenExpiration)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate refresh token")
	}

	return refreshToken, nil
}

// Register registers a new user in the system
func (a *AuthUseCase) Register(ctx context.Context, tenantID, username, email, password string, roles []string) (string, error) {
	// Validate input parameters
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID is required")
	}
	if username == "" {
		return "", errors.NewValidationError("username is required")
	}
	if email == "" {
		return "", errors.NewValidationError("email is required")
	}
	if password == "" {
		return "", errors.NewValidationError("password is required")
	}

	// Check if tenant exists and is active
	tenant, err := a.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", errors.NewValidationError("invalid tenant ID")
		}
		return "", errors.Wrap(err, "failed to retrieve tenant")
	}

	// Verify tenant is active
	if !strings.EqualFold(tenant.Status, "active") {
		return "", errors.NewValidationError("tenant is not active")
	}

	// Check if username is already taken
	exists, err := a.userRepo.ExistsByUsername(ctx, username, tenantID)
	if err != nil {
		return "", errors.Wrap(err, "failed to check username availability")
	}
	if exists {
		return "", errors.NewValidationError("username already exists")
	}

	// Check if email is already taken
	exists, err = a.userRepo.ExistsByEmail(ctx, email, tenantID)
	if err != nil {
		return "", errors.Wrap(err, "failed to check email availability")
	}
	if exists {
		return "", errors.NewValidationError("email already exists")
	}

	// Create a new User instance
	user := models.NewUser(username, email, tenantID)

	// Set password using user.SetPassword
	err = user.SetPassword(password)
	if err != nil {
		return "", errors.Wrap(err, "failed to set password")
	}

	// Add roles to the user
	for _, role := range roles {
		user.AddRole(role)
	}

	// Create the user in the repository
	userID, err := a.userRepo.Create(ctx, user)
	if err != nil {
		return "", errors.Wrap(err, "failed to create user")
	}

	return userID, nil
}

// ValidateToken validates an access token and extracts user information
func (a *AuthUseCase) ValidateToken(ctx context.Context, token string) (string, []string, error) {
	// Validate token parameter
	if token == "" {
		return "", nil, errors.NewAuthenticationError("token is required")
	}

	// Call authService.ValidateToken to validate the token
	tenantID, roles, err := a.authService.ValidateToken(ctx, token)
	if err != nil {
		return "", nil, errors.Wrap(err, "token validation failed")
	}

	return tenantID, roles, nil
}

// RefreshToken refreshes an access token using a refresh token
func (a *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Validate refresh token parameter
	if refreshToken == "" {
		return "", errors.NewAuthenticationError("refresh token is required")
	}

	// Call authService.RefreshToken to refresh the token
	newRefreshToken, err := a.authService.RefreshToken(ctx, refreshToken)
	if err != nil {
		return "", errors.Wrap(err, "token refresh failed")
	}

	return newRefreshToken, nil
}

// Logout logs out a user by invalidating their token
func (a *AuthUseCase) Logout(ctx context.Context, token string) error {
	// Validate token parameter
	if token == "" {
		return errors.NewValidationError("token is required")
	}

	// Call authService.InvalidateToken to invalidate the token
	err := a.authService.InvalidateToken(ctx, token)
	if err != nil {
		return errors.Wrap(err, "token invalidation failed")
	}

	return nil
}

// ChangePassword changes a user's password
func (a *AuthUseCase) ChangePassword(ctx context.Context, userID, tenantID, currentPassword, newPassword string) error {
	// Validate input parameters
	if userID == "" {
		return errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID is required")
	}
	if currentPassword == "" {
		return errors.NewValidationError("current password is required")
	}
	if newPassword == "" {
		return errors.NewValidationError("new password is required")
	}
	if len(newPassword) < 8 {
		return errors.NewValidationError("new password must be at least 8 characters long")
	}

	// Get user from repository
	user, err := a.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewAuthenticationError("user not found")
		}
		return errors.Wrap(err, "failed to retrieve user")
	}

	// Verify user belongs to the specified tenant
	if user.TenantID != tenantID {
		return errors.NewAuthenticationError("user does not belong to the specified tenant")
	}

	// Verify current password
	match, err := user.VerifyPassword(currentPassword)
	if err != nil {
		return errors.Wrap(err, "password verification failed")
	}
	if !match {
		return errors.NewAuthenticationError("current password is incorrect")
	}

	// Set new password
	err = user.SetPassword(newPassword)
	if err != nil {
		return errors.Wrap(err, "failed to set new password")
	}

	// Update user in repository
	err = a.userRepo.Update(ctx, user)
	if err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	return nil
}

// ResetPassword resets a user's password (admin function)
func (a *AuthUseCase) ResetPassword(ctx context.Context, adminUserID, userID, tenantID, newPassword string) error {
	// Validate input parameters
	if adminUserID == "" {
		return errors.NewValidationError("admin user ID is required")
	}
	if userID == "" {
		return errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID is required")
	}
	if newPassword == "" {
		return errors.NewValidationError("new password is required")
	}
	if len(newPassword) < 8 {
		return errors.NewValidationError("new password must be at least 8 characters long")
	}

	// Get admin user from repository
	adminUser, err := a.userRepo.GetByID(ctx, adminUserID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewAuthenticationError("admin user not found")
		}
		return errors.Wrap(err, "failed to retrieve admin user")
	}

	// Verify admin user belongs to the specified tenant
	if adminUser.TenantID != tenantID {
		return errors.NewAuthenticationError("admin user does not belong to the specified tenant")
	}

	// Verify admin user has administrator role
	if !adminUser.HasRole("administrator") {
		return errors.NewAuthorizationError("user does not have administrator privileges")
	}

	// Get target user from repository
	user, err := a.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewResourceNotFoundError("user not found")
		}
		return errors.Wrap(err, "failed to retrieve user")
	}

	// Verify target user belongs to the specified tenant
	if user.TenantID != tenantID {
		return errors.NewAuthorizationError("user does not belong to the specified tenant")
	}

	// Set new password
	err = user.SetPassword(newPassword)
	if err != nil {
		return errors.Wrap(err, "failed to set new password")
	}

	// Update user in repository
	err = a.userRepo.Update(ctx, user)
	if err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	return nil
}

// VerifyPermission verifies if a user has a specific permission
func (a *AuthUseCase) VerifyPermission(ctx context.Context, userID, tenantID, permission string) (bool, error) {
	// Validate input parameters
	if userID == "" {
		return false, errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID is required")
	}
	if permission == "" {
		return false, errors.NewValidationError("permission is required")
	}

	// Call authService.VerifyPermission to check the permission
	hasPermission, err := a.authService.VerifyPermission(ctx, userID, tenantID, permission)
	if err != nil {
		return false, errors.Wrap(err, "permission verification failed")
	}

	return hasPermission, nil
}

// VerifyResourceAccess verifies if a user has access to a specific resource
func (a *AuthUseCase) VerifyResourceAccess(ctx context.Context, userID, tenantID, resourceType, resourceID, accessType string) (bool, error) {
	// Validate input parameters
	if userID == "" {
		return false, errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID is required")
	}
	if resourceType == "" {
		return false, errors.NewValidationError("resource type is required")
	}
	if resourceID == "" {
		return false, errors.NewValidationError("resource ID is required")
	}
	if accessType == "" {
		return false, errors.NewValidationError("access type is required")
	}

	// Call authService.VerifyResourceAccess to check the resource access
	hasAccess, err := a.authService.VerifyResourceAccess(ctx, userID, tenantID, resourceType, resourceID, accessType)
	if err != nil {
		return false, errors.Wrap(err, "resource access verification failed")
	}

	return hasAccess, nil
}

// GetUserRoles gets the roles of a user
func (a *AuthUseCase) GetUserRoles(ctx context.Context, userID, tenantID string) ([]string, error) {
	// Validate input parameters
	if userID == "" {
		return nil, errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return nil, errors.NewValidationError("tenant ID is required")
	}

	// Get user from repository
	user, err := a.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return nil, errors.NewResourceNotFoundError("user not found")
		}
		return nil, errors.Wrap(err, "failed to retrieve user")
	}

	// Verify user belongs to the specified tenant
	if user.TenantID != tenantID {
		return nil, errors.NewAuthorizationError("user does not belong to the specified tenant")
	}

	return user.Roles, nil
}

// AddUserRole adds a role to a user
func (a *AuthUseCase) AddUserRole(ctx context.Context, adminUserID, userID, tenantID, role string) error {
	// Validate input parameters
	if adminUserID == "" {
		return errors.NewValidationError("admin user ID is required")
	}
	if userID == "" {
		return errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID is required")
	}
	if role == "" {
		return errors.NewValidationError("role is required")
	}

	// Get admin user from repository
	adminUser, err := a.userRepo.GetByID(ctx, adminUserID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewAuthenticationError("admin user not found")
		}
		return errors.Wrap(err, "failed to retrieve admin user")
	}

	// Verify admin user belongs to the specified tenant
	if adminUser.TenantID != tenantID {
		return errors.NewAuthenticationError("admin user does not belong to the specified tenant")
	}

	// Verify admin user has administrator role
	if !adminUser.HasRole("administrator") {
		return errors.NewAuthorizationError("user does not have administrator privileges")
	}

	// Get target user from repository
	user, err := a.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewResourceNotFoundError("user not found")
		}
		return errors.Wrap(err, "failed to retrieve user")
	}

	// Verify target user belongs to the specified tenant
	if user.TenantID != tenantID {
		return errors.NewAuthorizationError("user does not belong to the specified tenant")
	}

	// Add role to user
	err = a.userRepo.AddRole(ctx, userID, role, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to add role to user")
	}

	return nil
}

// RemoveUserRole removes a role from a user
func (a *AuthUseCase) RemoveUserRole(ctx context.Context, adminUserID, userID, tenantID, role string) error {
	// Validate input parameters
	if adminUserID == "" {
		return errors.NewValidationError("admin user ID is required")
	}
	if userID == "" {
		return errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return errors.NewValidationError("tenant ID is required")
	}
	if role == "" {
		return errors.NewValidationError("role is required")
	}

	// Get admin user from repository
	adminUser, err := a.userRepo.GetByID(ctx, adminUserID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewAuthenticationError("admin user not found")
		}
		return errors.Wrap(err, "failed to retrieve admin user")
	}

	// Verify admin user belongs to the specified tenant
	if adminUser.TenantID != tenantID {
		return errors.NewAuthenticationError("admin user does not belong to the specified tenant")
	}

	// Verify admin user has administrator role
	if !adminUser.HasRole("administrator") {
		return errors.NewAuthorizationError("user does not have administrator privileges")
	}

	// Get target user from repository
	user, err := a.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return errors.NewResourceNotFoundError("user not found")
		}
		return errors.Wrap(err, "failed to retrieve user")
	}

	// Verify target user belongs to the specified tenant
	if user.TenantID != tenantID {
		return errors.NewAuthorizationError("user does not belong to the specified tenant")
	}

	// Remove role from user
	err = a.userRepo.RemoveRole(ctx, userID, role, tenantID)
	if err != nil {
		return errors.Wrap(err, "failed to remove role from user")
	}

	return nil
}

// SetTokenExpiration sets the token expiration duration
func (a *AuthUseCase) SetTokenExpiration(expiration time.Duration) {
	// Validate that expiration is positive
	if expiration <= 0 {
		return
	}

	// Set the tokenExpiration field
	a.tokenExpiration = expiration

	// Update the authService token expiration
	a.authService.SetTokenExpiration(expiration)
}

// SetRefreshTokenExpiration sets the refresh token expiration duration
func (a *AuthUseCase) SetRefreshTokenExpiration(expiration time.Duration) {
	// Validate that expiration is positive
	if expiration <= 0 {
		return
	}

	// Set the refreshTokenExpiration field
	a.refreshTokenExpiration = expiration

	// Update the authService refresh token expiration
	a.authService.SetRefreshTokenExpiration(expiration)
}