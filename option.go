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
//
// Note: Setting an explicit type will clear any inferer, and vice versa.
func WithType(errType ErrorType) Option {
	return func(e *Error) {
		e.errType = errType
		e.typeInferer = nil    // Clear any inferer when explicit type is set
		e.computedErrType = "" // Clear cache
		e.computing = false    // Reset computing flag
	}
}

// WithTypeInferer sets a dynamic type inferer function that determines
// the error type at runtime based on the error's attributes.
// This enables flexible error classification based on patterns in
// the error ID, stack traces, or other error properties.
//
// Use built-in helper functions for common patterns, or create custom inferers.
//
// Example with pattern matching:
//
//	inferer := errorsx.IDPatternInferer(map[string]errorsx.ErrorType{
//		"auth.*":       TypeAuthentication,
//		"validation.*": errorsx.TypeValidation,
//	})
//	err := errorsx.New("auth.failed", errorsx.WithTypeInferer(inferer))
//
// Example with chaining:
//
//	inferer := errorsx.ChainInferers(
//		errorsx.IDContainsInferer(map[string]errorsx.ErrorType{
//			"auth": TypeAuthentication,
//		}),
//		func(e *errorsx.Error) errorsx.ErrorType {
//			// Custom logic based on stack traces, etc.
//			return errorsx.TypeUnknown
//		},
//	)
//
// Note: Setting an inferer will clear any explicit type, and vice versa.
func WithTypeInferer(inferer ErrorTypeInferer) Option {
	return func(e *Error) {
		e.typeInferer = inferer
		e.errType = TypeUnknown // Reset explicit type when inferer is set
		e.computedErrType = ""  // Clear cache
		e.computing = false     // Reset computing flag
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

// WithRetryable marks the error as retryable.
// This indicates that the operation that caused the error can be safely retried,
// such as temporary network failures or transient service unavailability.
//
// Example:
//
//	err := errorsx.New("service.unavailable",
//		errorsx.WithRetryable(),
//		errorsx.WithHTTPStatus(503),
//	)
func WithRetryable() Option {
	return func(e *Error) {
		e.isRetryable = true
	}
}
