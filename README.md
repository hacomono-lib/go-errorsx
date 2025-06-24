# go-errorsx

[![Go Reference](https://pkg.go.dev/badge/github.com/hacomono-lib/go-errorsx.svg)](https://pkg.go.dev/github.com/hacomono-lib/go-errorsx)
[![Go Report Card](https://goreportcard.com/badge/github.com/hacomono-lib/go-errorsx)](https://goreportcard.com/report/github.com/hacomono-lib/go-errorsx)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive error handling library for Go that provides structured, chainable errors with stack traces, error types, and enhanced context.

## Features

- **Structured Errors**: Create errors with IDs, types, and custom messages
- **Stack Traces**: Automatic stack trace capture with customizable cleaning
- **Error Chaining**: Chain errors with cause relationships
- **Type Classification**: Categorize errors with custom types
- **HTTP Integration**: Built-in HTTP status code support
- **Validation Errors**: Specialized support for form validation with field-level errors
- **JSON Marshaling**: Seamless JSON serialization for API responses
- **Error Joining**: Combine multiple errors into a single error
- **Message Extraction**: Type-safe message data extraction

## Installation

```bash
go get github.com/hacomono-lib/go-errorsx
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/hacomono-lib/go-errorsx"
)

func main() {
    // Create a simple error
    err := errorsx.New("user.not.found").
        WithReason("User with ID %d not found", 123).
        WithType(errorsx.TypeValidation).
        WithHTTPStatus(404)
    
    fmt.Println(err.Error()) // Output: User with ID 123 not found
}
```

## Core Concepts

### Error Creation

Create errors with unique identifiers and optional configurations:

```go
// Basic error
err := errorsx.New("database.connection.failed")

// Error with reason and type
err := errorsx.New("validation.failed").
    WithReason("Invalid email format").
    WithType(errorsx.TypeValidation)

// Error with cause (automatically captures stack trace)
baseErr := errors.New("connection refused")
err := errorsx.New("database.error").
    WithCause(baseErr)
```

### Error Types

Classify errors using built-in or custom types:

```go
const (
    TypeBusiness   errorsx.ErrorType = "business"
    TypeInfrastructure errorsx.ErrorType = "infrastructure"
)

err := errorsx.New("payment.failed").WithType(TypeBusiness)

// Check error type
if errorsx.HasType(err, TypeBusiness) {
    // Handle business logic error
}

// Filter errors by type
businessErrors := errorsx.FilterByType(err, TypeBusiness)
```

### Validation Errors

Handle form validation with field-level error details:

```go
validationErr := errorsx.NewValidationError("form.validation.failed").
    WithHTTPStatus(400)

validationErr.AddFieldError("email", "validation.required")
validationErr.AddFieldError("password", "validation.min_length", 8)

// JSON output will include structured field errors
jsonData, _ := json.Marshal(validationErr)
```

### Message Data

Attach structured data to errors for UI display:

```go
type UserErrorData struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
}

err := errorsx.New("user.not.found").
    WithMessage(UserErrorData{
        UserID:   123,
        Username: "johndoe",
    })

// Extract message data with type safety
if data, ok := errorsx.Message[UserErrorData](err); ok {
    fmt.Printf("User %s (ID: %d) not found", data.Username, data.UserID)
}

// Or use with fallback
data := errorsx.MessageOr(err, UserErrorData{UserID: -1, Username: "unknown"})
```

### Error Joining

Combine multiple errors:

```go
err1 := errorsx.New("validation.email")
err2 := errorsx.New("validation.password")
err3 := errors.New("network.timeout")

combined := errorsx.Join(err1, err2, err3)
fmt.Println(combined.Error()) // All errors joined with "; "
```

### Stack Traces

Capture and clean stack traces:

```go
// Manually capture stack trace
err := errorsx.New("process.failed").WithCallerStack()

// Stack trace is automatically captured when using WithCause
baseErr := errors.New("underlying error")
err := errorsx.New("wrapper.error").WithCause(baseErr)

// Get full stack trace
stackTrace := errorsx.FullStackTrace(err)

// Use custom stack trace cleaner
err = errorsx.New("error").
    WithCallerStack().
    WithStackTraceCleaner(func(frames []string) []string {
        // Custom cleaning logic
        return frames
    })
```

## JSON Logging

Errors can be easily serialized to JSON for structured logging:

```go
// Basic error with JSON output
err := errorsx.New("user.not.found").
    WithType(errorsx.TypeValidation).
    WithHTTPStatus(404).
    WithMessage(map[string]interface{}{
        "user_id": 123,
        "action":  "fetch_profile",
    })

jsonData, _ := json.Marshal(err)
fmt.Println(string(jsonData))
```

**Output:**
```json
{
  "id": "user.not.found",
  "type": "errorsx.validation",
  "message_data": {
    "user_id": 123,
    "action": "fetch_profile"
  }
}
```

### Validation Error JSON

```go
validationErr := errorsx.NewValidationError("form.validation.failed").
    WithHTTPStatus(400)

validationErr.AddFieldError("email", "validation.required")
validationErr.AddFieldError("password", "validation.min_length", 8)
validationErr.AddFieldError("age", "validation.range", 18, 65)

jsonData, _ := json.Marshal(validationErr)
fmt.Println(string(jsonData))
```

**Output:**
```json
{
  "id": "form.validation.failed",
  "type": "errorsx.validation",
  "message_data": "validation.summary",
  "message": "validation.summary",
  "field_errors": [
    {
      "field": "email",
      "message_key": "validation.required",
      "message_params": [],
      "message": "validation.required"
    },
    {
      "field": "password",
      "message_key": "validation.min_length",
      "message_params": [8],
      "message": "validation.min_length"
    },
    {
      "field": "age",
      "message_key": "validation.range",
      "message_params": [18, 65],
      "message": "validation.range"
    }
  ]
}
```

### Logging Integration

```go
import (
    "log/slog"
    "github.com/hacomono-lib/go-errorsx"
)

func logError(err error) {
    var xerr *errorsx.Error
    if errors.As(err, &xerr) {
        // slog automatically handles JSON marshaling for errorsx.Error
        slog.Error("Operation failed", "error", xerr)
        
        // Or with additional context
        slog.Error("Operation failed",
            "error", xerr,
            "user_id", 123,
            "operation", "fetch_user",
        )
    } else {
        slog.Error("Operation failed", "error", err)
    }
}
```

## HTTP Integration

Seamlessly integrate with HTTP handlers:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    user, err := getUserByID(123)
    if err != nil {
        var xerr *errorsx.Error
        if errors.As(err, &xerr) {
            w.WriteHeader(xerr.HTTPStatus())
            json.NewEncoder(w).Encode(xerr)
            return
        }
        // Handle non-errorsx errors
        w.WriteHeader(500)
        return
    }
    
    json.NewEncoder(w).Encode(user)
}
```

## Advanced Usage

### Custom Error Types

Define domain-specific error types:

```go
const (
    TypeAuthentication errorsx.ErrorType = "auth"
    TypeAuthorization  errorsx.ErrorType = "authz"
    TypeRateLimit      errorsx.ErrorType = "rate_limit"
)

err := errorsx.New("auth.invalid.token").WithType(TypeAuthentication)
```

### Error Wrapping and Unwrapping

```go
originalErr := errors.New("database connection failed")
wrappedErr := errorsx.New("user.fetch.failed").WithCause(originalErr)

// Unwrap to get the original error
if errors.Is(wrappedErr, originalErr) {
    // Handle database connection issues
}
```

### Validation with Custom Translators

```go
validationErr := errorsx.NewValidationError("form.invalid").
    WithSummaryTranslator(func(fieldErrors []errorsx.FieldError, key string, params ...any) string {
        return fmt.Sprintf("Form has %d validation errors", len(fieldErrors))
    }).
    WithFieldTranslator(func(key string, params ...any) string {
        // Custom field error translation
        return translateMessage(key, params...)
    })
```

## API Reference

### Core Types

- `Error`: Main error type with ID, type, message, and stack trace
- `ErrorType`: String-based error classification
- `ValidationError`: Specialized error for form validation
- `FieldError`: Individual field validation error

### Key Functions

- `New(id string, opts ...Option) *Error`: Create new error
- `Join(errs ...error) error`: Combine multiple errors
- `Message[T](err error) (T, bool)`: Extract typed message data
- `FilterByType(err error, typ ErrorType) []*Error`: Filter errors by type
- `HasType(err error, typ ErrorType) bool`: Check if error has specific type

### Options

- `WithType(ErrorType)`: Set error type
- `WithHTTPStatus(int)`: Set HTTP status code
- `WithCallerStack()`: Capture stack trace from caller
- `WithCause(error)`: Set underlying cause and automatically capture stack trace
- `WithMessage(any)`: Attach message data

**Note**: `WithCause` and `WithCallerStack` are mutually exclusive. `WithCause` automatically captures the stack trace, so using both together is not necessary and the second one will be ignored.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.