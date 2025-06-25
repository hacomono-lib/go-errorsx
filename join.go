package errorsx

import (
	"strings"
)

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// Returns nil if all errors are nil.
// The error formats as the concatenation of the strings obtained
// by calling the Error method of each element with a separating "; ".
//
// This function is compatible with Go's standard errors.Join behavior
// and supports both errors.Is and errors.As for unwrapping.
//
// Example:
//
//	err1 := errorsx.New("validation.email")
//	err2 := errorsx.New("validation.password")
//	combined := errorsx.Join(err1, err2)
//	// combined.Error() returns "validation.email; validation.password"
func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{
		errs: make([]error, 0, n),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	var b strings.Builder
	for i, err := range e.errs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

func (e *joinError) Unwrap() []error {
	return e.errs
}
