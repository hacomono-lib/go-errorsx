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

	// Test message data is nil by default
	if data, ok := errorsx.Message[any](validationErr.BaseError); ok && data != nil {
		suite.T().Errorf("Expected message data to be nil, got: %v", data)
	}
}

func (suite *ValidationErrorTestSuite) TestAddFieldError() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	field := "email"
	code := "required"
	message := "Email is required"

	// Act
	validationErr.AddFieldError(field, code, message)

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 1)
	fieldError := validationErr.FieldErrors[0]
	assert.Equal(suite.T(), field, fieldError.Field)
	assert.Equal(suite.T(), code, fieldError.Code)
	assert.Equal(suite.T(), message, fieldError.Message)
}

func (suite *ValidationErrorTestSuite) TestAddMultipleFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	validationErr.AddFieldError("email", "required", "Email is required")
	validationErr.AddFieldError("password", "min_length", map[string]any{"min": 8, "current": 3})

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 2)
	assert.Equal(suite.T(), "email", validationErr.FieldErrors[0].Field)
	assert.Equal(suite.T(), "required", validationErr.FieldErrors[0].Code)
	assert.Equal(suite.T(), "password", validationErr.FieldErrors[1].Field)
	assert.Equal(suite.T(), "min_length", validationErr.FieldErrors[1].Code)
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
	validationErr.AddFieldError("email", "required", "Email is required")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: email: Email is required"
	assert.Equal(suite.T(), expected, errorMsg)
}

func (suite *ValidationErrorTestSuite) TestError_WithMultipleFieldErrors() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "required", "Email is required")
	validationErr.AddFieldError("password", "min_length", "Password must be at least 8 characters")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: email: Email is required; password: Password must be at least 8 characters"
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
	validationErr.AddFieldError("email", "required", "Email is required")
	validationErr.AddFieldError("password", "min_length", map[string]any{"min": 8, "current": 3})

	// Act
	jsonBytes, err := json.Marshal(validationErr)

	// Assert
	assert.NoError(suite.T(), err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), "validation.failed", result["id"])
	assert.Equal(suite.T(), "errorsx.validation", result["type"])
	assert.Contains(suite.T(), result, "message")

	fieldErrors, ok := result["field_errors"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), fieldErrors, 2)

	// Check first field error
	firstError, ok := fieldErrors[0].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "email", firstError["field"])
	assert.Equal(suite.T(), "required", firstError["code"])
	assert.Equal(suite.T(), "Email is required", firstError["message"])
	assert.Contains(suite.T(), firstError, "translated_message")

	// Check second field error
	secondError, ok := fieldErrors[1].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "password", secondError["field"])
	assert.Equal(suite.T(), "min_length", secondError["code"])
	assert.Contains(suite.T(), secondError, "translated_message")

	// Check message object
	messageObj, ok := secondError["message"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), float64(8), messageObj["min"])
	assert.Equal(suite.T(), float64(3), messageObj["current"])
}

func (suite *ValidationErrorTestSuite) TestMarshalJSON_WithMessageData() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("name", "max_length", map[string]any{"max": 50, "current": 75})
	validationErr.WithMessage("Form validation failed")

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
	assert.Equal(suite.T(), "Form validation failed", result["message_data"])
	// When messageData is set, DefaultSummaryTranslator uses it as the message
	assert.Equal(suite.T(), "Form validation failed", result["message"])

	// Verify field_errors array
	fieldErrors, ok := result["field_errors"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), fieldErrors, 1)

	// Verify first field error
	firstError, ok := fieldErrors[0].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "name", firstError["field"])
	assert.Equal(suite.T(), "max_length", firstError["code"])
	assert.Contains(suite.T(), firstError, "translated_message")

	// Verify message object
	messageObj, ok := firstError["message"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), float64(50), messageObj["max"])
	assert.Equal(suite.T(), float64(75), messageObj["current"])
}

// Error case tests
func (suite *ValidationErrorTestSuite) TestError_EmptyFieldErrorMessage() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "required", "")

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: email: "
	assert.Equal(suite.T(), expected, errorMsg)
}

func (suite *ValidationErrorTestSuite) TestAddFieldError_EmptyField() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	validationErr.AddFieldError("", "required", "Field is required")

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 1)
	assert.Empty(suite.T(), validationErr.FieldErrors[0].Field)
	assert.Equal(suite.T(), "required", validationErr.FieldErrors[0].Code)
	assert.Equal(suite.T(), "Field is required", validationErr.FieldErrors[0].Message)
}

func (suite *ValidationErrorTestSuite) TestAddFieldError_NilMessage() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")

	// Act
	validationErr.AddFieldError("email", "required", nil)

	// Assert
	assert.Len(suite.T(), validationErr.FieldErrors, 1)
	assert.Equal(suite.T(), "email", validationErr.FieldErrors[0].Field)
	assert.Equal(suite.T(), "required", validationErr.FieldErrors[0].Code)
	assert.Nil(suite.T(), validationErr.FieldErrors[0].Message)
}

func (suite *ValidationErrorTestSuite) TestMarshalJSON_ConsistentStructure() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.WithHTTPStatus(422)
	validationErr.AddFieldError("email", "invalid_format", "Invalid email format")

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

	// Verify field error structure
	fieldErrors, ok := result["field_errors"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), fieldErrors, 1)

	fieldError, ok := fieldErrors[0].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "email", fieldError["field"])
	assert.Equal(suite.T(), "invalid_format", fieldError["code"])
	assert.Equal(suite.T(), "Invalid email format", fieldError["message"])
	assert.Contains(suite.T(), fieldError, "translated_message")
}

func (suite *ValidationErrorTestSuite) TestError_WithNonStringMessage() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("age", "min_value", map[string]any{"min": 18, "current": 16})

	// Act
	errorMsg := validationErr.Error()

	// Assert
	expected := "validation.failed: age: map[current:16 min:18]"
	assert.Equal(suite.T(), expected, errorMsg)
}

func (suite *ValidationErrorTestSuite) TestWithSummaryTranslator() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "required", "Email is required")
	validationErr.AddFieldError("password", "min_length", map[string]any{"min": 8})

	customTranslator := func(fieldErrors []errorsx.FieldError, messageData any) string {
		return fmt.Sprintf("Form has %d validation errors", len(fieldErrors))
	}

	// Act
	result := validationErr.WithSummaryTranslator(customTranslator)

	// Assert
	assert.Equal(suite.T(), validationErr, result) // Should return self for chaining

	// Test JSON marshaling uses the custom translator
	jsonBytes, err := json.Marshal(validationErr)
	assert.NoError(suite.T(), err)

	var jsonResult map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonResult)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Form has 2 validation errors", jsonResult["message"])
}

func (suite *ValidationErrorTestSuite) TestWithFieldTranslator() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed")
	validationErr.AddFieldError("email", "required", map[string]any{"field": "email"})

	customTranslator := func(field, code string, message any) string {
		if code == "required" {
			return fmt.Sprintf("The %s field is required", field)
		}
		return fmt.Sprintf("%v", message)
	}

	// Act
	result := validationErr.WithFieldTranslator(customTranslator)

	// Assert
	assert.Equal(suite.T(), validationErr, result) // Should return self for chaining

	// Test Error() uses the custom translator
	errorMsg := validationErr.Error()
	expected := "validation.failed: email: The email field is required"
	assert.Equal(suite.T(), expected, errorMsg)

	// Test JSON marshaling uses the custom translator
	jsonBytes, err := json.Marshal(validationErr)
	assert.NoError(suite.T(), err)

	var jsonResult map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonResult)
	assert.NoError(suite.T(), err)

	fieldErrors, ok := jsonResult["field_errors"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), fieldErrors, 1)

	firstError, ok := fieldErrors[0].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "The email field is required", firstError["translated_message"])
}

func (suite *ValidationErrorTestSuite) TestTranslatorChaining() {
	// Arrange
	validationErr := errorsx.NewValidationError("validation.failed").
		WithSummaryTranslator(func(fieldErrors []errorsx.FieldError, messageData any) string {
			return "Custom summary"
		}).
		WithFieldTranslator(func(field, code string, message any) string {
			return "Custom field message"
		})

	validationErr.AddFieldError("test", "code", "message")

	// Act & Assert
	errorMsg := validationErr.Error()
	expected := "validation.failed: test: Custom field message"
	assert.Equal(suite.T(), expected, errorMsg)

	// Test JSON output
	jsonBytes, err := json.Marshal(validationErr)
	assert.NoError(suite.T(), err)

	var jsonResult map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonResult)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Custom summary", jsonResult["message"])
}
