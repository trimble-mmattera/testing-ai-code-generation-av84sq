# ADR 0005: Elasticsearch for Search

## Status

Accepted

## Context

The Document Management Platform requires robust search capabilities to enable users to find documents based on content and metadata. The system needs to support full-text search across document content, metadata filtering, and combined searches with tenant isolation. The search functionality must handle the expected scale of 10,000 document uploads daily, provide sub-2-second response times for search queries, and maintain strict tenant boundaries. While document content is stored in AWS S3 and metadata in PostgreSQL, a specialized search solution is needed to provide efficient and feature-rich search capabilities.

## Decision

We will use Elasticsearch (version 8.0+) as the primary search engine for the Document Management Platform. Elasticsearch will index document content extracted from files stored in S3 along with relevant metadata from PostgreSQL. The Search Service microservice will provide an abstraction layer over Elasticsearch, implementing the SearchService interface defined in the domain layer, consistent with our Clean Architecture approach.

## Search Architecture

The search architecture will consist of the following components:

### Cluster Configuration

- **Deployment Model**: Multi-node Elasticsearch cluster with dedicated master, data, and client nodes
- **High Availability**: Minimum of 3 nodes across multiple availability zones
- **Scaling Strategy**: Horizontal scaling of data nodes based on index size and query volume
- **Resource Allocation**: Appropriate CPU, memory, and storage based on document volume and query patterns

### Index Structure

- **Index Naming**: Tenant-specific indices with format `{prefix}-{tenant-id}`
- **Sharding Strategy**: Multiple primary shards per index for parallelization
- **Replication**: At least one replica per shard for high availability
- **Refresh Interval**: Configurable based on indexing vs. search performance requirements

### Document Mapping

- **Content Field**: Full-text field with appropriate analyzers for document content
- **Metadata Fields**: Keyword fields for exact matching on metadata attributes
- **Tenant Field**: Keyword field for tenant isolation
- **Folder Field**: Keyword field for folder-scoped searches
- **Date Fields**: Date type for temporal queries
- **Dynamic Mapping**: Controlled dynamic mapping for flexible metadata

### Integration Points

- **Document Service**: Provides documents for indexing
- **Storage Service**: Provides document content from S3
- **Folder Service**: Provides folder structure for scoped searches
- **Event Service**: Triggers indexing on document changes

## Indexing Strategy

The indexing strategy will ensure efficient and accurate document indexing:

### Content Extraction

- **Text Extraction**: Extract plain text from various document formats (PDF, Office, etc.)
- **Content Processing**: Apply text normalization, stemming, and other linguistic processing
- **Metadata Enrichment**: Combine extracted text with document metadata
- **Format-Specific Handling**: Custom extraction logic for different document types

### Indexing Process

- **Event-Driven**: Index documents in response to upload and update events
- **Bulk Indexing**: Use bulk API for efficient indexing of multiple documents
- **Asynchronous Processing**: Queue indexing tasks to avoid blocking user operations
- **Retry Mechanism**: Handle temporary failures with exponential backoff
- **Validation**: Verify document existence and tenant context before indexing

### Index Maintenance

- **Reindexing Strategy**: Approach for handling mapping changes
- **Index Aliases**: Use aliases for zero-downtime reindexing
- **Index Lifecycle Management**: Policies for index retention and optimization
- **Monitoring**: Track indexing performance and error rates

## Tenant Isolation

Strict tenant isolation will be maintained through multiple mechanisms:

### Index-Level Isolation

Each tenant will have dedicated indices with naming pattern `{prefix}-{tenant-id}`, ensuring complete physical separation of search data between tenants.

### Query-Level Isolation

All search queries will be scoped to the specific tenant's index, preventing cross-tenant data access even if index names are known.

### Application-Level Enforcement

The Search Service will enforce tenant context validation on all operations, preventing cross-tenant access attempts.

### Security Configuration

Elasticsearch security features will be configured to restrict access to indices based on authentication context.

## Query Capabilities

The search implementation will support the following query capabilities:

### Full-Text Search

- **Content Search**: Search within document content with relevance ranking
- **Fuzzy Matching**: Handle typos and spelling variations
- **Phrase Matching**: Support for exact phrase queries
- **Boosting**: Prioritize certain fields or terms in results
- **Highlighting**: Highlight matching terms in results

### Metadata Search

- **Exact Matching**: Precise filtering on metadata attributes
- **Range Queries**: For numeric and date fields
- **Prefix/Wildcard**: Partial matching on text fields
- **Existence Queries**: Filter based on presence/absence of fields

### Combined Search

- **Boolean Queries**: Combine content and metadata criteria
- **Filtering**: Apply metadata filters to content search results
- **Scoring Customization**: Adjust relevance based on multiple factors

### Folder-Scoped Search

- **Folder Filtering**: Limit search to specific folders
- **Hierarchical Navigation**: Support for folder tree traversal
- **Permission-Aware**: Respect folder access permissions

## Performance Optimization

To meet the performance requirements of the system, we will implement several optimization strategies:

### Query Optimization

- **Query Profiling**: Analyze and optimize slow queries
- **Field Selection**: Retrieve only necessary fields
- **Query Caching**: Cache frequent search results
- **Filter Caching**: Cache commonly used filters
- **Scroll API**: Efficient pagination for large result sets

### Index Optimization

- **Field Data Caching**: Optimize for aggregations and sorting
- **Doc Values**: Use for sorting and aggregations on keyword fields
- **Index Compression**: Balance between storage and performance
- **Refresh Interval**: Tune based on indexing vs. search requirements

### Resource Allocation

- **Memory Settings**: Appropriate heap size configuration
- **CPU Allocation**: Sufficient CPU for search and indexing
- **Disk I/O**: High-performance storage for indices
- **Network**: Sufficient bandwidth for cluster communication

### Scaling Strategy

- **Horizontal Scaling**: Add nodes as document volume grows
- **Shard Allocation**: Distribute shards across nodes
- **Hot-Warm Architecture**: Separate nodes for active vs. historical indices
- **Cross-Cluster Search**: For very large deployments

## Consequences

### Positive

- Powerful full-text search capabilities across document content
- High-performance query execution meeting sub-2-second response time requirements
- Flexible query capabilities supporting various search patterns
- Scalable architecture to handle growing document volumes
- Rich relevance ranking and scoring capabilities
- Support for complex queries combining content and metadata criteria
- Efficient tenant isolation through index-level separation
- Robust ecosystem with monitoring, analysis, and management tools
- Mature client libraries for Go integration

### Negative

- Additional infrastructure component to manage and maintain
- Increased operational complexity with cluster management
- Potential consistency lag between primary data stores and search index
- Resource requirements for maintaining search indices
- Learning curve for query optimization and cluster tuning
- Need for careful capacity planning as document volume grows
- Requires monitoring and maintenance for optimal performance

## Implementation

The implementation will follow these guidelines:

### Search Service Interface

Following Clean Architecture principles, we will define interfaces in the domain layer:

```go
type SearchIndexer interface {
    IndexDocument(ctx context.Context, document *models.Document, content []byte) error
    RemoveDocument(ctx context.Context, documentID, tenantID string) error
}

type SearchQueryExecutor interface {
    ExecuteContentSearch(ctx context.Context, query, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
    ExecuteMetadataSearch(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
    ExecuteCombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
    ExecuteFolderSearch(ctx context.Context, folderID, query, tenantID string, pagination *utils.Pagination) ([]string, int64, error)
}

type SearchService interface {
    SearchByContent(ctx context.Context, query, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
    SearchByMetadata(ctx context.Context, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
    CombinedSearch(ctx context.Context, contentQuery string, metadata map[string]string, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
    SearchInFolder(ctx context.Context, folderID, query, tenantID string, pagination *utils.Pagination) (utils.PaginatedResult[models.Document], error)
    IndexDocument(ctx context.Context, documentID, tenantID string, content []byte) error
    RemoveDocumentFromIndex(ctx context.Context, documentID, tenantID string) error
}
```

### Elasticsearch Implementation

The infrastructure layer will provide Elasticsearch implementations of these interfaces:

```go
type ElasticsearchClient struct {
    client *elasticsearch.Client
    logger logger.Logger
}

type DocumentIndex struct {
    client      *ElasticsearchClient
    indexPrefix string
    logger      logger.Logger
}

type elasticsearchIndexer struct {
    documentIndex *DocumentIndex
    logger        logger.Logger
}

type elasticsearchQueryExecutor struct {
    client *ElasticsearchClient
    logger logger.Logger
}
```

### Error Handling

The Elasticsearch implementation will handle various error conditions including:

- Network connectivity issues
- Index not found errors
- Query syntax errors
- Timeout errors
- Resource limitation errors

Errors will be translated to domain-specific errors when appropriate.

### Performance Monitoring

The implementation will include:

- Detailed logging of search operations
- Metrics for query latency and throughput
- Index size and growth monitoring
- Error rate tracking
- Integration with Prometheus and Grafana for visualization

## Alternatives Considered

### PostgreSQL Full-Text Search

Using PostgreSQL's built-in full-text search capabilities. Provides integration with existing metadata storage but lacks the advanced search features, scalability, and performance of a dedicated search engine like Elasticsearch.

### Solr

Another open-source search platform based on Lucene. Comparable to Elasticsearch in many aspects but has a smaller ecosystem in cloud environments and less robust client libraries for Go.

### AWS CloudSearch

Managed search service from AWS. Provides good integration with AWS services but offers less flexibility and fewer advanced features compared to Elasticsearch.

### Algolia

Hosted search API with excellent performance. Provides great developer experience but has higher costs at scale and less control over the underlying infrastructure.

### Custom Inverted Index

Building a custom search solution. Would provide maximum control but would require significant development effort and would likely not match the features and performance of established search platforms.

## References

- Elasticsearch Documentation (https://www.elastic.co/guide/en/elasticsearch/reference/8.0/index.html)
- Go Elasticsearch Client (https://github.com/elastic/go-elasticsearch)
- Technical Specifications Section 2.2.3: Document Search Requirements
- Technical Specifications Section 5.3.2: Data Storage Solution Rationale
- Technical Specifications Section 6.4: Search Service Design
- ADR-0001: Use Clean Architecture
- ADR-0003: S3 for Document Storage
- ADR-0004: PostgreSQL for Metadata