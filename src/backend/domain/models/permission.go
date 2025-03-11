// Package models contains the domain models for the document management platform.
package models

import (
	"errors"
	"time"
)

// Resource types
const (
	ResourceTypeDocument = "document"
	ResourceTypeFolder   = "folder"
)

// Permission types
const (
	PermissionTypeRead   = "read"
	PermissionTypeWrite  = "write"
	PermissionTypeDelete = "delete"
	PermissionTypeAdmin  = "admin"
)

// Error definitions
var (
	ErrResourceTypeEmpty     = errors.New("resource type cannot be empty")
	ErrResourceIDEmpty       = errors.New("resource ID cannot be empty")
	ErrRoleIDEmpty           = errors.New("role ID cannot be empty")
	ErrTenantIDEmpty         = errors.New("tenant ID cannot be empty")
	ErrPermissionTypeEmpty   = errors.New("permission type cannot be empty")
	ErrInvalidResourceType   = errors.New("invalid resource type")
	ErrInvalidPermissionType = errors.New("invalid permission type")
)

// Permission represents a permission in the system that grants a role specific access to a resource.
// Permissions are used to implement the role-based access control system and support tenant isolation.
type Permission struct {
	ID             string    // Unique identifier for the permission
	TenantID       string    // ID of the tenant this permission belongs to for isolation
	RoleID         string    // ID of the role this permission is assigned to
	ResourceType   string    // Type of resource (document or folder)
	ResourceID     string    // ID of the resource this permission applies to
	PermissionType string    // Type of permission (read, write, delete, admin)
	Inherited      bool      // Whether this permission is inherited from a parent resource
	CreatedBy      string    // ID of the user who created this permission
	CreatedAt      time.Time // When this permission was created
	UpdatedAt      time.Time // When this permission was last updated
}

// NewPermission creates a new Permission instance with the given parameters.
func NewPermission(roleID, resourceType, resourceID, permissionType, tenantID, createdBy string) *Permission {
	now := time.Now()
	return &Permission{
		RoleID:         roleID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		PermissionType: permissionType,
		TenantID:       tenantID,
		CreatedBy:      createdBy,
		Inherited:      false, // Not inherited by default
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsValidResourceType validates if a given resource type is one of the predefined valid types.
func IsValidResourceType(resourceType string) bool {
	return resourceType == ResourceTypeDocument || resourceType == ResourceTypeFolder
}

// IsValidPermissionType validates if a given permission type is one of the predefined valid types.
func IsValidPermissionType(permissionType string) bool {
	return permissionType == PermissionTypeRead ||
		permissionType == PermissionTypeWrite ||
		permissionType == PermissionTypeDelete ||
		permissionType == PermissionTypeAdmin
}

// Validate ensures that the permission has all required fields and valid values.
func (p *Permission) Validate() error {
	if p.ResourceType == "" {
		return ErrResourceTypeEmpty
	}
	if p.ResourceID == "" {
		return ErrResourceIDEmpty
	}
	if p.RoleID == "" {
		return ErrRoleIDEmpty
	}
	if p.TenantID == "" {
		return ErrTenantIDEmpty
	}
	if p.PermissionType == "" {
		return ErrPermissionTypeEmpty
	}
	if !IsValidResourceType(p.ResourceType) {
		return ErrInvalidResourceType
	}
	if !IsValidPermissionType(p.PermissionType) {
		return ErrInvalidPermissionType
	}
	return nil
}

// IsForDocument checks if this permission is for a document resource.
func (p *Permission) IsForDocument() bool {
	return p.ResourceType == ResourceTypeDocument
}

// IsForFolder checks if this permission is for a folder resource.
func (p *Permission) IsForFolder() bool {
	return p.ResourceType == ResourceTypeFolder
}

// IsReadPermission checks if this permission is a read permission.
func (p *Permission) IsReadPermission() bool {
	return p.PermissionType == PermissionTypeRead
}

// IsWritePermission checks if this permission is a write permission.
func (p *Permission) IsWritePermission() bool {
	return p.PermissionType == PermissionTypeWrite
}

// IsDeletePermission checks if this permission is a delete permission.
func (p *Permission) IsDeletePermission() bool {
	return p.PermissionType == PermissionTypeDelete
}

// IsAdminPermission checks if this permission is an admin permission.
func (p *Permission) IsAdminPermission() bool {
	return p.PermissionType == PermissionTypeAdmin
}

// MarkAsInherited marks this permission as inherited from a parent resource.
func (p *Permission) MarkAsInherited() {
	p.Inherited = true
	p.UpdatedAt = time.Now()
}

// IsInherited checks if this permission is inherited from a parent resource.
func (p *Permission) IsInherited() bool {
	return p.Inherited
}

// Clone creates a clone of this permission with a new resource ID.
// This is useful for propagating permissions from parent to child resources.
func (p *Permission) Clone(newResourceID string) *Permission {
	now := time.Now()
	return &Permission{
		TenantID:       p.TenantID,
		RoleID:         p.RoleID,
		ResourceType:   p.ResourceType,
		ResourceID:     newResourceID,
		PermissionType: p.PermissionType,
		Inherited:      true, // Cloned permissions are inherited by default
		CreatedBy:      p.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}