// Package dto defines Data Transfer Objects (DTOs) for folder-related API operations
// in the Document Management Platform.
package dto

import (
	"time" // standard library - For handling time-related fields in DTOs

	"../../domain/models"
	"../../pkg/utils/pagination"
	"../../pkg/utils/time_utils"
)

// FolderDTO represents a folder in API responses
type FolderDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parentId,omitempty"`
	Path      string `json:"path"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// FolderCreateRequest represents the payload for folder creation
type FolderCreateRequest struct {
	Name     string `json:"name" binding:"required"`
	ParentID string `json:"parentId,omitempty"`
}

// FolderUpdateRequest represents the payload for folder update
type FolderUpdateRequest struct {
	Name string `json:"name" binding:"required"`
}

// FolderMoveRequest represents the payload for moving a folder to a new parent
type FolderMoveRequest struct {
	NewParentID string `json:"newParentId" binding:"required"`
}

// FolderListRequest represents the parameters for folder listing
type FolderListRequest struct {
	ParentID  string `form:"parentId" json:"parentId"`
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	SortBy    string `form:"sortBy" json:"sortBy"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
}

// FolderSearchRequest represents the parameters for folder search
type FolderSearchRequest struct {
	Query    string `form:"query" json:"query" binding:"required"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

// FolderCreateResponse represents the response for folder creation
type FolderCreateResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parentId,omitempty"`
	Path      string `json:"path"`
	CreatedAt string `json:"createdAt"`
}

// PaginatedFolderResponse represents a paginated response for folder listings
type PaginatedFolderResponse struct {
	Folders    []FolderDTO        `json:"folders"`
	Pagination pagination.PageInfo `json:"pagination"`
}

// FolderToDTO converts a domain Folder model to a FolderDTO
func FolderToDTO(folder *models.Folder) FolderDTO {
	return FolderDTO{
		ID:        folder.ID,
		Name:      folder.Name,
		ParentID:  folder.ParentID,
		Path:      folder.Path,
		CreatedAt: timeutils.FormatTime(folder.CreatedAt, ""),
		UpdatedAt: timeutils.FormatTime(folder.UpdatedAt, ""),
	}
}

// FolderCreateRequestToModel converts a FolderCreateRequest to a domain Folder model
func FolderCreateRequestToModel(request FolderCreateRequest, tenantID, ownerID string) *models.Folder {
	return models.NewFolder(request.Name, request.ParentID, tenantID, ownerID)
}

// CreatePaginatedFolderResponse creates a paginated response for folder listings
func CreatePaginatedFolderResponse(result pagination.PaginatedResult[models.Folder]) PaginatedFolderResponse {
	folders := make([]FolderDTO, len(result.Items))
	for i, folder := range result.Items {
		folders[i] = FolderToDTO(&folder)
	}

	return PaginatedFolderResponse{
		Folders:    folders,
		Pagination: result.Pagination,
	}
}