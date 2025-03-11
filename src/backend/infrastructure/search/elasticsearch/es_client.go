// Package elasticsearch provides Elasticsearch client implementation for the Document Management Platform.
// It enables searching, indexing, and managing documents in Elasticsearch with tenant isolation.
package elasticsearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8" // v8.0.0+
	"github.com/elastic/go-elasticsearch/v8/esapi" // v8.0.0+
	"github.com/elastic/go-elasticsearch/v8/esutil" // v8.0.0+

	"../../../pkg/config"
	"../../../pkg/errors"
	"../../../pkg/logger"
	"../../../domain/models"
)

// Default index settings for Elasticsearch
var defaultIndexSettings = map[string]interface{}{
	"number_of_shards":   3,
	"number_of_replicas": 1,
	"analysis": map[string]interface{}{
		"analyzer": map[string]interface{}{
			"content_analyzer": map[string]interface{}{
				"type":      "custom",
				"tokenizer": "standard",
				"filter":    []string{"lowercase", "asciifolding", "stop", "snowball"},
			},
		},
	},
}

// Default index mappings for Elasticsearch
var defaultIndexMappings = map[string]interface{}{
	"properties": map[string]interface{}{
		"document_id": map[string]interface{}{
			"type": "keyword",
		},
		"tenant_id": map[string]interface{}{
			"type": "keyword",
		},
		"folder_id": map[string]interface{}{
			"type": "keyword",
		},
		"name": map[string]interface{}{
			"type": "text",
			"fields": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type": "keyword",
				},
			},
		},
		"content": map[string]interface{}{
			"type":     "text",
			"analyzer": "content_analyzer",
		},
		"content_type": map[string]interface{}{
			"type": "keyword",
		},
		"size": map[string]interface{}{
			"type": "long",
		},
		"status": map[string]interface{}{
			"type": "keyword",
		},
		"owner_id": map[string]interface{}{
			"type": "keyword",
		},
		"created_at": map[string]interface{}{
			"type": "date",
		},
		"updated_at": map[string]interface{}{
			"type": "date",
		},
		"metadata": map[string]interface{}{
			"type": "nested",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type": "keyword",
				},
				"value": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type": "keyword",
						},
					},
				},
			},
		},
		"tags": map[string]interface{}{
			"type": "keyword",
		},
	},
}

// Default bulk indexer configuration
var defaultBulkIndexerConfig = esutil.BulkIndexerConfig{
	FlushBytes:    5e+6,  // 5MB
	FlushInterval: 30 * time.Second,
	NumWorkers:    3,
	Timeout:       30 * time.Second,
}

// ElasticsearchClient represents a client for interacting with Elasticsearch
type ElasticsearchClient struct {
	client *elasticsearch.Client
	logger logger.Logger
}

// NewElasticsearchClient creates a new ElasticsearchClient instance with the provided configuration
func NewElasticsearchClient(esConfig config.ElasticsearchConfig) (*ElasticsearchClient, error) {
	if len(esConfig.Addresses) == 0 {
		return nil, errors.NewValidationError("Elasticsearch addresses cannot be empty")
	}

	// Create Elasticsearch client configuration
	cfg := elasticsearch.Config{
		Addresses: esConfig.Addresses,
		Username:  esConfig.Username,
		Password:  esConfig.Password,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Initialize Elasticsearch client
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("Failed to create Elasticsearch client: %s", err.Error()))
	}

	// Verify connection to Elasticsearch
	resp, err := client.Info()
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("Failed to connect to Elasticsearch: %s", err.Error()))
	}
	defer resp.Body.Close()

	if resp.IsError() {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, errors.NewDependencyError(fmt.Sprintf("Elasticsearch info request failed: %s", string(bodyBytes)))
	}

	logger.Info("Connected to Elasticsearch", "addresses", esConfig.Addresses)

	return &ElasticsearchClient{
		client: client,
		logger: logger.WithField("component", "elasticsearch_client"),
	}, nil
}

// Search executes a search query against Elasticsearch
func (c *ElasticsearchClient) Search(ctx context.Context, index string, query map[string]interface{}, from, size int) (map[string]interface{}, error) {
	c.logger.InfoContext(ctx, "Executing Elasticsearch search", "index", index, "from", from, "size", size)

	// Marshal query to JSON
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("Failed to encode search query: %s", err.Error()))
	}

	// Execute search request
	res, err := c.client.Search(
		c.client.Search.WithContext(ctx),
		c.client.Search.WithIndex(index),
		c.client.Search.WithBody(&buf),
		c.client.Search.WithFrom(from),
		c.client.Search.WithSize(size),
	)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("Elasticsearch search request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// Check for errors in the response
	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, errors.NewDependencyError(fmt.Sprintf("Failed to parse error response: %s", err.Error()))
		}
		return nil, errors.NewDependencyError(fmt.Sprintf("Elasticsearch search error: %v", e))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("Failed to parse search response: %s", err.Error()))
	}

	return result, nil
}

// Index indexes a document in Elasticsearch
func (c *ElasticsearchClient) Index(ctx context.Context, index string, id string, document interface{}) error {
	c.logger.InfoContext(ctx, "Indexing document in Elasticsearch", "index", index, "id", id)

	// Marshal document to JSON
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(document); err != nil {
		return errors.NewValidationError(fmt.Sprintf("Failed to encode document: %s", err.Error()))
	}

	// Execute index request
	res, err := c.client.Index(
		index,
		&buf,
		c.client.Index.WithContext(ctx),
		c.client.Index.WithDocumentID(id),
		c.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch index request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// Check for errors in the response
	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return errors.NewDependencyError(fmt.Sprintf("Failed to parse error response: %s", err.Error()))
		}
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch index error: %v", e))
	}

	return nil
}

// Delete deletes a document from Elasticsearch
func (c *ElasticsearchClient) Delete(ctx context.Context, index string, id string) error {
	c.logger.InfoContext(ctx, "Deleting document from Elasticsearch", "index", index, "id", id)

	// Execute delete request
	res, err := c.client.Delete(
		index,
		id,
		c.client.Delete.WithContext(ctx),
		c.client.Delete.WithRefresh("true"),
	)
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch delete request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// Check for errors in the response
	if res.IsError() && res.StatusCode != 404 { // 404 is acceptable as it means the document doesn't exist
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return errors.NewDependencyError(fmt.Sprintf("Failed to parse error response: %s", err.Error()))
		}
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch delete error: %v", e))
	}

	return nil
}

// CreateIndex creates an Elasticsearch index with the specified settings and mappings
func (c *ElasticsearchClient) CreateIndex(ctx context.Context, index string, settings map[string]interface{}, mappings map[string]interface{}) error {
	c.logger.InfoContext(ctx, "Creating Elasticsearch index", "index", index)

	// Check if index already exists
	exists, err := c.IndexExists(ctx, index)
	if err != nil {
		return err
	}
	if exists {
		c.logger.InfoContext(ctx, "Index already exists", "index", index)
		return nil
	}

	// Create index body with settings and mappings
	body := map[string]interface{}{
		"settings": settings,
		"mappings": mappings,
	}

	// Marshal index body to JSON
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return errors.NewValidationError(fmt.Sprintf("Failed to encode index body: %s", err.Error()))
	}

	// Execute create index request
	res, err := c.client.Indices.Create(
		index,
		c.client.Indices.Create.WithContext(ctx),
		c.client.Indices.Create.WithBody(&buf),
	)
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch create index request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// Check for errors in the response
	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return errors.NewDependencyError(fmt.Sprintf("Failed to parse error response: %s", err.Error()))
		}
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch create index error: %v", e))
	}

	return nil
}

// IndexExists checks if an Elasticsearch index exists
func (c *ElasticsearchClient) IndexExists(ctx context.Context, index string) (bool, error) {
	// Execute index exists request
	res, err := c.client.Indices.Exists(
		[]string{index},
		c.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return false, errors.NewDependencyError(fmt.Sprintf("Elasticsearch index exists request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// 200 means the index exists, 404 means it doesn't
	return res.StatusCode == 200, nil
}

// DeleteIndex deletes an Elasticsearch index
func (c *ElasticsearchClient) DeleteIndex(ctx context.Context, index string) error {
	c.logger.InfoContext(ctx, "Deleting Elasticsearch index", "index", index)

	// Execute delete index request
	res, err := c.client.Indices.Delete(
		[]string{index},
		c.client.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch delete index request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// Check for errors in the response (404 is acceptable as it means the index doesn't exist)
	if res.IsError() && res.StatusCode != 404 {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return errors.NewDependencyError(fmt.Sprintf("Failed to parse error response: %s", err.Error()))
		}
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch delete index error: %v", e))
	}

	return nil
}

// Refresh refreshes an Elasticsearch index to make recent changes available for search
func (c *ElasticsearchClient) Refresh(ctx context.Context, index string) error {
	// Execute refresh request
	res, err := c.client.Indices.Refresh(
		c.client.Indices.Refresh.WithContext(ctx),
		c.client.Indices.Refresh.WithIndex(index),
	)
	if err != nil {
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch refresh request failed: %s", err.Error()))
	}
	defer res.Body.Close()

	// Check for errors in the response
	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return errors.NewDependencyError(fmt.Sprintf("Failed to parse error response: %s", err.Error()))
		}
		return errors.NewDependencyError(fmt.Sprintf("Elasticsearch refresh error: %v", e))
	}

	return nil
}

// BuildContentQuery builds a content search query for Elasticsearch
func (c *ElasticsearchClient) BuildContentQuery(query string) map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"content": query,
			},
		},
	}
}

// BuildMetadataQuery builds a metadata search query for Elasticsearch
func (c *ElasticsearchClient) BuildMetadataQuery(metadata map[string]string) map[string]interface{} {
	must := make([]map[string]interface{}, 0, len(metadata))

	for key, value := range metadata {
		must = append(must, map[string]interface{}{
			"nested": map[string]interface{}{
				"path": "metadata",
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"must": []map[string]interface{}{
							{
								"term": map[string]interface{}{
									"metadata.key": key,
								},
							},
							{
								"match": map[string]interface{}{
									"metadata.value": value,
								},
							},
						},
					},
				},
			},
		})
	}

	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
	}
}

// BuildCombinedQuery builds a combined content and metadata search query for Elasticsearch
func (c *ElasticsearchClient) BuildCombinedQuery(contentQuery string, metadata map[string]string) map[string]interface{} {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{},
		},
	}

	boolQuery := query["query"].(map[string]interface{})["bool"].(map[string]interface{})

	// Add content query if provided
	if contentQuery != "" {
		boolQuery["should"] = []map[string]interface{}{
			{
				"match": map[string]interface{}{
					"content": contentQuery,
				},
			},
		}
		boolQuery["minimum_should_match"] = 1
	}

	// Add metadata queries if provided
	if len(metadata) > 0 {
		must := make([]map[string]interface{}, 0, len(metadata))

		for key, value := range metadata {
			must = append(must, map[string]interface{}{
				"nested": map[string]interface{}{
					"path": "metadata",
					"query": map[string]interface{}{
						"bool": map[string]interface{}{
							"must": []map[string]interface{}{
								{
									"term": map[string]interface{}{
										"metadata.key": key,
									},
								},
								{
									"match": map[string]interface{}{
										"metadata.value": value,
									},
								},
							},
						},
					},
				},
			})
		}

		boolQuery["must"] = must
	}

	return query
}

// BuildFolderQuery builds a folder-scoped search query for Elasticsearch
func (c *ElasticsearchClient) BuildFolderQuery(folderID string, query string) map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"folder_id": folderID,
						},
					},
					{
						"match": map[string]interface{}{
							"content": query,
						},
					},
				},
			},
		},
	}
}

// CreateBulkIndexer creates a bulk indexer for efficient document indexing
func (c *ElasticsearchClient) CreateBulkIndexer(config esutil.BulkIndexerConfig) (esutil.BulkIndexer, error) {
	// Apply default configuration values if not provided
	if config.FlushBytes == 0 {
		config.FlushBytes = defaultBulkIndexerConfig.FlushBytes
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = defaultBulkIndexerConfig.FlushInterval
	}
	if config.NumWorkers == 0 {
		config.NumWorkers = defaultBulkIndexerConfig.NumWorkers
	}
	if config.Timeout == 0 {
		config.Timeout = defaultBulkIndexerConfig.Timeout
	}
	config.Client = c.client

	// Create bulk indexer
	bulkIndexer, err := esutil.NewBulkIndexer(config)
	if err != nil {
		return nil, errors.NewDependencyError(fmt.Sprintf("Failed to create bulk indexer: %s", err.Error()))
	}

	return bulkIndexer, nil
}

// DocumentIndex manages document indices in Elasticsearch with tenant isolation
type DocumentIndex struct {
	client      *ElasticsearchClient
	indexPrefix string
	logger      logger.Logger
}

// NewDocumentIndex creates a new DocumentIndex instance with the provided client and configuration
func NewDocumentIndex(client *ElasticsearchClient, esConfig config.ElasticsearchConfig) (*DocumentIndex, error) {
	if client == nil {
		return nil, errors.NewValidationError("Elasticsearch client cannot be nil")
	}

	indexPrefix := esConfig.IndexPrefix
	if indexPrefix == "" {
		indexPrefix = "documents"
	}

	return &DocumentIndex{
		client:      client,
		indexPrefix: indexPrefix,
		logger:      logger.WithField("component", "elasticsearch_document_index"),
	}, nil
}

// GetTenantIndex gets the Elasticsearch index name for a tenant
func (di *DocumentIndex) GetTenantIndex(tenantID string) string {
	return fmt.Sprintf("%s-%s", di.indexPrefix, tenantID)
}

// EnsureTenantIndex ensures that a tenant-specific index exists, creating it if necessary
func (di *DocumentIndex) EnsureTenantIndex(ctx context.Context, tenantID string) (string, error) {
	indexName := di.GetTenantIndex(tenantID)

	// Check if index exists, create if not
	exists, err := di.client.IndexExists(ctx, indexName)
	if err != nil {
		return "", err
	}

	if !exists {
		// Create index with default settings and mappings
		err = di.client.CreateIndex(ctx, indexName, defaultIndexSettings, defaultIndexMappings)
		if err != nil {
			return "", err
		}
		di.logger.InfoContext(ctx, "Created tenant index", "index", indexName, "tenant_id", tenantID)
	}

	return indexName, nil
}

// IndexDocument indexes a document in the tenant-specific index
func (di *DocumentIndex) IndexDocument(ctx context.Context, document *models.Document, content []byte) error {
	di.logger.InfoContext(ctx, "Indexing document", "document_id", document.ID, "tenant_id", document.TenantID)

	if document == nil {
		return errors.NewValidationError("Document cannot be nil")
	}

	if content == nil || len(content) == 0 {
		return errors.NewValidationError("Document content cannot be empty")
	}

	// Ensure tenant index exists
	indexName, err := di.EnsureTenantIndex(ctx, document.TenantID)
	if err != nil {
		return err
	}

	// Extract text from document content
	textContent, err := di.extractText(content, document.ContentType)
	if err != nil {
		di.logger.ErrorContext(ctx, "Failed to extract text from document", "error", err.Error())
		// Continue with empty content rather than failing the indexing
		textContent = ""
	}

	// Create document mapping
	docMapping := map[string]interface{}{
		"document_id":  document.ID,
		"tenant_id":    document.TenantID,
		"folder_id":    document.FolderID,
		"name":         document.Name,
		"content":      textContent,
		"content_type": document.ContentType,
		"size":         document.Size,
		"status":       document.Status,
		"owner_id":     document.OwnerID,
		"created_at":   document.CreatedAt,
		"updated_at":   document.UpdatedAt,
	}

	// Add metadata if available
	if len(document.Metadata) > 0 {
		metadata := make([]map[string]string, len(document.Metadata))
		for i, m := range document.Metadata {
			metadata[i] = map[string]string{
				"key":   m.Key,
				"value": m.Value,
			}
		}
		docMapping["metadata"] = metadata
	}

	// Add tags if available
	if len(document.Tags) > 0 {
		tags := make([]string, len(document.Tags))
		for i, t := range document.Tags {
			tags[i] = t.Name
		}
		docMapping["tags"] = tags
	}

	// Index document
	err = di.client.Index(ctx, indexName, document.ID, docMapping)
	if err != nil {
		return err
	}

	// Refresh index to make document searchable immediately
	err = di.client.Refresh(ctx, indexName)
	if err != nil {
		return err
	}

	di.logger.InfoContext(ctx, "Document indexed successfully", "document_id", document.ID, "index", indexName)
	return nil
}

// RemoveDocument removes a document from the tenant-specific index
func (di *DocumentIndex) RemoveDocument(ctx context.Context, documentID string, tenantID string) error {
	di.logger.InfoContext(ctx, "Removing document", "document_id", documentID, "tenant_id", tenantID)

	if documentID == "" {
		return errors.NewValidationError("Document ID cannot be empty")
	}

	if tenantID == "" {
		return errors.NewValidationError("Tenant ID cannot be empty")
	}

	// Get tenant index
	indexName := di.GetTenantIndex(tenantID)

	// Delete document
	err := di.client.Delete(ctx, indexName, documentID)
	if err != nil {
		return err
	}

	// Refresh index to update search results immediately
	err = di.client.Refresh(ctx, indexName)
	if err != nil {
		return err
	}

	di.logger.InfoContext(ctx, "Document removed successfully", "document_id", documentID, "index", indexName)
	return nil
}

// extractText extracts searchable text from document content
func (di *DocumentIndex) extractText(content []byte, contentType string) (string, error) {
	// For plain text, just return the content as string
	if strings.HasPrefix(contentType, "text/") {
		return string(content), nil
	}

	// For PDF, extract text using a PDF parser
	if contentType == "application/pdf" {
		// Note: This is a simplified implementation
		// In a real implementation, you would use a PDF extraction library
		// such as pdfcpu, unipdf, or integrate with an external service
		return string(content), nil
	}

	// For Office documents, extract text using appropriate parser
	if strings.Contains(contentType, "office") || 
	   strings.Contains(contentType, "word") || 
	   strings.Contains(contentType, "excel") || 
	   strings.Contains(contentType, "powerpoint") {
		// Note: This is a simplified implementation
		// In a real implementation, you would use a document extraction library
		// such as tika-server or integrate with an external service
		return string(content), nil
	}

	// For unsupported document types, return empty string or error
	return "", fmt.Errorf("unsupported content type for text extraction: %s", contentType)
}