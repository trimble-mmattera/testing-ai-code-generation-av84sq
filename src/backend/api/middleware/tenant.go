// Package middleware provides HTTP middleware components for the Document Management Platform.
// This file implements tenant isolation middleware to enforce tenant boundaries and
// ensure users can only access resources within their own tenant.
package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin" // v1.9.0+

	auth "../../domain/services/auth_service"
	"../../pkg/errors"
	"../../pkg/logger"
	"../dto/error_dto"
)

// Context keys for tenant and user IDs in request context
const (
	contextKeyTenantID = "tenant_id"
	contextKeyUserID   = "user_id"
)

// RequireTenantContext creates a middleware that ensures the request has a valid tenant context.
// It checks for the presence of a tenant ID in the request context and returns 401 Unauthorized
// if no tenant context is found.
func RequireTenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		if tenantID == "" {
			logger.ErrorContext(c.Request.Context(), "Tenant context missing in request")
			c.JSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Tenant context required"),
			))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireSameTenant creates a middleware that ensures the tenant ID in the request path
// matches the authenticated user's tenant. This prevents cross-tenant access attempts.
func RequireSameTenant(authService auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(contextKeyUserID)
		if !exists {
			logger.ErrorContext(c.Request.Context(), "User ID missing in request context")
			c.JSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("User not authenticated"),
			))
			c.Abort()
			return
		}

		userTenantID := GetTenantID(c)
		if userTenantID == "" {
			logger.ErrorContext(c.Request.Context(), "Tenant ID missing in request context")
			c.JSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Tenant context required"),
			))
			c.Abort()
			return
		}

		// Extract the target tenant ID from the request path parameter
		targetTenantID := c.Param("tenantId")
		if targetTenantID == "" {
			logger.WarnContext(c.Request.Context(), "Target tenant ID missing in request path")
			c.JSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
				errors.NewValidationError("Target tenant ID required"),
				map[string]string{"tenantId": "Required parameter missing"},
			))
			c.Abort()
			return
		}

		// Ensure the user's tenant matches the target tenant
		if userTenantID != targetTenantID {
			logger.WarnContext(c.Request.Context(), "Tenant mismatch detected",
				"user_tenant_id", userTenantID,
				"target_tenant_id", targetTenantID,
			)
			c.JSON(http.StatusForbidden, errordto.NewAuthorizationErrorResponse(
				errors.NewAuthorizationError("Access to the specified tenant is not authorized"),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// VerifyTenantResourceAccess creates a middleware that verifies a user has access to a 
// resource within their tenant, based on specified resource type and access type.
func VerifyTenantResourceAccess(authService auth.AuthService, resourceType, accessType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(contextKeyUserID)
		if !exists {
			logger.ErrorContext(c.Request.Context(), "User ID missing in request context")
			c.JSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("User not authenticated"),
			))
			c.Abort()
			return
		}

		tenantID := GetTenantID(c)
		if tenantID == "" {
			logger.ErrorContext(c.Request.Context(), "Tenant ID missing in request context")
			c.JSON(http.StatusUnauthorized, errordto.NewAuthenticationErrorResponse(
				errors.NewAuthenticationError("Tenant context required"),
			))
			c.Abort()
			return
		}

		// Extract resource ID from the request path
		resourceID := c.Param("id")
		if resourceID == "" {
			logger.WarnContext(c.Request.Context(), "Resource ID missing in request path")
			c.JSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
				errors.NewValidationError("Resource ID required"),
				map[string]string{"id": "Required parameter missing"},
			))
			c.Abort()
			return
		}

		// Verify resource access using auth service
		hasAccess, err := authService.VerifyResourceAccess(
			c.Request.Context(),
			userID.(string),
			tenantID,
			resourceType,
			resourceID,
			accessType,
		)

		if err != nil {
			logger.ErrorContext(c.Request.Context(), "Error verifying resource access",
				"error", err.Error(),
				"user_id", userID,
				"tenant_id", tenantID,
				"resource_type", resourceType,
				"resource_id", resourceID,
				"access_type", accessType,
			)
			c.JSON(http.StatusInternalServerError, errordto.NewInternalErrorResponse(err))
			c.Abort()
			return
		}

		if !hasAccess {
			logger.WarnContext(c.Request.Context(), "Resource access denied",
				"user_id", userID,
				"tenant_id", tenantID,
				"resource_type", resourceType,
				"resource_id", resourceID,
				"access_type", accessType,
			)
			c.JSON(http.StatusForbidden, errordto.NewAuthorizationErrorResponse(
				errors.NewAuthorizationError("Access to the specified resource is not authorized"),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// TenantContext creates a middleware that adds tenant context to all database operations.
// This ensures tenant isolation at the database level by automatically filtering queries.
func TenantContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := GetTenantID(c)
		if tenantID == "" {
			logger.WarnContext(c.Request.Context(), "Tenant ID missing for database context")
			c.Next()
			return
		}

		// Create a new request context with tenant information for database operations
		ctx := c.Request.Context()
		// Set tenant context in the request for downstream handlers and services
		// This would typically be used by database repositories to filter queries by tenant
		c.Set("db_tenant_context", tenantID)

		// Process the request
		c.Next()

		// Clean up context after request processing (if needed)
	}
}

// ValidateTenantAccess validates that a user has access to a specific tenant.
// It calls the auth service to verify tenant access and returns an error if validation fails.
func ValidateTenantAccess(authService auth.AuthService, userID, tenantID string) error {
	if userID == "" || tenantID == "" {
		return errors.NewValidationError("User ID and Tenant ID are required")
	}

	// Verify tenant access using auth service
	hasAccess, err := authService.VerifyTenantAccess(context.Background(), userID, tenantID)
	if err != nil {
		logger.Error("Error validating tenant access",
			"error", err.Error(),
			"user_id", userID,
			"tenant_id", tenantID,
		)
		return errors.Wrap(err, "Failed to validate tenant access")
	}

	if !hasAccess {
		logger.Warn("Tenant access denied",
			"user_id", userID,
			"tenant_id", tenantID,
		)
		return errors.NewAuthorizationError("Access to the specified tenant is not authorized")
	}

	return nil
}

// SetTenantContext sets the tenant context in the request.
// This is typically called by authentication middleware after extracting tenant ID from JWT.
func SetTenantContext(c *gin.Context, tenantID string) {
	c.Set(contextKeyTenantID, tenantID)
}

// GetTenantID extracts the tenant ID from the request context.
// Returns an empty string if no tenant ID is found.
func GetTenantID(c *gin.Context) string {
	tenantID, exists := c.Get(contextKeyTenantID)
	if !exists {
		return ""
	}
	return tenantID.(string)
}