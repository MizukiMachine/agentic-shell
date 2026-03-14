// Package llm provides LLM client functionality for agentic-shell
// このパッケージはClaude CLIをsubprocessとして呼び出すLLMクライアントを提供します
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ClaudeClient はClaude CLIをsubprocessで呼び出すクライアントです
type ClaudeClient struct {
	timeout time.Duration
	cliPath string // Claude CLIのパス（デフォルト: "claude"）
	cliArgs []string
}

// ClientOption はClaudeClientのオプション設定です
type ClientOption func(*ClaudeClient)

// NewClaudeClient は新しいClaudeClientを作成します
func NewClaudeClient(opts ...ClientOption) *ClaudeClient {
	client := &ClaudeClient{
		timeout: 5 * time.Minute, // デフォルトタイムアウト
		cliPath: "claude",
		cliArgs: []string{"-p"}, // デフォルトは非対話モード
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithTimeout はタイムアウトを設定するオプションです
func WithTimeout(d time.Duration) ClientOption {
	return func(c *ClaudeClient) {
		c.timeout = d
	}
}

// WithCLIPath はClaude CLIのパスを設定するオプションです
func WithCLIPath(path string) ClientOption {
	return func(c *ClaudeClient) {
		c.cliPath = path
	}
}

// WithCLIArgs は追加のCLI引数を設定するオプションです
func WithCLIArgs(args ...string) ClientOption {
	return func(c *ClaudeClient) {
		c.cliArgs = append(c.cliArgs, args...)
	}
}

// Execute はClaude CLIを非対話モードで実行し、結果を返します
func (c *ClaudeClient) Execute(ctx context.Context, prompt string) (string, error) {
	// タイムアウト付きコンテキストを作成
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// コマンド引数を構築
	args := append(c.cliArgs, prompt)

	// コマンド作成
	cmd := exec.CommandContext(ctx, c.cliPath, args...)

	// 出力を取得
	output, err := cmd.Output()
	if err != nil {
		// タイムアウトチェック
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %v: %w", c.timeout, err)
		}
		// 終了コードエラーの場合は詳細を返す
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("command failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

// ExecuteJSON はClaude CLIを実行し、JSONレスポンスをパースします
func (c *ClaudeClient) ExecuteJSON(ctx context.Context, prompt string, target interface{}) error {
	// プロンプトにJSON要求を追加
	jsonPrompt := fmt.Sprintf("%s\n\n重要: 必ず有効なJSON形式で回答してください。余計な説明は不要です。", prompt)

	output, err := c.Execute(ctx, jsonPrompt)
	if err != nil {
		return fmt.Errorf("execute failed: %w", err)
	}

	// JSONを抽出
	jsonStr, err := extractJSON(output)
	if err != nil {
		return fmt.Errorf("json extraction failed: %w", err)
	}

	// JSONをパース
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("json unmarshal failed: %w (input: %s)", err, jsonStr)
	}

	return nil
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

// SetTimeout はタイムアウトを動的に設定します
func (c *ClaudeClient) SetTimeout(d time.Duration) {
	c.timeout = d
}

// GetTimeout は現在のタイムアウト設定を返します
func (c *ClaudeClient) GetTimeout() time.Duration {
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
var _ LLMClient = (*ClaudeClient)(nil)
