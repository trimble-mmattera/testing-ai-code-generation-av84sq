// Package usecases implements application use cases for the Document Management Platform.
package usecases

import (
	"context"

	"../../domain/services"
	"../../domain/models"
	"../../pkg/utils"
	"../../pkg/errors"
	"../../pkg/logger"
)

// FolderUseCase implements use cases for folder management operations
type FolderUseCase struct {
	folderService services.FolderService
	eventService  services.EventServiceInterface
}

// NewFolderUseCase creates a new FolderUseCase instance with the provided dependencies
func NewFolderUseCase(
	folderService services.FolderService,
	eventService services.EventServiceInterface,
) *FolderUseCase {
	// Validate that folderService is not nil
	if folderService == nil {
		panic("folderService cannot be nil")
	}
	
	// Validate that eventService is not nil
	if eventService == nil {
		panic("eventService cannot be nil")
	}
	
	return &FolderUseCase{
		folderService: folderService,
		eventService:  eventService,
	}
}

// CreateFolder creates a new folder with proper tenant isolation and permission checks
func (uc *FolderUseCase) CreateFolder(ctx context.Context, name, parentID, tenantID, userID string) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder creation attempt with provided parameters
	log.Info("Creating folder", 
		"name", name, 
		"parentID", parentID, 
		"tenantID", tenantID,
		"userID", userID)
	
	// Call folderService.CreateFolder with the provided parameters
	folderID, err := uc.folderService.CreateFolder(ctx, name, parentID, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to create folder")
		return "", errors.Wrap(err, "failed to create folder")
	}
	
	// If successful, log folder creation success with folder ID
	log.Info("Folder created successfully", "folderID", folderID)
	
	return folderID, nil
}

// GetFolder retrieves a folder by its ID with tenant isolation and permission checks
func (uc *FolderUseCase) GetFolder(ctx context.Context, id, tenantID, userID string) (*models.Folder, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder retrieval attempt with folder ID
	log.Info("Getting folder", "folderID", id, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.GetFolder with the provided parameters
	folder, err := uc.folderService.GetFolder(ctx, id, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to get folder", "folderID", id)
		return nil, errors.Wrap(err, "failed to get folder")
	}
	
	// If successful, log folder retrieval success
	log.Info("Folder retrieved successfully", "folderID", id)
	
	return folder, nil
}

// UpdateFolder updates a folder's metadata with tenant isolation and permission checks
func (uc *FolderUseCase) UpdateFolder(ctx context.Context, id, name, tenantID, userID string) error {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder update attempt with folder ID and new name
	log.Info("Updating folder", "folderID", id, "name", name, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.UpdateFolder with the provided parameters
	err := uc.folderService.UpdateFolder(ctx, id, name, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to update folder", "folderID", id)
		return errors.Wrap(err, "failed to update folder")
	}
	
	// If successful, log folder update success
	log.Info("Folder updated successfully", "folderID", id)
	
	return nil
}

// DeleteFolder deletes a folder with tenant isolation and permission checks
func (uc *FolderUseCase) DeleteFolder(ctx context.Context, id, tenantID, userID string) error {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder deletion attempt with folder ID
	log.Info("Deleting folder", "folderID", id, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.DeleteFolder with the provided parameters
	err := uc.folderService.DeleteFolder(ctx, id, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to delete folder", "folderID", id)
		return errors.Wrap(err, "failed to delete folder")
	}
	
	// If successful, log folder deletion success
	log.Info("Folder deleted successfully", "folderID", id)
	
	return nil
}

// ListFolderContents lists the contents of a folder with pagination, tenant isolation, and permission checks
func (uc *FolderUseCase) ListFolderContents(ctx context.Context, id, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], utils.PaginatedResult[models.Document], error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder contents listing attempt with folder ID
	log.Info("Listing folder contents", "folderID", id, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.ListFolderContents with the provided parameters
	folders, documents, err := uc.folderService.ListFolderContents(ctx, id, tenantID, userID, pagination)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to list folder contents", "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to list folder contents")
	}
	
	// If successful, log folder contents listing success with counts
	log.Info("Folder contents listed successfully", 
		"folderID", id, 
		"folderCount", len(folders.Items), 
		"documentCount", len(documents.Items))
	
	return folders, documents, nil
}

// ListRootFolders lists root folders for a tenant with pagination and permission checks
func (uc *FolderUseCase) ListRootFolders(ctx context.Context, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log root folders listing attempt
	log.Info("Listing root folders", "tenantID", tenantID, "userID", userID)
	
	// Call folderService.ListRootFolders with the provided parameters
	folders, err := uc.folderService.ListRootFolders(ctx, tenantID, userID, pagination)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to list root folders", "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, errors.Wrap(err, "failed to list root folders")
	}
	
	// If successful, log root folders listing success with count
	log.Info("Root folders listed successfully", "tenantID", tenantID, "count", len(folders.Items))
	
	return folders, nil
}

// MoveFolder moves a folder to a new parent with tenant isolation and permission checks
func (uc *FolderUseCase) MoveFolder(ctx context.Context, id, newParentID, tenantID, userID string) error {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder move attempt with folder ID and new parent ID
	log.Info("Moving folder", "folderID", id, "newParentID", newParentID, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.MoveFolder with the provided parameters
	err := uc.folderService.MoveFolder(ctx, id, newParentID, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to move folder", "folderID", id, "newParentID", newParentID)
		return errors.Wrap(err, "failed to move folder")
	}
	
	// If successful, log folder move success
	log.Info("Folder moved successfully", "folderID", id, "newParentID", newParentID)
	
	return nil
}

// SearchFolders searches folders by name with tenant isolation and permission checks
func (uc *FolderUseCase) SearchFolders(ctx context.Context, query, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder search attempt with query
	log.Info("Searching folders", "query", query, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.SearchFolders with the provided parameters
	folders, err := uc.folderService.SearchFolders(ctx, query, tenantID, userID, pagination)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to search folders", "query", query)
		return utils.PaginatedResult[models.Folder]{}, errors.Wrap(err, "failed to search folders")
	}
	
	// If successful, log folder search success with result count
	log.Info("Folders searched successfully", "query", query, "count", len(folders.Items))
	
	return folders, nil
}

// GetFolderByPath retrieves a folder by its path with tenant isolation and permission checks
func (uc *FolderUseCase) GetFolderByPath(ctx context.Context, path, tenantID, userID string) (*models.Folder, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder retrieval by path attempt
	log.Info("Getting folder by path", "path", path, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.GetFolderByPath with the provided parameters
	folder, err := uc.folderService.GetFolderByPath(ctx, path, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to get folder by path", "path", path)
		return nil, errors.Wrap(err, "failed to get folder by path")
	}
	
	// If successful, log folder retrieval success
	log.Info("Folder retrieved by path successfully", "path", path, "folderID", folder.ID)
	
	return folder, nil
}

// CreateFolderPermission creates a permission for a folder with tenant isolation and permission checks
func (uc *FolderUseCase) CreateFolderPermission(ctx context.Context, folderID, roleID, permissionType, tenantID, userID string) (string, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder permission creation attempt
	log.Info("Creating folder permission", 
		"folderID", folderID, 
		"roleID", roleID, 
		"permissionType", permissionType, 
		"tenantID", tenantID, 
		"userID", userID)
	
	// Call folderService.CreateFolderPermission with the provided parameters
	permissionID, err := uc.folderService.CreateFolderPermission(ctx, folderID, roleID, permissionType, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to create folder permission", "folderID", folderID)
		return "", errors.Wrap(err, "failed to create folder permission")
	}
	
	// If successful, log permission creation success with permission ID
	log.Info("Folder permission created successfully", "permissionID", permissionID, "folderID", folderID)
	
	return permissionID, nil
}

// DeleteFolderPermission deletes a permission for a folder with tenant isolation and permission checks
func (uc *FolderUseCase) DeleteFolderPermission(ctx context.Context, permissionID, tenantID, userID string) error {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder permission deletion attempt
	log.Info("Deleting folder permission", "permissionID", permissionID, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.DeleteFolderPermission with the provided parameters
	err := uc.folderService.DeleteFolderPermission(ctx, permissionID, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to delete folder permission", "permissionID", permissionID)
		return errors.Wrap(err, "failed to delete folder permission")
	}
	
	// If successful, log permission deletion success
	log.Info("Folder permission deleted successfully", "permissionID", permissionID)
	
	return nil
}

// GetFolderPermissions retrieves permissions for a folder with tenant isolation and permission checks
func (uc *FolderUseCase) GetFolderPermissions(ctx context.Context, folderID, tenantID, userID string) ([]*models.Permission, error) {
	// Get logger with context
	log := logger.WithContext(ctx)
	
	// Log folder permissions retrieval attempt
	log.Info("Getting folder permissions", "folderID", folderID, "tenantID", tenantID, "userID", userID)
	
	// Call folderService.GetFolderPermissions with the provided parameters
	permissions, err := uc.folderService.GetFolderPermissions(ctx, folderID, tenantID, userID)
	if err != nil {
		// If error occurs, log error and wrap it with context
		log.WithError(err).Error("Failed to get folder permissions", "folderID", folderID)
		return nil, errors.Wrap(err, "failed to get folder permissions")
	}
	
	// If successful, log permissions retrieval success with count
	log.Info("Folder permissions retrieved successfully", "folderID", folderID, "count", len(permissions))
	
	return permissions, nil
}