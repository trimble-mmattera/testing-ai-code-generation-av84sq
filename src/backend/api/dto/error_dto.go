// Package dto provides Data Transfer Objects for the Document Management Platform API.
// This file contains error response DTOs and utility functions for creating standardized
// error responses. These DTOs ensure consistent error handling and presentation across all
// API endpoints.
package dto

import (
	"net/http" // standard library
	"time"     // standard library

	"../../pkg/errors"
	timeutils "../../pkg/utils/time_utils"
)

// ErrorDetail contains detailed information about an error
type ErrorDetail struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

// ErrorResponse represents a standard error response for API endpoints
type ErrorResponse struct {
	Success   bool        `json:"success"`
	Timestamp string      `json:"timestamp"`
	Error     ErrorDetail `json:"error"`
}

// ValidationErrorResponse represents a validation error response for API endpoints
type ValidationErrorResponse struct {
	Success          bool              `json:"success"`
	Timestamp        string            `json:"timestamp"`
	Error            ErrorDetail       `json:"error"`
	ValidationErrors map[string]string `json:"validation_errors"`
}

// NewErrorResponse creates a new ErrorResponse with the given error
func NewErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.GetErrorType(err),
			Message:    err.Error(),
			StatusCode: errors.GetStatusCode(err),
		},
	}
}

// NewValidationErrorResponse creates a new ValidationErrorResponse with the given validation errors
func NewValidationErrorResponse(err error, validationErrors map[string]string) ValidationErrorResponse {
	return ValidationErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.ErrorTypeValidation,
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
		},
		ValidationErrors: validationErrors,
	}
}

// NewResourceNotFoundErrorResponse creates a new ErrorResponse for resource not found errors
func NewResourceNotFoundErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.ErrorTypeNotFound,
			Message:    err.Error(),
			StatusCode: http.StatusNotFound,
		},
	}
}

// NewAuthorizationErrorResponse creates a new ErrorResponse for authorization errors
func NewAuthorizationErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.ErrorTypeAuthorization,
			Message:    err.Error(),
			StatusCode: http.StatusForbidden,
		},
	}
}

// NewAuthenticationErrorResponse creates a new ErrorResponse for authentication errors
func NewAuthenticationErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.ErrorTypeAuthentication,
			Message:    err.Error(),
			StatusCode: http.StatusUnauthorized,
		},
	}
}

// NewInternalErrorResponse creates a new ErrorResponse for internal server errors
func NewInternalErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.ErrorTypeInternal,
			Message:    "An internal server error occurred",
			StatusCode: http.StatusInternalServerError,
		},
	}
}

// NewDependencyErrorResponse creates a new ErrorResponse for dependency errors
func NewDependencyErrorResponse(err error) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Timestamp: timeutils.FormatTime(time.Now(), ""),
		Error: ErrorDetail{
			Type:       errors.ErrorTypeDependency,
			Message:    err.Error(),
			StatusCode: http.StatusServiceUnavailable,
		},
	}
}