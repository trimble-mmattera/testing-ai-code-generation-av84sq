// Package services provides domain service implementations for the Document Management Platform.
package services

import (
	"context"
	"fmt"
	"strings"

	"../models"
	"../repositories"
	"../../pkg/errors"
	"../../pkg/logger"
	"../../pkg/utils"
)

// Error constants for folder-related operations
var (
	ErrFolderNotFound           = errors.NewResourceNotFoundError("folder not found")
	ErrFolderAlreadyExists      = errors.NewValidationError("folder with this name already exists in the parent folder")
	ErrParentFolderNotFound     = errors.NewResourceNotFoundError("parent folder not found")
	ErrCannotDeleteNonEmptyFolder = errors.NewValidationError("cannot delete non-empty folder")
	ErrInvalidFolderName        = errors.NewValidationError("invalid folder name")
	ErrPermissionDenied         = errors.NewPermissionDeniedError("permission denied for folder operation")
)

// Event type constants for folder operations
const (
	FolderEventCreated = "folder.created"
	FolderEventUpdated = "folder.updated"
	FolderEventDeleted = "folder.deleted"
	FolderEventMoved   = "folder.moved"
)

// FolderService defines the interface for folder management operations
type FolderService interface {
	// CreateFolder creates a new folder with proper tenant isolation and permission checks
	CreateFolder(ctx context.Context, name, parentID, tenantID, userID string) (string, error)
	
	// GetFolder retrieves a folder by its ID with tenant isolation and permission checks
	GetFolder(ctx context.Context, id, tenantID, userID string) (*models.Folder, error)
	
	// UpdateFolder updates a folder's metadata with tenant isolation and permission checks
	UpdateFolder(ctx context.Context, id, name, tenantID, userID string) error
	
	// DeleteFolder deletes a folder with tenant isolation and permission checks
	DeleteFolder(ctx context.Context, id, tenantID, userID string) error
	
	// ListFolderContents lists the contents of a folder with pagination, tenant isolation, and permission checks
	ListFolderContents(ctx context.Context, id, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], utils.PaginatedResult[models.Document], error)
	
	// ListRootFolders lists root folders for a tenant with pagination and permission checks
	ListRootFolders(ctx context.Context, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error)
	
	// MoveFolder moves a folder to a new parent with tenant isolation and permission checks
	MoveFolder(ctx context.Context, id, newParentID, tenantID, userID string) error
	
	// SearchFolders searches folders by name with tenant isolation and permission checks
	SearchFolders(ctx context.Context, query, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error)
	
	// GetFolderByPath retrieves a folder by its path with tenant isolation and permission checks
	GetFolderByPath(ctx context.Context, path, tenantID, userID string) (*models.Folder, error)
	
	// CreateFolderPermission creates a permission for a folder with tenant isolation and permission checks
	CreateFolderPermission(ctx context.Context, folderID, roleID, permissionType, tenantID, userID string) (string, error)
	
	// DeleteFolderPermission deletes a permission for a folder with tenant isolation and permission checks
	DeleteFolderPermission(ctx context.Context, permissionID, tenantID, userID string) error
	
	// GetFolderPermissions retrieves permissions for a folder with tenant isolation and permission checks
	GetFolderPermissions(ctx context.Context, folderID, tenantID, userID string) ([]*models.Permission, error)
}

// folderService implements the FolderService interface
type folderService struct {
	folderRepo      repositories.FolderRepository
	documentRepo    repositories.DocumentRepository
	permissionRepo  repositories.PermissionRepository
	authService     AuthService
	eventService    EventServiceInterface
	logger          *logger.Logger
}

// NewFolderService creates a new FolderService instance
func NewFolderService(
	folderRepo repositories.FolderRepository,
	documentRepo repositories.DocumentRepository,
	permissionRepo repositories.PermissionRepository,
	authService AuthService,
	eventService EventServiceInterface,
) FolderService {
	// Validate required dependencies
	if folderRepo == nil {
		panic("folderRepo cannot be nil")
	}
	if documentRepo == nil {
		panic("documentRepo cannot be nil")
	}
	if permissionRepo == nil {
		panic("permissionRepo cannot be nil")
	}
	if authService == nil {
		panic("authService cannot be nil")
	}
	if eventService == nil {
		panic("eventService cannot be nil")
	}
	
	return &folderService{
		folderRepo:      folderRepo,
		documentRepo:    documentRepo,
		permissionRepo:  permissionRepo,
		authService:     authService,
		eventService:    eventService,
		logger:          logger.WithField("service", "folder_service"),
	}
}

// CreateFolder creates a new folder with proper tenant isolation and permission checks
func (s *folderService) CreateFolder(ctx context.Context, name, parentID, tenantID, userID string) (string, error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if err := s.validateFolderName(name); err != nil {
		log.Error("Invalid folder name", "name", name)
		return "", err
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return "", errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return "", errors.NewValidationError("user ID is required")
	}
	
	// Verify user has permission to create folders
	hasPermission, err := s.authService.VerifyPermission(ctx, userID, tenantID, PermissionManageFolders)
	if err != nil {
		log.WithError(err).Error("Failed to verify user permission")
		return "", errors.Wrap(err, "failed to verify user permission")
	}
	
	if !hasPermission {
		log.Error("User does not have permission to create folders", "userID", userID, "tenantID", tenantID)
		return "", ErrPermissionDenied
	}
	
	// Check parent folder if specified
	var parentFolder *models.Folder
	var parentPath string
	
	if parentID != "" {
		parentFolder, err = s.folderRepo.GetByID(ctx, parentID, tenantID)
		if err != nil {
			log.WithError(err).Error("Failed to get parent folder", "parentID", parentID)
			return "", errors.Wrap(err, "failed to get parent folder")
		}
		
		if parentFolder == nil {
			log.Error("Parent folder not found", "parentID", parentID)
			return "", ErrParentFolderNotFound
		}
		
		// Verify user has write permission for the parent folder
		hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, parentID, PermissionWrite)
		if err != nil {
			log.WithError(err).Error("Failed to verify folder access", "folderID", parentID)
			return "", errors.Wrap(err, "failed to verify folder access")
		}
		
		if !hasAccess {
			log.Error("User does not have write permission for parent folder", "userID", userID, "folderID", parentID)
			return "", ErrPermissionDenied
		}
		
		parentPath = parentFolder.Path
	}
	
	// Check if folder with the same name already exists in the parent folder
	exists, err := s.checkFolderExists(ctx, name, parentID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to check if folder exists", "name", name, "parentID", parentID)
		return "", errors.Wrap(err, "failed to check if folder exists")
	}
	
	if exists {
		log.Error("Folder with this name already exists in the parent folder", "name", name, "parentID", parentID)
		return "", ErrFolderAlreadyExists
	}
	
	// Create the folder
	folder := models.NewFolder(name, parentID, tenantID, userID)
	
	// Set folder path
	if parentFolder != nil {
		folder.SetPath(folder.BuildPath(parentPath))
	} else {
		folder.SetPath(folder.BuildPath(""))
	}
	
	// Save folder to repository
	folderID, err := s.folderRepo.Create(ctx, folder)
	if err != nil {
		log.WithError(err).Error("Failed to create folder", "name", name)
		return "", errors.Wrap(err, "failed to create folder")
	}
	
	// Create default permissions for the folder
	ownerPermission := models.NewPermission(
		"owner", // This should be a role ID for the owner
		models.ResourceTypeFolder,
		folderID,
		models.PermissionTypeAdmin,
		tenantID,
		userID,
	)
	
	_, err = s.permissionRepo.Create(ctx, ownerPermission)
	if err != nil {
		log.WithError(err).Error("Failed to create folder permission", "folderID", folderID)
		// We don't return error here as the folder was already created
	}
	
	// If parent folder exists, propagate permissions from parent
	if parentFolder != nil {
		err = s.permissionRepo.PropagatePermissions(ctx, folderID, tenantID)
		if err != nil {
			log.WithError(err).Error("Failed to propagate permissions", "folderID", folderID)
			// We don't return error here as the folder was already created
		}
	}
	
	// Publish folder created event
	additionalData := map[string]interface{}{
		"name":      name,
		"parentID":  parentID,
		"path":      folder.Path,
		"createdBy": userID,
	}
	
	_, err = s.eventService.CreateAndPublishFolderEvent(ctx, FolderEventCreated, tenantID, folderID, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish folder created event", "folderID", folderID)
		// We don't return error here as the folder was already created
	}
	
	log.Info("Folder created successfully", "folderID", folderID, "name", name, "parentID", parentID)
	return folderID, nil
}

// GetFolder retrieves a folder by its ID with tenant isolation and permission checks
func (s *folderService) GetFolder(ctx context.Context, id, tenantID, userID string) (*models.Folder, error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(id) == "" {
		log.Error("Folder ID cannot be empty")
		return nil, errors.NewValidationError("folder ID is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return nil, errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return nil, errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", id)
		return nil, errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", id)
		return nil, ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", id, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return nil, ErrFolderNotFound
	}
	
	// Verify user has read permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, id, PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", id)
		return nil, errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have read permission for folder", "userID", userID, "folderID", id)
		return nil, ErrPermissionDenied
	}
	
	log.Info("Retrieved folder successfully", "folderID", id)
	return folder, nil
}

// UpdateFolder updates a folder's metadata with tenant isolation and permission checks
func (s *folderService) UpdateFolder(ctx context.Context, id, name, tenantID, userID string) error {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(id) == "" {
		log.Error("Folder ID cannot be empty")
		return errors.NewValidationError("folder ID is required")
	}
	
	if err := s.validateFolderName(name); err != nil {
		log.Error("Invalid folder name", "name", name)
		return err
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", id)
		return errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", id)
		return ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", id, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return ErrFolderNotFound
	}
	
	// Verify user has write permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, id, PermissionWrite)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", id)
		return errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have write permission for folder", "userID", userID, "folderID", id)
		return ErrPermissionDenied
	}
	
	// Check if name is changing and if a folder with the same name already exists in the parent folder
	if name != folder.Name {
		exists, err := s.checkFolderExists(ctx, name, folder.ParentID, tenantID)
		if err != nil {
			log.WithError(err).Error("Failed to check if folder exists", "name", name, "parentID", folder.ParentID)
			return errors.Wrap(err, "failed to check if folder exists")
		}
		
		if exists {
			log.Error("Folder with this name already exists in the parent folder", "name", name, "parentID", folder.ParentID)
			return ErrFolderAlreadyExists
		}
	}
	
	// Update folder
	folder.Update(name)
	
	// Save changes to repository
	err = s.folderRepo.Update(ctx, folder)
	if err != nil {
		log.WithError(err).Error("Failed to update folder", "folderID", id)
		return errors.Wrap(err, "failed to update folder")
	}
	
	// Publish folder updated event
	additionalData := map[string]interface{}{
		"name":      name,
		"updatedBy": userID,
	}
	
	_, err = s.eventService.CreateAndPublishFolderEvent(ctx, FolderEventUpdated, tenantID, id, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish folder updated event", "folderID", id)
		// We don't return error here as the folder was already updated
	}
	
	log.Info("Folder updated successfully", "folderID", id, "name", name)
	return nil
}

// DeleteFolder deletes a folder with tenant isolation and permission checks
func (s *folderService) DeleteFolder(ctx context.Context, id, tenantID, userID string) error {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(id) == "" {
		log.Error("Folder ID cannot be empty")
		return errors.NewValidationError("folder ID is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", id)
		return errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", id)
		return ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", id, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return ErrFolderNotFound
	}
	
	// Verify user has delete permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, id, PermissionDelete)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", id)
		return errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have delete permission for folder", "userID", userID, "folderID", id)
		return ErrPermissionDenied
	}
	
	// Check if folder has child folders
	childFolders, err := s.folderRepo.GetChildren(ctx, id, tenantID, &utils.Pagination{Page: 1, PageSize: 1})
	if err != nil {
		log.WithError(err).Error("Failed to check for child folders", "folderID", id)
		return errors.Wrap(err, "failed to check for child folders")
	}
	
	if len(childFolders.Items) > 0 {
		log.Error("Cannot delete folder with child folders", "folderID", id)
		return ErrCannotDeleteNonEmptyFolder
	}
	
	// Check if folder has documents
	documents, err := s.documentRepo.ListByFolder(ctx, id, tenantID, &utils.Pagination{Page: 1, PageSize: 1})
	if err != nil {
		log.WithError(err).Error("Failed to check for documents in folder", "folderID", id)
		return errors.Wrap(err, "failed to check for documents in folder")
	}
	
	if len(documents.Items) > 0 {
		log.Error("Cannot delete folder with documents", "folderID", id)
		return ErrCannotDeleteNonEmptyFolder
	}
	
	// Delete folder from repository
	err = s.folderRepo.Delete(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to delete folder", "folderID", id)
		return errors.Wrap(err, "failed to delete folder")
	}
	
	// Delete folder permissions
	err = s.permissionRepo.DeleteByResourceID(ctx, models.ResourceTypeFolder, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to delete folder permissions", "folderID", id)
		// We don't return error here as the folder was already deleted
	}
	
	// Publish folder deleted event
	additionalData := map[string]interface{}{
		"name":      folder.Name,
		"parentID":  folder.ParentID,
		"deletedBy": userID,
	}
	
	_, err = s.eventService.CreateAndPublishFolderEvent(ctx, FolderEventDeleted, tenantID, id, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish folder deleted event", "folderID", id)
		// We don't return error here as the folder was already deleted
	}
	
	log.Info("Folder deleted successfully", "folderID", id)
	return nil
}

// ListFolderContents lists the contents of a folder with pagination, tenant isolation, and permission checks
func (s *folderService) ListFolderContents(ctx context.Context, id, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], utils.PaginatedResult[models.Document], error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(id) == "" {
		log.Error("Folder ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.NewValidationError("folder ID is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.NewValidationError("user ID is required")
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", id, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, ErrFolderNotFound
	}
	
	// Verify user has read permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, id, PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have read permission for folder", "userID", userID, "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, ErrPermissionDenied
	}
	
	// Get child folders
	childFolders, err := s.folderRepo.GetChildren(ctx, id, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("Failed to get child folders", "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to get child folders")
	}
	
	// Get documents in folder
	documents, err := s.documentRepo.ListByFolder(ctx, id, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("Failed to get documents in folder", "folderID", id)
		return utils.PaginatedResult[models.Folder]{}, utils.PaginatedResult[models.Document]{}, errors.Wrap(err, "failed to get documents in folder")
	}
	
	// Filter items based on user permissions
	// This would be a more comprehensive implementation in a real-world scenario
	
	log.Info("Folder contents listed successfully", "folderID", id, "childFolders", len(childFolders.Items), "documents", len(documents.Items))
	return childFolders, documents, nil
}

// ListRootFolders lists root folders for a tenant with pagination and permission checks
func (s *folderService) ListRootFolders(ctx context.Context, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("user ID is required")
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Verify user belongs to tenant
	hasAccess, err := s.authService.VerifyTenantAccess(ctx, userID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to verify tenant access", "userID", userID, "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, errors.Wrap(err, "failed to verify tenant access")
	}
	
	if !hasAccess {
		log.Error("User does not belong to tenant", "userID", userID, "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, ErrPermissionDenied
	}
	
	// Get root folders
	rootFolders, err := s.folderRepo.GetRootFolders(ctx, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("Failed to get root folders", "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, errors.Wrap(err, "failed to get root folders")
	}
	
	// Filter folders based on user permissions
	// This would be a more comprehensive implementation in a real-world scenario
	
	log.Info("Root folders listed successfully", "tenantID", tenantID, "count", len(rootFolders.Items))
	return rootFolders, nil
}

// MoveFolder moves a folder to a new parent with tenant isolation and permission checks
func (s *folderService) MoveFolder(ctx context.Context, id, newParentID, tenantID, userID string) error {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(id) == "" {
		log.Error("Folder ID cannot be empty")
		return errors.NewValidationError("folder ID is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", id)
		return errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", id)
		return ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", id, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return ErrFolderNotFound
	}
	
	// Verify user has write permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, id, PermissionWrite)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", id)
		return errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have write permission for folder", "userID", userID, "folderID", id)
		return ErrPermissionDenied
	}
	
	// If new parent is provided, verify it exists and belongs to the tenant
	if newParentID != "" {
		newParentFolder, err := s.folderRepo.GetByID(ctx, newParentID, tenantID)
		if err != nil {
			log.WithError(err).Error("Failed to get new parent folder", "newParentID", newParentID)
			return errors.Wrap(err, "failed to get new parent folder")
		}
		
		if newParentFolder == nil {
			log.Error("New parent folder not found", "newParentID", newParentID)
			return ErrParentFolderNotFound
		}
		
		// Verify user has write permission for the new parent folder
		hasAccess, err = s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, newParentID, PermissionWrite)
		if err != nil {
			log.WithError(err).Error("Failed to verify folder access", "folderID", newParentID)
			return errors.Wrap(err, "failed to verify folder access")
		}
		
		if !hasAccess {
			log.Error("User does not have write permission for new parent folder", "userID", userID, "folderID", newParentID)
			return ErrPermissionDenied
		}
		
		// Check if a folder with the same name already exists in the new parent folder
		exists, err := s.checkFolderExists(ctx, folder.Name, newParentID, tenantID)
		if err != nil {
			log.WithError(err).Error("Failed to check if folder exists", "name", folder.Name, "parentID", newParentID)
			return errors.Wrap(err, "failed to check if folder exists")
		}
		
		if exists {
			log.Error("Folder with this name already exists in the new parent folder", "name", folder.Name, "parentID", newParentID)
			return ErrFolderAlreadyExists
		}
		
		// Check for circular reference (cannot move a folder to its own descendant)
		isCircular, err := s.checkCircularReference(ctx, id, newParentID, tenantID)
		if err != nil {
			log.WithError(err).Error("Failed to check for circular reference", "folderID", id, "newParentID", newParentID)
			return errors.Wrap(err, "failed to check for circular reference")
		}
		
		if isCircular {
			log.Error("Cannot move folder to its own descendant", "folderID", id, "newParentID", newParentID)
			return errors.NewValidationError("cannot move folder to its own descendant")
		}
	}
	
	// Move folder
	err = s.folderRepo.Move(ctx, id, newParentID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to move folder", "folderID", id, "newParentID", newParentID)
		return errors.Wrap(err, "failed to move folder")
	}
	
	// Publish folder moved event
	additionalData := map[string]interface{}{
		"name":        folder.Name,
		"oldParentID": folder.ParentID,
		"newParentID": newParentID,
		"movedBy":     userID,
	}
	
	_, err = s.eventService.CreateAndPublishFolderEvent(ctx, FolderEventMoved, tenantID, id, additionalData)
	if err != nil {
		log.WithError(err).Error("Failed to publish folder moved event", "folderID", id)
		// We don't return error here as the folder was already moved
	}
	
	log.Info("Folder moved successfully", "folderID", id, "oldParentID", folder.ParentID, "newParentID", newParentID)
	return nil
}

// SearchFolders searches folders by name with tenant isolation and permission checks
func (s *folderService) SearchFolders(ctx context.Context, query, tenantID, userID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Folder], error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(query) == "" {
		log.Error("Search query cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("search query is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return utils.PaginatedResult[models.Folder]{}, errors.NewValidationError("user ID is required")
	}
	
	// Set default pagination if not provided
	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}
	
	// Verify user belongs to tenant
	hasAccess, err := s.authService.VerifyTenantAccess(ctx, userID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to verify tenant access", "userID", userID, "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, errors.Wrap(err, "failed to verify tenant access")
	}
	
	if !hasAccess {
		log.Error("User does not belong to tenant", "userID", userID, "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, ErrPermissionDenied
	}
	
	// Search folders
	folders, err := s.folderRepo.Search(ctx, query, tenantID, pagination)
	if err != nil {
		log.WithError(err).Error("Failed to search folders", "query", query, "tenantID", tenantID)
		return utils.PaginatedResult[models.Folder]{}, errors.Wrap(err, "failed to search folders")
	}
	
	// Filter folders based on user permissions
	// This would be a more comprehensive implementation in a real-world scenario
	
	log.Info("Folders searched successfully", "query", query, "tenantID", tenantID, "count", len(folders.Items))
	return folders, nil
}

// GetFolderByPath retrieves a folder by its path with tenant isolation and permission checks
func (s *folderService) GetFolderByPath(ctx context.Context, path, tenantID, userID string) (*models.Folder, error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(path) == "" {
		log.Error("Folder path cannot be empty")
		return nil, errors.NewValidationError("folder path is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return nil, errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return nil, errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByPath(ctx, path, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder by path", "path", path)
		return nil, errors.Wrap(err, "failed to get folder by path")
	}
	
	if folder == nil {
		log.Error("Folder not found", "path", path)
		return nil, ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "path", path, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return nil, ErrFolderNotFound
	}
	
	// Verify user has read permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, folder.ID, PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", folder.ID)
		return nil, errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have read permission for folder", "userID", userID, "folderID", folder.ID)
		return nil, ErrPermissionDenied
	}
	
	log.Info("Retrieved folder by path successfully", "path", path, "folderID", folder.ID)
	return folder, nil
}

// CreateFolderPermission creates a permission for a folder with tenant isolation and permission checks
func (s *folderService) CreateFolderPermission(ctx context.Context, folderID, roleID, permissionType, tenantID, userID string) (string, error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(folderID) == "" {
		log.Error("Folder ID cannot be empty")
		return "", errors.NewValidationError("folder ID is required")
	}
	
	if strings.TrimSpace(roleID) == "" {
		log.Error("Role ID cannot be empty")
		return "", errors.NewValidationError("role ID is required")
	}
	
	if strings.TrimSpace(permissionType) == "" {
		log.Error("Permission type cannot be empty")
		return "", errors.NewValidationError("permission type is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return "", errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return "", errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, folderID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", folderID)
		return "", errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", folderID)
		return "", ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", folderID, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return "", ErrFolderNotFound
	}
	
	// Verify user has admin permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, folderID, models.PermissionTypeAdmin)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", folderID)
		return "", errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have admin permission for folder", "userID", userID, "folderID", folderID)
		return "", ErrPermissionDenied
	}
	
	// Create permission
	permission := models.NewPermission(roleID, models.ResourceTypeFolder, folderID, permissionType, tenantID, userID)
	
	// Save permission to repository
	permissionID, err := s.permissionRepo.Create(ctx, permission)
	if err != nil {
		log.WithError(err).Error("Failed to create folder permission", "folderID", folderID, "roleID", roleID)
		return "", errors.Wrap(err, "failed to create folder permission")
	}
	
	// Propagate permission to subfolders if needed
	err = s.permissionRepo.PropagatePermissions(ctx, folderID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to propagate permissions", "folderID", folderID)
		// We don't return error here as the permission was already created
	}
	
	log.Info("Folder permission created successfully", "folderID", folderID, "roleID", roleID, "permissionType", permissionType)
	return permissionID, nil
}

// DeleteFolderPermission deletes a permission for a folder with tenant isolation and permission checks
func (s *folderService) DeleteFolderPermission(ctx context.Context, permissionID, tenantID, userID string) error {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(permissionID) == "" {
		log.Error("Permission ID cannot be empty")
		return errors.NewValidationError("permission ID is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return errors.NewValidationError("user ID is required")
	}
	
	// Get permission from repository
	permission, err := s.permissionRepo.GetByID(ctx, permissionID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get permission", "permissionID", permissionID)
		return errors.Wrap(err, "failed to get permission")
	}
	
	if permission == nil {
		log.Error("Permission not found", "permissionID", permissionID)
		return errors.NewResourceNotFoundError("permission not found")
	}
	
	// Verify tenant isolation
	if permission.TenantID != tenantID {
		log.Error("Permission tenant mismatch", "permissionID", permissionID, "permissionTenantID", permission.TenantID, "requestTenantID", tenantID)
		return errors.NewResourceNotFoundError("permission not found")
	}
	
	// Verify permission is for a folder
	if permission.ResourceType != models.ResourceTypeFolder {
		log.Error("Permission is not for a folder", "permissionID", permissionID, "resourceType", permission.ResourceType)
		return errors.NewValidationError("permission is not for a folder")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, permission.ResourceID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", permission.ResourceID)
		return errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", permission.ResourceID)
		return ErrFolderNotFound
	}
	
	// Verify user has admin permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, folder.ID, models.PermissionTypeAdmin)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", folder.ID)
		return errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have admin permission for folder", "userID", userID, "folderID", folder.ID)
		return ErrPermissionDenied
	}
	
	// Delete permission from repository
	err = s.permissionRepo.Delete(ctx, permissionID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to delete folder permission", "permissionID", permissionID)
		return errors.Wrap(err, "failed to delete folder permission")
	}
	
	log.Info("Folder permission deleted successfully", "permissionID", permissionID, "folderID", folder.ID)
	return nil
}

// GetFolderPermissions retrieves permissions for a folder with tenant isolation and permission checks
func (s *folderService) GetFolderPermissions(ctx context.Context, folderID, tenantID, userID string) ([]*models.Permission, error) {
	log := logger.WithContext(ctx)
	
	// Validate input
	if strings.TrimSpace(folderID) == "" {
		log.Error("Folder ID cannot be empty")
		return nil, errors.NewValidationError("folder ID is required")
	}
	
	if strings.TrimSpace(tenantID) == "" {
		log.Error("Tenant ID cannot be empty")
		return nil, errors.NewValidationError("tenant ID is required")
	}
	
	if strings.TrimSpace(userID) == "" {
		log.Error("User ID cannot be empty")
		return nil, errors.NewValidationError("user ID is required")
	}
	
	// Get folder from repository
	folder, err := s.folderRepo.GetByID(ctx, folderID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder", "folderID", folderID)
		return nil, errors.Wrap(err, "failed to get folder")
	}
	
	if folder == nil {
		log.Error("Folder not found", "folderID", folderID)
		return nil, ErrFolderNotFound
	}
	
	// Verify tenant isolation
	if folder.TenantID != tenantID {
		log.Error("Folder tenant mismatch", "folderID", folderID, "folderTenantID", folder.TenantID, "requestTenantID", tenantID)
		return nil, ErrFolderNotFound
	}
	
	// Verify user has read permission for the folder
	hasAccess, err := s.authService.VerifyResourceAccess(ctx, userID, tenantID, ResourceTypeFolder, folderID, PermissionRead)
	if err != nil {
		log.WithError(err).Error("Failed to verify folder access", "folderID", folderID)
		return nil, errors.Wrap(err, "failed to verify folder access")
	}
	
	if !hasAccess {
		log.Error("User does not have read permission for folder", "userID", userID, "folderID", folderID)
		return nil, ErrPermissionDenied
	}
	
	// Get direct permissions for the folder
	permissions, err := s.permissionRepo.GetByResourceID(ctx, models.ResourceTypeFolder, folderID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get folder permissions", "folderID", folderID)
		return nil, errors.Wrap(err, "failed to get folder permissions")
	}
	
	// Get inherited permissions for the folder
	inheritedPermissions, err := s.permissionRepo.GetInheritedPermissions(ctx, folderID, tenantID)
	if err != nil {
		log.WithError(err).Error("Failed to get inherited folder permissions", "folderID", folderID)
		// We don't return error here as we already have direct permissions
	}
	
	// Combine direct and inherited permissions
	allPermissions := permissions
	if inheritedPermissions != nil {
		allPermissions = append(allPermissions, inheritedPermissions...)
	}
	
	log.Info("Folder permissions retrieved successfully", "folderID", folderID, "count", len(allPermissions))
	return allPermissions, nil
}

// checkFolderExists checks if a folder with the given name exists in the parent folder
func (s *folderService) checkFolderExists(ctx context.Context, name, parentID, tenantID string) (bool, error) {
	// If parentID is empty, check root folders
	if parentID == "" {
		rootFolders, err := s.folderRepo.GetRootFolders(ctx, tenantID, &utils.Pagination{Page: 1, PageSize: 100})
		if err != nil {
			return false, err
		}
		
		for _, folder := range rootFolders.Items {
			if folder.Name == name {
				return true, nil
			}
		}
		
		return false, nil
	}
	
	// Check child folders of parent
	childFolders, err := s.folderRepo.GetChildren(ctx, parentID, tenantID, &utils.Pagination{Page: 1, PageSize: 100})
	if err != nil {
		return false, err
	}
	
	for _, folder := range childFolders.Items {
		if folder.Name == name {
			return true, nil
		}
	}
	
	return false, nil
}

// checkCircularReference checks if moving a folder would create a circular reference
func (s *folderService) checkCircularReference(ctx context.Context, folderID, newParentID, tenantID string) (bool, error) {
	// If folder ID is the same as new parent ID, it's a circular reference
	if folderID == newParentID {
		return true, nil
	}
	
	// Get the folder path
	folderPath, err := s.folderRepo.GetFolderPath(ctx, folderID, tenantID)
	if err != nil {
		return false, err
	}
	
	// Get the new parent folder path
	newParentPath, err := s.folderRepo.GetFolderPath(ctx, newParentID, tenantID)
	if err != nil {
		return false, err
	}
	
	// Check if the new parent is a descendant of the folder
	// Ensure paths are properly formatted for comparison
	if !strings.HasSuffix(folderPath, "/") {
		folderPath = folderPath + "/"
	}
	
	return strings.HasPrefix(newParentPath, folderPath), nil
}

// validateFolderName validates a folder name according to system rules
func (s *folderService) validateFolderName(name string) error {
	// Check if name is empty
	if strings.TrimSpace(name) == "" {
		return ErrInvalidFolderName
	}
	
	// Check for invalid characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return errors.NewValidationError(fmt.Sprintf("folder name contains invalid character: %s", char))
		}
	}
	
	// Check name length (max 255 characters)
	if len(name) > 255 {
		return errors.NewValidationError("folder name is too long (max 255 characters)")
	}
	
	return nil
}