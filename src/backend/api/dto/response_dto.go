// Package dto provides Data Transfer Objects for the Document Management Platform API.
// This file contains standard response structures used across all API endpoints to ensure
// consistent response formats. It defines base response types, success responses, and
// pagination wrappers that are used by all feature-specific DTOs.
package dto

import (
	"time" // standard library

	"../../pkg/utils/pagination" // For pagination information in paginated responses
	"../../pkg/utils/time_utils" // For formatting timestamps in response DTOs
)

// Response is the base response structure for all API responses.
type Response struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
}

// DataResponse is a response structure for endpoints that return data.
type DataResponse struct {
	Success   bool        `json:"success"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// PaginatedResponse is a response structure for endpoints that return paginated data.
type PaginatedResponse struct {
	Success    bool               `json:"success"`
	Timestamp  string             `json:"timestamp"`
	Items      interface{}        `json:"items"`
	Pagination pagination.PageInfo `json:"pagination"`
}

// MessageResponse is a response structure for endpoints that return a message.
type MessageResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// CountResponse is a response structure for endpoints that return a count.
type CountResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	Count     int64  `json:"count"`
}

// IDResponse is a response structure for endpoints that return an ID.
type IDResponse struct {
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	ID        string `json:"id"`
}

// NewResponse creates a new Response with success status.
func NewResponse() Response {
	return Response{
		Success:   true,
		Timestamp: time_utils.FormatTime(time.Now(), ""),
	}
}

// NewDataResponse creates a new DataResponse with the given data.
func NewDataResponse(data interface{}) DataResponse {
	return DataResponse{
		Success:   true,
		Timestamp: time_utils.FormatTime(time.Now(), ""),
		Data:      data,
	}
}

// NewPaginatedResponse creates a new PaginatedResponse with the given items and pagination info.
func NewPaginatedResponse(items interface{}, pageInfo pagination.PageInfo) PaginatedResponse {
	return PaginatedResponse{
		Success:    true,
		Timestamp:  time_utils.FormatTime(time.Now(), ""),
		Items:      items,
		Pagination: pageInfo,
	}
}

// NewMessageResponse creates a new MessageResponse with the given message.
func NewMessageResponse(message string) MessageResponse {
	return MessageResponse{
		Success:   true,
		Timestamp: time_utils.FormatTime(time.Now(), ""),
		Message:   message,
	}
}