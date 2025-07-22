package errorsx

import "errors"

// WithRetryable returns a copy of the error marked as retryable.
// This indicates that the operation that caused the error can be safely retried.
//
// Example:
//
//	err := errorsx.New("network.timeout").
//		WithRetryable().
//		WithHTTPStatus(503)
func (e *Error) WithRetryable() *Error {
	clone := *e
	clone.isRetryable = true
	return &clone
}

// IsRetryable returns true if this error represents a retryable condition.
// This provides a semantic way to check if an operation can be safely retried.
func (e *Error) IsRetryable() bool {
	return e.isRetryable
}

// NewRetryable creates a new retryable error with the given ID.
// This is a convenience constructor for creating errors that indicate
// the operation can be safely retried.
//
// Example:
//
//	err := errorsx.NewRetryable("connection.timeout")
//	// Equivalent to: errorsx.New("connection.timeout").WithRetryable()
func NewRetryable(idOrMsg string) *Error {
	return New(idOrMsg).WithRetryable()
}

// IsRetryable checks if any error in the error chain represents a retryable condition.
// This function works with any error type and traverses the error chain to find
// errorsx.Error instances marked as retryable.
//
// Example:
//
//	if errorsx.IsRetryable(err) {
//		// Retry the operation
//		return retryOperation()
//	}
//
// Returns false if err is nil or no retryable errors are found in the chain.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var e *Error
	if errors.As(err, &e) {
		return e.IsRetryable()
	}
	return false
}
