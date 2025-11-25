package errorsx

import (
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

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

// ErrorTypeInferer is a function that dynamically determines the ErrorType
// based on the Error instance. This enables runtime type determination
// based on error attributes like stack traces, messages, or ID patterns.
//
// Example:
//
//	// Using built-in pattern matching
//	inferer := errorsx.IDPatternInferer(map[string]ErrorType{
//		"auth.*":       TypeAuthentication,
//		"db.*":         TypeDatabase,
//		"validation.*": TypeValidation,
//	})
//	err := errorsx.New("auth.failed", errorsx.WithTypeInferer(inferer))
//
//	// Custom inferer
//	customInferer := func(e *Error) ErrorType {
//		if len(e.stacks) > 0 {
//			return TypeValidation
//		}
//		return TypeUnknown
//	}
type ErrorTypeInferer func(*Error) ErrorType

// Predefined error types for common error categories.
// Applications can define additional custom types as needed.
const (
	// TypeInitialization represents errors that occur during system or component initialization.
	TypeInitialization ErrorType = "errorsx.initialization"

	// TypeUnknown is the default error type when no specific type is assigned.
	TypeUnknown ErrorType = "errorsx.unknown"

	// TypeValidation represents errors related to input validation and data constraints.
	TypeValidation ErrorType = "errorsx.validation"

	// TypeNotFound represents errors where a requested resource or entity cannot be found.
	TypeNotFound ErrorType = "errorsx.not_found"
)

var (
	// globalInferer is a single global ErrorTypeInferer that is
	// applied to errors when no specific inferer is set on the error instance.
	globalInferer ErrorTypeInferer //nolint:gochecknoglobals
	infererMutex  sync.RWMutex     //nolint:gochecknoglobals
)

// SetGlobalTypeInferer sets a global ErrorTypeInferer that will be
// consulted when determining error types for errors without instance-specific inferers.
//
// This is useful for handling sentinel errors from different packages
// that should be classified under the same ErrorType based on patterns.
//
// Example:
//
//	errorsx.SetGlobalTypeInferer(errorsx.IDPatternInferer(map[string]ErrorType{
//		"*database*": TypeDatabase,
//		"*sql*":      TypeDatabase,
//		"*auth*":     TypeAuthentication,
//	}))
func SetGlobalTypeInferer(inferer ErrorTypeInferer) {
	infererMutex.Lock()
	defer infererMutex.Unlock()
	globalInferer = inferer
}

// ClearGlobalTypeInferer removes the registered global type inferer.
// This is primarily useful for testing.
func ClearGlobalTypeInferer() {
	infererMutex.Lock()
	defer infererMutex.Unlock()
	globalInferer = nil
}

// IDPatternInferer creates a reusable ErrorTypeInferer that matches error IDs
// against glob-style patterns. This enables easy pattern-based error classification.
//
// Patterns support '*' wildcards and are case-sensitive. The first matching
// pattern determines the ErrorType.
//
// Example:
//
//	inferer := errorsx.IDPatternInferer(map[string]ErrorType{
//		"auth.*":       TypeAuthentication,
//		"*database*":   TypeDatabase,
//		"validation.*": TypeValidation,
//	})
//
// This inferer can be reused across multiple errors and is thread-safe.
func IDPatternInferer(patterns map[string]ErrorType) ErrorTypeInferer {
	return func(e *Error) ErrorType {
		id := e.ID()
		for pattern, errType := range patterns {
			if matched, _ := filepath.Match(pattern, id); matched {
				return errType
			}
		}
		return TypeUnknown
	}
}

// ChainInferers combines multiple ErrorTypeInferers into a single inferer.
// The inferers are evaluated in order, and the first non-TypeUnknown result is returned.
// This allows composing complex inference logic from simple, reusable components.
//
// Example:
//
//	inferer := errorsx.ChainInferers(
//		errorsx.IDPatternInferer(map[string]ErrorType{
//			"auth.*": TypeAuthentication,
//		}),
//		func(e *Error) ErrorType {
//			if len(e.stacks) > 0 {
//				return TypeValidation
//			}
//			return TypeUnknown
//		},
//	)
func ChainInferers(inferers ...ErrorTypeInferer) ErrorTypeInferer {
	return func(e *Error) ErrorType {
		for _, inferer := range inferers {
			if typ := inferer(e); typ != TypeUnknown {
				return typ
			}
		}
		return TypeUnknown
	}
}

// extractErrorFrame extracts the first frame from an error's stack trace.
func extractErrorFrame(e *Error) (runtime.Frame, bool) {
	stacks := e.Stacks()
	if len(stacks) == 0 || len(stacks[0].Frames) == 0 {
		return runtime.Frame{}, false
	}

	frames := runtime.CallersFrames(stacks[0].Frames)
	if frame, more := frames.Next(); more || frame.PC != 0 {
		return frame, true
	}

	return runtime.Frame{}, false
}

// getCauseTypeName returns the detailed type information for an error.
// This is similar to the causeTypeName function in marshal.go but reusable.
// For errorsx.Error, it returns the complete type including inferred types.
// Infinite recursion is prevented by the caching mechanism in Type().
func getCauseTypeName(err error) string {
	if e, ok := err.(*Error); ok {
		// Now safe to call Type() due to caching mechanism
		return string(e.Type())
	}

	// Use reflection to get detailed type information for external errors
	t := reflect.TypeOf(err)
	if t == nil {
		return "undefined"
	}

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Get package path and type name
	pkgPath := t.PkgPath()
	typeName := t.Name()

	if pkgPath == "" || typeName == "" {
		// Fallback to the full type string
		return reflect.TypeOf(err).String()
	}

	return pkgPath + "." + typeName
}

// StackTraceInferer creates a reusable ErrorTypeInferer that uses stack trace
// information and cause type information to dynamically determine error types.
// The matcher function receives both the error handling location and the detailed
// type information of the underlying cause error.
//
// The errorFrame represents where the current error was handled (e.g., where
// WithCause or WithCallerStack was called), while causeType provides detailed
// type information of the underlying error.
//
// Example:
//
//	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeType string) ErrorType {
//		// Handle external library errors by type
//		switch {
//		case strings.Contains(causeType, "database/sql"):
//			return ErrorType("database.error")
//		case strings.Contains(causeType, "encoding/json"):
//			return ErrorType("serialization.error")
//		case strings.Contains(causeType, "errorsx.validation"):
//			return ErrorType("validation.error")
//		}
//
//		// Handle based on error handling location
//		if strings.Contains(errorFrame.File, "/handler/") {
//			return ErrorType("web.error")
//		}
//		if strings.Contains(errorFrame.File, "/service/") {
//			return ErrorType("service.error")
//		}
//
//		return TypeUnknown
//	})
func StackTraceInferer(matcher func(errorFrame runtime.Frame, causeType string) ErrorType) ErrorTypeInferer {
	return func(e *Error) ErrorType {
		errorFrame, hasErrorFrame := extractErrorFrame(e)
		if !hasErrorFrame {
			return TypeUnknown
		}

		// Get cause type information
		var causeType string
		if cause := e.Unwrap(); cause != nil {
			causeType = getCauseTypeName(cause)
		}

		return matcher(errorFrame, causeType)
	}
}

// IDContainsInferer creates a reusable ErrorTypeInferer that checks if the error ID
// contains specific substrings. This is a simpler alternative to IDPatternInferer
// for basic substring matching.
//
// Example:
//
//	inferer := errorsx.IDContainsInferer(map[string]ErrorType{
//		"auth":       TypeAuthentication,
//		"database":   TypeDatabase,
//		"validation": TypeValidation,
//	})
func IDContainsInferer(substrings map[string]ErrorType) ErrorTypeInferer {
	return func(e *Error) ErrorType {
		id := e.ID()
		for substring, errType := range substrings {
			if strings.Contains(id, substring) {
				return errType
			}
		}
		return TypeUnknown
	}
}

// WithType returns a copy of the error with the specified ErrorType.
// This method allows changing the error type while preserving all other attributes.
//
// Example:
//
//	err := errorsx.New("generic.error")
//	typedErr := err.WithType(errorsx.TypeValidation)
//
// Note: Setting an explicit type will clear any inferer, and vice versa.
func (e *Error) WithType(typ ErrorType) *Error {
	clone := *e
	clone.errType = typ
	clone.typeInferer = nil    // Clear inferer when explicit type is set
	clone.computedErrType = "" // Clear cache
	clone.computing = false    // Reset computing flag

	return &clone
}

// WithTypeInferer returns a copy of the error with the specified ErrorTypeInferer.
// The inferer function will be called when Type() is invoked to dynamically
// determine the error type based on the error's attributes.
//
// This enables runtime type determination based on patterns in the error ID,
// stack traces, or other error attributes, which is useful for handling
// sentinel errors from different packages under a unified classification.
//
// Example:
//
//	inferer := func(e *Error) ErrorType {
//		if strings.Contains(e.ID(), "auth") {
//			return TypeAuthentication
//		}
//		return TypeUnknown
//	}
//	err := errorsx.New("auth.failed").WithTypeInferer(inferer)
//
// Note: Setting an inferer will clear any explicit type, and vice versa.
func (e *Error) WithTypeInferer(inferer ErrorTypeInferer) *Error {
	clone := *e
	clone.typeInferer = inferer
	clone.errType = TypeUnknown // Reset explicit type when inferer is set
	clone.computedErrType = ""  // Clear cache
	clone.computing = false     // Reset computing flag

	return &clone
}

// Type returns the ErrorType of the error.
// Priority order:
// 1. Explicit type (if set) - highest priority
// 2. Instance-specific inferer (if set)
// 3. Global inferer (if set)
// 4. TypeUnknown (default).
//
// Results are cached to prevent infinite recursion when inferers
// call Type() on related errors.
// computeErrorType computes the ErrorType for this error without caching.
func (e *Error) computeErrorType() ErrorType {
	// 1. Use explicit type if set (and not TypeUnknown) - highest priority
	if e.errType != TypeUnknown {
		return e.errType
	}

	// 2. Use instance-specific inferer if set
	if e.typeInferer != nil {
		if typ := e.typeInferer(e); typ != TypeUnknown {
			return typ
		}
	}

	// 3. Try global inferer if no result yet
	return e.tryGlobalInferer()
}

// tryGlobalInferer attempts to use the global inferer to determine error type.
func (e *Error) tryGlobalInferer() ErrorType {
	infererMutex.RLock()
	inferer := globalInferer
	infererMutex.RUnlock()

	if inferer != nil {
		if typ := inferer(e); typ != TypeUnknown {
			return typ
		}
	}

	// Default to unknown
	return TypeUnknown
}

// handleRecursion handles the recursion case for Type() method.
func (e *Error) handleRecursion() ErrorType {
	if e.errType != TypeUnknown {
		return e.errType
	}
	return TypeUnknown
}

func (e *Error) Type() ErrorType {
	// Return cached result if available
	if e.computedErrType != "" {
		return e.computedErrType
	}

	// Prevent infinite recursion by checking if we're already computing
	if e.computing {
		return e.handleRecursion()
	}

	// Mark as computing to prevent recursion
	e.computing = true
	defer func() {
		e.computing = false
	}()

	// Compute the result
	result := e.computeErrorType()

	// Cache the result
	e.computedErrType = result
	return result
}

// Type extracts the ErrorType from a generic error.
// If the error is not an errorsx.Error, returns TypeUnknown.
//
// This function enables type checking for any error, including
// wrapped errors and errors from external libraries. It will
// use dynamic type inference if configured on the error.
func Type(err error) ErrorType {
	if e, ok := err.(*Error); ok {
		return e.Type()
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
			if e.Type() == typ {
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
