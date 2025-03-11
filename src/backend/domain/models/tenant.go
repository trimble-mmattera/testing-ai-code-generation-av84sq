// Package models contains the core domain models for the Document Management Platform
package models

import (
	"errors" // standard library - For error handling in validation methods
	"time"   // standard library - For timestamp fields like CreatedAt and UpdatedAt
)

// Tenant status constants
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusInactive  = "inactive"
)

// Error constants for tenant-related validation errors
var (
	ErrTenantNameEmpty = errors.New("tenant name cannot be empty")
)

// Tenant represents a customer organization in the document management platform.
// It serves as the foundation for multi-tenancy, ensuring complete data isolation
// between different customer organizations.
type Tenant struct {
	ID        string            // Unique identifier for the tenant
	Name      string            // Name of the tenant organization
	Status    string            // Current status of the tenant (active, suspended, inactive)
	CreatedAt time.Time         // Timestamp when the tenant was created
	UpdatedAt time.Time         // Timestamp when the tenant was last updated
	Settings  map[string]string // Tenant-specific configuration settings
}

// NewTenant creates a new Tenant with the given name and initializes it with default values
func NewTenant(name string) *Tenant {
	now := time.Now()
	return &Tenant{
		Name:      name,
		Status:    TenantStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Settings:  make(map[string]string),
	}
}

// Validate ensures that the tenant has all required fields and valid values
func (t *Tenant) Validate() error {
	if t.Name == "" {
		return ErrTenantNameEmpty
	}
	return nil
}

// IsActive checks if the tenant is in active status
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsSuspended checks if the tenant is in suspended status
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// IsInactive checks if the tenant is in inactive status
func (t *Tenant) IsInactive() bool {
	return t.Status == TenantStatusInactive
}

// Activate sets the tenant status to active and updates the UpdatedAt timestamp
func (t *Tenant) Activate() {
	t.Status = TenantStatusActive
	t.UpdatedAt = time.Now()
}

// Suspend sets the tenant status to suspended and updates the UpdatedAt timestamp
func (t *Tenant) Suspend() {
	t.Status = TenantStatusSuspended
	t.UpdatedAt = time.Now()
}

// Deactivate sets the tenant status to inactive and updates the UpdatedAt timestamp
func (t *Tenant) Deactivate() {
	t.Status = TenantStatusInactive
	t.UpdatedAt = time.Now()
}

// GetSetting retrieves a tenant setting by key
// Returns an empty string if the setting doesn't exist
func (t *Tenant) GetSetting(key string) string {
	if t.Settings == nil {
		return ""
	}
	return t.Settings[key]
}

// SetSetting adds or updates a tenant setting and updates the UpdatedAt timestamp
func (t *Tenant) SetSetting(key, value string) {
	if t.Settings == nil {
		t.Settings = make(map[string]string)
	}
	t.Settings[key] = value
	t.UpdatedAt = time.Now()
}

// DeleteSetting removes a tenant setting by key and updates the UpdatedAt timestamp
// Returns true if the setting was deleted, false if it didn't exist
func (t *Tenant) DeleteSetting(key string) bool {
	if t.Settings == nil {
		return false
	}
	if _, exists := t.Settings[key]; !exists {
		return false
	}
	delete(t.Settings, key)
	t.UpdatedAt = time.Now()
	return true
}

// HasSetting checks if a tenant has a specific setting
func (t *Tenant) HasSetting(key string) bool {
	if t.Settings == nil {
		return false
	}
	_, exists := t.Settings[key]
	return exists
}