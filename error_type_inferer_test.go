package errorsx

import (
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeFrame *runtime.Frame) ErrorType {
		// Check if error was handled in test file
		if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
			// If there's a cause, classify based on cause
			if causeFrame != nil && strings.Contains(causeFrame.Function, "simulateDatabase") {
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeFrame *runtime.Frame) ErrorType {
		// Check for specific file patterns
		if causeFrame != nil {
			if strings.Contains(causeFrame.File, "_test.go") &&
				strings.Contains(causeFrame.Function, "simulateValidation") {
				return ErrorType("validation.error")
			}
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeFrame *runtime.Frame) ErrorType {
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
	inferer := StackTraceInferer(func(errorFrame runtime.Frame, causeFrame *runtime.Frame) ErrorType {
		// Detailed classification based on error propagation
		if causeFrame != nil {
			// Database errors handled in different layers
			if strings.Contains(causeFrame.Function, "simulateDatabase") {
				if strings.Contains(errorFrame.Function, "TestStackTraceInferer") {
					return ErrorType("web.database_error")
				}
			}

			// Network errors
			if strings.Contains(causeFrame.Function, "simulateNetwork") {
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
	return New("database.connection_failed").WithCallerStack()
}

func simulateValidationError() *Error {
	return New("validation.required_field").WithCallerStack()
}

func simulateNetworkError() *Error {
	return New("network.timeout").WithCallerStack()
}
