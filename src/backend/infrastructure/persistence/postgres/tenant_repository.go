// Package postgres provides PostgreSQL implementations of the repository interfaces.
package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid" // v1.3.0+
	"gorm.io/gorm" // v1.25.0+

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/errors"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

// tenantRepository is a PostgreSQL implementation of the TenantRepository interface.
type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository creates a new PostgreSQL implementation of the TenantRepository interface.
func NewTenantRepository(db *gorm.DB) repositories.TenantRepository {
	if db == nil {
		logger.Error("nil db parameter passed to NewTenantRepository")
		panic("nil db parameter")
	}

	return &tenantRepository{
		db: db,
	}
}

// Create creates a new tenant in the database.
func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) (string, error) {
	if err := tenant.Validate(); err != nil {
		return "", errors.NewValidationError("invalid tenant data: " + err.Error())
	}

	// Generate a new UUID for the tenant ID if not provided
	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}

	// Set timestamps if not already set
	now := time.Now()
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = now
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = now
	}

	// Create the tenant in the database
	if err := r.db.WithContext(ctx).Create(tenant).Error; err != nil {
		logger.ErrorContext(ctx, "failed to create tenant", "error", err, "tenant_name", tenant.Name)
		return "", errors.NewDatabaseError("failed to create tenant: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant created successfully", "tenant_id", tenant.ID, "tenant_name", tenant.Name)
	return tenant.ID, nil
}

// GetByID retrieves a tenant by its ID.
func (r *tenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if id == "" {
		return nil, errors.NewValidationError("tenant ID cannot be empty")
	}

	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError("tenant not found")
		}
		logger.ErrorContext(ctx, "failed to get tenant by ID", "error", err, "tenant_id", id)
		return nil, errors.NewDatabaseError("failed to get tenant: " + err.Error())
	}

	return &tenant, nil
}

// GetByName retrieves a tenant by its name.
func (r *tenantRepository) GetByName(ctx context.Context, name string) (*models.Tenant, error) {
	if name == "" {
		return nil, errors.NewValidationError("tenant name cannot be empty")
	}

	var tenant models.Tenant
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewResourceNotFoundError("tenant not found")
		}
		logger.ErrorContext(ctx, "failed to get tenant by name", "error", err, "tenant_name", name)
		return nil, errors.NewDatabaseError("failed to get tenant: " + err.Error())
	}

	return &tenant, nil
}

// Update updates an existing tenant.
func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	if err := tenant.Validate(); err != nil {
		return errors.NewValidationError("invalid tenant data: " + err.Error())
	}

	// Check if the tenant exists
	exists, err := r.Exists(ctx, tenant.ID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewResourceNotFoundError("tenant not found")
	}

	// Update the UpdatedAt timestamp
	tenant.UpdatedAt = time.Now()

	// Update the tenant in the database
	if err := r.db.WithContext(ctx).Save(tenant).Error; err != nil {
		logger.ErrorContext(ctx, "failed to update tenant", "error", err, "tenant_id", tenant.ID)
		return errors.NewDatabaseError("failed to update tenant: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant updated successfully", "tenant_id", tenant.ID)
	return nil
}

// Delete deletes a tenant by its ID.
func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Check if the tenant exists
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewResourceNotFoundError("tenant not found")
	}

	// Delete the tenant from the database
	if err := r.db.WithContext(ctx).Delete(&models.Tenant{}, "id = ?", id).Error; err != nil {
		logger.ErrorContext(ctx, "failed to delete tenant", "error", err, "tenant_id", id)
		return errors.NewDatabaseError("failed to delete tenant: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant deleted successfully", "tenant_id", id)
	return nil
}

// List lists all tenants with pagination.
func (r *tenantRepository) List(ctx context.Context, pagination *utils.Pagination) (utils.PaginatedResult[models.Tenant], error) {
	var tenants []models.Tenant
	var total int64

	// Count total number of tenants
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Count(&total).Error; err != nil {
		logger.ErrorContext(ctx, "failed to count tenants", "error", err)
		return utils.PaginatedResult[models.Tenant]{}, errors.NewDatabaseError("failed to count tenants: " + err.Error())
	}

	// Get paginated tenants
	if err := r.db.WithContext(ctx).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&tenants).Error; err != nil {
		logger.ErrorContext(ctx, "failed to list tenants", "error", err)
		return utils.PaginatedResult[models.Tenant]{}, errors.NewDatabaseError("failed to list tenants: " + err.Error())
	}

	return utils.NewPaginatedResult(tenants, pagination, total), nil
}

// ListByStatus lists tenants by status with pagination.
func (r *tenantRepository) ListByStatus(ctx context.Context, status string, pagination *utils.Pagination) (utils.PaginatedResult[models.Tenant], error) {
	if status == "" {
		return utils.PaginatedResult[models.Tenant]{}, errors.NewValidationError("status cannot be empty")
	}

	var tenants []models.Tenant
	var total int64

	// Count total number of tenants with the given status
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("status = ?", status).Count(&total).Error; err != nil {
		logger.ErrorContext(ctx, "failed to count tenants by status", "error", err, "status", status)
		return utils.PaginatedResult[models.Tenant]{}, errors.NewDatabaseError("failed to count tenants: " + err.Error())
	}

	// Get paginated tenants with the given status
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&tenants).Error; err != nil {
		logger.ErrorContext(ctx, "failed to list tenants by status", "error", err, "status", status)
		return utils.PaginatedResult[models.Tenant]{}, errors.NewDatabaseError("failed to list tenants: " + err.Error())
	}

	return utils.NewPaginatedResult(tenants, pagination, total), nil
}

// UpdateStatus updates the status of a tenant.
func (r *tenantRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	if id == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}
	if status == "" {
		return errors.NewValidationError("status cannot be empty")
	}

	// Check if the tenant exists
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewResourceNotFoundError("tenant not found")
	}

	// Update the tenant status in the database
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error; err != nil {
		logger.ErrorContext(ctx, "failed to update tenant status", "error", err, "tenant_id", id, "status", status)
		return errors.NewDatabaseError("failed to update tenant status: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant status updated successfully", "tenant_id", id, "status", status)
	return nil
}

// UpdateSettings updates the settings of a tenant.
func (r *tenantRepository) UpdateSettings(ctx context.Context, id string, settings map[string]string) error {
	if id == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}

	// Get the tenant
	tenant, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Update settings
	if tenant.Settings == nil {
		tenant.Settings = settings
	} else {
		for key, value := range settings {
			tenant.Settings[key] = value
		}
	}

	// Update the tenant in the database
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"settings":   tenant.Settings,
			"updated_at": time.Now(),
		}).Error; err != nil {
		logger.ErrorContext(ctx, "failed to update tenant settings", "error", err, "tenant_id", id)
		return errors.NewDatabaseError("failed to update tenant settings: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant settings updated successfully", "tenant_id", id)
	return nil
}

// GetSetting gets a specific setting of a tenant.
func (r *tenantRepository) GetSetting(ctx context.Context, id string, key string) (string, error) {
	if id == "" {
		return "", errors.NewValidationError("tenant ID cannot be empty")
	}
	if key == "" {
		return "", errors.NewValidationError("setting key cannot be empty")
	}

	// Get the tenant
	tenant, err := r.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	// Check if the setting exists
	if !tenant.HasSetting(key) {
		return "", errors.NewResourceNotFoundError("setting not found")
	}

	// Return the setting value
	return tenant.GetSetting(key), nil
}

// SetSetting sets a specific setting of a tenant.
func (r *tenantRepository) SetSetting(ctx context.Context, id string, key string, value string) error {
	if id == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}
	if key == "" {
		return errors.NewValidationError("setting key cannot be empty")
	}

	// Get the tenant
	tenant, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Set the setting
	tenant.SetSetting(key, value)

	// Update the tenant in the database
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"settings":   tenant.Settings,
			"updated_at": time.Now(),
		}).Error; err != nil {
		logger.ErrorContext(ctx, "failed to set tenant setting", "error", err, "tenant_id", id, "key", key)
		return errors.NewDatabaseError("failed to set tenant setting: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant setting set successfully", "tenant_id", id, "key", key)
	return nil
}

// DeleteSetting deletes a specific setting of a tenant.
func (r *tenantRepository) DeleteSetting(ctx context.Context, id string, key string) error {
	if id == "" {
		return errors.NewValidationError("tenant ID cannot be empty")
	}
	if key == "" {
		return errors.NewValidationError("setting key cannot be empty")
	}

	// Get the tenant
	tenant, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete the setting
	if !tenant.DeleteSetting(key) {
		return errors.NewResourceNotFoundError("setting not found")
	}

	// Update the tenant in the database
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"settings":   tenant.Settings,
			"updated_at": time.Now(),
		}).Error; err != nil {
		logger.ErrorContext(ctx, "failed to delete tenant setting", "error", err, "tenant_id", id, "key", key)
		return errors.NewDatabaseError("failed to delete tenant setting: " + err.Error())
	}

	logger.InfoContext(ctx, "tenant setting deleted successfully", "tenant_id", id, "key", key)
	return nil
}

// Exists checks if a tenant exists by ID.
func (r *tenantRepository) Exists(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, errors.NewValidationError("tenant ID cannot be empty")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("id = ?", id).Count(&count).Error; err != nil {
		logger.ErrorContext(ctx, "failed to check if tenant exists", "error", err, "tenant_id", id)
		return false, errors.NewDatabaseError("failed to check if tenant exists: " + err.Error())
	}

	return count > 0, nil
}

// ExistsByName checks if a tenant exists by name.
func (r *tenantRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, errors.NewValidationError("tenant name cannot be empty")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("name = ?", name).Count(&count).Error; err != nil {
		logger.ErrorContext(ctx, "failed to check if tenant exists by name", "error", err, "tenant_name", name)
		return false, errors.NewDatabaseError("failed to check if tenant exists by name: " + err.Error())
	}

	return count > 0, nil
}

// Count counts the total number of tenants.
func (r *tenantRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Count(&count).Error; err != nil {
		logger.ErrorContext(ctx, "failed to count tenants", "error", err)
		return 0, errors.NewDatabaseError("failed to count tenants: " + err.Error())
	}

	return count, nil
}

// CountByStatus counts the number of tenants with a specific status.
func (r *tenantRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	if status == "" {
		return 0, errors.NewValidationError("status cannot be empty")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("status = ?", status).Count(&count).Error; err != nil {
		logger.ErrorContext(ctx, "failed to count tenants by status", "error", err, "status", status)
		return 0, errors.NewDatabaseError("failed to count tenants by status: " + err.Error())
	}

	return count, nil
}