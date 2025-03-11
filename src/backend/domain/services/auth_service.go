// Package services provides domain service interfaces for the Document Management Platform.
package services

import (
	"context"
	"time"
)

// Permission constants define the available permission types in the system
const (
	PermissionRead          = "read"
	PermissionWrite         = "write"
	PermissionDelete        = "delete"
	PermissionManageFolders = "manage_folders"
)

// Resource type constants define the types of resources that can be accessed
const (
	ResourceTypeDocument = "document"
	ResourceTypeFolder   = "folder"
)

// AuthService defines the contract for authentication and authorization operations
// in the Document Management Platform. It follows Clean Architecture principles by
// defining the interface at the domain level without implementation details.
type AuthService interface {
	// Authenticate validates user credentials and returns a refresh token if successful.
	// The implementation should internally generate access tokens and handle their secure delivery.
	// Parameters:
	//   - ctx: Context for the operation
	//   - tenantID: The ID of the tenant the user belongs to
	//   - usernameOrEmail: The username or email of the user
	//   - password: The password of the user
	// Returns:
	//   - string: Refresh token
	//   - error: Error if authentication fails
	Authenticate(ctx context.Context, tenantID, usernameOrEmail, password string) (string, error)

	// ValidateToken validates an access token and extracts user information.
	// Parameters:
	//   - ctx: Context for the operation
	//   - token: The access token to validate
	// Returns:
	//   - string: Tenant ID extracted from the token
	//   - []string: User roles extracted from the token
	//   - error: Error if token is invalid
	ValidateToken(ctx context.Context, token string) (string, []string, error)

	// RefreshToken generates a new access token using a refresh token.
	// Parameters:
	//   - ctx: Context for the operation
	//   - refreshToken: The refresh token to use
	// Returns:
	//   - string: New refresh token
	//   - error: Error if refresh fails
	RefreshToken(ctx context.Context, refreshToken string) (string, error)

	// InvalidateToken invalidates a token (for logout).
	// Parameters:
	//   - ctx: Context for the operation
	//   - token: The token to invalidate
	// Returns:
	//   - error: Error if invalidation fails
	InvalidateToken(ctx context.Context, token string) error

	// VerifyPermission checks if a user has a specific permission.
	// Parameters:
	//   - ctx: Context for the operation
	//   - userID: The ID of the user
	//   - tenantID: The ID of the tenant
	//   - permission: The permission to check
	// Returns:
	//   - bool: True if the user has the permission
	//   - error: Error if verification fails
	VerifyPermission(ctx context.Context, userID, tenantID, permission string) (bool, error)

	// VerifyResourceAccess checks if a user has access to a specific resource.
	// Parameters:
	//   - ctx: Context for the operation
	//   - userID: The ID of the user
	//   - tenantID: The ID of the tenant
	//   - resourceType: The type of resource (document, folder)
	//   - resourceID: The ID of the resource
	//   - accessType: The type of access (read, write, delete)
	// Returns:
	//   - bool: True if the user has access
	//   - error: Error if verification fails
	VerifyResourceAccess(ctx context.Context, userID, tenantID, resourceType, resourceID, accessType string) (bool, error)

	// VerifyTenantAccess checks if a user belongs to a specific tenant.
	// Parameters:
	//   - ctx: Context for the operation
	//   - userID: The ID of the user
	//   - tenantID: The ID of the tenant
	// Returns:
	//   - bool: True if the user belongs to the tenant
	//   - error: Error if verification fails
	VerifyTenantAccess(ctx context.Context, userID, tenantID string) (bool, error)

	// GenerateToken creates a new access token for a user.
	// Parameters:
	//   - ctx: Context for the operation
	//   - userID: The ID of the user
	//   - tenantID: The ID of the tenant
	//   - roles: The roles of the user
	//   - expiration: The token expiration duration
	// Returns:
	//   - string: Generated access token
	//   - error: Error if generation fails
	GenerateToken(ctx context.Context, userID, tenantID string, roles []string, expiration time.Duration) (string, error)

	// GenerateRefreshToken creates a new refresh token for a user.
	// Parameters:
	//   - ctx: Context for the operation
	//   - userID: The ID of the user
	//   - tenantID: The ID of the tenant
	//   - expiration: The token expiration duration
	// Returns:
	//   - string: Generated refresh token
	//   - error: Error if generation fails
	GenerateRefreshToken(ctx context.Context, userID, tenantID string, expiration time.Duration) (string, error)

	// SetTokenExpiration sets the default token expiration duration.
	// Parameters:
	//   - expiration: The token expiration duration
	SetTokenExpiration(expiration time.Duration)

	// SetRefreshTokenExpiration sets the default refresh token expiration duration.
	// Parameters:
	//   - expiration: The refresh token expiration duration
	SetRefreshTokenExpiration(expiration time.Duration)
}