// Package validators provides validation functions for webhook-related API requests.
// This file contains validators for webhook creation, update, and management to ensure 
// data integrity and proper error handling.
package validators

import (
	"fmt"    // standard library
	"net/url" // standard library
	"strings" // standard library

	"../dto" // v1.0.0+
	"../../pkg/errors" // v1.0.0+
	"../../pkg/validator" // v1.0.0+
	"../../domain/models" // v1.0.0+
)

// Constants for validation limits
const (
	MaxWebhookURLLength         = 2048
	MaxWebhookDescriptionLength = 1024
	MaxEventTypesCount          = 20
	MaxSecretKeyLength          = 256
)

// ValidWebhookStatuses contains all valid webhook statuses
var ValidWebhookStatuses = []string{models.WebhookStatusActive, models.WebhookStatusInactive}

// ValidateCreateWebhookRequest validates a webhook creation request
func ValidateCreateWebhookRequest(request *dto.CreateWebhookRequest) error {
	if request == nil {
		return errors.NewValidationError("webhook creation request cannot be nil")
	}

	// Validate the request struct
	if err := validator.Validate(request); err != nil {
		return err
	}

	// Validate URL
	if err := validateWebhookURL(request.URL); err != nil {
		return err
	}

	// Validate event types
	if err := validateEventTypes(request.EventTypes); err != nil {
		return err
	}

	// Validate description length if provided
	if request.Description != "" {
		if err := validator.ValidateMaxLength(request.Description, MaxWebhookDescriptionLength, "description"); err != nil {
			return err
		}
	}

	// Validate secret key length if provided
	if request.SecretKey != "" {
		if err := validator.ValidateMaxLength(request.SecretKey, MaxSecretKeyLength, "secret_key"); err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdateWebhookRequest validates a webhook update request
func ValidateUpdateWebhookRequest(request *dto.UpdateWebhookRequest) error {
	if request == nil {
		return errors.NewValidationError("webhook update request cannot be nil")
	}

	// Check if at least one field is provided for update
	if request.URL == "" && 
	   request.EventTypes == nil && 
	   request.Description == nil && 
	   request.Status == "" && 
	   request.SecretKey == "" {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	// Validate URL if provided
	if request.URL != "" {
		if err := validateWebhookURL(request.URL); err != nil {
			return err
		}
	}

	// Validate event types if provided
	if request.EventTypes != nil && len(request.EventTypes) > 0 {
		if err := validateEventTypes(request.EventTypes); err != nil {
			return err
		}
	}

	// Validate description if provided
	if request.Description != nil {
		if err := validator.ValidateMaxLength(*request.Description, MaxWebhookDescriptionLength, "description"); err != nil {
			return err
		}
	}

	// Validate status if provided
	if request.Status != "" {
		found := false
		for _, status := range ValidWebhookStatuses {
			if request.Status == status {
				found = true
				break
			}
		}
		if !found {
			return errors.NewValidationError(fmt.Sprintf("status '%s' is not valid, must be one of: %s", 
				request.Status, strings.Join(ValidWebhookStatuses, ", ")))
		}
	}

	// Validate secret key if provided
	if request.SecretKey != "" {
		if err := validator.ValidateMaxLength(request.SecretKey, MaxSecretKeyLength, "secret_key"); err != nil {
			return err
		}
	}

	return nil
}

// validateWebhookURL validates a webhook URL format and length
func validateWebhookURL(urlStr string) error {
	if err := validator.ValidateRequired(urlStr, "url"); err != nil {
		return err
	}

	if err := validator.ValidateMaxLength(urlStr, MaxWebhookURLLength, "url"); err != nil {
		return err
	}

	// Parse URL to validate format
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.NewValidationError(fmt.Sprintf("invalid URL format: %s", err.Error()))
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.NewValidationError("URL scheme must be http or https")
	}

	return nil
}

// validateEventTypes validates webhook event types
func validateEventTypes(eventTypes []string) error {
	if err := validator.ValidateRequired(eventTypes, "event_types"); err != nil {
		return err
	}

	if len(eventTypes) > MaxEventTypesCount {
		return errors.NewValidationError(fmt.Sprintf("maximum of %d event types allowed", MaxEventTypesCount))
	}

	// Validate each event type
	for _, eventType := range eventTypes {
		isValid := false
		for _, supportedType := range dto.SupportedEventTypes {
			if eventType == supportedType {
				isValid = true
				break
			}
		}

		if !isValid {
			return errors.NewValidationError(fmt.Sprintf("event_type '%s' is not supported, must be one of: %s", 
				eventType, strings.Join(dto.SupportedEventTypes, ", ")))
		}
	}

	return nil
}