// Package handlers provides HTTP handlers for the Document Management Platform API.
// This file implements health check endpoints for Kubernetes liveness, readiness, and deep health probes.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin" // v1.9.0+
	"context" // standard library
	"fmt" // standard library
	"time" // standard library

	"../dto" // For creating standardized API responses
	"../../pkg/logger" // For structured logging of health check operations
	"../../pkg/errors" // For standardized error handling
)

// HealthChecker is an interface for components that can be health-checked
type HealthChecker interface {
	// Check performs a health check on the component
	Check(ctx context.Context) (interface{}, error)
}

// HealthHandler handles health check endpoints
type HealthHandler struct {
	checkers map[string]HealthChecker
}

// NewHealthHandler creates a new HealthHandler with the provided health checkers
func NewHealthHandler(checkers map[string]HealthChecker) *HealthHandler {
	return &HealthHandler{
		checkers: checkers,
	}
}

// LivenessCheck handles the liveness probe endpoint
// This is a basic check that the application is running
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	logger.InfoContext(c.Request.Context(), "Liveness check requested")
	c.JSON(http.StatusOK, dto.NewDataResponse(map[string]bool{"alive": true}))
}

// ReadinessCheck handles the readiness probe endpoint
// This checks if the application is ready to handle requests
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	logger.InfoContext(c.Request.Context(), "Readiness check requested")
	
	// Create context with timeout for readiness check
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	// Check all dependencies
	status, err := h.checkDependencies(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Readiness check failed", "error", err.Error())
		c.JSON(http.StatusServiceUnavailable, dto.NewDependencyErrorResponse(err.Error()))
		return
	}
	
	// Return success with dependency status
	c.JSON(http.StatusOK, dto.NewDataResponse(status))
}

// DeepHealthCheck handles the deep health check endpoint
// This performs a more thorough check of all system components
func (h *HealthHandler) DeepHealthCheck(c *gin.Context) {
	logger.InfoContext(c.Request.Context(), "Deep health check requested")
	
	// Create context with longer timeout for deep health check
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	
	// Check all dependencies
	status, err := h.checkDependencies(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Deep health check failed", "error", err.Error())
		c.JSON(http.StatusServiceUnavailable, dto.NewDependencyErrorResponse(err.Error()))
		return
	}
	
	// Return success with detailed status information
	c.JSON(http.StatusOK, dto.NewDataResponse(status))
}

// checkDependencies checks the health of all registered dependencies
func (h *HealthHandler) checkDependencies(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})
	
	// Check each registered health checker
	for name, checker := range h.checkers {
		result, err := checker.Check(ctx)
		if err != nil {
			return nil, errors.NewDependencyError(fmt.Sprintf("dependency %s check failed: %s", name, err.Error()))
		}
		status[name] = result
	}
	
	return status, nil
}

// DatabaseHealthChecker implements health checking for database connections
type DatabaseHealthChecker struct {
	// In a real implementation, this would have a reference to the database
}

// NewDatabaseHealthChecker creates a new DatabaseHealthChecker
func NewDatabaseHealthChecker() *DatabaseHealthChecker {
	return &DatabaseHealthChecker{}
}

// Check performs a health check on the database
func (c *DatabaseHealthChecker) Check(ctx context.Context) (interface{}, error) {
	// In a real implementation, this would execute a simple query to verify connection
	// and collect connection pool statistics
	
	// Simulate database check
	// If database connection fails, return an error instead
	return map[string]interface{}{
		"status": "connected",
		"latency_ms": 5,
		"connections": map[string]interface{}{
			"active": 5,
			"idle": 10,
			"max": 25,
		},
	}, nil
}

// StorageHealthChecker implements health checking for S3 storage
type StorageHealthChecker struct {
	storageService services.StorageService
}

// NewStorageHealthChecker creates a new StorageHealthChecker
func NewStorageHealthChecker(storageService services.StorageService) *StorageHealthChecker {
	return &StorageHealthChecker{
		storageService: storageService,
	}
}

// Check performs a health check on the storage service
func (c *StorageHealthChecker) Check(ctx context.Context) (interface{}, error) {
	// In a real implementation, this would check connectivity to S3
	// and collect metrics about storage service operations
	
	// Simulate storage service check
	// If S3 connectivity fails, return an error instead
	return map[string]interface{}{
		"status": "connected",
		"buckets_accessible": true,
		"operations": map[string]interface{}{
			"uploads_last_minute": 12,
			"downloads_last_minute": 35,
			"avg_latency_ms": 125,
		},
	}, nil
}

// SearchHealthChecker implements health checking for Elasticsearch service
type SearchHealthChecker struct {
	searchService services.SearchService
}

// NewSearchHealthChecker creates a new SearchHealthChecker
func NewSearchHealthChecker(searchService services.SearchService) *SearchHealthChecker {
	return &SearchHealthChecker{
		searchService: searchService,
	}
}

// Check performs a health check on the search service
func (c *SearchHealthChecker) Check(ctx context.Context) (interface{}, error) {
	// In a real implementation, this would check Elasticsearch cluster health
	// and collect metrics about search service operations
	
	// Simulate search service check
	// If Elasticsearch is unhealthy, return an error instead
	return map[string]interface{}{
		"status": "green",
		"nodes": 3,
		"indices": 5,
		"search_performance": map[string]interface{}{
			"queries_last_minute": 47,
			"avg_query_time_ms": 75,
		},
	}, nil
}