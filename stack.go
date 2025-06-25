package errorsx

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

const (
	// MaxStackFrames defines the maximum number of stack frames to capture
	// when creating a stack trace. This prevents excessive memory usage
	// while still providing sufficient debugging information.
	MaxStackFrames = 32
)

// StackTrace represents a captured call stack with an associated message.
// It stores the raw program counter values and a descriptive message
// about when/why the stack trace was captured.
//
// The stack trace can be formatted and displayed using various methods,
// and can be customized with StackTraceCleaner functions.
type StackTrace struct {
	// Frames contains the raw program counter values for each stack frame.
	Frames []uintptr

	// Msg is a descriptive message about when this stack trace was captured.
	Msg string
}

// StackTraceCleaner is a function type for customizing stack trace output.
// It receives a slice of formatted stack frame strings and returns a modified
// slice, allowing for filtering, formatting, or annotating stack traces.
//
// Example use cases:
//   - Filtering out internal framework code
//   - Highlighting application-specific code
//   - Adding source code context
//   - Anonymizing sensitive paths
//
// Example implementation:
//
//	cleaner := func(frames []string) []string {
//		var cleaned []string
//		for _, frame := range frames {
//			if !strings.Contains(frame, "internal/") {
//				cleaned = append(cleaned, frame)
//			}
//		}
//		return cleaned
//	}
type StackTraceCleaner func(frames []string) []string

// WithStack returns a copy of the error with a stack trace captured at the specified skip level.
// The skip parameter determines how many stack frames to skip when capturing the trace,
// allowing you to exclude wrapper functions from the stack trace.
//
// If the error already has a stack trace (isStacked is true), it returns the error unchanged
// to prevent duplicate stack traces in the same error chain.
//
// Parameters:
//   - skip: Number of stack frames to skip (0 = include WithStack call, 1 = exclude it)
//
// Example:
//
//	// Capture stack trace including this function call
//	err := errorsx.New("something.failed").WithStack(0)
//
//	// Capture stack trace excluding this function call
//	err := errorsx.New("something.failed").WithStack(1)
func (e *Error) WithStack(skip int) *Error {
	if e.isStacked {
		return e
	}

	clone := *e
	clone.stacks = append([]StackTrace{{Frames: callersWithSkip(skip), Msg: e.msg}}, clone.stacks...)
	clone.isStacked = true
	return &clone
}

// WithCallerStack returns a copy of the error with a stack trace captured from the caller's location.
// This is a convenience method equivalent to WithStack(1), which excludes the WithCallerStack
// call itself from the stack trace.
//
// This is the most commonly used method for adding stack traces, as it captures the
// location where the error was created or wrapped, not the location of the WithCallerStack call.
//
// Example:
//
//	func processUser(id string) error {
//		if user := findUser(id); user == nil {
//			return errorsx.New("user.not_found").WithCallerStack()
//			// Stack trace will point to this line, not inside WithCallerStack
//		}
//		return nil
//	}
func (e *Error) WithCallerStack() *Error {
	return e.WithStack(1)
}

// WithStackTraceCleaner returns a copy of the error with a custom stack trace cleaner function.
// The cleaner function will be applied when the stack trace is formatted for display,
// allowing for customization of the stack trace output.
//
// Example:
//
//	cleaner := func(frames []string) []string {
//		// Filter out Go runtime frames
//		var filtered []string
//		for _, frame := range frames {
//			if !strings.Contains(frame, "runtime.") {
//				filtered = append(filtered, frame)
//			}
//		}
//		return filtered
//	}
//
//	err := errorsx.New("db.connection_failed").
//		WithCallerStack().
//		WithStackTraceCleaner(cleaner)
func (e *Error) WithStackTraceCleaner(cleaner StackTraceCleaner) *Error {
	clone := *e
	clone.stackTraceCleaner = cleaner
	return &clone
}

// WithCause returns a copy of the error with the specified underlying cause.
// If the error doesn't already have a stack trace, this method automatically
// captures one to preserve the error's origin point.
//
// This method is essential for error chaining, allowing you to wrap lower-level
// errors with higher-level context while maintaining the full error chain.
//
// Parameters:
//   - cause: The underlying error that caused this error
//
// Example:
//
//	// Wrap a database error with business logic context
//	dbErr := db.Query("SELECT ...")
//	if dbErr != nil {
//		return errorsx.New("user.fetch_failed").
//			WithCause(dbErr).
//			WithMessage("Failed to fetch user data")
//	}
//
// The resulting error chain allows for:
//   - errors.Is(err, dbErr) // true
//   - errors.Unwrap(err) // returns dbErr
//   - Stack trace pointing to the WithCause call location
func (e *Error) WithCause(cause error) *Error {
	if e.isStacked {
		return e
	}

	clone := *e
	clone.cause = cause
	clone.stacks = append([]StackTrace{{Frames: callers(), Msg: e.msg}}, clone.stacks...)
	clone.isStacked = true

	// If the cause error is of type *Error, also keep its stack trace
	if causeErr, ok := cause.(*Error); ok && len(causeErr.stacks) > 0 {
		clone.stacks = append(clone.stacks, causeErr.stacks...)
	}

	return &clone
}

func callersWithSkip(skip int) []uintptr {
	const depth = MaxStackFrames
	var pcs [depth]uintptr
	n := runtime.Callers(3+skip, pcs[:])
	return pcs[:n]
}

func callers() []uintptr {
	return callersWithSkip(1)
}

// Stacks returns the stack traces associated with the error.
func (e *Error) Stacks() []StackTrace {
	return e.stacks
}

// RootCause returns the deepest error in the error chain.
// If an *Error with a cause is found, it follows the cause; otherwise, it unwraps.
// Returns the last error in the chain (the root cause).
func RootCause(err error) error {
	var last error
	for err != nil {
		last = err
		if e, ok := err.(*Error); ok && e.cause != nil {
			err = e.cause
		} else {
			err = errors.Unwrap(err)
		}
	}
	return last
}

// RootStackTrace returns the stack trace of the root cause error, if available.
func RootStackTrace(err error) string {
	for err != nil {
		if e, ok := err.(*Error); ok && len(e.stacks) > 0 {
			return formatStackTrace(e.stacks[len(e.stacks)-1])
		}
		err = errors.Unwrap(err)
	}
	return ""
}

// FullStackTrace returns the full stack trace chain for the error.
func FullStackTrace(err error) string {
	var b strings.Builder
	for err != nil {
		if e, ok := err.(*Error); ok {
			for i := len(e.stacks) - 1; i >= 0; i-- {
				fmt.Fprintf(&b, "\n--- stack (msg: %s) ---\n", e.stacks[i].Msg)
				b.WriteString(formatStackTrace(e.stacks[i]))
			}
		}
		err = errors.Unwrap(err)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatStackTrace(st StackTrace) string {
	return strings.Join(toStackTraceLines(st), "\n")
}

func toStackTraceLines(st StackTrace) []string {
	var s []string
	frames := runtime.CallersFrames(st.Frames)
	for {
		frame, more := frames.Next()
		s = append(s, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, trimFunction(frame.Function)))
		if !more {
			break
		}
	}
	return s
}
