package llm

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// MockClient はテスト用のモッククライアントです
type MockClient struct {
	ExecuteFunc     func(ctx context.Context, prompt string) (string, error)
	ExecuteJSONFunc func(ctx context.Context, prompt string, target interface{}) error
	timeout         time.Duration
}

// NewMockClient は新しいモッククライアントを作成します
func NewMockClient() *MockClient {
	return &MockClient{
		timeout: 5 * time.Minute,
	}
}

// Execute はモックのExecute実装です
func (m *MockClient) Execute(ctx context.Context, prompt string) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, prompt)
	}
	return "mock response", nil
}

// ExecuteJSON はモックのExecuteJSON実装です
func (m *MockClient) ExecuteJSON(ctx context.Context, prompt string, target interface{}) error {
	if m.ExecuteJSONFunc != nil {
		return m.ExecuteJSONFunc(ctx, prompt, target)
	}
	// デフォルトは空のJSONオブジェクトを返す
	return json.Unmarshal([]byte("{}"), target)
}

// SetTimeout はタイムアウトを設定します
func (m *MockClient) SetTimeout(d time.Duration) {
	m.timeout = d
}

// GetTimeout はタイムアウトを返します
func (m *MockClient) GetTimeout() time.Duration {
	return m.timeout
}

// Compile-time interface check for MockClient
var _ LLMClient = (*MockClient)(nil)

// TestNewClaudeClient tests ClaudeClient creation
func TestNewClaudeClient(t *testing.T) {
	tests := []struct {
		name     string
		opts     []ClientOption
		expected struct {
			timeout time.Duration
			cliPath string
		}
	}{
		{
			name: "default options",
			opts: nil,
			expected: struct {
				timeout time.Duration
				cliPath string
			}{
				timeout: 5 * time.Minute,
				cliPath: "claude",
			},
		},
		{
			name: "with custom timeout",
			opts: []ClientOption{WithTimeout(10 * time.Minute)},
			expected: struct {
				timeout time.Duration
				cliPath string
			}{
				timeout: 10 * time.Minute,
				cliPath: "claude",
			},
		},
		{
			name: "with custom cli path",
			opts: []ClientOption{WithCLIPath("/usr/local/bin/claude")},
			expected: struct {
				timeout time.Duration
				cliPath string
			}{
				timeout: 5 * time.Minute,
				cliPath: "/usr/local/bin/claude",
			},
		},
		{
			name: "with multiple options",
			opts: []ClientOption{
				WithTimeout(15 * time.Minute),
				WithCLIPath("/custom/claude"),
				WithCLIArgs("--dangerously-skip-permissions"),
			},
			expected: struct {
				timeout time.Duration
				cliPath string
			}{
				timeout: 15 * time.Minute,
				cliPath: "/custom/claude",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClaudeClient(tt.opts...)
			if client.GetTimeout() != tt.expected.timeout {
				t.Errorf("timeout = %v, want %v", client.GetTimeout(), tt.expected.timeout)
			}
			if client.cliPath != tt.expected.cliPath {
				t.Errorf("cliPath = %v, want %v", client.cliPath, tt.expected.cliPath)
			}
		})
	}
}

// TestSetTimeout tests SetTimeout method
func TestSetTimeout(t *testing.T) {
	client := NewClaudeClient()
	newTimeout := 3 * time.Minute
	client.SetTimeout(newTimeout)

	if client.GetTimeout() != newTimeout {
		t.Errorf("GetTimeout() = %v, want %v", client.GetTimeout(), newTimeout)
	}
}

// TestExtractJSON tests JSON extraction from various formats
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "raw json object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
			hasError: false,
		},
		{
			name:     "raw json array",
			input:    `[1, 2, 3]`,
			expected: `[1, 2, 3]`,
			hasError: false,
		},
		{
			name:     "markdown json code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
			hasError: false,
		},
		{
			name:     "markdown generic code block with json",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
			hasError: false,
		},
		{
			name:     "json embedded in text",
			input:    "Here is the result: {\"key\": \"value\"} and some text",
			expected: `{"key": "value"}`,
			hasError: false,
		},
		{
			name:     "json array embedded in text",
			input:    "Results: [1, 2, 3] done",
			expected: `[1, 2, 3]`,
			hasError: false,
		},
		{
			name:     "no json",
			input:    "This is just plain text",
			expected: "",
			hasError: true,
		},
		{
			name:     "complex nested json",
			input:    `{"user": {"name": "test", "items": [1, 2, 3]}}`,
			expected: `{"user": {"name": "test", "items": [1, 2, 3]}}`,
			hasError: false,
		},
		{
			name:     "json with whitespace",
			input:    "\n\n  {\"key\": \"value\"}  \n\n",
			expected: `{"key": "value"}`,
			hasError: false,
		},
		{
			name:     "markdown block with extra whitespace",
			input:    "```json\n\n  {\"key\": \"value\"}  \n\n```",
			expected: `{"key": "value"}`,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractJSON(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("extractJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMockClient tests the mock client implementation
func TestMockClient(t *testing.T) {
	t.Run("default execute", func(t *testing.T) {
		mock := NewMockClient()
		ctx := context.Background()

		result, err := mock.Execute(ctx, "test prompt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "mock response" {
			t.Errorf("result = %v, want 'mock response'", result)
		}
	})

	t.Run("custom execute", func(t *testing.T) {
		mock := NewMockClient()
		mock.ExecuteFunc = func(ctx context.Context, prompt string) (string, error) {
			return "custom response for: " + prompt, nil
		}
		ctx := context.Background()

		result, err := mock.Execute(ctx, "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != "custom response for: test" {
			t.Errorf("result = %v, want 'custom response for: test'", result)
		}
	})

	t.Run("execute json with struct", func(t *testing.T) {
		mock := NewMockClient()
		mock.ExecuteJSONFunc = func(ctx context.Context, prompt string, target interface{}) error {
			// JSONとして値を設定
			data := []byte(`{"name":"test-value"}`)
			return json.Unmarshal(data, target)
		}
		ctx := context.Background()

		type TestTarget struct {
			Name string `json:"name"`
		}
		var target TestTarget

		err := mock.ExecuteJSON(ctx, "test", &target)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if target.Name != "test-value" {
			t.Errorf("target.Name = %v, want 'test-value'", target.Name)
		}
	})

	t.Run("timeout methods", func(t *testing.T) {
		mock := NewMockClient()
		mock.SetTimeout(10 * time.Minute)

		if mock.GetTimeout() != 10*time.Minute {
			t.Errorf("GetTimeout() = %v, want 10m", mock.GetTimeout())
		}
	})
}

// TestClientInterface verifies the interface implementation
func TestClientInterface(t *testing.T) {
	// This test ensures both ClaudeClient and MockClient implement the LLMClient interface.
	var _ LLMClient = NewClaudeClient()
	var _ LLMClient = NewMockClient()
}

func TestLLMClientUsesTimeoutMethods(t *testing.T) {
	var client LLMClient = NewMockClient()
	client.SetTimeout(2 * time.Minute)

	if client.GetTimeout() != 2*time.Minute {
		t.Fatalf("GetTimeout() = %v, want 2m", client.GetTimeout())
	}
}

// TestTruncate tests the truncate helper function
func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long string", 10, "this is a ..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// BenchmarkExtractJSON benchmarks the JSON extraction
func BenchmarkExtractJSON(b *testing.B) {
	input := "```json\n{\"key\": \"value\", \"nested\": {\"a\": 1, \"b\": 2}}\n```"
	for i := 0; i < b.N; i++ {
		_, _ = extractJSON(input)
	}
}

// BenchmarkExecute benchmarks the Execute method (without actual CLI call)
func BenchmarkNewClaudeClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewClaudeClient(WithTimeout(5 * time.Minute))
	}
}
