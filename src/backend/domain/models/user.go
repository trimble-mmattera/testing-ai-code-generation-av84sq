// Package models provides domain models for the Document Management Platform
package models

import (
	"errors"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt" // v0.0.0-20220622213112-05595931fe9d
)

// User status constants
const (
	UserStatusActive    = "active"
	UserStatusInactive  = "inactive"
	UserStatusSuspended = "suspended"
)

// Error constants for user validation
var (
	ErrUsernameTooShort = errors.New("username must be at least 3 characters long")
	ErrEmailInvalid     = errors.New("email address is invalid")
	ErrPasswordTooWeak  = errors.New("password must be at least 8 characters long")
	ErrTenantIDEmpty    = errors.New("tenant ID cannot be empty")
	DefaultBcryptCost   = bcrypt.DefaultCost
)

// emailRegex is a simple regex for basic email validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// User represents a user in the document management platform
type User struct {
	ID           string            // Unique identifier for the user
	TenantID     string            // ID of the tenant this user belongs to
	Username     string            // User's username for login
	Email        string            // User's email address
	PasswordHash string            // Bcrypt hash of the user's password
	Status       string            // User status: active, inactive, suspended
	Roles        []string          // User's assigned roles
	CreatedAt    time.Time         // When the user was created
	UpdatedAt    time.Time         // When the user was last updated
	Settings     map[string]string // User-specific settings
}

// NewUser creates a new User with the given username, email, and tenant ID
func NewUser(username, email, tenantID string) *User {
	return &User{
		Username:  username,
		Email:     email,
		TenantID:  tenantID,
		Status:    UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles:     []string{},
		Settings:  make(map[string]string),
	}
}

// Validate checks that the user has all required fields
func (u *User) Validate() error {
	if len(u.Username) < 3 {
		return ErrUsernameTooShort
	}

	if !emailRegex.MatchString(u.Email) {
		return ErrEmailInvalid
	}

	if u.TenantID == "" {
		return ErrTenantIDEmpty
	}

	return nil
}

// SetPassword sets the user's password by hashing it
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooWeak
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now()
	return nil
}

// VerifyPassword verifies if the provided password matches the stored hash
func (u *User) VerifyPassword(password string) (bool, error) {
	if u.PasswordHash == "" {
		return false, nil
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsActive checks if the user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsInactive checks if the user is inactive
func (u *User) IsInactive() bool {
	return u.Status == UserStatusInactive
}

// IsSuspended checks if the user is suspended
func (u *User) IsSuspended() bool {
	return u.Status == UserStatusSuspended
}

// Activate activates the user
func (u *User) Activate() {
	u.Status = UserStatusActive
	u.UpdatedAt = time.Now()
}

// Deactivate deactivates the user
func (u *User) Deactivate() {
	u.Status = UserStatusInactive
	u.UpdatedAt = time.Now()
}

// Suspend suspends the user
func (u *User) Suspend() {
	u.Status = UserStatusSuspended
	u.UpdatedAt = time.Now()
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AddRole adds a role to the user if they don't already have it
func (u *User) AddRole(role string) bool {
	if u.HasRole(role) {
		return false
	}
	
	u.Roles = append(u.Roles, role)
	u.UpdatedAt = time.Now()
	return true
}

// RemoveRole removes a role from the user
func (u *User) RemoveRole(role string) bool {
	if !u.HasRole(role) {
		return false
	}
	
	var newRoles []string
	for _, r := range u.Roles {
		if r != role {
			newRoles = append(newRoles, r)
		}
	}
	
	u.Roles = newRoles
	u.UpdatedAt = time.Now()
	return true
}

// GetSetting gets a user setting by key
func (u *User) GetSetting(key string) string {
	if u.Settings == nil {
		return ""
	}
	return u.Settings[key]
}

// SetSetting sets a user setting
func (u *User) SetSetting(key, value string) {
	if u.Settings == nil {
		u.Settings = make(map[string]string)
	}
	u.Settings[key] = value
	u.UpdatedAt = time.Now()
}

// DeleteSetting deletes a user setting
func (u *User) DeleteSetting(key string) bool {
	if u.Settings == nil {
		return false
	}
	
	_, exists := u.Settings[key]
	if !exists {
		return false
	}
	
	delete(u.Settings, key)
	u.UpdatedAt = time.Now()
	return true
}

// HasSetting checks if a user has a specific setting
func (u *User) HasSetting(key string) bool {
	if u.Settings == nil {
		return false
	}
	
	_, exists := u.Settings[key]
	return exists
}

// CanRead checks if the user has read permissions
// All users have read permissions by default
func (u *User) CanRead() bool {
	return true
}

// CanWrite checks if the user has write permissions
func (u *User) CanWrite() bool {
	return u.HasRole("contributor") || u.HasRole("editor") || 
	       u.HasRole("administrator") || u.HasRole("system")
}

// CanDelete checks if the user has delete permissions
func (u *User) CanDelete() bool {
	return u.HasRole("editor") || u.HasRole("administrator") || u.HasRole("system")
}

// CanManageFolders checks if the user can manage folders
func (u *User) CanManageFolders() bool {
	return u.HasRole("administrator") || u.HasRole("system")
}