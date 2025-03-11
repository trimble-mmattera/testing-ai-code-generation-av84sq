// Package redis implements Redis-based cache providers for the Document Management Platform.
package redis

import (
	"context" // standard library
	"fmt"     // standard library
	"time"    // standard library

	"../../../domain/models"
	"../../../domain/repositories"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

const (
	// documentCacheTTL defines the time-to-live for document cache entries (15 minutes)
	documentCacheTTL = 15 * time.Minute

	// Cache key prefixes for different types of cached data
	documentKeyPrefix       = "document:"
	documentVersionKeyPrefix = "document:version:"
	documentListKeyPrefix   = "document:list:"
	documentSearchKeyPrefix = "document:search:"
)

// DocumentCache implements the DocumentRepository interface with Redis caching
type DocumentCache struct {
	redisClient *RedisClient
	repository  repositories.DocumentRepository
}

// NewDocumentCache creates a new DocumentCache instance that wraps a DocumentRepository
func NewDocumentCache(redisClient *RedisClient, repository repositories.DocumentRepository) repositories.DocumentRepository {
	return &DocumentCache{
		redisClient: redisClient,
		repository:  repository,
	}
}

// Create creates a new document and invalidates related cache entries
func (c *DocumentCache) Create(ctx context.Context, document *models.Document) (string, error) {
	// Delegate document creation to the underlying repository
	id, err := c.repository.Create(ctx, document)
	if err != nil {
		return "", err
	}

	// Invalidate folder document list cache
	if document.FolderID != "" {
		if err := c.invalidateFolderListCache(ctx, document.FolderID, document.TenantID); err != nil {
			logger.Error("Failed to invalidate folder list cache", "error", err, "folder_id", document.FolderID)
		}
	}

	return id, nil
}

// GetByID retrieves a document by ID, using cache when available
func (c *DocumentCache) GetByID(ctx context.Context, id string, tenantID string) (*models.Document, error) {
	// Generate cache key using document ID and tenant ID
	key := c.generateDocumentKey(id, tenantID)

	// Try to get document from cache
	var document *models.Document
	exist, err := c.redisClient.Get(ctx, key, &document)
	if err != nil {
		logger.Error("Error getting document from cache", "error", err, "id", id)
	}

	// If found in cache, return the document
	if exist && document != nil {
		logger.Debug("Cache hit for document", "id", id, "tenant_id", tenantID)
		return document, nil
	}

	// If not in cache, get from repository
	logger.Debug("Cache miss for document", "id", id, "tenant_id", tenantID)
	document, err = c.repository.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// If found in repository, store in cache with TTL
	if document != nil {
		if err := c.redisClient.Set(ctx, key, document, documentCacheTTL); err != nil {
			logger.Error("Failed to cache document", "error", err, "id", id)
		}
	}

	return document, nil
}

// Update updates a document and updates or invalidates related cache entries
func (c *DocumentCache) Update(ctx context.Context, document *models.Document) error {
	// Delegate document update to the underlying repository
	if err := c.repository.Update(ctx, document); err != nil {
		return err
	}

	// If successful, update document in cache
	key := c.generateDocumentKey(document.ID, document.TenantID)
	if err := c.redisClient.Set(ctx, key, document, documentCacheTTL); err != nil {
		logger.Error("Failed to update document in cache", "error", err, "id", document.ID)
	}

	// Invalidate folder document list cache
	if err := c.invalidateFolderListCache(ctx, document.FolderID, document.TenantID); err != nil {
		logger.Error("Failed to invalidate folder list cache", "error", err, "folder_id", document.FolderID)
	}

	return nil
}

// Delete deletes a document and invalidates related cache entries
func (c *DocumentCache) Delete(ctx context.Context, id string, tenantID string) error {
	// Get document from repository to determine folder ID
	document, err := c.repository.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// Delegate document deletion to the underlying repository
	if err := c.repository.Delete(ctx, id, tenantID); err != nil {
		return err
	}

	// If successful, delete document from cache
	key := c.generateDocumentKey(id, tenantID)
	if err := c.redisClient.Del(ctx, key); err != nil {
		logger.Error("Failed to delete document from cache", "error", err, "id", id)
	}

	// Invalidate folder document list cache
	if document != nil && document.FolderID != "" {
		if err := c.invalidateFolderListCache(ctx, document.FolderID, tenantID); err != nil {
			logger.Error("Failed to invalidate folder list cache", "error", err, "folder_id", document.FolderID)
		}
	}

	return nil
}

// ListByFolder lists documents in a folder with pagination, using cache when available
func (c *DocumentCache) ListByFolder(ctx context.Context, folderID string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key using folder ID, tenant ID, and pagination parameters
	key := c.generateListKey(folderID, tenantID, pagination)

	// Try to get document list from cache
	var result utils.PaginatedResult[models.Document]
	exist, err := c.redisClient.Get(ctx, key, &result)
	if err != nil {
		logger.Error("Error getting folder documents from cache", "error", err, "folder_id", folderID)
	}

	// If found in cache, return the paginated result
	if exist {
		logger.Debug("Cache hit for folder documents", "folder_id", folderID, "tenant_id", tenantID)
		return result, nil
	}

	// If not in cache, get from repository
	logger.Debug("Cache miss for folder documents", "folder_id", folderID, "tenant_id", tenantID)
	result, err = c.repository.ListByFolder(ctx, folderID, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// If found in repository, store in cache with TTL
	if err := c.redisClient.Set(ctx, key, result, documentCacheTTL); err != nil {
		logger.Error("Failed to cache folder documents", "error", err, "folder_id", folderID)
	}

	return result, nil
}

// ListByTenant lists all documents for a tenant with pagination, using cache when available
func (c *DocumentCache) ListByTenant(ctx context.Context, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key using tenant ID and pagination parameters
	key := c.generateTenantListKey(tenantID, pagination)

	// Try to get document list from cache
	var result utils.PaginatedResult[models.Document]
	exist, err := c.redisClient.Get(ctx, key, &result)
	if err != nil {
		logger.Error("Error getting tenant documents from cache", "error", err, "tenant_id", tenantID)
	}

	// If found in cache, return the paginated result
	if exist {
		logger.Debug("Cache hit for tenant documents", "tenant_id", tenantID)
		return result, nil
	}

	// If not in cache, get from repository
	logger.Debug("Cache miss for tenant documents", "tenant_id", tenantID)
	result, err = c.repository.ListByTenant(ctx, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// If found in repository, store in cache with TTL
	if err := c.redisClient.Set(ctx, key, result, documentCacheTTL); err != nil {
		logger.Error("Failed to cache tenant documents", "error", err, "tenant_id", tenantID)
	}

	return result, nil
}

// SearchByContent searches documents by content with pagination, using cache when available
func (c *DocumentCache) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key using search query, tenant ID, and pagination parameters
	key := c.generateSearchKey(query, tenantID, pagination)

	// Try to get search results from cache
	var result utils.PaginatedResult[models.Document]
	exist, err := c.redisClient.Get(ctx, key, &result)
	if err != nil {
		logger.Error("Error getting search results from cache", "error", err, "query", query)
	}

	// If found in cache, return the paginated result
	if exist {
		logger.Debug("Cache hit for content search", "query", query, "tenant_id", tenantID)
		return result, nil
	}

	// If not in cache, get from repository
	logger.Debug("Cache miss for content search", "query", query, "tenant_id", tenantID)
	result, err = c.repository.SearchByContent(ctx, query, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// If found in repository, store in cache with TTL
	if err := c.redisClient.Set(ctx, key, result, documentCacheTTL); err != nil {
		logger.Error("Failed to cache search results", "error", err, "query", query)
	}

	return result, nil
}

// SearchByMetadata searches documents by metadata with pagination, using cache when available
func (c *DocumentCache) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key using metadata map, tenant ID, and pagination parameters
	var metadataKey string
	for k, v := range metadata {
		metadataKey += fmt.Sprintf("%s:%s,", k, v)
	}
	key := fmt.Sprintf("%s%s:tenant:%s:page:%d:pageSize:%d", documentSearchKeyPrefix, metadataKey, tenantID, pagination.Page, pagination.PageSize)

	// Try to get search results from cache
	var result utils.PaginatedResult[models.Document]
	exist, err := c.redisClient.Get(ctx, key, &result)
	if err != nil {
		logger.Error("Error getting metadata search results from cache", "error", err, "metadata", metadataKey)
	}

	// If found in cache, return the paginated result
	if exist {
		logger.Debug("Cache hit for metadata search", "metadata", metadataKey, "tenant_id", tenantID)
		return result, nil
	}

	// If not in cache, get from repository
	logger.Debug("Cache miss for metadata search", "metadata", metadataKey, "tenant_id", tenantID)
	result, err = c.repository.SearchByMetadata(ctx, metadata, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// If found in repository, store in cache with TTL
	if err := c.redisClient.Set(ctx, key, result, documentCacheTTL); err != nil {
		logger.Error("Failed to cache metadata search results", "error", err, "metadata", metadataKey)
	}

	return result, nil
}

// AddVersion adds a new version to a document and invalidates related cache entries
func (c *DocumentCache) AddVersion(ctx context.Context, version *models.DocumentVersion) (string, error) {
	// Delegate version creation to the underlying repository
	id, err := c.repository.AddVersion(ctx, version)
	if err != nil {
		return "", err
	}

	// If successful, invalidate document cache
	if err := c.invalidateDocumentCache(ctx, version.DocumentID, ""); err != nil {
		logger.Error("Failed to invalidate document cache", "error", err, "document_id", version.DocumentID)
	}

	return id, nil
}

// GetVersionByID retrieves a document version by ID, using cache when available
func (c *DocumentCache) GetVersionByID(ctx context.Context, versionID string, tenantID string) (*models.DocumentVersion, error) {
	// Generate cache key using version ID and tenant ID
	key := c.generateVersionKey(versionID, tenantID)

	// Try to get version from cache
	var version *models.DocumentVersion
	exist, err := c.redisClient.Get(ctx, key, &version)
	if err != nil {
		logger.Error("Error getting version from cache", "error", err, "version_id", versionID)
	}

	// If found in cache, return the version
	if exist && version != nil {
		logger.Debug("Cache hit for document version", "version_id", versionID, "tenant_id", tenantID)
		return version, nil
	}

	// If not in cache, get from repository
	logger.Debug("Cache miss for document version", "version_id", versionID, "tenant_id", tenantID)
	version, err = c.repository.GetVersionByID(ctx, versionID, tenantID)
	if err != nil {
		return nil, err
	}

	// If found in repository, store in cache with TTL
	if version != nil {
		if err := c.redisClient.Set(ctx, key, version, documentCacheTTL); err != nil {
			logger.Error("Failed to cache document version", "error", err, "version_id", versionID)
		}
	}

	return version, nil
}

// UpdateVersionStatus updates a document version status and invalidates related cache entries
func (c *DocumentCache) UpdateVersionStatus(ctx context.Context, versionID string, status string, tenantID string) error {
	// Delegate version status update to the underlying repository
	if err := c.repository.UpdateVersionStatus(ctx, versionID, status, tenantID); err != nil {
		return err
	}

	// If successful, invalidate version cache
	if err := c.invalidateVersionCache(ctx, versionID, tenantID); err != nil {
		logger.Error("Failed to invalidate version cache", "error", err, "version_id", versionID)
	}

	// Get the version to find the document ID
	version, err := c.repository.GetVersionByID(ctx, versionID, tenantID)
	if err == nil && version != nil {
		// Invalidate document cache
		if err := c.invalidateDocumentCache(ctx, version.DocumentID, tenantID); err != nil {
			logger.Error("Failed to invalidate document cache", "error", err, "document_id", version.DocumentID)
		}
	}

	return nil
}

// AddMetadata adds metadata to a document and invalidates related cache entries
func (c *DocumentCache) AddMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) (string, error) {
	// Delegate metadata creation to the underlying repository
	id, err := c.repository.AddMetadata(ctx, documentID, key, value, tenantID)
	if err != nil {
		return "", err
	}

	// If successful, invalidate document cache
	if err := c.invalidateDocumentCache(ctx, documentID, tenantID); err != nil {
		logger.Error("Failed to invalidate document cache", "error", err, "document_id", documentID)
	}

	// Invalidate search cache
	if err := c.invalidateSearchCache(ctx, tenantID); err != nil {
		logger.Error("Failed to invalidate search cache", "error", err, "tenant_id", tenantID)
	}

	return id, nil
}

// UpdateMetadata updates document metadata and invalidates related cache entries
func (c *DocumentCache) UpdateMetadata(ctx context.Context, documentID string, key string, value string, tenantID string) error {
	// Delegate metadata update to the underlying repository
	if err := c.repository.UpdateMetadata(ctx, documentID, key, value, tenantID); err != nil {
		return err
	}

	// If successful, invalidate document cache
	if err := c.invalidateDocumentCache(ctx, documentID, tenantID); err != nil {
		logger.Error("Failed to invalidate document cache", "error", err, "document_id", documentID)
	}

	// Invalidate search cache
	if err := c.invalidateSearchCache(ctx, tenantID); err != nil {
		logger.Error("Failed to invalidate search cache", "error", err, "tenant_id", tenantID)
	}

	return nil
}

// DeleteMetadata deletes document metadata and invalidates related cache entries
func (c *DocumentCache) DeleteMetadata(ctx context.Context, documentID string, key string, tenantID string) error {
	// Delegate metadata deletion to the underlying repository
	if err := c.repository.DeleteMetadata(ctx, documentID, key, tenantID); err != nil {
		return err
	}

	// If successful, invalidate document cache
	if err := c.invalidateDocumentCache(ctx, documentID, tenantID); err != nil {
		logger.Error("Failed to invalidate document cache", "error", err, "document_id", documentID)
	}

	// Invalidate search cache
	if err := c.invalidateSearchCache(ctx, tenantID); err != nil {
		logger.Error("Failed to invalidate search cache", "error", err, "tenant_id", tenantID)
	}

	return nil
}

// GetDocumentsByIDs retrieves multiple documents by their IDs, using cache when available
func (c *DocumentCache) GetDocumentsByIDs(ctx context.Context, ids []string, tenantID string) ([]*models.Document, error) {
	// Initialize result slice
	result := make([]*models.Document, 0, len(ids))
	missingIDs := make([]string, 0)

	// Check cache for each document ID
	for _, id := range ids {
		key := c.generateDocumentKey(id, tenantID)
		
		var document *models.Document
		exist, err := c.redisClient.Get(ctx, key, &document)
		if err != nil {
			logger.Error("Error getting document from cache", "error", err, "id", id)
		}

		if exist && document != nil {
			logger.Debug("Cache hit for document in batch", "id", id, "tenant_id", tenantID)
			result = append(result, document)
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	// If any documents not found in cache, get them from repository
	if len(missingIDs) > 0 {
		logger.Debug("Cache miss for documents in batch", "count", len(missingIDs), "tenant_id", tenantID)
		docs, err := c.repository.GetDocumentsByIDs(ctx, missingIDs, tenantID)
		if err != nil {
			return nil, err
		}

		// Store documents from repository in cache
		for _, doc := range docs {
			key := c.generateDocumentKey(doc.ID, tenantID)
			if err := c.redisClient.Set(ctx, key, doc, documentCacheTTL); err != nil {
				logger.Error("Failed to cache document", "error", err, "id", doc.ID)
			}
			result = append(result, doc)
		}
	}

	return result, nil
}

// generateDocumentKey generates a cache key for a document
func (c *DocumentCache) generateDocumentKey(id string, tenantID string) string {
	return fmt.Sprintf("%s%s:tenant:%s", documentKeyPrefix, id, tenantID)
}

// generateVersionKey generates a cache key for a document version
func (c *DocumentCache) generateVersionKey(versionID string, tenantID string) string {
	return fmt.Sprintf("%s%s:tenant:%s", documentVersionKeyPrefix, versionID, tenantID)
}

// generateListKey generates a cache key for a document list
func (c *DocumentCache) generateListKey(folderID string, tenantID string, pagination *utils.Pagination) string {
	return fmt.Sprintf("%sfolder:%s:tenant:%s:page:%d:pageSize:%d", 
		documentListKeyPrefix, folderID, tenantID, pagination.Page, pagination.PageSize)
}

// generateTenantListKey generates a cache key for a tenant document list
func (c *DocumentCache) generateTenantListKey(tenantID string, pagination *utils.Pagination) string {
	return fmt.Sprintf("%stenant:%s:page:%d:pageSize:%d", 
		documentListKeyPrefix, tenantID, pagination.Page, pagination.PageSize)
}

// generateSearchKey generates a cache key for search results
func (c *DocumentCache) generateSearchKey(query string, tenantID string, pagination *utils.Pagination) string {
	return fmt.Sprintf("%squery:%s:tenant:%s:page:%d:pageSize:%d", 
		documentSearchKeyPrefix, query, tenantID, pagination.Page, pagination.PageSize)
}

// invalidateDocumentCache invalidates cache entries for a document
func (c *DocumentCache) invalidateDocumentCache(ctx context.Context, documentID string, tenantID string) error {
	key := c.generateDocumentKey(documentID, tenantID)
	err := c.redisClient.Del(ctx, key)
	if err != nil {
		return err
	}
	logger.Debug("Invalidated document cache", "document_id", documentID, "tenant_id", tenantID)
	return nil
}

// invalidateVersionCache invalidates cache entries for a document version
func (c *DocumentCache) invalidateVersionCache(ctx context.Context, versionID string, tenantID string) error {
	key := c.generateVersionKey(versionID, tenantID)
	err := c.redisClient.Del(ctx, key)
	if err != nil {
		return err
	}
	logger.Debug("Invalidated version cache", "version_id", versionID, "tenant_id", tenantID)
	return nil
}

// invalidateFolderListCache invalidates cache entries for a folder's document list
func (c *DocumentCache) invalidateFolderListCache(ctx context.Context, folderID string, tenantID string) error {
	pattern := fmt.Sprintf("%sfolder:%s:tenant:%s:*", documentListKeyPrefix, folderID, tenantID)
	err := c.redisClient.DelByPattern(ctx, pattern)
	if err != nil {
		return err
	}
	logger.Debug("Invalidated folder list cache", "folder_id", folderID, "tenant_id", tenantID)
	return nil
}

// invalidateSearchCache invalidates all search cache entries for a tenant
func (c *DocumentCache) invalidateSearchCache(ctx context.Context, tenantID string) error {
	pattern := fmt.Sprintf("%s*tenant:%s*", documentSearchKeyPrefix, tenantID)
	err := c.redisClient.DelByPattern(ctx, pattern)
	if err != nil {
		return err
	}
	logger.Debug("Invalidated search cache", "tenant_id", tenantID)
	return nil
}