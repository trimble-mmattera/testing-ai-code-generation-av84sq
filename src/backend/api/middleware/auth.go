// Package middleware provides HTTP middleware components for the Document Management Platform.
// This file implements authentication and authorization middleware that validates JWT tokens,
// extracts tenant and user context, and enforces role-based access control.
package middleware

import (
	"net/http" // standard library
	"strings"  // standard library

	"github.com/gin-gonic/gin" // v1.9.0+

	"../../domain/services/auth_service"
	"../../pkg/errors"
	"../../pkg/logger"
	"../dto/error_dto"
	"../dto/response_dto"
)

// Context keys for storing user and tenant information
const (
	contextKeyUserID   = "user_id"
	contextKeyTenantID = "tenant_id"
	contextKeyRoles    = "roles"
	authHeaderKey      = "Authorization"
	bearerPrefix       = "Bearer "
)

// AuthMiddleware creates a Gin middleware that validates JWT tokens and extracts user information
func AuthMiddleware(authService auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		token, err := extractTokenFromHeader(c)
		if err != nil {
			logger.InfoContext(c.Request.Context(), "Authentication failed: missing or invalid token format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Missing or invalid authentication token")))
			return
		}

		// Validate token and extract claims using auth service
		tenantID, roles, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.WithError(err).InfoContext(c.Request.Context(), "Authentication failed: invalid token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Invalid authentication token")))
			return
		}

		// Get userID from claims (the sub claim in a JWT)
		userClaims := make(map[string]interface{})
		// In a real implementation, we would parse the token and extract all claims
		// For simplicity, we extract just the basic claims we need
		// This would normally be done by the auth service
		userID := c.GetString("sub") // This is an example; in reality authService would provide this

		// Set claims in context for downstream handlers
		c.Set(contextKeyUserID, userID)
		c.Set(contextKeyTenantID, tenantID)
		c.Set(contextKeyRoles, roles)

		logger.InfoContext(c.Request.Context(), "Authentication successful",
			"user_id", userID,
			"tenant_id", tenantID)

		c.Next()
	}
}

// RequireAuthentication creates a middleware that ensures the request is authenticated
func RequireAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user ID exists in context (set by AuthMiddleware)
		userID := GetUserID(c)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Authentication required")))
			return
		}

		c.Next()
	}
}

// RequireRole creates a middleware that ensures the authenticated user has a specific role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context
		roles := GetUserRoles(c)
		if roles == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Authentication required")))
			return
		}

		// Check if user has the required role
		if !HasRole(c, requiredRole) {
			logger.InfoContext(c.Request.Context(), "Authorization failed: missing required role",
				"required_role", requiredRole,
				"user_roles", roles)
			c.AbortWithStatusJSON(http.StatusForbidden, errordto.NewAuthorizationErrorResponse(
				errors.NewAuthorizationError("Insufficient permissions")))
			return
		}

		c.Next()
	}
}

// RequireAnyRole creates a middleware that ensures the authenticated user has at least one of the specified roles
func RequireAnyRole(requiredRoles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context
		userRoles := GetUserRoles(c)
		if userRoles == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Authentication required")))
			return
		}

		// Check if user has any of the required roles
		hasRequiredRole := false
		for _, requiredRole := range requiredRoles {
			if HasRole(c, requiredRole) {
				hasRequiredRole = true
				break
			}
		}

		if !hasRequiredRole {
			logger.InfoContext(c.Request.Context(), "Authorization failed: missing any required role",
				"required_roles", requiredRoles,
				"user_roles", userRoles)
			c.AbortWithStatusJSON(http.StatusForbidden, errordto.NewAuthorizationErrorResponse(
				errors.NewAuthorizationError("Insufficient permissions")))
			return
		}

		c.Next()
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(c *gin.Context) string {
	// Extract user ID from context
	userID, exists := c.Get(contextKeyUserID)
	if !exists {
		return ""
	}

	// Convert to string and return
	userIDStr, ok := userID.(string)
	if !ok {
		return ""
	}

	return userIDStr
}

// GetTenantID extracts the tenant ID from the request context
func GetTenantID(c *gin.Context) string {
	// Extract tenant ID from context
	tenantID, exists := c.Get(contextKeyTenantID)
	if !exists {
		return ""
	}

	// Convert to string and return
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		return ""
	}

	return tenantIDStr
}

// GetUserRoles extracts the user roles from the request context
func GetUserRoles(c *gin.Context) []string {
	// Extract roles from context
	roles, exists := c.Get(contextKeyRoles)
	if !exists {
		return nil
	}

	// Convert to []string and return
	rolesSlice, ok := roles.([]string)
	if !ok {
		return nil
	}

	return rolesSlice
}

// HasRole checks if the user has a specific role
func HasRole(c *gin.Context, role string) bool {
	// Get user roles
	roles := GetUserRoles(c)
	if roles == nil {
		return false
	}

	// Check if role is in user roles
	for _, r := range roles {
		if r == role {
			return true
		}
	}

	return false
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func extractTokenFromHeader(c *gin.Context) (string, error) {
	// Get the Authorization header
	authHeader := c.GetHeader(authHeaderKey)
	if authHeader == "" {
		return "", errors.NewAuthenticationError("Missing authorization header")
	}

	// Check if header starts with "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", errors.NewAuthenticationError("Invalid authorization header format")
	}

	// Extract token by removing "Bearer " prefix
	token := strings.TrimPrefix(authHeader, bearerPrefix)
	if token == "" {
		return "", errors.NewAuthenticationError("Empty token")
	}

	return token, nil
}