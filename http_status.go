package errorsx

// WithHTTPStatus returns a copy of the error with the specified HTTP status code.
// This is useful for web applications that need to map errors to appropriate
// HTTP response codes.
//
// Example:
//
//	err := errorsx.New("user.not_found").
//		WithHTTPStatus(404).
//		WithMessage("User not found")
//
// Common HTTP status codes for errors:
//   - 400 Bad Request: Client errors, validation failures
//   - 401 Unauthorized: Authentication required
//   - 403 Forbidden: Access denied
//   - 404 Not Found: Resource not found
//   - 409 Conflict: Resource conflicts
//   - 422 Unprocessable Entity: Semantic validation errors
//   - 500 Internal Server Error: Server-side errors
func (e *Error) WithHTTPStatus(status int) *Error {
	clone := *e
	clone.status = status
	return &clone
}

// HTTPStatus returns the HTTP status code associated with this error.
// Returns 0 if no HTTP status code was set.
//
// This method is typically used by web frameworks or middleware to
// determine the appropriate HTTP response code for an error.
func (e *Error) HTTPStatus() int {
	return e.status
}

// HTTPStatus extracts the HTTP status code from any error.
// If the error is an errorsx.Error with a status code, returns that code.
// Otherwise returns 0.
//
// This function enables HTTP status code extraction from any error in
// an error chain, making it useful for middleware and error handlers.
//
// Example:
//
//	status := errorsx.HTTPStatus(err)
//	if status != 0 {
//		w.WriteHeader(status)
//	} else {
//		w.WriteHeader(500) // Default to internal server error
//	}
//
// Returns 0 if no HTTP status is found or if err is nil.
func HTTPStatus(err error) int {
	if e, ok := err.(*Error); ok && e.status != 0 {
		return e.status
	}
	return 0
}
