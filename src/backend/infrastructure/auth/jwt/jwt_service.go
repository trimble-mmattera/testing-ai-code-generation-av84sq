// Package jwt provides JWT-based implementation of the authentication service.
package jwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/golang-jwt/jwt/v5" // v5.0.0+

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../domain/services"
	"../../../pkg/config"
	"../../../pkg/errors"
)

// Default token expiration durations
var (
	defaultTokenExpiration        = time.Hour
	defaultRefreshTokenExpiration = time.Hour * 24 * 7
)

// jwtService implements the auth.AuthService interface using JWT
type jwtService struct {
	userRepo               repositories.UserRepository
	tenantRepo             repositories.TenantRepository
	privateKey             *rsa.PrivateKey
	publicKey              *rsa.PublicKey
	issuer                 string
	tokenExpiration        time.Duration
	refreshTokenExpiration time.Duration
}

// customClaims defines the JWT claims structure
type customClaims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles,omitempty"`
	Type     string   `json:"type,omitempty"`
}

// NewJWTService creates a new JWT authentication service
func NewJWTService(userRepo repositories.UserRepository, tenantRepo repositories.TenantRepository, cfg config.JWTConfig) (services.AuthService, error) {
	// Validate input parameters
	if userRepo == nil {
		return nil, errors.NewValidationError("user repository is required")
	}
	if tenantRepo == nil {
		return nil, errors.NewValidationError("tenant repository is required")
	}

	// Parse private key from PEM format
	privateKeyBlock, _ := pem.Decode([]byte(cfg.PrivateKey))
	if privateKeyBlock == nil {
		return nil, errors.NewValidationError("failed to parse private key PEM")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse private key")
	}

	// Parse public key from PEM format
	publicKeyBlock, _ := pem.Decode([]byte(cfg.PublicKey))
	if publicKeyBlock == nil {
		return nil, errors.NewValidationError("failed to parse public key PEM")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse public key")
	}

	// Create and return the JWT service
	service := &jwtService{
		userRepo:               userRepo,
		tenantRepo:             tenantRepo,
		privateKey:             privateKey,
		publicKey:              publicKey,
		issuer:                 cfg.Issuer,
		tokenExpiration:        defaultTokenExpiration,
		refreshTokenExpiration: defaultRefreshTokenExpiration,
	}

	return service, nil
}

// Authenticate validates user credentials and returns a refresh token if successful
func (s *jwtService) Authenticate(ctx context.Context, tenantID, usernameOrEmail, password string) (string, error) {
	// Validate inputs
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID is required")
	}
	if usernameOrEmail == "" {
		return "", errors.NewValidationError("username or email is required")
	}
	if password == "" {
		return "", errors.NewValidationError("password is required")
	}

	// Get tenant and verify it's active
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", errors.NewAuthenticationError("invalid tenant")
		}
		return "", errors.Wrap(err, "failed to get tenant")
	}
	if tenant.Status != "active" {
		return "", errors.NewAuthenticationError("tenant is not active")
	}

	// Try to get user by username
	var user *models.User
	user, err = s.userRepo.GetByUsername(ctx, usernameOrEmail, tenantID)
	if err != nil && !errors.IsResourceNotFoundError(err) {
		return "", errors.Wrap(err, "failed to get user by username")
	}

	// If not found by username, try by email
	if user == nil {
		user, err = s.userRepo.GetByEmail(ctx, usernameOrEmail, tenantID)
		if err != nil {
			if errors.IsResourceNotFoundError(err) {
				return "", errors.NewAuthenticationError("invalid credentials")
			}
			return "", errors.Wrap(err, "failed to get user by email")
		}
	}

	// Verify user belongs to the tenant
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
		return "", errors.Wrap(err, "failed to verify password")
	}
	if !match {
		return "", errors.NewAuthenticationError("invalid credentials")
	}

	// Generate access token with user ID, tenant ID, and roles
	accessToken, err := s.GenerateToken(ctx, user.ID, user.TenantID, user.Roles, s.tokenExpiration)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate access token")
	}

	// Generate refresh token with user ID and tenant ID
	refreshToken, err := s.GenerateRefreshToken(ctx, user.ID, user.TenantID, s.refreshTokenExpiration)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate refresh token")
	}

	// In a real implementation, the access token would typically be returned alongside
	// the refresh token or set in a secure HTTP-only cookie
	// For the purpose of this implementation, we'll just return the refresh token as specified
	// in the interface
	return refreshToken, nil
}

// ValidateToken validates an access token and extracts user information
func (s *jwtService) ValidateToken(ctx context.Context, token string) (string, []string, error) {
	// Parse and validate token
	parsedToken, err := s.parseToken(token)
	if err != nil {
		return "", nil, errors.NewAuthenticationError("invalid token: " + err.Error())
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, errors.NewAuthenticationError("invalid token claims")
	}

	// Validate claims
	if err := s.validateClaims(claims); err != nil {
		return "", nil, err
	}

	// Extract user ID, tenant ID, and roles
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", nil, errors.NewAuthenticationError("invalid token: missing user ID")
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok || tenantID == "" {
		return "", nil, errors.NewAuthenticationError("invalid token: missing tenant ID")
	}

	// Extract roles (if present)
	var roles []string
	if rolesInterface, ok := claims["roles"]; ok {
		if rolesArr, ok := rolesInterface.([]interface{}); ok {
			roles = make([]string, len(rolesArr))
			for i, role := range rolesArr {
				if roleStr, ok := role.(string); ok {
					roles[i] = roleStr
				}
			}
		}
	}

	// Verify user exists and is active
	user, err := s.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", nil, errors.NewAuthenticationError("user not found")
		}
		return "", nil, errors.Wrap(err, "failed to get user")
	}

	if !user.IsActive() {
		return "", nil, errors.NewAuthenticationError("user account is not active")
	}

	// Verify tenant exists and is active
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", nil, errors.NewAuthenticationError("tenant not found")
		}
		return "", nil, errors.Wrap(err, "failed to get tenant")
	}

	if tenant.Status != "active" {
		return "", nil, errors.NewAuthenticationError("tenant is not active")
	}

	return tenantID, roles, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *jwtService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Parse and validate token
	parsedToken, err := s.parseToken(refreshToken)
	if err != nil {
		return "", errors.NewAuthenticationError("invalid refresh token: " + err.Error())
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.NewAuthenticationError("invalid token claims")
	}

	// Validate claims
	if err := s.validateClaims(claims); err != nil {
		return "", err
	}

	// Check token type is refresh
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return "", errors.NewAuthenticationError("invalid token type")
	}

	// Extract user ID and tenant ID
	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", errors.NewAuthenticationError("invalid token: missing user ID")
	}

	tenantID, ok := claims["tenant_id"].(string)
	if !ok || tenantID == "" {
		return "", errors.NewAuthenticationError("invalid token: missing tenant ID")
	}

	// Verify user exists and is active
	user, err := s.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", errors.NewAuthenticationError("user not found")
		}
		return "", errors.Wrap(err, "failed to get user")
	}

	if !user.IsActive() {
		return "", errors.NewAuthenticationError("user account is not active")
	}

	// Verify tenant exists and is active
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return "", errors.NewAuthenticationError("tenant not found")
		}
		return "", errors.Wrap(err, "failed to get tenant")
	}

	if tenant.Status != "active" {
		return "", errors.NewAuthenticationError("tenant is not active")
	}

	// Generate new tokens
	accessToken, err := s.GenerateToken(ctx, user.ID, user.TenantID, user.Roles, s.tokenExpiration)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate access token")
	}

	newRefreshToken, err := s.GenerateRefreshToken(ctx, user.ID, user.TenantID, s.refreshTokenExpiration)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate refresh token")
	}

	// Return new refresh token as specified in the interface
	return newRefreshToken, nil
}

// InvalidateToken invalidates a token (logout)
func (s *jwtService) InvalidateToken(ctx context.Context, token string) error {
	// JWT is stateless by design and cannot be invalidated without maintaining a blacklist
	// In a production system, you would typically implement a token blacklist using Redis
	// or another data store to track invalidated tokens until they expire
	// For now, this is essentially a no-op since the interface expects it
	return nil
}

// VerifyPermission verifies if a user has a specific permission
func (s *jwtService) VerifyPermission(ctx context.Context, userID, tenantID, permission string) (bool, error) {
	// Validate inputs
	if userID == "" {
		return false, errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID is required")
	}
	if permission == "" {
		return false, errors.NewValidationError("permission is required")
	}

	// Get user from repository
	user, err := s.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return false, nil // User not found, no permission
		}
		return false, errors.Wrap(err, "failed to get user")
	}

	// Verify user belongs to the specified tenant
	if user.TenantID != tenantID {
		return false, nil // User doesn't belong to the tenant, no permission
	}

	// Check permission based on user roles
	switch permission {
	case services.PermissionRead:
		return user.CanRead(), nil
	case services.PermissionWrite:
		return user.CanWrite(), nil
	case services.PermissionDelete:
		return user.CanDelete(), nil
	case services.PermissionManageFolders:
		return user.CanManageFolders(), nil
	default:
		return false, errors.NewValidationError("invalid permission: " + permission)
	}
}

// VerifyResourceAccess verifies if a user has access to a specific resource
func (s *jwtService) VerifyResourceAccess(ctx context.Context, userID, tenantID, resourceType, resourceID, accessType string) (bool, error) {
	// Validate inputs
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

	// First, verify tenant context (user belongs to the tenant)
	hasTenantAccess, err := s.VerifyTenantAccess(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	if !hasTenantAccess {
		return false, nil // User doesn't belong to the tenant, no access
	}

	// Map access type to permission
	var permission string
	switch accessType {
	case "read":
		permission = services.PermissionRead
	case "write":
		permission = services.PermissionWrite
	case "delete":
		permission = services.PermissionDelete
	case "manage_folders":
		permission = services.PermissionManageFolders
	default:
		return false, errors.NewValidationError("invalid access type: " + accessType)
	}

	// Check if user has the required permission
	return s.VerifyPermission(ctx, userID, tenantID, permission)
}

// VerifyTenantAccess verifies if a user belongs to a specific tenant
func (s *jwtService) VerifyTenantAccess(ctx context.Context, userID, tenantID string) (bool, error) {
	// Validate inputs
	if userID == "" {
		return false, errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return false, errors.NewValidationError("tenant ID is required")
	}

	// Get user from repository
	user, err := s.userRepo.GetByID(ctx, userID, tenantID)
	if err != nil {
		if errors.IsResourceNotFoundError(err) {
			return false, nil // User not found, no access
		}
		return false, errors.Wrap(err, "failed to get user")
	}

	// Check if user belongs to the tenant
	return user.TenantID == tenantID, nil
}

// GenerateToken generates a new access token for a user
func (s *jwtService) GenerateToken(ctx context.Context, userID, tenantID string, roles []string, expiration time.Duration) (string, error) {
	// Validate inputs
	if userID == "" {
		return "", errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID is required")
	}
	if roles == nil {
		roles = []string{} // Default to empty array if not provided
	}
	if expiration <= 0 {
		expiration = s.tokenExpiration // Use default if not specified
	}

	// Create token with claims
	now := time.Now()
	claims := customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			Issuer:    s.issuer,
		},
		TenantID: tenantID,
		Roles:    roles,
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to sign token")
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a new refresh token for a user
func (s *jwtService) GenerateRefreshToken(ctx context.Context, userID, tenantID string, expiration time.Duration) (string, error) {
	// Validate inputs
	if userID == "" {
		return "", errors.NewValidationError("user ID is required")
	}
	if tenantID == "" {
		return "", errors.NewValidationError("tenant ID is required")
	}
	if expiration <= 0 {
		expiration = s.refreshTokenExpiration // Use default if not specified
	}

	// Create token with claims
	now := time.Now()
	claims := customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			Issuer:    s.issuer,
		},
		TenantID: tenantID,
		Type:     "refresh", // Mark as refresh token
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to sign refresh token")
	}

	return signedToken, nil
}

// SetTokenExpiration sets the token expiration duration
func (s *jwtService) SetTokenExpiration(expiration time.Duration) {
	if expiration > 0 {
		s.tokenExpiration = expiration
	}
}

// SetRefreshTokenExpiration sets the refresh token expiration duration
func (s *jwtService) SetRefreshTokenExpiration(expiration time.Duration) {
	if expiration > 0 {
		s.refreshTokenExpiration = expiration
	}
}

// parseToken is an internal helper to parse and validate a JWT token
func (s *jwtService) parseToken(tokenString string) (*jwt.Token, error) {
	// Parse the token with the public key
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.NewAuthenticationError("unexpected signing method: " + token.Method.Alg())
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

// validateClaims is an internal helper to validate token claims
func (s *jwtService) validateClaims(claims jwt.MapClaims) error {
	// Check required claims are present
	if claims["sub"] == nil {
		return errors.NewAuthenticationError("invalid token: missing subject claim")
	}
	if claims["tenant_id"] == nil {
		return errors.NewAuthenticationError("invalid token: missing tenant_id claim")
	}
	if claims["exp"] == nil {
		return errors.NewAuthenticationError("invalid token: missing expiration claim")
	}
	if claims["iat"] == nil {
		return errors.NewAuthenticationError("invalid token: missing issued at claim")
	}
	if claims["iss"] == nil {
		return errors.NewAuthenticationError("invalid token: missing issuer claim")
	}

	// Verify issuer matches the configured issuer
	if issuer, ok := claims["iss"].(string); !ok || issuer != s.issuer {
		return errors.NewAuthenticationError("invalid token: invalid issuer")
	}

	// Verify expiration time is in the future
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return errors.NewAuthenticationError("invalid token: invalid expiration format")
	}
	exp := time.Unix(int64(expFloat), 0)
	if time.Now().After(exp) {
		return errors.NewAuthenticationError("token has expired")
	}

	// Verify issued at time is in the past
	iatFloat, ok := claims["iat"].(float64)
	if !ok {
		return errors.NewAuthenticationError("invalid token: invalid issued at format")
	}
	iat := time.Unix(int64(iatFloat), 0)
	if time.Now().Before(iat) {
		return errors.NewAuthenticationError("token used before issued time")
	}

	return nil
}