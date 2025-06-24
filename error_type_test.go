package errorsx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/hacomono-lib/go-errorsx"
)

type ErrorTypeTestSuite struct {
	suite.Suite
}

func TestErrorTypeSuite(t *testing.T) {
	suite.Run(t, new(ErrorTypeTestSuite))
}

func (suite *ErrorTypeTestSuite) TestFilterByType() {
	// Test cases
	tests := []struct {
		name     string
		setup    func() error
		typ      errorsx.ErrorType
		expected int // expected number of errors
	}{
		{
			name: "Single error with matching type",
			setup: func() error {
				return errorsx.New("test.error").WithType(errorsx.ErrorType("test"))
			},
			typ:      errorsx.ErrorType("test"),
			expected: 1,
		},
		{
			name: "Single error with non-matching type",
			setup: func() error {
				return errorsx.New("test.error").WithType(errorsx.ErrorType("other"))
			},
			typ:      errorsx.ErrorType("test"),
			expected: 0,
		},
		{
			name: "Nested errors with matching types",
			setup: func() error {
				inner := errorsx.New("inner.error").WithType(errorsx.ErrorType("test"))
				return fmt.Errorf("outer.error: %w", inner)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 1,
		},
		{
			name: "Multiple errors with matching types",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				return errors.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 2,
		},
		{
			name: "Nil error",
			setup: func() error {
				return nil
			},
			typ:      errorsx.ErrorType("test"),
			expected: 0,
		},
		{
			name: "Mixed with standard error",
			setup: func() error {
				stdErr := errors.New("standard error")
				customErr := errorsx.New("custom.error").WithType(errorsx.ErrorType("test"))
				return errors.Join(stdErr, customErr)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 1,
		},
		{
			name: "Nested with standard error",
			setup: func() error {
				stdErr := errors.New("standard error")
				customErr := errorsx.New("custom.error").WithType(errorsx.ErrorType("test"))
				return fmt.Errorf("outer error: %w, %w", stdErr, customErr)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 1,
		},
		{
			name: "Using errorsx.Join",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				return errorsx.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 2,
		},
		{
			name: "Using errors.Join",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				return errors.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 2,
		},
		{
			name: "Mixed join errors",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				joined1 := errorsx.Join(err1, err2)
				err3 := errorsx.New("error3").WithType(errorsx.ErrorType("test"))
				return errors.Join(joined1, err3)
			},
			typ:      errorsx.ErrorType("test"),
			expected: 2,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			// Arrange
			err := tt.setup()

			// Act
			result := errorsx.FilterByType(err, tt.typ)

			// Assert
			assert.Len(t, result, tt.expected)
			for _, e := range result {
				assert.Equal(t, tt.typ, e.Type())
			}
		})
	}
}

func (suite *ErrorTypeTestSuite) TestHasType() {
	// Test cases
	tests := []struct {
		name     string
		setup    func() error
		typ      errorsx.ErrorType
		expected bool
	}{
		{
			name: "Error with matching type",
			setup: func() error {
				return errorsx.New("test.error").WithType(errorsx.ErrorType("test"))
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Error with non-matching type",
			setup: func() error {
				return errorsx.New("test.error").WithType(errorsx.ErrorType("other"))
			},
			typ:      errorsx.ErrorType("test"),
			expected: false,
		},
		{
			name: "Nested error with matching type",
			setup: func() error {
				inner := errorsx.New("inner.error").WithType(errorsx.ErrorType("test"))
				return fmt.Errorf("outer.error: %w", inner)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Multiple errors with matching type",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("other"))
				return errors.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Nil error",
			setup: func() error {
				return nil
			},
			typ:      errorsx.ErrorType("test"),
			expected: false,
		},
		{
			name: "Mixed with standard error",
			setup: func() error {
				stdErr := errors.New("standard error")
				customErr := errorsx.New("custom.error").WithType(errorsx.ErrorType("test"))
				return errors.Join(stdErr, customErr)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Nested with standard error",
			setup: func() error {
				stdErr := errors.New("standard error")
				customErr := errorsx.New("custom.error").WithType(errorsx.ErrorType("test"))
				return fmt.Errorf("outer error: %w, %w", stdErr, customErr)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Only standard errors",
			setup: func() error {
				err1 := errors.New("error1")
				err2 := errors.New("error2")
				return errors.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: false,
		},
		{
			name: "Using errorsx.Join",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				return errorsx.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Using errors.Join",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				return errors.Join(err1, err2)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
		{
			name: "Mixed join errors",
			setup: func() error {
				err1 := errorsx.New("error1").WithType(errorsx.ErrorType("test"))
				err2 := errorsx.New("error2").WithType(errorsx.ErrorType("test"))
				joined1 := errorsx.Join(err1, err2)
				err3 := errorsx.New("error3").WithType(errorsx.ErrorType("test"))
				return errors.Join(joined1, err3)
			},
			typ:      errorsx.ErrorType("test"),
			expected: true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			// Arrange
			err := tt.setup()

			// Act
			result := errorsx.HasType(err, tt.typ)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}
