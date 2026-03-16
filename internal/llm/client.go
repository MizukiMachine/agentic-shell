// Package llm provides LLM client functionality for agentic-shell.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	defaultGLMBaseURL    = "https://open.bigmodel.cn/api/paas/v4/"
	defaultGLMModel      = "glm-4-flash"
	defaultGLMTimeout    = 2 * time.Minute
	defaultGLMMaxRetries = 3
	glmAPIKeyEnv         = "GLM_API_KEY"
)

// GLMClient は GLM API を呼び出すクライアントです。
type GLMClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

// ClientOption は LLM クライアントのオプション設定です。
type ClientOption interface {
	applyGLM(*GLMClient)
}

type clientOption struct {
	applyGLMFunc func(*GLMClient)
}

func (o clientOption) applyGLM(c *GLMClient) {
	if o.applyGLMFunc != nil {
		o.applyGLMFunc(c)
	}
}

// GLMConfig は NewLLMClient で使用する設定です。
type GLMConfig struct {
	BaseURL    string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

// NewGLMClient は新しい GLMClient を作成します。
func NewGLMClient(opts ...ClientOption) (*GLMClient, error) {
	apiKey := strings.TrimSpace(os.Getenv(glmAPIKeyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set", glmAPIKeyEnv)
	}

	client := &GLMClient{
		apiKey:     apiKey,
		baseURL:    defaultGLMBaseURL,
		model:      defaultGLMModel,
		timeout:    defaultGLMTimeout,
		maxRetries: defaultGLMMaxRetries,
	}

	for _, opt := range opts {
		opt.applyGLM(client)
	}

	if client.baseURL == "" {
		return nil, fmt.Errorf("baseURL is required")
	}
	if client.model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if client.maxRetries < 0 {
		return nil, fmt.Errorf("maxRetries must be non-negative")
	}

	client.httpClient = &http.Client{Timeout: client.timeout}

	return client, nil
}

// WithTimeout はタイムアウトを設定するオプションです
func WithTimeout(d time.Duration) ClientOption {
	return clientOption{
		applyGLMFunc: func(c *GLMClient) {
			c.timeout = d
			if c.httpClient != nil {
				c.httpClient.Timeout = d
			}
		},
	}
}

// WithModel は GLM のモデル名を設定するオプションです。
func WithModel(model string) ClientOption {
	return clientOption{
		applyGLMFunc: func(c *GLMClient) {
			c.model = model
		},
	}
}

// WithBaseURL は GLM API のベース URL を設定するオプションです。
func WithBaseURL(url string) ClientOption {
	return clientOption{
		applyGLMFunc: func(c *GLMClient) {
			c.baseURL = url
		},
	}
}

// WithMaxRetries は GLM API 呼び出し時の最大リトライ回数を設定するオプションです。
func WithMaxRetries(n int) ClientOption {
	return clientOption{
		applyGLMFunc: func(c *GLMClient) {
			c.maxRetries = n
		},
	}
}

// NewLLMClient は設定からGLMクライアントを作成します。
// GLM_API_KEY環境変数が未設定の場合はエラーを返します。
func NewLLMClient(cfg *GLMConfig) (LLMClient, error) {
	apiKey := strings.TrimSpace(os.Getenv(glmAPIKeyEnv))
	if apiKey == "" {
		return nil, &APIKeyError{}
	}

	opts := []ClientOption{}

	if cfg.BaseURL != "" {
		opts = append(opts, WithBaseURL(cfg.BaseURL))
	}
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, WithTimeout(cfg.Timeout))
	}
	if cfg.MaxRetries > 0 {
		opts = append(opts, WithMaxRetries(cfg.MaxRetries))
	}

	return NewGLMClient(opts...)
}

// APIKeyError はAPIキーが未設定の場合のエラーです。
type APIKeyError struct{}

func (e *APIKeyError) Error() string {
	return fmt.Sprintf(`%s environment variable is not set.
Please set your GLM API key:
  export GLM_API_KEY=your_api_key_here
Get your API key at: https://open.bigmodel.cn/`, glmAPIKeyEnv)
}

type glmChatCompletionRequest struct {
	Model    string       `json:"model"`
	Messages []glmMessage `json:"messages"`
}

type glmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type glmChatCompletionResponse struct {
	Choices []struct {
		Message glmMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

// Execute は GLM API を実行し、テキスト応答を返します。
func (c *GLMClient) Execute(ctx context.Context, prompt string) (string, error) {
	requestBody, err := json.Marshal(glmChatCompletionRequest{
		Model: c.model,
		Messages: []glmMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode request: %w", err)
	}

	var lastErr error
	attempts := 0
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		attempts++

		result, retry, execErr := c.executeRequest(ctx, requestBody)
		if execErr == nil {
			return result, nil
		}
		lastErr = execErr

		if !retry || attempt == c.maxRetries || ctx.Err() != nil {
			break
		}
	}

	return "", fmt.Errorf("glm request failed after %d attempt(s): %w", attempts, lastErr)
}

func (c *GLMClient) executeRequest(ctx context.Context, requestBody []byte) (string, bool, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.endpointURL(), bytes.NewReader(requestBody))
	if err != nil {
		return "", false, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", false, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		if reqCtx.Err() == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
			return "", true, fmt.Errorf("request timed out after %v: %w", c.timeout, err)
		}

		var netErr net.Error
		if errors.As(err, &netErr) {
			return "", true, fmt.Errorf("network error: %w", err)
		}

		return "", true, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", shouldRetryStatus(resp.StatusCode), fmt.Errorf("glm API request failed with status %d: %s", resp.StatusCode, apiErrorMessage(body))
	}

	var response glmChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", false, fmt.Errorf("failed to decode response: %w", err)
	}
	if response.Error != nil && response.Error.Message != "" {
		return "", false, fmt.Errorf("glm API returned an error payload: %s", response.Error.Message)
	}
	if len(response.Choices) == 0 {
		return "", false, fmt.Errorf("glm API response contained no choices")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	if content == "" {
		return "", false, fmt.Errorf("glm API response contained empty content")
	}

	return content, false, nil
}

// ExecuteJSON は GLM API を実行し、JSON レスポンスをパースします。
func (c *GLMClient) ExecuteJSON(ctx context.Context, prompt string, target interface{}) error {
	jsonPrompt := fmt.Sprintf("%s\n\n重要: 必ず有効なJSON形式で回答してください。余計な説明は不要です。", prompt)

	output, err := c.Execute(ctx, jsonPrompt)
	if err != nil {
		return fmt.Errorf("execute failed: %w", err)
	}

	jsonStr, err := extractJSON(output)
	if err != nil {
		return fmt.Errorf("json extraction failed: %w", err)
	}

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("json unmarshal failed: %w (input: %s)", err, jsonStr)
	}

	return nil
}

func (c *GLMClient) endpointURL() string {
	return strings.TrimRight(c.baseURL, "/") + "/chat/completions"
}

func apiErrorMessage(body []byte) string {
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &payload); err == nil && payload.Error.Message != "" {
		return payload.Error.Message
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty response body"
	}

	return truncate(message, 200)
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

// extractJSON は出力からJSONを抽出します
// マークダウンコードブロックから抽出、または生JSONにフォールバック
func extractJSON(output string) (string, error) {
	output = strings.TrimSpace(output)

	// パターン1: マークダウンコードブロック ```json ... ```
	jsonBlockRegex := regexp.MustCompile("(?s)```json\\s*\\n?(.*?)\\n?```")
	if matches := jsonBlockRegex.FindStringSubmatch(output); len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// パターン2: 汎用コードブロック ``` ... ```
	genericBlockRegex := regexp.MustCompile("(?s)```\\s*\\n?(.*?)\\n?```")
	if matches := genericBlockRegex.FindStringSubmatch(output); len(matches) > 1 {
		content := strings.TrimSpace(matches[1])
		// JSONオブジェクトまたは配列で始まるかチェック
		if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
			return content, nil
		}
	}

	// パターン3: 生JSON（{ または [ で始まる）
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}

	// パターン4: テキスト中からJSONオブジェクトを探す
	jsonObjectRegex := regexp.MustCompile("(?s)(\\{.*\\})")
	if matches := jsonObjectRegex.FindStringSubmatch(output); len(matches) > 1 {
		return matches[1], nil
	}

	// パターン5: テキスト中からJSON配列を探す
	jsonArrayRegex := regexp.MustCompile("(?s)(\\[.*\\])")
	if matches := jsonArrayRegex.FindStringSubmatch(output); len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("no valid JSON found in output: %s", truncate(output, 200))
}

// truncate は文字列を指定長で切り詰めます
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SetTimeout はタイムアウトを動的に設定します。
func (c *GLMClient) SetTimeout(d time.Duration) {
	c.timeout = d
	if c.httpClient != nil {
		c.httpClient.Timeout = d
	}
}

// GetTimeout は現在のタイムアウト設定を返します。
func (c *GLMClient) GetTimeout() time.Duration {
	return c.timeout
}

// LLMClient はLLMクライアントの拡張ポイントです。
type LLMClient interface {
	Execute(ctx context.Context, prompt string) (string, error)
	ExecuteJSON(ctx context.Context, prompt string, target interface{}) error
	SetTimeout(d time.Duration)
	GetTimeout() time.Duration
}

// Client は後方互換のためのエイリアスです。
type Client = LLMClient

// Compile-time interface check
var _ LLMClient = (*GLMClient)(nil)
