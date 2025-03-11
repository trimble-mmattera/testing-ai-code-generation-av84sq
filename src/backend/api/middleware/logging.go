// Package middleware provides HTTP middleware components for the Document Management Platform API.
// It includes middleware for logging, authentication, authorization, and other cross-cutting concerns.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin" // v1.9.0+
	"github.com/google/uuid"    // v1.3.0+

	"../../pkg/logger"
)

// Context and header constants for request identification
const (
	// contextKeyRequestID is the key used to store request ID in the context
	contextKeyRequestID = "request_id"
	
	// headerRequestID is the HTTP header name for the request ID
	headerRequestID = "X-Request-ID"
)

// LoggingMiddleware creates a Gin middleware that logs HTTP requests and responses.
// It captures request details, generates unique request IDs, measures response times,
// and logs the outcome of each API call in a structured format.
//
// This middleware helps with:
// - Request tracking across microservices
// - Performance monitoring of API endpoints
// - Security audit logging of API access
// - Debugging of API issues
//
// It integrates with the application's structured logging system to ensure
// consistent log format across all components.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a unique request ID
		requestID := generateRequestID()

		// Set request ID in context and response header
		setRequestIDInContext(c, requestID)
		c.Header(headerRequestID, requestID)

		// Record start time
		startTime := time.Now()

		// Log incoming request with structured fields
		logger.InfoContext(c.Request.Context(), "API Request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"request_id", requestID,
		)

		// Process request (continue to the next middleware/handler)
		c.Next()

		// Calculate request duration after processing
		duration := time.Since(startTime)

		// Get status code after request is processed
		statusCode := c.Writer.Status()

		// Prepare log fields for the response
		logFields := []interface{}{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"request_id", requestID,
			"bytes_out", c.Writer.Size(),
		}

		// If path has query parameters, log them
		if c.Request.URL.RawQuery != "" {
			logFields = append(logFields, "query", c.Request.URL.RawQuery)
		}

		// If there are errors from handlers, include them in the log
		if len(c.Errors) > 0 {
			logFields = append(logFields, "errors", c.Errors.String())
		}

		// Log at appropriate level based on status code
		if statusCode >= http.StatusBadRequest {
			logger.ErrorContext(c.Request.Context(), "API Response", logFields...)
		} else {
			logger.InfoContext(c.Request.Context(), "API Response", logFields...)
		}
	}
}

// GetRequestID extracts the request ID from the context.
// Returns an empty string if the request ID is not found.
//
// This function is useful for other middleware or handlers that need
// to access the request ID for correlation or logging purposes.
func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get(contextKeyRequestID)
	if !exists {
		return ""
	}
	return requestID.(string)
}

// generateRequestID creates a new unique request ID using UUID.
// This ensures that each request can be uniquely identified across the system.
func generateRequestID() string {
	return uuid.New().String()
}

// setRequestIDInContext sets the request ID in both the Gin context and request context.
// This ensures the request ID is available both to Gin middleware/handlers and through
// the standard context mechanism for use with other packages.
func setRequestIDInContext(c *gin.Context, requestID string) {
	// Set in Gin context
	c.Set(contextKeyRequestID, requestID)

	// Set in request context
	ctx := context.WithValue(c.Request.Context(), contextKeyRequestID, requestID)
	c.Request = c.Request.WithContext(ctx)
}