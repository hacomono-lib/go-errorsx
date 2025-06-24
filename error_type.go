package errorsx

import "errors"

// ErrorType represents a string-based error category or classification.
type ErrorType string

const (
	TypeInitialization ErrorType = "errorsx.initialization" // for initialization errors
	TypeUnknown        ErrorType = "errorsx.unknown"        // default value
	TypeValidation     ErrorType = "errorsx.validation"     // for validation errors
)

// WithType returns a copy of the error with the given ErrorType.
func (e *Error) WithType(typ ErrorType) *Error {
	clone := *e
	clone.errType = typ

	return &clone
}

// Type returns the ErrorType of the error.
func (e *Error) Type() ErrorType {
	return e.errType
}

// Type extracts the ErrorType from a generic error if available.
func Type(err error) ErrorType {
	if e, ok := err.(*Error); ok {
		return e.errType
	}

	return TypeUnknown
}

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

func HasType(err error, typ ErrorType) bool {
	return len(FilterByType(err, typ)) > 0
}
