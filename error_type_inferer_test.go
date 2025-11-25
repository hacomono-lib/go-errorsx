package errorsx

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

const (
	// テスト用のカスタム ErrorType
	TypeAuthentication ErrorType = "test.authentication"
	TypeDatabase       ErrorType = "test.database"
	TypeNetwork        ErrorType = "test.network"
)

func TestErrorTypeInferer_InstanceSpecific(t *testing.T) {
	// インスタンス固有のInfererをテスト
	inferer := func(e *Error) ErrorType {
		if strings.Contains(e.ID(), "auth") {
			return TypeAuthentication
		}
		if strings.Contains(e.ID(), "db") {
			return TypeDatabase
		}
		return TypeUnknown
	}

	tests := []struct {
		name     string
		errorID  string
		expected ErrorType
	}{
		{"authentication error", "auth.failed", TypeAuthentication},
		{"database error", "db.connection_failed", TypeDatabase},
		{"unknown error", "something.else", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errorID, WithTypeInferer(inferer))
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorTypeInferer_ExplicitTypeOverridesInferer(t *testing.T) {
	// 明示的なTypeがInfererをオーバーライドすることをテスト
	inferer := func(e *Error) ErrorType {
		return TypeAuthentication // 常にAuthenticationを返す
	}

	err := New("test.error", WithTypeInferer(inferer))
	if got := err.Type(); got != TypeAuthentication {
		t.Errorf("Type() before explicit type = %v, want %v", got, TypeAuthentication)
	}

	// WithTypeで明示的な型を設定すると、Infererが無効になる
	errWithType := err.WithType(TypeDatabase)
	if got := errWithType.Type(); got != TypeDatabase {
		t.Errorf("Type() after WithType = %v, want %v", got, TypeDatabase)
	}
}

func TestErrorTypeInferer_WithTypeInfererOverridesExplicitType(t *testing.T) {
	// WithTypeInfererが明示的なTypeをリセットすることをテスト
	err := New("test.error", WithType(TypeDatabase))
	if got := err.Type(); got != TypeDatabase {
		t.Errorf("Type() before inferer = %v, want %v", got, TypeDatabase)
	}

	inferer := func(e *Error) ErrorType {
		return TypeAuthentication
	}

	errWithInferer := err.WithTypeInferer(inferer)
	if got := errWithInferer.Type(); got != TypeAuthentication {
		t.Errorf("Type() after WithTypeInferer = %v, want %v", got, TypeAuthentication)
	}
}

func TestGlobalTypeInferer(t *testing.T) {
	// テスト前にグローバルInfererをクリア
	ClearGlobalTypeInferer()
	defer ClearGlobalTypeInferer()

	// グローバルInfererを設定（複数のパターンを1つのInfererで処理）
	SetGlobalTypeInferer(IDContainsInferer(map[string]ErrorType{
		"network": TypeNetwork,
		"timeout": TypeNetwork, // timeoutもNetworkエラーとして扱う
	}))

	tests := []struct {
		name     string
		errorID  string
		expected ErrorType
	}{
		{"network error", "network.connection_failed", TypeNetwork},
		{"timeout error", "timeout.occurred", TypeNetwork},
		{"unknown error", "something.else", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errorID) // Infererを設定せずに作成
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInfererPriority(t *testing.T) {
	// Infererの優先順位をテスト
	ClearGlobalTypeInferer()
	defer ClearGlobalTypeInferer()

	// グローバルInfererを設定
	SetGlobalTypeInferer(func(e *Error) ErrorType {
		if strings.Contains(e.ID(), "test") {
			return TypeDatabase
		}
		return TypeUnknown
	})

	// インスタンス固有のInferer
	instanceInferer := func(e *Error) ErrorType {
		if strings.Contains(e.ID(), "test") {
			return TypeAuthentication
		}
		return TypeUnknown
	}

	tests := []struct {
		name     string
		options  []Option
		expected ErrorType
	}{
		{
			"instance inferer takes priority over global",
			[]Option{WithTypeInferer(instanceInferer)},
			TypeAuthentication,
		},
		{
			"explicit type takes priority over instance inferer",
			[]Option{WithTypeInferer(instanceInferer), WithType(TypeNetwork)},
			TypeNetwork,
		},
		{
			"global inferer used when no instance inferer",
			[]Option{},
			TypeDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New("test.error", tt.options...)
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTypeFunction_WithInferer(t *testing.T) {
	// グローバルType関数がInfererを正しく使用することをテスト
	inferer := func(e *Error) ErrorType {
		return TypeAuthentication
	}

	err := New("test.error", WithTypeInferer(inferer))

	// Type関数（グローバル）を使用
	if got := Type(err); got != TypeAuthentication {
		t.Errorf("Type() = %v, want %v", got, TypeAuthentication)
	}
}

func TestSentinelErrorUnification(t *testing.T) {
	// 異なるパッケージのセンチネルエラーを統一する例
	ClearGlobalTypeInferer()
	defer ClearGlobalTypeInferer()

	// パターンマッチングを使って複数のパッケージからのセンチネルエラーを統一
	SetGlobalTypeInferer(IDPatternInferer(map[string]ErrorType{
		"pkg1.notfound":       TypeNotFound,
		"pkg2.user_not_found": TypeNotFound,
		"pkg3.item_missing":   TypeNotFound,
	}))

	tests := []struct {
		name     string
		errorID  string
		expected ErrorType
	}{
		{"pkg1 not found", "pkg1.notfound", TypeNotFound},
		{"pkg2 user not found", "pkg2.user_not_found", TypeNotFound},
		{"pkg3 item missing", "pkg3.item_missing", TypeNotFound},
		{"other error", "pkg1.invalid", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errorID)
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInfererWithStackTrace(t *testing.T) {
	// スタックトレースを使用するInfererの例
	inferer := func(e *Error) ErrorType {
		// スタックトレースにアクセスできることを確認
		// 実際の使用例では、特定の関数名やファイルパスをチェックすることがある
		if len(e.stacks) > 0 {
			return TypeValidation
		}
		return TypeUnknown
	}

	err := New("test.error", WithTypeInferer(inferer))

	// スタックトレースを追加
	errWithStack := err.WithCallerStack()

	if got := errWithStack.Type(); got != TypeValidation {
		t.Errorf("Type() with stack = %v, want %v", got, TypeValidation)
	}
}

func TestIDPatternInferer(t *testing.T) {
	// IDパターンマッチングのテスト
	inferer := IDPatternInferer(map[string]ErrorType{
		"auth.*":       TypeAuthentication,
		"*database*":   TypeDatabase,
		"validation.*": TypeValidation,
	})

	tests := []struct {
		name     string
		errorID  string
		expected ErrorType
	}{
		{"auth exact match", "auth.failed", TypeAuthentication},
		{"auth wildcard", "auth.token.expired", TypeAuthentication},
		{"database substring", "service.database.error", TypeDatabase},
		{"validation match", "validation.required", TypeValidation},
		{"no match", "network.timeout", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errorID, WithTypeInferer(inferer))
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIDContainsInferer(t *testing.T) {
	// ID部分文字列マッチングのテスト
	inferer := IDContainsInferer(map[string]ErrorType{
		"auth":       TypeAuthentication,
		"database":   TypeDatabase,
		"validation": TypeValidation,
	})

	tests := []struct {
		name     string
		errorID  string
		expected ErrorType
	}{
		{"auth substring", "service.auth.failed", TypeAuthentication},
		{"database substring", "app.database.connection", TypeDatabase},
		{"validation substring", "user.validation.error", TypeValidation},
		{"no match", "network.timeout", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errorID, WithTypeInferer(inferer))
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChainInferers(t *testing.T) {
	// Infererチェインのテスト
	authInferer := IDContainsInferer(map[string]ErrorType{
		"auth": TypeAuthentication,
	})

	dbInferer := IDContainsInferer(map[string]ErrorType{
		"database": TypeDatabase,
	})

	stackInferer := func(e *Error) ErrorType {
		if len(e.stacks) > 0 {
			return TypeValidation
		}
		return TypeUnknown
	}

	chainedInferer := ChainInferers(authInferer, dbInferer, stackInferer)

	tests := []struct {
		name      string
		errorID   string
		withStack bool
		expected  ErrorType
	}{
		{"first inferer matches", "auth.failed", false, TypeAuthentication},
		{"second inferer matches", "database.error", false, TypeDatabase},
		{"third inferer matches", "unknown.error", true, TypeValidation},
		{"no match", "network.timeout", false, TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.errorID, WithTypeInferer(chainedInferer))
			if tt.withStack {
				err = err.WithCallerStack()
			}
			if got := err.Type(); got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStackTraceInferer(t *testing.T) {
	// Test basic stack trace inferer functionality
	inferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
		// Check if error was handled in test file
		if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
			// If there's a cause (errorsx.Error type), classify as database_error
			if strings.Contains(rootCauseType, "go-errorsx.Error") {
				return ErrorType("test.database_error")
			}
			// No errorsx cause (external error or no cause)
			return ErrorType("test.direct_error")
		}
		return TypeUnknown
	})

	t.Run("direct error without cause", func(t *testing.T) {
		err := New("test.error", WithTypeInferer(inferer)).WithCallerStack()

		got := err.Type()
		// RootCause returns the error itself, so rootCauseType is ""
		// No errorsx cause (rootCauseType == ""), returns test.direct_error
		expected := ErrorType("test.direct_error")
		if got != expected {
			t.Errorf("Type() = %v, want %v", got, expected)
		}
	})

	t.Run("error with cause", func(t *testing.T) {
		// Simulate a database error
		causeErr := simulateDatabaseError()

		// Wrap it in our test
		err := New("test.wrapper", WithTypeInferer(inferer)).
			WithCause(causeErr)

		got := err.Type()
		expected := ErrorType("test.database_error")
		if got != expected {
			t.Errorf("Type() = %v, want %v", got, expected)
		}
	})
}

func TestStackTraceInferer_FilePathMatching(t *testing.T) {
	// Test file path based matching
	inferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
		// Check for errorsx.Error type
		if strings.Contains(rootCauseType, "go-errorsx.Error") {
			return ErrorType("validation.error")
		}

		// Check error handling location
		if strings.Contains(errorFrame.File, "_test.go") {
			return ErrorType("test.handled")
		}

		return TypeUnknown
	})

	t.Run("validation cause detected", func(t *testing.T) {
		causeErr := simulateValidationError()
		err := New("wrapper.error", WithTypeInferer(inferer)).
			WithCause(causeErr)

		got := err.Type()
		expected := ErrorType("validation.error")
		if got != expected {
			t.Errorf("Type() = %v, want %v", got, expected)
		}
	})
}

func TestStackTraceInferer_NoStackTrace(t *testing.T) {
	// Test behavior when no stack trace is available
	inferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
		// This should not be called if no stack trace
		t.Error("matcher should not be called when no stack trace available")
		return ErrorType("should.not.happen")
	})

	// Create error without stack trace
	err := New("test.error", WithTypeInferer(inferer))

	got := err.Type()
	expected := TypeUnknown
	if got != expected {
		t.Errorf("Type() = %v, want %v", got, expected)
	}
}

func TestStackTraceInferer_ComplexScenario(t *testing.T) {
	// Test complex error propagation scenario
	inferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
		// Since reflectErrorType returns the same type for all errorsx.Error,
		// we classify based on function name
		if strings.Contains(rootCauseType, "go-errorsx.Error") {
			if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
				return ErrorType("web.database_error")
			}
			// Non-test context
			return ErrorType("network.error")
		}

		// No errorsx.Error cause
		return TypeUnknown
	})

	tests := []struct {
		name     string
		setupErr func() *Error
		expected ErrorType
	}{
		{
			name: "database error in web context",
			setupErr: func() *Error {
				causeErr := simulateDatabaseError()
				return New("web.error", WithTypeInferer(inferer)).
					WithCause(causeErr)
			},
			expected: ErrorType("web.database_error"),
		},
		{
			name: "network error",
			setupErr: func() *Error {
				causeErr := simulateNetworkError()
				return New("service.error", WithTypeInferer(inferer)).
					WithCause(causeErr)
			},
			// Function name contains TestStackTraceInferer, so returns web.database_error
			expected: ErrorType("web.database_error"),
		},
		{
			name: "direct error",
			setupErr: func() *Error {
				return New("direct.error", WithTypeInferer(inferer)).
					WithCallerStack()
			},
			// RootCause returns the error itself, so rootCauseType is ""
			// No errorsx.Error cause, returns TypeUnknown
			expected: TypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupErr()
			got := err.Type()
			if got != tt.expected {
				t.Errorf("Type() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper functions to simulate errors from different components
func simulateDatabaseError() *Error {
	return New("database.connection_failed", WithType(ErrorType("errorsx.database"))).WithCallerStack()
}

func simulateValidationError() *Error {
	return New("validation.required_field", WithType(ErrorType("errorsx.validation"))).WithCallerStack()
}

func simulateNetworkError() *Error {
	return New("network.timeout", WithType(ErrorType("errorsx.network"))).WithCallerStack()
}

func TestExternalErrorMarshal(t *testing.T) {
	// Test that external errors get detailed type information in JSON
	// Create a JSON marshal error using an invalid type (function)
	//nolint:errchkjson,staticcheck // intentionally creating a marshal error for testing
	_, jsonErr := json.Marshal(make(chan int))

	// Wrap external error in errorsx.Error
	err := New("serialization.failed").WithCause(jsonErr)

	// Marshal to JSON
	jsonBytes, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal: %v", marshalErr)
	}

	// Parse JSON to check cause type
	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonBytes, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal: %v", unmarshalErr)
	}

	cause, ok := result["cause"].(map[string]interface{})
	if !ok {
		t.Fatal("cause not found in JSON")
	}

	causeType, ok := cause["type"].(string)
	if !ok {
		t.Fatal("cause type not found in JSON")
	}

	// Should not be "undefined" anymore
	if causeType == "undefined" {
		t.Error("Expected detailed type info, but got 'undefined'")
	}

	// Should contain package path for external errors
	if !strings.Contains(causeType, "json") {
		t.Errorf("Expected JSON-related type, got: %s", causeType)
	}

	t.Logf("External error cause type: %s", causeType)
}

func TestStackTraceInferer_NoInfiniteRecursion(t *testing.T) {
	// Test that StackTraceInferer doesn't cause infinite recursion
	// when errorsx.Error instances reference each other

	inferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
		// This inferer checks rootCauseType, which should not trigger recursion
		if strings.Contains(rootCauseType, "go-errorsx.Error") {
			return ErrorType("inferred.database")
		}
		return TypeUnknown
	})

	t.Run("errorsx cause with explicit type", func(t *testing.T) {
		// Create a cause error with explicit type
		causeErr := New("db.error", WithType(ErrorType("errorsx.database")))

		// Create wrapper error with inferer
		wrapperErr := New("wrapper.error", WithTypeInferer(inferer)).WithCause(causeErr)

		// This should not cause infinite recursion
		gotType := wrapperErr.Type()
		expectedType := ErrorType("inferred.database")

		if gotType != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, gotType)
		}
	})

	t.Run("multiple levels deep", func(t *testing.T) {
		// Test multiple levels to ensure no stack overflow
		err1 := New("level1.error", WithType(ErrorType("errorsx.database")))
		err2 := New("level2.error", WithTypeInferer(inferer)).WithCause(err1)
		err3 := New("level3.error", WithTypeInferer(inferer)).WithCause(err2)

		// With RootCause, err3's root cause is err1 (errorsx.Error type)
		// The inferer matches "go-errorsx.Error" so it returns "inferred.database"
		gotType := err3.Type()
		expectedType := ErrorType("inferred.database")

		if gotType != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, gotType)
		}

		// err2's root cause is also err1, so it should infer correctly
		err2Type := err2.Type()
		expectedErr2Type := ErrorType("inferred.database")

		if err2Type != expectedErr2Type {
			t.Errorf("err2: Expected %v, got %v", expectedErr2Type, err2Type)
		}
	})
}

func TestStackTraceInferer_WithCallerStackAndExplicitType(t *testing.T) {
	// Test that when an error with explicit type and WithCallerStack is used as a cause,
	// the inferer receives the correct type information
	inferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
		t.Logf("Received errorType: %v, rootCauseType: %s", errorType, rootCauseType)
		if strings.Contains(rootCauseType, "go-errorsx.Error") {
			return ErrorType("inferred.from.database")
		}
		return TypeUnknown
	})

	t.Run("cause with explicit type and WithCallerStack", func(t *testing.T) {
		// Create an error with explicit type and WithCallerStack
		causeErr := New("database.error", WithType(ErrorType("explicit.database"))).WithCallerStack()

		// Use it as a cause for another error with inferer
		wrapperErr := New("wrapper.error", WithTypeInferer(inferer)).WithCause(causeErr)

		gotType := wrapperErr.Type()
		expectedType := ErrorType("inferred.from.database")

		if gotType != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, gotType)
		}
	})

	t.Run("self as root cause with explicit type and WithCallerStack", func(t *testing.T) {
		// Create an error with explicit type, inferer, and WithCallerStack
		err := New("test.error",
			WithType(ErrorType("explicit.validation")),
			WithTypeInferer(inferer),
		).WithCallerStack()

		// Since WithTypeInferer is called after WithType, errType will be reset to TypeUnknown
		// RootCause returns the error itself, so rootCauseType is ""
		// No errorsx.Error cause (rootCauseType == ""), inferer returns TypeUnknown
		gotType := err.Type()
		expectedType := TypeUnknown

		if gotType != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, gotType)
		}
	})

	t.Run("self as root cause with explicit type only", func(t *testing.T) {
		// Create wrapper inferer that infers from location
		wrapperInferer := StackTraceInferer(func(errorType ErrorType, errorFrame runtime.Frame, rootCauseType string) ErrorType {
			t.Logf("Received errorType: %v, rootCauseType: %s", errorType, rootCauseType)
			// Infer from location regardless of rootCauseType
			if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
				return ErrorType("inferred.from.location")
			}
			return TypeUnknown
		})

		// Set explicit type first, then set inferer (this clears the explicit type)
		err1 := New("test1.error", WithType(ErrorType("explicit.network"))).WithTypeInferer(wrapperInferer).WithCallerStack()
		// errType is TypeUnknown after WithTypeInferer
		// RootCause returns the error itself (errorsx.Error)
		// Inferer checks location and returns "inferred.from.location"
		gotType1 := err1.Type()
		expectedType1 := ErrorType("inferred.from.location")
		if gotType1 != expectedType1 {
			t.Errorf("Case 1: Expected %v, got %v", expectedType1, gotType1)
		}

		// Set inferer first, then modify to add explicit type
		err2 := New("test2.error", WithTypeInferer(wrapperInferer)).WithCallerStack()
		err2Modified := err2.WithType(ErrorType("explicit.network"))
		// Now errType is set, but WithType clears the inferer
		gotType2 := err2Modified.Type()
		expectedType2 := ErrorType("explicit.network")
		if gotType2 != expectedType2 {
			t.Errorf("Case 2: Expected %v, got %v", expectedType2, gotType2)
		}
	})
}
