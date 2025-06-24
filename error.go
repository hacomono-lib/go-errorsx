package errorsx

import (
	"errors"
	"fmt"
)

// Error represents a structured, chainable error with stack trace and attributes.
type Error struct {
	id                string
	msg               string
	errType           ErrorType
	status            int
	messageData       any
	stacks            []StackTrace
	cause             error
	stackTraceCleaner StackTraceCleaner
	isNotFound        bool
	isStacked         bool
}

// New creates a new Error with the given id and options.
func New(id string, opts ...Option) *Error {
	e := &Error{
		id:         id,
		msg:        id,
		errType:    TypeUnknown,
		stacks:     nil,
		isNotFound: false,
		isStacked:  false,
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// ID returns the error id.
func (e *Error) ID() string {
	return e.id
}

// WithReason returns a copy of the error with the given technical message.
func (e *Error) WithReason(reason string, params ...any) *Error {
	clone := *e
	clone.msg = fmt.Sprintf(reason, params...)

	return &clone
}

// Error implements the error interface and returns the error message.
func (e *Error) Error() string {
	return e.msg
}

// Unwrap returns the cause of the error for error unwrapping.
func (e *Error) Unwrap() error {
	return e.cause
}

// Is implements custom error comparison for errors.Is. It compares both id and errType.
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

// WithMessage returns a copy of the error with the given message data for UI display.
func (e *Error) WithMessage(data any) *Error {
	clone := *e
	clone.messageData = data

	return &clone
}

// ReplaceMessage replaces the error message.
// If the error is an errorsx.Error, it uses WithMessage.
// Otherwise, it creates a new errorsx.Error.
func ReplaceMessage(err error, data any) error {
	var xerr *Error
	if errors.As(err, &xerr) {
		return xerr.WithMessage(data)
	}

	return New("unknown.error").
		WithMessage(data).
		WithCause(err)
}

// Message extracts the message data from a generic error and performs type assertion.
// Returns the message data of type T and a boolean indicating success.
// If the error is not an errorsx.Error or the type assertion fails, returns zero value and false.
func Message[T any](err error) (T, bool) {
	var zero T
	if e, ok := err.(*Error); ok && e.messageData != nil {
		if data, ok := e.messageData.(T); ok {
			return data, true
		}
	}

	return zero, false
}

// MessageOr extracts the message data from a generic error with a fallback value.
// If the type assertion fails, returns the provided fallback value.
func MessageOr[T any](err error, fallback T) T {
	if data, ok := Message[T](err); ok {
		return data
	}

	return fallback
}
