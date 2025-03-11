// Package redis provides Redis-based implementations for caching services.
package redis

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"../../../domain/models"
	"../../../domain/services"
	"../../../pkg/logger"
	"../../../pkg/utils"
)

// Default TTL for search cache entries
const searchCacheTTL = 5 * time.Minute

// Cache key prefixes for different search types
const contentSearchKeyPrefix = "search:content:"
const metadataSearchKeyPrefix = "search:metadata:"
const combinedSearchKeyPrefix = "search:combined:"
const folderSearchKeyPrefix = "search:folder:"

// SearchCache implements the SearchService interface with Redis-based caching.
// It wraps a SearchService and adds caching capabilities to improve performance.
type SearchCache struct {
	redisClient   *RedisClient
	searchService services.SearchService
}

// NewSearchCache creates a new SearchCache instance that wraps a SearchService.
func NewSearchCache(redisClient *RedisClient, searchService services.SearchService) services.SearchService {
	return &SearchCache{
		redisClient:   redisClient,
		searchService: searchService,
	}
}

// SearchByContent searches documents by their content, using cache when available.
func (c *SearchCache) SearchByContent(ctx context.Context, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key
	cacheKey := c.generateContentSearchKey(query, tenantID, pagination)

	// Try to get from cache
	var result utils.PaginatedResult[models.Document]
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err == nil && data != nil {
		// Cache hit
		if err := json.Unmarshal(data, &result); err == nil {
			logger.Debug("Cache hit for content search", "query", query, "tenantID", tenantID)
			return result, nil
		}
		// Error unmarshaling, log and continue with service call
		logger.Error("Failed to unmarshal cached search results", "error", err)
	}

	// Cache miss or error, call the search service
	result, err = c.searchService.SearchByContent(ctx, query, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// Cache the result
	if resultData, err := json.Marshal(result); err == nil {
		if setErr := c.redisClient.SetWithExpiration(ctx, cacheKey, resultData, searchCacheTTL); setErr != nil {
			logger.Error("Failed to cache search results", "error", setErr)
		}
	} else {
		logger.Error("Failed to marshal search results for caching", "error", err)
	}

	return result, nil
}

// SearchByMetadata searches documents by their metadata, using cache when available.
func (c *SearchCache) SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key
	cacheKey := c.generateMetadataSearchKey(metadata, tenantID, pagination)

	// Try to get from cache
	var result utils.PaginatedResult[models.Document]
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err == nil && data != nil {
		// Cache hit
		if err := json.Unmarshal(data, &result); err == nil {
			logger.Debug("Cache hit for metadata search", "metadata", metadata, "tenantID", tenantID)
			return result, nil
		}
		// Error unmarshaling, log and continue with service call
		logger.Error("Failed to unmarshal cached search results", "error", err)
	}

	// Cache miss or error, call the search service
	result, err = c.searchService.SearchByMetadata(ctx, metadata, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// Cache the result
	if resultData, err := json.Marshal(result); err == nil {
		if setErr := c.redisClient.SetWithExpiration(ctx, cacheKey, resultData, searchCacheTTL); setErr != nil {
			logger.Error("Failed to cache search results", "error", setErr)
		}
	} else {
		logger.Error("Failed to marshal search results for caching", "error", err)
	}

	return result, nil
}

// CombinedSearch performs a search using both content and metadata criteria, using cache when available.
func (c *SearchCache) CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key
	cacheKey := c.generateCombinedSearchKey(contentQuery, metadata, tenantID, pagination)

	// Try to get from cache
	var result utils.PaginatedResult[models.Document]
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err == nil && data != nil {
		// Cache hit
		if err := json.Unmarshal(data, &result); err == nil {
			logger.Debug("Cache hit for combined search", "contentQuery", contentQuery, "metadata", metadata, "tenantID", tenantID)
			return result, nil
		}
		// Error unmarshaling, log and continue with service call
		logger.Error("Failed to unmarshal cached search results", "error", err)
	}

	// Cache miss or error, call the search service
	result, err = c.searchService.CombinedSearch(ctx, contentQuery, metadata, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// Cache the result
	if resultData, err := json.Marshal(result); err == nil {
		if setErr := c.redisClient.SetWithExpiration(ctx, cacheKey, resultData, searchCacheTTL); setErr != nil {
			logger.Error("Failed to cache search results", "error", setErr)
		}
	} else {
		logger.Error("Failed to marshal search results for caching", "error", err)
	}

	return result, nil
}

// SearchInFolder searches documents within a specific folder, using cache when available.
func (c *SearchCache) SearchInFolder(ctx context.Context, folderID string, query string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error) {
	// Generate cache key
	cacheKey := c.generateFolderSearchKey(folderID, query, tenantID, pagination)

	// Try to get from cache
	var result utils.PaginatedResult[models.Document]
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err == nil && data != nil {
		// Cache hit
		if err := json.Unmarshal(data, &result); err == nil {
			logger.Debug("Cache hit for folder search", "folderID", folderID, "query", query, "tenantID", tenantID)
			return result, nil
		}
		// Error unmarshaling, log and continue with service call
		logger.Error("Failed to unmarshal cached search results", "error", err)
	}

	// Cache miss or error, call the search service
	result, err = c.searchService.SearchInFolder(ctx, folderID, query, tenantID, pagination)
	if err != nil {
		return utils.PaginatedResult[models.Document]{}, err
	}

	// Cache the result
	if resultData, err := json.Marshal(result); err == nil {
		if setErr := c.redisClient.SetWithExpiration(ctx, cacheKey, resultData, searchCacheTTL); setErr != nil {
			logger.Error("Failed to cache search results", "error", setErr)
		}
	} else {
		logger.Error("Failed to marshal search results for caching", "error", err)
	}

	return result, nil
}

// IndexDocument indexes a document for search and invalidates related cache entries.
func (c *SearchCache) IndexDocument(ctx context.Context, documentID string, tenantID string, content []byte) error {
	// Call the underlying service
	err := c.searchService.IndexDocument(ctx, documentID, tenantID, content)
	if err != nil {
		return err
	}

	// Invalidate cache for the tenant to maintain consistency
	if invalidateErr := c.invalidateSearchCache(ctx, tenantID); invalidateErr != nil {
		logger.Error("Failed to invalidate search cache after indexing", "error", invalidateErr, "documentID", documentID, "tenantID", tenantID)
	}

	return nil
}

// RemoveDocumentFromIndex removes a document from the search index and invalidates related cache entries.
func (c *SearchCache) RemoveDocumentFromIndex(ctx context.Context, documentID string, tenantID string) error {
	// Call the underlying service
	err := c.searchService.RemoveDocumentFromIndex(ctx, documentID, tenantID)
	if err != nil {
		return err
	}

	// Invalidate cache for the tenant to maintain consistency
	if invalidateErr := c.invalidateSearchCache(ctx, tenantID); invalidateErr != nil {
		logger.Error("Failed to invalidate search cache after removing document", "error", invalidateErr, "documentID", documentID, "tenantID", tenantID)
	}

	return nil
}

// generateContentSearchKey generates a cache key for content search results.
func (c *SearchCache) generateContentSearchKey(query string, tenantID string, pagination *utils.Pagination) string {
	return fmt.Sprintf("%s%s:%s:p%d:s%d", contentSearchKeyPrefix, tenantID, query, pagination.Page, pagination.PageSize)
}

// generateMetadataSearchKey generates a cache key for metadata search results.
func (c *SearchCache) generateMetadataSearchKey(metadata map[string]string, tenantID string, pagination *utils.Pagination) string {
	metadataHash := c.hashMetadata(metadata)
	return fmt.Sprintf("%s%s:%s:p%d:s%d", metadataSearchKeyPrefix, tenantID, metadataHash, pagination.Page, pagination.PageSize)
}

// generateCombinedSearchKey generates a cache key for combined search results.
func (c *SearchCache) generateCombinedSearchKey(contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) string {
	metadataHash := c.hashMetadata(metadata)
	return fmt.Sprintf("%s%s:%s:%s:p%d:s%d", combinedSearchKeyPrefix, tenantID, contentQuery, metadataHash, pagination.Page, pagination.PageSize)
}

// generateFolderSearchKey generates a cache key for folder search results.
func (c *SearchCache) generateFolderSearchKey(folderID string, query string, tenantID string, pagination *utils.Pagination) string {
	return fmt.Sprintf("%s%s:%s:%s:p%d:s%d", folderSearchKeyPrefix, tenantID, folderID, query, pagination.Page, pagination.PageSize)
}

// invalidateSearchCache invalidates all search cache entries for a tenant.
func (c *SearchCache) invalidateSearchCache(ctx context.Context, tenantID string) error {
	// Generate patterns for all search cache key types
	patterns := []string{
		fmt.Sprintf("%s%s:*", contentSearchKeyPrefix, tenantID),
		fmt.Sprintf("%s%s:*", metadataSearchKeyPrefix, tenantID),
		fmt.Sprintf("%s%s:*", combinedSearchKeyPrefix, tenantID),
		fmt.Sprintf("%s%s:*", folderSearchKeyPrefix, tenantID),
	}

	// Delete all matching keys
	var err error
	for _, pattern := range patterns {
		if delErr := c.redisClient.DeleteByPattern(ctx, pattern); delErr != nil {
			logger.Error("Failed to delete cache keys", "error", delErr, "pattern", pattern)
			err = delErr
		}
	}

	if err == nil {
		logger.Debug("Invalidated search cache", "tenantID", tenantID)
	}

	return err
}

// hashMetadata creates a hash of the metadata map for consistent cache keys.
func (c *SearchCache) hashMetadata(metadata map[string]string) string {
	if metadata == nil || len(metadata) == 0 {
		return "empty"
	}

	// Sort keys for consistent ordering
	keys := make([]string, 0, len(metadata))
	for k := range metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create a string representation of the metadata
	var str string
	for _, k := range keys {
		str += k + ":" + metadata[k] + ","
	}

	// Generate MD5 hash
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}