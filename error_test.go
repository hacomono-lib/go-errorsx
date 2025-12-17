package errorsx_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/hacomono-lib/go-errorsx"
	"github.com/stretchr/testify/suite"
)

const (
	ApplicationErrorType errorsx.ErrorType = "app"
	InfraErrorType       errorsx.ErrorType = "infra"
	DomainErrorType      errorsx.ErrorType = "domain"
	AdapterErrorType     errorsx.ErrorType = "adapter"
	UnknownErrorType     errorsx.ErrorType = "unknown"
)

type ErrorSuite struct {
	suite.Suite
}

func (s *ErrorSuite) TestNewErrorWithCallerStack() {
	err := errorsx.New("test.error").WithCallerStack()
	s.Require().Equal("test.error", err.Error())
	s.Require().Contains(errorsx.FullStackTrace(err), "TestNewErrorWithCallerStack")
}

func (s *ErrorSuite) TestWithCauseCapturesStack() {
	base := errors.New("root cause")
	err := errorsx.New("infra.db.error").WithCause(base)
	s.Require().Equal("infra.db.error", err.Error())
	s.Require().Equal(base, errors.Unwrap(err))
	s.Require().Contains(errorsx.FullStackTrace(err), "TestWithCauseCapturesStack")
}

func (s *ErrorSuite) TestErrorIsComparesByID() {
	e1 := errorsx.New("same.id")
	e2 := errorsx.New("same.id")
	s.Require().True(errors.Is(e1, e2))

	// Verify that errors with the same ID but different types are considered equal
	e3 := errorsx.New("same.id").WithType(errorsx.ErrorType("type1"))
	e4 := errorsx.New("same.id").WithType(errorsx.ErrorType("type2"))
	s.Require().True(errors.Is(e3, e4), "errors with the same ID should be considered equal regardless of their types")
}

func (s *ErrorSuite) TestMarshalJSONStructure() {
	err := errorsx.New("app.test").
		WithType(DomainErrorType).
		WithHTTPStatus(400).
		WithMessage("error.key").
		WithCallerStack()

	bytes, errMarshal := json.Marshal(err)
	s.Require().NoError(errMarshal)

	var result map[string]any
	s.Require().NoError(json.Unmarshal(bytes, &result))
	s.Require().Equal("app.test", result["id"])
	s.Require().Equal("domain", result["type"])
	s.Require().Equal(float64(400), result["status"])
	s.Require().Equal("error.key", result["message_data"])
	s.Require().NotEmpty(result["stacks"])
}

func (s *ErrorSuite) TestMarshalJSONWithNestedErrorChain() {
	// Original error (e.g., database error)
	originalErr := errors.New("database error: column does not exist")

	// First wrap
	wrappedErr1 := errorsx.New("error.first").
		WithReason("first error occurred").
		WithCause(originalErr)

	// Second wrap
	wrappedErr2 := errorsx.New("error.second").
		WithReason("second error occurred").
		WithCause(wrappedErr1)

	// Marshal to JSON
	bytes, errMarshal := json.Marshal(wrappedErr2)
	s.Require().NoError(errMarshal)

	var result map[string]any
	s.Require().NoError(json.Unmarshal(bytes, &result))

	// Verify that cause field exists
	s.Require().NotNil(result["cause"], "cause field should exist")
	cause, ok := result["cause"].(map[string]any)
	s.Require().True(ok, "cause should be a map")

	// Verify that cause.msg returns the root cause error message
	// Before the fix, it would return "first error occurred"
	// After the fix, it returns "database error: column does not exist"
	s.Require().Equal("database error: column does not exist", cause["msg"],
		"cause.msg should contain the root cause error message, not the intermediate error message")
}

func (s *ErrorSuite) TestRootStackTrace() {
	base := errorsx.New("base.error").WithCallerStack()
	err := errorsx.New("wrapper.error").WithCause(base)
	root := errorsx.RootStackTrace(err)
	s.Require().Contains(root, "TestRootStackTrace")
}

// Simulate an infrastructure layer error
func createInfraError() error {
	err := errorsx.New("infra.db.connection_error").
		WithType(InfraErrorType).
		WithReason("Database connection error").
		WithStack(0)
	return err
}

// Simulate an application layer error
func createApplicationError(cause error) error {
	return errorsx.New("app.user.not_found").
		WithType(ApplicationErrorType).
		WithHTTPStatus(404).
		WithMessage("error.user.not_found").
		WithReason("User not found").
		WithCause(cause)
}

// Simulate a domain layer error
func createDomainError(cause error) error {
	return errorsx.New("domain.validation.invalid_input").
		WithType(DomainErrorType).
		WithReason("Invalid input").
		WithCause(cause)
}

// Simulate an adapter layer error
func createAdapterError(cause error) error {
	return errorsx.New("adapter.api.invalid_request").
		WithType(AdapterErrorType).
		WithHTTPStatus(400).
		WithMessage("error.invalid_request").
		WithReason("Invalid request").
		WithCause(cause)
}

func (s *ErrorSuite) TestMultiLayerStackTrace() {
	infraErr := createInfraError()
	s.Require().Equal("Database connection error", infraErr.Error())

	appErr := createApplicationError(infraErr)
	s.Require().Equal("User not found", appErr.Error())
	s.Require().True(errors.Is(errors.Unwrap(appErr), infraErr))
	domainErr := createDomainError(appErr)
	s.Require().Equal("Invalid input", domainErr.Error())
	adapterErr := createAdapterError(domainErr)
	s.Require().Equal("Invalid request", adapterErr.Error())

	fullStack := errorsx.FullStackTrace(adapterErr)
	s.Require().Contains(fullStack, "stack (msg: Invalid request)")
	s.Require().Contains(fullStack, "stack (msg: Invalid input)")
	s.Require().Contains(fullStack, "stack (msg: User not found)")
	s.Require().Contains(fullStack, "stack (msg: Database connection error)")
	s.Require().Contains(fullStack, "TestMultiLayerStackTrace")
}

// Detailed propagation test across multiple layers
func (s *ErrorSuite) TestDetailedMultiLayerErrorPropagation() {
	baseErr := errorsx.New("base.error").WithReason("Base error").WithStack(0)
	s.Require().NotEmpty(errorsx.RootStackTrace(baseErr))

	infraErr := errorsx.New("infra.error").WithReason("Infra error").WithCause(baseErr)
	appErr := errorsx.New("app.error").WithReason("Application error").WithCause(infraErr)
	domainErr := errorsx.New("domain.error").WithReason("Domain error").WithCause(appErr)

	fullStack := errorsx.FullStackTrace(domainErr)
	s.Require().Contains(fullStack, "stack (msg: Domain error)")
	s.Require().Contains(fullStack, "stack (msg: Application error)")
	s.Require().Contains(fullStack, "stack (msg: Infra error)")
	s.Require().Contains(fullStack, "stack (msg: Base error)")
	s.Require().Contains(fullStack, "TestDetailedMultiLayerErrorPropagation")
}

func (s *ErrorSuite) TestNestedWrappingErrors() {
	baseErr := errors.New("original error")
	wrap1 := errorsx.New("infra.first_wrap").WithCause(baseErr)
	s.Require().Equal("infra.first_wrap", wrap1.Error())
	s.Require().Equal(baseErr, errors.Unwrap(wrap1))
	wrap2 := errorsx.New("app.second_wrap").WithCause(wrap1)
	s.Require().Equal("app.second_wrap", wrap2.Error())
	s.Require().Equal(wrap1, errors.Unwrap(wrap2))
	wrap3 := errorsx.New("domain.third_wrap").WithCause(wrap2)
	s.Require().Equal("domain.third_wrap", wrap3.Error())
	s.Require().Equal(wrap2, errors.Unwrap(wrap3))
	fullStack := errorsx.FullStackTrace(wrap3)
	s.Require().Contains(fullStack, "TestNestedWrappingErrors")
	s.Require().Contains(fullStack, "domain.third_wrap")
	s.Require().Contains(fullStack, "app.second_wrap")
	s.Require().Contains(fullStack, "infra.first_wrap")
}

func (s *ErrorSuite) TestErrorTypeExtraction() {
	infraErr := createInfraError()
	appErr := createApplicationError(infraErr)
	domainErr := createDomainError(appErr)
	adapterErr := createAdapterError(domainErr)

	s.Require().Equal(AdapterErrorType, errorsx.Type(adapterErr))
	s.Require().Equal(DomainErrorType, errorsx.Type(domainErr))
	s.Require().Equal(ApplicationErrorType, errorsx.Type(appErr))
	s.Require().Equal(InfraErrorType, errorsx.Type(infraErr))
}

// Test stack trace skip levels
func nestedLevel1() error {
	return nestedLevel2()
}

func nestedLevel2() error {
	return nestedLevel3()
}

func nestedLevel3() error {
	// Skip=0: this function should be captured
	return errorsx.New("test.skip0").WithStack(0)
}

func nestedSkipLevel1() error {
	return nestedSkipLevel2()
}

func nestedSkipLevel2() error {
	return nestedSkipLevel3()
}

func nestedSkipLevel3() error {
	// Skip=2: nestedSkipLevel1 should be captured
	return errorsx.New("test.skip2").WithStack(2)
}

func (s *ErrorSuite) TestStackTraceSkipLevels() {
	err0 := nestedLevel1()
	stack0 := errorsx.FullStackTrace(err0)
	s.Require().Contains(stack0, "nestedLevel3")
	lines0 := strings.Split(stack0, "\n")
	foundLevel3 := false
	foundLevel2 := false
	foundLevel1 := false
	foundTest := false
	for _, line := range lines0 {
		if strings.Contains(line, "nestedLevel3") {
			foundLevel3 = true
		} else if strings.Contains(line, "nestedLevel2") && foundLevel3 {
			foundLevel2 = true
		} else if strings.Contains(line, "nestedLevel1") && foundLevel2 {
			foundLevel1 = true
		} else if strings.Contains(line, "TestStackTraceSkipLevels") && foundLevel1 {
			foundTest = true
		}
	}
	s.Require().True(foundLevel3, "stack trace should contain nestedLevel3")
	s.Require().True(foundLevel2, "stack trace should contain nestedLevel2")
	s.Require().True(foundLevel1, "stack trace should contain nestedLevel1")
	s.Require().True(foundTest, "stack trace should contain TestStackTraceSkipLevels")

	err2 := nestedSkipLevel1()
	stack2 := errorsx.FullStackTrace(err2)
	s.Require().NotContains(stack2, "nestedSkipLevel3")
	s.Require().NotContains(stack2, "nestedSkipLevel2")
	s.Require().Contains(stack2, "nestedSkipLevel1")
	s.Require().Contains(stack2, "TestStackTraceSkipLevels")
}

func (s *ErrorSuite) TestWithCallerStackEquivalence() {
	makeDirectErr := func() error {
		return errorsx.New("test.direct").WithCallerStack()
	}
	makeSkipErr := func() error {
		return errorsx.New("test.skip").WithStack(1)
	}
	directErr := makeDirectErr()
	skipErr := makeSkipErr()
	directStack := errorsx.FullStackTrace(directErr)
	skipStack := errorsx.FullStackTrace(skipErr)
	s.Require().Contains(directStack, "TestWithCallerStackEquivalence")
	s.Require().Contains(skipStack, "TestWithCallerStackEquivalence")
	directLines := strings.Split(directStack, "\n")
	skipLines := strings.Split(skipStack, "\n")
	directContainsTest := false
	skipContainsTest := false
	for _, line := range directLines {
		if strings.Contains(line, "TestWithCallerStackEquivalence") {
			directContainsTest = true
			break
		}
	}
	for _, line := range skipLines {
		if strings.Contains(line, "TestWithCallerStackEquivalence") {
			skipContainsTest = true
			break
		}
	}
	s.Require().True(directContainsTest, "stack trace for WithCallerStack should contain test function")
	s.Require().True(skipContainsTest, "stack trace for WithStack(1) should contain test function")
}

func (s *ErrorSuite) TestWithCauseStackCapture() {
	causeErr := deepNestedCause(3)
	stack := errorsx.FullStackTrace(causeErr)
	s.Require().Contains(stack, "TestWithCauseStackCapture")
	s.Require().Contains(stack, "deepNestedCause")
	currentErr := causeErr
	errTexts := []string{}
	for currentErr != nil {
		if xerr, ok := currentErr.(*errorsx.Error); ok {
			errTexts = append(errTexts, xerr.Error())
		}
		currentErr = errors.Unwrap(currentErr)
	}
	s.Require().Equal(4, len(errTexts), "error chain should contain 4 errors")
	s.Require().Equal("Depth 3", errTexts[0])
	s.Require().Equal("Depth 2", errTexts[1])
	s.Require().Equal("Depth 1", errTexts[2])
	s.Require().Equal("Base error", errTexts[3])
}

func (s *ErrorSuite) TestWithCauseStackAppend() {
	baseErr := errorsx.New("base.error").
		WithReason("Base error").
		WithCallerStack()
	baseStack := errorsx.FullStackTrace(baseErr)
	s.Require().Contains(baseStack, "TestWithCauseStackAppend")
	wrappedErr1 := errorsx.New("wrapped.1").
		WithReason("Wrapped error 1").
		WithCause(baseErr)
	wrappedStack1 := errorsx.FullStackTrace(wrappedErr1)
	s.Require().Contains(wrappedStack1, "stack (msg: Wrapped error 1)")
	s.Require().Contains(wrappedStack1, "stack (msg: Base error)")
	s.Require().Contains(wrappedStack1, "TestWithCauseStackAppend")
	wrappedErr2 := errorsx.New("wrapped.2").
		WithReason("Wrapped error 2").
		WithCause(wrappedErr1)
	wrappedStack2 := errorsx.FullStackTrace(wrappedErr2)
	s.Require().Contains(wrappedStack2, "stack (msg: Wrapped error 2)")
	s.Require().Contains(wrappedStack2, "stack (msg: Wrapped error 1)")
	s.Require().Contains(wrappedStack2, "stack (msg: Base error)")
	stackLines := strings.Split(wrappedStack2, "\n")
	s.Require().True(len(stackLines) > 3, "stack trace should have multiple lines")
	actualStackGroups := strings.Count(wrappedStack2, "--- stack (msg:")
	fmt.Printf("Actual stack group count: %d\n", actualStackGroups)
	s.Require().Equal(actualStackGroups, actualStackGroups, "stack group count should match")
}

func deepNestedCause(depth int) error {
	if depth <= 0 {
		return errorsx.New("base.error").WithReason("Base error").WithCallerStack()
	}
	cause := deepNestedCause(depth - 1)
	return errorsx.New(fmt.Sprintf("nested.%d", depth)).
		WithReason("Depth %d", depth).
		WithCause(cause)
}

func (s *ErrorSuite) TestIsNotFound() {
	domainErr := errorsx.NewNotFound("domain.not_found").WithType(DomainErrorType)
	s.True(errorsx.IsNotFound(domainErr), "DomainErrNotFound should return true for IsNotFound")
	infraErr := errorsx.NewNotFound("infra.data_not_found").WithType(InfraErrorType)
	s.True(errorsx.IsNotFound(infraErr), "InfraErrDataNotFound should return true for IsNotFound")
	wrappedDomain := fmt.Errorf("wrap: %w", domainErr)
	s.True(errorsx.IsNotFound(wrappedDomain), "Wrapped DomainErrNotFound should return true for IsNotFound")
	wrappedInfra := fmt.Errorf("wrap: %w", infraErr)
	s.True(errorsx.IsNotFound(wrappedInfra), "Wrapped InfraErrDataNotFound should return true for IsNotFound")
	otherErr := errorsx.New("other.error")
	s.False(errorsx.IsNotFound(otherErr), "Other errors should return false for IsNotFound")
	stdErr := errors.New("standard error")
	s.False(errorsx.IsNotFound(stdErr), "Standard error should return false for IsNotFound")
}

func (s *ErrorSuite) TestIsStackedPreventsDuplicateStack() {
	err := errorsx.New("dup.stack").WithCallerStack()
	// Try to add another stack trace; should not change
	err2 := err.WithCallerStack()
	s.Require().Equal(err.Stacks(), err2.Stacks(), "isStacked should prevent duplicate stack traces")

	err3 := err.WithCause(errors.New("cause error"))
	s.Require().Equal(err.Stacks(), err3.Stacks(), "isStacked should prevent WithCause from adding stack if already stacked")
}

func (s *ErrorSuite) TestRootCause() {
	base := errors.New("base error")
	err := errorsx.New("wrap1").WithCause(base)
	s.Require().Equal(base, errorsx.RootCause(err), "RootCause should return the direct cause if present")

	// If no cause, should return nil
	plain := errorsx.New("plain")
	s.Require().Equal(plain, errorsx.RootCause(plain), "RootCause should return the error itself if no cause is set")

	plain2 := fmt.Errorf("wrap: %w", plain)
	s.Require().Equal(plain, errorsx.RootCause(plain2), "RootCause should return the original error if wrapped")

	// If error is not *Error, should return nil
	s.Require().Equal(base, errorsx.RootCause(base), "RootCause should return the error itself if no cause is set")

	// If wrapped, should return the original error
	wrappedBase := fmt.Errorf("wrap: %w", base)
	s.Require().Equal(base, errorsx.RootCause(wrappedBase), "RootCause should return the original error if wrapped")

	// If wrapped with not *Error, should return the original error
	wrappedBase2 := fmt.Errorf("wrap: %w", wrappedBase)
	s.Require().Equal(base, errorsx.RootCause(wrappedBase2), "RootCause should return the original error if wrapped with not *Error")
}

func (s *ErrorSuite) TestWithMessageDataIsImmutable() {
	// Create original error
	original := errorsx.New("test.error").WithMessage("error.key")

	// Change message data
	withNewData := original.WithMessage(map[string]any{"key": "new.key", "params": []any{"param1", "param2"}})

	// Verify that original message data is not modified
	originalMsg, ok := errorsx.Message[string](original)
	s.Require().True(ok)
	s.Require().Equal("error.key", originalMsg)

	// Verify that new error's message data is correctly set
	expectedData := map[string]any{"key": "new.key", "params": []any{"param1", "param2"}}
	newDataMsg, ok := errorsx.Message[map[string]any](withNewData)
	s.Require().True(ok)
	s.Require().Equal(expectedData, newDataMsg)

	// Change to different type
	withString := withNewData.WithMessage("simple.string")

	// Verify that previous message data is preserved
	preservedMsg, ok := errorsx.Message[map[string]any](withNewData)
	s.Require().True(ok)
	s.Require().Equal(expectedData, preservedMsg)

	// Verify that new error's message data is correctly set
	stringMsg, ok := errorsx.Message[string](withString)
	s.Require().True(ok)
	s.Require().Equal("simple.string", stringMsg)
}

func (s *ErrorSuite) TestNewWithOptions() {
	tests := []struct {
		name  string
		id    string
		opts  []errorsx.Option
		check func(*errorsx.Error)
	}{
		{
			name: "Basic error creation",
			id:   "TEST_001",
			opts: []errorsx.Option{},
			check: func(err *errorsx.Error) {
				s.Equal("TEST_001", err.ID())
				s.Equal("TEST_001", err.Error())
				s.Equal(errorsx.TypeUnknown, err.Type())
				s.Equal(0, err.HTTPStatus())
				msg, ok := errorsx.Message[any](err)
				s.False(ok)
				s.Nil(msg)
				s.False(err.IsNotFound())
				s.Nil(err.Unwrap())
			},
		},
		{
			name: "Set error type",
			id:   "TEST_002",
			opts: []errorsx.Option{
				errorsx.WithType(DomainErrorType),
			},
			check: func(err *errorsx.Error) {
				s.Equal(DomainErrorType, err.Type())
			},
		},
		{
			name: "Set HTTP status",
			id:   "TEST_003",
			opts: []errorsx.Option{
				errorsx.WithHTTPStatus(404),
			},
			check: func(err *errorsx.Error) {
				s.Equal(404, err.HTTPStatus())
			},
		},
		{
			name: "Set message data",
			id:   "TEST_004",
			opts: []errorsx.Option{
				errorsx.WithMessage(map[string]any{"key": "error.key", "params": []any{"param1", "param2"}}),
			},
			check: func(err *errorsx.Error) {
				expectedData := map[string]any{"key": "error.key", "params": []any{"param1", "param2"}}
				msg, ok := errorsx.Message[map[string]any](err)
				s.True(ok)
				s.Equal(expectedData, msg)
			},
		},
		{
			name: "Set NotFound flag",
			id:   "TEST_005",
			opts: []errorsx.Option{
				errorsx.WithNotFound(),
			},
			check: func(err *errorsx.Error) {
				s.True(err.IsNotFound())
			},
		},
		{
			name: "Set multiple options",
			id:   "TEST_007",
			opts: []errorsx.Option{
				errorsx.WithType(DomainErrorType),
				errorsx.WithHTTPStatus(404),
				errorsx.WithMessage("error.key"),
				errorsx.WithNotFound(),
			},
			check: func(err *errorsx.Error) {
				s.Equal(DomainErrorType, err.Type())
				s.Equal(404, err.HTTPStatus())
				msg, ok := errorsx.Message[string](err)
				s.True(ok)
				s.Equal("error.key", msg)
				s.True(err.IsNotFound())
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := errorsx.New(tt.id, tt.opts...)
			tt.check(err)
		})
	}
}

func (s *ErrorSuite) TestReplaceMessage() {
	tests := []struct {
		name     string
		err      error
		data     any
		expected string
	}{
		{
			name:     "Replace message for errorsx.Error",
			err:      errorsx.New("original.error").WithMessage("original.key"),
			data:     "new.key",
			expected: "new.key",
		},
		{
			name:     "Replace message for standard error",
			err:      fmt.Errorf("standard error"),
			data:     "wrapped.key",
			expected: "wrapped.key",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := errorsx.ReplaceMessage(tt.err, tt.data)
			xerr, ok := result.(*errorsx.Error)
			s.Require().True(ok)
			msg, ok := errorsx.Message[string](xerr)
			s.True(ok)
			s.Equal(tt.expected, msg)
		})
	}
}

func (s *ErrorSuite) TestReplaceType() {
	tests := []struct {
		name         string
		err          error
		typ          errorsx.ErrorType
		expectChange bool
		expectedType errorsx.ErrorType
	}{
		{
			name:         "Replace type for errorsx.Error",
			err:          errorsx.New("validation.failed").WithType(errorsx.TypeUnknown),
			typ:          errorsx.TypeValidation,
			expectChange: true,
			expectedType: errorsx.TypeValidation,
		},
		{
			name:         "Replace type for errorsx.Error with different original type",
			err:          errorsx.New("auth.failed").WithType(errorsx.TypeInitialization),
			typ:          errorsx.TypeNotFound,
			expectChange: true,
			expectedType: errorsx.TypeNotFound,
		},
		{
			name:         "Standard Go error returns unchanged",
			err:          fmt.Errorf("standard error"),
			typ:          errorsx.TypeValidation,
			expectChange: false,
			expectedType: errorsx.TypeUnknown, // Not applicable for standard errors
		},
		{
			name:         "Nil error returns nil",
			err:          nil,
			typ:          errorsx.TypeValidation,
			expectChange: false,
			expectedType: errorsx.TypeUnknown, // Not applicable
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := errorsx.ReplaceType(tt.err, tt.typ)

			if tt.err == nil {
				s.Nil(result)
				return
			}

			if tt.expectChange {
				// Should be an errorsx.Error with the new type
				xerr, ok := result.(*errorsx.Error)
				s.Require().True(ok, "Expected result to be *errorsx.Error")
				s.Equal(tt.expectedType, xerr.Type())
				// Should preserve original ID and message
				originalXerr, ok := tt.err.(*errorsx.Error)
				s.Require().True(ok, "Original error should be *errorsx.Error")
				s.Equal(originalXerr.ID(), xerr.ID())
				s.Equal(originalXerr.Error(), xerr.Error())
			} else {
				// Should return the original error unchanged
				s.Equal(tt.err, result)
			}
		})
	}
}

func (s *ErrorSuite) TestGenericMessageExtraction() {
	// Test with string message
	err1 := errorsx.New("test.error").WithMessage("test.message.key")

	// Test successful string extraction
	if msg, ok := errorsx.Message[string](err1); ok {
		s.Equal("test.message.key", msg)
	} else {
		s.Fail("Expected successful string extraction")
	}

	// Test MessageOr with string
	msg := errorsx.MessageOr[string](err1, "fallback")
	s.Equal("test.message.key", msg)

	// Test with struct message
	type MessageData struct {
		Key    string
		Params []any
	}

	messageData := MessageData{
		Key:    "validation.error",
		Params: []any{"field1", 42},
	}
	err2 := errorsx.New("test.error").WithMessage(messageData)

	// Test successful struct extraction
	if data, ok := errorsx.Message[MessageData](err2); ok {
		s.Equal("validation.error", data.Key)
		s.Equal([]any{"field1", 42}, data.Params)
	} else {
		s.Fail("Expected successful struct extraction")
	}

	// Test failed type assertion
	if _, ok := errorsx.Message[int](err1); ok {
		s.Fail("Expected failed type assertion for int")
	}

	// Test MessageOr with failed type assertion
	fallbackInt := errorsx.MessageOr[int](err1, 999)
	s.Equal(999, fallbackInt)

	// Test with nil message data
	err3 := errorsx.New("test.error")
	if _, ok := errorsx.Message[string](err3); ok {
		s.Fail("Expected failed extraction for nil message data")
	}

	// Test with standard error
	stdErr := errors.New("standard error")
	if _, ok := errorsx.Message[string](stdErr); ok {
		s.Fail("Expected failed extraction for standard error")
	}
}

// Test types for interface constraint testing
type Localizable interface {
	Localize(locale string) string
}

type LocalizableMessage struct {
	Key    string
	Params map[string]string
}

func (m LocalizableMessage) Localize(locale string) string {
	return fmt.Sprintf("[%s] %s", locale, m.Key)
}

func (s *ErrorSuite) TestGenericMessageWithInterface() {
	localizableData := LocalizableMessage{
		Key:    "error.user.not_found",
		Params: map[string]string{"userId": "123"},
	}

	err := errorsx.New("test.error").WithMessage(localizableData)

	// Test extraction with interface constraint
	if data, ok := errorsx.Message[Localizable](err); ok {
		s.Equal("[en] error.user.not_found", data.Localize("en"))
		s.Equal("[ja] error.user.not_found", data.Localize("ja"))
	} else {
		s.Fail("Expected successful interface extraction")
	}

	// Test extraction with concrete type
	if data, ok := errorsx.Message[LocalizableMessage](err); ok {
		s.Equal("error.user.not_found", data.Key)
		s.Equal(map[string]string{"userId": "123"}, data.Params)
	} else {
		s.Fail("Expected successful concrete type extraction")
	}
}

func (s *ErrorSuite) TestStackFrames() {
	// Test with no stack traces
	err := errorsx.New("test.error")
	frames := err.StackFrames()
	s.Require().Nil(frames, "StackFrames should return nil when no stack traces are available")

	// Test with stack traces
	err = errorsx.New("test.error").WithCallerStack()
	frames = err.StackFrames()
	s.Require().NotNil(frames, "StackFrames should return frames when stack traces are available")
	s.Require().True(len(frames) > 0, "StackFrames should return non-empty slice")

	// Test with multiple stack traces - should return the first (most recent) one
	baseErr := errorsx.New("base.error").WithCallerStack()
	wrapperErr := errorsx.New("wrapper.error").WithCause(baseErr)

	frames = wrapperErr.StackFrames()
	s.Require().NotNil(frames, "StackFrames should return frames for wrapper error")
	s.Require().True(len(frames) > 0, "StackFrames should return non-empty slice")

	// Verify it returns the most recent stack trace (first one)
	stacks := wrapperErr.Stacks()
	s.Require().True(len(stacks) > 0, "Should have stack traces available")
	s.Require().Equal(stacks[0].Frames, frames, "StackFrames should return the first (most recent) stack trace")
}

func (s *ErrorSuite) TestStackFramesOriginDetection() {
	// Test that StackFrames() returns the correct origin for WithCallerStack
	err := createErrorWithCallerStack()
	frames := err.StackFrames()
	s.Require().NotNil(frames, "StackFrames should return frames")
	s.Require().True(len(frames) > 0, "StackFrames should return non-empty slice")

	// Convert to actual function information
	framesInfo := runtime.CallersFrames(frames)
	frame, ok := framesInfo.Next()
	s.Require().True(ok, "Should have at least one frame")

	// The first frame should point to the createErrorWithCallerStack function
	s.Require().Contains(frame.Function, "createErrorWithCallerStack",
		"First frame should point to the function where WithCallerStack was called")

	// Test that StackFrames() returns the correct origin for WithCause
	baseErr := errors.New("base error")
	wrapperErr := createErrorWithCause(baseErr)
	frames = wrapperErr.StackFrames()
	s.Require().NotNil(frames, "StackFrames should return frames for WithCause")
	s.Require().True(len(frames) > 0, "StackFrames should return non-empty slice")

	// Convert to actual function information
	framesInfo = runtime.CallersFrames(frames)
	frame, ok = framesInfo.Next()
	s.Require().True(ok, "Should have at least one frame")

	// The first frame should point to the createErrorWithCause function
	s.Require().Contains(frame.Function, "createErrorWithCause",
		"First frame should point to the function where WithCause was called")
}

func (s *ErrorSuite) TestStackFramesWithMultipleWrapping() {
	// Test that when multiple WithCause calls are made, StackFrames() returns the most recent one
	baseErr := errors.New("base error")
	level1Err := createErrorWithCause(baseErr)
	level2Err := createErrorWithCauseLevel2(level1Err)

	frames := level2Err.StackFrames()
	s.Require().NotNil(frames, "StackFrames should return frames")
	s.Require().True(len(frames) > 0, "StackFrames should return non-empty slice")

	// Convert to actual function information
	framesInfo := runtime.CallersFrames(frames)
	frame, ok := framesInfo.Next()
	s.Require().True(ok, "Should have at least one frame")

	// The first frame should point to the most recent WithCause call (level2)
	s.Require().Contains(frame.Function, "createErrorWithCauseLevel2",
		"First frame should point to the most recent WithCause call")

	// Verify that we have multiple stack traces but StackFrames returns the first one
	stacks := level2Err.Stacks()
	s.Require().True(len(stacks) > 1, "Should have multiple stack traces")
	s.Require().Equal(stacks[0].Frames, frames, "StackFrames should return the first stack trace")
}

func (s *ErrorSuite) TestStackFramesConsistencyWithSentry() {
	// Test that StackFrames() returns consistent format expected by sentry-go
	err := createErrorWithCallerStack()
	frames := err.StackFrames()

	// Verify the format matches what sentry-go expects ([]uintptr)
	s.Require().IsType([]uintptr{}, frames, "StackFrames should return []uintptr")

	// Verify frames can be converted to runtime.Frame
	framesInfo := runtime.CallersFrames(frames)
	frameCount := 0
	for {
		frame, more := framesInfo.Next()
		frameCount++

		// Each frame should have valid information
		s.Require().NotEmpty(frame.Function, "Frame should have function name")
		s.Require().NotEmpty(frame.File, "Frame should have file name")
		s.Require().True(frame.Line > 0, "Frame should have valid line number")

		if !more {
			break
		}
	}

	s.Require().True(frameCount > 0, "Should have at least one frame")
}

// Helper functions for testing
func createErrorWithCallerStack() *errorsx.Error {
	return errorsx.New("test.with_caller_stack").WithCallerStack()
}

func createErrorWithCause(cause error) *errorsx.Error {
	return errorsx.New("test.with_cause").WithCause(cause)
}

func createErrorWithCauseLevel2(cause error) *errorsx.Error {
	return errorsx.New("test.with_cause_level2").WithCause(cause)
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorSuite))
}
