package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

// TestNewLLMClient tests LLM client factory function
func TestNewLLMClient(t *testing.T) {
	t.Run("missing api key", func(t *testing.T) {
		t.Setenv(glmAPIKeyEnv, "")

		cfg := &GLMConfig{
			BaseURL:    "https://test.example.com",
			Model:      "test-model",
			Timeout:    time.Minute,
			MaxRetries: 3,
		}

		_, err := NewLLMClient(cfg)
		if err == nil {
			t.Fatal("expected error for missing API key")
		}

		var apiKeyErr *APIKeyError
		if !strings.Contains(err.Error(), glmAPIKeyEnv) {
			t.Fatalf("error = %v, want reference to %s", err, glmAPIKeyEnv)
		}
		// Check it's an APIKeyError
		if !errors.As(err, &apiKeyErr) {
			t.Fatalf("error = %v, want APIKeyError", err)
		}
	})

	t.Run("with valid config", func(t *testing.T) {
		t.Setenv(glmAPIKeyEnv, "test-key")

		cfg := &GLMConfig{
			BaseURL:    "https://test.example.com",
			Model:      "test-model",
			Timeout:    time.Minute,
			MaxRetries: 3,
		}

		client, err := NewLLMClient(cfg)
		if err != nil {
			t.Fatalf("NewLLMClient() error = %v", err)
		}

		// Verify it implements LLMClient interface
		var _ LLMClient = client
	})
}

// TestAPIKeyError tests APIKeyError error message
func TestAPIKeyError(t *testing.T) {
	err := &APIKeyError{}

	if !strings.Contains(err.Error(), glmAPIKeyEnv) {
		t.Errorf("error message should contain %s", glmAPIKeyEnv)
	}
	if !strings.Contains(err.Error(), "export GLM_API_KEY") {
		t.Error("error message should contain export instruction")
	}
	if !strings.Contains(err.Error(), "https://open.bigmodel.cn/") {
		t.Error("error message should contain API key URL")
	}
}

func TestGLMClientExecute(t *testing.T) {
	client := newTestGLMClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-api-key" {
			t.Fatalf("Authorization = %q, want Bearer test-api-key", got)
		}

		var req glmChatCompletionRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if req.Model != "glm-test-model" {
			t.Fatalf("model = %q, want glm-test-model", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello glm" {
			t.Fatalf("messages = %#v, want prompt hello glm", req.Messages)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "hello from glm",
					},
				},
			},
		})
	}), WithModel("glm-test-model"), WithMaxRetries(0))

	result, err := client.Execute(context.Background(), "hello glm")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "hello from glm" {
		t.Fatalf("Execute() = %q, want hello from glm", result)
	}
}

func TestGLMClientExecuteJSON(t *testing.T) {
	client := newTestGLMClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req glmChatCompletionRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !strings.Contains(req.Messages[0].Content, "有効なJSON形式") {
			t.Fatalf("prompt = %q, want JSON instruction", req.Messages[0].Content)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "```json\n{\"name\":\"glm\",\"enabled\":true}\n```",
					},
				},
			},
		})
	}), WithMaxRetries(0))

	var target struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}

	if err := client.ExecuteJSON(context.Background(), "json please", &target); err != nil {
		t.Fatalf("ExecuteJSON() error = %v", err)
	}
	if target.Name != "glm" || !target.Enabled {
		t.Fatalf("target = %+v, want decoded JSON", target)
	}
}

func TestGLMClientTimeout(t *testing.T) {
	client := newTestGLMClientWithTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case <-time.After(150 * time.Millisecond):
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(recorder).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": "late response",
						},
					},
				},
			})
			return recorder.Result(), nil
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}), WithMaxRetries(0))
	client.SetTimeout(50 * time.Millisecond)

	if client.GetTimeout() != 50*time.Millisecond {
		t.Fatalf("GetTimeout() = %v, want 50ms", client.GetTimeout())
	}
	if client.httpClient.Timeout != 50*time.Millisecond {
		t.Fatalf("httpClient.Timeout = %v, want 50ms", client.httpClient.Timeout)
	}

	_, err := client.Execute(context.Background(), "slow request")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error = %v, want timeout message", err)
	}
}

func TestGLMClientRetry(t *testing.T) {
	var attempts int32

	client := newTestGLMClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"message": "temporary failure"},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "recovered",
					},
				},
			},
		})
	}), WithMaxRetries(2))

	result, err := client.Execute(context.Background(), "retry request")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "recovered" {
		t.Fatalf("Execute() = %q, want recovered", result)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
}

func TestGLMClientMissingAPIKey(t *testing.T) {
	t.Setenv(glmAPIKeyEnv, "")

	client, err := NewGLMClient()
	if err == nil {
		t.Fatalf("expected error, got client %#v", client)
	}
	if !strings.Contains(err.Error(), glmAPIKeyEnv) {
		t.Fatalf("error = %v, want reference to %s", err, glmAPIKeyEnv)
	}
}

func TestGLMClientAPIError(t *testing.T) {
	var attempts int32

	client := newTestGLMClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "invalid request"},
		})
	}))

	_, err := client.Execute(context.Background(), "bad request")
	if err == nil {
		t.Fatal("expected API error, got nil")
	}
	if !strings.Contains(err.Error(), "status 400") || !strings.Contains(err.Error(), "invalid request") {
		t.Fatalf("error = %v, want status 400 and invalid request", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func newTestGLMClient(t *testing.T, handler http.Handler, opts ...ClientOption) *GLMClient {
	t.Helper()

	return newTestGLMClientWithTransport(t, handlerRoundTripper{handler: handler}, opts...)
}

func newTestGLMClientWithTransport(t *testing.T, transport http.RoundTripper, opts ...ClientOption) *GLMClient {
	t.Helper()

	t.Setenv(glmAPIKeyEnv, "test-api-key")

	allOpts := append([]ClientOption{WithBaseURL("https://glm.example.test")}, opts...)
	client, err := NewGLMClient(allOpts...)
	if err != nil {
		t.Fatalf("NewGLMClient() error = %v", err)
	}
	client.httpClient.Transport = transport

	return client
}

type handlerRoundTripper struct {
	handler http.Handler
}

func (rt handlerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	rt.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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
	// This test ensures GLMClient and MockClient implement the LLMClient interface.
	t.Setenv(glmAPIKeyEnv, "test-key")

	glclient, err := NewGLMClient()
	if err != nil {
		t.Fatalf("NewGLMClient() error = %v", err)
	}
	var _ LLMClient = glclient
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

// BenchmarkNewGLMClient benchmarks the GLMClient creation
func BenchmarkNewGLMClient(b *testing.B) {
	b.Setenv(glmAPIKeyEnv, "test-key")
	for i := 0; i < b.N; i++ {
		_, _ = NewGLMClient(WithTimeout(5 * time.Minute))
	}
}
