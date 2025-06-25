package errorsx

import "errors"

// WithNotFound returns a copy of the error marked as a "not found" error.
// This is a convenience method for common "not found" scenarios.
//
// Example:
//
//	err := errorsx.New("user.not_found").
//		WithNotFound().
//		WithHTTPStatus(404)
func (e *Error) WithNotFound() *Error {
	clone := *e
	clone.isNotFound = true
	return &clone
}

// IsNotFound returns true if this error represents a "not found" condition.
// This provides a semantic way to check for missing resources or entities.
func (e *Error) IsNotFound() bool {
	return e.isNotFound
}

// NewNotFound creates a new "not found" error with the given ID.
// This is a convenience constructor for the common pattern of creating
// errors that represent missing resources.
//
// Example:
//
//	err := errorsx.NewNotFound("user.not_found")
//	// Equivalent to: errorsx.New("user.not_found").WithNotFound()
func NewNotFound(idOrMsg string) *Error {
	return New(idOrMsg).WithNotFound()
}

// IsNotFound checks if any error in the error chain represents a "not found" condition.
// This function works with any error type and traverses the error chain to find
// errorsx.Error instances marked as "not found".
//
// Example:
//
//	if errorsx.IsNotFound(err) {
//		// Handle not found case - typically return 404
//		return handleNotFound()
//	}
//
// Returns false if err is nil or no "not found" errors are found in the chain.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *Error
	if errors.As(err, &e) {
		return e.IsNotFound()
	}
	return false
}
