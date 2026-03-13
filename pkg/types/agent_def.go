// Package types provides core type definitions for the agentic-shell.
// This file contains ClaudeAgentDefinition types - the final output format.
package types

import (
	"encoding/json"
	"fmt"
)

// ============================================================================
// ClaudeAgentDefinition - 最終出力 (Final Output)
// ============================================================================

// ToolDefinition represents a Claude tool definition.
type ToolDefinition struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	InputSchema map[string]interface{} `json:"input_schema" yaml:"input_schema"`
}

// Validate checks if the ToolDefinition is valid.
func (t *ToolDefinition) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if t.Description == "" {
		return fmt.Errorf("tool description is required")
	}
	if t.InputSchema == nil {
		return fmt.Errorf("tool input_schema is required")
	}
	return nil
}

// PromptConfig represents prompt configuration.
type PromptConfig struct {
	SystemPrompt       string            `json:"system_prompt" yaml:"system_prompt"`
	UserPrompt         string            `json:"user_prompt,omitempty" yaml:"user_prompt,omitempty"`
	Examples           []PromptExample   `json:"examples,omitempty" yaml:"examples,omitempty"`
	Traits             map[string]string `json:"traits,omitempty" yaml:"traits,omitempty"`
	CommunicationStyle string            `json:"communication_style,omitempty" yaml:"communication_style,omitempty"`
}

// Validate checks if the PromptConfig is valid.
func (p *PromptConfig) Validate() error {
	if p.SystemPrompt == "" {
		return fmt.Errorf("system_prompt is required")
	}
	for i, e := range p.Examples {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("example[%d]: %w", i, err)
		}
	}
	return nil
}

// PromptExample represents an example for the prompt.
type PromptExample struct {
	Input  string `json:"input" yaml:"input"`
	Output string `json:"output" yaml:"output"`
}

// Validate checks if the PromptExample is valid.
func (e *PromptExample) Validate() error {
	if e.Input == "" {
		return fmt.Errorf("example input is required")
	}
	if e.Output == "" {
		return fmt.Errorf("example output is required")
	}
	return nil
}

// ModelConfig represents model configuration.
type ModelConfig struct {
	ModelID          string   `json:"model_id" yaml:"model_id"`
	MaxTokens        int      `json:"max_tokens" yaml:"max_tokens"`
	Temperature      float64  `json:"temperature" yaml:"temperature"`
	TopP             float64  `json:"top_p,omitempty" yaml:"top_p,omitempty"`
	TopK             int      `json:"top_k,omitempty" yaml:"top_k,omitempty"`
	StopSequences    []string `json:"stop_sequences,omitempty" yaml:"stop_sequences,omitempty"`
	StreamingEnabled bool     `json:"streaming_enabled" yaml:"streaming_enabled"`
}

// Validate checks if the ModelConfig is valid.
func (m *ModelConfig) Validate() error {
	if m.ModelID == "" {
		return fmt.Errorf("model_id is required")
	}
	if m.MaxTokens < 1 {
		return fmt.Errorf("max_tokens must be at least 1, got: %d", m.MaxTokens)
	}
	if m.Temperature < 0 || m.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1, got: %f", m.Temperature)
	}
	if m.TopP < 0 || m.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1, got: %f", m.TopP)
	}
	return nil
}

// ContextConfig represents context configuration.
type ContextConfig struct {
	MaxContextTokens  int    `json:"max_context_tokens" yaml:"max_context_tokens"`
	IncludeHistory    bool   `json:"include_history" yaml:"include_history"`
	HistoryLimit      int    `json:"history_limit" yaml:"history_limit"`
	ContextStrategy   string `json:"context_strategy" yaml:"context_strategy"` // sliding_window, summarize, prioritize
	ReservedTokens    int    `json:"reserved_tokens" yaml:"reserved_tokens"`
	SystemPromptFirst bool   `json:"system_prompt_first" yaml:"system_prompt_first"`
}

// Validate checks if the ContextConfig is valid.
func (c *ContextConfig) Validate() error {
	if c.MaxContextTokens < 1 {
		return fmt.Errorf("max_context_tokens must be at least 1, got: %d", c.MaxContextTokens)
	}
	if c.HistoryLimit < 0 {
		return fmt.Errorf("history_limit must be non-negative, got: %d", c.HistoryLimit)
	}
	return nil
}

// SafetyConfig represents safety configuration.
type SafetyConfig struct {
	ContentFiltering  bool     `json:"content_filtering" yaml:"content_filtering"`
	AllowedContent    []string `json:"allowed_content,omitempty" yaml:"allowed_content,omitempty"`
	BlockedContent    []string `json:"blocked_content,omitempty" yaml:"blocked_content,omitempty"`
	MaxResponseLength int      `json:"max_response_length,omitempty" yaml:"max_response_length,omitempty"`
	PiiHandling       string   `json:"pii_handling" yaml:"pii_handling"` // mask, remove, ignore
}

// Validate checks if the SafetyConfig is valid.
func (s *SafetyConfig) Validate() error {
	return nil
}

// OutputConfig represents output configuration.
type OutputConfig struct {
	Format           string `json:"format" yaml:"format"` // text, json, markdown
	IncludeMetadata  bool   `json:"include_metadata" yaml:"include_metadata"`
	PrettyPrint      bool   `json:"pretty_print" yaml:"pretty_print"`
	IncludeReasoning bool   `json:"include_reasoning" yaml:"include_reasoning"`
	Language         string `json:"language" yaml:"language"` // en, ja, etc.
	Tone             string `json:"tone" yaml:"tone"`         // formal, casual, technical
}

// Validate checks if the OutputConfig is valid.
func (o *OutputConfig) Validate() error {
	if o.Format == "" {
		return fmt.Errorf("output format is required")
	}
	return nil
}

// LoggingConfig represents logging configuration.
type LoggingConfig struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	Level        string `json:"level" yaml:"level"`             // debug, info, warn, error
	Format       string `json:"format" yaml:"format"`           // json, text
	Destination  string `json:"destination" yaml:"destination"` // stdout, file, syslog
	FilePath     string `json:"file_path,omitempty" yaml:"file_path,omitempty"`
	MaxSize      int    `json:"max_size,omitempty" yaml:"max_size,omitempty"` // in MB
	MaxBackups   int    `json:"max_backups,omitempty" yaml:"max_backups,omitempty"`
	MaxAge       int    `json:"max_age,omitempty" yaml:"max_age,omitempty"` // in days
	Compress     bool   `json:"compress" yaml:"compress"`
	IncludeTrace bool   `json:"include_trace" yaml:"include_trace"`
}

// Validate checks if the LoggingConfig is valid.
func (l *LoggingConfig) Validate() error {
	if l.Level != "" && l.Level != "debug" && l.Level != "info" && l.Level != "warn" && l.Level != "error" {
		return fmt.Errorf("invalid log level: %s", l.Level)
	}
	return nil
}

// AgentMetrics represents metrics configuration.
type AgentMetrics struct {
	Enabled           bool     `json:"enabled" yaml:"enabled"`
	CollectLatency    bool     `json:"collect_latency" yaml:"collect_latency"`
	CollectTokenUsage bool     `json:"collect_token_usage" yaml:"collect_token_usage"`
	CollectErrors     bool     `json:"collect_errors" yaml:"collect_errors"`
	ExportInterval    int      `json:"export_interval" yaml:"export_interval"` // in seconds
	ExportFormat      string   `json:"export_format" yaml:"export_format"`     // json, prometheus
	Endpoints         []string `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
}

// Validate checks if the AgentMetrics is valid.
func (m *AgentMetrics) Validate() error {
	return nil
}

// AgentMetadata represents metadata for a Claude agent.
type AgentMetadata struct {
	Name             string            `json:"name" yaml:"name"`
	Version          string            `json:"version" yaml:"version"`
	Author           string            `json:"author,omitempty" yaml:"author,omitempty"`
	Description      string            `json:"description" yaml:"description"`
	CreatedAt        string            `json:"created_at" yaml:"created_at"`
	UpdatedAt        string            `json:"updated_at" yaml:"updated_at"`
	Tags             []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Category         string            `json:"category,omitempty" yaml:"category,omitempty"`
	SourceIntentID   string            `json:"source_intent_id,omitempty" yaml:"source_intent_id,omitempty"`
	SourceSpecID     string            `json:"source_spec_id,omitempty" yaml:"source_spec_id,omitempty"`
	Labels           map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	DocumentationURL string            `json:"documentation_url,omitempty" yaml:"documentation_url,omitempty"`
}

// Validate checks if the AgentMetadata is valid.
func (m *AgentMetadata) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metadata name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("metadata version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("metadata description is required")
	}
	return nil
}

// ClaudeAgentDefinition represents the final agent definition for Claude.
// This is the complete configuration that can be used to instantiate a Claude agent.
type ClaudeAgentDefinition struct {
	// Metadata for the agent
	Metadata AgentMetadata `json:"metadata" yaml:"metadata"`

	// Prompt configuration
	Prompt PromptConfig `json:"prompt" yaml:"prompt"`

	// Model configuration
	Model ModelConfig `json:"model" yaml:"model"`

	// Context configuration
	Context ContextConfig `json:"context" yaml:"context"`

	// Tools available to the agent
	Tools []ToolDefinition `json:"tools" yaml:"tools"`

	// Safety configuration
	Safety SafetyConfig `json:"safety" yaml:"safety"`

	// Output configuration
	Output OutputConfig `json:"output" yaml:"output"`

	// Logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`

	// Metrics configuration
	Metrics AgentMetrics `json:"metrics" yaml:"metrics"`
}

// Validate checks if the ClaudeAgentDefinition is valid.
func (d *ClaudeAgentDefinition) Validate() error {
	if err := d.Metadata.Validate(); err != nil {
		return fmt.Errorf("metadata: %w", err)
	}
	if err := d.Prompt.Validate(); err != nil {
		return fmt.Errorf("prompt: %w", err)
	}
	if err := d.Model.Validate(); err != nil {
		return fmt.Errorf("model: %w", err)
	}
	if err := d.Context.Validate(); err != nil {
		return fmt.Errorf("context: %w", err)
	}
	for i, t := range d.Tools {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("tool[%d]: %w", i, err)
		}
	}
	if err := d.Safety.Validate(); err != nil {
		return fmt.Errorf("safety: %w", err)
	}
	if err := d.Output.Validate(); err != nil {
		return fmt.Errorf("output: %w", err)
	}
	if err := d.Logging.Validate(); err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	if err := d.Metrics.Validate(); err != nil {
		return fmt.Errorf("metrics: %w", err)
	}
	return nil
}

// ToJSON serializes ClaudeAgentDefinition to JSON.
func (d *ClaudeAgentDefinition) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

// FromJSON deserializes ClaudeAgentDefinition from JSON.
func (d *ClaudeAgentDefinition) FromJSON(data []byte) error {
	return json.Unmarshal(data, d)
}

// ToYAML converts the definition to YAML format string.
// Note: Requires gopkg.in/yaml.v3 for full YAML support.
func (d *ClaudeAgentDefinition) ToYAML() (string, error) {
	// Simple implementation using JSON as intermediate
	data, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// NewClaudeAgentDefinition creates a new ClaudeAgentDefinition with default values.
func NewClaudeAgentDefinition(name, description string) *ClaudeAgentDefinition {
	return &ClaudeAgentDefinition{
		Metadata: AgentMetadata{
			Name:        name,
			Version:     "1.0.0",
			Description: description,
			Tags:        []string{},
			Labels:      map[string]string{},
		},
		Prompt: PromptConfig{
			SystemPrompt:       "",
			Examples:           []PromptExample{},
			Traits:             map[string]string{},
			CommunicationStyle: "professional",
		},
		Model: ModelConfig{
			ModelID:          "claude-sonnet-4-6",
			MaxTokens:        4096,
			Temperature:      0.7,
			StreamingEnabled: true,
		},
		Context: ContextConfig{
			MaxContextTokens:  200000,
			IncludeHistory:    true,
			HistoryLimit:      10,
			ContextStrategy:   "sliding_window",
			ReservedTokens:    1000,
			SystemPromptFirst: true,
		},
		Tools: []ToolDefinition{},
		Safety: SafetyConfig{
			ContentFiltering: true,
			PiiHandling:      "mask",
		},
		Output: OutputConfig{
			Format:           "text",
			IncludeMetadata:  false,
			PrettyPrint:      true,
			IncludeReasoning: false,
			Language:         "en",
			Tone:             "technical",
		},
		Logging: LoggingConfig{
			Enabled:      true,
			Level:        "info",
			Format:       "json",
			Destination:  "stdout",
			Compress:     false,
			IncludeTrace: false,
		},
		Metrics: AgentMetrics{
			Enabled:           true,
			CollectLatency:    true,
			CollectTokenUsage: true,
			CollectErrors:     true,
			ExportInterval:    60,
			ExportFormat:      "json",
			Endpoints:         []string{},
		},
	}
}
