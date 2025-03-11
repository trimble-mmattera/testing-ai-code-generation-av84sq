// Package errors provides a standardized error handling system for the Document Management Platform.
// It implements custom error types with additional context such as error type, status code,
// and wrapped cause. This package enables consistent error handling, proper HTTP status code
// mapping, and detailed error information throughout the application.
package errors

import (
	"errors" // standard library
	"fmt"    // standard library
	"net/http" // standard library
)

// Error type constants for categorizing different errors
const (
	ErrorTypeValidation    = "validation"
	ErrorTypeNotFound      = "not_found"
	ErrorTypeAuthorization = "authorization"
	ErrorTypeAuthentication = "authentication"
	ErrorTypeSecurity      = "security"
	ErrorTypeInternal      = "internal"
	ErrorTypeDependency    = "dependency"
)

// AppError is a custom error type that provides additional context for application errors
// including error type, HTTP status code, message, and original cause.
type AppError struct {
	errorType  string
	statusCode int
	message    string
	cause      error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

// Unwrap implements the unwrap interface for error chains.
func (e *AppError) Unwrap() error {
	return e.cause
}

// WithMessage sets the error message and returns the AppError for chaining.
func (e *AppError) WithMessage(message string) *AppError {
	e.message = message
	return e
}

// WithStatusCode sets the HTTP status code and returns the AppError for chaining.
func (e *AppError) WithStatusCode(statusCode int) *AppError {
	e.statusCode = statusCode
	return e
}

// WithCause sets the cause of the error and returns the AppError for chaining.
func (e *AppError) WithCause(cause error) *AppError {
	e.cause = cause
	return e
}

// Type gets the error type.
func (e *AppError) Type() string {
	return e.errorType
}

// StatusCode gets the HTTP status code.
func (e *AppError) StatusCode() int {
	return e.statusCode
}

// Cause gets the cause of the error.
func (e *AppError) Cause() error {
	return e.cause
}

// NewValidationError creates a new validation error with the given message.
func NewValidationError(message string) error {
	return &AppError{
		errorType:  ErrorTypeValidation,
		statusCode: http.StatusBadRequest,
		message:    message,
	}
}

// NewResourceNotFoundError creates a new resource not found error with the given message.
func NewResourceNotFoundError(message string) error {
	return &AppError{
		errorType:  ErrorTypeNotFound,
		statusCode: http.StatusNotFound,
		message:    message,
	}
}

// NewAuthorizationError creates a new authorization error with the given message.
func NewAuthorizationError(message string) error {
	return &AppError{
		errorType:  ErrorTypeAuthorization,
		statusCode: http.StatusForbidden,
		message:    message,
	}
}

// NewAuthenticationError creates a new authentication error with the given message.
func NewAuthenticationError(message string) error {
	return &AppError{
		errorType:  ErrorTypeAuthentication,
		statusCode: http.StatusUnauthorized,
		message:    message,
	}
}

// NewSecurityError creates a new security error with the given message.
func NewSecurityError(message string) error {
	return &AppError{
		errorType:  ErrorTypeSecurity,
		statusCode: http.StatusForbidden,
		message:    message,
	}
}

// NewInternalError creates a new internal error with the given message.
func NewInternalError(message string) error {
	return &AppError{
		errorType:  ErrorTypeInternal,
		statusCode: http.StatusInternalServerError,
		message:    message,
	}
}

// NewDependencyError creates a new dependency error with the given message.
func NewDependencyError(message string) error {
	return &AppError{
		errorType:  ErrorTypeDependency,
		statusCode: http.StatusServiceUnavailable,
		message:    message,
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		// If it's already an AppError, create a new one with the same type and status code
		return &AppError{
			errorType:  appErr.errorType,
			statusCode: appErr.statusCode,
			message:    message,
			cause:      err,
		}
	}

	// For other errors, wrap them as internal errors
	return &AppError{
		errorType:  ErrorTypeInternal,
		statusCode: http.StatusInternalServerError,
		message:    message,
		cause:      err,
	}
}

// GetErrorType extracts the error type from an error if it's an AppError.
func GetErrorType(err error) string {
	if err == nil {
		return ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.errorType
	}

	return ""
}

// GetStatusCode extracts the HTTP status code from an error if it's an AppError.
func GetStatusCode(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.statusCode
	}

	return http.StatusInternalServerError
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	return GetErrorType(err) == ErrorTypeValidation
}

// IsResourceNotFoundError checks if an error is a resource not found error.
func IsResourceNotFoundError(err error) bool {
	return GetErrorType(err) == ErrorTypeNotFound
}

// IsAuthorizationError checks if an error is an authorization error.
func IsAuthorizationError(err error) bool {
	return GetErrorType(err) == ErrorTypeAuthorization
}

// IsAuthenticationError checks if an error is an authentication error.
func IsAuthenticationError(err error) bool {
	return GetErrorType(err) == ErrorTypeAuthentication
}

// IsSecurityError checks if an error is a security error.
func IsSecurityError(err error) bool {
	return GetErrorType(err) == ErrorTypeSecurity
}

// IsDependencyError checks if an error is a dependency error.
func IsDependencyError(err error) bool {
	return GetErrorType(err) == ErrorTypeDependency
}