// Package api provides the HTTP API layer for the Document Management Platform.
package api

import (
	"github.com/gin-gonic/gin" // v1.9.0+
	"net/http" // standard library
	"github.com/project/handlers" // latest
	"github.com/project/middleware" // latest
	"github.com/project/config" // latest
	"github.com/sirupsen/logrus" // v1.9.0+
	"github.com/project/application/usecases" // latest
	"github.com/project/domain/services/auth" // latest
)

// apiVersionPrefix defines the API version prefix for all routes
const apiVersionPrefix = "/api/v1"

// SetupRouter sets up the main router for the Document Management Platform API
// It configures all routes, middleware, and connects API endpoints to the appropriate use cases
func SetupRouter(
	cfg config.Config,
	documentUseCase usecases.DocumentUseCase,
	folderUseCase usecases.FolderUseCase,
	searchUseCase usecases.SearchUseCase,
	webhookUseCase usecases.WebhookUseCase,
	authService auth.AuthService,
) *gin.Engine {
	// Set Gin to release mode in production
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a new Gin router
	router := gin.New()

	// Apply global middleware
	router.Use(gin.Recovery())                             // Recover from panics
	router.Use(middleware.Logger(cfg.LogLevel))            // Request logging
	router.Use(middleware.CORS(cfg.CORSAllowOrigins))      // CORS handling
	router.Use(middleware.RateLimiter(cfg.GlobalRateLimit)) // Global rate limiting

	// Create handler instances
	healthHandler := handlers.NewHealthHandler(cfg)
	documentHandler := handlers.NewDocumentHandler(documentUseCase)
	folderHandler := handlers.NewFolderHandler(folderUseCase)
	searchHandler := handlers.NewSearchHandler(searchUseCase)
	webhookHandler := handlers.NewWebhookHandler(webhookUseCase)

	// Set up health check endpoints (no auth required)
	setupHealthRoutes(router, healthHandler)

	// Create API v1 route group with authentication middleware
	api := router.Group(apiVersionPrefix)
	api.Use(middleware.Authentication(authService)) // JWT validation

	// Set up resource-specific routes
	setupDocumentRoutes(api, documentHandler, cfg)
	setupFolderRoutes(api, folderHandler, documentHandler, cfg)
	setupSearchRoutes(api, searchHandler, cfg)
	setupWebhookRoutes(api, webhookHandler, cfg)

	return router
}

// setupHealthRoutes sets up health check endpoints for the API
func setupHealthRoutes(router *gin.Engine, healthHandler *handlers.HealthHandler) {
	health := router.Group("/health")
	// Simple liveness check to indicate the service is running
	health.GET("/liveness", healthHandler.Liveness)
	// Readiness check to verify the service can handle requests
	health.GET("/readiness", healthHandler.Readiness)
	// Deep health check to verify connections to dependencies
	health.GET("/deep", healthHandler.DeepCheck)
}

// setupDocumentRoutes sets up document-related API routes
func setupDocumentRoutes(api *gin.RouterGroup, documentHandler *handlers.DocumentHandler, cfg config.Config) {
	// Document routes with authentication
	documents := api.Group("/documents")
	
	// Apply upload rate limiting middleware to upload endpoint
	uploadLimiter := middleware.UploadRateLimiter(cfg.UploadRateLimit)
	
	// Document operations
	// Upload a new document
	documents.POST("", uploadLimiter, middleware.Authorization("contributor"), documentHandler.UploadDocument)
	// Get document metadata
	documents.GET("/:id", middleware.Authorization("reader"), documentHandler.GetDocument)
	// Download document content
	documents.GET("/:id/content", middleware.Authorization("reader"), documentHandler.DownloadDocument)
	// Get a presigned URL for document download
	documents.GET("/:id/content/url", middleware.Authorization("reader"), documentHandler.GetDocumentURL)
	// Download multiple documents as a zip archive
	documents.POST("/batch/download", middleware.Authorization("reader"), documentHandler.BatchDownload)
	// Get a presigned URL for batch document download
	documents.POST("/batch/download/url", middleware.Authorization("reader"), documentHandler.GetBatchDownloadURL)
	// Check the processing status of a document
	documents.GET("/:id/status", middleware.Authorization("reader"), documentHandler.GetDocumentStatus)
	// Get a document thumbnail
	documents.GET("/:id/thumbnail", middleware.Authorization("reader"), documentHandler.GetDocumentThumbnail)
	// Get a presigned URL for document thumbnail
	documents.GET("/:id/thumbnail/url", middleware.Authorization("reader"), documentHandler.GetDocumentThumbnailURL)
	// Update document metadata
	documents.PUT("/:id", middleware.Authorization("contributor"), documentHandler.UpdateDocument)
	// Delete a document
	documents.DELETE("/:id", middleware.Authorization("editor"), documentHandler.DeleteDocument)
}

// setupFolderRoutes sets up folder-related API routes
func setupFolderRoutes(api *gin.RouterGroup, folderHandler *handlers.FolderHandler, documentHandler *handlers.DocumentHandler, cfg config.Config) {
	// Folder routes with authentication
	folders := api.Group("/folders")
	
	// Folder operations
	// Create a new folder
	folders.POST("", middleware.Authorization("contributor"), folderHandler.CreateFolder)
	// Get folder details
	folders.GET("/:id", middleware.Authorization("reader"), folderHandler.GetFolder)
	// Update folder metadata
	folders.PUT("/:id", middleware.Authorization("contributor"), folderHandler.UpdateFolder)
	// Delete a folder
	folders.DELETE("/:id", middleware.Authorization("editor"), folderHandler.DeleteFolder)
	// List top-level folders or folders within a parent folder
	folders.GET("", middleware.Authorization("reader"), folderHandler.ListFolders)
	// Move a folder to a different parent
	folders.PUT("/:id/move", middleware.Authorization("contributor"), folderHandler.MoveFolder)
	// Search for folders by name or metadata
	folders.GET("/search", middleware.Authorization("reader"), folderHandler.SearchFolders)
	// Get a folder by its path
	folders.GET("/path", middleware.Authorization("reader"), folderHandler.GetFolderByPath)
	// List documents within a folder
	folders.GET("/:id/documents", middleware.Authorization("reader"), documentHandler.ListDocumentsInFolder)
}

// setupSearchRoutes sets up search-related API routes
func setupSearchRoutes(api *gin.RouterGroup, searchHandler *handlers.SearchHandler, cfg config.Config) {
	// Search routes with authentication and search rate limiting
	search := api.Group("/search")
	search.Use(middleware.SearchRateLimiter(cfg.SearchRateLimit))
	
	// Search operations
	// Search documents by content
	search.POST("/content", middleware.Authorization("reader"), searchHandler.SearchContent)
	// Search documents by metadata
	search.POST("/metadata", middleware.Authorization("reader"), searchHandler.SearchMetadata)
	// Combined search (content + metadata)
	search.POST("", middleware.Authorization("reader"), searchHandler.Search)
	// Search within a specific folder
	search.POST("/folder", middleware.Authorization("reader"), searchHandler.SearchInFolder)
}

// setupWebhookRoutes sets up webhook-related API routes
func setupWebhookRoutes(api *gin.RouterGroup, webhookHandler *handlers.WebhookHandler, cfg config.Config) {
	// Webhook routes with authentication
	webhooks := api.Group("/webhooks")
	
	// Webhook operations
	// Register a new webhook
	webhooks.POST("", middleware.Authorization("administrator"), webhookHandler.CreateWebhook)
	// List all webhooks for the tenant
	webhooks.GET("", middleware.Authorization("reader"), webhookHandler.ListWebhooks)
	// Get webhook details
	webhooks.GET("/:id", middleware.Authorization("reader"), webhookHandler.GetWebhook)
	// Update webhook configuration
	webhooks.PUT("/:id", middleware.Authorization("administrator"), webhookHandler.UpdateWebhook)
	// Delete a webhook
	webhooks.DELETE("/:id", middleware.Authorization("administrator"), webhookHandler.DeleteWebhook)
	// Get all supported event types
	webhooks.GET("/event-types", middleware.Authorization("reader"), webhookHandler.GetEventTypes)
	// List delivery attempts for a webhook
	webhooks.GET("/:id/deliveries", middleware.Authorization("reader"), webhookHandler.ListWebhookDeliveries)
	// Get details of a specific delivery attempt
	webhooks.GET("/deliveries/:id", middleware.Authorization("reader"), webhookHandler.GetDeliveryStatus)
	// Retry a failed webhook delivery
	webhooks.POST("/deliveries/:id/retry", middleware.Authorization("administrator"), webhookHandler.RetryDelivery)
}