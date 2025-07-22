package errorsx_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hacomono-lib/go-errorsx"
	"github.com/stretchr/testify/suite"
)

type RetryableSuite struct {
	suite.Suite
}

func (s *RetryableSuite) TestWithRetryable() {
	err := errorsx.New("network.timeout").WithRetryable()
	s.Require().True(err.IsRetryable())
	s.Require().Equal("network.timeout", err.Error())
}

func (s *RetryableSuite) TestWithRetryableOption() {
	err := errorsx.New("service.unavailable",
		errorsx.WithRetryable(),
		errorsx.WithHTTPStatus(503),
	)
	s.Require().True(err.IsRetryable())
	s.Require().Equal(503, err.HTTPStatus())
}

func (s *RetryableSuite) TestNewRetryable() {
	err := errorsx.NewRetryable("connection.timeout")
	s.Require().True(err.IsRetryable())
	s.Require().Equal("connection.timeout", err.Error())
}

func (s *RetryableSuite) TestIsRetryableFunction() {
	// Test with retryable error
	err1 := errorsx.New("network.error").WithRetryable()
	s.Require().True(errorsx.IsRetryable(err1))

	// Test with non-retryable error
	err2 := errorsx.New("validation.error")
	s.Require().False(errorsx.IsRetryable(err2))

	// Test with nil
	s.Require().False(errorsx.IsRetryable(nil))

	// Test with standard Go error
	err3 := errors.New("standard error")
	s.Require().False(errorsx.IsRetryable(err3))
}

func (s *RetryableSuite) TestIsRetryableWithWrappedError() {
	// Create a retryable error and wrap it
	baseErr := errorsx.New("database.connection_lost").WithRetryable()
	wrappedErr := errorsx.New("operation.failed").WithCause(baseErr)

	// The wrapped error itself is not retryable
	s.Require().False(wrappedErr.IsRetryable())

	// But the base error is still retryable when unwrapped
	var unwrappedErr *errorsx.Error
	s.Require().True(errors.As(wrappedErr, &unwrappedErr))
	if errors.Is(wrappedErr, baseErr) {
		s.Require().True(baseErr.IsRetryable())
	}
}

func (s *RetryableSuite) TestRetryableWithMessage() {
	err := errorsx.New("rate.limit.exceeded").
		WithRetryable().
		WithMessage(map[string]string{
			"en": "Rate limit exceeded. Please try again later.",
			"ja": "レート制限を超えました。後でもう一度お試しください。",
		}).
		WithHTTPStatus(429)

	s.Require().True(err.IsRetryable())
	s.Require().Equal(429, err.HTTPStatus())

	msg, ok := errorsx.Message[map[string]string](err)
	s.Require().True(ok)
	s.Require().Equal("Rate limit exceeded. Please try again later.", msg["en"])
}

func (s *RetryableSuite) TestRetryableWithType() {
	err := errorsx.New("temporary.failure").
		WithRetryable().
		WithType(errorsx.ErrorType("temporary"))

	s.Require().True(err.IsRetryable())
	s.Require().Equal(errorsx.ErrorType("temporary"), err.Type())
}

func (s *RetryableSuite) TestRetryableIsPreservedOnCopy() {
	// Test that retryable flag is preserved when creating variations
	err1 := errorsx.New("original").WithRetryable()
	err2 := err1.WithReason("Modified reason")
	err3 := err1.WithMessage("User friendly message")
	err4 := err1.WithType(errorsx.TypeValidation)

	s.Require().True(err1.IsRetryable())
	s.Require().True(err2.IsRetryable())
	s.Require().True(err3.IsRetryable())
	s.Require().True(err4.IsRetryable())
}

func (s *RetryableSuite) TestRetryableAndNotFoundCanCoexist() {
	// Test that an error can be both retryable and not found
	err := errorsx.New("resource.temporarily_unavailable").
		WithRetryable().
		WithNotFound().
		WithHTTPStatus(503)

	s.Require().True(err.IsRetryable())
	s.Require().True(err.IsNotFound())
	s.Require().True(errorsx.IsRetryable(err))
	s.Require().True(errorsx.IsNotFound(err))
}

func (s *RetryableSuite) TestRetryableJSONMarshal() {
	err := errorsx.New("rate.limit.exceeded").
		WithRetryable().
		WithHTTPStatus(429).
		WithMessage("Rate limit exceeded")

	jsonData, marshalErr := err.MarshalJSON()
	s.Require().NoError(marshalErr)

	var result map[string]interface{}
	s.Require().NoError(json.Unmarshal(jsonData, &result))

	s.Require().Equal("rate.limit.exceeded", result["id"])
	s.Require().Equal(true, result["is_retryable"])
	s.Require().Equal(float64(429), result["status"])
	s.Require().Equal("Rate limit exceeded", result["message_data"])
}

func (s *RetryableSuite) TestNonRetryableJSONMarshal() {
	// Test that non-retryable errors don't include is_retryable field
	err := errorsx.New("validation.error").
		WithHTTPStatus(400)

	jsonData, marshalErr := err.MarshalJSON()
	s.Require().NoError(marshalErr)

	var result map[string]interface{}
	s.Require().NoError(json.Unmarshal(jsonData, &result))

	// is_retryable should be omitted when false due to omitempty tag
	_, hasRetryable := result["is_retryable"]
	s.Require().False(hasRetryable, "is_retryable field should be omitted when false")
}

func TestRetryableSuite(t *testing.T) {
	suite.Run(t, new(RetryableSuite))
}
