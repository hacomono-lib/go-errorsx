package errorsx

import "errors"

func (e *Error) WithNotFound() *Error {
	clone := *e
	clone.isNotFound = true
	return &clone
}

func (e *Error) IsNotFound() bool {
	return e.isNotFound
}

func NewNotFound(idOrMsg string) *Error {
	return New(idOrMsg).WithNotFound()
}

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
