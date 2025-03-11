// Package validator provides a standardized validation framework for the Document Management Platform.
// It offers utilities for validating structs and individual fields using struct tags and custom validation
// rules, ensuring data integrity across the application.
package validator

import (
	"fmt"    // standard library
	"reflect" // standard library
	"regexp"  // standard library
	"strings" // standard library

	"github.com/go-playground/validator/v10" // v10.11.0+

	"../errors" // For creating standardized validation errors
)

var (
	// validate is a singleton instance of the validator
	validate = validator.New()

	// emailRegex is a compiled regular expression for validating email addresses
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	// uuidRegex is a compiled regular expression for validating UUID strings
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// Validate validates a struct using struct tags and returns validation errors.
// It supports all the validation tags provided by github.com/go-playground/validator/v10.
func Validate(s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	
	if val.Kind() != reflect.Struct {
		return errors.NewValidationError("validation requires a struct or a pointer to a struct")
	}

	err := validate.Struct(s)
	if err != nil {
		return errors.NewValidationError(formatValidationErrors(err))
	}
	
	return nil
}

// ValidateField validates a single field against a specific validation tag.
// For example, ValidateField("test@example.com", "email") or ValidateField(42, "gte=0,lte=100").
func ValidateField(field interface{}, tag string) error {
	err := validate.Var(field, tag)
	if err != nil {
		return errors.NewValidationError(formatValidationErrors(err))
	}
	
	return nil
}

// ValidateEmail validates an email address format.
// Returns a validation error if the email is not valid.
func ValidateEmail(email string) error {
	if email == "" {
		return errors.NewValidationError("email address is required")
	}
	
	if !emailRegex.MatchString(email) {
		return errors.NewValidationError("invalid email address format")
	}
	
	return nil
}

// ValidateUUID validates a UUID string format.
// Returns a validation error if the UUID is not valid.
func ValidateUUID(uuid string) error {
	if uuid == "" {
		return errors.NewValidationError("UUID is required")
	}
	
	if !uuidRegex.MatchString(uuid) {
		return errors.NewValidationError("invalid UUID format")
	}
	
	return nil
}

// ValidateRequired validates that a value is not empty.
// Returns a validation error if the value is empty.
func ValidateRequired(value interface{}, fieldName string) error {
	val := reflect.ValueOf(value)
	
	// Check if value is nil or zero value
	isZero := false
	
	switch val.Kind() {
	case reflect.Ptr, reflect.Interface:
		isZero = val.IsNil()
	case reflect.String:
		isZero = val.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		isZero = val.Len() == 0
	default:
		isZero = val.IsZero()
	}
	
	if isZero {
		return errors.NewValidationError(fmt.Sprintf("%s is required", fieldName))
	}
	
	return nil
}

// ValidateMinLength validates that a string has at least the minimum length.
// Returns a validation error if the string is too short.
func ValidateMinLength(value string, minLength int, fieldName string) error {
	if len(value) < minLength {
		return errors.NewValidationError(fmt.Sprintf("%s must be at least %d characters long", fieldName, minLength))
	}
	
	return nil
}

// ValidateMaxLength validates that a string does not exceed the maximum length.
// Returns a validation error if the string is too long.
func ValidateMaxLength(value string, maxLength int, fieldName string) error {
	if len(value) > maxLength {
		return errors.NewValidationError(fmt.Sprintf("%s must not exceed %d characters", fieldName, maxLength))
	}
	
	return nil
}

// ValidateRange validates that a numeric value is within a specified range.
// Returns a validation error if the value is out of range.
func ValidateRange(value interface{}, min, max float64, fieldName string) error {
	val := reflect.ValueOf(value)
	var floatVal float64
	
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		floatVal = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		floatVal = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		floatVal = val.Float()
	default:
		return errors.NewValidationError(fmt.Sprintf("%s must be a numeric value", fieldName))
	}
	
	if floatVal < min || floatVal > max {
		return errors.NewValidationError(fmt.Sprintf("%s must be between %v and %v", fieldName, min, max))
	}
	
	return nil
}

// ValidateOneOf validates that a value is one of the allowed values.
// Returns a validation error if the value is not in the allowed values.
func ValidateOneOf(value interface{}, allowedValues []interface{}, fieldName string) error {
	for _, allowed := range allowedValues {
		if reflect.DeepEqual(value, allowed) {
			return nil
		}
	}
	
	// Format allowed values for error message
	allowedStrs := make([]string, len(allowedValues))
	for i, v := range allowedValues {
		allowedStrs[i] = fmt.Sprintf("%v", v)
	}
	
	return errors.NewValidationError(fmt.Sprintf("%s must be one of: %s", fieldName, strings.Join(allowedStrs, ", ")))
}

// formatValidationErrors formats validation errors from the validator library into a user-friendly message.
// It extracts field names and validation tags from the error.
func formatValidationErrors(err error) string {
	if err == nil {
		return ""
	}
	
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}
	
	errorMessages := make([]string, 0, len(validationErrors))
	
	for _, e := range validationErrors {
		fieldName := e.Field()
		tag := e.Tag()
		param := e.Param()
		
		var errorMessage string
		
		switch tag {
		case "required":
			errorMessage = fmt.Sprintf("%s is required", fieldName)
		case "email":
			errorMessage = fmt.Sprintf("%s must be a valid email address", fieldName)
		case "min":
			errorMessage = fmt.Sprintf("%s must be at least %s characters long", fieldName, param)
		case "max":
			errorMessage = fmt.Sprintf("%s must not exceed %s characters", fieldName, param)
		case "gte":
			errorMessage = fmt.Sprintf("%s must be greater than or equal to %s", fieldName, param)
		case "lte":
			errorMessage = fmt.Sprintf("%s must be less than or equal to %s", fieldName, param)
		case "oneof":
			errorMessage = fmt.Sprintf("%s must be one of: %s", fieldName, param)
		case "uuid":
			errorMessage = fmt.Sprintf("%s must be a valid UUID", fieldName)
		default:
			errorMessage = fmt.Sprintf("%s failed validation for tag %s: %s", fieldName, tag, param)
		}
		
		errorMessages = append(errorMessages, errorMessage)
	}
	
	return strings.Join(errorMessages, "; ")
}