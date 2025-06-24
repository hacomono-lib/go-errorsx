package errorsx

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MessageKeyValidationSummary = "validation.summary"
)

// SummaryTranslator is a function type that translates a summary message key and its
// parameters into a localized message. fieldErrors is passed by reference to access the number of errors.
type SummaryTranslator func(fieldErrors []FieldError, key string, params ...any) string

// FieldTranslator is a function type that translates a field error message key and its
// parameters into a localized message.
type FieldTranslator func(key string, params ...any) string

// DefaultSummaryTranslator returns the message key as is.
func DefaultSummaryTranslator(_ []FieldError, key string, _ ...any) string {
	return key
}

// DefaultFieldTranslator returns the message key as is.
func DefaultFieldTranslator(key string, _ ...any) string {
	return key
}

// FieldError represents individual error information that occurred for a specific field.
// - Field: Input field name on the form (e.g., "email", "password", etc.)
// - MessageKey: Message key (e.g., "validation.required", "validation.email", etc.)
// - MessageParams: Parameters for i18n or embedded values. Use as needed.
type FieldError struct {
	Field         string `json:"field"`          // Form field name
	MessageKey    string `json:"message_key"`    // Message key
	MessageParams []any  `json:"message_params"` // Message parameters
	Message       string `json:"message"`        // Translated message (for JSON output)
}

// ValidationError is an error type that "holds multiple field errors together".
//   - By having an existing *Error internally, it can be easily combined with business layer errors.
//   - Implements Error() to satisfy the error interface, and additionally
//     implements MarshalJSON() to return field errors for JSON output.
type ValidationError struct {
	BaseError         *Error            `json:"-"`            // Existing errorsx.Error (ID, Type, HTTPStatus, etc.)
	FieldErrors       []FieldError      `json:"field_errors"` // Error list for each field name
	summaryTranslator SummaryTranslator // Translator for summary message
	fieldTranslator   FieldTranslator   // Translator for field error messages
}

// NewValidationError creates a new instance as a form input validation error.
//
//	idOrMsg: ID or message that uniquely identifies the validation error (e.g., "validation.failed")
func NewValidationError(idOrMsg string) *ValidationError {
	base := New(idOrMsg, WithType(TypeValidation), WithMessage(MessageKeyValidationSummary))

	return &ValidationError{
		BaseError:         base,
		FieldErrors:       nil,
		summaryTranslator: DefaultSummaryTranslator,
		fieldTranslator:   DefaultFieldTranslator,
	}
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

func (v *ValidationError) WithHTTPStatus(status int) *ValidationError {
	v.BaseError.status = status
	return v
}

// AddFieldError adds error information for a specific field.
//
//	field: Form field name (e.g., "email")
//	messageKey: Message key (e.g., "validation.required")
//	params: Placeholders for i18n, etc.
func (v *ValidationError) AddFieldError(field, messageKey string, params ...any) {
	v.FieldErrors = append(v.FieldErrors, FieldError{
		Field:         field,
		MessageKey:    messageKey,
		MessageParams: params,
	})
}

// Error() is a method that satisfies the error interface.
// Here, it returns a simple string that just says "there are some validation errors".
func (v *ValidationError) Error() string {
	if len(v.FieldErrors) == 0 {
		return v.BaseError.msg
	}
	// Example: "validation failed: email is required; password is too short"
	var parts []string
	for _, fe := range v.FieldErrors {
		// Combine field name + translated message
		msg := v.fieldTranslator(fe.MessageKey, fe.MessageParams...)
		parts = append(parts, fmt.Sprintf("%s: %s", fe.Field, msg))
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

// MarshalJSON() is a method called when converting to JSON, returning a structure like:
//
//	{
//	  "id": "validation.failed",
//	  "type": "validation",
//	  "message_data": "validation.summary",
//	  "message": "There are 2 errors in the input",
//	  "field_errors": [
//	      {
//	        "field": "email",
//	        "message_key": "validation.required",
//	        "message_params": [],
//	        "message": "Email is required"
//	      },
//	      {
//	        "field": "password",
//	        "message_key": "validation.min_length",
//	        "message_params": [8],
//	        "message": "Password must be at least 8 characters"
//	      }
//	  ]
//	}
func (v *ValidationError) MarshalJSON() ([]byte, error) {
	type alias struct {
		ID          string       `json:"id"`
		Type        ErrorType    `json:"type"`
		MessageData any          `json:"message_data"`
		Message     string       `json:"message"` // Translated message for base error
		FieldErrors []FieldError `json:"field_errors"`
	}

	// Create a copy of field errors with translated messages
	fieldErrors := make([]FieldError, len(v.FieldErrors))
	for i, fe := range v.FieldErrors {
		fieldErrors[i] = FieldError{
			Field:         fe.Field,
			MessageKey:    fe.MessageKey,
			MessageParams: fe.MessageParams,
			Message:       v.fieldTranslator(fe.MessageKey, fe.MessageParams...),
		}
	}

	// Extract messageKey from messageData for backward compatibility
	var messageKey string
	if key, ok := v.BaseError.messageData.(string); ok {
		messageKey = key
	}

	out := alias{
		ID:          v.BaseError.id,
		Type:        v.BaseError.errType,
		MessageData: v.BaseError.messageData,
		Message:     v.summaryTranslator(v.FieldErrors, messageKey, []any{}...),
		FieldErrors: fieldErrors,
	}
	return json.Marshal(out)
}
