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

// WithSummaryTranslator sets a custom translator for the summary message.
func (v *ValidationError) WithSummaryTranslator(t SummaryTranslator) *ValidationError {
	v.summaryTranslator = t
	return v
}

// WithFieldTranslator sets a custom translator for field error messages.
func (v *ValidationError) WithFieldTranslator(t FieldTranslator) *ValidationError {
	v.fieldTranslator = t
	return v
}

// AddFieldError adds error information for a specific field.
//
//	field: Form field name (e.g., "email")
//	code: Error code (e.g., "required", "invalid_format")
//	message: Message data (can be string, object, or any type)
func (v *ValidationError) AddFieldError(field, code string, message any) {
	v.FieldErrors = append(v.FieldErrors, FieldError{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// Error returns a string representation of the validation error.
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

// Unwrap() is a method for unwrapping errors.
// This allows BaseError to be traced with errors.Is / errors.As.
func (v *ValidationError) Unwrap() error {
	return v.BaseError
}

// HTTPStatus() returns HTTP status for validation errors.
func (v *ValidationError) HTTPStatus() int {
	return v.BaseError.status
}

// MarshalJSON serializes the validation error to JSON.
// Returns a structure like:
//
//	{
//	  "id": "validation.failed",
//	  "type": "validation",
//	  "message_data": "Form validation failed",
//	  "message": "Validation failed with 2 error(s)",
//	  "field_errors": [
//	      {
//	        "field": "email",
//	        "code": "required",
//	        "message": "Email is required",
//	        "translated_message": "Email is required"
//	      },
//	      {
//	        "field": "password",
//	        "code": "min_length",
//	        "message": {"min": 8, "current": 3},
//	        "translated_message": "Password must be at least 8 characters"
//	      }
//	  ]
//	}
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
