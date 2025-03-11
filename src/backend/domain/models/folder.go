// Package models defines domain models for the document management system.
package models

import (
	"errors" // standard library - for error handling in validation methods
	"path"   // standard library - for path manipulation in folder hierarchy 
	"strings" // standard library - for string manipulation in path handling
	"time"   // standard library - for timestamp fields like CreatedAt and UpdatedAt
)

// PathSeparator is the separator used in folder paths
const PathSeparator = "/"

// Error definitions for folder validation
var (
	ErrFolderNameEmpty = errors.New("folder name cannot be empty")
	ErrTenantIDEmpty   = errors.New("tenant ID cannot be empty")
	ErrOwnerIDEmpty    = errors.New("owner ID cannot be empty")
)

// Folder represents a folder in the document management system with hierarchical structure.
// It maintains tenant isolation through the TenantID field and tracks ownership and timestamps.
type Folder struct {
	ID        string    // Unique identifier for the folder
	Name      string    // Display name of the folder
	ParentID  string    // ID of the parent folder (empty for root folders)
	Path      string    // Full path to the folder (used for hierarchical operations)
	TenantID  string    // ID of the tenant owning the folder (for tenant isolation)
	OwnerID   string    // ID of the user who created the folder
	CreatedAt time.Time // Timestamp when the folder was created
	UpdatedAt time.Time // Timestamp when the folder was last updated
}

// NewFolder creates a new Folder instance with the given parameters
func NewFolder(name, parentID, tenantID, ownerID string) *Folder {
	now := time.Now()
	return &Folder{
		Name:      name,
		ParentID:  parentID,
		TenantID:  tenantID,
		OwnerID:   ownerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate checks if the folder has all required fields
func (f *Folder) Validate() error {
	if strings.TrimSpace(f.Name) == "" {
		return ErrFolderNameEmpty
	}
	if strings.TrimSpace(f.TenantID) == "" {
		return ErrTenantIDEmpty
	}
	if strings.TrimSpace(f.OwnerID) == "" {
		return ErrOwnerIDEmpty
	}
	return nil
}

// IsRoot checks if the folder is a root folder (no parent)
func (f *Folder) IsRoot() bool {
	return f.ParentID == ""
}

// BuildPath builds the full path for the folder based on parent path and folder name
func (f *Folder) BuildPath(parentPath string) string {
	if parentPath == "" {
		return PathSeparator + f.Name
	}
	
	// Ensure parentPath starts with a separator
	if !strings.HasPrefix(parentPath, PathSeparator) {
		parentPath = PathSeparator + parentPath
	}
	
	// Ensure parentPath ends with a separator
	if !strings.HasSuffix(parentPath, PathSeparator) {
		parentPath = parentPath + PathSeparator
	}
	
	return parentPath + f.Name
}

// SetPath sets the path for the folder
func (f *Folder) SetPath(path string) {
	f.Path = path
	f.UpdatedAt = time.Now()
}

// GetName returns the folder name
func (f *Folder) GetName() string {
	return f.Name
}

// GetPath returns the folder path
func (f *Folder) GetPath() string {
	return f.Path
}

// GetParentID returns the parent folder ID
func (f *Folder) GetParentID() string {
	return f.ParentID
}

// GetTenantID returns the tenant ID
func (f *Folder) GetTenantID() string {
	return f.TenantID
}

// GetOwnerID returns the owner ID
func (f *Folder) GetOwnerID() string {
	return f.OwnerID
}

// IsDescendantOf checks if this folder is a descendant of the specified folder
func (f *Folder) IsDescendantOf(ancestorPath string) bool {
	if ancestorPath == "" || f.Path == "" {
		return false
	}
	
	// Ensure ancestorPath ends with a separator for proper prefix checking
	if !strings.HasSuffix(ancestorPath, PathSeparator) {
		ancestorPath = ancestorPath + PathSeparator
	}
	
	return strings.HasPrefix(f.Path, ancestorPath)
}

// Update updates the folder's metadata
func (f *Folder) Update(name string) {
	f.Name = name
	f.UpdatedAt = time.Now()
}

// SetParent updates the folder's parent ID
func (f *Folder) SetParent(parentID string) {
	f.ParentID = parentID
	f.UpdatedAt = time.Now()
}