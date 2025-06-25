package errorsx

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SummaryTranslator is a function type that translates a summary message.
// It receives field errors and base message data to generate a localized summary.
type SummaryTranslator func(fieldErrors []FieldError, messageData any) string

// FieldTranslator is a function type that translates individual field error messages.
// It receives field name, error code, and message data to generate a localized message.
type FieldTranslator func(field, code string, message any) string

// DefaultSummaryTranslator returns a simple summary message.
func DefaultSummaryTranslator(fieldErrors []FieldError, messageData any) string {
	if messageData != nil {
		if msg, ok := messageData.(string); ok {
			return msg
		}
		return fmt.Sprintf("%v", messageData)
	}
	return fmt.Sprintf("Validation failed with %d error(s)", len(fieldErrors))
}

// DefaultFieldTranslator returns the message as-is if it's a string, otherwise formats it.
func DefaultFieldTranslator(field, code string, message any) string {
	if message == nil {
		return code
	}
	if msg, ok := message.(string); ok {
		return msg
	}
	return fmt.Sprintf("%v", message)
}

// FieldError represents individual error information that occurred for a specific field.
type FieldError struct {
	Field   string `json:"field"`   // Form field name
	Code    string `json:"code"`    // Error code (e.g., "required", "invalid_format")
	Message any    `json:"message"` // Message data (can be string, object, or any type)
}

// ValidationError is an error type that holds multiple field errors together.
// By having an existing *Error internally, it can be easily combined with business layer errors.
// Implements Error() to satisfy the error interface, and additionally
// implements MarshalJSON() to return field errors for JSON output.
type ValidationError struct {
	BaseError         *Error            `json:"-"`            // Existing errorsx.Error (ID, Type, HTTPStatus, etc.)
	FieldErrors       []FieldError      `json:"field_errors"` // Error list for each field name
	summaryTranslator SummaryTranslator // Translator for summary message
	fieldTranslator   FieldTranslator   // Translator for field error messages
}

// NewValidationError creates a new instance as a form input validation error.
//
//	id: ID that uniquely identifies the validation error (e.g., "validation.failed")
func NewValidationError(id string) *ValidationError {
	base := New(id, WithType(TypeValidation))

	return &ValidationError{
		BaseError:         base,
		FieldErrors:       nil,
		summaryTranslator: DefaultSummaryTranslator,
		fieldTranslator:   DefaultFieldTranslator,
	}
}

// WithHTTPStatus sets the HTTP status code for the validation error.
func (v *ValidationError) WithHTTPStatus(status int) *ValidationError {
	v.BaseError.status = status
	return v
}

// WithMessage sets the message data for the base error.
func (v *ValidationError) WithMessage(data any) *ValidationError {
	v.BaseError.messageData = data
	return v
}

// WithSummaryTranslator sets a custom translator for generating the overall
// validation error summary message. This is useful for internationalization
// or custom error message formatting.
//
// Example:
//
//	customSummary := func(fieldErrors []FieldError, messageData any) string {
//		return fmt.Sprintf("Validation failed for %d fields", len(fieldErrors))
//	}
//	verr.WithSummaryTranslator(customSummary)
func (v *ValidationError) WithSummaryTranslator(t SummaryTranslator) *ValidationError {
	v.summaryTranslator = t
	return v
}

// WithFieldTranslator sets a custom translator for individual field error messages.
// This enables localization and custom formatting of field-specific error messages.
//
// Example:
//
//	customFieldTranslator := func(field, code string, message any) string {
//		switch code {
//		case "required":
//			return fmt.Sprintf("The %s field is required", field)
//		default:
//			return fmt.Sprintf("%v", message)
//		}
//	}
//	verr.WithFieldTranslator(customFieldTranslator)
func (v *ValidationError) WithFieldTranslator(t FieldTranslator) *ValidationError {
	v.fieldTranslator = t
	return v
}

// AddFieldError adds validation error information for a specific field.
// This method accumulates field errors that will be included in the final
// validation error response.
//
// Parameters:
//   - field: The name of the form field (e.g., "email", "password")
//   - code: Machine-readable error code (e.g., "required", "min_length", "invalid_format")
//   - message: Human-readable message or structured data for the error
//
// Examples:
//
//	// Simple string message
//	verr.AddFieldError("email", "required", "Email address is required")
//
//	// Structured message data for complex validation rules
//	verr.AddFieldError("password", "min_length", map[string]int{
//		"min": 8,
//		"current": 3,
//	})
//
//	// Internationalization data
//	verr.AddFieldError("username", "taken", map[string]string{
//		"en": "Username is already taken",
//		"ja": "ユーザー名は既に使用されています",
//	})
func (v *ValidationError) AddFieldError(field, code string, message any) {
	v.FieldErrors = append(v.FieldErrors, FieldError{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// Error implements the standard error interface.
// It returns a human-readable string that includes the base error message
// and details about each field error. The format is suitable for logging
// and debugging purposes.
//
// Example output: "validation failed: email: required; password: too short".
func (v *ValidationError) Error() string {
	if len(v.FieldErrors) == 0 {
		return v.BaseError.msg
	}
	// Example: "validation failed: email is required; password is too short"
	var parts []string
	for _, fe := range v.FieldErrors {
		// Use field translator to convert message to string
		msgStr := v.fieldTranslator(fe.Field, fe.Code, fe.Message)
		parts = append(parts, fmt.Sprintf("%s: %s", fe.Field, msgStr))
	}
	return fmt.Sprintf("%s: %s", v.BaseError.msg, strings.Join(parts, "; "))
}

// Unwrap returns the underlying base error, enabling Go's error unwrapping
// functionality. This allows ValidationError to participate in error chains
// and be compatible with errors.Is() and errors.As().
//
// Example:
//
//	if errors.Is(validationErr, someSpecificError) {
//		// Handle specific error type
//	}
func (v *ValidationError) Unwrap() error {
	return v.BaseError
}

// HTTPStatus returns the HTTP status code associated with this validation error.
// This is typically used by web frameworks to set appropriate HTTP response codes.
// Defaults to 0 if no status was explicitly set.
//
// Common validation error status codes:
//   - 400 Bad Request: General validation failures
//   - 422 Unprocessable Entity: Semantic validation errors
func (v *ValidationError) HTTPStatus() int {
	return v.BaseError.status
}

// MarshalJSON implements json.Marshaler to provide custom JSON serialization.
// This method creates a structured JSON representation suitable for API responses,
// including both the raw message data and translated messages for each field error.
//
// The JSON structure includes:
//   - id: The unique error identifier
//   - type: The error type (typically "validation")
//   - message_data: The raw message data from WithMessage()
//   - message: The translated summary message
//   - field_errors: Array of field-specific errors with translations
//
// Example JSON output:
//
//	{
//	  "id": "user.validation_failed",
//	  "type": "validation",
//	  "message_data": "Please fix the following errors",
//	  "message": "Validation failed with 2 error(s)",
//	  "field_errors": [
//	    {
//	      "field": "email",
//	      "code": "required",
//	      "message": "Email is required",
//	      "translated_message": "Email is required"
//	    },
//	    {
//	      "field": "password",
//	      "code": "min_length",
//	      "message": {"min": 8, "current": 3},
//	      "translated_message": "Password must be at least 8 characters"
//	    }
//	  ]
//	}
//
// This format enables both programmatic error handling (using codes and structured data)
// and user-friendly error display (using translated messages).
func (v *ValidationError) MarshalJSON() ([]byte, error) {
	type fieldErrorWithTranslation struct {
		Field             string `json:"field"`
		Code              string `json:"code"`
		Message           any    `json:"message"`
		TranslatedMessage string `json:"translated_message"`
	}

	type alias struct {
		ID          string                      `json:"id"`
		Type        ErrorType                   `json:"type"`
		MessageData any                         `json:"message_data,omitempty"`
		Message     string                      `json:"message"`
		FieldErrors []fieldErrorWithTranslation `json:"field_errors"`
	}

	// Create field errors with translated messages
	fieldErrors := make([]fieldErrorWithTranslation, len(v.FieldErrors))
	for i, fe := range v.FieldErrors {
		fieldErrors[i] = fieldErrorWithTranslation{
			Field:             fe.Field,
			Code:              fe.Code,
			Message:           fe.Message,
			TranslatedMessage: v.fieldTranslator(fe.Field, fe.Code, fe.Message),
		}
	}

	out := alias{
		ID:          v.BaseError.id,
		Type:        v.BaseError.errType,
		MessageData: v.BaseError.messageData,
		Message:     v.summaryTranslator(v.FieldErrors, v.BaseError.messageData),
		FieldErrors: fieldErrors,
	}

	return json.Marshal(out)
}
