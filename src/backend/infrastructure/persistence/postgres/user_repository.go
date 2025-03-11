// Package postgres provides PostgreSQL implementations of repositories.
package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid" // v1.3.0+
	"gorm.io/gorm" // v1.25.0+

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/errors"
	"../../../pkg/utils"
)

// userRepository is a PostgreSQL implementation of the UserRepository interface.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new PostgreSQL user repository instance.
func NewUserRepository(db *gorm.DB) (repositories.UserRepository, error) {
	if db == nil {
		return nil, errors.NewValidationError("database connection cannot be nil")
	}
	return &userRepository{db: db}, nil
}

// Create creates a new user in the database.
func (r *userRepository) Create(ctx context.Context, user *models.User) (string, error) {
	if err := user.Validate(); err != nil {
		return "", errors.Wrap(err, "invalid user")
	}

	// Generate a new UUID for the user ID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return "", errors.Wrap(tx.Error, "failed to begin transaction")
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return "", errors.Wrap(err, "failed to create user")
	}

	if err := tx.Commit().Error; err != nil {
		return "", errors.Wrap(err, "failed to commit transaction")
	}

	return user.ID, nil
}

// GetByID retrieves a user by its ID with tenant isolation.
func (r *userRepository) GetByID(ctx context.Context, id string, tenantID string) (*models.User, error) {
	if id == "" || tenantID == "" {
		return nil, errors.NewValidationError("user ID and tenant ID cannot be empty")
	}

	var user models.User
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return nil, errors.Wrap(err, "failed to get user by ID")
	}

	return &user, nil
}

// GetByUsername retrieves a user by username with tenant isolation.
func (r *userRepository) GetByUsername(ctx context.Context, username string, tenantID string) (*models.User, error) {
	if username == "" || tenantID == "" {
		return nil, errors.NewValidationError("username and tenant ID cannot be empty")
	}

	var user models.User
	err := r.db.WithContext(ctx).Where("username = ? AND tenant_id = ?", username, tenantID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("user with username %s not found", username))
		}
		return nil, errors.Wrap(err, "failed to get user by username")
	}

	return &user, nil
}

// GetByEmail retrieves a user by email with tenant isolation.
func (r *userRepository) GetByEmail(ctx context.Context, email string, tenantID string) (*models.User, error) {
	if email == "" || tenantID == "" {
		return nil, errors.NewValidationError("email and tenant ID cannot be empty")
	}

	var user models.User
	err := r.db.WithContext(ctx).Where("email = ? AND tenant_id = ?", email, tenantID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError(fmt.Sprintf("user with email %s not found", email))
		}
		return nil, errors.Wrap(err, "failed to get user by email")
	}

	return &user, nil
}

// Update updates an existing user with tenant isolation.
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	if err := user.Validate(); err != nil {
		return errors.Wrap(err, "invalid user")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if user exists and belongs to the tenant
	var existingUser models.User
	err := tx.Where("id = ? AND tenant_id = ?", user.ID, user.TenantID).First(&existingUser).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", user.ID))
		}
		return errors.Wrap(err, "failed to check if user exists")
	}

	// Update the user
	err = tx.Save(user).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update user")
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// Delete deletes a user by its ID with tenant isolation.
func (r *userRepository) Delete(ctx context.Context, id string, tenantID string) error {
	if id == "" || tenantID == "" {
		return errors.NewValidationError("user ID and tenant ID cannot be empty")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if user exists and belongs to the tenant
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to check if user exists")
	}

	// Delete user settings if stored separately
	if user.Settings != nil && len(user.Settings) > 0 {
		// If settings are stored in a separate table, delete them here
		// This example assumes the settings are embedded in the user object as JSONB
    }

	// Delete the user
	err = tx.Delete(&user).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to delete user")
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// ListByTenant lists all users for a tenant with pagination.
func (r *userRepository) ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.User], error) {
	if tenantID == "" {
		return utils.PaginatedResult[models.User]{}, errors.NewValidationError("tenant ID cannot be empty")
	}

	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	var users []models.User
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	
	var totalItems int64
	err := query.Model(&models.User{}).Count(&totalItems).Error
	if err != nil {
		return utils.PaginatedResult[models.User]{}, errors.Wrap(err, "failed to count users")
	}

	err = query.Limit(pagination.GetLimit()).Offset(pagination.GetOffset()).Find(&users).Error
	if err != nil {
		return utils.PaginatedResult[models.User]{}, errors.Wrap(err, "failed to list users")
	}

	return utils.NewPaginatedResult(users, pagination, totalItems), nil
}

// ListByRole lists users with a specific role with tenant isolation and pagination.
func (r *userRepository) ListByRole(ctx context.Context, role string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.User], error) {
	if role == "" || tenantID == "" {
		return utils.PaginatedResult[models.User]{}, errors.NewValidationError("role and tenant ID cannot be empty")
	}

	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	var users []models.User
	// Using PostgreSQL JSONB array contains operator to check if roles array contains the specified role
	query := r.db.WithContext(ctx).Where("tenant_id = ? AND roles @> ?", tenantID, fmt.Sprintf("[\"%s\"]", role))
	
	var totalItems int64
	err := query.Model(&models.User{}).Count(&totalItems).Error
	if err != nil {
		return utils.PaginatedResult[models.User]{}, errors.Wrap(err, "failed to count users by role")
	}

	err = query.Limit(pagination.GetLimit()).Offset(pagination.GetOffset()).Find(&users).Error
	if err != nil {
		return utils.PaginatedResult[models.User]{}, errors.Wrap(err, "failed to list users by role")
	}

	return utils.NewPaginatedResult(users, pagination, totalItems), nil
}

// ListByStatus lists users with a specific status with tenant isolation and pagination.
func (r *userRepository) ListByStatus(ctx context.Context, status string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.User], error) {
	if status == "" || tenantID == "" {
		return utils.PaginatedResult[models.User]{}, errors.NewValidationError("status and tenant ID cannot be empty")
	}

	if pagination == nil {
		pagination = utils.NewPagination(utils.DefaultPage, utils.DefaultPageSize)
	}

	var users []models.User
	query := r.db.WithContext(ctx).Where("tenant_id = ? AND status = ?", tenantID, status)
	
	var totalItems int64
	err := query.Model(&models.User{}).Count(&totalItems).Error
	if err != nil {
		return utils.PaginatedResult[models.User]{}, errors.Wrap(err, "failed to count users by status")
	}

	err = query.Limit(pagination.GetLimit()).Offset(pagination.GetOffset()).Find(&users).Error
	if err != nil {
		return utils.PaginatedResult[models.User]{}, errors.Wrap(err, "failed to list users by status")
	}

	return utils.NewPaginatedResult(users, pagination, totalItems), nil
}

// UpdateStatus updates the status of a user with tenant isolation.
func (r *userRepository) UpdateStatus(ctx context.Context, id string, status string, tenantID string) error {
	if id == "" || status == "" || tenantID == "" {
		return errors.NewValidationError("user ID, status, and tenant ID cannot be empty")
	}

	// Validate status
	validStatuses := []string{models.UserStatusActive, models.UserStatusInactive, models.UserStatusSuspended}
	isValidStatus := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return errors.NewValidationError(fmt.Sprintf("invalid status: %s", status))
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if user exists and belongs to the tenant
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to check if user exists")
	}

	// Update the status
	err = tx.Model(&user).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update user status")
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// AddRole adds a role to a user with tenant isolation.
func (r *userRepository) AddRole(ctx context.Context, id string, role string, tenantID string) error {
	if id == "" || role == "" || tenantID == "" {
		return errors.NewValidationError("user ID, role, and tenant ID cannot be empty")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Get user with tenant isolation
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to get user")
	}

	// Check if user already has the role and add it if not
	if user.AddRole(role) {
		// Role was added, update the user
		err = tx.Save(&user).Error
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to add role to user")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// RemoveRole removes a role from a user with tenant isolation.
func (r *userRepository) RemoveRole(ctx context.Context, id string, role string, tenantID string) error {
	if id == "" || role == "" || tenantID == "" {
		return errors.NewValidationError("user ID, role, and tenant ID cannot be empty")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Get user with tenant isolation
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to get user")
	}

	// Check if user has the role and remove it if present
	if user.RemoveRole(role) {
		// Role was removed, update the user
		err = tx.Save(&user).Error
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to remove role from user")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// UpdatePassword updates the password of a user with tenant isolation.
func (r *userRepository) UpdatePassword(ctx context.Context, id string, passwordHash string, tenantID string) error {
	if id == "" || passwordHash == "" || tenantID == "" {
		return errors.NewValidationError("user ID, password hash, and tenant ID cannot be empty")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Check if user exists and belongs to the tenant
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to check if user exists")
	}

	// Update the password hash
	err = tx.Model(&user).Updates(map[string]interface{}{
		"password_hash": passwordHash,
		"updated_at":    time.Now(),
	}).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to update user password")
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// SetSetting sets a user setting with tenant isolation.
func (r *userRepository) SetSetting(ctx context.Context, id string, key string, value string, tenantID string) error {
	if id == "" || key == "" || tenantID == "" {
		return errors.NewValidationError("user ID, key, and tenant ID cannot be empty")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Get user with tenant isolation
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to get user")
	}

	// Set the setting using the domain model method
	user.SetSetting(key, value)

	// Update the user
	err = tx.Save(&user).Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "failed to set user setting")
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// DeleteSetting deletes a user setting with tenant isolation.
func (r *userRepository) DeleteSetting(ctx context.Context, id string, key string, tenantID string) error {
	if id == "" || key == "" || tenantID == "" {
		return errors.NewValidationError("user ID, key, and tenant ID cannot be empty")
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return errors.Wrap(tx.Error, "failed to begin transaction")
	}

	// Get user with tenant isolation
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return errors.Wrap(err, "failed to get user")
	}

	// Delete the setting using the domain model method
	if user.DeleteSetting(key) {
		// Setting was deleted, update the user
		err = tx.Save(&user).Error
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "failed to delete user setting")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// GetSetting gets a user setting with tenant isolation.
func (r *userRepository) GetSetting(ctx context.Context, id string, key string, tenantID string) (string, error) {
	if id == "" || key == "" || tenantID == "" {
		return "", errors.NewValidationError("user ID, key, and tenant ID cannot be empty")
	}

	// Get user with tenant isolation
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.NewResourceNotFoundError(fmt.Sprintf("user with ID %s not found", id))
		}
		return "", errors.Wrap(err, "failed to get user")
	}

	// Get the setting value using the domain model method
	value := user.GetSetting(key)
	if value == "" && !user.HasSetting(key) {
		return "", errors.NewResourceNotFoundError(fmt.Sprintf("setting %s not found", key))
	}

	return value, nil
}

// Exists checks if a user exists by ID with tenant isolation.
func (r *userRepository) Exists(ctx context.Context, id string, tenantID string) (bool, error) {
	if id == "" || tenantID == "" {
		return false, errors.NewValidationError("user ID and tenant ID cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ? AND tenant_id = ?", id, tenantID).Count(&count).Error
	if err != nil {
		return false, errors.Wrap(err, "failed to check if user exists")
	}

	return count > 0, nil
}

// ExistsByUsername checks if a user exists by username with tenant isolation.
func (r *userRepository) ExistsByUsername(ctx context.Context, username string, tenantID string) (bool, error) {
	if username == "" || tenantID == "" {
		return false, errors.NewValidationError("username and tenant ID cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("username = ? AND tenant_id = ?", username, tenantID).Count(&count).Error
	if err != nil {
		return false, errors.Wrap(err, "failed to check if user exists by username")
	}

	return count > 0, nil
}

// ExistsByEmail checks if a user exists by email with tenant isolation.
func (r *userRepository) ExistsByEmail(ctx context.Context, email string, tenantID string) (bool, error) {
	if email == "" || tenantID == "" {
		return false, errors.NewValidationError("email and tenant ID cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ? AND tenant_id = ?", email, tenantID).Count(&count).Error
	if err != nil {
		return false, errors.Wrap(err, "failed to check if user exists by email")
	}

	return count > 0, nil
}

// Count counts the total number of users for a tenant.
func (r *userRepository) Count(ctx context.Context, tenantID string) (int64, error) {
	if tenantID == "" {
		return 0, errors.NewValidationError("tenant ID cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("tenant_id = ?", tenantID).Count(&count).Error
	if err != nil {
		return 0, errors.Wrap(err, "failed to count users")
	}

	return count, nil
}

// CountByStatus counts the number of users with a specific status for a tenant.
func (r *userRepository) CountByStatus(ctx context.Context, status string, tenantID string) (int64, error) {
	if status == "" || tenantID == "" {
		return 0, errors.NewValidationError("status and tenant ID cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("tenant_id = ? AND status = ?", tenantID, status).Count(&count).Error
	if err != nil {
		return 0, errors.Wrap(err, "failed to count users by status")
	}

	return count, nil
}

// CountByRole counts the number of users with a specific role for a tenant.
func (r *userRepository) CountByRole(ctx context.Context, role string, tenantID string) (int64, error) {
	if role == "" || tenantID == "" {
		return 0, errors.NewValidationError("role and tenant ID cannot be empty")
	}

	var count int64
	// Using PostgreSQL JSONB array contains operator to check if roles array contains the specified role
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("tenant_id = ? AND roles @> ?", tenantID, fmt.Sprintf("[\"%s\"]", role)).Count(&count).Error
	if err != nil {
		return 0, errors.Wrap(err, "failed to count users by role")
	}

	return count, nil
}