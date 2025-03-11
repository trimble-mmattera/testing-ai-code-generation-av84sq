// Package middleware provides a set of middleware functions for the Document Management Platform API.
// This file implements rate limiting middleware for the Document Management Platform API.
// This middleware restricts the number of requests a client can make within a specified time period
// to prevent abuse and ensure fair usage of the API. It supports both global rate limiting and
// endpoint-specific rate limiting with configurable limits.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin" // v1.9.0+
	"github.com/ulule/limiter/v3" // v3.10.0+
	"github.com/ulule/limiter/v3/drivers/store/memory" // v3.10.0+
	"github.com/ulule/limiter/v3/drivers/store/redis" // v3.10.0+

	"../../pkg/config"
	"../../pkg/errors"
	"../../pkg/logger"
	"../dto/error_dto"
)

// Default rate limits
const (
	defaultRate = "100-M"
	uploadRate = "10-M"
	searchRate = "50-M"
	rateLimitExceededMessage = "Rate limit exceeded. Please try again later."
)

// Rate limit headers
const (
	headerRateLimit = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset = "X-RateLimit-Reset"
)

// RateLimiterMiddleware creates a Gin middleware that applies rate limiting to requests
func RateLimiterMiddleware(cfg config.Config) gin.HandlerFunc {
	// Create a new rate limiter with default rate from configuration
	rate, err := parseRate(defaultRate)
	if err != nil {
		logger.Error("Failed to parse rate limit", "error", err.Error(), "rate", defaultRate)
		// Fallback to a sensible default if parsing fails
		rate = limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  100,
		}
	}
	
	// Create a store (memory or Redis based on configuration)
	store, err := createStore(cfg)
	if err != nil {
		logger.Error("Failed to create rate limit store", "error", err.Error())
		// Fallback to memory store if Redis fails
		store = memory.NewStore()
	}
	
	// Create a new limiter with the parsed rate and store
	rateLimiter := limiter.New(store, rate)
	
	// Return a Gin handler function that takes a gin.Context parameter
	return func(c *gin.Context) {
		// Extract client IP address from request
		clientIP := getClientIP(c)
		
		// Get rate limit context for the client
		limiterContext, err := rateLimiter.Get(c, clientIP)
		if err != nil {
			logger.ErrorContext(c.Request.Context(), "Failed to get rate limit context", "error", err.Error())
			c.Next()
			return
		}
		
		// Check if rate limit is exceeded
		if limiterContext.Reached {
			logger.InfoContext(c.Request.Context(), "Rate limit exceeded", 
				"client_ip", clientIP,
				"limit", limiterContext.Limit,
				"remaining", limiterContext.Remaining,
				"reset", limiterContext.Reset)
			
			// If exceeded, log the event and abort with 429 Too Many Requests
			validationErr := errors.NewValidationError(rateLimitExceededMessage)
			errorResponse := errordto.NewErrorResponse(validationErr)
			c.JSON(http.StatusTooManyRequests, errorResponse)
			c.Abort()
			return
		}
		
		// Set rate limit headers in the response
		setRateLimitHeaders(c, limiterContext)
		
		// Call the next handler in the chain
		c.Next()
	}
}

// UploadRateLimiterMiddleware creates a Gin middleware that applies stricter rate limiting to upload endpoints
func UploadRateLimiterMiddleware(cfg config.Config) gin.HandlerFunc {
	// Create a new rate limiter with upload-specific rate from configuration
	rate, err := parseRate(uploadRate)
	if err != nil {
		logger.Error("Failed to parse upload rate limit", "error", err.Error(), "rate", uploadRate)
		// Fallback to a sensible default if parsing fails
		rate = limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  10,
		}
	}
	
	// Create a store (memory or Redis based on configuration)
	store, err := createStore(cfg)
	if err != nil {
		logger.Error("Failed to create rate limit store for uploads", "error", err.Error())
		// Fallback to memory store if Redis fails
		store = memory.NewStore()
	}
	
	// Create a new limiter with the parsed rate and store
	rateLimiter := limiter.New(store, rate)
	
	// Return a Gin handler function that takes a gin.Context parameter
	return func(c *gin.Context) {
		// Extract client IP address and tenant ID from request
		clientIP := getClientIP(c)
		
		// Extract tenant ID from context (set by authentication middleware)
		tenantID, exists := c.Get("tenant_id")
		var key string
		if exists {
			// Create a composite key using IP and tenant ID
			key = clientIP + ":" + tenantID.(string)
		} else {
			key = clientIP
		}
		
		// Get rate limit context for the composite key
		limiterContext, err := rateLimiter.Get(c, key)
		if err != nil {
			logger.ErrorContext(c.Request.Context(), "Failed to get rate limit context for upload", "error", err.Error())
			c.Next()
			return
		}
		
		// Check if rate limit is exceeded
		if limiterContext.Reached {
			logger.WarnContext(c.Request.Context(), "Upload rate limit exceeded", 
				"client_ip", clientIP,
				"tenant_id", tenantID,
				"limit", limiterContext.Limit,
				"remaining", limiterContext.Remaining,
				"reset", limiterContext.Reset)
			
			// If exceeded, log the event and abort with 429 Too Many Requests
			validationErr := errors.NewValidationError(rateLimitExceededMessage)
			errorResponse := errordto.NewErrorResponse(validationErr)
			c.JSON(http.StatusTooManyRequests, errorResponse)
			c.Abort()
			return
		}
		
		// Set rate limit headers in the response
		setRateLimitHeaders(c, limiterContext)
		
		// Call the next handler in the chain
		c.Next()
	}
}

// SearchRateLimiterMiddleware creates a Gin middleware that applies specific rate limiting to search endpoints
func SearchRateLimiterMiddleware(cfg config.Config) gin.HandlerFunc {
	// Create a new rate limiter with search-specific rate from configuration
	rate, err := parseRate(searchRate)
	if err != nil {
		logger.Error("Failed to parse search rate limit", "error", err.Error(), "rate", searchRate)
		// Fallback to a sensible default if parsing fails
		rate = limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  50,
		}
	}
	
	// Create a store (memory or Redis based on configuration)
	store, err := createStore(cfg)
	if err != nil {
		logger.Error("Failed to create rate limit store for search", "error", err.Error())
		// Fallback to memory store if Redis fails
		store = memory.NewStore()
	}
	
	// Create a new limiter with the parsed rate and store
	rateLimiter := limiter.New(store, rate)
	
	// Return a Gin handler function that takes a gin.Context parameter
	return func(c *gin.Context) {
		// Extract client IP address and tenant ID from request
		clientIP := getClientIP(c)
		
		// Extract tenant ID from context (set by authentication middleware)
		tenantID, exists := c.Get("tenant_id")
		var key string
		if exists {
			// Create a composite key using IP and tenant ID
			key = clientIP + ":" + tenantID.(string)
		} else {
			key = clientIP
		}
		
		// Get rate limit context for the composite key
		limiterContext, err := rateLimiter.Get(c, key)
		if err != nil {
			logger.ErrorContext(c.Request.Context(), "Failed to get rate limit context for search", "error", err.Error())
			c.Next()
			return
		}
		
		// Check if rate limit is exceeded
		if limiterContext.Reached {
			logger.WarnContext(c.Request.Context(), "Search rate limit exceeded", 
				"client_ip", clientIP,
				"tenant_id", tenantID,
				"limit", limiterContext.Limit,
				"remaining", limiterContext.Remaining,
				"reset", limiterContext.Reset)
			
			// If exceeded, log the event and abort with 429 Too Many Requests
			validationErr := errors.NewValidationError(rateLimitExceededMessage)
			errorResponse := errordto.NewErrorResponse(validationErr)
			c.JSON(http.StatusTooManyRequests, errorResponse)
			c.Abort()
			return
		}
		
		// Set rate limit headers in the response
		setRateLimitHeaders(c, limiterContext)
		
		// Call the next handler in the chain
		c.Next()
	}
}

// TenantRateLimiterMiddleware creates a Gin middleware that applies tenant-specific rate limiting
func TenantRateLimiterMiddleware(cfg config.Config, rate string) gin.HandlerFunc {
	// Create a new rate limiter with the specified rate
	parsedRate, err := parseRate(rate)
	if err != nil {
		logger.Error("Failed to parse tenant rate limit", "error", err.Error(), "rate", rate)
		// Fallback to a sensible default if parsing fails
		parsedRate = limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  100,
		}
	}
	
	// Create a store (memory or Redis based on configuration)
	store, err := createStore(cfg)
	if err != nil {
		logger.Error("Failed to create rate limit store for tenant", "error", err.Error())
		// Fallback to memory store if Redis fails
		store = memory.NewStore()
	}
	
	// Create a new limiter with the parsed rate and store
	rateLimiter := limiter.New(store, parsedRate)
	
	// Return a Gin handler function that takes a gin.Context parameter
	return func(c *gin.Context) {
		// Extract tenant ID from request context
		tenantID, exists := c.Get("tenant_id")
		var key string
		if exists {
			key = tenantID.(string)
		} else {
			// If tenant ID is not found, use IP address as key
			key = getClientIP(c)
		}
		
		// Get rate limit context for the tenant ID
		limiterContext, err := rateLimiter.Get(c, key)
		if err != nil {
			logger.ErrorContext(c.Request.Context(), "Failed to get rate limit context for tenant", "error", err.Error())
			c.Next()
			return
		}
		
		// Check if rate limit is exceeded
		if limiterContext.Reached {
			logger.WarnContext(c.Request.Context(), "Tenant rate limit exceeded", 
				"tenant_id", tenantID,
				"limit", limiterContext.Limit,
				"remaining", limiterContext.Remaining,
				"reset", limiterContext.Reset)
			
			// If exceeded, log the event and abort with 429 Too Many Requests
			validationErr := errors.NewValidationError(rateLimitExceededMessage)
			errorResponse := errordto.NewErrorResponse(validationErr)
			c.JSON(http.StatusTooManyRequests, errorResponse)
			c.Abort()
			return
		}
		
		// Set rate limit headers in the response
		setRateLimitHeaders(c, limiterContext)
		
		// Call the next handler in the chain
		c.Next()
	}
}

// createStore creates a rate limiter store based on configuration
func createStore(cfg config.Config) (limiter.Store, error) {
	// Check if Redis is configured in the configuration
	// This is a simplified check - in a real implementation, you would check for Redis configuration
	// in the Config struct based on its actual structure
	hasRedis := false
	
	// If Redis is configured, create and return a Redis store
	if hasRedis {
		// This would use actual Redis configuration from the Config struct
		options := &redis.Options{
			Addr:     "localhost:6379", // Would be cfg.Redis.Address in real implementation
			Password: "",               // Would be cfg.Redis.Password in real implementation
			DB:       0,                // Would be cfg.Redis.DB in real implementation
		}
		
		return redis.NewStore(options)
	}
	
	// Otherwise, create and return an in-memory store
	return memory.NewStore(), nil
}

// parseRate parses a rate string into a limiter.Rate
func parseRate(rate string) (limiter.Rate, error) {
	// Use limiter.DefaultParser to parse the rate string
	return limiter.ParseRate(rate)
}

// setRateLimitHeaders sets rate limit headers in the response
func setRateLimitHeaders(c *gin.Context, context limiter.Context) {
	// Set X-RateLimit-Limit header with the rate limit
	c.Header(headerRateLimit, string(context.Limit))
	// Set X-RateLimit-Remaining header with remaining requests
	c.Header(headerRateRemaining, string(context.Remaining))
	// Set X-RateLimit-Reset header with reset time
	c.Header(headerRateReset, time.Unix(context.Reset, 0).Format(time.RFC3339))
}

// getClientIP extracts the client IP address from the request
func getClientIP(c *gin.Context) string {
	// Check for X-Forwarded-For header
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// If present, extract the first IP address
		ips := strings.Split(xff, ",")
		ip := strings.TrimSpace(ips[0])
		return ip
	}
	
	// If not present, use the request's remote address
	return c.ClientIP()
}