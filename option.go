package errorsx

// Option represents a function that configures an Error.
type Option func(*Error)

// WithType sets the error type.
func WithType(errType ErrorType) Option {
	return func(e *Error) {
		e.errType = errType
	}
}

// WithHTTPStatus sets the HTTP status code.
func WithHTTPStatus(status int) Option {
	return func(e *Error) {
		e.status = status
	}
}

// WithMessage sets the message data.
func WithMessage(data any) Option {
	return func(e *Error) {
		e.messageData = data
	}
}

// WithNotFound sets the not found flag.
func WithNotFound() Option {
	return func(e *Error) {
		e.isNotFound = true
	}
}
