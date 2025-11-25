# go-errorsx

[![Go Reference](https://pkg.go.dev/badge/github.com/hacomono-lib/go-errorsx.svg)](https://pkg.go.dev/github.com/hacomono-lib/go-errorsx)
[![Go Report Card](https://goreportcard.com/badge/github.com/hacomono-lib/go-errorsx)](https://goreportcard.com/report/github.com/hacomono-lib/go-errorsx)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/hacomono-lib/go-errorsx/workflows/CI/badge.svg)](https://github.com/hacomono-lib/go-errorsx/actions/workflows/ci.yml)

A comprehensive error handling library for Go that provides structured, chainable errors with stack traces, error types, and enhanced context.

## Features

- **Structured Errors**: Create errors with IDs, types, and custom messages
- **Stack Traces**: Automatic stack trace capture with customizable cleaning
- **Error Chaining**: Chain errors with cause relationships
- **Type Classification**: Categorize errors with custom types
- **Dynamic Error Type Inference**: Runtime error type determination based on stack traces and patterns
- **HTTP Integration**: Built-in HTTP status code support
- **Validation Errors**: Specialized support for form validation with field-level errors
- **JSON Marshaling**: Seamless JSON serialization for API responses
- **Error Joining**: Combine multiple errors into a single error
- **Message Extraction**: Type-safe message data extraction
- **Retryable Errors**: Mark errors as retryable for resilient operations

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

#### Design Philosophy: ID vs Type

The library follows a clear separation of concerns:

- **Error ID**: Represents the unique identity of an error (e.g., `"user.not.found"`, `"payment.failed"`)
- **Error Type**: Represents the category or classification for grouping errors (e.g., `"validation"`, `"business"`, `"infrastructure"`)

This design allows:

```go
// Same logical error in different contexts
userErr := errorsx.New("user.not.found").WithType(TypeBusiness)
adminErr := errorsx.New("user.not.found").WithType(TypeSecurity)

// They are considered the same error for handling purposes
if errors.Is(userErr, adminErr) {
    // This returns true - same ID means same logical error
    // regardless of classification
}

// But can be handled differently based on type
if errorsx.HasType(userErr, TypeBusiness) {
    // Handle as business logic error
}
if errorsx.HasType(adminErr, TypeSecurity) {
    // Handle as security-related error
}
```

**Key Benefits:**
- **Flexible Error Handling**: The same logical error can be handled consistently across different contexts
- **Clear Separation**: Identity (what happened) vs Classification (how to categorize it)
- **Reusable Error Definitions**: Error IDs can be reused across different layers or modules

### Dynamic Error Type Inference

The library supports runtime error type determination using inferer functions:

```go
// Pattern-based inferer for ID matching
idInferer := errorsx.IDContainsInferer(map[string]errorsx.ErrorType{
    "auth":     errorsx.ErrorType("security.auth"),
    "database": errorsx.ErrorType("data.persistence"), 
})

// Stack trace-based inferer for context-aware classification  
stackInferer := errorsx.StackTraceInferer(func(errorFrame runtime.Frame, causeType string) errorsx.ErrorType {
    // Handle external library errors by type
    switch {
    case strings.Contains(causeType, "database/sql"):
        return errorsx.ErrorType("database.error")
    case strings.Contains(causeType, "encoding/json"):
        return errorsx.ErrorType("serialization.error")
    case strings.Contains(causeType, "errorsx.validation"):
        return errorsx.ErrorType("validation.error")
    }
    
    // Handle based on error handling location
    if strings.Contains(errorFrame.File, "/handler/") {
        return errorsx.ErrorType("web.error")
    }
    return errorsx.TypeUnknown
})

// Chain multiple inferers for fallback logic
combinedInferer := errorsx.ChainInferers(stackInferer, idInferer)

// Apply to errors
err := errorsx.New("auth.failed", errorsx.WithTypeInferer(combinedInferer))

// Or set globally
errorsx.SetGlobalTypeInferer(idInferer)
```

**Key Benefits:**
- **Unified Error Classification**: Handle errors from different packages consistently
- **Context-Aware Handling**: Different classification based on error propagation path  
- **Runtime Flexibility**: Determine error types dynamically based on runtime context

### Validation Errors

Handle form validation with field-level error details:

```go
validationErr := errorsx.NewValidationError("form.validation.failed").
    WithHTTPStatus(400).
    WithMessage("Form validation failed")

// Simple string messages
validationErr.AddFieldError("email", "required", "Email is required")

// Complex message data
validationErr.AddFieldError("password", "min_length", map[string]any{
    "min":     8,
    "current": 3,
})

// Translation data
validationErr.AddFieldError("age", "range", struct {
    Key    string `json:"key"`
    Params map[string]any `json:"params"`
}{
    Key:    "validation.age.range",
    Params: map[string]any{"min": 18, "max": 65},
})

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

#### External Error Type Information

When wrapping external errors (from standard library or other packages), errorsx now provides detailed type information instead of generic "undefined" labels:

```go
// Example with JSON serialization error
_, jsonErr := json.Marshal(make(chan int))
err := errorsx.New("serialization.failed").WithCause(jsonErr)

jsonBytes, _ := json.Marshal(err)
// JSON output includes detailed cause type:
// {
//   "id": "serialization.failed", 
//   "msg": "serialization.failed",
//   "cause": {
//     "msg": "json: unsupported type: chan int",
//     "type": "encoding/json.UnsupportedTypeError"  // ← Detailed type info!
//   }
// }

// Database errors
dbErr := sql.ErrNoRows
wrappedErr := errorsx.New("user.not_found").WithCause(dbErr)
// Cause type will be: "database/sql.Error"

// Network errors  
httpErr := &url.Error{Op: "Get", URL: "http://example.com", Err: errors.New("timeout")}
netErr := errorsx.New("request.failed").WithCause(httpErr)
// Cause type will be: "net/url.Error"
```

This detailed type information enables more sophisticated error handling and type-based error classification using `StackTraceInferer`.

### Validation Error JSON

```go
validationErr := errorsx.NewValidationError("form.validation.failed").
    WithHTTPStatus(400).
    WithMessage("Form validation failed")

validationErr.AddFieldError("email", "required", "Email is required")
validationErr.AddFieldError("password", "min_length", map[string]any{
    "min": 8, "current": 3,
})
validationErr.AddFieldError("age", "range", struct {
    Min int `json:"min"`
    Max int `json:"max"`
}{Min: 18, Max: 65})

jsonData, _ := json.Marshal(validationErr)
fmt.Println(string(jsonData))
```

**Output:**
```json
{
  "id": "form.validation.failed",
  "type": "errorsx.validation",
  "message_data": "Form validation failed",
  "message": "Validation failed with 3 error(s)",
  "field_errors": [
    {
      "field": "email",
      "code": "required",
      "message": "Email is required",
      "translated_message": "Email is required"
    },
    {
      "field": "password",
      "code": "min_length",
      "message": {
        "min": 8,
        "current": 3
      },
      "translated_message": "Password must be at least 8 characters"
    },
    {
      "field": "age",
      "code": "range",
      "message": {
        "min": 18,
        "max": 65
      },
      "translated_message": "Age must be between 18 and 65"
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

### Retryable Errors

Mark errors as retryable to implement resilient error handling:

```go
// Create retryable error
err := errorsx.New("service.unavailable").WithRetryable()

// Or use the convenience constructor
err := errorsx.NewRetryable("connection.timeout")

// Or use the option
err := errorsx.New("rate.limit.exceeded", errorsx.WithRetryable())

// Check if error is retryable
if errorsx.IsRetryable(err) {
    // Implement retry logic
    for i := 0; i < maxRetries; i++ {
        if err := operation(); err == nil {
            break
        } else if !errorsx.IsRetryable(err) {
            return err // Don't retry non-retryable errors
        }
        time.Sleep(backoff)
    }
}

// Retryable status is preserved in JSON
jsonData, _ := json.Marshal(err)
// Output includes: "is_retryable": true
```

### Validation with Translation Support

The library provides built-in translation support for both summary messages and individual field errors:

```go
// Custom translators
summaryTranslator := func(fieldErrors []errorsx.FieldError, messageData any) string {
    count := len(fieldErrors)
    if count == 1 {
        return "There is 1 validation error"
    }
    return fmt.Sprintf("There are %d validation errors", count)
}

fieldTranslator := func(field, code string, message any) string {
    switch code {
    case "required":
        return fmt.Sprintf("%s is required", strings.Title(field))
    case "min_length":
        if data, ok := message.(map[string]any); ok {
            if min, ok := data["min"].(int); ok {
                return fmt.Sprintf("%s must be at least %d characters", strings.Title(field), min)
            }
        }
    }
    return fmt.Sprintf("%v", message)
}

// Apply translators
validationErr := errorsx.NewValidationError("form.invalid").
    WithSummaryTranslator(summaryTranslator).
    WithFieldTranslator(fieldTranslator).
    WithMessage("Form validation failed")

// Add field errors - translators will be applied automatically
validationErr.AddFieldError("email", "required", nil)
validationErr.AddFieldError("password", "min_length", map[string]any{"min": 8})

// Error() and JSON output will use translated messages
fmt.Println(validationErr.Error())
// Output: form.invalid: Email: Email is required; Password: Password must be at least 8 characters
```

#### Translation with i18n Libraries

```go
// Example with go-i18n or similar
fieldTranslator := func(field, code string, message any) string {
    key := fmt.Sprintf("validation.%s.%s", field, code)
    
    // Use your i18n library
    return i18n.Translate(key, message)
}

validationErr.WithFieldTranslator(fieldTranslator)
```

## API Reference

### Core Types

- `Error`: Main error type with ID, type, message, and stack trace
- `ErrorType`: String-based error classification
- `ValidationError`: Specialized error for form validation
- `FieldError`: Individual field validation error

### Key Functions

- `New(id string, opts ...Option) *Error`: Create new error
- `NewRetryable(id string, opts ...Option) *Error`: Create new retryable error
- `Join(errs ...error) error`: Combine multiple errors
- `Message[T](err error) (T, bool)`: Extract typed message data
- `FilterByType(err error, typ ErrorType) []*Error`: Filter errors by type
- `HasType(err error, typ ErrorType) bool`: Check if error has specific type
- `IsRetryable(err error) bool`: Check if error is retryable

#### Dynamic Error Type Inference Functions

- `IDPatternInferer(patterns map[string]ErrorType) ErrorTypeInferer`: Create inferer using glob patterns
- `IDContainsInferer(substrings map[string]ErrorType) ErrorTypeInferer`: Create inferer using substring matching
- `StackTraceInferer(matcher func(runtime.Frame, string) ErrorType) ErrorTypeInferer`: Create inferer using stack trace and cause type analysis
- `ChainInferers(inferers ...ErrorTypeInferer) ErrorTypeInferer`: Combine multiple inferers
- `SetGlobalTypeInferer(inferer ErrorTypeInferer)`: Set global inferer for all errors
- `ClearGlobalTypeInferer()`: Remove global inferer

### Options

- `WithType(ErrorType)`: Set error type
- `WithTypeInferer(ErrorTypeInferer)`: Set dynamic type inferer for runtime classification
- `WithHTTPStatus(int)`: Set HTTP status code
- `WithCallerStack()`: Capture stack trace from caller
- `WithCause(error)`: Set underlying cause and automatically capture stack trace
- `WithMessage(any)`: Attach message data
- `WithRetryable()`: Mark error as retryable

**Note**: `WithCause` and `WithCallerStack` are mutually exclusive. `WithCause` automatically captures the stack trace, so using both together is not necessary and the second one will be ignored.

## Development

### Local Development

```bash
# Install development tools
make install-tools

# Run tests
make test

# Run tests with coverage
make test-cover

# Run linter
make lint

# Run all checks
make ci
```

### Container Development

For developers who prefer containerized development or want to avoid polluting their local environment:

#### Using Docker Compose

```bash
# Start development environment
make docker-dev

# Run tests in container
make docker-test

# Run linter in container
make docker-lint

# Run security scan in container
make docker-security

# Clean up containers
make docker-clean
```

#### Using VS Code Dev Containers

1. Install the "Dev Containers" extension in VS Code
2. Open the project in VS Code
3. Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on Mac)
4. Select "Dev Containers: Reopen in Container"

The development container includes:
- Go 1.25 with all development tools
- golangci-lint for code quality
- gosec for security scanning
- Git and GitHub CLI
- Optimized VS Code settings for Go development

### Performance Optimizations

The CI/CD pipeline includes several performance optimizations:
- **Multi-level caching**: Go modules, build cache, and tool binaries
- **Parallel execution**: Tests run concurrently across multiple Go versions
- **Incremental builds**: Only rebuild when necessary
- **Container layer caching**: Optimized Docker builds

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.