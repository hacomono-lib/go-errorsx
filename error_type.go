package errorsx

import "errors"

// ErrorType represents a string-based error category for classification and filtering.
// Error types enable systematic error handling by allowing code to identify
// and respond to different categories of errors consistently.
//
// Example usage:
//
//	// Define custom error types
//	const TypeAuthentication ErrorType = \"myapp.authentication\"
//	
//	err := errorsx.New("auth.failed", errorsx.WithType(TypeAuthentication))
//	if errorsx.HasType(err, TypeAuthentication) {
//		// Handle authentication errors
//	}
type ErrorType string

// Predefined error types for common error categories.
// Applications can define additional custom types as needed.
const (
	// TypeInitialization represents errors that occur during system or component initialization.
	TypeInitialization ErrorType = "errorsx.initialization"
	
	// TypeUnknown is the default error type when no specific type is assigned.
	TypeUnknown ErrorType = "errorsx.unknown"
	
	// TypeValidation represents errors related to input validation and data constraints.
	TypeValidation ErrorType = "errorsx.validation"
)

// WithType returns a copy of the error with the specified ErrorType.
// This method allows changing the error type while preserving all other attributes.
//
// Example:
//
//	err := errorsx.New("generic.error")
//	typedErr := err.WithType(errorsx.TypeValidation)
func (e *Error) WithType(typ ErrorType) *Error {
	clone := *e
	clone.errType = typ

	return &clone
}

// Type returns the ErrorType of the error.
// Returns TypeUnknown if no specific type was set.
func (e *Error) Type() ErrorType {
	return e.errType
}

// Type extracts the ErrorType from a generic error.
// If the error is not an errorsx.Error, returns TypeUnknown.
//
// This function enables type checking for any error, including
// wrapped errors and errors from external libraries.
func Type(err error) ErrorType {
	if e, ok := err.(*Error); ok {
		return e.errType
	}

	return TypeUnknown
}

// FilterByType recursively searches an error chain and returns all errorsx.Error
// instances that match the specified ErrorType. This function traverses both
// simple error chains (via Unwrap()) and joined errors (multiple errors).
//
// The function prevents duplicate results by tracking already-seen errors.
//
// Example:
//
//	err1 := errorsx.New("validation.required", errorsx.WithType(errorsx.TypeValidation))
//	err2 := errorsx.New("validation.format", errorsx.WithType(errorsx.TypeValidation))
//	combined := errorsx.Join(err1, err2)
//	
//	validationErrors := errorsx.FilterByType(combined, errorsx.TypeValidation)
//	// Returns []*Error containing both validation errors
//
// Returns an empty slice if no errors of the specified type are found.
func FilterByType(err error, typ ErrorType) []*Error {
	var result []*Error
	seen := map[*Error]struct{}{}
	var walk func(error)

	walk = func(err error) {
		if err == nil {
			return
		}

		// Extract and type check *errorsx.Error
		var e *Error
		if errors.As(err, &e) {
			if _, ok := seen[e]; ok {
				return
			}
			seen[e] = struct{}{}
			if e.errType == typ {
				result = append(result, e)
			}
		}

		// Handle joinError: when Unwrap() returns []error
		if unwrapper, ok := err.(interface{ Unwrap() []error }); ok {
			for _, ue := range unwrapper.Unwrap() {
				if ue != nil {
					walk(ue)
				}
			}
			return
		}

		// Handle normal Unwrap() returning a single error
		if ue := errors.Unwrap(err); ue != nil {
			walk(ue)
		}
	}

	walk(err)

	return result
}

// HasType checks if an error chain contains any errors of the specified ErrorType.
// This is a convenience function that returns true if FilterByType would return
// a non-empty slice.
//
// Example:
//
//	if errorsx.HasType(err, errorsx.TypeValidation) {
//		// Handle validation errors
//		return handleValidationError(err)
//	}
//
// This function is more efficient than FilterByType when you only need to check
// for the presence of a specific error type without accessing the errors themselves.
func HasType(err error, typ ErrorType) bool {
	return len(FilterByType(err, typ)) > 0
}
