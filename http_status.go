package errorsx

// WithHTTPStatus returns a copy of the error with the given HTTP status code.
func (e *Error) WithHTTPStatus(status int) *Error {
	clone := *e
	clone.status = status
	return &clone
}

// HTTPStatus returns the HTTP status code associated with the error.
func (e *Error) HTTPStatus() int {
	return e.status
}

// HTTPStatus extracts the HTTP status code from a generic error if available.
func HTTPStatus(err error) int {
	if e, ok := err.(*Error); ok && e.status != 0 {
		return e.status
	}
	return 0
}
