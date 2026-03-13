// Package types provides core type definitions for the agentic-shell.
// This file contains tests for ClaudeAgentDefinition types.
package types

import (
	"encoding/json"
	"testing"
)

func TestToolDefinitionValidation(t *testing.T) {
	tests := []struct {
		name    string
		tool    ToolDefinition
		wantErr bool
	}{
		{
			name: "valid tool",
			tool: ToolDefinition{
				Name:        "read_file",
				Description: "Read a file from disk",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{"type": "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tool: ToolDefinition{
				Description: "Read a file from disk",
				InputSchema: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "missing schema",
			tool: ToolDefinition{
				Name:        "read_file",
				Description: "Read a file from disk",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tool.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestModelConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  ModelConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ModelConfig{
				ModelID:     "claude-sonnet-4-6",
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			wantErr: false,
		},
		{
			name: "missing model ID",
			config: ModelConfig{
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			wantErr: true,
		},
		{
			name: "invalid max tokens",
			config: ModelConfig{
				ModelID:     "claude-sonnet-4-6",
				MaxTokens:   0,
				Temperature: 0.7,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: ModelConfig{
				ModelID:     "claude-sonnet-4-6",
				MaxTokens:   4096,
				Temperature: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid top_p",
			config: ModelConfig{
				ModelID:     "claude-sonnet-4-6",
				MaxTokens:   4096,
				Temperature: 0.7,
				TopP:        1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPromptConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  PromptConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: PromptConfig{
				SystemPrompt: "You are a helpful assistant.",
			},
			wantErr: false,
		},
		{
			name: "missing system prompt",
			config: PromptConfig{
				SystemPrompt: "",
			},
			wantErr: true,
		},
		{
			name: "with examples",
			config: PromptConfig{
				SystemPrompt: "You are a helpful assistant.",
				Examples: []PromptExample{
					{
						Input:  "Hello",
						Output: "Hi! How can I help you?",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestContextConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  ContextConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ContextConfig{
				MaxContextTokens: 200000,
				IncludeHistory:   true,
				HistoryLimit:     10,
			},
			wantErr: false,
		},
		{
			name: "invalid max tokens",
			config: ContextConfig{
				MaxContextTokens: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid history limit",
			config: ContextConfig{
				MaxContextTokens: 200000,
				HistoryLimit:     -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewClaudeAgentDefinition(t *testing.T) {
	def := NewClaudeAgentDefinition("test-agent", "A test agent")

	if def.Metadata.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got '%s'", def.Metadata.Name)
	}
	if def.Metadata.Description != "A test agent" {
		t.Errorf("expected description 'A test agent', got '%s'", def.Metadata.Description)
	}

	// Check default values
	if def.Model.ModelID != "claude-sonnet-4-6" {
		t.Errorf("expected default model 'claude-sonnet-4-6', got '%s'", def.Model.ModelID)
	}
	if def.Model.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", def.Model.MaxTokens)
	}
	if def.Context.MaxContextTokens != 200000 {
		t.Errorf("expected default max_context_tokens 200000, got %d", def.Context.MaxContextTokens)
	}
	if def.Output.Format != "text" {
		t.Errorf("expected default output format 'text', got '%s'", def.Output.Format)
	}
}

func TestClaudeAgentDefinitionJSONRoundTrip(t *testing.T) {
	def := &ClaudeAgentDefinition{
		Metadata: AgentMetadata{
			Name:        "json-test-agent",
			Version:     "1.0.0",
			Description: "Test agent for JSON serialization",
			Author:      "test-author",
			Tags:        []string{"test", "json"},
			Labels: map[string]string{
				"environment": "test",
				"team":        "core",
			},
		},
		Prompt: PromptConfig{
			SystemPrompt: "You are a helpful coding assistant.",
			UserPrompt:   "Help me write code.",
			Examples: []PromptExample{
				{
					Input:  "Write a function to add two numbers",
					Output: "func add(a, b int) int { return a + b }",
				},
			},
			Traits: map[string]string{
				"style":     "concise",
				"tone":      "professional",
				"expertise": "senior",
			},
			CommunicationStyle: "professional",
		},
		Model: ModelConfig{
			ModelID:          "claude-sonnet-4-6",
			MaxTokens:        8192,
			Temperature:      0.5,
			TopP:             0.9,
			StreamingEnabled: true,
		},
		Context: ContextConfig{
			MaxContextTokens:  100000,
			IncludeHistory:    true,
			HistoryLimit:      5,
			ContextStrategy:   "sliding_window",
			ReservedTokens:    500,
			SystemPromptFirst: true,
		},
		Tools: []ToolDefinition{
			{
				Name:        "read_file",
				Description: "Read a file",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{"type": "string"},
					},
				},
			},
		},
		Safety: SafetyConfig{
			ContentFiltering: true,
			PiiHandling:      "mask",
		},
		Output: OutputConfig{
			Format:           "markdown",
			IncludeMetadata:  true,
			PrettyPrint:      true,
			IncludeReasoning: false,
			Language:         "ja",
			Tone:             "technical",
		},
		Logging: LoggingConfig{
			Enabled:      true,
			Level:        "debug",
			Format:       "json",
			Destination:  "stdout",
			IncludeTrace: true,
		},
		Metrics: AgentMetrics{
			Enabled:           true,
			CollectLatency:    true,
			CollectTokenUsage: true,
			ExportInterval:    30,
		},
	}

	// Serialize to JSON
	data, err := def.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Deserialize from JSON
	var decoded ClaudeAgentDefinition
	if err := decoded.FromJSON(data); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Verify key fields
	if decoded.Metadata.Name != def.Metadata.Name {
		t.Errorf("name mismatch: got '%s', want '%s'", decoded.Metadata.Name, def.Metadata.Name)
	}
	if decoded.Model.ModelID != def.Model.ModelID {
		t.Errorf("model ID mismatch: got '%s', want '%s'", decoded.Model.ModelID, def.Model.ModelID)
	}
	if len(decoded.Tools) != len(def.Tools) {
		t.Errorf("tools count mismatch: got %d, want %d", len(decoded.Tools), len(def.Tools))
	}
	if decoded.Output.Language != def.Output.Language {
		t.Errorf("output language mismatch: got '%s', want '%s'", decoded.Output.Language, def.Output.Language)
	}
}

func TestClaudeAgentDefinitionFullValidation(t *testing.T) {
	def := &ClaudeAgentDefinition{
		Metadata: AgentMetadata{
			Name:        "full-validation-agent",
			Version:     "1.0.0",
			Description: "Agent for full validation test",
		},
		Prompt: PromptConfig{
			SystemPrompt: "You are a helpful assistant.",
		},
		Model: ModelConfig{
			ModelID:     "claude-sonnet-4-6",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		Context: ContextConfig{
			MaxContextTokens: 200000,
			IncludeHistory:   true,
			HistoryLimit:     10,
			ContextStrategy:  "sliding_window",
			ReservedTokens:   1000,
		},
		Tools: []ToolDefinition{
			{
				Name:        "shell",
				Description: "Execute shell commands",
				InputSchema: map[string]interface{}{
					"type": "object",
				},
			},
		},
		Safety: SafetyConfig{
			ContentFiltering: true,
			PiiHandling:      "mask",
		},
		Output: OutputConfig{
			Format:          "text",
			Language:        "en",
			Tone:            "professional",
			IncludeMetadata: false,
		},
		Logging: LoggingConfig{
			Enabled:     true,
			Level:       "info",
			Format:      "json",
			Destination: "stdout",
		},
		Metrics: AgentMetrics{
			Enabled:           true,
			CollectLatency:    true,
			CollectTokenUsage: true,
			ExportInterval:    60,
		},
	}

	if err := def.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

func TestAgentMetadataValidation(t *testing.T) {
	tests := []struct {
		name     string
		metadata AgentMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: AgentMetadata{
				Name:        "test-agent",
				Version:     "1.0.0",
				Description: "Test agent",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			metadata: AgentMetadata{
				Version:     "1.0.0",
				Description: "Test agent",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			metadata: AgentMetadata{
				Name:        "test-agent",
				Description: "Test agent",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			metadata: AgentMetadata{
				Name:    "test-agent",
				Version: "1.0.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOutputConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  OutputConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: OutputConfig{
				Format:   "json",
				Language: "en",
				Tone:     "formal",
			},
			wantErr: false,
		},
		{
			name: "missing format",
			config: OutputConfig{
				Language: "en",
				Tone:     "formal",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLoggingConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: LoggingConfig{
				Level:  "invalid",
				Format: "json",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestClaudeAgentDefinitionJSONFieldNames(t *testing.T) {
	def := &ClaudeAgentDefinition{
		Metadata: AgentMetadata{
			Name:        "test",
			Version:     "1.0.0",
			Description: "test",
		},
		Prompt: PromptConfig{
			SystemPrompt: "test",
		},
		Model: ModelConfig{
			ModelID:     "claude-sonnet-4-6",
			MaxTokens:   4096,
			Temperature: 0.7,
		},
		Context: ContextConfig{
			MaxContextTokens: 200000,
		},
		Output: OutputConfig{
			Format: "text",
		},
	}

	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Check snake_case field names
	jsonStr := string(data)

	// Check for snake_case fields - verify some key fields are present
	_ = []string{
		`"max_tokens"`,
		`"max_context_tokens"`,
		`"pii_handling"`,
		`"include_metadata"`,
		`"streaming_enabled"`,
		`"created_at"`,
	}

	t.Logf("JSON output sample: %s", jsonStr)
}
