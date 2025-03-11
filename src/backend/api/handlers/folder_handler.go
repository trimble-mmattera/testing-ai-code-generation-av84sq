// Package handlers implements HTTP handlers for folder management operations in the Document Management Platform.
package handlers

import (
	"net/http" // standard library - For HTTP status codes
	"strconv" // standard library - For string conversions

	"github.com/gin-gonic/gin" // v1.9.0+ - Web framework for handling HTTP requests

	"../../application/usecases" // Import folder use cases for business logic
	"../dto"                      // Import folder DTOs for request/response handling
	"../middleware"               // Import middleware utilities for extracting user and tenant context
	"../validators"               // Import folder validators for request validation
	"../../pkg/errors"             // Import error utilities for error handling
	"../../pkg/logger"             // Import logging utilities for request logging
	"../../pkg/utils/pagination"   // Import pagination utilities for paginated responses
	"github.com/aws/aws-sdk-go-v2/aws"
	responsedto "src/backend/api/dto"
	errordto "src/backend/api/dto"
)

// FolderHandler handles HTTP requests for folder management operations
type FolderHandler struct {
	folderUseCase *usecases.FolderUseCase
}

// NewFolderHandler creates a new FolderHandler with the provided folder use case
func NewFolderHandler(folderUseCase *usecases.FolderUseCase) *FolderHandler {
	// Validate that folderUseCase is not nil
	if folderUseCase == nil {
		panic("folderUseCase cannot be nil")
	}

	// Create and return a new FolderHandler with the provided folderUseCase
	return &FolderHandler{
		folderUseCase: folderUseCase,
	}
}

// CreateFolder handles requests to create a new folder
func (h *FolderHandler) CreateFolder(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Log folder creation attempt
	log.Info("Attempting to create folder", "userID", userID, "tenantID", tenantID)

	// Bind the request body to a FolderCreateRequest struct
	var request dto.FolderCreateRequest
	if err := c.BindJSON(&request); err != nil {
		// If binding fails, return a bad request error
		log.WithError(err).Error("Invalid request body")
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Invalid request body"),
			nil,
		))
		return
	}

	// Validate the request using ValidateCreateFolderRequest
	if err := validators.ValidateCreateFolderRequest(&request); err != nil {
		// If validation fails, return a validation error response
		log.WithError(err).Error("Validation failed")
		validationErrors := errors.GetValidationErrors(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Validation failed"),
			validationErrors,
		))
		return
	}

	// Convert the request to a domain model using FolderCreateRequestToModel
	folder := dto.FolderCreateRequestToModel(request, tenantID, userID)

	// Call folderUseCase.CreateFolder with the appropriate parameters
	folderID, err := h.folderUseCase.CreateFolder(c.Request.Context(), folder.Name, folder.ParentID, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// If successful, get the created folder using folderUseCase.GetFolder
	createdFolder, err := h.folderUseCase.GetFolder(c.Request.Context(), folderID, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Convert the folder to a DTO and return a success response
	folderDTO := dto.FolderToDTO(createdFolder)
	c.JSON(http.StatusCreated, responsedto.NewDataResponse(folderDTO))

	// Log successful folder creation
	log.Info("Folder created successfully", "folderID", folderID)
}

// GetFolder handles requests to retrieve a folder by ID
func (h *FolderHandler) GetFolder(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Extract folder ID from the URL path parameter
	id := c.Param("id")

	// Log folder retrieval attempt
	log.Info("Attempting to retrieve folder", "folderID", id, "userID", userID, "tenantID", tenantID)

	// Call folderUseCase.GetFolder with the appropriate parameters
	folder, err := h.folderUseCase.GetFolder(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Convert the folder to a DTO and return a success response
	folderDTO := dto.FolderToDTO(folder)
	c.JSON(http.StatusOK, responsedto.NewDataResponse(folderDTO))

	// Log successful folder retrieval
	log.Info("Folder retrieved successfully", "folderID", id)
}

// UpdateFolder handles requests to update a folder
func (h *FolderHandler) UpdateFolder(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Extract folder ID from the URL path parameter
	id := c.Param("id")

	// Log folder update attempt
	log.Info("Attempting to update folder", "folderID", id, "userID", userID, "tenantID", tenantID)

	// Bind the request body to a FolderUpdateRequest struct
	var request dto.FolderUpdateRequest
	if err := c.BindJSON(&request); err != nil {
		// If binding fails, return a bad request error
		log.WithError(err).Error("Invalid request body")
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Invalid request body"),
			nil,
		))
		return
	}

	// Validate the request using ValidateUpdateFolderRequest
	if err := validators.ValidateUpdateFolderRequest(&request); err != nil {
		// If validation fails, return a validation error response
		log.WithError(err).Error("Validation failed")
		validationErrors := errors.GetValidationErrors(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Validation failed"),
			validationErrors,
		))
		return
	}

	// Call folderUseCase.UpdateFolder with the appropriate parameters
	err := h.folderUseCase.UpdateFolder(c.Request.Context(), id, request.Name, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// If successful, get the updated folder using folderUseCase.GetFolder
	updatedFolder, err := h.folderUseCase.GetFolder(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Convert the folder to a DTO and return a success response
	folderDTO := dto.FolderToDTO(updatedFolder)
	c.JSON(http.StatusOK, responsedto.NewDataResponse(folderDTO))

	// Log successful folder update
	log.Info("Folder updated successfully", "folderID", id)
}

// DeleteFolder handles requests to delete a folder
func (h *FolderHandler) DeleteFolder(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Extract folder ID from the URL path parameter
	id := c.Param("id")

	// Log folder deletion attempt
	log.Info("Attempting to delete folder", "folderID", id, "userID", userID, "tenantID", tenantID)

	// Call folderUseCase.DeleteFolder with the appropriate parameters
	err := h.folderUseCase.DeleteFolder(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Return a success message response
	c.JSON(http.StatusOK, responsedto.NewMessageResponse("Folder deleted successfully"))

	// Log successful folder deletion
	log.Info("Folder deleted successfully", "folderID", id)
}

// ListFolders handles requests to list folders with pagination
func (h *FolderHandler) ListFolders(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Bind query parameters to a FolderListRequest struct
	var request dto.FolderListRequest
	if err := c.BindQuery(&request); err != nil {
		// If binding fails, return a bad request error
		log.WithError(err).Error("Invalid query parameters")
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Invalid query parameters"),
			nil,
		))
		return
	}

	// Validate the request using ValidateFolderListRequest
	if err := validators.ValidateFolderListRequest(&request); err != nil {
		// If validation fails, return a validation error response
		log.WithError(err).Error("Validation failed")
		validationErrors := errors.GetValidationErrors(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Validation failed"),
			validationErrors,
		))
		return
	}

	// Create pagination parameters from the request
	paginationParams := pagination.NewPagination(request.Page, request.PageSize)

	// Log folder listing attempt
	log.Info("Attempting to list folders", "userID", userID, "tenantID", tenantID, "parentID", request.ParentID, "page", request.Page, "pageSize", request.PageSize)

	// If parentID is provided, call folderUseCase.ListFolderContents
	var paginatedResponse dto.PaginatedFolderResponse
	if request.ParentID != "" {
		// Call folderUseCase.ListFolderContents
		folders, _, err := h.folderUseCase.ListFolderContents(c.Request.Context(), request.ParentID, tenantID, userID, paginationParams)
		if err != nil {
			// If an error occurs, handle it based on error type and return appropriate error response
			h.handleError(c, err)
			return
		}

		// Create a paginated response with the folder results
		paginatedResponse = dto.CreatePaginatedFolderResponse(folders)
	} else {
		// If parentID is not provided, call folderUseCase.ListRootFolders
		folders, err := h.folderUseCase.ListRootFolders(c.Request.Context(), tenantID, userID, paginationParams)
		if err != nil {
			// If an error occurs, handle it based on error type and return appropriate error response
			h.handleError(c, err)
			return
		}

		// Create a paginated response with the folder results
		paginatedResponse = dto.CreatePaginatedFolderResponse(folders)
	}

	// Return the paginated response
	c.JSON(http.StatusOK, responsedto.NewPaginatedResponse(paginatedResponse.Folders, paginatedResponse.Pagination))

	// Log successful folder listing
	log.Info("Folders listed successfully", "userID", userID, "tenantID", tenantID, "parentID", request.ParentID, "count", paginatedResponse.Pagination.TotalItems)
}

// MoveFolder handles requests to move a folder to a new parent
func (h *FolderHandler) MoveFolder(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Extract folder ID from the URL path parameter
	id := c.Param("id")

	// Log folder move attempt
	log.Info("Attempting to move folder", "folderID", id, "userID", userID, "tenantID", tenantID)

	// Bind the request body to a FolderMoveRequest struct
	var request dto.FolderMoveRequest
	if err := c.BindJSON(&request); err != nil {
		// If binding fails, return a bad request error
		log.WithError(err).Error("Invalid request body")
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Invalid request body"),
			nil,
		))
		return
	}

	// Validate the request using ValidateMoveFolderRequest
	if err := validators.ValidateMoveFolderRequest(&request); err != nil {
		// If validation fails, return a validation error response
		log.WithError(err).Error("Validation failed")
		validationErrors := errors.GetValidationErrors(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Validation failed"),
			validationErrors,
		))
		return
	}

	// Call folderUseCase.MoveFolder with the appropriate parameters
	err := h.folderUseCase.MoveFolder(c.Request.Context(), id, request.NewParentID, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// If successful, get the moved folder using folderUseCase.GetFolder
	movedFolder, err := h.folderUseCase.GetFolder(c.Request.Context(), id, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Convert the folder to a DTO and return a success response
	folderDTO := dto.FolderToDTO(movedFolder)
	c.JSON(http.StatusOK, responsedto.NewDataResponse(folderDTO))

	// Log successful folder move
	log.Info("Folder moved successfully", "folderID", id)
}

// SearchFolders handles requests to search folders by name
func (h *FolderHandler) SearchFolders(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Bind query parameters to a FolderSearchRequest struct
	var request dto.FolderSearchRequest
	if err := c.BindQuery(&request); err != nil {
		// If binding fails, return a bad request error
		log.WithError(err).Error("Invalid query parameters")
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Invalid query parameters"),
			nil,
		))
		return
	}

	// Validate the request using ValidateFolderSearchRequest
	if err := validators.ValidateFolderSearchRequest(&request); err != nil {
		// If validation fails, return a validation error response
		log.WithError(err).Error("Validation failed")
		validationErrors := errors.GetValidationErrors(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Validation failed"),
			validationErrors,
		))
		return
	}

	// Create pagination parameters from the request
	paginationParams := pagination.NewPagination(request.Page, request.PageSize)

	// Log folder search attempt
	log.Info("Attempting to search folders", "userID", userID, "tenantID", tenantID, "query", request.Query, "page", request.Page, "pageSize", request.PageSize)

	// Call folderUseCase.SearchFolders with the appropriate parameters
	folders, err := h.folderUseCase.SearchFolders(c.Request.Context(), request.Query, tenantID, userID, paginationParams)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Create a paginated response with the search results
	paginatedResponse := dto.CreatePaginatedFolderResponse(folders)

	// Return the paginated response
	c.JSON(http.StatusOK, responsedto.NewPaginatedResponse(paginatedResponse.Folders, paginatedResponse.Pagination))

	// Log successful folder search
	log.Info("Folders searched successfully", "userID", userID, "tenantID", tenantID, "query", request.Query, "count", paginatedResponse.Pagination.TotalItems)
}

// GetFolderByPath handles requests to retrieve a folder by its path
func (h *FolderHandler) GetFolderByPath(c *gin.Context) {
	// Extract user ID and tenant ID from the request context
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	// Get logger with context
	log := logger.WithContext(c.Request.Context())

	// Extract folder path from the query parameter
	path := c.Query("path")

	// Log folder retrieval attempt
	log.Info("Attempting to retrieve folder by path", "path", path, "userID", userID, "tenantID", tenantID)

	// Call folderUseCase.GetFolderByPath with the appropriate parameters
	folder, err := h.folderUseCase.GetFolderByPath(c.Request.Context(), path, tenantID, userID)
	if err != nil {
		// If an error occurs, handle it based on error type and return appropriate error response
		h.handleError(c, err)
		return
	}

	// Convert the folder to a DTO and return a success response
	folderDTO := dto.FolderToDTO(folder)
	c.JSON(http.StatusOK, responsedto.NewDataResponse(folderDTO))

	// Log successful folder retrieval
	log.Info("Folder retrieved successfully", "path", path, "folderID", folder.ID)
}

// handleError handles errors and returns appropriate HTTP responses
func (h *FolderHandler) handleError(c *gin.Context, err error) {
	// Log the error with context
	log := logger.WithContext(c.Request.Context()).WithError(err)
	log.Error("Handling error")

	// Check error type using error package utilities
	if errors.IsValidationError(err) {
		// If validation error, return validation error response with validation errors
		validationErrors := errors.GetValidationErrors(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, errordto.NewValidationErrorResponse(
			errors.NewValidationError("Validation failed"),
			validationErrors,
		))
		return
	}

	if errors.IsResourceNotFoundError(err) {
		// If not found error, return resource not found error response
		c.AbortWithStatusJSON(http.StatusNotFound, errordto.NewResourceNotFoundErrorResponse(err))
		return
	}

	if errors.IsAuthorizationError(err) {
		// If authorization error, return authorization error response
		c.AbortWithStatusJSON(http.StatusForbidden, errordto.NewAuthorizationErrorResponse(err))
		return
	}

	// Otherwise, return internal server error response
	c.AbortWithStatusJSON(http.StatusInternalServerError, errordto.NewInternalErrorResponse(err))
}
func (h *FolderHandler) GetFolderByPath(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	log := logger.WithContext(c.Request.Context())

	path := c.Query("path")

	log.Info("Attempting to retrieve folder by path", "path", path, "userID", userID, "tenantID", tenantID)

	folder, err := h.folderUseCase.GetFolderByPath(c.Request.Context(), path, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	folderDTO := dto.FolderToDTO(folder)
	c.JSON(http.StatusOK, responsedto.NewDataResponse(folderDTO))

	log.Info("Folder retrieved successfully", "path", path, "folderID", folder.ID)
}