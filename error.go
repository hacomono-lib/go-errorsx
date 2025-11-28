// Package errorsx provides structured error handling with advanced features
// for Go applications, including stack traces, error chaining, type classification,
// and JSON serialization.
//
// This package is designed for production web applications that need:
//   - Structured error information with unique IDs
//   - Stack trace capture and management
//   - Error type classification for different handling strategies
//   - HTTP status code mapping
//   - Internationalization support through message data
//   - Validation error aggregation
//   - JSON serialization for API responses
//
// Basic Usage:
//
//	err := errorsx.New("user.not_found",
//		errorsx.WithType(errorsx.TypeNotFound),
//		errorsx.WithHTTPStatus(404),
//		errorsx.WithMessage(map[string]string{"en": "User not found"}),
//	)
//
// Error Chaining:
//
//	err := errorsx.New("database.connection_failed").
//		WithCause(originalErr).
//		WithCallerStack()
//
// Validation Errors:
//
//	verr := errorsx.NewValidationError("validation.failed")
//	verr.AddFieldError("email", "required", "Email is required")
//	verr.AddFieldError("age", "min_value", map[string]int{"min": 18})
package errorsx

import (
	"errors"
	"fmt"
)

// Error represents a structured, chainable error with stack trace and attributes.
// It provides enhanced error handling capabilities beyond Go's standard error interface,
// including unique identification, type classification, HTTP status mapping,
// and structured message data for internationalization.
//
// Error implements the standard error interface and supports error unwrapping
// through the Unwrap() method, making it compatible with Go's error handling
// patterns including errors.Is() and errors.As().
type Error struct {
	id                string
	msg               string
	errType           ErrorType
	typeInferer       ErrorTypeInferer
	status            int
	messageData       any
	stacks            []StackTrace
	cause             error
	stackTraceCleaner StackTraceCleaner
	isNotFound        bool
	isRetryable       bool
	isStacked         bool
}

// New creates a new Error with the given id and options.
// The id serves as a unique identifier for the error type and is used
// for error comparison with errors.Is().
//
// Example:
//
//	err := errorsx.New("user.not_found",
//		errorsx.WithType(errorsx.TypeNotFound),
//		errorsx.WithHTTPStatus(404),
//		errorsx.WithMessage("User not found"),
//	)
//
// The id should follow a hierarchical naming convention (e.g., "domain.operation.reason")
// to facilitate error categorization and handling.
func New(id string, opts ...Option) *Error {
	e := &Error{
		id:          id,
		msg:         id,
		errType:     TypeUnknown,
		stacks:      nil,
		isNotFound:  false,
		isRetryable: false,
		isStacked:   false,
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ID returns the unique identifier of the error.
// This ID is used for error comparison and categorization.
func (e *Error) ID() string {
	return e.id
}

// WithReason returns a copy of the error with the given technical message.
// This message is intended for logging and debugging purposes and supports
// format string parameters similar to fmt.Sprintf.
//
// Example:
//
//	err := errorsx.New("database.query_failed").
//		WithReason("Failed to execute query: %s", query)
//
// Note: This creates a shallow copy of the error, preserving the original
// error's stack traces and other attributes.
func (e *Error) WithReason(reason string, params ...any) *Error {
	clone := *e
	clone.msg = fmt.Sprintf(reason, params...)

	return &clone
}

// Error implements the standard Go error interface.
// It returns the technical message set by WithReason(), or the error ID
// if no specific message was provided.
func (e *Error) Error() string {
	return e.msg
}

// Unwrap returns the underlying cause error, enabling Go's error unwrapping
// functionality. This allows errors.Is() and errors.As() to traverse the
// error chain to find specific error types or values.
//
// Returns nil if no underlying cause was set.
func (e *Error) Unwrap() error {
	return e.cause
}

// Is implements custom error comparison for errors.Is().
// Two errorsx.Error instances are considered equal if they have the same ID.
// For non-errorsx errors, it delegates to the underlying error's Is method
// or compares the cause error.
//
// This enables error identification based on semantic meaning rather than
// instance equality, supporting error handling patterns like:
//
//	if errors.Is(err, errorsx.New("user.not_found")) {
//		// Handle user not found error
//	}
func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	if e == target {
		return true
	}
	t, ok := target.(*Error)
	if !ok {
		return errors.Is(e.cause, target)
	}

	return e.id == t.id
}

// WithMessage returns a copy of the error with the given message data.
// This data is typically used for user-facing messages and can be any type,
// often a string or a map for internationalization.
//
// Example with string message:
//
//	err := errorsx.New("user.not_found").WithMessage("User not found")
//
// Example with i18n data:
//
//	err := errorsx.New("user.not_found").WithMessage(map[string]string{
//		"en": "User not found",
//		"ja": "ユーザーが見つかりません",
//	})
//
// The message data can be extracted using the Message() or MessageOr() functions.
func (e *Error) WithMessage(data any) *Error {
	clone := *e
	clone.messageData = data

	return &clone
}

// ReplaceMessage replaces the message data of any error.
// If the error is an errorsx.Error, it returns a copy with the new message data.
// If the error is a standard Go error, it wraps it in a new errorsx.Error
// with the provided message data.
//
// This function is useful for adding user-friendly messages to errors
// that may come from external libraries or system calls.
//
// Example:
//
//	err := os.Open("nonexistent.txt") // returns *os.PathError
//	userErr := errorsx.ReplaceMessage(err, "File not found")
//	// userErr is now an errorsx.Error with the original error as cause
func ReplaceMessage(err error, data any) error {
	var xerr *Error
	if errors.As(err, &xerr) {
		return xerr.WithMessage(data)
	}

	return New("unknown.error").
		WithMessage(data).
		WithCause(err)
}

// Message extracts typed message data from an error.
// It performs type assertion to convert the message data to the specified type T.
// Returns the message data and true if successful, or zero value and false if
// the error is not an errorsx.Error or the type assertion fails.
//
// Example:
//
//	err := errorsx.New("user.not_found").WithMessage(map[string]string{
//		"en": "User not found",
//	})
//
//	if msg, ok := errorsx.Message[map[string]string](err); ok {
//		fmt.Println(msg["en"]) // "User not found"
//	}
//
// This function is particularly useful for extracting structured message data
// such as translation maps or validation error details.
func Message[T any](err error) (T, bool) {
	var zero T
	if e, ok := err.(*Error); ok && e.messageData != nil {
		if data, ok := e.messageData.(T); ok {
			return data, true
		}
	}

	return zero, false
}

// MessageOr extracts typed message data from an error with a fallback value.
// If the error is not an errorsx.Error or the type assertion fails,
// it returns the provided fallback value instead of a zero value.
//
// Example:
//
//	msg := errorsx.MessageOr(err, "Unknown error")
//	// msg will be the extracted message or "Unknown error" if extraction fails
//
// This function provides a convenient way to safely extract message data
// without needing to check the boolean return value.
func MessageOr[T any](err error, fallback T) T {
	if data, ok := Message[T](err); ok {
		return data
	}

	return fallback
}

// ReplaceType replaces the error type of any error.
// If the error is an errorsx.Error, it returns a copy with the new error type.
// If the error is a standard Go error, it returns the original error unchanged.
//
// This function is useful for changing the error type of errorsx.Error instances
// received from lower layers while preserving other error types as-is.
//
// Example:
//
//	// Replace type of an errorsx.Error
//	err := errorsx.New("validation.failed")
//	typedErr := errorsx.ReplaceType(err, errorsx.TypeValidation)
//
//	// Standard Go error remains unchanged
//	osErr := os.Open("nonexistent.txt") // returns *os.PathError
//	result := errorsx.ReplaceType(osErr, errorsx.TypeNotFound)
//	// result == osErr (unchanged)
func ReplaceType(err error, typ ErrorType) error {
	if err == nil {
		return nil
	}

	var xerr *Error
	if errors.As(err, &xerr) {
		return xerr.WithType(typ)
	}

	return err
}
