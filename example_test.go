package errorsx_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/hacomono-lib/go-errorsx"
)

const (
	errorCodeRequired = "required"
)

// Example demonstrates basic error creation and configuration.
func Example() {
	err := errorsx.New("user.not_found",
		errorsx.WithHTTPStatus(404),
		errorsx.WithMessage("User not found"),
	).WithNotFound()

	fmt.Println("Error ID:", err.ID())
	fmt.Println("Error Type:", err.Type())
	fmt.Println("Error Message:", err.Error())

	// Output:
	// Error ID: user.not_found
	// Error Type: errorsx.unknown
	// Error Message: user.not_found
}

// ExampleError_WithReason demonstrates setting technical error messages.
func ExampleError_WithReason() {
	err := errorsx.New("database.query_failed").
		WithReason("Failed to execute query: %s", "SELECT * FROM users")

	fmt.Println(err.Error())

	// Output:
	// Failed to execute query: SELECT * FROM users
}

// ExampleError_WithMessage demonstrates setting user-facing message data.
func ExampleError_WithMessage() {
	// Simple string message
	err1 := errorsx.New("user.not_found").
		WithMessage("User not found")

	// Internationalization data
	err2 := errorsx.New("user.not_found").
		WithMessage(map[string]string{
			"en": "User not found",
			"ja": "ユーザーが見つかりません",
		})

	// Extract English message
	if msg, ok := errorsx.Message[map[string]string](err2); ok {
		fmt.Println("English:", msg["en"])
		fmt.Println("Japanese:", msg["ja"])
	}

	// Using MessageOr for fallback
	simpleMsg := errorsx.MessageOr[string](err1, "Unknown error")
	fmt.Println("Simple message:", simpleMsg)

	// Output:
	// English: User not found
	// Japanese: ユーザーが見つかりません
	// Simple message: User not found
}

// ExampleError_WithCause demonstrates error chaining and stack traces.
func ExampleError_WithCause() {
	// Simulate a low-level error
	dbErr := errors.New("connection timeout")

	// Wrap with business logic error
	err := errorsx.New("user.fetch_failed").
		WithCause(dbErr).
		WithMessage("Failed to fetch user data")

	// Check if the original error is in the chain
	if errors.Is(err, dbErr) {
		fmt.Println("Original database error found in chain")
	}

	// Unwrap to get the original error
	originalErr := errors.Unwrap(err)
	fmt.Println("Original error:", originalErr.Error())

	// Output:
	// Original database error found in chain
	// Original error: connection timeout
}

// ExampleFilterByType demonstrates filtering errors by type.
func ExampleFilterByType() {
	// Define custom error type
	const TypeAuthentication errorsx.ErrorType = "myapp.authentication"

	// Create multiple errors of different types
	validationErr := errorsx.New("email.invalid", errorsx.WithType(errorsx.TypeValidation))
	authErr := errorsx.New("token.expired", errorsx.WithType(TypeAuthentication))

	// Join errors together
	combined := errorsx.Join(validationErr, authErr)

	// Filter by validation type
	validationErrors := errorsx.FilterByType(combined, errorsx.TypeValidation)
	fmt.Printf("Found %d validation errors\n", len(validationErrors))

	// Check if authentication errors exist
	hasAuth := errorsx.HasType(combined, TypeAuthentication)
	fmt.Printf("Has authentication errors: %v\n", hasAuth)

	// Output:
	// Found 1 validation errors
	// Has authentication errors: true
}

// ExampleNewValidationError demonstrates validation error handling.
func ExampleNewValidationError() {
	verr := errorsx.NewValidationError("user.validation_failed")
	verr.WithHTTPStatus(422)
	verr.WithMessage("Please fix the following errors")

	// Add individual field errors
	verr.AddFieldError("email", errorCodeRequired, "Email is required")
	verr.AddFieldError("age", "min_value", map[string]int{
		"min":     18,
		"current": 16,
	})
	verr.AddFieldError("username", "taken", "Username is already taken")

	// Demonstrate error message
	fmt.Println("Error message:", verr.Error())

	// Demonstrate JSON serialization
	jsonData, err := json.MarshalIndent(verr, "", "  ")
	if err != nil {
		fmt.Printf("JSON marshaling error: %v\n", err)
		return
	}
	fmt.Println("JSON representation:")
	fmt.Println(string(jsonData))

	// Output:
	// Error message: user.validation_failed: email: Email is required; age: map[current:16 min:18]; username: Username is already taken
	// JSON representation:
	// {
	//   "id": "user.validation_failed",
	//   "type": "errorsx.validation",
	//   "message_data": "Please fix the following errors",
	//   "message": "Please fix the following errors",
	//   "field_errors": [
	//     {
	//       "field": "email",
	//       "code": "required",
	//       "message": "Email is required",
	//       "translated_message": "Email is required"
	//     },
	//     {
	//       "field": "age",
	//       "code": "min_value",
	//       "message": {
	//         "current": 16,
	//         "min": 18
	//       },
	//       "translated_message": "map[current:16 min:18]"
	//     },
	//     {
	//       "field": "username",
	//       "code": "taken",
	//       "message": "Username is already taken",
	//       "translated_message": "Username is already taken"
	//     }
	//   ]
	// }
}

// ExampleValidationError_WithFieldTranslator demonstrates custom field error formatting.
func ExampleValidationError_WithFieldTranslator() {
	// Custom field translator for better error messages
	fieldTranslator := func(field, code string, message any) string {
		switch code {
		case errorCodeRequired:
			return fmt.Sprintf("The %s field is required", field)
		case "min_value":
			if data, ok := message.(map[string]int); ok {
				return fmt.Sprintf("The %s must be at least %d (current: %d)",
					field, data["min"], data["current"])
			}
		case "taken":
			return fmt.Sprintf("The %s '%v' is already taken", field, message)
		}
		return fmt.Sprintf("%v", message)
	}

	verr := errorsx.NewValidationError("user.validation_failed").
		WithFieldTranslator(fieldTranslator)

	verr.AddFieldError("email", errorCodeRequired, nil)
	verr.AddFieldError("age", "min_value", map[string]int{"min": 18, "current": 16})
	verr.AddFieldError("username", "taken", "john_doe")

	fmt.Println(verr.Error())

	// Output:
	// user.validation_failed: email: The email field is required; age: The age must be at least 18 (current: 16); username: The username 'john_doe' is already taken
}

// ExampleReplaceMessage demonstrates adding user-friendly messages to any error.
func ExampleReplaceMessage() {
	// Start with a standard Go error
	originalErr := errors.New("sql: no rows in result set")

	// Add user-friendly message
	userErr := errorsx.ReplaceMessage(originalErr, "User not found")

	fmt.Println("Original error:", originalErr.Error())
	fmt.Println("User-friendly error:", userErr.Error())

	// The original error is still accessible
	if errors.Is(userErr, originalErr) {
		fmt.Println("Original error is still in the chain")
	}

	// Output:
	// Original error: sql: no rows in result set
	// User-friendly error: unknown.error
	// Original error is still in the chain
}

// ExampleError_WithCallerStack demonstrates stack trace capture.
func ExampleError_WithCallerStack() {
	err := createUserError()

	// Check if error has stack trace
	if xerr, ok := err.(*errorsx.Error); ok {
		stacks := xerr.Stacks()
		if len(stacks) > 0 {
			fmt.Printf("Stack trace captured with %d frames\n", len(stacks[0].Frames))
			fmt.Printf("Stack message: %s\n", stacks[0].Msg)
		}
	}

	// Output:
	// Stack trace captured with 8 frames
	// Stack message: user.creation_failed
}

func createUserError() error {
	return errorsx.New("user.creation_failed").
		WithCallerStack().
		WithMessage("Failed to create user account")
}

// ExampleJoin demonstrates joining multiple errors together.
func ExampleJoin() {
	err1 := errorsx.New("validation.email", errorsx.WithType(errorsx.TypeValidation))
	err2 := errorsx.New("validation.password", errorsx.WithType(errorsx.TypeValidation))
	err3 := errors.New("database connection failed")

	// Join multiple errors
	combined := errorsx.Join(err1, err2, err3)

	fmt.Println("Combined error:", combined.Error())

	// Check for specific errors
	if errors.Is(combined, err1) {
		fmt.Println("Email validation error found")
	}

	// Filter by type
	validationErrors := errorsx.FilterByType(combined, errorsx.TypeValidation)
	fmt.Printf("Found %d validation errors\n", len(validationErrors))

	// Output:
	// Combined error: validation.email; validation.password; database connection failed
	// Email validation error found
	// Found 2 validation errors
}

// Example_webAPI demonstrates a complete web API error handling pattern.
func Example_webAPI() {
	// Simulate a web API handler
	err := handleUserCreation("", "weak")
	if err != nil {
		// Handle different error types
		switch {
		case errorsx.HasType(err, errorsx.TypeValidation):
			log.Printf("Validation error: %v", err)
			// Return HTTP 422 with validation details
			if verr, ok := err.(*errorsx.ValidationError); ok {
				jsonData, jsonErr := json.Marshal(verr)
				if jsonErr != nil {
					fmt.Printf("JSON marshaling error: %v\n", jsonErr)
					return
				}
				fmt.Printf("HTTP 422 Response: %s\n", jsonData)
			}
		case errorsx.HasType(err, errorsx.TypeUnknown):
			log.Printf("Unknown error: %v", err)
			// Return HTTP 500 with generic message
			fmt.Println("HTTP 500 Response: Internal server error")
		default:
			log.Printf("Unknown error: %v", err)
			fmt.Println("HTTP 500 Response: Unknown error")
		}
	}

	// Output:
	// HTTP 422 Response: {"id":"user.validation_failed","type":"errorsx.validation","message_data":"Please fix the validation errors","message":"Please fix the validation errors","field_errors":[{"field":"email","code":"required","message":"Email is required","translated_message":"Email is required"},{"field":"password","code":"weak","message":"Password is too weak","translated_message":"Password is too weak"}]}
}

func handleUserCreation(email, password string) error {
	// Validate input
	verr := errorsx.NewValidationError("user.validation_failed").
		WithHTTPStatus(422).
		WithMessage("Please fix the validation errors")

	if email == "" {
		verr.AddFieldError("email", errorCodeRequired, "Email is required")
	}
	if password == "weak" {
		verr.AddFieldError("password", "weak", "Password is too weak")
	}

	if len(verr.FieldErrors) > 0 {
		return verr
	}

	// Simulate successful creation
	return nil
}
