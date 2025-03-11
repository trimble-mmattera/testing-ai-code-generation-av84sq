package models

import (
	"errors"  // v1.21+ (standard library)
	"time"    // v1.21+ (standard library)
)

// Tag represents a metadata tag that can be associated with documents
type Tag struct {
	ID        string    // Unique identifier for the tag
	Name      string    // Name of the tag
	TenantID  string    // ID of the tenant this tag belongs to, ensures tenant isolation
	CreatedAt time.Time // Timestamp when the tag was created
	UpdatedAt time.Time // Timestamp when the tag was last updated
}

// NewTag creates a new Tag with the given name and tenant ID
func NewTag(name string, tenantID string) Tag {
	now := time.Now()
	return Tag{
		Name:      name,
		TenantID:  tenantID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate checks if the tag has all required fields
func (t *Tag) Validate() error {
	if t.Name == "" {
		return errors.New("tag name cannot be empty")
	}
	
	if t.TenantID == "" {
		return errors.New("tenant ID cannot be empty")
	}
	
	return nil
}

// Equals checks if this tag is equal to another tag
func (t *Tag) Equals(other *Tag) bool {
	if other == nil {
		return false
	}
	
	return t.ID == other.ID
}

// Clone creates a deep copy of the tag
func (t *Tag) Clone() *Tag {
	clone := *t
	return &clone
}