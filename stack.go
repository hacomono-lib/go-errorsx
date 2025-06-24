package errorsx

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

const (
	MaxStackFrames = 32
)

// StackTrace represents a captured stack trace with a message.
type StackTrace struct {
	Frames []uintptr
	Msg    string
}

// StackTraceCleaner is a function type for customizing stack trace output.
type StackTraceCleaner func(frames []string) []string

// WithStack returns a copy of the error with a stack trace captured at the given skip level.
func (e *Error) WithStack(skip int) *Error {
	if e.isStacked {
		return e
	}

	clone := *e
	clone.stacks = append([]StackTrace{{Frames: callersWithSkip(skip), Msg: e.msg}}, clone.stacks...)
	clone.isStacked = true
	return &clone
}

// WithCallerStack returns a copy of the error with a stack trace captured from the caller.
func (e *Error) WithCallerStack() *Error {
	return e.WithStack(1)
}

// WithStackTraceCleaner returns a copy of the error with a custom stack trace cleaner function.
func (e *Error) WithStackTraceCleaner(cleaner StackTraceCleaner) *Error {
	clone := *e
	clone.stackTraceCleaner = cleaner
	return &clone
}

// WithCause returns a copy of the error with the given cause and captures a stack trace if not already stacked.
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
