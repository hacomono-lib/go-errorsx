package errorsx_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hacomono-lib/go-errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ValidationErrorTestSuite struct {
	suite.Suite
}

func TestValidationErrorSuite(t *testing.T) {
	suite.Run(t, new(ValidationErrorTestSuite))
}

func (suite *ValidationErrorTestSuite) TestNewValidationError() {
	// Arrange
	idOrMsg := "validation.failed"

	// Act
	validationErr := errorsx.NewValidationError(idOrMsg)

	// Assert
	assert.NotNil(suite.T(), validationErr)
	assert.NotNil(suite.T(), validationErr.BaseError)
	assert.Empty(suite.T(), validationErr.FieldErrors)
	assert.Equal(suite.T(), errorsx.TypeValidation, validationErr.BaseError.Type())

	// Test with generic Message function
	if key, ok := errorsx.Message[string](validationErr.BaseError); ok {
		assert.Equal(suite.T(), errorsx.MessageKeyValidationSummary, key)
	} else {
		suite.T().Fail()
	}
}

func (suite *ValidationErrorTestSuite) TestAddFieldError() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	field := "email"
	messageKey := "validation.required"
	params := []any{"test", 123}

	// Act
	validationErr.AddFieldError(field, messageKey, params...)

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 1)
	fieldError := validationErr.FieldErrors[0]
	assert.Equal(suite.T(), field, fieldError.Field)
	assert.Equal(suite.T(), messageKey, fieldError.MessageKey)
	assert.Equal(suite.T(), params, fieldError.MessageParams)
}

func (suite *ValidationErrorTestSuite) TestAddMultipleFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	validationErr.AddFieldError("email", "validation.required")
	validationErr.AddFieldError("password", "validation.min_length")

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 2)
	assert.Equal(suite.T(), "email", validationErr.FieldErrors[0].Field)
	assert.Equal(suite.T(), "password", validationErr.FieldErrors[1].Field)
}

func (suite *ValidationErrorTestSuite) TestError_WithoutFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	assert.Equal(suite.T(), "validation.failed", errorMsg)
}

func (suite *ValidationErrorTestSuite) TestError_WithSingleFieldError() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "validation.required")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: email: validation.required"
	assert.Equal(suite.T(), expected, errorMsg)
}

func (suite *ValidationErrorTestSuite) TestError_WithMultipleFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "validation.required")
	validationErr.AddFieldError("password", "validation.min_length")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: email: validation.required; password: validation.min_length"
	assert.Equal(suite.T(), expected, errorMsg)
}

func (suite *ValidationErrorTestSuite) TestUnwrap() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	unwrapped := validationErr.Unwrap()

	// Assert
	assert.NotNil(suite.T(), unwrapped)
	assert.Equal(suite.T(), validationErr.BaseError, unwrapped)
}

func (suite *ValidationErrorTestSuite) TestUnwrap_WithErrorsIs() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act & Assert
	assert.True(suite.T(), errors.Is(validationErr, validationErr.BaseError))
}

func (suite *ValidationErrorTestSuite) TestHTTPStatus_DefaultValue() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	status := validationErr.HTTPStatus()

	// Assert
	assert.Equal(suite.T(), 0, status) // BaseError.status is 0 by default after user's modification
}

func (suite *ValidationErrorTestSuite) TestWithHTTPStatus() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	expectedStatus := 422

	// Act
	result := validationErr.WithHTTPStatus(expectedStatus)

	// Assert
	assert.Equal(suite.T(), validationErr, result) // Should return self for chaining
	assert.Equal(suite.T(), expectedStatus, validationErr.HTTPStatus())
}

func (suite *ValidationErrorTestSuite) TestWithHTTPStatus_Chaining() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	result := validationErr.WithHTTPStatus(400).WithHTTPStatus(422)

	// Assert
	assert.Equal(suite.T(), validationErr, result)
	assert.Equal(suite.T(), 422, validationErr.HTTPStatus())
}

func (suite *ValidationErrorTestSuite) TestMarshalJSON_WithoutFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	jsonBytes, err := json.Marshal(validationErr)

	// Assert
	assert.NoError(suite.T(), err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), "validation.failed", result["id"])
	assert.Equal(suite.T(), "errorsx.validation", result["type"])
	assert.Empty(suite.T(), result["field_errors"])
}

func (suite *ValidationErrorTestSuite) TestMarshalJSON_WithFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "validation.required", "param1")
	validationErr.AddFieldError("password", "validation.min_length")

	// Act
	jsonBytes, err := json.Marshal(validationErr)

	// Assert
	assert.NoError(suite.T(), err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), "validation.failed", result["id"])
	assert.Equal(suite.T(), "errorsx.validation", result["type"])

	fieldErrors, ok := result["field_errors"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), fieldErrors, 2)

	// Check first field error
	firstError, ok := fieldErrors[0].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "email", firstError["field"])
	assert.Equal(suite.T(), "validation.required", firstError["message_key"])

	// Check message_params
	messageParams, ok := firstError["message_params"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), messageParams, 1)
	assert.Equal(suite.T(), "param1", messageParams[0])
}

func (suite *ValidationErrorTestSuite) TestMarshalJSON_WithMessageData() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("name", "validation.max_length", 50)

	// Set custom translators
	validationErr.WithSummaryTranslator(func(fieldErrors []errorsx.FieldError, key string, params ...any) string {
		if key == errorsx.MessageKeyValidationSummary {
			return fmt.Sprintf("There are %d errors in the input", len(fieldErrors))
		}
		return key
	})

	validationErr.WithFieldTranslator(func(key string, params ...any) string {
		if key == "validation.max_length" {
			return fmt.Sprintf("Maximum length is %d characters", params[0])
		}
		return key
	})

	// Act
	jsonBytes, err := json.Marshal(validationErr)

	// Assert
	assert.NoError(suite.T(), err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(suite.T(), err)

	// Verify base error fields
	assert.Equal(suite.T(), "validation.failed", result["id"])
	assert.Equal(suite.T(), "errorsx.validation", result["type"])
	assert.Equal(suite.T(), errorsx.MessageKeyValidationSummary, result["message_data"])
	assert.Equal(suite.T(), "There are 1 errors in the input", result["message"])

	// Verify field_errors array
	fieldErrors, ok := result["field_errors"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), fieldErrors, 1)

	// Verify first field error
	firstError, ok := fieldErrors[0].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "name", firstError["field"])
	assert.Equal(suite.T(), "validation.max_length", firstError["message_key"])

	// Verify message_params
	messageParams, ok := firstError["message_params"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), messageParams, 1)
	assert.Equal(suite.T(), float64(50), messageParams[0]) // JSON numbers are unmarshaled as float64

	// Verify translated message
	assert.Equal(suite.T(), "Maximum length is 50 characters", firstError["message"])
}

// Error case tests
func (suite *ValidationErrorTestSuite) TestError_EmptyFieldErrorMessage() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "validation.required")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: email: validation.required"
	assert.Equal(suite.T(), expected, errorMsg)
}

func (suite *ValidationErrorTestSuite) TestAddFieldError_EmptyField() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	validationErr.AddFieldError("", "validation.required")

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 1)
	assert.Empty(suite.T(), validationErr.FieldErrors[0].Field)
}

func (suite *ValidationErrorTestSuite) TestAddFieldError_NilParams() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	validationErr.AddFieldError("email", "validation.required", nil)

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 1)
	assert.Equal(suite.T(), []interface{}{nil}, validationErr.FieldErrors[0].MessageParams)
}

func (suite *ValidationErrorTestSuite) TestMarshalJSON_ConsistentStructure() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.WithHTTPStatus(422)
	validationErr.AddFieldError("email", "validation.email", "Invalid email format")

	// Act
	jsonBytes, err := json.Marshal(validationErr)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify JSON structure matches expected format
	expectedKeys := []string{"id", "type", "field_errors"}
	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(suite.T(), err)

	for _, key := range expectedKeys {
		assert.Contains(suite.T(), result, key)
	}
}
