// Package handlers implements HTTP handlers for document-related operations in the Document Management Platform.
// This file contains the DocumentHandler struct and its methods for handling document upload, download, search, listing, and management API endpoints.
// It follows the Clean Architecture principles by translating HTTP requests to use case calls and formatting responses according to API standards.
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin" // v1.9.0+

	"../../application/usecases"
	"../dto/document_dto"
	errdto "../dto/error_dto"
	"../dto/response_dto"
	"../middleware"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/validator"
	"../../pkg/utils/pagination"
)

// DocumentHandler handles HTTP requests for document-related operations
type DocumentHandler struct {
	documentUseCase usecases.DocumentUseCase
	logger          *logger.Logger
}

// NewDocumentHandler creates a new DocumentHandler with the provided document use case
func NewDocumentHandler(documentUseCase usecases.DocumentUseCase) (*DocumentHandler, error) {
	// Validate that documentUseCase is not nil
	if documentUseCase == nil {
		return nil, fmt.Errorf("documentUseCase cannot be nil")
	}

	// Create and return a new DocumentHandler with the provided documentUseCase
	return &DocumentHandler{
		documentUseCase: documentUseCase,
		logger:          logger.WithField("handler", "document"),
	}, nil
}

// RegisterRoutes registers document-related routes with the provided router group
func (h *DocumentHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Register POST /documents for document upload
	router.POST("/documents", h.UploadDocument)

	// Register GET /documents/:id for getting document metadata
	router.GET("/documents/:id", h.GetDocument)

	// Register GET /documents/:id/content for document download
	router.GET("/documents/:id/content", h.DownloadDocument)

	// Register GET /documents/:id/content/url for getting document download URL
	router.GET("/documents/:id/content/url", h.GetDocumentDownloadURL)

	// Register POST /documents/batch/download for batch document download
	router.POST("/documents/batch/download", h.BatchDownloadDocuments)

	// Register POST /documents/batch/download/url for getting batch download URL
	router.POST("/documents/batch/download/url", h.GetBatchDownloadURL)

	// Register GET /documents/:id/status for checking document status
	router.GET("/documents/:id/status", h.GetDocumentStatus)

	// Register GET /documents/:id/thumbnail for getting document thumbnail
	router.GET("/documents/:id/thumbnail", h.GetDocumentThumbnail)

	// Register GET /documents/:id/thumbnail/url for getting thumbnail URL
	router.GET("/documents/:id/thumbnail/url", h.GetDocumentThumbnailURL)

	// Register PUT /documents/:id for updating document metadata
	router.PUT("/documents/:id", h.UpdateDocument)

	// Register DELETE /documents/:id for deleting a document
	router.DELETE("/documents/:id", h.DeleteDocument)

	// Register GET /folders/:id/documents for listing documents in a folder
	router.GET("/folders/:id/documents", h.ListDocumentsByFolder)

	// Register POST /search/documents for searching documents
	router.POST("/search/documents", h.SearchDocuments)
}

// UploadDocument handles document upload requests
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Parse multipart form data
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		log.WithError(err).Error("Failed to parse multipart form data")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid form data: " + err.Error())))
		return
	}
	defer file.Close()

	// Bind request to CreateDocumentRequest struct
	var req document_dto.CreateDocumentRequest
	if err := c.ShouldBind(&req); err != nil {
		log.WithError(err).Error("Failed to bind request to CreateDocumentRequest struct")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid request payload: " + err.Error())))
		return
	}
	req.File = header

	// Validate the request
	if err := validator.Validate(req); err != nil {
		log.WithError(err).Error("Invalid request")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(err))
		return
	}

	// Open the uploaded file
	src, err := header.Open()
	if err != nil {
		log.WithError(err).Error("Failed to open uploaded file")
		c.AbortWithStatusJSON(http.StatusInternalServerError, errdto.NewErrorResponse(errors.NewInternalError("failed to open uploaded file: " + err.Error())))
		return
	}
	defer src.Close()

	// Call documentUseCase.UploadDocument with the request data
	documentID, err := h.documentUseCase.UploadDocument(c.Request.Context(), req.Name, header.Header.Get("Content-Type"), header.Size, req.FolderID, tenantID, userID, src, req.Metadata)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return 202 Accepted with document ID and status
	c.JSON(http.StatusAccepted, response_dto.NewDataResponse(document_dto.DocumentUploadResponse{
		DocumentID: documentID,
		Status:     "processing",
	}))
}

// GetDocument handles requests to get document metadata
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Call documentUseCase.GetDocument with the document ID
	document, err := h.documentUseCase.GetDocument(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Convert the document model to DTO
	documentDTO := document_dto.DocumentToDTO(*document)

	// Log successful document retrieval
	log.Info("Document retrieved successfully", "documentID", id)

	// Return 200 OK with document metadata
	c.JSON(http.StatusOK, response_dto.NewDataResponse(documentDTO))
}

// DownloadDocument handles document download requests
func (h *DocumentHandler) DownloadDocument(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Call documentUseCase.DownloadDocument with the document ID
	contentStream, fileName, err := h.documentUseCase.DownloadDocument(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer contentStream.Close()

	// Set appropriate content headers
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")

	// Stream the document content to the response
	_, err = io.Copy(c.Writer, contentStream)
	if err != nil {
		log.WithError(err).Error("Failed to stream document content to response")
		c.AbortWithStatusJSON(http.StatusInternalServerError, errdto.NewErrorResponse(errors.NewInternalError("failed to stream document content: " + err.Error())))
		return
	}
}

// GetDocumentDownloadURL handles requests to get a presigned URL for document download
func (h *DocumentHandler) GetDocumentDownloadURL(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Parse expiration time from query parameters
	expirationStr := c.DefaultQuery("expires_in", "3600") // Default to 1 hour
	expirationSeconds, err := strconv.Atoi(expirationStr)
	if err != nil {
		log.WithError(err).Error("Invalid expiration time in query parameters")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid expiration time: " + err.Error())))
		return
	}

	// Call documentUseCase.GetDocumentPresignedURL with the document ID
	downloadURL, err := h.documentUseCase.GetDocumentPresignedURL(c.Request.Context(), id, tenantID, userID, expirationSeconds)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return 200 OK with download URL and expiration
	c.JSON(http.StatusOK, response_dto.NewDataResponse(document_dto.DocumentDownloadResponse{
		DocumentID:  id,
		DownloadURL: downloadURL,
		ExpiresIn:   expirationSeconds,
	}))
}

// BatchDownloadDocuments handles batch document download requests
func (h *DocumentHandler) BatchDownloadDocuments(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Bind request to BatchDownloadRequest struct
	var req document_dto.BatchDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Error("Failed to bind request to BatchDownloadRequest struct")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid request payload: " + err.Error())))
		return
	}

	// Validate the request
	if err := validator.Validate(req); err != nil {
		log.WithError(err).Error("Invalid request")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(err))
		return
	}

	// Call documentUseCase.BatchDownloadDocuments with the document IDs
	contentStream, err := h.documentUseCase.BatchDownloadDocuments(c.Request.Context(), req.DocumentIDs, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer contentStream.Close()

	// Set appropriate content headers
	c.Header("Content-Disposition", "attachment; filename=documents.zip")
	c.Header("Content-Type", "application/zip")

	// Stream the archive content to the response
	_, err = io.Copy(c.Writer, contentStream)
	if err != nil {
		log.WithError(err).Error("Failed to stream archive content to response")
		c.AbortWithStatusJSON(http.StatusInternalServerError, errdto.NewErrorResponse(errors.NewInternalError("failed to stream archive content: " + err.Error())))
		return
	}
}

// GetBatchDownloadURL handles requests to get a presigned URL for batch document download
func (h *DocumentHandler) GetBatchDownloadURL(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Bind request to BatchDownloadRequest struct
	var req document_dto.BatchDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Error("Failed to bind request to BatchDownloadRequest struct")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid request payload: " + err.Error())))
		return
	}

	// Validate the request
	if err := validator.Validate(req); err != nil {
		log.WithError(err).Error("Invalid request")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(err))
		return
	}

	// Parse expiration time from query parameters
	expirationStr := c.DefaultQuery("expires_in", "3600") // Default to 1 hour
	expirationSeconds, err := strconv.Atoi(expirationStr)
	if err != nil {
		log.WithError(err).Error("Invalid expiration time in query parameters")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid expiration time: " + err.Error())))
		return
	}

	// Call documentUseCase.GetBatchDownloadPresignedURL with the document IDs
	downloadURL, err := h.documentUseCase.GetBatchDownloadPresignedURL(c.Request.Context(), req.DocumentIDs, tenantID, userID, expirationSeconds)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return 200 OK with download URL and expiration
	c.JSON(http.StatusOK, response_dto.NewDataResponse(document_dto.BatchDownloadResponse{
		ArchiveName:   "documents.zip", // TODO: Make this configurable
		DocumentCount: len(req.DocumentIDs),
		DownloadURL:   downloadURL,
		ExpiresIn:   expirationSeconds,
	}))
}

// GetDocumentStatus handles requests to check document processing status
func (h *DocumentHandler) GetDocumentStatus(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Call documentUseCase.GetDocumentStatus with the document ID
	status, err := h.documentUseCase.GetDocumentStatus(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return 200 OK with document status information
	c.JSON(http.StatusOK, response_dto.NewDataResponse(document_dto.DocumentStatusResponse{
		DocumentID: id,
		Status:     status,
	}))
}

// GetDocumentThumbnail handles requests to get document thumbnail
func (h *DocumentHandler) GetDocumentThumbnail(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Call documentUseCase.GetDocumentThumbnail with the document ID
	contentStream, err := h.documentUseCase.GetDocumentThumbnail(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer contentStream.Close()

	// Set appropriate content headers
	c.Header("Content-Type", "image/png") // Assuming thumbnails are always PNG

	// Stream the thumbnail content to the response
	_, err = io.Copy(c.Writer, contentStream)
	if err != nil {
		log.WithError(err).Error("Failed to stream thumbnail content to response")
		c.AbortWithStatusJSON(http.StatusInternalServerError, errdto.NewErrorResponse(errors.NewInternalError("failed to stream thumbnail content: " + err.Error())))
		return
	}
}

// GetDocumentThumbnailURL handles requests to get a presigned URL for document thumbnail
func (h *DocumentHandler) GetDocumentThumbnailURL(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Parse expiration time from query parameters
	expirationStr := c.DefaultQuery("expires_in", "3600") // Default to 1 hour
	expirationSeconds, err := strconv.Atoi(expirationStr)
	if err != nil {
		log.WithError(err).Error("Invalid expiration time in query parameters")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid expiration time: " + err.Error())))
		return
	}

	// Call documentUseCase.GetDocumentThumbnailURL with the document ID
	thumbnailURL, err := h.documentUseCase.GetDocumentThumbnailURL(c.Request.Context(), id, tenantID, userID, expirationSeconds)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return 200 OK with thumbnail URL and expiration
	c.JSON(http.StatusOK, response_dto.NewDataResponse(document_dto.DocumentDownloadResponse{
		DocumentID:  id,
		DownloadURL: thumbnailURL,
		ExpiresIn:   expirationSeconds,
	}))
}

// UpdateDocument handles requests to update document metadata
func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Bind request to UpdateDocumentRequest struct
	var req document_dto.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.WithError(err).Error("Failed to bind request to UpdateDocumentRequest struct")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(errors.NewValidationError("invalid request payload: " + err.Error())))
		return
	}

	// Validate the request
	if err := validator.Validate(req); err != nil {
		log.WithError(err).Error("Invalid request")
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(err))
		return
	}

	// Call documentUseCase.GetDocument to retrieve the document
	document, err := h.documentUseCase.GetDocument(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Update the document with the request data
	// Call documentUseCase to save the updated document
	// Return 200 OK with updated document metadata
	fmt.Println("Implement UpdateDocument")
}

// DeleteDocument handles requests to delete a document
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	// Extract document ID from the URL path
	id := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Call documentUseCase.DeleteDocument with the document ID
	// Return 200 OK with success message
	fmt.Println("Implement DeleteDocument")
}

// ListDocumentsByFolder handles requests to list documents in a folder
func (h *DocumentHandler) ListDocumentsByFolder(c *gin.Context) {
	// Extract folder ID from the URL path
	folderID := c.Param("id")

	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Parse pagination parameters from query string
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	paginationParams := pagination.ParsePaginationFromStrings(pageStr, pageSizeStr)

	// Call documentUseCase.ListDocumentsByFolder with the folder ID
	// Convert document models to DTOs
	// Return 200 OK with paginated document list
	fmt.Println("Implement ListDocumentsByFolder")
}

// SearchDocuments handles document search requests
func (h *DocumentHandler) SearchDocuments(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := h.logger.WithContext(c.Request.Context())

	// Parse search query and metadata from request body
	// Parse pagination parameters from query string
	// Call documentUseCase.CombinedSearch with the search criteria
	// Convert document models to DTOs
	// Return 200 OK with paginated search results
	fmt.Println("Implement SearchDocuments")
}

// handleError handles errors and returns appropriate HTTP responses
func (h *DocumentHandler) handleError(c *gin.Context, err error) {
	// Log the error with context
	h.logger.WithError(err).Error("Handling error")

	// Check error type using errors package functions
	// Return appropriate error response based on error type
	switch {
	case errors.IsValidationError(err):
		// For validation errors, return 400 Bad Request
		c.AbortWithStatusJSON(http.StatusBadRequest, errdto.NewErrorResponse(err))
	case errors.IsResourceNotFoundError(err):
		// For resource not found errors, return 404 Not Found
		c.AbortWithStatusJSON(http.StatusNotFound, errdto.NewErrorResponse(err))
	case errors.IsAuthorizationError(err):
		// For authorization errors, return 403 Forbidden
		c.AbortWithStatusJSON(http.StatusForbidden, errdto.NewErrorResponse(err))
	default:
		// For other errors, return 500 Internal Server Error
		c.AbortWithStatusJSON(http.StatusInternalServerError, errdto.NewErrorResponse(errors.NewInternalErrorResponse(err)))
	}
}