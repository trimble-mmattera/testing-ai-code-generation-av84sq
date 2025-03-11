// Package middleware provides HTTP middleware components for the Document Management Platform API.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin" // v1.9.0+

	"../../pkg/errors"
	"../../pkg/logger"
	errordto "../dto/error_dto"
)

// RecoveryMiddleware creates a Gin middleware that recovers from panics and returns a standardized error response
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Defer recovery function to catch any panics
		defer func() {
			if r := recover(); r != nil {
				// Get stack trace
				stack := debug.Stack()
				
				// Create a standardized internal error
				err := errors.NewInternalError("An unexpected error occurred")
				
				// Log the error and stack trace for investigation
				logger.ErrorContext(
					c.Request.Context(),
					"Panic recovered in API request",
					"error", err.Error(),
					"stack", string(stack),
					"panic", r,
				)
				
				// Abort the request chain with HTTP 500 status code
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					errordto.NewInternalErrorResponse(err),
				)
			}
		}()
		
		// Proceed with the next handler in the chain if no panic occurs
		c.Next()
	}
}