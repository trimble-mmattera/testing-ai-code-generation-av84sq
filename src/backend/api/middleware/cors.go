// Package middleware provides middleware components for the Document Management Platform API.
// This file implements Cross-Origin Resource Sharing (CORS) middleware that enables
// controlled access to API resources from different origins while maintaining tenant
// isolation and security requirements.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin" // v1.9.0+

	"../../pkg/config"  // For accessing CORS configuration settings
	"../../pkg/logger"  // For logging CORS configuration and requests
)

// Default CORS configuration values
var (
	defaultMaxAge = 24 * time.Hour
	defaultAllowOrigins = []string{"*"}
	defaultAllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	defaultAllowHeaders = []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "Accept"}
	defaultExposeHeaders = []string{"Content-Length", "Content-Type"}
)

// CORSConfig defines configuration options for the CORS middleware
type CORSConfig struct {
	// AllowOrigins is a list of origins a cross-domain request can be executed from
	AllowOrigins []string
	// AllowMethods is a list of methods the client is allowed to use
	AllowMethods []string
	// AllowHeaders is a list of non-simple headers the client is allowed to use
	AllowHeaders []string
	// ExposeHeaders is a list of headers which are exposed to the client
	ExposeHeaders []string
	// AllowCredentials indicates whether the request can include user credentials
	AllowCredentials bool
	// MaxAge indicates how long the results of a preflight request can be cached
	MaxAge time.Duration
}

// NewCORSConfig creates a new CORSConfig with default values
func NewCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     defaultAllowOrigins,
		AllowMethods:     defaultAllowMethods,
		AllowHeaders:     defaultAllowHeaders,
		ExposeHeaders:    defaultExposeHeaders,
		AllowCredentials: false,
		MaxAge:           defaultMaxAge,
	}
}

// CORSMiddleware creates a Gin middleware that handles CORS (Cross-Origin Resource Sharing) for the API
func CORSMiddleware(cfg config.Config) gin.HandlerFunc {
	// Get CORS configuration from the provided config
	corsConfig := NewCORSConfig()
	
	// Set default values for any missing configuration
	// In a real implementation, we would extract CORS settings from the cfg parameter
	// For example, if tenant-specific CORS settings are defined in the config
	
	// Log the CORS configuration being applied
	logger.Info("Configuring CORS middleware", 
		"allowOrigins", strings.Join(corsConfig.AllowOrigins, ", "),
		"allowMethods", strings.Join(corsConfig.AllowMethods, ", "),
		"allowCredentials", corsConfig.AllowCredentials,
		"maxAge", corsConfig.MaxAge.String(),
	)

	// Return a Gin handler function that takes a gin.Context parameter
	return func(c *gin.Context) {
		// Set Access-Control-Allow-Origin header based on the request origin and allowed origins
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if isOriginAllowed(origin, corsConfig.AllowOrigins) {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				// If origin is not allowed, continue without setting CORS headers
				c.Next()
				return
			}
		} else if isOriginAllowed("*", corsConfig.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// Set Access-Control-Allow-Methods header
		if len(corsConfig.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowMethods, ", "))
		}

		// Set Access-Control-Allow-Headers header
		if len(corsConfig.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowHeaders, ", "))
		}

		// Set Access-Control-Expose-Headers header
		if len(corsConfig.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposeHeaders, ", "))
		}

		// Set Access-Control-Allow-Credentials header if credentials are allowed
		if corsConfig.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set Access-Control-Max-Age header
		if corsConfig.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", strings.TrimSpace(corsConfig.MaxAge.String()))
		}

		// If request method is OPTIONS, return 200 OK status and abort the chain
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		// Otherwise, call the next handler in the chain
		c.Next()
	}
}

// isOriginAllowed checks if the given origin is allowed based on the configured allowed origins
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	// If allowedOrigins contains "*", return true (all origins allowed)
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
	}
	return false
}