package models

import (
	"errors" // standard library
	"time"   // standard library
)

// Role constants define the standard roles available in the system
const (
	RoleReader        = "reader"
	RoleContributor   = "contributor"
	RoleEditor        = "editor"
	RoleAdministrator = "administrator"
	RoleSystem        = "system"
)

// Error constants for role validation
var (
	ErrNameEmpty        = errors.New("role name cannot be empty")
	ErrTenantIDEmpty    = errors.New("tenant ID cannot be empty")
	ErrDescriptionEmpty = errors.New("role description cannot be empty")
)

// Role represents a role in the document management platform that defines a set of permissions
type Role struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewRole creates a new Role with the given name, description, and tenant ID
func NewRole(name, description, tenantID string) *Role {
	return &Role{
		Name:        name,
		Description: description,
		TenantID:    tenantID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// Validate checks if the role has all required fields
func (r *Role) Validate() error {
	if r.Name == "" {
		return ErrNameEmpty
	}
	if r.TenantID == "" {
		return ErrTenantIDEmpty
	}
	if r.Description == "" {
		return ErrDescriptionEmpty
	}
	return nil
}

// IsReader checks if this role is the Reader role
func (r *Role) IsReader() bool {
	return r.Name == RoleReader
}

// IsContributor checks if this role is the Contributor role
func (r *Role) IsContributor() bool {
	return r.Name == RoleContributor
}

// IsEditor checks if this role is the Editor role
func (r *Role) IsEditor() bool {
	return r.Name == RoleEditor
}

// IsAdministrator checks if this role is the Administrator role
func (r *Role) IsAdministrator() bool {
	return r.Name == RoleAdministrator
}

// IsSystem checks if this role is the System role
func (r *Role) IsSystem() bool {
	return r.Name == RoleSystem
}

// IsSystemRole checks if a role name is one of the predefined system roles
func IsSystemRole(roleName string) bool {
	return roleName == RoleReader ||
		roleName == RoleContributor ||
		roleName == RoleEditor ||
		roleName == RoleAdministrator ||
		roleName == RoleSystem
}

// IsSystemRole checks if this role is one of the predefined system roles
func (r *Role) IsSystemRole() bool {
	return IsSystemRole(r.Name)
}

// CanRead checks if this role has read permissions
func (r *Role) CanRead() bool {
	// All roles have read permissions
	return true
}

// CanWrite checks if this role has write permissions
func (r *Role) CanWrite() bool {
	// Contributor, Editor, Administrator, and System roles can write
	return r.IsContributor() || r.IsEditor() || r.IsAdministrator() || r.IsSystem()
}

// CanDelete checks if this role has delete permissions
func (r *Role) CanDelete() bool {
	// Editor, Administrator, and System roles can delete
	return r.IsEditor() || r.IsAdministrator() || r.IsSystem()
}

// CanManageFolders checks if this role can manage folders
func (r *Role) CanManageFolders() bool {
	// Only Administrator and System roles can manage folders
	return r.IsAdministrator() || r.IsSystem()
}