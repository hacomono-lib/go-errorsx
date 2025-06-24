package errorsx_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/hacomono-lib/go-errorsx"
)

type JoinTestSuite struct {
	suite.Suite
}

func TestJoinSuite(t *testing.T) {
	suite.Run(t, new(JoinTestSuite))
}

func (s *JoinTestSuite) TestJoinWithNilErrors() {
	// Returns nil if all errors are nil
	s.Nil(errorsx.Join(nil, nil))
}

func (s *JoinTestSuite) TestJoinWithMixedErrors() {
	// Combines non-nil errors when some are nil
	err1 := errorsx.New("error1")
	err2 := errorsx.New("error2")
	err := errorsx.Join(err1, nil, err2)
	s.NotNil(err)
	s.Equal("error1; error2", err.Error())
}

func (s *JoinTestSuite) TestJoinWithErrorsIs() {
	err1 := errorsx.New("error1")
	err2 := errorsx.New("error2")
	joined := errorsx.Join(err1, err2)

	// Verify errors.Is behavior
	s.True(errors.Is(joined, err1))
	s.True(errors.Is(joined, err2))
	s.False(errors.Is(joined, errorsx.New("error3")))
}

func (s *JoinTestSuite) TestJoinWithErrorsAs() {
	customErr := errorsx.New("custom")
	standardErr := errors.New("standard")
	joined := errorsx.Join(customErr, standardErr)

	// Verify errors.As behavior
	var ce *errorsx.Error
	s.True(errors.As(joined, &ce))
	s.Equal("custom", ce.Error())
}

func (s *JoinTestSuite) TestJoinWithMultipleErrors() {
	// Test combining multiple errors
	err1 := errorsx.New("error 1")
	err2 := errorsx.New("error 2")
	err3 := errorsx.New("error 3")
	joined := errorsx.Join(err1, err2, err3)

	// Verify error message format
	s.Equal("error 1; error 2; error 3", joined.Error())

	// Verify errors.Is behavior
	s.True(errors.Is(joined, err1))
	s.True(errors.Is(joined, err2))
	s.True(errors.Is(joined, err3))

	// Verify errors.As behavior
	var ce *errorsx.Error
	s.True(errors.As(joined, &ce))
}

func (s *JoinTestSuite) TestJoinWithMixedErrorTypes_IsAndAs() {
	err1 := errorsx.New("custom error 1")
	stdErr := errors.New("standard error")
	valErr := errorsx.NewValidationError("validation.failed")
	valErr.AddFieldError("email", "validation.required")

	joined := errorsx.Join(err1, stdErr, valErr)

	// Check if each error is included using errors.Is
	s.True(errors.Is(joined, err1), "errors.Is should detect custom error")
	s.True(errors.Is(joined, stdErr), "errors.Is should detect standard error")
	s.True(errors.Is(joined, valErr), "errors.Is should detect ValidationError")

	// Check if type assertion is possible using errors.As
	var gotCustom *errorsx.Error
	s.True(errors.As(joined, &gotCustom), "errors.As should extract *Error")
	s.Equal(err1.Error(), gotCustom.Error())

	var gotVal *errorsx.ValidationError
	s.True(errors.As(joined, &gotVal), "errors.As should extract *ValidationError")
	s.Equal(valErr.Error(), gotVal.Error())
}
