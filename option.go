package errorsx

// Option represents a function that configures an Error during creation.
// Options follow the functional options pattern, allowing flexible
// and extensible error configuration.
type Option func(*Error)

// WithType sets the error type for classification and filtering.
// Error types help categorize errors for different handling strategies.
//
// Example:
//
//	err := errorsx.New("input.invalid",
//		errorsx.WithType(errorsx.TypeValidation),
//	)
func WithType(errType ErrorType) Option {
	return func(e *Error) {
		e.errType = errType
	}
}

// WithHTTPStatus sets the HTTP status code for web API responses.
// This allows automatic HTTP status code mapping in web handlers.
//
// Example:
//
//	err := errorsx.New("user.not_found",
//		errorsx.WithHTTPStatus(404),
//	)
func WithHTTPStatus(status int) Option {
	return func(e *Error) {
		e.status = status
	}
}

// WithMessage sets the message data for user-facing display.
// The data can be any type - string for simple messages,
// or structured data (maps, structs) for internationalization.
//
// Example with simple message:
//
//	err := errorsx.New("user.not_found",
//		errorsx.WithMessage("User not found"),
//	)
//
// Example with i18n data:
//
//	err := errorsx.New("user.not_found",
//		errorsx.WithMessage(map[string]string{
//			"en": "User not found",
//			"ja": "ユーザーが見つかりません",
//		}),
//	)
func WithMessage(data any) Option {
	return func(e *Error) {
		e.messageData = data
	}
}

// WithNotFound marks the error as a "not found" error.
// This is a convenience method that's equivalent to WithType(TypeNotFound)
// but can be used in combination with other types for more specific classification.
//
// Example:
//
//	err := errorsx.New("user.not_found",
//		errorsx.WithNotFound(),
//		errorsx.WithHTTPStatus(404),
//	)
func WithNotFound() Option {
	return func(e *Error) {
		e.isNotFound = true
	}
}
