// Package utils provides utility functions for the Document Management Platform.
package utils

import (
	"math" // standard library - For mathematical operations in pagination calculations
	"strconv" // standard library - For string to integer conversions in pagination parameters
)

// Default pagination constants
const (
	// DefaultPage is the default page number when not specified
	DefaultPage = 1
	// DefaultPageSize is the default number of items per page when not specified
	DefaultPageSize = 20
	// MaxPageSize is the maximum allowed page size to prevent excessive resource usage
	MaxPageSize = 100
)

// Pagination represents pagination parameters for requests.
type Pagination struct {
	Page     int
	PageSize int
}

// GetOffset calculates the offset for database queries based on page and page size.
func (p *Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the limit (page size) for database queries.
func (p *Pagination) GetLimit() int {
	return p.PageSize
}

// PageInfo contains pagination metadata for responses.
type PageInfo struct {
	Page        int   `json:"page"`
	PageSize    int   `json:"pageSize"`
	TotalPages  int   `json:"totalPages"`
	TotalItems  int64 `json:"totalItems"`
	HasNext     bool  `json:"hasNext"`
	HasPrevious bool  `json:"hasPrevious"`
}

// PaginatedResult is a generic container for paginated results of any type.
type PaginatedResult[T any] struct {
	Items      []T      `json:"items"`
	Pagination PageInfo `json:"pagination"`
}

// NewPagination creates a new Pagination instance with the specified page and page size.
// It validates the parameters and applies defaults if necessary.
func NewPagination(page, pageSize int) *Pagination {
	// Validate page number (must be >= 1, default to DefaultPage if invalid)
	if page < 1 {
		page = DefaultPage
	}

	// Validate page size (must be >= 1 and <= MaxPageSize, default to DefaultPageSize if invalid)
	if pageSize < 1 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Create and return a new Pagination instance with the validated parameters
	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// ParsePaginationFromStrings parses pagination parameters from string values (typically from query parameters).
func ParsePaginationFromStrings(pageStr, pageSizeStr string) *Pagination {
	// Parse pageStr to integer using strconv.Atoi, default to DefaultPage if parsing fails
	page := DefaultPage
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	// Parse pageSizeStr to integer using strconv.Atoi, default to DefaultPageSize if parsing fails
	pageSize := DefaultPageSize
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil {
			pageSize = ps
		}
	}

	// Call NewPagination with the parsed values to create a validated Pagination instance
	return NewPagination(page, pageSize)
}

// NewPageInfo creates a new PageInfo instance with pagination metadata.
func NewPageInfo(pagination *Pagination, totalItems int64) PageInfo {
	// Calculate total pages based on totalItems and pagination.PageSize
	totalPages := int(math.Ceil(float64(totalItems) / float64(pagination.PageSize)))
	
	// Create a new PageInfo instance with the pagination parameters and calculated values
	return PageInfo{
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
		HasNext:     pagination.Page < totalPages,
		HasPrevious: pagination.Page > 1,
	}
}

// NewPaginatedResult creates a new PaginatedResult instance with items and pagination information.
func NewPaginatedResult[T any](items []T, pagination *Pagination, totalItems int64) PaginatedResult[T] {
	// Create a new PageInfo using NewPageInfo with pagination and totalItems
	pageInfo := NewPageInfo(pagination, totalItems)
	
	// Create a new PaginatedResult with the provided items and the created PageInfo
	return PaginatedResult[T]{
		Items:      items,
		Pagination: pageInfo,
	}
}