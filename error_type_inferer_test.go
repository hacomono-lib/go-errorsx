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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeType string) ErrorType {
		// Check if error was handled in test file
		if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
			// If there's a cause, classify based on cause type
			if strings.Contains(causeType, "errorsx") && strings.Contains(causeType, "database") {
				return ErrorType("test.database_error")
			}
			// No cause, direct error
			return ErrorType("test.direct_error")
		}
		return TypeUnknown
	})

	t.Run("direct error without cause", func(t *testing.T) {
		err := New("test.error", WithTypeInferer(inferer)).WithCallerStack()

		got := err.Type()
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeType string) ErrorType {
		// Check for validation errors by type
		if strings.Contains(causeType, "errorsx") && strings.Contains(causeType, "validation") {
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeType string) ErrorType {
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeType string) ErrorType {
		// Detailed classification based on error propagation
		if causeType != "" {
			// Database errors handled in different layers
			if strings.Contains(causeType, "database") {
				if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
					return ErrorType("web.database_error")
				}
			}

			// Network errors
			if strings.Contains(causeType, "network") {
				return ErrorType("network.error")
			}
		}

		// Direct errors in test context
		if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
			return ErrorType("test.direct")
		}

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
			expected: ErrorType("network.error"),
		},
		{
			name: "direct error",
			setupErr: func() *Error {
				return New("direct.error", WithTypeInferer(inferer)).
					WithCallerStack()
			},
			expected: ErrorType("test.direct"),
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

func TestType_Caching(t *testing.T) {
	// Test that Type() results are cached to prevent re-computation
	callCount := 0
	inferer := func(e *Error) ErrorType {
		callCount++
		return ErrorType("computed.type")
	}

	err := New("test.error", WithTypeInferer(ErrorTypeInferer(inferer)))

	// First call should compute
	type1 := err.Type()
	if callCount != 1 {
		t.Errorf("Expected inferer to be called once, got %d", callCount)
	}
	if type1 != ErrorType("computed.type") {
		t.Errorf("Expected 'computed.type', got %v", type1)
	}

	// Second call should use cache
	type2 := err.Type()
	if callCount != 1 {
		t.Errorf("Expected inferer to be called only once, got %d", callCount)
	}
	if type2 != ErrorType("computed.type") {
		t.Errorf("Expected 'computed.type', got %v", type2)
	}
}

func TestJSONMarshal_InferredTypes(t *testing.T) {
	// Test that JSON marshaling includes inferred types for errorsx causes
	inferer := IDContainsInferer(map[string]ErrorType{
		"database": ErrorType("inferred.database"),
	})

	// Create cause error with inferer (no explicit type)
	causeErr := New("database.connection.failed", WithTypeInferer(inferer))

	// Create wrapper
	wrapperErr := New("operation.failed").WithCause(causeErr)

	// Marshal to JSON
	jsonBytes, err := json.Marshal(wrapperErr)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Parse JSON to check cause type
	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	cause, ok := result["cause"].(map[string]interface{})
	if !ok {
		t.Fatal("cause not found in JSON")
	}

	causeType, ok := cause["type"].(string)
	if !ok {
		t.Fatal("cause type not found in JSON")
	}

	// Should include inferred type (not just "errorsx.unknown")
	expected := "inferred.database"
	if causeType != expected {
		t.Errorf("Expected %s, got %s", expected, causeType)
	}

	t.Logf("JSON cause type (with inference): %s", causeType)
}

func TestStackTraceInferer_NoInfiniteRecursion(t *testing.T) {
	// Test that StackTraceInferer doesn't cause infinite recursion
	// when errorsx.Error instances reference each other

	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeType string) ErrorType {
		// This inferer checks causeType, which should not trigger recursion
		if strings.Contains(causeType, "errorsx.database") {
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

	t.Run("errorsx cause with inferer", func(t *testing.T) {
		// Create a cause error that also has an inferer
		causeInferer := func(e *Error) ErrorType {
			return ErrorType("errorsx.database")
		}
		causeErr := New("db.error", WithTypeInferer(ErrorTypeInferer(causeInferer)))

		// Create wrapper error with inferer
		wrapperErr := New("wrapper.error", WithTypeInferer(inferer)).WithCause(causeErr)

		// This should not cause infinite recursion
		// Now getCauseTypeName uses Type(), so it returns inferred types too
		gotType := wrapperErr.Type()

		// causeErr.Type() returns "errorsx.database", which matches the inferer pattern
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

		// err3's cause is err2, which has no explicit type, so causeType will be "errorsx.Error"
		// The inferer won't match "errorsx.database" so it returns TypeUnknown
		gotType := err3.Type()
		expectedType := TypeUnknown

		if gotType != expectedType {
			t.Errorf("Expected %v, got %v", expectedType, gotType)
		}

		// But err2's cause is err1, which has explicit type, so it should infer correctly
		err2Type := err2.Type()
		expectedErr2Type := ErrorType("inferred.database")

		if err2Type != expectedErr2Type {
			t.Errorf("err2: Expected %v, got %v", expectedErr2Type, err2Type)
		}
	})
}
